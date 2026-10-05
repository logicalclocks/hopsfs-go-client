//go:build linux

package rpc

import (
	"syscall"
	"time"
)

// tcpUserTimeoutOption is TCP_USER_TIMEOUT from <linux/tcp.h>. The syscall
// package does not export it.
const tcpUserTimeoutOption = 18

func tcpUserTimeoutControl(timeout time.Duration) func(network, address string, rc syscall.RawConn) error {
	ms := int(timeout / time.Millisecond)
	return func(network, address string, rc syscall.RawConn) error {
		var sockErr error
		err := rc.Control(func(fd uintptr) {
			sockErr = syscall.SetsockoptInt(int(fd), syscall.IPPROTO_TCP, tcpUserTimeoutOption, ms)
		})
		if err != nil {
			return err
		}
		return sockErr
	}
}
