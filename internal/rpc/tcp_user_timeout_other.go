//go:build !linux

package rpc

import (
	"syscall"
	"time"
)

// tcpUserTimeoutControl returns nil: TCP_USER_TIMEOUT is Linux only.
func tcpUserTimeoutControl(timeout time.Duration) func(network, address string, rc syscall.RawConn) error {
	return nil
}
