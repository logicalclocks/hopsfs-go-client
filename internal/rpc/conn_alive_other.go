//go:build !linux && !darwin

package rpc

import "net"

// connAlive cannot peek a socket on this platform: a connection closed while
// idle shows only when its call fails.
func connAlive(conn net.Conn) bool {
	return true
}
