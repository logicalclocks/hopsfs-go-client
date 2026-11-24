package transfer

import (
	"context"
	"encoding/binary"
	"errors"
	"io"
	"net"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	hdfs "github.com/colinmarc/hdfs/v2/internal/protocol/hadoop_hdfs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
)

// trackedConn is the client end of a connection to the fake DataNode. It
// records whether the client closed it, and can fail SetDeadline.
type trackedConn struct {
	net.Conn
	closed          atomic.Bool
	failWrite       bool
	failSetDeadline bool
}

func (c *trackedConn) Write(b []byte) (int, error) {
	if c.failWrite {
		return 0, errors.New("write failed")
	}
	return c.Conn.Write(b)
}

func (c *trackedConn) Close() error {
	c.closed.Store(true)
	return c.Conn.Close()
}

func (c *trackedConn) SetDeadline(t time.Time) error {
	if c.failSetDeadline {
		return errors.New("set deadline failed")
	}
	return c.Conn.SetDeadline(t)
}

// datanodeReply is how the fake DataNode answers one read request.
type datanodeReply func(conn net.Conn, req *hdfs.OpReadBlockProto)

// fakeDatanode answers the read request on the nth dial with replies[n]; the
// last reply repeats. Its dial method is a BlockReader.DialFunc.
type fakeDatanode struct {
	replies         []datanodeReply
	failWrite       bool
	failSetDeadline bool

	mu    sync.Mutex
	addrs []string
	conns []*trackedConn
	reqs  []*hdfs.OpReadBlockProto
}

func (d *fakeDatanode) dial(ctx context.Context, network, addr string) (net.Conn, error) {
	client, server := net.Pipe()
	conn := &trackedConn{Conn: client, failWrite: d.failWrite,
		failSetDeadline: d.failSetDeadline}

	d.mu.Lock()
	i := len(d.addrs)
	if i >= len(d.replies) {
		i = len(d.replies) - 1
	}
	reply := d.replies[i]
	d.addrs = append(d.addrs, addr)
	d.conns = append(d.conns, conn)
	d.mu.Unlock()

	go func() {
		defer server.Close()
		header := make([]byte, 3)
		if _, err := io.ReadFull(server, header); err != nil {
			return
		}
		req := &hdfs.OpReadBlockProto{}
		if err := readPrefixedMessage(server, req); err != nil {
			return
		}
		d.mu.Lock()
		d.reqs = append(d.reqs, req)
		d.mu.Unlock()
		reply(server, req)
	}()
	return conn, nil
}

func (d *fakeDatanode) dials() []string {
	d.mu.Lock()
	defer d.mu.Unlock()
	return append([]string(nil), d.addrs...)
}

func writeResponse(conn net.Conn, resp *hdfs.BlockOpResponseProto) {
	msg, _ := makePrefixedMessage(resp)
	conn.Write(msg)
}

func statusReply(status hdfs.Status, message string) datanodeReply {
	return func(conn net.Conn, _ *hdfs.OpReadBlockProto) {
		writeResponse(conn, &hdfs.BlockOpResponseProto{
			Status:  status.Enum(),
			Message: proto.String(message),
		})
	}
}

// truncatedReply sends one byte of a response and hangs up.
func truncatedReply(conn net.Conn, _ *hdfs.OpReadBlockProto) {
	conn.Write([]byte{0x10})
}

// dataReply serves data from the requested offset in one packet, without
// checksums.
func dataReply(data []byte) datanodeReply {
	return checksumReply(hdfs.ChecksumTypeProto_CHECKSUM_NULL, data)
}

func checksumReply(checksumType hdfs.ChecksumTypeProto, data []byte) datanodeReply {
	return func(conn net.Conn, req *hdfs.OpReadBlockProto) {
		off := req.GetOffset()
		writeResponse(conn, &hdfs.BlockOpResponseProto{
			Status: hdfs.Status_SUCCESS.Enum(),
			ReadOpChecksumInfo: &hdfs.ReadOpChecksumInfoProto{
				Checksum: &hdfs.ChecksumProto{
					Type:             checksumType.Enum(),
					BytesPerChecksum: proto.Uint32(512),
				},
				ChunkOffset: proto.Uint64(off),
			},
		})

		payload := data[off:]
		header, _ := proto.Marshal(&hdfs.PacketHeaderProto{
			OffsetInBlock:     proto.Int64(int64(off)),
			Seqno:             proto.Int64(0),
			LastPacketInBlock: proto.Bool(true),
			DataLen:           proto.Int32(int32(len(payload))),
		})
		prefix := make([]byte, 6)
		binary.BigEndian.PutUint32(prefix, uint32(4+len(payload)))
		binary.BigEndian.PutUint16(prefix[4:], uint16(len(header)))
		conn.Write(append(append(prefix, header...), payload...))
	}
}

// readBlock is a located block of numBytes bytes with one location per port.
// A non-empty bucket makes it a cloud block.
func readBlock(bucket string, numBytes uint64, ports ...uint32) *hdfs.LocatedBlockProto {
	locs := []*hdfs.DatanodeInfoProto{}
	for _, port := range ports {
		locs = append(locs, &hdfs.DatanodeInfoProto{Id: &hdfs.DatanodeIDProto{
			IpAddr:       proto.String("127.0.0.1"),
			HostName:     proto.String("localhost"),
			DatanodeUuid: proto.String("dn"),
			XferPort:     proto.Uint32(port),
			InfoPort:     proto.Uint32(50075),
			IpcPort:      proto.Uint32(50020),
		}})
	}
	return &hdfs.LocatedBlockProto{
		B: &hdfs.ExtendedBlockProto{
			PoolId:          proto.String("bp"),
			BlockId:         proto.Uint64(1),
			GenerationStamp: proto.Uint64(1001),
			NumBytes:        proto.Uint64(numBytes),
			CloudBucket:     proto.String(bucket),
		},
		Offset: proto.Uint64(0),
		Locs:   locs,
	}
}

// Every failed read handshake closes its connection, and a failure after
// the stream was built leaves the reader unconnected.
func TestReadHandshakeFailureClosesConnection(t *testing.T) {
	data := []byte("data")
	cases := map[string]*fakeDatanode{
		"request write fails": {replies: []datanodeReply{dataReply(data)}, failWrite: true},
		"error status":        {replies: []datanodeReply{statusReply(hdfs.Status_ERROR, "failed")}},
		"truncated response":  {replies: []datanodeReply{truncatedReply}},
		"unknown checksum":    {replies: []datanodeReply{checksumReply(hdfs.ChecksumTypeProto(7), data)}},
		"set deadline fails":  {replies: []datanodeReply{dataReply(data)}, failSetDeadline: true},
	}

	for name, dn := range cases {
		t.Run(name, func(t *testing.T) {
			br := &BlockReader{ClientName: "cl", Block: readBlock("", uint64(len(data)), 50010), DialFunc: dn.dial}
			_, err := br.Read(make([]byte, len(data)))
			require.Error(t, err)
			require.Len(t, dn.conns, 1)
			assert.True(t, dn.conns[0].closed.Load())
			assert.Nil(t, br.conn)
			assert.Nil(t, br.stream)
		})
	}
}
