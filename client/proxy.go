package client

import (
	"errors"
	"io"
	"net"
	"time"

	"github.com/ray1422/yamrp/utils/async"
	log "github.com/sirupsen/logrus"

	"github.com/pion/webrtc/v4"
)

var errProxyStooping = errors.New("proxy is stopping")

// Proxy is the proxy of the listener. a proxy will create a Proxy for each connection.
type Proxy struct {
	conn        net.Conn
	dataChannel DataChannelAbstract // *webrtc.DataChannel
	writingBuf  chan []byte
	// EoLSig needs at least one buffer
	EoLSig       chan<- error
	stopSigWrite chan struct{}
	stopSigRead  chan struct{}
	id           string

	// used for control stooping, must be buffered with size 1
	stopped chan struct{}
}

// NewProxy creates a new ListenerProxy.
// proxy doesn't own [conn, dataChannel, eolSig], so it should close by listener.
//
// usage:
//
//	// initialize connection, data channel, EoL signal, etc.
//	proxy := NewProxy(id, conn, dataChannel, eolSig)
//	// do something
//	// ...
//	listen for EoL signal
//	EoL := <-eolSig
//	// close the proxy
//	proxy.Close()
//	// now close connection, data channel, EoL signal, etc.
func NewProxy(id string, conn net.Conn, dataChannel DataChannelAbstract, eolSig chan<- error) Proxy {
	lp := Proxy{
		conn:         conn,
		id:           id,
		dataChannel:  dataChannel,
		writingBuf:   make(chan []byte, 1024),
		EoLSig:       eolSig,
		stopSigWrite: make(chan struct{}),
		stopSigRead:  make(chan struct{}),
		stopped:      make(chan struct{}, 1),
	}
	lp.dataChannel.OnMessage(func(msg webrtc.DataChannelMessage) {
		// log.Debugf(lp.id, "received %d bytes", ", msg.Data)
		lp.writingBuf <- msg.Data
	})
	go lp.readingLoop()
	go lp.writingLoop()
	return lp
}

// Close the connection. Close should only be called by listener and only once.
// for closing proxy inside, it should sending EoL signal to the listener.
func (lp *Proxy) Close() error {

	select {
	case lp.stopped <- struct{}{}:
	default:
		log.Errorf("%s failed to send stop signal to writing loop", lp.id)
		return errProxyStooping
	}
	err := lp.conn.Close()
	if err != nil {
		log.Infof("%s failed to close connection, probably already closed: ", lp.id)
	}
	log.Debugf("Proxy %s closing", lp.id)
	select {
	case lp.stopSigWrite <- struct{}{}:
	default:
		log.Errorf("%s failed to send stop signal to writing loop", lp.id)
	}

	select {
	case lp.stopSigRead <- struct{}{}:
	default:
		log.Errorf("%s failed to send stop signal to reading loop", lp.id)
	}

	close(lp.stopSigWrite)
	close(lp.stopSigRead)

	/**
	 *		!!! data channel should be closed by listener !!!
	 */

	// err := lp.dataChannel.Close()
	// if err != nil {
	// 	log.Warnf("%s failed to close data channel, but still need to close the connection: %v", lp.id, err)
	// }
	return nil
}

// trySendEoL returns true if and only if EOL sent immediately, otherwise returns false.
//
// usage:
//
//	if trySendEoL(eolSig, err) != nil {
//		// can't send EoL signal, break loop
//	}
func trySendEoL(eolSig chan<- error, err error) error {
	select {
	case eolSig <- err:
		log.Infof("EoL signal sent")
		return nil
	default:
		log.Infof("EoL signal may be already sent by other goroutine")
		return errors.New("EoL signal may be already sent by other goroutine")
	}
}

// readingLoop keeps reading from socket and write to the data channel.
func (lp *Proxy) readingLoop() {
	readBuf := make([]byte, 1024)
	for {
		select {
		case _, ok := <-lp.stopSigRead:
			if !ok {
				log.Warnf("stop reading loop as channel closed")
			} else {
				log.Infof("stop reading loop as stop signal received")
			}
			return
		// error will be return if and only if stopping it should stop the loop
		case err := <-async.Async(func() error {
			n, err := lp.conn.Read(readBuf)
			if err == net.ErrClosed {
				err = io.EOF
			}
			if n > 0 {
				err := lp.dataChannel.Send(readBuf[:n])
				if err != nil {
					log.Errorf("%s failed to write:%v", lp.id, err)
				}
				return err
			} else if err == io.EOF || err == io.ErrClosedPipe {
				log.Debugf("%s connection closed by here (EOF)", lp.id)
			} else {
				log.Errorf("%s failed to read: %v", lp.id, err)
			}
			return err
		}).Ch():
			// err != nil implies that the connection is closed
			// so we should send EoL signal to the listener
			// once eolHelper fails, it means that the EoL signal
			// is already sent by other goroutine, so we can break
			// the loop
			if err != nil {
				if err == net.ErrClosed || err == io.ErrClosedPipe {
					err = io.EOF
				}
				if err != io.EOF && err != io.ErrClosedPipe {
					log.Errorf("%s failed to read: %v", lp.id, err)
				} else {
					log.Debugf("%s connection closed here (EOF)", lp.id)
				}
				if trySendEoL(lp.EoLSig, err) != nil {
					return
				}
			}

		}
	}
}

// writingLoop handles the incoming data as data channel handler push the data to `WritingBuf`.
func (lp *Proxy) writingLoop() {
	for {
		select {
		case _, ok := <-lp.stopSigWrite:
			if !ok {
				log.Warnf("stop writing loop as channel closed")
				return
			}
			log.Infof("stop writing loop as stop signal received")
			return
		case data := <-lp.writingBuf:
			select {
			case err := <-async.Async(func() error {
				// data should less than or equals to TCP buffer size
				_, err := lp.conn.Write(data)
				if err == io.EOF || err == io.ErrClosedPipe {
					log.Debugf("%s connection closed by client", lp.id)
					err = io.EOF
				}
				if err != nil && err != io.EOF {
					log.Errorf(lp.id, "failed to write: %v", err)
				}
				return err
			}).Ch():
				if err != nil && trySendEoL(lp.EoLSig, err) != nil {
					return
				}
			case <-time.After(100 * time.Millisecond):
				log.Warnf("failed to write to socket, timeout")
				if trySendEoL(lp.EoLSig, errors.New("failed to write to socket, timeout")) != nil {
					return
				}
			case <-lp.stopSigWrite:
				log.Infof("stop writing loop as stop signal received")
				return
			}

		}
	}
}
