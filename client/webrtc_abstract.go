package client

import "github.com/pion/webrtc/v4"

// this file contains the abstraction of the webrtc package

// PeerConnWrapper is the wrapper of the peer connection. to implement
// createDataChannel method.
type PeerConnWrapper struct {
	// must be non-nil
	*webrtc.PeerConnection
}

var _ peerConnAbstract = (*PeerConnWrapper)(nil)

// NewPeerConnWrapperWithErr creates a new peer connection wrapper.
func NewPeerConnWrapperWithErr(pc *webrtc.PeerConnection, err error) (*PeerConnWrapper, error) {
	if pc == nil {
		return nil, err
	}
	return &PeerConnWrapper{pc}, err
}

// NewPeerConnWrapper creates a new peer connection wrapper.
func NewPeerConnWrapper(pc *webrtc.PeerConnection) *PeerConnWrapper {
	return &PeerConnWrapper{pc}
}

// CreateDataChannel calls the original method.
func (p *PeerConnWrapper) CreateDataChannel(
	label string, dataChannelInit *webrtc.DataChannelInit) (
	DataChannelAbstract, error) {
	return p.PeerConnection.CreateDataChannel(label, dataChannelInit)
}

// NewPeerConnBuilder creates a new peer connection builder.
func NewPeerConnBuilder() PeerConnBuilder {
	return peerConnBuilderImpl{}
}

// peerConnBuilder
type peerConnBuilderImpl struct{}

func (p peerConnBuilderImpl) NewPeerConnection(config webrtc.Configuration) (peerConnAbstract, error) {
	return NewPeerConnWrapperWithErr(webrtc.NewPeerConnection(config))
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
	CreateDataChannel(label string, dataChannelInit *webrtc.DataChannelInit) (DataChannelAbstract, error)
	OnDataChannel(func(*webrtc.DataChannel))
	RemoteDescription() *webrtc.SessionDescription
	Close() error
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
