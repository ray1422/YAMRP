package mock

import (
	"context"
	"io"
	"net"
	"sync"
	"sync/atomic"
)

// Listener is a mock listener.
// copy from somewhere in the internet.
// thanks to the author.
type Listener struct {
	ch      chan net.Conn
	closeCh chan struct{}
	closed  uint32
	sync.Mutex
}

// NewMockListener returns a mock listener.
func NewMockListener() *Listener {
	return &Listener{
		ch:      make(chan net.Conn),
		closeCh: make(chan struct{}),
	}
}

// Accept returns a mock connection.
func (l *Listener) Accept() (net.Conn, error) {
	select {
	case c := <-l.ch:
		return c, nil
	case <-l.closeCh:
		e := io.EOF
		return nil, e
	}
	// }
	// server, client := net.Pipe()
	// m.Clients = append(m.Clients, client)
	// return server, nil
}

// Close closes the listener.
func (l *Listener) Close() error {
	l.Lock()
	defer l.Unlock()
	if l.closed == 1 {
		return io.EOF
	}
	l.closed = 1
	close(l.closeCh)
	return nil
}

// Dial dials to the listener.
func (l *Listener) Dial(network, addr string) (net.Conn, error) {
	return l.DialContext(context.Background(), addr)
}

// DialContext dials to the listener.
func (l *Listener) DialContext(ctx context.Context, addr string) (conn net.Conn, e error) {
	// check closed
	if atomic.LoadUint32(&l.closed) == 1 {
		return nil, io.EOF
	}
	// create a new connection
	// pipe
	c0, c1 := net.Pipe()
	// waiting accepted or closed or done
	select {
	case <-ctx.Done():
		e = ctx.Err()
	case l.ch <- c0:
		conn = c1
	case <-l.closeCh:
		c0.Close()
		c1.Close()
		e = io.EOF
	}
	return

}

type addr struct{}

func (a *addr) Network() string {
	return "tcp"
}
func (a *addr) String() string {
	return "test_addr"
}

// Addr returns a mock address.
func (l *Listener) Addr() net.Addr {
	return &addr{}
}
