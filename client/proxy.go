package client

import (
	"net"

	log "github.com/sirupsen/logrus"

	"github.com/pion/webrtc/v4"
)

// DataChannelAbstract is the abstraction of the data channel.
type DataChannelAbstract interface {
	OnOpen(func())
	Close() error
	OnClose(func())
	Send([]byte) error
	OnMessage(func(webrtc.DataChannelMessage))
}

var _ DataChannelAbstract = &webrtc.DataChannel{}

// Proxy is the proxy of the listener. a proxy will create a Proxy for each connection.
type Proxy struct {
	peerConn    peerConnAbstract
	conn        net.Conn
	dataChannel DataChannelAbstract // *webrtc.DataChannel
	writingBuf  chan []byte

	stopSigWrite chan struct{}
	stopSigRead  chan struct{}

	id string
}

// NewProxy creates a new ListenerProxy.
func NewProxy(id string, conn net.Conn, dataChannel DataChannelAbstract) Proxy {

	lp := Proxy{
		conn:         conn,
		id:           id,
		dataChannel:  dataChannel,
		writingBuf:   make(chan []byte, 1024),
		stopSigWrite: make(chan struct{}, 1),
		stopSigRead:  make(chan struct{}, 1),
	}
	lp.dataChannel.OnMessage(func(msg webrtc.DataChannelMessage) {
		// log.Debugf(lp.id, "received %d bytes", ", msg.Data)
		lp.writingBuf <- msg.Data
	})
	go lp.readingLoop()
	go lp.writingLoop()
	return lp
}

// Close the connection.
func (lp *Proxy) Close() error {
	go func() {
		lp.stopSigWrite <- struct{}{}
		lp.stopSigRead <- struct{}{}
	}()
	err := lp.dataChannel.Close()
	if err != nil {
		log.Errorf("%s failed to close data channel, but still need to close the connection: %v", lp.id, err)

	}
	return lp.conn.Close()
}

// readingLoop keeps reading from socket and write to the data channel.
func (lp *Proxy) readingLoop() {
	readBuf := make([]byte, 1024)
	for {
		select {
		case <-lp.stopSigRead:
			return
		default:
			n, err := lp.conn.Read(readBuf)
			if err == nil && n > 0 {

				err := lp.dataChannel.Send(readBuf[:n])
				if err != nil {
					log.Errorf("%s failed to write:%v", lp.id, err)
				}

				// log.Debugf(lp.id, "received %d bytes", n)
			}
		}

	}
}

// writingLoop handles the incoming data as data channel handler push the data to `WritingBuf`.
func (lp *Proxy) writingLoop() {
	for {
		select {
		case <-lp.stopSigWrite:
			return
		case data := <-lp.writingBuf:
			// data should less than or equals to 1024 bytes
			_, err := lp.conn.Write(data)
			if err != nil {
				log.Errorf(lp.id, "failed to write: %v", err)
			}
		}

	}
}
