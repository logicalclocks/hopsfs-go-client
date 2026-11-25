package rpc

import (
	"fmt"
	"io"
)

// ConnectionError is a namenode call that failed on its connection rather
// than with an answer from the namenode.
type ConnectionError struct {
	Method string
	// Sent is true when a request of this call went out and no answer came
	// back, so the namenode may have run it.
	Sent bool
	Err  error
}

func (e *ConnectionError) Error() string {
	if e.Sent {
		return fmt.Sprintf("%s: namenode connection lost after the request was sent: %v", e.Method, e.Err)
	}
	return fmt.Sprintf("%s: namenode connection failed before the request was sent: %v", e.Method, e.Err)
}

func (e *ConnectionError) Unwrap() error {
	return e.Err
}

// readOnlyMethods are the calls that change nothing on the namenode, so one
// that was sent and got no answer can be sent again. renewLease only extends
// leases the client already holds.
var readOnlyMethods = map[string]bool{
	"getActiveNamenodesForClient": true,
	"getBlockLocations":           true,
	"getContentSummary":           true,
	"getDataEncryptionKey":        true,
	"getFileInfo":                 true,
	"getFsStats":                  true,
	"getListing":                  true,
	"getNNEpochMS":                true,
	"getServerDefaults":           true,
	"getXAttrs":                   true,
	"listXAttrs":                  true,
	"renewLease":                  true,
}

// countingReader counts the bytes read through it, to tell a connection that
// died before the answer from one that died in the middle of it.
type countingReader struct {
	r io.Reader
	n int
}

func (cr *countingReader) Read(p []byte) (int, error) {
	n, err := cr.r.Read(p)
	cr.n += n
	return n, err
}
