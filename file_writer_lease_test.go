package hdfs

import (
	"errors"
	"testing"

	"github.com/colinmarc/hdfs/v2/internal/rpc"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAFailedWriterReleasesItsLeaseOnceOnClose(t *testing.T) {
	namenode := &rpc.NamenodeConnection{}
	namenode.AcquireLease()
	failure := errors.New("pipeline failed")
	writer := &FileWriter{
		client:    &Client{namenode: namenode},
		leaseHeld: true,
		lastError: failure,
	}

	assert.Equal(t, failure, writer.Close())
	assert.False(t, writer.leaseHeld)
	assert.Equal(t, int64(0), namenode.OpenWriters())

	assert.Equal(t, failure, writer.Close())
	assert.Equal(t, int64(0), namenode.OpenWriters())
}

func TestAWriterWhoseCompleteFailsReleasesItsLease(t *testing.T) {
	// The connection has no namenode to call, so complete fails.
	namenode := &rpc.NamenodeConnection{}
	namenode.AcquireLease()
	writer := &FileWriter{
		client:    &Client{namenode: namenode},
		name:      "/file",
		leaseHeld: true,
	}

	err := writer.Close()
	require.Error(t, err)
	assert.False(t, writer.leaseHeld)
	assert.Equal(t, int64(0), namenode.OpenWriters())

	assert.Equal(t, err, writer.Close())
	assert.Equal(t, int64(0), namenode.OpenWriters())
}
