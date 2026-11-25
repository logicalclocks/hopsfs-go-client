//go:build linux || darwin

package rpc

import (
	"crypto/tls"
	"errors"
	"net"
	"syscall"
	"time"
)

// tlsPendingReadWait bounds the read connAlive makes on a TLS connection with
// bytes waiting. The bytes are already in the socket buffer, so the read
// returns at once unless they are only a post-handshake message (a session
// ticket), which the read consumes before it waits for more.
const tlsPendingReadWait = 5 * time.Millisecond

// connAlive reports whether an idle connection can still carry a request. The
// namenode sends nothing between calls, so a connection with no bytes waiting
// is open, and one that reads end of stream was closed by the far side.
//
// The check peeks the socket without blocking: a read deadline in the past
// does not do it, since Go's poller returns the timeout before it reads.
func connAlive(conn net.Conn) bool {
	tlsConn, isTLS := conn.(*tls.Conn)
	raw := conn
	if isTLS {
		raw = tlsConn.NetConn()
	}
	sc, ok := raw.(syscall.Conn)
	if !ok {
		return true
	}
	rc, err := sc.SyscallConn()
	if err != nil {
		return false
	}

	var n int
	var peekErr error
	buf := make([]byte, 1)
	err = rc.Read(func(fd uintptr) bool {
		n, _, peekErr = syscall.Recvfrom(int(fd), buf, syscall.MSG_PEEK|syscall.MSG_DONTWAIT)
		return true
	})
	if err != nil {
		return false
	}
	switch {
	case errors.Is(peekErr, syscall.EAGAIN), errors.Is(peekErr, syscall.EWOULDBLOCK):
		return true
	case peekErr != nil:
		return false
	case n == 0:
		return false
	case !isTLS:
		// Bytes the namenode sent with no call outstanding.
		return false
	}
	return tlsReadsNothing(tlsConn)
}

// tlsReadsNothing reads the waiting TLS records: a session ticket is consumed
// and the read times out, which leaves the connection usable; a close_notify
// reads end of stream and application data is a byte no call asked for.
func tlsReadsNothing(conn *tls.Conn) bool {
	if err := conn.SetReadDeadline(time.Now().Add(tlsPendingReadWait)); err != nil {
		return false
	}
	buf := make([]byte, 1)
	n, err := conn.Read(buf)
	if clearErr := conn.SetReadDeadline(time.Time{}); clearErr != nil {
		return false
	}
	var netErr net.Error
	return n == 0 && errors.As(err, &netErr) && netErr.Timeout()
}
