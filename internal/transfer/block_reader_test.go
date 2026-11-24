package transfer

import (
	"context"
	"encoding/binary"
	"errors"
	"io"
	"net"
	"strings"
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
	times []time.Time
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
	d.times = append(d.times, now())
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

func (d *fakeDatanode) requests() []*hdfs.OpReadBlockProto {
	d.mu.Lock()
	defer d.mu.Unlock()
	return append([]*hdfs.OpReadBlockProto(nil), d.reqs...)
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
		writeSuccess(conn, checksumType, req.GetOffset())
		writePacket(conn, req.GetOffset(), data[req.GetOffset():], true)
	}
}

// partialReply serves n bytes from the requested offset, then hangs up in
// the header of the next packet.
func partialReply(data []byte, n uint64) datanodeReply {
	return func(conn net.Conn, req *hdfs.OpReadBlockProto) {
		off := req.GetOffset()
		writeSuccess(conn, hdfs.ChecksumTypeProto_CHECKSUM_NULL, off)
		writePacket(conn, off, data[off:off+n], false)
		conn.Write([]byte{0, 0, 0})
	}
}

func writeSuccess(conn net.Conn, checksumType hdfs.ChecksumTypeProto, off uint64) {
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
}

func writePacket(conn net.Conn, off uint64, payload []byte, last bool) {
	header, _ := proto.Marshal(&hdfs.PacketHeaderProto{
		OffsetInBlock:     proto.Int64(int64(off)),
		Seqno:             proto.Int64(0),
		LastPacketInBlock: proto.Bool(last),
		DataLen:           proto.Int32(int32(len(payload))),
	})
	prefix := make([]byte, 6)
	binary.BigEndian.PutUint32(prefix, uint32(4+len(payload)))
	binary.BigEndian.PutUint16(prefix[4:], uint16(len(header)))
	conn.Write(append(append(prefix, header...), payload...))
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

const (
	replicaNotFound = "opReadBlock BP-1:blk_1_1001 received exception " +
		"org.apache.hadoop.hdfs.server.datanode.ReplicaNotFoundException: Replica not found"
	s3NoSuchKey = "opReadBlock BP-1:blk_1_1001 received exception java.io.IOException: " +
		"software.amazon.awssdk.services.s3.model.NoSuchKeyException: The specified key does not exist."
	gcsNotFound = "opReadBlock BP-1:blk_1_1001 received exception java.io.IOException: " +
		"Object: bp/blk_1_1001.data not found in the bucket: bucket"
)

var blockData = []byte("0123456789abcdef")

func repeat(reply datanodeReply, n int) []datanodeReply {
	replies := make([]datanodeReply, n)
	for i := range replies {
		replies[i] = reply
	}
	return replies
}

// fakeClock stands in for now and sleep: sleep advances the clock and
// records the wait.
type fakeClock struct {
	start  time.Time
	t      time.Time
	sleeps []time.Duration
}

func useFakeClock(t *testing.T) *fakeClock {
	c := &fakeClock{start: time.Unix(1000000, 0), t: time.Unix(1000000, 0)}
	origNow, origSleep := now, sleep
	now = func() time.Time { return c.t }
	sleep = func(d time.Duration) {
		c.sleeps = append(c.sleeps, d)
		c.t = c.t.Add(d)
	}
	t.Cleanup(func() { now, sleep = origNow, origSleep })
	return c
}

func (c *fakeClock) slept() time.Duration {
	return c.t.Sub(c.start)
}

// useLoadBalancer points every DataNode address at host lb, as the SDK
// does for external clients, and forgets lb's recorded failures afterwards.
func useLoadBalancer(t *testing.T) {
	t.Setenv(HOPSFS_CLOUD_DATANODE_HOSTNAME_OVERRIDE_ENV, "lb")
	t.Setenv(HOPSFS_CLOUD_DATANODE_PORT_OVERRIDE_ENV, "")
	t.Cleanup(func() {
		datanodeFailuresLock.Lock()
		defer datanodeFailuresLock.Unlock()
		for addr := range datanodeFailures {
			if strings.HasPrefix(addr, "lb:") {
				delete(datanodeFailures, addr)
			}
		}
	})
}

func cloudReader(dn *fakeDatanode, ports ...uint32) *BlockReader {
	return &BlockReader{
		ClientName: "cl",
		Block:      readBlock("bucket", uint64(len(blockData)), ports...),
		DialFunc:   dn.dial,
	}
}

func TestCloudReadRetriesUntilBlockIsVisible(t *testing.T) {
	useLoadBalancer(t)
	clock := useFakeClock(t)
	dn := &fakeDatanode{replies: append(repeat(statusReply(hdfs.Status_ERROR, replicaNotFound), 3), dataReply(blockData))}

	got, err := io.ReadAll(cloudReader(dn, 50010))
	require.NoError(t, err)
	assert.Equal(t, blockData, got)
	assert.Equal(t, []string{"lb:50010", "lb:50010", "lb:50010", "lb:50010"}, dn.dials())
	assert.Equal(t, []time.Duration{200 * time.Millisecond, 400 * time.Millisecond, 800 * time.Millisecond}, clock.sleeps)
}

func TestCloudReadRetriesObjectStoreMisses(t *testing.T) {
	for name, message := range map[string]string{"s3": s3NoSuchKey, "gcs": gcsNotFound} {
		t.Run(name, func(t *testing.T) {
			useLoadBalancer(t)
			useFakeClock(t)
			dn := &fakeDatanode{replies: []datanodeReply{statusReply(hdfs.Status_ERROR, message), dataReply(blockData)}}

			got, err := io.ReadAll(cloudReader(dn, 50010))
			require.NoError(t, err)
			assert.Equal(t, blockData, got)
			assert.Len(t, dn.dials(), 2)
		})
	}
}

// The retry starts only after every location failed once.
func TestCloudReadRetriesAfterEveryLocation(t *testing.T) {
	useLoadBalancer(t)
	clock := useFakeClock(t)
	dn := &fakeDatanode{replies: append(repeat(statusReply(hdfs.Status_ERROR, replicaNotFound), 2), dataReply(blockData))}

	got, err := io.ReadAll(cloudReader(dn, 50010, 50011))
	require.NoError(t, err)
	assert.Equal(t, blockData, got)
	assert.ElementsMatch(t, []string{"lb:50010", "lb:50011"}, dn.dials()[:2])
	assert.Equal(t, []time.Time{clock.start, clock.start, clock.start.Add(200 * time.Millisecond)}, dn.times)
}

// A failure recorded for the load balancer by an earlier read does not stop
// the retry.
func TestCloudReadRetriesAfterRecordedFailure(t *testing.T) {
	useLoadBalancer(t)
	useFakeClock(t)
	datanodeFailuresLock.Lock()
	datanodeFailures["lb:50010"] = time.Now()
	datanodeFailuresLock.Unlock()
	dn := &fakeDatanode{replies: []datanodeReply{statusReply(hdfs.Status_ERROR, replicaNotFound), dataReply(blockData)}}

	got, err := io.ReadAll(cloudReader(dn, 50010))
	require.NoError(t, err)
	assert.Equal(t, blockData, got)
}

// 14 waits of the backoff add up to 51.2 s: more than half the budget.
var fiftySecondsOfMisses = repeat(statusReply(hdfs.Status_ERROR, replicaNotFound), 14)

// Each block, so each BlockReader, has its own budget.
func TestCloudReadBudgetPerBlockReader(t *testing.T) {
	useLoadBalancer(t)
	clock := useFakeClock(t)

	for i := 0; i < 2; i++ {
		dn := &fakeDatanode{replies: append(fiftySecondsOfMisses, dataReply(blockData))}
		got, err := io.ReadAll(cloudReader(dn, 50010))
		require.NoError(t, err)
		assert.Equal(t, blockData, got)
	}
	assert.Equal(t, 2*51200*time.Millisecond, clock.slept())
}

// A successful connect restarts the budget: a failure later in the same
// block gets a full budget again.
func TestCloudReadBudgetRestartsAfterConnect(t *testing.T) {
	useLoadBalancer(t)
	clock := useFakeClock(t)
	misses := func(n int) []datanodeReply { return repeat(statusReply(hdfs.Status_ERROR, replicaNotFound), n) }
	// Two locations: the first round and each retry dial both. The partial
	// read's connection breaks, the other location then misses.
	replies := append(misses(2+13*2), partialReply(blockData, 8))
	replies = append(replies, misses(1+13*2)...)
	replies = append(replies, dataReply(blockData))
	dn := &fakeDatanode{replies: replies}

	got, err := io.ReadAll(cloudReader(dn, 50010, 50011))
	require.NoError(t, err)
	assert.Equal(t, blockData, got)
	assert.Equal(t, 2*51200*time.Millisecond, clock.slept())
}

func TestCloudReadRetryOutOfBudget(t *testing.T) {
	useLoadBalancer(t)
	clock := useFakeClock(t)
	dn := &fakeDatanode{replies: []datanodeReply{statusReply(hdfs.Status_ERROR, replicaNotFound)}}

	_, err := cloudReader(dn, 50010).Read(make([]byte, len(blockData)))
	require.EqualError(t, err, "read failed: ERROR ("+replicaNotFound+")")
	assert.Equal(t, cloudReadBudget, clock.slept())
}

func TestCloudReadRetryStopsAtDeadline(t *testing.T) {
	useLoadBalancer(t)
	clock := useFakeClock(t)
	dn := &fakeDatanode{replies: []datanodeReply{statusReply(hdfs.Status_ERROR, replicaNotFound)}}
	br := cloudReader(dn, 50010)
	deadline := clock.start.Add(2500 * time.Millisecond)
	require.NoError(t, br.SetDeadline(deadline))

	_, err := br.Read(make([]byte, len(blockData)))
	require.EqualError(t, err, "read failed: ERROR ("+replicaNotFound+")")
	// The fourth wait, 1.6 s, would reach the deadline, where the read would
	// time out at once: no attempt starts there.
	assert.Equal(t, []time.Duration{200 * time.Millisecond, 400 * time.Millisecond,
		800 * time.Millisecond}, clock.sleeps)
	assert.Len(t, dn.dials(), 4)
	for _, dialed := range dn.times {
		assert.True(t, dialed.Before(deadline), "dialed at the deadline")
	}
}

// A wait that ends past the deadline, as a late wake-up does, starts no
// attempt, and Read returns the last DataNode error.
func TestCloudReadRetryLateWakeUpStartsNoAttempt(t *testing.T) {
	useLoadBalancer(t)
	clock := useFakeClock(t)
	sleep = func(d time.Duration) {
		clock.sleeps = append(clock.sleeps, d)
		clock.t = clock.t.Add(d + 500*time.Millisecond)
	}
	dn := &fakeDatanode{replies: []datanodeReply{statusReply(hdfs.Status_ERROR, replicaNotFound)}}
	br := cloudReader(dn, 50010)
	require.NoError(t, br.SetDeadline(clock.start.Add(2500*time.Millisecond)))

	_, err := br.Read(make([]byte, len(blockData)))
	require.EqualError(t, err, "read failed: ERROR ("+replicaNotFound+")")
	// Dials at 0, 0.7 and 1.6 s; the third wait (0.8 s) ends at 2.9 s.
	assert.Len(t, dn.dials(), 3)
}

// A retry attempt that runs past the deadline leaves the remaining
// locations untried.
func TestCloudReadRetryStopsBetweenLocationsAtDeadline(t *testing.T) {
	useLoadBalancer(t)
	clock := useFakeClock(t)
	dn := &fakeDatanode{replies: []datanodeReply{statusReply(hdfs.Status_ERROR, replicaNotFound)}}
	br := cloudReader(dn, 50010, 50011)
	// Each dial and handshake takes 1.5 s.
	br.DialFunc = func(ctx context.Context, network, addr string) (net.Conn, error) {
		clock.t = clock.t.Add(1500 * time.Millisecond)
		return dn.dial(ctx, network, addr)
	}
	require.NoError(t, br.SetDeadline(clock.start.Add(3500*time.Millisecond)))

	_, err := br.Read(make([]byte, len(blockData)))
	require.EqualError(t, err, "read failed: ERROR ("+replicaNotFound+")")
	// Both locations in the first pass (ending at 3 s), then one retry
	// attempt starting at 3.2 s and ending at 4.7 s, past the deadline.
	assert.Len(t, dn.dials(), 3)
}

// Slow handshakes cannot carry a retry round past the budget: no attempt
// starts after its end.
func TestCloudReadRetryStopsBetweenLocationsAtBudgetEnd(t *testing.T) {
	useLoadBalancer(t)
	clock := useFakeClock(t)
	dn := &fakeDatanode{replies: []datanodeReply{statusReply(hdfs.Status_ERROR, replicaNotFound)}}
	br := cloudReader(dn, 50010, 50011)
	// Each dial and handshake takes 6 s.
	br.DialFunc = func(ctx context.Context, network, addr string) (net.Conn, error) {
		clock.t = clock.t.Add(6 * time.Second)
		return dn.dial(ctx, network, addr)
	}

	_, err := br.Read(make([]byte, len(blockData)))
	require.EqualError(t, err, "read failed: ERROR ("+replicaNotFound+")")
	// The first pass ends at 12 s, so the budget ends at 72 s. Four retry
	// rounds of two attempts, then the fifth round's first attempt starts at
	// 66.2 s and ends at 72.2 s, past the budget: its second is not tried.
	assert.Len(t, dn.dials(), 11)
}

// A retry asks for the same offset, as a read after Seek does.
func TestCloudReadRetryAtOffset(t *testing.T) {
	useLoadBalancer(t)
	useFakeClock(t)
	dn := &fakeDatanode{replies: []datanodeReply{statusReply(hdfs.Status_ERROR, replicaNotFound), dataReply(blockData)}}
	br := cloudReader(dn, 50010)
	br.Offset = 5

	got, err := io.ReadAll(br)
	require.NoError(t, err)
	assert.Equal(t, blockData[5:], got)
	reqs := dn.requests()
	require.Len(t, reqs, 2)
	assert.EqualValues(t, 5, reqs[1].GetOffset())
}

// Without the load balancer, for blocks that are not cloud blocks, and for
// other errors, Read tries each location once and returns the error as is.
func TestCloudReadNoRetry(t *testing.T) {
	cases := []struct {
		name         string
		hostOverride string
		portOverride string
		bucket       string
		status       hdfs.Status
		message      string
	}{
		{"no override", "", "", "bucket", hdfs.Status_ERROR, replicaNotFound},
		{"port override only", "", "50010", "bucket", hdfs.Status_ERROR, replicaNotFound},
		{"not a cloud block", "lb", "", "", hdfs.Status_ERROR, replicaNotFound},
		{"other error", "lb", "", "bucket", hdfs.Status_ERROR, "opReadBlock received exception java.io.IOException: disk failed"},
		{"access token", "lb", "", "bucket", hdfs.Status_ERROR_ACCESS_TOKEN, "Block token is expired"},
		{"cold storage", "lb", "", "bucket", hdfs.Status_ERROR_BLK_IN_COLD_STORAGE, "Block is in cold storage"},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			useLoadBalancer(t)
			t.Setenv(HOPSFS_CLOUD_DATANODE_HOSTNAME_OVERRIDE_ENV, c.hostOverride)
			t.Setenv(HOPSFS_CLOUD_DATANODE_PORT_OVERRIDE_ENV, c.portOverride)
			clock := useFakeClock(t)
			dn := &fakeDatanode{replies: []datanodeReply{statusReply(c.status, c.message)}}
			br := &BlockReader{ClientName: "cl", Block: readBlock(c.bucket, uint64(len(blockData)), 50010, 50011), DialFunc: dn.dial}

			_, err := br.Read(make([]byte, len(blockData)))
			require.EqualError(t, err, "read failed: "+c.status.String()+" ("+c.message+")")
			assert.Len(t, dn.dials(), 2)
			assert.Zero(t, clock.slept())
		})
	}
}
