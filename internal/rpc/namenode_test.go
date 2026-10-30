package rpc

import (
	"io"
	"net"
	"testing"
	"time"

	hadoop "github.com/colinmarc/hdfs/v2/internal/protocol/hadoop_common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
)

// answeringTransport stands in for the wire: every readResponse pops the
// next answer, a nil for success or a NamenodeError, and records the call ids
// the requests carried.
type answeringTransport struct {
	answers    []error
	requestIDs []int32
}

func (t *answeringTransport) writeRequest(w io.Writer, method string, requestID int32, req proto.Message) error {
	t.requestIDs = append(t.requestIDs, requestID)
	return nil
}

func (t *answeringTransport) readResponse(r io.Reader, method string, requestID int32, resp proto.Message) error {
	if len(t.answers) == 0 {
		return nil
	}
	answer := t.answers[0]
	t.answers = t.answers[1:]
	return answer
}

func namenodeError(exception, message string) *NamenodeError {
	return &NamenodeError{
		method:    "getFileInfo",
		code:      int(hadoop.RpcResponseHeaderProto_ERROR_APPLICATION),
		exception: exception,
		message:   message,
	}
}

// subtreeLocked is the answer as HopsFS sends it: a RetriableException whose
// message is the wrapped exception's toString and its stack trace.
func subtreeLocked() *NamenodeError {
	return namenodeError(retriableExceptionClass,
		subtreeLockedExceptionClass+": The subtree mlruns is locked by Namenode Id: 1. Active Namenodes are: [1]\n\tat io.hops.transaction.lock.INodeLock.checkSubtreeLock(INodeLock.java:278)")
}

func notStarted() *NamenodeError {
	return namenodeError(retriableExceptionClass, "NameNode still not started")
}

func repeat(answer error, n int) []error {
	answers := make([]error, n)
	for i := range answers {
		answers[i] = answer
	}
	return answers
}

// connectedWith returns a connection whose dial has already happened, so
// Execute goes straight to the transport.
func connectedWith(t *testing.T, transport transport) *NamenodeConnection {
	client, server := net.Pipe()
	t.Cleanup(func() { client.Close(); server.Close() })
	host := &namenodeHost{address: "namenode:8020"}
	return &NamenodeConnection{
		conn:      client,
		host:      host,
		hostList:  []*namenodeHost{host},
		transport: transport,
		done:      make(chan struct{}),
	}
}

func shortRetries(t *testing.T, retriable, subtreeLocked int) {
	initial, max, limit := retriableInitialBackoff, retriableMaxBackoff, retriableMaxRetries
	lockedMax, lockedLimit := subtreeLockedMaxBackoff, subtreeLockedMaxRetries
	retriableInitialBackoff, retriableMaxBackoff, retriableMaxRetries = time.Millisecond, 4*time.Millisecond, retriable
	subtreeLockedMaxBackoff, subtreeLockedMaxRetries = 4*time.Millisecond, subtreeLocked
	t.Cleanup(func() {
		retriableInitialBackoff, retriableMaxBackoff, retriableMaxRetries = initial, max, limit
		subtreeLockedMaxBackoff, subtreeLockedMaxRetries = lockedMax, lockedLimit
	})
}

func execute(c *NamenodeConnection) error {
	return c.Execute("getFileInfo", &hadoop.RpcRequestHeaderProto{}, &hadoop.RpcResponseHeaderProto{})
}

func TestExecuteRetriesASubtreeLockUntilItSucceeds(t *testing.T) {
	shortRetries(t, 1, 10)
	transport := &answeringTransport{answers: append(repeat(subtreeLocked(), 3), nil)}
	c := connectedWith(t, transport)

	require.NoError(t, execute(c))
	assert.Equal(t, []int32{1, 2, 3, 4}, transport.requestIDs, "every attempt carries a new call id")
	assert.NotNil(t, c.conn, "the connection stays open across the retries")
	assert.Nil(t, c.host.lastError, "a retriable answer is not a host failure")
}

func TestExecuteGivesUpOnASubtreeLockAfterItsRetries(t *testing.T) {
	shortRetries(t, 1, 3)
	transport := &answeringTransport{answers: repeat(subtreeLocked(), 100)}
	c := connectedWith(t, transport)

	err := execute(c)

	require.Error(t, err)
	nerr, ok := err.(*NamenodeError)
	require.True(t, ok, "the namenode's own exception is returned, not a wrapped one")
	assert.Equal(t, retriableExceptionClass, nerr.exception)
	assert.Len(t, transport.requestIDs, 4, "the first attempt plus the subtree lock's three retries")
	assert.NotNil(t, c.conn)
}

func TestExecuteGivesOtherRetriableExceptionsTheShorterBudget(t *testing.T) {
	shortRetries(t, 2, 5)
	transport := &answeringTransport{answers: repeat(notStarted(), 100)}
	c := connectedWith(t, transport)

	require.Error(t, execute(c))
	assert.Len(t, transport.requestIDs, 3, "the first attempt plus two retries, not the subtree lock's five")
}

func TestExecuteReturnsOtherNamenodeExceptionsAtOnce(t *testing.T) {
	shortRetries(t, 10, 10)
	transport := &answeringTransport{answers: []error{
		namenodeError("java.io.FileNotFoundException", "File does not exist: /a"),
		nil,
	}}
	c := connectedWith(t, transport)

	require.Error(t, execute(c))
	assert.Equal(t, []int32{1}, transport.requestIDs, "no retry")
	assert.NotNil(t, c.conn, "the connection is kept: the namenode answered")
}
