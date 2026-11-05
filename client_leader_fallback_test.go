package hdfs

import (
	"context"
	"encoding/binary"
	"io"
	"net"
	"os"
	"sync"
	"syscall"
	"testing"

	hadoop "github.com/colinmarc/hdfs/v2/internal/protocol/hadoop_common"
	hdfs "github.com/colinmarc/hdfs/v2/internal/protocol/hadoop_hdfs"
	"github.com/colinmarc/hdfs/v2/internal/rpc"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
)

const (
	rpcPreambleLength  = 7 // "hrpc", version, service class, auth protocol
	errorApplicationRC = hadoop.RpcResponseHeaderProto_ERROR_APPLICATION
)

// fakeReply is what a fakeNamenode answers to one rpc method: an exception
// class with an optional message, or a response message.
type fakeReply struct {
	exception string
	message   string
	resp      proto.Message
}

// notALeaderRefusals holds both forms of the refusal unwrapHopsFSException
// accepts: the exception class itself, and an IOException whose message
// starts with it.
var notALeaderRefusals = map[string]fakeReply{
	"class":   {exception: notALeaderException},
	"wrapped": {exception: javaIOException, message: notALeaderException + ": not the leader"},
}

// fakeNamenode speaks enough Hadoop RPC over net.Pipe to answer a client's
// calls from a fixed table and record the methods it was asked.
type fakeNamenode struct {
	t       *testing.T
	replies map[string]fakeReply

	mu    sync.Mutex
	calls []string
}

func (n *fakeNamenode) dial(ctx context.Context, network, address string) (net.Conn, error) {
	client, server := net.Pipe()
	go n.serve(server)
	return client, nil
}

func (n *fakeNamenode) serve(conn net.Conn) {
	defer conn.Close()
	if _, err := io.ReadFull(conn, make([]byte, rpcPreambleLength)); err != nil {
		return
	}

	for {
		var length uint32
		if err := binary.Read(conn, binary.BigEndian, &length); err != nil {
			return
		}
		packet := make([]byte, length)
		if _, err := io.ReadFull(conn, packet); err != nil {
			return
		}

		header := &hadoop.RpcRequestHeaderProto{}
		rest, ok := n.unmarshalPrefixed(packet, header)
		if !ok {
			return
		}
		// The handshake's connection context has a negative call id and
		// gets no reply.
		if header.GetCallId() < 0 {
			continue
		}
		request := &hadoop.RequestHeaderProto{}
		if _, ok := n.unmarshalPrefixed(rest, request); !ok {
			return
		}
		method := request.GetMethodName()
		n.mu.Lock()
		n.calls = append(n.calls, method)
		n.mu.Unlock()

		reply, known := n.replies[method]
		if !known {
			n.t.Errorf("fake namenode asked for %s, which it does not answer", method)
			return
		}
		rrh := &hadoop.RpcResponseHeaderProto{
			CallId: proto.Uint32(uint32(header.GetCallId())),
			Status: hadoop.RpcResponseHeaderProto_SUCCESS.Enum(),
		}
		msgs := []proto.Message{rrh}
		if reply.exception != "" {
			message := reply.message
			if message == "" {
				message = reply.exception + ": refused by the fake namenode"
			}
			rrh.Status = hadoop.RpcResponseHeaderProto_ERROR.Enum()
			rrh.ErrorDetail = errorApplicationRC.Enum()
			rrh.ExceptionClassName = proto.String(reply.exception)
			rrh.ErrorMsg = proto.String(message)
		} else {
			msgs = append(msgs, reply.resp)
		}
		if _, err := conn.Write(n.packet(msgs...)); err != nil {
			return
		}
	}
}

func (n *fakeNamenode) unmarshalPrefixed(b []byte, msg proto.Message) ([]byte, bool) {
	length, read := binary.Uvarint(b)
	if read <= 0 || length > uint64(len(b)-read) {
		n.t.Errorf("fake namenode: malformed request packet")
		return nil, false
	}
	if err := proto.Unmarshal(b[read:read+int(length)], msg); err != nil {
		n.t.Errorf("fake namenode: %v", err)
		return nil, false
	}
	return b[read+int(length):], true
}

func (n *fakeNamenode) packet(msgs ...proto.Message) []byte {
	body := []byte{}
	for _, msg := range msgs {
		b, err := proto.Marshal(msg)
		if err != nil {
			n.t.Errorf("fake namenode: %v", err)
			return nil
		}
		prefix := make([]byte, binary.MaxVarintLen64)
		body = append(body, prefix[:binary.PutUvarint(prefix, uint64(len(b)))]...)
		body = append(body, b...)
	}
	packet := make([]byte, 4, 4+len(body))
	binary.BigEndian.PutUint32(packet, uint32(len(body)))
	return append(packet, body...)
}

func (n *fakeNamenode) methods() []string {
	n.mu.Lock()
	defer n.mu.Unlock()
	return append([]string{}, n.calls...)
}

func (n *fakeNamenode) connect(t *testing.T) *rpc.NamenodeConnection {
	conn, err := rpc.NewNamenodeConnection(rpc.NamenodeConnectionOptions{
		Addresses: []string{"namenode:8020"},
		User:      "test",
		DialFunc:  n.dial,
	})
	require.NoError(t, err)
	return conn
}

func existingFile() *hdfs.GetFileInfoResponseProto {
	return &hdfs.GetFileInfoResponseProto{Fs: &hdfs.HdfsFileStatusProto{
		FileType:         hdfs.HdfsFileStatusProto_IS_FILE.Enum(),
		Path:             []byte("f"),
		Length:           proto.Uint64(0),
		Permission:       &hdfs.FsPermissionProto{Perm: proto.Uint32(0644)},
		Owner:            proto.String("test"),
		Group:            proto.String("test"),
		ModificationTime: proto.Uint64(0),
		AccessTime:       proto.Uint64(0),
	}}
}

// twoNamenodes returns a client whose own namenode answers from nonLeader
// and whose leader connection answers from leader.
func twoNamenodes(t *testing.T, nonLeader, leader map[string]fakeReply) (*Client, *fakeNamenode, *fakeNamenode) {
	first := &fakeNamenode{t: t, replies: nonLeader}
	second := &fakeNamenode{t: t, replies: leader}
	client := &Client{namenode: first.connect(t), leaderNamenode: second.connect(t)}
	t.Cleanup(func() { client.Close() })
	return client, first, second
}

func TestRemoveRepeatsOnTheLeaderWhenTheNamenodeIsNotIt(t *testing.T) {
	for form, refusal := range notALeaderRefusals {
		t.Run(form, func(t *testing.T) {
			client, nonLeader, leader := twoNamenodes(t,
				map[string]fakeReply{
					"getFileInfo": {resp: existingFile()},
					"delete":      refusal,
				},
				map[string]fakeReply{
					"delete": {resp: &hdfs.DeleteResponseProto{Result: proto.Bool(true)}},
				})

			require.NoError(t, client.Remove("/d/f"))
			assert.Equal(t, []string{"getFileInfo", "delete"}, nonLeader.methods())
			assert.Equal(t, []string{"delete"}, leader.methods())
		})
	}
}

func TestRenameRepeatsOnTheLeaderWhenTheNamenodeIsNotIt(t *testing.T) {
	renames := map[string]func(*Client) error{
		"Rename":  func(c *Client) error { return c.Rename("/d/f", "/d/g") },
		"Rename2": func(c *Client) error { return c.Rename2("/d/f", "/d/g", RENAME_OPTION_NONE) },
	}
	for name, rename := range renames {
		for form, refusal := range notALeaderRefusals {
			t.Run(name+"/"+form, func(t *testing.T) {
				client, nonLeader, leader := twoNamenodes(t,
					map[string]fakeReply{
						"getFileInfo": {resp: &hdfs.GetFileInfoResponseProto{}},
						"rename2":     refusal,
					},
					map[string]fakeReply{
						"rename2": {resp: &hdfs.Rename2ResponseProto{}},
					})

				require.NoError(t, rename(client))
				assert.Equal(t, []string{"getFileInfo", "rename2"}, nonLeader.methods())
				assert.Equal(t, []string{"rename2"}, leader.methods())
			})
		}
	}
}

func TestRemoveLeavesTheLeaderAloneOnAnyOtherError(t *testing.T) {
	client, nonLeader, leader := twoNamenodes(t,
		map[string]fakeReply{
			"getFileInfo": {resp: existingFile()},
			"delete":      {exception: fileNotFoundException},
		},
		map[string]fakeReply{})

	err := client.Remove("/d/f")
	assert.True(t, os.IsNotExist(err), "got %v", err)
	assert.Equal(t, []string{"getFileInfo", "delete"}, nonLeader.methods())
	assert.Empty(t, leader.methods())
}

// EAGAIN is what the mount retries on, rebuilding its client and so finding
// the new leader.
func TestRemoveReturnsEAGAINWhenTheLeaderRefusesToo(t *testing.T) {
	for form, refusal := range notALeaderRefusals {
		t.Run(form, func(t *testing.T) {
			client, nonLeader, leader := twoNamenodes(t,
				map[string]fakeReply{
					"getFileInfo": {resp: existingFile()},
					"delete":      refusal,
				},
				map[string]fakeReply{
					"delete": refusal,
				})

			err := client.Remove("/d/f")
			var pathErr *os.PathError
			require.ErrorAs(t, err, &pathErr)
			assert.Equal(t, syscall.EAGAIN, pathErr.Err)
			assert.Equal(t, []string{"getFileInfo", "delete"}, nonLeader.methods())
			assert.Equal(t, []string{"delete"}, leader.methods())
		})
	}
}

func TestRemoveStaysOnTheNamenodeThatServesIt(t *testing.T) {
	client, nonLeader, leader := twoNamenodes(t,
		map[string]fakeReply{
			"getFileInfo": {resp: existingFile()},
			"delete":      {resp: &hdfs.DeleteResponseProto{Result: proto.Bool(true)}},
		},
		map[string]fakeReply{})

	require.NoError(t, client.Remove("/d/f"))
	assert.Equal(t, []string{"getFileInfo", "delete"}, nonLeader.methods())
	assert.Empty(t, leader.methods())
}
