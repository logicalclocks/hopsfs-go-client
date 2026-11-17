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
// and epochs the requests carried and when they were sent. When answered is
// set, each answer is also announced on it; onAnswer runs before each answer.
type answeringTransport struct {
	answers      []error
	requestIDs   []int32
	epochs       []int64
	requestTimes []time.Time
	answered     chan struct{}
	onAnswer     func()
}

func (t *answeringTransport) writeRequest(w io.Writer, method string, requestID int32, epoch int64, req proto.Message) error {
	t.requestIDs = append(t.requestIDs, requestID)
	t.epochs = append(t.epochs, epoch)
	t.requestTimes = append(t.requestTimes, time.Now())
	return nil
}

func (t *answeringTransport) readResponse(r io.Reader, method string, requestID int32, resp proto.Message) error {
	if t.onAnswer != nil {
		t.onAnswer()
	}
	if t.answered != nil {
		t.answered <- struct{}{}
	}
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
	epoch, reportTime := serverReportedEpoch, epochReportTime
	t.Cleanup(func() { SetEpoch(reportTime, epoch) })
	SetEpoch(time.Now(), 1_700_000_000_000)
	transport := &answeringTransport{answers: append(repeat(subtreeLocked(), 3), nil)}
	// Real retries are seconds apart, so the namenode's epoch moves on between them.
	transport.onAnswer = func() { SetEpoch(time.Now(), serverReportedEpoch+5000) }
	c := connectedWith(t, transport)

	require.NoError(t, execute(c))
	// The namenode's retry cache finds a call by client id, call id and epoch.
	assert.Equal(t, []int32{1, 1, 1, 1}, transport.requestIDs, "every retry carries the request's call id")
	assert.Equal(t, []int64{1_700_000_000, 1_700_000_000, 1_700_000_000, 1_700_000_000}, transport.epochs,
		"every retry carries the request's epoch")
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

// After a subtree lock's long waits, another RetriableException waits no
// longer than its own cap, and each kind keeps its own count: three lock
// waits do not use up the two retries the other kind is allowed, nor the
// reverse.
func TestExecuteGivesEachRetriableKindItsOwnCapAndCount(t *testing.T) {
	shortRetries(t, 2, 3)
	retriableInitialBackoff, retriableMaxBackoff, subtreeLockedMaxBackoff = 100*time.Millisecond, 100*time.Millisecond, 800*time.Millisecond
	answers := append(repeat(subtreeLocked(), 3), repeat(notStarted(), 2)...)
	transport := &answeringTransport{answers: append(answers, nil)}
	c := connectedWith(t, transport)

	require.NoError(t, execute(c))
	require.Len(t, transport.requestIDs, 6)
	// The lock waits doubled the backoff to 800 ms; the wait before the
	// first retry of the other kind is capped at 100 ms, at most 125 ms
	// with the jitter.
	gap := transport.requestTimes[4].Sub(transport.requestTimes[3])
	assert.Less(t, gap, 400*time.Millisecond, "waited %v", gap)

	transport = &answeringTransport{answers: append(append(repeat(notStarted(), 2), repeat(subtreeLocked(), 3)...), nil)}
	c = connectedWith(t, transport)
	require.NoError(t, execute(c))
	assert.Len(t, transport.requestIDs, 6)
}

// Close may run while Execute waits with the request lock released. The
// request must stop there, not reconnect and run after the close.
func TestExecuteStopsRetryingWhenClosedDuringTheWait(t *testing.T) {
	shortRetries(t, 1, 10)
	// An hour before the cap too, so only Close can end the wait.
	retriableInitialBackoff, subtreeLockedMaxBackoff = time.Hour, time.Hour
	transport := &answeringTransport{answers: repeat(subtreeLocked(), 100), answered: make(chan struct{}, 100)}
	c := connectedWith(t, transport)

	result := make(chan error)
	go func() { result <- execute(c) }()
	<-transport.answered
	require.NoError(t, c.Close())

	select {
	case err := <-result:
		nerr, ok := err.(*NamenodeError)
		require.True(t, ok, "the last namenode exception is returned: %v", err)
		assert.Equal(t, retriableExceptionClass, nerr.exception)
	case <-time.After(2 * time.Second):
		t.Fatal("Execute kept waiting after Close")
	}
	assert.Len(t, transport.requestIDs, 1, "no attempt after Close")
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
