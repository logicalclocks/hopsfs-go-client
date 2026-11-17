package transfer

import (
	"bytes"
	"testing"

	hadoop "github.com/colinmarc/hdfs/v2/internal/protocol/hadoop_common"
	hdfs "github.com/colinmarc/hdfs/v2/internal/protocol/hadoop_hdfs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
)

func TestPacketSize(t *testing.T) {
	bws := &blockWriteStream{}
	bws.buf.Write(make([]byte, outboundPacketSize*3))
	packet := bws.makePacket()

	assert.EqualValues(t, outboundPacketSize, len(packet.data))
}

func TestPacketSizeUndersize(t *testing.T) {
	bws := &blockWriteStream{}
	bws.buf.Write(make([]byte, outboundPacketSize-5))
	packet := bws.makePacket()

	assert.EqualValues(t, outboundPacketSize-5, len(packet.data))
}

func TestPacketSizeAlignment(t *testing.T) {
	bws := &blockWriteStream{}
	bws.buf.Write(make([]byte, outboundPacketSize*3))

	bws.offset = 5
	packet := bws.makePacket()

	assert.EqualValues(t, outboundChunkSize-5, len(packet.data))
}

func locatedBlockForWrite(storageIDs []string) *hdfs.LocatedBlockProto {
	locs := []*hdfs.DatanodeInfoProto{}
	types := []hdfs.StorageTypeProto{}
	for i := 0; i < 2; i++ {
		locs = append(locs, &hdfs.DatanodeInfoProto{Id: &hdfs.DatanodeIDProto{
			IpAddr:       proto.String("127.0.0.1"),
			HostName:     proto.String("localhost"),
			DatanodeUuid: proto.String("dn"),
			XferPort:     proto.Uint32(50010),
			InfoPort:     proto.Uint32(50075),
			IpcPort:      proto.Uint32(50020),
		}})
		types = append(types, hdfs.StorageTypeProto_DISK)
	}
	return &hdfs.LocatedBlockProto{
		B: &hdfs.ExtendedBlockProto{
			PoolId:          proto.String("bp"),
			BlockId:         proto.Uint64(1),
			GenerationStamp: proto.Uint64(1001),
		},
		Offset:       proto.Uint64(0),
		Locs:         locs,
		Corrupt:      proto.Bool(false),
		BlockToken:   &hadoop.TokenProto{Identifier: []byte{}, Password: []byte{}, Kind: proto.String(""), Service: proto.String("")},
		StorageTypes: types,
		StorageIDs:   storageIDs,
	}
}

// decodeWriteRequest parses the operation a BlockWriter sent: the two-byte
// protocol version, the op code and the length-prefixed request message.
func decodeWriteRequest(t *testing.T, raw []byte) *hdfs.OpWriteBlockProto {
	require.EqualValues(t, writeBlockOp, raw[2])
	op := &hdfs.OpWriteBlockProto{}
	require.NoError(t, readPrefixedMessage(bytes.NewReader(raw[3:]), op))
	return op
}

// The write request names the storage of every node in the pipeline when the
// NameNode supplied storage ids, and none when it did not.
func TestWriteRequestNamesPipelineStorages(t *testing.T) {
	bw := &BlockWriter{ClientName: "cl",
		Block: locatedBlockForWrite([]string{"storage-a", "storage-b"})}
	var buf bytes.Buffer
	require.NoError(t, bw.writeBlockWriteRequest(&buf))
	op := decodeWriteRequest(t, buf.Bytes())
	assert.Equal(t, "storage-a", op.GetStorageId())
	assert.Equal(t, []string{"storage-b"}, op.GetTargetStorageIds())

	bw = &BlockWriter{ClientName: "cl", Block: locatedBlockForWrite(nil)}
	buf.Reset()
	require.NoError(t, bw.writeBlockWriteRequest(&buf))
	op = decodeWriteRequest(t, buf.Bytes())
	assert.Nil(t, op.StorageId)
	assert.Empty(t, op.GetTargetStorageIds())
}
