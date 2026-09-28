package hdfs

import (
	"bufio"
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"math/rand"
	"net"
	"os"
	"strings"
	"sync"
	"testing"

	hdfs "github.com/colinmarc/hdfs/v2/internal/protocol/hadoop_hdfs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
)

// These tests run against the real cluster in HADOOP_CONF_DIR, like the rest
// of the package. The datanode fault is injected on the client side: the
// client's DatanodeDialFunc hands the BlockWriter a connection to a fake
// datanode that refuses the block (or fails to connect at all) instead of a
// connection to the real one. Everything else, i.e. abandonBlock, addBlock
// with excludeNodes, block placement and the resulting file state, is the
// real namenode and real datanodes.

// faultyDatanodeDialer wraps the real datanode dialer and refuses the dials
// selected by `refuse`. In "status" mode the fake datanode answers the write
// request with Status_ERROR (the incident's failure mode); in "dial" mode the
// connection itself fails.
type faultyDatanodeDialer struct {
	real   func(ctx context.Context, network, addr string) (net.Conn, error)
	mode   string
	refuse func(dial int, addr string) bool

	mu      sync.Mutex
	dials   []string
	refused []string
}

func (d *faultyDatanodeDialer) dial(ctx context.Context, network, addr string) (net.Conn, error) {
	d.mu.Lock()
	d.dials = append(d.dials, addr)
	refuse := d.refuse(len(d.dials), addr)
	if refuse {
		d.refused = append(d.refused, addr)
	}
	d.mu.Unlock()

	if !refuse {
		return d.real(ctx, network, addr)
	}
	if d.mode == "dial" {
		return nil, fmt.Errorf("dial tcp %s: connection refused (injected by test)", addr)
	}
	client, server := net.Pipe()
	go refuseWriteBlock(server)
	return client, nil
}

func writePrefixed(w io.Writer, msg proto.Message) error {
	b, err := proto.Marshal(msg)
	if err != nil {
		return err
	}
	prefix := make([]byte, binary.MaxVarintLen32)
	n := binary.PutUvarint(prefix, uint64(len(b)))
	_, err = w.Write(append(prefix[:n], b...))
	return err
}

// readWriteBlockRequest reads the OP_WRITE_BLOCK request a BlockWriter sends
// when it opens a block: protocol version, op code and the length-prefixed
// OpWriteBlockProto.
func readWriteBlockRequest(r *bufio.Reader) (*hdfs.OpWriteBlockProto, error) {
	if _, err := io.ReadFull(r, make([]byte, 3)); err != nil { // version + op
		return nil, err
	}
	n, err := binary.ReadUvarint(r)
	if err != nil {
		return nil, err
	}
	reqBytes := make([]byte, n)
	if _, err := io.ReadFull(r, reqBytes); err != nil {
		return nil, err
	}
	op := &hdfs.OpWriteBlockProto{}
	if err := proto.Unmarshal(reqBytes, op); err != nil {
		return nil, err
	}
	return op, nil
}

// refuseWriteBlock plays a datanode that cannot store the block: it reads the
// write request and answers Status_ERROR, as a datanode whose rbw directory
// is gone does ("java.io.IOException: No such file or directory" from
// BlockPoolSlice.createRbwFile), then closes the connection.
func refuseWriteBlock(conn net.Conn) {
	defer conn.Close()
	if _, err := readWriteBlockRequest(bufio.NewReader(conn)); err != nil {
		return
	}
	writePrefixed(conn, &hdfs.BlockOpResponseProto{
		Status:       hdfs.Status_ERROR.Enum(),
		FirstBadLink: proto.String(""),
		Message:      proto.String("java.io.IOException: No such file or directory"),
	})
}

func (d *faultyDatanodeDialer) dialed() []string {
	d.mu.Lock()
	defer d.mu.Unlock()
	return append([]string{}, d.dials...)
}

func (d *faultyDatanodeDialer) refusedAddrs() []string {
	d.mu.Lock()
	defer d.mu.Unlock()
	return append([]string{}, d.refused...)
}

// getClientWithFaultyDatanodes returns a fresh (uncached) test client whose
// datanode dials go through the returned faultyDatanodeDialer.
func getClientWithFaultyDatanodes(t *testing.T, mode string, refuse func(dial int, addr string) bool) (*Client, *faultyDatanodeDialer) {
	options := getClientOptionsForUser(t, "gohdfs1")
	d := &faultyDatanodeDialer{real: options.DatanodeDialFunc, mode: mode, refuse: refuse}
	if d.real == nil {
		d.real = (&net.Dialer{}).DialContext
	}
	options.DatanodeDialFunc = d.dial

	client, err := NewClient(options)
	require.NoError(t, err)

	if mode == "status" {
		// The fake datanode speaks the plain data transfer protocol; with
		// SASL/encryption on the datanode connection the client would send
		// a handshake first. Use the "dial" mode in that setup.
		defaults, err := client.fetchDefaults()
		require.NoError(t, err)
		if options.DataTransferProtection != "" || defaults.GetEncryptDataTransfer() {
			t.Skip("fake datanode does not speak SASL; skipping status-refusal test")
		}
	}
	return client, d
}

// liveDatanodeCount asks the namenode how many datanodes are live. The RPC
// needs superuser privilege, so it goes through the harness's superuser
// client; if that is not a superuser on this cluster either, ok is false and
// the caller decides from the namenode's block placement errors instead.
func liveDatanodeCount(t *testing.T) (count int, ok bool) {
	req := &hdfs.GetDatanodeReportRequestProto{Type: hdfs.DatanodeReportTypeProto_LIVE.Enum()}
	resp := &hdfs.GetDatanodeReportResponseProto{}
	if err := getClientForSuperUser(t).namenode.Execute("getDatanodeReport", req, resp); err != nil {
		t.Logf("cannot read the datanode report (%v); relying on block placement errors instead", err)
		return 0, false
	}
	return len(resp.GetDi()), true
}

func requireLiveDatanodes(t *testing.T, n int) {
	if count, ok := liveDatanodeCount(t); ok && count < n {
		t.Skipf("test needs at least %d live datanodes, cluster has %d", n, count)
	}
}

// skipIfNoSpareDatanode skips the test when the write failed because the
// namenode had no datanode left outside the excluded set, which means the
// retry worked as designed but the cluster is too small to show it.
func skipIfNoSpareDatanode(t *testing.T, err error) {
	if err == nil {
		return
	}
	msg := err.Error()
	if strings.Contains(msg, "could only be replicated to 0 nodes") ||
		strings.Contains(msg, "excluded in this operation") ||
		strings.Contains(msg, "NotEnoughReplicasException") {
		t.Skipf("no datanode left to retry on after excluding the refusing one: %v", err)
	}
}

func blockLocations(t *testing.T, c *Client, path string) []*hdfs.LocatedBlockProto {
	info, err := c.Stat(path)
	require.NoError(t, err)
	req := &hdfs.GetBlockLocationsRequestProto{
		Src:    proto.String(path),
		Offset: proto.Uint64(0),
		Length: proto.Uint64(uint64(info.Size()) + 1),
	}
	resp := &hdfs.GetBlockLocationsResponseProto{}
	require.NoError(t, c.namenode.Execute("getBlockLocations", req, resp))
	return resp.GetLocations().GetBlocks()
}

// datanodeAddrs returns the addresses a BlockWriter may dial for the node
// (ip:port and hostname:port), to match against the dialer's records.
func datanodeAddrs(node *hdfs.DatanodeInfoProto) []string {
	id := node.GetId()
	return []string{
		fmt.Sprintf("%s:%d", id.GetIpAddr(), id.GetXferPort()),
		fmt.Sprintf("%s:%d", id.GetHostName(), id.GetXferPort()),
	}
}

func assertNoBlockOn(t *testing.T, blocks []*hdfs.LocatedBlockProto, refusedAddrs []string) {
	for _, b := range blocks {
		for _, loc := range b.GetLocs() {
			for _, addr := range datanodeAddrs(loc) {
				for _, refused := range refusedAddrs {
					assert.NotEqual(t, refused, addr, "block %d was placed on the datanode that refused a block", b.GetB().GetBlockId())
				}
			}
		}
	}
}

func randomPayload(n int) []byte {
	b := make([]byte, n)
	rand.Read(b)
	return b
}

func readBack(t *testing.T, path string) []byte {
	reader, err := getClient(t).Open(path)
	require.NoError(t, err)
	defer reader.Close()
	data, err := io.ReadAll(reader)
	require.NoError(t, err)
	return data
}

// The datanode of the first block refuses the write. The writer abandons
// the block, excludes that datanode and lands the same data on another one;
// the file reads back intact and none of its blocks is on the refused node.
func TestClusterWriteRetriesRefusedBlock(t *testing.T) {
	for _, mode := range []string{"status", "dial"} {
		t.Run(mode, func(t *testing.T) {
			path := "/_test/retry/refused-" + mode + ".bin"
			mkdirp(t, "/_test/retry")
			baleet(t, path)

			client, dialer := getClientWithFaultyDatanodes(t, mode,
				func(dial int, addr string) bool { return dial == 1 })
			defer client.Close()
			requireLiveDatanodes(t, 2)

			writer, err := client.CreateFile(path, 1, 1<<20, 0644, false, false)
			require.NoError(t, err)

			payload := randomPayload(200 * 1024) // several packets, one block
			n, err := writer.Write(payload)
			skipIfNoSpareDatanode(t, err)
			require.NoError(t, err)
			assert.Equal(t, len(payload), n)
			assertClose(t, writer)

			dials := dialer.dialed()
			require.Len(t, dials, 2, "expected one refused dial and one successful retry, got %v", dials)
			assert.NotEqual(t, dials[0], dials[1], "retry must go to a different datanode")
			assert.Equal(t, []string{dials[0]}, dialer.refusedAddrs())

			assert.Equal(t, payload, readBack(t, path))
			blocks := blockLocations(t, getClient(t), path)
			require.Len(t, blocks, 1)
			assertNoBlockOn(t, blocks, dialer.refusedAddrs())
		})
	}
}

// A refused block in the middle of a multi-block file is re-allocated after
// the last good block, the datanode stays excluded for the rest of the file,
// and the content is intact.
func TestClusterWriteRetriesRefusedMiddleBlock(t *testing.T) {
	path := "/_test/retry/refused-middle.bin"
	mkdirp(t, "/_test/retry")
	baleet(t, path)

	client, dialer := getClientWithFaultyDatanodes(t, "status",
		func(dial int, addr string) bool { return dial == 2 })
	defer client.Close()
	requireLiveDatanodes(t, 2)

	const blockSize = 1 << 20
	writer, err := client.CreateFile(path, 1, blockSize, 0644, false, false)
	require.NoError(t, err)

	payload := randomPayload(2*blockSize + blockSize/2) // 3 blocks
	n, err := writer.Write(payload)
	skipIfNoSpareDatanode(t, err)
	require.NoError(t, err)
	assert.Equal(t, len(payload), n)
	assertClose(t, writer)

	dials := dialer.dialed()
	require.Len(t, dials, 4, "block 1, refused block 2, its retry, block 3: got %v", dials)
	refused := dialer.refusedAddrs()
	require.Equal(t, []string{dials[1]}, refused)
	for _, addr := range dials[2:] {
		assert.NotEqual(t, refused[0], addr, "the refused datanode must stay excluded for later blocks")
	}

	assert.Equal(t, payload, readBack(t, path))
	blocks := blockLocations(t, getClient(t), path)
	require.Len(t, blocks, 3)
	assertNoBlockOn(t, blocks[1:], refused)
	var total uint64
	for _, b := range blocks {
		total += b.GetB().GetNumBytes()
	}
	assert.EqualValues(t, len(payload), total)
}

// When no datanode accepts the block the write fails, Close reports the same
// error instead of completing the file, and the file is left under
// construction with every refused block abandoned (no block, length 0), so
// it can be removed or lease-recovered. This is the incident scenario with
// the fix: no complete on a replica-less block.
func TestClusterWriteFailsWhenEveryDatanodeRefuses(t *testing.T) {
	path := "/_test/retry/all-refused.bin"
	mkdirp(t, "/_test/retry")
	baleet(t, path)

	client, dialer := getClientWithFaultyDatanodes(t, "dial",
		func(dial int, addr string) bool { return true })
	defer client.Close()

	writer, err := client.CreateFile(path, 1, 1<<20, 0644, false, false)
	require.NoError(t, err)

	_, err = writer.Write([]byte("this never reaches a datanode"))
	require.Error(t, err)
	assert.IsType(t, &os.PathError{}, err)

	assert.Equal(t, err, writer.Flush(), "Flush must not report a failed writer as flushed")

	closeErr := writer.Close()
	assert.Equal(t, err, closeErr, "Close must surface the write error, not complete the file")

	_, again := writer.Write([]byte("more"))
	assert.Equal(t, err, again)
	assert.Equal(t, err, writer.Flush())

	// With N live datanodes and 3 retries the writer dials at most 4 times;
	// with fewer datanodes the namenode runs out of nodes to exclude first.
	// Either way at least one block was refused and abandoned.
	require.NotEmpty(t, dialer.refusedAddrs())
	assert.LessOrEqual(t, len(dialer.dialed()), 1+defaultBlockWriteRetries)

	plain := getClient(t)
	info, err := plain.Stat(path)
	require.NoError(t, err, "the file stays (under construction) for lease recovery")
	assert.EqualValues(t, 0, info.Size())
	assert.Empty(t, blockLocations(t, plain, path), "every refused block must have been abandoned")

	// What hopsfs-mount does on a failed flush: the file must be removable.
	require.NoError(t, plain.Remove(path))
}

// Small files that start in the DB buffer and overflow it are written to a
// datanode like any other; a refusal there must be retried and, if it cannot
// be recovered, must not complete the file.
func TestClusterSmallFileOverflowRetriesRefusedBlock(t *testing.T) {
	path := "/_test/retry/overflow.bin"
	mkdirp(t, "/_test/retry")
	baleet(t, path)

	client, dialer := getClientWithFaultyDatanodes(t, "status",
		func(dial int, addr string) bool { return dial == 1 })
	defer client.Close()
	requireLiveDatanodes(t, 2)

	writer, err := client.Create(path)
	require.NoError(t, err)

	payload := randomPayload(MaxSmallFileSize + 4096)
	_, err = writer.Write(payload[:MaxSmallFileSize])
	require.NoError(t, err)
	_, err = writer.Write(payload[MaxSmallFileSize:]) // overflows the DB buffer
	skipIfNoSpareDatanode(t, err)
	require.NoError(t, err)
	assertClose(t, writer)

	require.Len(t, dialer.dialed(), 2)
	assert.Equal(t, payload, readBack(t, path))
	assertNoBlockOn(t, blockLocations(t, getClient(t), path), dialer.refusedAddrs())
}

// getClientWithFaultyDatanodesAndRetries is getClientWithFaultyDatanodes
// with an explicit dfs.client.block.write.retries value.
func getClientWithFaultyDatanodesAndRetries(t *testing.T, mode string, retries int, refuse func(dial int, addr string) bool) (*Client, *faultyDatanodeDialer) {
	client, dialer := getClientWithFaultyDatanodes(t, mode, refuse)
	client.options.BlockWriteRetries = retries
	return client, dialer
}

// The retry budget is per block, as in DataStreamer.nextBlockOutputStream:
// with a budget of one retry, a refusal in the first block and another in
// the third are each retried, since neither block is refused more than once.
// Block 1 lands on a node other than its refuser, and block 3 on a node
// other than both refusers.
func TestClusterBlockWriteRetriesAreCountedPerBlock(t *testing.T) {
	path := "/_test/retry/per-block-budget.bin"
	mkdirp(t, "/_test/retry")
	baleet(t, path)

	client, dialer := getClientWithFaultyDatanodesAndRetries(t, "status", 1,
		func(dial int, addr string) bool { return dial == 1 || dial == 4 })
	defer client.Close()
	requireLiveDatanodes(t, 3)

	const blockSize = 1 << 20
	writer, err := client.CreateFile(path, 1, blockSize, 0644, false, false)
	require.NoError(t, err)

	payload := randomPayload(2*blockSize + blockSize/2) // 3 blocks
	n, err := writer.Write(payload)
	skipIfNoSpareDatanode(t, err)
	require.NoError(t, err)
	assert.Equal(t, len(payload), n)
	assertClose(t, writer)

	dials := dialer.dialed()
	require.Len(t, dials, 5, "refused block 1, its retry, block 2, refused block 3, its retry: got %v", dials)
	refused := dialer.refusedAddrs()
	require.Equal(t, []string{dials[0], dials[3]}, refused)
	assert.NotEqual(t, refused[0], refused[1], "the second refusal is on a node that was not excluded yet")
	for _, addr := range dials[1:] {
		assert.NotEqual(t, refused[0], addr, "the first refused datanode must stay excluded for the rest of the file")
	}
	assert.NotEqual(t, refused[1], dials[4], "the retry of block 3 must avoid its refuser")

	assert.Equal(t, payload, readBack(t, path))
	blocks := blockLocations(t, getClient(t), path)
	require.Len(t, blocks, 3)
	assertNoBlockOn(t, blocks, refused[:1])
	assertNoBlockOn(t, blocks[2:], refused)
	var total uint64
	for _, b := range blocks {
		total += b.GetB().GetNumBytes()
	}
	assert.EqualValues(t, len(payload), total)
}

// The per-block budget is still a budget: with one retry allowed, a block
// refused twice in a row fails the write even though the earlier block's
// single refusal was retried fine. Close reports the same error and the
// file is not completed.
func TestClusterBlockWriteRetriesStillCapConsecutiveRefusals(t *testing.T) {
	path := "/_test/retry/per-block-cap.bin"
	mkdirp(t, "/_test/retry")
	baleet(t, path)

	client, dialer := getClientWithFaultyDatanodesAndRetries(t, "status", 1,
		func(dial int, addr string) bool { return dial != 2 })
	defer client.Close()
	requireLiveDatanodes(t, 3)

	const blockSize = 1 << 20
	writer, err := client.CreateFile(path, 1, blockSize, 0644, false, false)
	require.NoError(t, err)

	payload := randomPayload(blockSize + blockSize/2) // 2 blocks
	n, err := writer.Write(payload)
	skipIfNoSpareDatanode(t, err)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unable to create new block after 2 attempts")
	assert.Equal(t, blockSize, n, "block 1 was written before block 2 ran out of retries")

	dials := dialer.dialed()
	require.Len(t, dials, 4, "refused block 1, its retry, block 2 refused twice: got %v", dials)
	assert.Equal(t, []string{dials[0], dials[2], dials[3]}, dialer.refusedAddrs())

	assert.Equal(t, err, writer.Close(), "Close must surface the write error, not complete the file")

	// What hopsfs-mount does on a failed flush: the file must be removable.
	require.NoError(t, getClient(t).Remove(path))
}
