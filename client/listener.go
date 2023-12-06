package client

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/google/uuid"
	"github.com/pion/webrtc/v4"
	"github.com/ray1422/yamrp/proto"
	"github.com/ray1422/yamrp/utils/async"
)

var proxyStopTimeout = 5 * time.Second

// ListenerNetConn is on client side.
type ListenerNetConn struct {
	listener  net.Listener
	bindAddr  string
	network   string
	pcBuilder PeerConnBuilder
	peerConn  peerConnAbstract
	// user      UserModel
	offerAPI proto.YAMRPOffererClient
	hostAPI  proto.HostClient
	authAPI  proto.AuthClient
	// hostID is the ID of the host like origin address.
	hostID          string
	invitationToken string
	authToken       string

	// iceBuf buffer the candidates from onICECandidate callback
	iceBuf chan string

	proxies map[string]*Proxy

	ctx          context.Context
	ctxCancel    context.CancelFunc
	proxiesMutex sync.RWMutex
}

// NewListener creates a new TCP listener.
func NewListener(bindAddr string,
	network string,
	pcBuilder PeerConnBuilder,
	offerAPI proto.YAMRPOffererClient,
	authAPI proto.AuthClient,
	hostID string,
) (*ListenerNetConn, error) {
	ctx, cancel := context.WithCancel(context.Background())
	ret := &ListenerNetConn{
		bindAddr:  bindAddr,
		network:   network,
		pcBuilder: pcBuilder,
		offerAPI:  offerAPI,
		authAPI:   authAPI,
		hostID:    hostID,
		iceBuf:    make(chan string, 1024),

		ctxCancel: cancel,
		ctx:       ctx,
		proxies:   make(map[string]*Proxy),
	}
	return ret, nil
}

// // Login is the login function.
// func (l *ListenerNetConn) Login() error {
// 	// FIXME TODO
// 	panic("not implemented")
// 	return nil
// }

// BindListener binds with existing listener. useful for testing.
func (l *ListenerNetConn) BindListener(listen net.Listener) error {
	// TODO stop signal
	for {
		select {
		case <-l.ctx.Done():
			return l.ctx.Err()
		case ret := <-async.AsyncWithErr(func() (net.Conn, error) {
			return listen.Accept()
		}).Ch():
			conn, err := ret.Val, ret.Err
			log.Debugf("accepted connection from %s", conn.RemoteAddr().String())
			if err != nil && err != io.EOF {
				return err
			}
			id := uuid.New().String()
			// create data channel
			dataChannel, err := l.peerConn.CreateDataChannel(id, nil)
			if err != nil {
				log.Errorf("failed to create data channel")
				continue
			}
			dataChannel.OnOpen(func() {
				log.Debugf("data channel %s opened", id)
				// proxy forks its own goroutine to read from the data channel when constructed
				l.startProxy(id, conn, dataChannel)
				// proxy := NewProxy(id, conn, dataChannel)
				// // generate a UUID, maps this UUID to the proxy, so that we can find the proxy by UUID
				// l.proxiesMutex.Lock()
				// l.proxies[proxy.id] = &proxy
				// l.proxiesMutex.Unlock()
			})
		}
	}
}

// startProxy starts a proxy. life cycle of a proxy starts from here,
// and ends on l.cleanupProxy.
func (l *ListenerNetConn) startProxy(proxyID string, conn net.Conn,
	dataChannel DataChannelAbstract) (
	closed async.Future[error]) {
	closed = async.NewFuture[error]()
	eolSig := make(chan error, 1)
	dataChannel.OnClose(func() {
		l.cleanupProxy(proxyID)
		trySendEoL(eolSig, nil)
	})
	proxy := NewProxy(proxyID, conn, dataChannel, eolSig)

	l.proxiesMutex.Lock()
	l.proxies[proxy.id] = &proxy
	l.proxiesMutex.Unlock()

	go func() {
		var err error
		select {
		case errL := <-eolSig:
			err = errL
		case <-l.ctx.Done():
			err = context.Canceled
		}
		if err == nil {
			log.Infof("proxy %s closed due to connection closed without error", proxyID)
		} else if err == io.EOF || err == io.ErrClosedPipe {
			log.Infof("proxy %s closed due to connection closed (io.EOF)", proxyID)
		} else if err == context.Canceled {
			log.Infof("proxy %s closed due to context canceled", proxyID)
		} else {
			log.Errorf("proxy %s closed due to error: %v", proxyID, err)
		}
		l.cleanupProxy(proxyID)
		dataChannel.Close()
		// Clean up the proxies
		log.Debugf("proxy %s has been successfully cleaned up", proxyID)
		closed.Resolve(err)
	}()
	return
}

// cleanupProxy cleans up a proxy.
func (l *ListenerNetConn) cleanupProxy(proxyID string) {
	l.proxiesMutex.Lock()
	defer l.proxiesMutex.Unlock()
	proxy, ok := l.proxies[proxyID]
	if !ok {
		log.Infof("proxy %s not found, should be already cleaned up", proxyID)
		return
	}
	delete(l.proxies, proxyID)
	proxy.Close()

}

// Bind binds the listener.
func (l *ListenerNetConn) Bind() error {
	listen, err := net.Listen(l.network, l.bindAddr)
	if err != nil && err != io.EOF {
		return err
	}
	log.Infof("listening on %s", l.bindAddr)
	l.listener = listen
	go l.BindListener(l.listener)
	return nil
}

// Connect connects to the server. if config is nil, a default config will be used.
func (l *ListenerNetConn) Connect(config *webrtc.Configuration) chan error {
	ch := make(chan error, 1)
	if config == nil {
		config = &DefaultWebRTCConfig
	}
	go func() {
		select {
		case ch <- async.Await(l.setupWebRTC(*config)):
		case <-l.ctx.Done():
			ch <- l.ctx.Err()
		}

	}()
	return ch
}

// initAndSendOffer creates offer, returns answerID and error.
func (l *ListenerNetConn) initAndSendOffer(peerConn peerConnAbstract) (string, error) {

	_, err := peerConn.CreateDataChannel("dummy", nil)
	// dummy data channel doesn't matter since we are not going to use it
	time.Sleep(100 * time.Millisecond)

	sdp, err := l.peerConn.CreateOffer(nil)
	// set local description
	if err != nil && err != io.EOF {
		return "", err
	}
	err = l.peerConn.SetLocalDescription(sdp)
	if err != nil {
		return "", err
	}
	payload, err := json.Marshal(sdp)
	if err != nil {
		return "", err
	}
	log.Debugf("sending offer to %s", l.hostID)
	// send offer to server
	res, err := l.offerAPI.SendOffer(context.TODO(), &proto.SendOfferRequest{
		Token: &proto.AuthToken{
			Token: l.authToken,
		},
		Offer:  string(payload),
		HostId: l.hostID,
	})
	if err != nil && err != io.EOF {
		log.Errorf("failed to send offer to server%v", err)
		return "", err
	}
	log.Debugf("offer sent: %v", res)
	return res.AnswererId, nil
}

func (l *ListenerNetConn) waitForAnswer(answererID string, peerConn peerConnAbstract) async.Future[error] {
	outerRet := async.NewFuture[error]()
	go func() {
		// receive ret from server
		ret, err := l.offerAPI.WaitForAnswer(
			context.TODO(), &proto.WaitForAnswerRequest{
				Token: &proto.AuthToken{
					Token: l.authToken,
				},
				AnswererId: answererID,
			})

		if err != nil {
			outerRet.Resolve(err)
			return
		}

		// set remote description
		answerSDP := webrtc.SessionDescription{
			Type: webrtc.SDPTypeAnswer,
			SDP:  ret.Answer,
		}
		err = l.peerConn.SetRemoteDescription(answerSDP)
		if err != nil {
			outerRet.Resolve(err)
			return
		}
		// answer set
		outerRet.Resolve(nil)
	}()
	return outerRet

}

// exchangeICE should exchange ICE candidates until ctx is canceled,
// usually when the connection is established or failed.
func (l *ListenerNetConn) exchangeICE(ctx context.Context, answererID string, pendingIceCandidates chan string) error {
	// send ICE candidates to server
	remoteIceCh := make(chan *webrtc.ICECandidateInit, 1024)

	// receive remote ICE candidates from server
	go func(remoteIceCh chan *webrtc.ICECandidateInit) {
		conn, err := l.offerAPI.WaitForICECandidate(ctx, &proto.WaitForICECandidateRequest{
			Token: &proto.AuthToken{
				Token: l.authToken,
			},
			AnswererId: answererID,
		})
		if err != nil {
			log.Errorf("failed to receive ICE candidate from server: %v", err)
			return
		}
		for {
			select {
			case <-ctx.Done():
				return
			default:
				ice, err := conn.Recv()
				if err != nil && err != io.EOF {
					log.Errorf("failed to receive ICE candidate from server: %v", err)
					return
				}
				if err == io.EOF {
					log.Infof("ICE candidate stream closed")
					return
				}
				iceInit := webrtc.ICECandidateInit{
					Candidate: ice.Candidate,
				}
				select {
				case <-ctx.Done():
					return
				default:
					remoteIceCh <- &iceInit
				}

			}
		}
	}(remoteIceCh)
	client, err := l.offerAPI.SendIceCandidate(l.ctx)
	if err != nil {
		log.Errorf("failed to send ICE candidate to server: %v", err)
		return err
	}
	for {
		select {
		case <-ctx.Done():
			close(remoteIceCh)
			client.CloseSend()
			return ctx.Err()
		case ice := <-pendingIceCandidates:
			log.Infof("sending ICE candidate to server: %v", ice)
			err = client.Send(&proto.ReplyToRequest{
				Token: &proto.AuthToken{
					Token: l.authToken,
				},
				AnswererId: answererID,
				Body:       ice,
			})
			if err != nil {
				if err != io.EOF {
					log.Errorf("failed to send ICE candidate to server: %v", err)
					continue
				}
				log.Infof("ICE candidate stream closed")
				continue
			}

		case ice := <-remoteIceCh:
			log.Infof("adding ICE candidate: %v", ice)
			err := l.peerConn.AddICECandidate(*ice)
			if err != nil {
				log.Errorf("failed to add ICE candidate: %v", err)
				return err
			}
		}
	}
}

// initWebRTCAsOfferer refactor required
func (l *ListenerNetConn) initWebRTCAsOfferer(config webrtc.Configuration) error {
	ctxSetup, cancelSetup := context.WithCancelCause(l.ctx)
	defer cancelSetup(errors.New("finished setup"))

	peerConn, err := l.pcBuilder.NewPeerConnection(config)
	if err != nil && err != io.EOF {
		return err
	}
	connectedSig := make(chan struct{}, 1)
	connectionFailedSig := make(chan struct{}, 1)
	peerConn.OnConnectionStateChange(func(state webrtc.PeerConnectionState) {
		log.Infof("connection state changed to %s", state.String())
		if state == webrtc.PeerConnectionStateConnected {
			log.Infof("peer connection established")
			connectedSig <- struct{}{}
		}
		if state == webrtc.PeerConnectionStateFailed {
			log.Infof("peer connection failed")
			// l.ctxCancel()
			connectionFailedSig <- struct{}{}
		}
		if state == webrtc.PeerConnectionStateDisconnected {
			// l.ctxCancel()
			log.Infof("connection disconnected")
		}

	})
	// TODO shouldn't set the variable directly
	l.peerConn = peerConn
	pendingIceCandidates := make(chan string, 1024)
	peerConn.OnICECandidate(func(candidate *webrtc.ICECandidate) {
		if candidate == nil {
			return
		}
		log.Infof("OnIceCandidate: %v", candidate)
		pendingIceCandidates <- candidate.ToJSON().Candidate
		// l.iceBuf <- candidate.ToJSON().Candidate
	})
	answererID, err := l.initAndSendOffer(peerConn)
	if err != nil {
		return err
	}

	answerErrFut := l.waitForAnswer(answererID, peerConn)
	errICE := async.Async(func() error { return l.exchangeICE(ctxSetup, answererID, pendingIceCandidates) })

	errAns := answerErrFut.Await()

	if errAns != nil {
		return errAns
	}
	// FIXME: refactor
	// sending ICE candidates to server
	// go func() {
	// 	req, err := l.offerAPI.SendIceCandidate(l.ctx)
	// 	if err != nil && err != io.EOF {
	// 		log.Errorf("failed to send ICE candidate to server: %v", err)
	// 		return
	// 	}
	// 	timeoutCtx, cancel := context.WithTimeout(l.ctx, 2000*time.Second)
	// 	defer cancel()

	// 	for {
	// 		select {
	// 		case <-timeoutCtx.Done():
	// 			return
	// 		case <-req.Context().Done():
	// 			log.Infof("ICE candidate stream closed by server")
	// 			return
	// 		case ice := <-l.iceBuf:
	// 			log.Infof("sending ICE candidate to server: %v", ice)
	// 			err := req.Send(&proto.ReplyToRequest{
	// 				Body: ice,
	// 				Token: &proto.AuthToken{
	// 					Token: l.authToken,
	// 				},
	// 				AnswererId: answererID,
	// 			})
	// 			if err != nil && err != io.EOF {
	// 				log.Errorf("failed to send ICE candidate to server: %v", err)
	// 				return
	// 			}
	// 			if err == io.EOF {
	// 				log.Infof("ICE candidate stream closed")
	// 				return
	// 			}
	// 			log.Infof("ICE candidate sent")
	// 		}
	// 	}
	// }()

	// // receive ICE candidates from server
	// err, canceledErr := async.Async(
	// 	func() error {
	// 		// wait for ICE candidates
	// 		// keep receiving ICE candidates until connection established or server closed the stream
	// 		iceStream, err := l.offerAPI.WaitForICECandidate(context.TODO(), &proto.WaitForICECandidateRequest{
	// 			Token: &proto.AuthToken{
	// 				Token: l.authToken,
	// 			},
	// 			AnswererId: answererID,
	// 		})
	// 		if err != nil && err != io.EOF {
	// 			return err
	// 		}
	// 		type iceWithErr struct {
	// 			ice *webrtc.ICECandidateInit
	// 			err error
	// 		}
	// 		// recv is a helper function to receive ICE candidates asynchronously
	// 		recv := func(stream proto.YAMRPOfferer_WaitForICECandidateClient, alreadyEOF bool) chan iceWithErr {
	// 			if alreadyEOF {
	// 				return nil
	// 			}
	// 			ch := make(chan iceWithErr, 1)
	// 			go func() {
	// 				iceRet, err := stream.Recv()
	// 				if iceRet == nil || err != nil && err != io.EOF {
	// 					ch <- iceWithErr{
	// 						ice: nil,
	// 						err: err,
	// 					}
	// 					return
	// 				} else if err == io.EOF {
	// 					return
	// 				}
	// 				ice := webrtc.ICECandidateInit{
	// 					Candidate: iceRet.Candidate,
	// 				}
	// 				ch <- iceWithErr{
	// 					ice: &ice,
	// 					err: nil,
	// 				}
	// 				return
	// 			}()
	// 			return ch
	// 		}
	// 		connectionEstablishedSignal := make(chan struct{}, 1)
	// 		// TODO should use pub/sub pattern instead of overwriting the callback
	// 		l.peerConn.OnConnectionStateChange(func(state webrtc.PeerConnectionState) {
	// 			log.Infof("connection state changed to %s", state.String())
	// 			if state == webrtc.PeerConnectionStateConnected {
	// 				connectionEstablishedSignal <- struct{}{}
	// 			}
	// 		})
	// 		// timeout ch
	// 		timer := time.NewTimer(2000 * time.Second)
	// 		// TODO timeout should be configurable
	// 		alreadyEOF := false
	// 		for {
	// 			select {
	// 			case <-l.ctx.Done():
	// 				return errors.New("context canceled")
	// 			case <-connectionEstablishedSignal:
	// 				// TODO should un-subscribe the callback
	// 				log.Infof("webRTC connection established")
	// 				return nil
	// 			case iceWithErr := <-recv(iceStream, alreadyEOF):
	// 				ice, err := iceWithErr.ice, iceWithErr.err
	// 				if err != nil && err != io.EOF {
	// 					if err != io.EOF {
	// 						log.Errorf("failed to receive ICE candidate from server: %v", err)
	// 						return err
	// 					}
	// 					alreadyEOF = true

	// 					// return err
	// 				} else if ice != nil {
	// 					log.Infof("adding ICE candidate: %v", ice)
	// 					err := l.peerConn.AddICECandidate(*ice)
	// 					if err != nil && err != io.EOF {
	// 						log.Errorf("failed to add ICE candidate: %v", err)
	// 						// return err
	// 					}
	// 				} else {
	// 					log.Infof("ICE candidate stream closed")
	// 					return nil
	// 				}

	// 			case <-timer.C:
	// 				fmt.Println("timeout")
	// 				return errors.New("timeout")
	// 			}
	// 		}
	// 	},
	// ).AwaitWithCtx(l.ctx)
	// if canceledErr != nil {
	// 	return async.Ret(canceledErr)
	// }
	for {
		select {
		case <-ctxSetup.Done():
			return ctxSetup.Err()
		case <-connectedSig:
			log.Infof("connection established")
			cancelSetup(errors.New("connection established"))
			return nil
		case <-connectionFailedSig:
			log.Infof("connection failed")
			cancelSetup(errors.New("connection failed"))
			return errors.New("connection failed")
		case err := <-errICE.Ch():
			if err != nil {
				return err
			}
			log.Infof("ICE exchange finished. connection should be established")

		}
	}
}

func (l *ListenerNetConn) setupWebRTC(config webrtc.Configuration) async.Future[error] {
	ch := async.AwaitUnordered[error](
		async.Async(func() error { return l.initWebRTCAsOfferer(config) }),
		// may support other NAT passing methods in the future
	)

	for {
		if err, ok := <-ch; ok {
			if err == nil {
				return async.Ret[error](nil)
			} else {
				log.Errorf("failed to setup webrtc: %v", err)
			}
		} else { // channel closed
			return async.Ret[error](errors.New("failed to setup webrtc"))
		}
	}

}
