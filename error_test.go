package hdfs

import (
	"os"
	"syscall"
	"testing"

	"github.com/stretchr/testify/assert"
)

// remoteError stands in for the namenode's answer, whose type the rpc package
// keeps to itself.
type remoteError struct {
	exception string
	message   string
}

func (e remoteError) Method() string    { return "getFileInfo" }
func (e remoteError) Desc() string      { return "ERROR_APPLICATION" }
func (e remoteError) Exception() string { return e.exception }
func (e remoteError) Message() string   { return e.message }
func (e remoteError) Error() string     { return e.message }

func TestInterpretExceptionSubtreeLockIsEBUSY(t *testing.T) {
	err := interpretException(remoteError{
		exception: "org.apache.hadoop.ipc.RetriableException",
		message:   "org.apache.hadoop.ipc.SubtreeLockedException: The subtree mlruns is locked by Namenode Id: 1. Active Namenodes are: [1]\n\tat io.hops.transaction.lock.INodeLock.checkSubtreeLock(INodeLock.java:278)",
	})
	assert.Equal(t, syscall.EBUSY, err)
}

func TestInterpretExceptionOtherRetriableExceptionIsReturnedAsIs(t *testing.T) {
	in := remoteError{exception: "org.apache.hadoop.ipc.RetriableException", message: "NameNode still not started"}
	assert.Equal(t, in, interpretException(in))
}

// HopsFS wraps a namenode exception in an IOException whose message starts
// with the cause's class name.
func TestInterpretWrappedException(t *testing.T) {
	wrapped := func(cause string) error {
		return remoteError{javaIOException, cause + ": details\n\tat org.apache..."}
	}
	assert.Equal(t, os.ErrNotExist, interpretException(wrapped(fileNotFoundException)))
	assert.Equal(t, os.ErrExist, interpretException(wrapped(fileAlreadyExistsException)))
	assert.Equal(t, os.ErrNotExist, interpretException(remoteError{fileNotFoundException, "x"}))
}
