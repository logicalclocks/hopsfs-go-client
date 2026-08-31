package hdfs

import (
	"testing"
	"time"

	hdfs "github.com/colinmarc/hdfs/v2/internal/protocol/hadoop_hdfs"
	"google.golang.org/protobuf/proto"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// These tests reproduce the "phantom zero-byte block" defect that produced
// the production data-loss block (HOPSFS-380 investigation, block 2627361).
//
// Root cause, in the client (file_writer.go):
//
//	func (f *FileWriter) writeInternal(b []byte) (int, error) {
//	    if f.blockWriter == nil {
//	        err := f.startNewBlock()   // <-- addBlock BEFORE checking len(b)
//	        ...
//	    }
//	    for off := 0; off < len(b); { ... }   // empty b => loop never runs
//	    ...
//	}
//
// writeInternal allocates a block (startNewBlock -> addBlock at the NameNode)
// the moment it is first called, before it looks at how many bytes it was
// given. The datanode connection is opened lazily, only from
// BlockWriter.Write (internal/transfer/block_writer.go), so a block that
// receives zero bytes is allocated at the NameNode but never streamed to any
// datanode: its NumBytes stays at the allocation-time 0. On Close, closeInt
// sends that zero-byte block to `complete` as the file's last block.
//
// The Java client does not do this: its DataStreamer allocates the next block
// lazily, only when a data packet for it actually exists (see
// DFSOutputStream.endBlock + DataStreamer's run loop), so a zero-length write
// never leaves an allocated-but-unwritten block.
//
// Downstream, the zero-byte phantom block is exactly what the NameNode logged
// for production block 2627361: COMPLETE (on clusters with
// numCommittedAllowed>0, where completeLastProvidedBlock completes it without
// any datanode IBR), cloud_upload_pending=true, numBytes=0, absent from every
// datanode and from the object store, which the ProvidedBlocksChecker later
// marks corrupt ("BR DATA LOSS ... DN never uploaded").
//
// RUN against a running hopsfs-standalone cluster:
//
//	# terminal 1 — start the standalone (its NDB/RonDB backend must be up)
//	cd ~/code/hops/hopsfs-standalone && ./build-run
//
//	# terminal 2 — point the client at the generated config and run
//	export HADOOP_CONF_DIR=/tmp/hopsfs-conf
//	cd ~/code/hops/hopsfs-go/hopsfs-go-client
//	go test -run 'TestGoClientZeroLengthFirstWrite|TestGoClientEmptyFileNoWrite' -v
//
// The default standalone is plain HDFS (numCommittedAllowed=0): the phantom
// zero-byte last block has no datanode replica, so the NameNode cannot
// complete the file and Close fails — the reproduction shows up as a Close
// error plus a block allocated for a zero-byte file. On a cloud standalone
// with numCommittedAllowed>0, Close instead succeeds and the phantom block is
// left COMPLETE + cloud_upload_pending, reproducing the exact production
// state that later becomes a DATA LOSS corrupt verdict.

// blockCountForPath asks the NameNode for the located blocks of a path,
// independent of the client's cached view. A correct empty file has zero
// blocks; a phantom block shows up as >= 1.
func blockCountForPath(t *testing.T, client *Client, name string) int {
	t.Helper()
	req := &hdfs.GetBlockLocationsRequestProto{
		Src:    proto.String(name),
		Offset: proto.Uint64(0),
		Length: proto.Uint64(1 << 30),
	}
	resp := &hdfs.GetBlockLocationsResponseProto{}
	err := client.namenode.Execute("getBlockLocations", req, resp)
	require.NoError(t, err, "getBlockLocations for %s", name)
	return len(resp.GetLocations().GetBlocks())
}

// tryClose attempts to close the writer, tolerating the transient
// ErrReplicating retry signal, and returns the final error (nil on success).
func tryClose(w *FileWriter) error {
	var err error
	for i := 0; i < 10; i++ {
		err = w.Close()
		if IsErrReplicating(err) {
			time.Sleep(200 * time.Millisecond)
			continue
		}
		break
	}
	return err
}

// TestGoClientZeroLengthFirstWriteNoPhantomBlock is the regression test for
// the fix: a zero-length first write must NOT allocate a block. Before the fix
// writeInternal called startNewBlock()->addBlock before checking len(b), so a
// zero-length write left a phantom block at the NameNode that no datanode ever
// received (numBytes=0, never uploaded). After the fix, writing nothing is a
// no-op: the file closes cleanly with zero blocks, matching the Java client.
func TestGoClientZeroLengthFirstWriteNoPhantomBlock(t *testing.T) {
	client := getClient(t)
	mkdirp(t, "/_test/phantom")
	const name = "/_test/phantom/zero-length-first-write"
	_ = client.Remove(name)

	writer, err := client.Create(name)
	require.NoError(t, err)

	// First (and only) write carries zero bytes.
	n, err := writer.Write([]byte{})
	require.NoError(t, err)
	assert.Equal(t, 0, n)

	closeErr := tryClose(writer)
	require.NoError(t, closeErr, "zero-length write + close must succeed cleanly")

	// No phantom block: a 0-byte file must have zero blocks.
	blocks := blockCountForPath(t, client, name)
	t.Logf("zero-length-first-write: close error = %v; blocks allocated = %d",
		closeErr, blocks)
	assert.Equal(t, 0, blocks,
		"zero-length write must not allocate any block (no phantom)")

	fi, err := client.Stat(name)
	require.NoError(t, err)
	assert.EqualValues(t, 0, fi.Size(), "file must be zero length")

	_ = client.Remove(name)
}

// TestGoClientEmptyFileNoWriteClosesClean is the control: creating a file and
// closing it WITHOUT any write allocates no block and closes cleanly. This
// isolates the defect to the write path — it is the eager startNewBlock() in
// writeInternal, not create/close, that produces the phantom block.
func TestGoClientEmptyFileNoWriteClosesClean(t *testing.T) {
	client := getClient(t)
	mkdirp(t, "/_test/phantom")
	const name = "/_test/phantom/empty-no-write"
	_ = client.Remove(name)

	writer, err := client.Create(name)
	require.NoError(t, err)

	// No Write call at all.
	closeErr := tryClose(writer)
	require.NoError(t, closeErr, "empty file with no write must close cleanly")

	blocks := blockCountForPath(t, client, name)
	t.Logf("empty-no-write: close error = %v; blocks allocated = %d", closeErr, blocks)
	assert.Equal(t, 0, blocks, "a clean empty file must have no blocks")

	fi, err := client.Stat(name)
	require.NoError(t, err)
	assert.EqualValues(t, 0, fi.Size())

	_ = client.Remove(name)
}
