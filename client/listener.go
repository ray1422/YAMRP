package client

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
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

	proxies      map[string]*Proxy
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
	ret := &ListenerNetConn{
		bindAddr:  bindAddr,
		network:   network,
		pcBuilder: pcBuilder,
		offerAPI:  offerAPI,
		authAPI:   authAPI,
		hostID:    hostID,

		proxies: make(map[string]*Proxy),
	}
	return ret, nil
}

// Login is the login function.
func (l *ListenerNetConn) Login() error {
	// FIXME TODO
	panic("not implemented")
	return nil
}

// BindListener is the internal BindListener function
func (l *ListenerNetConn) BindListener(listen net.Listener) error {
	// TODO stop signal
	for {
		conn, err := listen.Accept()
		if err != nil && err != io.EOF {
			return err
		}
		id := uuid.New().String()
		// create data channel
		dataChannel, err := l.peerConn.CreateDataChannel(id, nil)
		if err != nil && err != io.EOF {
			log.Errorf("failed to create data channel")
			continue
		}
		dataChannel.OnOpen(func() {
			log.Debugf("data channel %s opened", dataChannel.Label())
			// proxy forks its own goroutine to read from the data channel when constructed
			proxy := NewProxy(id, conn, dataChannel)
			// generate a UUID, maps this UUID to the proxy, so that we can find the proxy by UUID
			l.proxiesMutex.Lock()
			l.proxies[proxy.id] = &proxy
			l.proxiesMutex.Unlock()
		})
	}
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
		ch <- async.Await(l.setupWebRTC(*config))
	}()
	return ch
}
func (l *ListenerNetConn) initWebRTCAsOfferer(config webrtc.Configuration) async.Future[error] {
	peerConn, err := l.pcBuilder.NewPeerConnection(config)
	if err != nil && err != io.EOF {
		return async.Ret(err)
	}

	// TODO shouldn't set the variable directly
	l.peerConn = peerConn
	// create offer
	// FIXME
	peerConn.CreateDataChannel("dummy", nil)
	time.Sleep(1 * time.Second)
	sdp, err := l.peerConn.CreateOffer(nil)
	// set local description
	if err != nil && err != io.EOF {
		return async.Ret(err)
	}
	err = l.peerConn.SetLocalDescription(sdp)
	if err != nil && err != io.EOF {
		return async.Ret(err)
	}
	payload, err := json.Marshal(sdp)
	if err != nil && err != io.EOF {
		return async.Ret(err)
	}
	log.Debugf("sending offer to %s", l.hostID)
	// send offer to server
	res, err := l.offerAPI.SendOffer(context.TODO(), &proto.SendOfferRequest{
		Token: &proto.AuthToken{
			Token: l.authToken,
		},
		Offer:  string(payload),
		HostId: l.hostID, // TODO: get host id
	})
	if err != nil && err != io.EOF {
		log.Errorf("failed to send offer to server%v", err)
		return async.Ret(err)
	}
	log.Debugf("offer sent: %v", res)

	// receive ret from server
	ret, err := l.offerAPI.WaitForAnswer(
		context.TODO(), &proto.WaitForAnswerRequest{
			Token: &proto.AuthToken{
				Token: l.authToken,
			},
			AnswererId: res.AnswererId,
		})

	if err != nil && err != io.EOF {
		return async.Ret(err)
	}

	// set remote description
	answerSDP := webrtc.SessionDescription{
		Type: webrtc.SDPTypeAnswer,
		SDP:  ret.Answer,
	}
	err = l.peerConn.SetRemoteDescription(answerSDP)
	if err != nil && err != io.EOF {
		return async.Ret(err)
	}

	// receive ICE candidates from server
	err = async.Await(async.Async(
		func() error {
			// wait for ICE candidates
			// keep receiving ICE candidates until connection established or server closed the stream
			iceStream, err := l.offerAPI.WaitForICECandidate(context.TODO(), &proto.WaitForICECandidateRequest{
				Token: &proto.AuthToken{
					Token: l.authToken,
				},
				AnswererId: res.AnswererId,
			})
			if err != nil && err != io.EOF {
				return err
			}
			type iceWithErr struct {
				ice *webrtc.ICECandidateInit
				err error
			}
			// recv is a helper function to receive ICE candidates asynchronously
			recv := func(stream proto.YAMRPOfferer_WaitForICECandidateClient, alreadyEOF bool) chan iceWithErr {
				if alreadyEOF {
					return nil
				}
				ch := make(chan iceWithErr, 1)
				go func() {
					iceRet, err := stream.Recv()
					if iceRet == nil || err != nil && err != io.EOF {
						ch <- iceWithErr{
							ice: nil,
							err: err,
						}
						return
					} else if err == io.EOF {
						return
					}
					ice := webrtc.ICECandidateInit{
						Candidate: iceRet.Candidate,
					}
					ch <- iceWithErr{
						ice: &ice,
						err: nil,
					}
					return
				}()
				return ch
			}
			connectionEstablishedSignal := make(chan struct{}, 1)
			// TODO should use pub/sub pattern instead of overwriting the callback
			l.peerConn.OnConnectionStateChange(func(state webrtc.PeerConnectionState) {
				if state == webrtc.PeerConnectionStateConnected {
					connectionEstablishedSignal <- struct{}{}
				}
			})
			// timeout ch
			timer := time.NewTimer(20 * time.Second)
			// TODO timeout should be configurable
			alreadyEOF := false
			for {
				select {
				case <-connectionEstablishedSignal:
					// TODO should un-subscribe the callback
					log.Infof("webRTC connection established")
					return nil
				case iceWithErr := <-recv(iceStream, alreadyEOF):
					ice, err := iceWithErr.ice, iceWithErr.err
					if err != nil && err != io.EOF {
						if err != io.EOF {
							log.Errorf("failed to receive ICE candidate from server: %v", err)
							return err
						}
						alreadyEOF = true

						// return err
					} else {
						l.peerConn.AddICECandidate(*ice)
					}

				case <-timer.C:
					fmt.Println("timeout")
					return errors.New("timeout")
				}
			}
		},
	))
	return async.Ret(err)
}

func (l *ListenerNetConn) setupWebRTC(config webrtc.Configuration) async.Future[error] {
	ch := async.AwaitUnordered[error](
		l.initWebRTCAsOfferer(config),
		// TODO should be init as answerer as well
		// async.Ret(async.Await(l.initWebRTCAsOfferer(config))),
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
