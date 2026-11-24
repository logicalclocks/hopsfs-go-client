package transfer

import (
	"context"
	"errors"
	"fmt"
	"hash/crc32"
	"io"
	"io/ioutil"
	"net"
	"strings"
	"time"

	hdfs "github.com/colinmarc/hdfs/v2/internal/protocol/hadoop_hdfs"
	"google.golang.org/protobuf/proto"
)

// BlockReader implements io.ReadCloser, for reading a block. It abstracts over
// reading from multiple datanodes, in order to be robust to connection
// failures, timeouts, and other shenanigans.
type BlockReader struct {
	// ClientName is the unique ID used by the NamenodeConnection to locate the
	// block.
	ClientName string
	// Block is the block location provided by the namenode.
	Block *hdfs.LocatedBlockProto
	// Offset is the current read offset in the block.
	Offset int64
	// UseDatanodeHostname specifies whether the datanodes should be connected to
	// via their hostnames (if true) or IP addresses (if false).
	UseDatanodeHostname bool
	// DialFunc is used to connect to the datanodes. If nil, then
	// (&net.Dialer{}).DialContext is used.
	DialFunc func(ctx context.Context, network, addr string) (net.Conn, error)

	datanodes *datanodeFailover
	stream    *blockReadStream
	conn      net.Conn
	deadline  time.Time
	closed    bool

	// retryStart is when retryCloudRead first ran since the last successful
	// connect, zero if it has not.
	retryStart   time.Time
	retryBackoff time.Duration
}

const maxSkip = 65536

const (
	cloudReadInitialBackoff = 200 * time.Millisecond
	cloudReadMaxBackoff     = 5 * time.Second
	cloudReadBudget         = 60 * time.Second
)

// Replaced by tests, which run the retry on a fake clock.
var (
	now   = time.Now
	sleep = time.Sleep
)

// SetDeadline sets the deadline for future Read calls. A zero value for t
// means Read will not time out.
func (br *BlockReader) SetDeadline(t time.Time) error {
	br.deadline = t
	if br.conn != nil {
		return br.conn.SetDeadline(t)
	}

	// Return the error at connection time.
	return nil
}

// Read implements io.Reader.
//
// In the case that a failure (such as a disconnect) occurs while reading, the
// BlockReader will failover to another datanode and continue reading
// transparently. In the case that all the datanodes fail, the error
// from the most recent attempt will be returned, unless retryCloudRead
// tries them again.
//
// Any datanode failures are recorded in a global cache, so subsequent reads,
// even reads for different blocks, will prioritize them lower.
func (br *BlockReader) Read(b []byte) (int, error) {
	if br.closed {
		return 0, io.ErrClosedPipe
	} else if uint64(br.Offset) >= br.Block.GetB().GetNumBytes() {
		br.Close()
		return 0, io.EOF
	}

	if br.datanodes == nil {
		br.datanodes = newDatanodeFailover(br.datanodeAddresses())
	}

	if len(br.Block.Data) > 0 {
		remaining := br.Block.Data[br.Offset:]
		if len(remaining) == 0 {
			return 0, io.EOF
		} else {
			copied := copy(b, remaining)
			br.Offset += int64(copied)
			return copied, nil
		}
	}

	// This is the main retry loop.
	for br.stream != nil || br.datanodes.numRemaining() > 0 || br.retryCloudRead() {
		// First, we try to connect. If this fails, we can just skip the datanode
		// and continue.
		if br.stream == nil {
			// A retry attempt starts only within the budget, whose last attempt
			// is at its end, and before the read deadline: an attempt at or after
			// the deadline could not read anything, and its dial and handshake
			// are not bounded by the deadline.
			if !br.retryStart.IsZero() {
				t := now()
				if t.After(br.retryStart.Add(cloudReadBudget)) ||
					(!br.deadline.IsZero() && !t.Before(br.deadline)) {
					break
				}
			}
			err := br.connectNext()
			if err != nil {
				br.datanodes.recordFailure(err)
				continue
			}
			br.retryStart = time.Time{}
		}

		// Then, try to read. If we fail here after reading some bytes, we return
		// a partial read (n < len(b)).
		n, err := br.stream.Read(b)
		br.Offset += int64(n)
		if err != nil && err != io.EOF {
			br.stream = nil
			br.datanodes.recordFailure(err)
			if n > 0 {
				return n, nil
			}

			continue
		}

		return n, err
	}

	err := br.datanodes.lastError()
	if err == nil {
		err = errors.New("no available datanodes")
	}

	return 0, err
}

// Skip attempts to discard bytes in the stream in order to skip forward. This
// is an optimization for the case that the amount to skip is very small. It
// returns an error if skip was not attempted at all (because the BlockReader
// isn't connected, or the offset is out of bounds or too far ahead) or the seek
// failed for some other reason.
func (br *BlockReader) Skip(fileOldOffset int64, fileNewOffset int64) error {

	if fileOldOffset == fileNewOffset {
		return nil
	}

	// small files
	if len(br.Block.Data) > 0 {
		if fileNewOffset < int64(len(br.Block.Data)) {
			br.Offset = fileNewOffset
			return nil
		} else {
			return io.ErrUnexpectedEOF
		}
	}

	blockStartOffset := int64(br.Block.GetOffset())
	blockEndOffset := int64(br.Block.GetOffset() + br.Block.GetB().GetNumBytes())

	if fileNewOffset < blockStartOffset || fileNewOffset >= blockEndOffset {
		return errors.New("unable to skip. Offset is not in the current block")
	}

	if br.stream == nil {
		return errors.New("unable to skip. br stream is not set")
	}

	if fileNewOffset < fileOldOffset {
		return errors.New("unable to skip. Backward seek is not yet implemented")
	}

	if fileNewOffset > fileOldOffset {

		if (fileNewOffset - fileOldOffset) > maxSkip {
			return errors.New("unable to skip. Exceedes maxSkip")
		}

		// skip forward
		amountToSkip := fileNewOffset - fileOldOffset
		w, err := io.CopyN(io.Discard, br.stream, amountToSkip)
		br.Offset += int64(w)
		if err != nil || w != amountToSkip {
			if err == io.EOF {
				err = io.ErrUnexpectedEOF
			}

			if w != amountToSkip {
				err = fmt.Errorf("failed to skip correct number of bytes. Expected to skip: %d, skipped: %d", amountToSkip, w)
			}

			br.stream = nil
			br.datanodes.recordFailure(err)
			return err
		}
	}
	return nil
}

// Close implements io.Closer.
func (br *BlockReader) Close() error {
	br.closed = true
	if br.conn != nil {
		br.conn.Close()
	}

	return nil
}

func (br *BlockReader) datanodeAddresses() []string {
	locs := br.Block.GetLocs()
	datanodes := make([]string, len(locs))
	for i, loc := range locs {
		datanodes[i] = getDatanodeAddress(loc.GetId(), br.UseDatanodeHostname)
	}
	return datanodes
}

// retryCloudRead waits, then gives Read the block's locations again, when
// the last location failed because a cloud block is not in the object store
// yet. With async cloud upload, only the DataNode that wrote the block has
// it until the upload finishes, and the NameNode returns that DataNode as
// the block's only location. With HOPSFS_CLOUD_DATANODE_HOSTNAME_OVERRIDE
// set, every location is the DataNode load balancer, which may pick any
// DataNode for a connection, so a new connection may reach the writer, and
// after the upload any DataNode serves the block. It gives up
// cloudReadBudget after its first wait since the last successful connect,
// its last attempt at the end of that budget, or when the next attempt would
// start at or after the read deadline, whose read would time out at once.
// Read then returns the last error.
func (br *BlockReader) retryCloudRead() bool {
	if getEnv(HOPSFS_CLOUD_DATANODE_HOSTNAME_OVERRIDE_ENV) == "" ||
		br.Block.GetB().GetCloudBucket() == "" ||
		!isCloudBlockNotVisible(br.datanodes.lastError()) {
		return false
	}

	t := now()
	if br.retryStart.IsZero() {
		br.retryStart = t
		br.retryBackoff = cloudReadInitialBackoff
	}
	stop := br.retryStart.Add(cloudReadBudget)
	if !t.Before(stop) {
		return false
	}

	wait := br.retryBackoff
	if wait > stop.Sub(t) {
		wait = stop.Sub(t)
	}
	if !br.deadline.IsZero() && !t.Add(wait).Before(br.deadline) {
		return false
	}
	sleep(wait)
	br.retryBackoff *= 2
	if br.retryBackoff > cloudReadMaxBackoff {
		br.retryBackoff = cloudReadMaxBackoff
	}

	// Keep the last error, which Read returns if the deadline stops this round
	// before its first attempt.
	datanodes := newDatanodeFailover(br.datanodeAddresses())
	datanodes.err = br.datanodes.lastError()
	br.datanodes = datanodes
	return true
}

// isCloudBlockNotVisible reports whether err is a DataNode's answer for a
// cloud block that is not in the object store yet: ReplicaNotFoundException
// before the upload, or the S3 or GCS miss on the .data object between the
// .meta and .data uploads.
func isCloudBlockNotVisible(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "ReplicaNotFoundException") ||
		strings.Contains(msg, "NoSuchKey") ||
		strings.Contains(msg, "not found in the bucket")
}

// connectNext pops a datanode from the list based on previous failures, and
// connects to it.
func (br *BlockReader) connectNext() error {
	address := br.datanodes.next()

	if br.DialFunc == nil {
		br.DialFunc = (&net.Dialer{}).DialContext
	}

	conn, err := br.DialFunc(context.Background(), "tcp", address)
	if err != nil {
		return err
	}

	err = br.writeBlockReadRequest(conn)
	if err != nil {
		conn.Close()
		return err
	}

	resp, err := readBlockOpResponse(conn)
	if err != nil {
		conn.Close()
		return err
	} else if resp.GetStatus() != hdfs.Status_SUCCESS {
		conn.Close()
		return fmt.Errorf("read failed: %s (%s)", resp.GetStatus().String(), resp.GetMessage())
	}

	readInfo := resp.GetReadOpChecksumInfo()
	checksumInfo := readInfo.GetChecksum()

	var checksumTab *crc32.Table
	var checksumSize int
	checksumType := checksumInfo.GetType()
	switch checksumType {
	case hdfs.ChecksumTypeProto_CHECKSUM_CRC32:
		checksumTab = crc32.IEEETable
		checksumSize = 4
	case hdfs.ChecksumTypeProto_CHECKSUM_CRC32C:
		checksumTab = crc32.MakeTable(crc32.Castagnoli)
		checksumSize = 4
	case hdfs.ChecksumTypeProto_CHECKSUM_NULL:
		checksumTab = nil
		checksumSize = 0
	default:
		conn.Close()
		return fmt.Errorf("unsupported checksum type: %d", checksumType)
	}

	chunkOffset := int64(readInfo.GetChunkOffset())
	chunkSize := int(checksumInfo.GetBytesPerChecksum())
	stream := newBlockReadStream(conn, chunkSize, checksumTab, checksumSize)

	// The read will start aligned to a chunk boundary, so we need to skip
	// forward to the requested offset.
	amountToDiscard := br.Offset - chunkOffset
	if amountToDiscard > 0 {
		_, err := io.CopyN(ioutil.Discard, stream, amountToDiscard)
		if err != nil {
			if err == io.EOF {
				err = io.ErrUnexpectedEOF
			}

			conn.Close()
			return err
		}
	}

	err = conn.SetDeadline(br.deadline)
	if err != nil {
		conn.Close()
		return err
	}

	br.stream = stream
	br.conn = conn
	return nil
}

// A read request to a datanode:
// +-----------------------------------------------------------+
// |  Data Transfer Protocol Version, int16                    |
// +-----------------------------------------------------------+
// |  Op code, 1 byte (READ_BLOCK = 0x51)                      |
// +-----------------------------------------------------------+
// |  varint length + OpReadBlockProto                         |
// +-----------------------------------------------------------+
func (br *BlockReader) writeBlockReadRequest(w io.Writer) error {
	needed := br.Block.GetB().GetNumBytes() - uint64(br.Offset)
	op := &hdfs.OpReadBlockProto{
		Header: &hdfs.ClientOperationHeaderProto{
			BaseHeader: &hdfs.BaseHeaderProto{
				Block: br.Block.GetB(),
				Token: br.Block.GetBlockToken(),
			},
			ClientName: proto.String(br.ClientName),
		},
		Offset: proto.Uint64(uint64(br.Offset)),
		Len:    proto.Uint64(needed),
	}

	return writeBlockOpRequest(w, readBlockOp, op)
}
