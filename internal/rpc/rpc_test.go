package rpc

import (
	"bytes"
	"context"
	"errors"
	"net"
	"runtime"
	"sync"
	"testing"
	"time"

	hadoop "github.com/colinmarc/hdfs/v2/internal/protocol/hadoop_common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
)

func allocatedDuring(f func()) uint64 {
	var before, after runtime.MemStats
	runtime.ReadMemStats(&before)
	f()
	runtime.ReadMemStats(&after)
	return after.TotalAlloc - before.TotalAlloc
}

func TestReadRPCPacketRefusesATLSRecordWithoutAllocatingIt(t *testing.T) {
	// A TLS alert record: content type 21, version 3.3, length 2, then the alert.
	alert := []byte{0x15, 0x03, 0x03, 0x00, 0x02, 0x02, 0x28}
	var err error
	allocated := allocatedDuring(func() {
		err = readRPCPacket(bytes.NewReader(alert), 0, &hadoop.RpcResponseHeaderProto{})
	})

	require.Error(t, err)
	assert.True(t, errors.Is(err, errInvalidResponse))
	assert.Contains(t, err.Error(), "TLS")
	assert.Less(t, allocated, uint64(1<<20))
}

func TestReadRPCPacketRefusesAnOversizedResponse(t *testing.T) {
	header := []byte{0x7f, 0xff, 0xff, 0xff}
	var err error
	allocated := allocatedDuring(func() {
		err = readRPCPacket(bytes.NewReader(header), 0, &hadoop.RpcResponseHeaderProto{})
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "exceeds")
	assert.Less(t, allocated, uint64(1<<20))
}

func TestReadRPCPacketReadsAResponseWithinTheLimit(t *testing.T) {
	sent := &hadoop.RpcResponseHeaderProto{
		CallId: proto.Uint32(7),
		Status: hadoop.RpcResponseHeaderProto_SUCCESS.Enum(),
	}
	packet, err := makeRPCPacket(sent)
	require.NoError(t, err)

	received := &hadoop.RpcResponseHeaderProto{}
	require.NoError(t, readRPCPacket(bytes.NewReader(packet), 0, received))
	assert.Equal(t, uint32(7), received.GetCallId())
}

func TestReadRPCPacketHonoursAConfiguredLimit(t *testing.T) {
	sent := &hadoop.RpcResponseHeaderProto{
		CallId: proto.Uint32(7),
		Status: hadoop.RpcResponseHeaderProto_SUCCESS.Enum(),
	}
	packet, err := makeRPCPacket(sent)
	require.NoError(t, err)

	err = readRPCPacket(bytes.NewReader(packet), 2, &hadoop.RpcResponseHeaderProto{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "exceeds the 2 byte limit")

	// A frame over the default bound is accepted when the connection raises it; the read then fails
	// only because the test supplies no body.
	header := []byte{0x08, 0x00, 0x00, 0x01}
	err = readRPCPacket(bytes.NewReader(header), DefaultMaxResponseLength*2, &hadoop.RpcResponseHeaderProto{})
	require.Error(t, err)
	assert.NotContains(t, err.Error(), "exceeds")
}

// pipeNamenode stands in for a namenode: it accepts connections over net.Pipe and records how many
// were dialled and how many bytes the client wrote after its handshake.
type pipeNamenode struct {
	mu      sync.Mutex
	dials   int
	written int
	servers []net.Conn
}

func (n *pipeNamenode) dial(ctx context.Context, network, address string) (net.Conn, error) {
	client, server := net.Pipe()
	n.mu.Lock()
	n.dials++
	n.servers = append(n.servers, server)
	n.mu.Unlock()
	go func() {
		buf := make([]byte, 4096)
		for {
			read, err := server.Read(buf)
			n.mu.Lock()
			n.written += read
			n.mu.Unlock()
			if err != nil {
				return
			}
		}
	}()
	return client, nil
}

func (n *pipeNamenode) counts() (int, int) {
	n.mu.Lock()
	defer n.mu.Unlock()
	return n.dials, n.written
}

func (n *pipeNamenode) hangUp() {
	n.mu.Lock()
	defer n.mu.Unlock()
	for _, server := range n.servers {
		server.Close()
	}
}

func TestLeasesAreRenewedOnlyWhileAFileIsOpenForWriting(t *testing.T) {
	interval := leaseRenewInterval
	leaseRenewInterval = 10 * time.Millisecond
	defer func() { leaseRenewInterval = interval }()

	namenode := &pipeNamenode{}
	conn, err := NewNamenodeConnection(NamenodeConnectionOptions{
		Addresses: []string{"namenode:8020"},
		User:      "test",
		DialFunc:  namenode.dial,
	})
	require.NoError(t, err)
	defer conn.Close()

	// Let the handshake drain before taking the baseline.
	time.Sleep(50 * time.Millisecond)
	dials, afterHandshake := namenode.counts()
	assert.Equal(t, 1, dials)

	time.Sleep(10 * leaseRenewInterval)
	_, idle := namenode.counts()
	assert.Equal(t, afterHandshake, idle, "an idle client renewed a lease it does not hold")

	conn.AcquireLease()
	assert.Eventually(t, func() bool {
		_, written := namenode.counts()
		return written > afterHandshake
	}, time.Second, leaseRenewInterval, "an open writer's lease was not renewed")

	// The test namenode never answers, so release the renewal blocked on its response.
	conn.ReleaseLease()
	namenode.hangUp()
}
