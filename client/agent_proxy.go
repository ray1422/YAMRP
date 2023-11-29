package client

import "net"

// AgentProxy is the connection of host side. That is, one incoming connection from the listener.
type AgentProxy struct {
	peerConn    peerConnAbstract
	conn        net.Conn
	dataChannel DataChannelAbstract // *webrtc.DataChannel
	writingBuf  chan []byte

	stopSigWrite chan struct{}
	stopSigRead  chan struct{}

	id string
}
