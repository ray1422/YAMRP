package client

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/pion/webrtc/v4"

	"github.com/ray1422/yamrp/proto"
	"github.com/ray1422/yamrp/utils"
)

// AgentNetConn is the TCP agent.
type AgentNetConn struct {
	addr string
	// network is either tcp or udp
	network string

	pcBuilder PeerConnBuilder
	peerConn  peerConnAbstract

	user   UserModel
	ansAPI proto.YAMRPAnswererClient
	hostID string
}

// NewAgent creates a new TCP agent.
func NewAgent(addr string,
	network string,
	pcBuilder PeerConnBuilder,
	user UserModel,
	ansAPI proto.YAMRPAnswererClient,
	hostID string,
	// hostAPI proto.HostClient, // FIXME
) (*AgentNetConn, error) {
	ret := &AgentNetConn{
		addr:    addr,
		network: network,

		user:      user,
		pcBuilder: pcBuilder,
		ansAPI:    ansAPI,
		hostID:    hostID,
	}
	return ret, nil
}

// DefaultWebRTCConfig is the default config for WebRTC.
var DefaultWebRTCConfig = webrtc.Configuration{
	ICEServers: []webrtc.ICEServer{
		{
			URLs: []string{
				"stun:stun.l.google.com:19302",
				"stun:stun1.l.google.com:19302",
				"stun:stun2.l.google.com:19302",
				"stun:stun3.l.google.com:19302",
				"stun:stun4.l.google.com:19302",
			},
		},
	},
}

// Connect connects to the server. if config is nil, a default config will be used.
func (a *AgentNetConn) Connect(config *webrtc.Configuration) chan error {
	ch := make(chan error, 1)
	// setup peer connection
	if config == nil {
		config = &DefaultWebRTCConfig
	}

	go func() {
		ch <- <-a.setupWebRTC(*config)
	}()
	return ch

}

func (a *AgentNetConn) setupWebRTC(config webrtc.Configuration) chan error {
	// TODO should refactor this function
	ch := make(chan error, 1)
	go func() {
		u := <-a.initWebRTCAsAnswerer(config)
		peerConn, err := u.Val, u.Err
		if err != nil && err != io.EOF {
			log.Errorf("failed to setup webrtc for agent: %v", err)
			ch <- err
			return
		}
		// success
		a.peerConn = peerConn
		a.peerConn.OnDataChannel(a.onDataChannelHandler)
		ch <- nil
		return
	}()
	return ch
}

func (a *AgentNetConn) initWebRTCAsAnswerer(config webrtc.Configuration) chan utils.Result[peerConnAbstract] {
	ch := make(chan utils.Result[peerConnAbstract], 1)
	peerConn, err := a.pcBuilder.NewPeerConnection(config)
	if err != nil && err != io.EOF {
		go func() { ch <- utils.Result[peerConnAbstract]{Err: err} }()
		return ch
	}

	// TODO shouldn't set the variable directly, return it instead
	// a.peerConn = peerConn
	// TODO timeout
	ret, err := a.ansAPI.WaitForOffer(context.TODO(), &proto.WaitForOfferRequest{
		Token: &proto.AuthToken{
			Token: a.user.Token(),
		},
		HostId: a.hostID,
	})
	if err != nil && err != io.EOF {
		go func() { ch <- utils.Result[peerConnAbstract]{Err: err} }()
		return ch
	}
	answerID := ret.AnswererId
	go func() {
		// recv ice candidate
		for {
			res, err := a.ansAPI.WaitForICECandidate(context.TODO(),
				&proto.WaitForICECandidateRequest{
					Token: &proto.AuthToken{
						Token: a.user.Token(),
					},
					AnswererId: answerID,
				},
			)
			if err != nil && err != io.EOF {
				log.Errorf("failed to receive ice candidate: %v", err)
				return
			}
			timeoutCtx, cancel := context.WithTimeout(context.Background(), 1000*time.Second)
			defer cancel()
			for {
				select {
				case <-timeoutCtx.Done():
					log.Warnf("timeout stop receiving ice candidate")
					return
				case <-res.Context().Done():
					log.Infof("ice candidate stream closed by server")
					return
				case ice := <-func() chan string {
					ch := make(chan string, 1)
					go func() {
						r, err := res.Recv()
						log.Debugf("received ice candidate %v", r)
						if err != nil && err != io.EOF {
							log.Warnf("error when receiving ice candidate: %v", err)
							return
						}
						if r == nil {
							log.Warnf("ice candidate stream closed")
							return
						}
						ch <- r.GetCandidate()
					}()
					return ch
				}():
					log.Debugf("add ice candidate %v", ice)
					err := peerConn.AddICECandidate(webrtc.ICECandidateInit{
						Candidate: ice,
					})
					if err != nil && err != io.EOF {
						log.Errorf("failed to add ice candidate: %v", err)
						return
					}
				}
			}
		}
	}()
	// send ice candidate
	// send ice candidate is stream
	iceCandidateStream, err := a.ansAPI.SendIceCandidate(context.TODO())
	if err != nil && err != io.EOF {
		go func() { ch <- utils.Result[peerConnAbstract]{Err: err} }()
		return ch
	}

	peerConn.OnICECandidate(func(c *webrtc.ICECandidate) {
		if c == nil {
			return
		}
		log.Infof("sending ice candidate %v", c)
		for peerConn.RemoteDescription() == nil {
			// FIXME should queued the ice candidate
			log.Warn("FIXME setRemoteDescription is not called yet")
			time.Sleep(1 * time.Second)
		}
		err = iceCandidateStream.Send(&proto.ReplyToRequest{
			Token: &proto.AuthToken{
				Token: a.user.Token(),
			},
			AnswererId: answerID,
			Body:       c.ToJSON().Candidate,
		})
		if err != nil && err != io.EOF {
			// just log
			log.Infof("failed to send ice candidate: %v", err)
		}
		log.Infof("sent ice candidate %v", c)
	})

	// get an offer
	descJSON := ret.Body
	// decode offer
	sdp := webrtc.SessionDescription{}
	err = json.Unmarshal([]byte(descJSON), &sdp)
	if err != nil && err != io.EOF {
		go func() { ch <- utils.Result[peerConnAbstract]{Err: err} }()
		return ch
	}
	// FIXME
	peerConn.CreateDataChannel("dummy_agent", nil)
	// time.Sleep(500 * time.Millisecond)
	// set remote description

	err = peerConn.SetRemoteDescription(sdp)
	if err != nil && err != io.EOF {
		fmt.Println("reach here, err:", err)
		go func() { ch <- utils.Result[peerConnAbstract]{Err: err} }()
		return ch
	}
	// create and send answer
	answer, err := peerConn.CreateAnswer(nil)
	if err != nil && err != io.EOF {
		go func() { ch <- utils.Result[peerConnAbstract]{Err: err} }()
		return ch
	}
	err = peerConn.SetLocalDescription(answer)
	if err != nil && err != io.EOF {
		go func() { ch <- utils.Result[peerConnAbstract]{Err: err} }()
		return ch
	}
	// send answer
	_, err = a.ansAPI.SendAnswer(context.TODO(), &proto.ReplyToRequest{
		Token: &proto.AuthToken{
			Token: a.user.Token(),
		},
		Body:       answer.SDP,
		AnswererId: answerID,
	})
	if err != nil && err != io.EOF {
		go func() { ch <- utils.Result[peerConnAbstract]{Err: err} }()
	}
	ch <- utils.Result[peerConnAbstract]{Err: nil, Val: peerConn}
	return ch
}

func (a *AgentNetConn) onDataChannelHandler(d *webrtc.DataChannel) {
	log.Debugf("received data channel")
	// create a proxy
	// TODO: should passing dial builder
	dial, err := net.Dial(a.network, a.addr)

	if err != nil && err != io.EOF {
		log.Errorf("failed to dial: %v", err)
		return
	}
	eolSig := make(chan error, 1)
	proxy := NewProxy(d.Label(), dial, d, eolSig)
	d.OnClose(func() {
		select {
		case eolSig <- io.EOF:
			log.Infof("received close signal from data channel, EoL signal sent")
		default:
			break
		}

	})
	go func() {
		<-eolSig
		proxy.Close()
		d.Close()
		log.Infof("proxy %s successfully closed", proxy.id)
	}()

}
