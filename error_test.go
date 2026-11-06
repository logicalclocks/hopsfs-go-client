package hdfs

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

// remoteError is a namenode exception as HopsFS sends it: wrapped in an IOException whose
// message starts with the cause's class name.
type remoteError struct{ exception, message string }

func (e remoteError) Method() string    { return "rename2" }
func (e remoteError) Desc() string      { return "ERROR_APPLICATION" }
func (e remoteError) Exception() string { return e.exception }
func (e remoteError) Message() string   { return e.message }
func (e remoteError) Error() string     { return e.message }

func TestInterpretWrappedException(t *testing.T) {
	wrapped := func(cause string) error {
		return remoteError{javaIOException, cause + ": details\n\tat org.apache..."}
	}
	assert.Equal(t, os.ErrNotExist, interpretException(wrapped(fileNotFoundException)))
	assert.Equal(t, os.ErrExist, interpretException(wrapped(fileAlreadyExistsException)))
	assert.Equal(t, os.ErrNotExist, interpretException(remoteError{fileNotFoundException, "x"}))
}
