package hdfs

import (
	"context"
	"errors"
	"net"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// recordingConn records whether the client closed it. A closeErr makes the
// close fail, as a TLS close_notify to a dead peer does.
type recordingConn struct {
	net.Conn
	closeErr error
	closed   atomic.Bool
}

func (c *recordingConn) Close() error {
	c.closed.Store(true)
	_ = c.Conn.Close()
	return c.closeErr
}

func dialRecording(n *fakeNamenode, conn *recordingConn) func(ctx context.Context, network, address string) (net.Conn, error) {
	return func(ctx context.Context, network, address string) (net.Conn, error) {
		c, err := n.dial(ctx, network, address)
		conn.Conn = c
		return conn, err
	}
}

// The mount builds a new client on every retry, so a namenode connection left
// open here leaks a socket and a goroutine per attempt while the leader is
// unreachable.
func TestNewClientClosesTheNamenodeWhenTheLeaderIsUnreachable(t *testing.T) {
	nnConn := &recordingConn{}
	dialNamenode := dialRecording(&fakeNamenode{t: t}, nnConn)
	dial := func(ctx context.Context, network, address string) (net.Conn, error) {
		if address == "leader:8020" {
			return nil, errors.New("connection refused")
		}
		return dialNamenode(ctx, network, address)
	}

	_, err := newClientInt(ClientOptions{User: "test", NamenodeDialFunc: dial},
		[]string{"namenode:8020"}, "leader:8020")
	require.Error(t, err)
	assert.True(t, nnConn.closed.Load())
}

func TestCloseClosesTheLeaderWhenClosingTheNamenodeFails(t *testing.T) {
	nnConn := &recordingConn{closeErr: errors.New("close_notify failed")}
	leaderConn := &recordingConn{}
	client := &Client{
		namenode:       connectWith(t, dialRecording(&fakeNamenode{t: t}, nnConn)),
		leaderNamenode: connectWith(t, dialRecording(&fakeNamenode{t: t}, leaderConn)),
	}

	assert.Error(t, client.Close())
	assert.True(t, leaderConn.closed.Load())
}
