package hdfs

import (
	"context"
	"encoding/binary"
	"errors"
	"io"
	"net"
	"os"
	"sync"
	"testing"
	"time"

	hadoop "github.com/colinmarc/hdfs/v2/internal/protocol/hadoop_common"
	hdfs "github.com/colinmarc/hdfs/v2/internal/protocol/hadoop_hdfs"
	"github.com/colinmarc/hdfs/v2/internal/rpc"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
)

// What a tcpNamenode does with the next request it reads.
const (
	answer     = ""
	hangUp     = "hang up"     // read the request, close with no answer
	halfAnswer = "half answer" // read the request, send part of the answer, close
)

// tcpNamenode serves the fakeNamenode replies on a loopback TCP socket, so the
// client meets a real kernel's FIN and RST, and can end its open connections
// the way a middlebox or an idle timeout does.
type tcpNamenode struct {
	fake *fakeNamenode
	l    net.Listener

	mu      sync.Mutex
	conns   []*net.TCPConn
	accepts int
	// The next request for method gets mode instead of an answer.
	method, mode string
}

func newTCPNamenode(t *testing.T) *tcpNamenode {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	n := &tcpNamenode{l: l, fake: &fakeNamenode{t: t, replies: map[string]fakeReply{
		"getFileInfo": {resp: existingFile()},
		"mkdirs":      {resp: &hdfs.MkdirsResponseProto{Result: proto.Bool(true)}},
	}}}
	t.Cleanup(func() { l.Close() })
	go func() {
		for {
			c, err := l.Accept()
			if err != nil {
				return
			}
			n.mu.Lock()
			n.conns = append(n.conns, c.(*net.TCPConn))
			n.accepts++
			n.mu.Unlock()
			go n.serve(c)
		}
	}()
	return n
}

func (n *tcpNamenode) serve(conn net.Conn) {
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
		rest, ok := n.fake.unmarshalPrefixed(packet, header)
		if !ok || header.GetCallId() < 0 {
			if !ok {
				return
			}
			continue
		}
		request := &hadoop.RequestHeaderProto{}
		body, ok := n.fake.unmarshalPrefixed(rest, request)
		if !ok {
			return
		}
		method := request.GetMethodName()
		n.fake.mu.Lock()
		n.fake.calls = append(n.fake.calls, method)
		n.fake.mu.Unlock()

		n.mu.Lock()
		mode := answer
		if method == n.method {
			mode, n.method = n.mode, ""
		}
		n.mu.Unlock()

		reply := n.fake.replies[method]
		if method == "getFileInfo" {
			// Only /f exists.
			req := &hdfs.GetFileInfoRequestProto{}
			if _, ok := n.fake.unmarshalPrefixed(body, req); !ok {
				return
			}
			if req.GetSrc() != "/f" {
				reply.resp = &hdfs.GetFileInfoResponseProto{}
			}
		}
		rrh := &hadoop.RpcResponseHeaderProto{
			CallId: proto.Uint32(uint32(header.GetCallId())),
			Status: hadoop.RpcResponseHeaderProto_SUCCESS.Enum(),
		}
		out := n.fake.packet(rrh, reply.resp)
		switch mode {
		case hangUp:
			return
		case halfAnswer:
			conn.Write(out[:len(out)/2])
			return
		}
		if _, err := conn.Write(out); err != nil {
			return
		}
	}
}

// drop ends every open connection, with a FIN or, if reset, a RST.
func (n *tcpNamenode) drop(reset bool) {
	n.mu.Lock()
	defer n.mu.Unlock()
	for _, c := range n.conns {
		if reset {
			c.SetLinger(0)
		}
		c.Close()
	}
	n.conns = nil
}

func (n *tcpNamenode) misbehaveOnNext(method, mode string) {
	n.mu.Lock()
	n.method, n.mode = method, mode
	n.mu.Unlock()
}

func (n *tcpNamenode) dials() int {
	n.mu.Lock()
	defer n.mu.Unlock()
	return n.accepts
}

func (n *tcpNamenode) client(t *testing.T) *Client {
	conn, err := rpc.NewNamenodeConnection(rpc.NamenodeConnectionOptions{
		Addresses:   []string{n.l.Addr().String()},
		User:        "test",
		RetryBudget: -1, // what the mount's -namenodeRetryBudget=0s sets
	})
	require.NoError(t, err)
	c := &Client{namenode: conn, leaderNamenode: conn}
	// One connection in both roles: Client.Close would close it twice.
	t.Cleanup(func() { conn.Close() })
	return c
}

// idleClient returns a client whose one connection served a call and then
// sat idle, as a mount's pooled connection does between uses.
func idleClient(t *testing.T, n *tcpNamenode) *Client {
	c := n.client(t)
	_, err := c.Stat("/f")
	require.NoError(t, err)
	require.Equal(t, 1, n.dials())
	return c
}

// waitClosed gives the kernel time to deliver the close to the client side.
func waitClosed() { time.Sleep(100 * time.Millisecond) }

func requireConnectionError(t *testing.T, err error, sent bool) {
	t.Helper()
	var ce *ConnectionError
	require.True(t, errors.As(err, &ce), "want a ConnectionError, got %T %v", err, err)
	require.Equal(t, sent, ce.Sent, "Sent of %v", err)
	var pe *os.PathError
	if errors.As(err, &pe) {
		require.NotEqual(t, io.EOF, pe.Err, "a connection failure must not read as end of file")
	}
}

func TestExecuteRedialsAConnectionClosedWhileIdle(t *testing.T) {
	for name, reset := range map[string]bool{"FIN": false, "RST": true} {
		t.Run(name, func(t *testing.T) {
			n := newTCPNamenode(t)
			c := idleClient(t, n)

			n.drop(reset)
			waitClosed()

			// Both calls succeed at once: the dead connection is found before
			// the request is written, so nothing waits out the host backoff.
			for i := 0; i < 2; i++ {
				_, err := c.Stat("/f")
				require.NoError(t, err)
			}
			require.Equal(t, 2, n.dials())
			require.Equal(t, []string{"getFileInfo", "getFileInfo", "getFileInfo"}, n.fake.methods())
		})
	}
}

func TestExecuteRedialsAChangeOnAConnectionClosedWhileIdle(t *testing.T) {
	n := newTCPNamenode(t)
	c := idleClient(t, n)

	n.drop(false)
	waitClosed()

	require.NoError(t, c.Mkdir("/d", 0755))
	require.Equal(t, []string{"getFileInfo", "getFileInfo", "mkdirs"}, n.fake.methods())
	require.Equal(t, 2, n.dials())
}

func TestExecuteResendsAReadOnceWhenTheConnectionDiesBeforeTheAnswer(t *testing.T) {
	n := newTCPNamenode(t)
	c := idleClient(t, n)

	n.misbehaveOnNext("getFileInfo", hangUp)
	_, err := c.Stat("/f")
	require.NoError(t, err)
	require.Equal(t, []string{"getFileInfo", "getFileInfo", "getFileInfo"}, n.fake.methods())
	require.Equal(t, 2, n.dials())
}

func TestExecuteDoesNotResendAChangeThatWasSent(t *testing.T) {
	n := newTCPNamenode(t)
	c := idleClient(t, n)

	n.misbehaveOnNext("mkdirs", hangUp)
	err := c.Mkdir("/d", 0755)
	requireConnectionError(t, err, true)
	require.Equal(t, []string{"getFileInfo", "getFileInfo", "mkdirs"}, n.fake.methods())
	require.Equal(t, 1, n.dials())
}

func TestExecuteDoesNotResendAfterResponseBytes(t *testing.T) {
	n := newTCPNamenode(t)
	c := idleClient(t, n)

	n.misbehaveOnNext("getFileInfo", halfAnswer)
	_, err := c.Stat("/f")
	requireConnectionError(t, err, true)
	require.Equal(t, []string{"getFileInfo", "getFileInfo"}, n.fake.methods())
	require.Equal(t, 1, n.dials())
}

func TestExecuteReturnsTheErrorWhenTheRedialFails(t *testing.T) {
	n := newTCPNamenode(t)
	c := idleClient(t, n)

	n.l.Close()
	n.drop(false)
	waitClosed()

	_, err := c.Stat("/f")
	requireConnectionError(t, err, false)
	err = c.Mkdir("/d", 0755)
	requireConnectionError(t, err, false)
	require.Equal(t, []string{"getFileInfo"}, n.fake.methods())
}

// A connection the client cannot peek (here a pipe) shows a close only when
// the request is written; the write fails, so the request was not sent.
func TestExecuteRedialsWhenTheWriteOnAnIdleConnectionFails(t *testing.T) {
	fake := &fakeNamenode{t: t, replies: map[string]fakeReply{
		"getFileInfo": {resp: &hdfs.GetFileInfoResponseProto{}},
		"mkdirs":      {resp: &hdfs.MkdirsResponseProto{Result: proto.Bool(true)}},
	}}
	var mu sync.Mutex
	var servers []net.Conn
	conn := connectWith(t, func(ctx context.Context, network, address string) (net.Conn, error) {
		client, server := net.Pipe()
		mu.Lock()
		servers = append(servers, server)
		mu.Unlock()
		go fake.serve(server)
		return client, nil
	})
	c := &Client{namenode: conn, leaderNamenode: conn}
	// One connection in both roles: Client.Close would close it twice.
	t.Cleanup(func() { conn.Close() })
	require.NoError(t, c.Mkdir("/a", 0755))

	mu.Lock()
	servers[0].Close()
	mu.Unlock()

	require.NoError(t, c.Mkdir("/b", 0755))
	require.Equal(t, []string{"getFileInfo", "mkdirs", "getFileInfo", "mkdirs"}, fake.methods())
	mu.Lock()
	require.Len(t, servers, 2)
	mu.Unlock()
}
