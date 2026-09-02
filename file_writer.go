package hdfs

import (
	"crypto/cipher"
	"errors"
	"os"
	"strings"
	"time"

	hdfs "github.com/colinmarc/hdfs/v2/internal/protocol/hadoop_hdfs"
	"github.com/colinmarc/hdfs/v2/internal/transfer"
	"google.golang.org/protobuf/proto"
)

const MaxSmallFileSize = 1024 * 64

var ErrReplicating = errors.New("replication in progress")

// IsErrReplicating returns true if the passed error is an os.PathError wrapping
// ErrReplicating.
func IsErrReplicating(err error) bool {
	pe, ok := err.(*os.PathError)
	return ok && pe.Err == ErrReplicating
}

// A FileWriter represents a writer for an open file in HDFS. It implements
// Writer and Closer, and can only be used for writes. For reads, see
// FileReader and Client.Open.
type FileWriter struct {
	client      *Client
	name        string
	replication int
	blockSize   int64
	fileId      *uint64

	blockWriter     *transfer.BlockWriter
	deadline        time.Time
	storeInDB       bool
	smallFileBuffer []byte
	pos             uint64
	lastError       error
	// newGS is the bumped generation stamp returned by
	// updateBlockForPipeline during Append(). It is the GS the DN
	// finalized at, so the `complete` RPC must report this value as the
	// lastBlock's GS, not the pre-bump GS stored on blockWriter.Block.B.
	newGS uint64

	// Key and IV for transparent encryption support.
	enc *transparentEncryptionInfo
}

// Create opens a new file in HDFS with the default replication, block size,
// and permissions (0644), and returns an io.WriteCloser for writing
// to it. Because of the way that HDFS writes are buffered and acknowledged
// asynchronously, it is very important that Close is called after all data has
// been written.
func (c *Client) Create(name string) (*FileWriter, error) {
	_, err := c.getFileInfo(name)
	err = interpretException(err)
	if err == nil {
		return nil, &os.PathError{Op: "create", Path: name, Err: os.ErrExist}
	} else if !os.IsNotExist(err) {
		return nil, &os.PathError{Op: "create", Path: name, Err: err}
	}

	defaults, err := c.fetchDefaults()
	if err != nil {
		return nil, err
	}

	replication := int(defaults.GetReplication())
	blockSize := int64(defaults.GetBlockSize())
	return c.CreateFile(name, replication, blockSize, 0644, false, false)
}

// CreateFile opens a new file in HDFS with the given replication, block size,
// and permissions, and returns an io.WriteCloser for writing to it. Because of
// the way that HDFS writes are buffered and acknowledged asynchronously, it is
// very important that Close is called after all data has been written.
func (c *Client) CreateFile(name string, replication int, blockSize int64, perm os.FileMode, overwrite bool, createParent bool) (*FileWriter, error) {
	return c.createFileWithGroup(name, replication, blockSize, perm, overwrite, createParent, "")
}

// CreateFileWithGroup opens a new file in HDFS with the given replication, block size,
// permissions, and group name, and returns an io.WriteCloser for writing to it.
// If groupname is empty, the file inherits the group from its parent directory.
// Because of the way that HDFS writes are buffered and acknowledged asynchronously,
// it is very important that Close is called after all data has been written.
func (c *Client) CreateFileWithGroup(name string, replication int, blockSize int64, perm os.FileMode, overwrite bool, createParent bool, groupname string) (*FileWriter, error) {
	return c.createFileWithGroup(name, replication, blockSize, perm, overwrite, createParent, groupname)
}

func (c *Client) createFileWithGroup(name string, replication int, blockSize int64, perm os.FileMode, overwrite bool, createParent bool, groupname string) (*FileWriter, error) {
	createFlag := proto.Uint32(1)
	if overwrite {
		createFlag = proto.Uint32(3) // 0x01 for Create and 0x10 for overwrite
	}

	createReq := &hdfs.CreateRequestProto{
		Src:                   proto.String(name),
		Masked:                &hdfs.FsPermissionProto{Perm: proto.Uint32(uint32(perm))},
		ClientName:            proto.String(c.namenode.ClientName),
		CreateFlag:            createFlag,
		CreateParent:          proto.Bool(createParent),
		Replication:           proto.Uint32(uint32(replication)),
		BlockSize:             proto.Uint64(uint64(blockSize)),
		CryptoProtocolVersion: []hdfs.CryptoProtocolVersionProto{hdfs.CryptoProtocolVersionProto_ENCRYPTION_ZONES},
	}

	// Set groupname if provided (non-empty)
	if groupname != "" {
		createReq.Groupname = proto.String(groupname)
	}

	createResp := &hdfs.CreateResponseProto{}

	err := c.namenode.Execute("create", createReq, createResp)
	if err != nil {
		return nil, &os.PathError{"create", name, interpretCreateException(err)}
	}

	storedInDB := false
	if *createResp.Fs.StoragePolicy == uint32(14) {
		storedInDB = true
	}

	var enc *transparentEncryptionInfo
	if createResp.GetFs().GetFileEncryptionInfo() != nil {
		enc, err = c.kmsGetKey(createResp.GetFs().GetFileEncryptionInfo())
		if err != nil {
			_ = c.Remove(name)
			return nil, &os.PathError{"create", name, err}
		}
	}

	return &FileWriter{
		client:          c,
		name:            name,
		replication:     replication,
		blockSize:       blockSize,
		fileId:          createResp.Fs.FileId,
		storeInDB:       storedInDB,
		smallFileBuffer: []byte{},
		pos:             0,
		enc:             enc,
	}, nil
}

// Append opens an existing file in HDFS and returns an io.WriteCloser for
// writing to it. Because of the way that HDFS writes are buffered and
// acknowledged asynchronously, it is very important that Close is called after
// all data has been written.
func (c *Client) Append(name string) (*FileWriter, error) {
	_, err := c.getFileInfo(name)
	if err != nil {
		return nil, &os.PathError{Op: "append", Path: name, Err: interpretException(err)}
	}

	appendReq := &hdfs.AppendRequestProto{
		Src:        proto.String(name),
		ClientName: proto.String(c.namenode.ClientName),
	}
	appendResp := &hdfs.AppendResponseProto{}

	initDelay := time.Duration(100)
	for i := 0; i < 9; i++ { // 1 min max
		err = c.namenode.Execute("append", appendReq, appendResp)
		// Retry on transient append-time errors:
		//   - NotReplicatedYetException: NN-side wraps this in RetriableException
		//     when the last block is COMMITTED (waiting for IBR to drive it to
		//     COMPLETE).
		//   - "is not sufficiently replicated yet": NN's appendFileInternal
		//     throws plain IOException when the last block is COMPLETE but
		//     liveReplicas < minReplication. This is also transient — it
		//     resolves once DN block reports land. The Java client never
		//     reproducibly hits this because its DataStreamer waits for ack
		//     on prior writes, but the Go client's tight loop in
		//     TestFileAppendRepeatedly races it.
		if err != nil &&
			(strings.Contains(err.Error(), "NotReplicatedYetException") ||
				strings.Contains(err.Error(), "is not sufficiently replicated yet")) {
			time.Sleep(initDelay * time.Millisecond)
			initDelay *= 2
		} else if err != nil {
			return nil, &os.PathError{Op: "append", Path: name, Err: interpretException(err)}
		} else {
			break
		}
	}
	// Retry budget exhausted while the last error was still retryable —
	// surface it instead of falling through with a nil appendResp.Stat,
	// which would NPE at the FileWriter construction below.
	if err != nil {
		return nil, &os.PathError{Op: "append", Path: name, Err: interpretException(err)}
	}

	var enc *transparentEncryptionInfo
	if appendResp.GetStat().GetFileEncryptionInfo() != nil {
		enc, err = c.kmsGetKey(appendResp.GetStat().GetFileEncryptionInfo())
		if err != nil {
			return nil, &os.PathError{"append", name, err}
		}
	}

	f := &FileWriter{
		client:          c,
		name:            name,
		replication:     int(appendResp.Stat.GetBlockReplication()),
		blockSize:       int64(appendResp.Stat.GetBlocksize()),
		fileId:          appendResp.Stat.FileId,
		storeInDB:       false,
		smallFileBuffer: []byte{},
		pos:             *appendResp.GetStat().Length,
		enc:             enc,
	}

	// This returns nil if there are no blocks (it's an empty file) or if the
	// last block is full (so we have to start a fresh block).
	block := appendResp.GetBlock()
	if block == nil {
		return f, nil
	}

	//handling appending to phantom block
	if len(appendResp.GetBlock().Data) > 0 {
		f.storeInDB = true
		f.smallFileBuffer = appendResp.GetBlock().Data
		return f, nil
	}

	// Bump the generation stamp for this append, mirroring the Java
	// DataStreamer's setupPipelineForAppendOrRecovery flow. The NN's
	// prepareFileForAppend returns the un-bumped block; without this
	// second RPC the DN sees latestGenerationStamp == block.GS and
	// CloudFsDatasetImpl.appendInternal adds the current GS into
	// ProvidedReplicaBeingWritten's oldGS list, scheduling a
	// self-delete of the just-uploaded cloud object (HOPSFS-345
	// partial-chunk-append self-delete).
	oldExtBlock := block.GetB()
	updReq := &hdfs.UpdateBlockForPipelineRequestProto{
		Block:      oldExtBlock,
		ClientName: proto.String(c.namenode.ClientName),
	}
	updResp := &hdfs.UpdateBlockForPipelineResponseProto{}
	err = c.namenode.Execute("updateBlockForPipeline", updReq, updResp)
	if err != nil {
		return nil, &os.PathError{Op: "append", Path: name, Err: interpretException(err)}
	}
	newLocatedBlock := updResp.GetBlock()
	newGS := newLocatedBlock.GetB().GetGenerationStamp()

	// Confirm the new GS + pipeline targets on the NN. Java's
	// DataStreamer calls updatePipeline only after writeBlock succeeds
	// (so it can retry with new nodes if the pipeline failed). The Go
	// client has no pipeline recovery, so we commit to the freshly
	// returned pipeline upfront and rely on the eventual `complete`
	// RPC to surface any node-level failure.
	//
	// The locations come from the original `append` response, NOT from
	// the updateBlockForPipeline response: FSNamesystem.updateBlockForPipeline
	// returns a LocatedBlock with empty DatanodeInfo[] (see
	// FSNamesystem.java:6591 — `new LocatedBlock(block, new DatanodeInfo[0])`).
	// Passing those empty locations into updatePipeline would NPE inside
	// BlockInfoContiguousUnderConstruction.setExpectedLocations.
	newExtBlock := &hdfs.ExtendedBlockProto{
		PoolId:          oldExtBlock.PoolId,
		BlockId:         oldExtBlock.BlockId,
		GenerationStamp: proto.Uint64(newGS),
		NumBytes:        oldExtBlock.NumBytes,
		// Preserve cloudBucket: HopsFS's FSNamesystem.commitOrCompleteLastBlock
		// (line 4312) and appendFileInternal (line 2386) branch on
		// commitBlock.isProvidedBlock(), which is true iff cloudBucket !=
		// NON_EXISTENT_BUCKET_NAME. Dropping this field makes the NN treat
		// the next append's last block as a regular HDFS block and wait
		// indefinitely for liveReplicas >= minReplication.
		CloudBucket: oldExtBlock.CloudBucket,
	}
	newNodes := make([]*hdfs.DatanodeIDProto, 0, len(block.GetLocs()))
	for _, info := range block.GetLocs() {
		newNodes = append(newNodes, info.GetId())
	}
	updPipReq := &hdfs.UpdatePipelineRequestProto{
		ClientName: proto.String(c.namenode.ClientName),
		OldBlock:   oldExtBlock,
		NewBlock:   newExtBlock,
		NewNodes:   newNodes,
		StorageIDs: block.GetStorageIDs(),
	}
	updPipResp := &hdfs.UpdatePipelineResponseProto{}
	err = c.namenode.Execute("updatePipeline", updPipReq, updPipResp)
	if err != nil {
		return nil, &os.PathError{Op: "append", Path: name, Err: interpretException(err)}
	}
	f.newGS = newGS

	dialFunc, err := f.client.wrapDatanodeDial(
		f.client.options.DatanodeDialFunc,
		block.GetBlockToken())
	if err != nil {
		return nil, err
	}

	f.blockWriter = &transfer.BlockWriter{
		ClientName:          f.client.namenode.ClientName,
		Block:               block,
		BlockSize:           f.blockSize,
		Offset:              int64(block.B.GetNumBytes()),
		Append:              true,
		NewGS:               newGS,
		UseDatanodeHostname: f.client.options.UseDatanodeHostname,
		RemoteAccess:        f.client.options.RemoteAccess,
		DialFunc:            dialFunc,
	}

	err = f.blockWriter.SetDeadline(f.deadline)
	if err != nil {
		return nil, err
	}

	return f, nil
}

// CreateEmptyFile creates a empty file at the given name, with the
// permissions 0644.
func (c *Client) CreateEmptyFile(name string) error {
	f, err := c.Create(name)
	if err != nil {
		return err
	}

	return f.Close()
}

// SetDeadline sets the deadline for future Write, Flush, and Close calls. A
// zero value for t means those calls will not time out.
//
// Note that because of buffering, Write calls that do not result in a blocking
// network call may still succeed after the deadline.
func (f *FileWriter) SetDeadline(t time.Time) error {
	f.deadline = t
	if f.blockWriter != nil {
		return f.blockWriter.SetDeadline(t)
	}

	// Return the error at connection time.
	return nil
}

// Write implements io.Writer for writing to a file in HDFS. Internally, it
// writes data to an internal buffer first, and then later out to HDFS. Because
// of this, it is important that Close is called after all data has been
// written.
func (f *FileWriter) Write(b []byte) (int, error) {
	if f.storeInDB {
		f.smallFileBuffer = append(f.smallFileBuffer, b...)
		if len(f.smallFileBuffer) <= MaxSmallFileSize {
			return len(b), nil // written successfully
		} else { // we have exceeded small file limit
			f.storeInDB = false
			_, err := f.writeInternal(f.smallFileBuffer)
			// we already acked for some data in the previous return statements
			return len(b), err
		}
	} else {
		n, err := f.writeInternal(b)
		if err != nil {
			f.lastError = err
		}
		return n, err
	}
}

func (f *FileWriter) writeInternal(b []byte) (int, error) {
	if f.blockWriter == nil {
		err := f.startNewBlock()
		if err != nil {
			return 0, err
		}
	}

	off := 0
	for off < len(b) {
		var n int
		var err error
		if f.enc != nil {
			if f.enc.stream == nil {
				f.enc.stream, err = aesCreateCTRStream(int64(f.pos), f.enc)
				if err != nil {
					return 0, err
				}
			}
			n, err = cipher.StreamWriter{S: f.enc.stream, W: f.blockWriter}.Write(b[off:])
			// If blockWriter writes less than expected bytes,
			// we must recreate stream chipher, since it's internal counter goes forward.
			if n != len(b[off:]) {
				f.enc.stream = nil
			}
		} else {
			n, err = f.blockWriter.Write(b[off:])
		}

		if n > 0 {
			off += n
			f.pos += uint64(n)
		}
		if err == transfer.ErrEndOfBlock {
			err = f.startNewBlock()
		}

		if err != nil {
			return off, err
		}
	}

	return off, nil
}

// Flush flushes any buffered data out to the datanodes. Even immediately after
// a call to Flush, it is still necessary to call Close once all data has been
// written.
func (f *FileWriter) Flush() error {
	// if we have buffered some data then we need to write it first
	if f.storeInDB {
		if len(f.smallFileBuffer) > 0 {
			_, err := f.writeInternal(f.smallFileBuffer)
			if err != nil {
				f.lastError = err
				return err
			}
			f.storeInDB = false
		}
	}

	if f.blockWriter != nil {
		err := f.blockWriter.Flush()
		if err != nil {
			f.lastError = err
			return err
		}
	}

	return nil
}

// Close closes the file, writing any remaining data out to disk and waiting
// for acknowledgements from the datanodes. It is important that Close is called
// after all data has been written.
//
// If the datanodes have acknowledged all writes but not yet to the namenode,
// it can return ErrReplicating (wrapped in an os.PathError). This indicates
// that all data has been written, but the lease is still open for the file.
// It is safe in this case to either ignore the error (and let the lease expire
// on its own) or to call Close multiple times until it completes without an
// error. The Java client, for context, always chooses to retry, with
// exponential backoff.
func (f *FileWriter) Close() error {
	err := f.closeInt()
	if err != nil {
		// if the close failed due to the DB throwing
		// OutOfExtents Exception then we retry the close
		// operation after writing the data to disk

		if f.storeInDB && len(f.smallFileBuffer) > 0 &&
			strings.Contains(err.Error(), "OutOfDBExtentsException") {

			_, err := f.writeInternal(f.smallFileBuffer)
			if err != nil {
				return err
			}
			f.storeInDB = false

			err = f.closeInt()
			if err != nil {
				return err
			}
		} else {
			return err
		}
	}
	return nil
}

func (f *FileWriter) closeInt() error {
	if f.lastError != nil {
		return f.lastError
	}

	var lastBlock *hdfs.ExtendedBlockProto = nil
	if !f.storeInDB {
		if f.blockWriter != nil {
			lastBlock = f.blockWriter.Block.GetB()
			if f.newGS != 0 {
				// The DN finalized at the bumped GS (Java protocol).
				// Send that GS on `complete` so the NN matches it
				// against the BlockInfoUC that updatePipeline already
				// updated. blockWriter.Block.B still carries the old
				// GS because BaseHeader.Block on the wire-level
				// writeBlock RPC must be the pre-bump value.
				// CloudBucket must be preserved: see comment in Append()
				// — NN's commitOrCompleteLastBlock branches on
				// isProvidedBlock(), which keys off cloudBucket.
				lastBlock = &hdfs.ExtendedBlockProto{
					PoolId:          lastBlock.PoolId,
					BlockId:         lastBlock.BlockId,
					GenerationStamp: proto.Uint64(f.newGS),
					NumBytes:        proto.Uint64(uint64(f.blockWriter.Offset)),
					CloudBucket:     lastBlock.CloudBucket,
				}
			}

			// Close the blockWriter, flushing any buffered packets.
			err := f.closeBlock()
			if err != nil {
				return err
			}
		}
	}

	completeReq := &hdfs.CompleteRequestProto{
		Src:        proto.String(f.name),
		ClientName: proto.String(f.client.namenode.ClientName),
		Last:       lastBlock,
	}

	if f.storeInDB {
		completeReq.Data = f.smallFileBuffer
	}

	completeResp := &hdfs.CompleteResponseProto{}

	sleep := time.Duration(100)
	for i := 0; i < 10; i++ {
		err := f.client.namenode.Execute("complete", completeReq, completeResp)
		if err != nil {
			return &os.PathError{Op: "create", Path: f.name, Err: interpretException(err)}
		}

		closed := *completeResp.Result

		if !closed { //retry after sleep
			time.Sleep(sleep * time.Millisecond)
			sleep *= 2
			continue
		} else {
			return nil
		}
	}

	return &os.PathError{Op: "create", Path: f.name, Err: errors.New("failed to close the file")}
}

func (f *FileWriter) startNewBlock() error {
	var previous *hdfs.ExtendedBlockProto
	if f.blockWriter != nil {
		previous = f.blockWriter.Block.GetB()

		// TODO: We don't actually need to wait for previous blocks to ack before
		// continuing.
		err := f.closeBlock()
		if err != nil {
			return err
		}
	}

	addBlockResp, err := f.addBlockWithRetry(previous)
	if err != nil {
		return &os.PathError{Op: "create", Path: f.name, Err: interpretException(err)}
	}

	block := addBlockResp.GetBlock()
	dialFunc, err := f.client.wrapDatanodeDial(
		f.client.options.DatanodeDialFunc, block.GetBlockToken())
	if err != nil {
		return err
	}

	f.blockWriter = &transfer.BlockWriter{
		ClientName:          f.client.namenode.ClientName,
		Block:               block,
		BlockSize:           f.blockSize,
		UseDatanodeHostname: f.client.options.UseDatanodeHostname,
		RemoteAccess:        f.client.options.RemoteAccess,
		DialFunc:            dialFunc,
	}

	return f.blockWriter.SetDeadline(f.deadline)
}

func (f *FileWriter) addBlockWithRetry(previous *hdfs.ExtendedBlockProto) (*hdfs.AddBlockResponseProto, error) {
	addBlockReq := &hdfs.AddBlockRequestProto{
		Src:        proto.String(f.name),
		ClientName: proto.String(f.client.namenode.ClientName),
		Previous:   previous,
	}

	addBlockResp := &hdfs.AddBlockResponseProto{}
	initDelay := time.Duration(400)
	var err error = nil

	for i := 0; i < 8; i++ { // 8 --> ~9.3 min
		err = f.client.namenode.Execute("addBlock", addBlockReq, addBlockResp)
		if err != nil && strings.Contains(err.Error(), "NotReplicatedYetException") {
			time.Sleep(initDelay * time.Millisecond)
			initDelay *= 2
		} else {
			break
		}
	}
	return addBlockResp, err
}

func (f *FileWriter) closeBlock() error {
	err := f.blockWriter.Close()
	if err != nil {
		return err
	}

	f.blockWriter = nil
	return nil
}

func (f *FileWriter) GetPos() uint64 {
	return f.pos
}
