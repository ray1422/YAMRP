package client

import "github.com/pion/webrtc/v4"

// this file contains the abstraction of the webrtc package

// NewPeerConnBuilder creates a new peer connection builder.
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

// DataChannelAbstract is the abstraction of the data channel.
type DataChannelAbstract interface {
	OnOpen(func())
	OnClose(func())
	Close() error
	Send([]byte) error
	OnMessage(func(webrtc.DataChannelMessage))
}

var _ DataChannelAbstract = &webrtc.DataChannel{}
