// Package rpc implements some of the lower-level functionality required to
// communicate with the namenode and datanodes.
package rpc

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"time"

	"google.golang.org/protobuf/proto"
)

var errInvalidResponse = errors.New("invalid response from namenode")

// DefaultMaxResponseLength bounds the buffer allocated for one RPC response unless a connection
// sets its own, as Hadoop's ipc.maximum.response.length (128 MB by default) does. The length
// comes off the wire, so a peer that is not speaking Hadoop RPC would otherwise make the client
// allocate up to 4 GB.
const DefaultMaxResponseLength = 128 << 20

// tlsRecordType reports whether a response length is really the start of a TLS record (content
// type 20 to 23, then major version 3), which is what a namenode that expects TLS answers to a
// plaintext client.
func tlsRecordType(packetLength uint32) bool {
	contentType, majorVersion := packetLength>>24, (packetLength>>16)&0xff
	return contentType >= 20 && contentType <= 23 && majorVersion == 3
}

// Used for client ID generation, below.
const chars = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

func newClientID() []byte {
	id := make([]byte, 16)

	rand.Seed(time.Now().UTC().UnixNano())
	for i := range id {
		id[i] = chars[rand.Intn(len(chars))]
	}

	return id
}

func makePrefixedMessage(msg proto.Message) ([]byte, error) {
	msgBytes, err := proto.Marshal(msg)
	if err != nil {
		return nil, err
	}

	lengthBytes := make([]byte, 10)
	n := binary.PutUvarint(lengthBytes, uint64(len(msgBytes)))
	return append(lengthBytes[:n], msgBytes...), nil
}

func makeRPCPacket(msgs ...proto.Message) ([]byte, error) {
	packet := make([]byte, 4, 128)

	length := 0
	for _, msg := range msgs {
		b, err := makePrefixedMessage(msg)
		if err != nil {
			return nil, err
		}

		packet = append(packet, b...)
		length += len(b)
	}

	binary.BigEndian.PutUint32(packet, uint32(length))
	return packet, nil
}

// readRPCPacket reads one length-prefixed RPC packet; maxLength of zero selects DefaultMaxResponseLength.
func readRPCPacket(r io.Reader, maxLength uint32, msgs ...proto.Message) error {
	var packetLength uint32
	err := binary.Read(r, binary.BigEndian, &packetLength)
	if err != nil {
		return err
	}

	if maxLength == 0 {
		maxLength = DefaultMaxResponseLength
	}
	if packetLength > maxLength {
		if tlsRecordType(packetLength) {
			return fmt.Errorf("%w: the namenode answered with TLS, but this client is not configured for TLS", errInvalidResponse)
		}
		return fmt.Errorf("%w: response of %d bytes exceeds the %d byte limit (NamenodeMaxResponseLength)", errInvalidResponse, packetLength, maxLength)
	}

	packet := make([]byte, packetLength)
	_, err = io.ReadFull(r, packet)
	if err != nil {
		return err
	}

	for _, msg := range msgs {
		// HDFS doesn't send all the response messages all the time (for example, if
		// the RpcResponseHeaderProto contains an error).
		if len(packet) == 0 {
			return nil
		}

		msgLength, n := binary.Uvarint(packet)
		if n <= 0 || msgLength > uint64(len(packet)) {
			return errInvalidResponse
		}

		packet = packet[n:]
		if msgLength != 0 {
			err = proto.Unmarshal(packet[:msgLength], msg)
			if err != nil {
				return err
			}

			packet = packet[msgLength:]
		}
	}

	if len(packet) > 0 {
		return errInvalidResponse
	}

	return nil
}
