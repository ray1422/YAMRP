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

	// iceBuf buffer the candidates from onICECandidate callback
	iceBuf chan string

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
		iceBuf:    make(chan string, 1024),

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

	peerConn.OnICECandidate(func(candidate *webrtc.ICECandidate) {
		if candidate == nil {
			return
		}
		log.Infof("OnIceCandidate: %v", candidate)
		l.iceBuf <- candidate.ToJSON().Candidate
	})

	// create offer
	// FIXME
	outputTracks := map[string]*webrtc.TrackLocalStaticRTP{}

	// Create Track that we send video back to browser on
	outputTrack, err := webrtc.NewTrackLocalStaticRTP(webrtc.RTPCodecCapability{MimeType: webrtc.MimeTypeVP8}, "video_q", "pion_q")
	if err != nil {
		panic(err)
	}
	outputTracks["q"] = outputTrack

	outputTrack, err = webrtc.NewTrackLocalStaticRTP(webrtc.RTPCodecCapability{MimeType: webrtc.MimeTypeVP8}, "video_h", "pion_h")
	if err != nil {
		panic(err)
	}
	outputTracks["h"] = outputTrack

	outputTrack, err = webrtc.NewTrackLocalStaticRTP(webrtc.RTPCodecCapability{MimeType: webrtc.MimeTypeVP8}, "video_f", "pion_f")
	if err != nil {
		panic(err)
	}
	outputTracks["f"] = outputTrack

	// Add this newly created track to the PeerConnection
	if _, err = peerConn.AddTrack(outputTracks["q"]); err != nil {
		panic(err)
	}
	if _, err = peerConn.AddTrack(outputTracks["h"]); err != nil {
		panic(err)
	}
	if _, err = peerConn.AddTrack(outputTracks["f"]); err != nil {
		panic(err)
	}
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

	// FIXME: refactor
	// sending ICE candidates to server
	go func() {
		req, err := l.offerAPI.SendIceCandidate(context.TODO())
		if err != nil && err != io.EOF {
			log.Errorf("failed to send ICE candidate to server: %v", err)
			return
		}
		timeoutCtx, cancel := context.WithTimeout(context.Background(), 2000*time.Second)
		defer cancel()

		for {
			select {
			case <-timeoutCtx.Done():
				return
			case <-req.Context().Done():
				log.Infof("ICE candidate stream closed by server")
				return
			case ice := <-l.iceBuf:
				log.Infof("sending ICE candidate to server: %v", ice)
				err := req.Send(&proto.ReplyToRequest{
					Body: ice,
					Token: &proto.AuthToken{
						Token: l.authToken,
					},
					AnswererId: res.AnswererId,
				})
				if err != nil && err != io.EOF {
					log.Errorf("failed to send ICE candidate to server: %v", err)
					return
				}
				if err == io.EOF {
					log.Infof("ICE candidate stream closed")
					return
				}
				log.Infof("ICE candidate sent")
			}
		}
	}()

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
				log.Infof("connection state changed to %s", state.String())
				if state == webrtc.PeerConnectionStateConnected {
					connectionEstablishedSignal <- struct{}{}
				}
			})
			// timeout ch
			timer := time.NewTimer(2000 * time.Second)
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
					} else if ice != nil {
						log.Infof("adding ICE candidate: %v", ice)
						err := l.peerConn.AddICECandidate(*ice)
						if err != nil && err != io.EOF {
							log.Errorf("failed to add ICE candidate: %v", err)
							// return err
						}
					} else {
						log.Infof("ICE candidate stream closed")
						return nil
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
