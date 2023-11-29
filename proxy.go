package yamrp

import (
	"io"
)

// Listener is on the client side.
type Listener interface {
	io.Reader
	io.Writer
	// Connect to server, return error channel. nil will be passed if successfully connected.
	Connect() chan error
	// Bind the listener. Must be called after chan of Connect() is returned.
	Bind() error
}

// Agent is on the host side.
type Agent interface {
	io.Reader
	io.Writer
	Connect() chan error
}

// Proxy is the client side (both agent and listener) of the YAMRP.
type Proxy struct {
	listeners []Listener
	Agents    []Agent
}
