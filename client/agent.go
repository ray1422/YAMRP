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

// NewPCBuilder creates a new peer connection builder.
func NewPeerConnBuilder() PeerConnBuilder {
	return peerConnBuilderImpl{}
}

// peerConnBuilder
type peerConnBuilderImpl struct{}

func (p peerConnBuilderImpl) NewPeerConnection(config webrtc.Configuration) (peerConnAbstract, error) {
	return webrtc.NewPeerConnection(config)
}

// PeerConnBuilder is the interface for peer connection builder.
type PeerConnBuilder interface {
	NewPeerConnection(config webrtc.Configuration) (peerConnAbstract, error)
}
type peerConnAbstract interface {
	OnICECandidate(func(*webrtc.ICECandidate))
	AddICECandidate(candidate webrtc.ICECandidateInit) error
	SetRemoteDescription(desc webrtc.SessionDescription) error
	SetLocalDescription(desc webrtc.SessionDescription) error
	CreateOffer(options *webrtc.OfferOptions) (webrtc.SessionDescription, error)
	CreateAnswer(options *webrtc.AnswerOptions) (webrtc.SessionDescription, error)
	OnConnectionStateChange(func(webrtc.PeerConnectionState))
	CreateDataChannel(label string, dataChannelInit *webrtc.DataChannelInit) (*webrtc.DataChannel, error)
	OnDataChannel(func(*webrtc.DataChannel))
	RemoteDescription() *webrtc.SessionDescription
}

// AgentNetConn is the TCP agent.
type AgentNetConn struct {
	addr string
	// network is either tcp or udp
	network string

	pcBuilder PeerConnBuilder
	peerConn  peerConnAbstract

	user   UserModel
	ansAPI proto.YAMRPAnswererClient
}

// NewAgent creates a new TCP agent.
func NewAgent(addr string,
	network string,
	pcBuilder PeerConnBuilder,
	user UserModel,
	ansAPI proto.YAMRPAnswererClient,
	// hostAPI proto.HostClient, // FIXME
) (*AgentNetConn, error) {
	ret := &AgentNetConn{
		addr:    addr,
		network: network,

		user:      user,
		pcBuilder: pcBuilder,
		ansAPI:    ansAPI,
	}
	return ret, nil
}

// DefaultWebRTCConfig is the default config for WebRTC.
var DefaultWebRTCConfig = webrtc.Configuration{
	ICEServers: []webrtc.ICEServer{
		{
			URLs: []string{
				"stun.l.google.com:19302",
				"stun1.l.google.com:19302",
				"stun2.l.google.com:19302",
				"stun3.l.google.com:19302",
				"stun4.l.google.com:19302",
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
	})
	if err != nil && err != io.EOF {
		go func() { ch <- utils.Result[peerConnAbstract]{Err: err} }()
		return ch
	}
	answerID := ret.AnswererId

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
		for peerConn.RemoteDescription() == nil {
			// FIXME should queued the ice candidate
			log.Warn("FIXME setRemoteDescription is not called yet")
			time.Sleep(1 * time.Second)
		}
		err = iceCandidateStream.Send(&proto.ReplyToAnswererRequest{
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
	// peerConn.CreateDataChannel("dummy", nil)
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
	_, err = a.ansAPI.SendAnswer(context.TODO(), &proto.ReplyToAnswererRequest{
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
	log.Debugf("received data channel")
	// create a proxy
	// TODO: should passing dial builder
	dial, err := net.Dial(a.network, a.addr)
	if err != nil && err != io.EOF {
		log.Errorf("failed to dial: %v", err)
		return
	}

	proxy := NewProxy(d.Label(), dial, d)
	d.OnClose(func() {
		proxy.Close()
	})

}
