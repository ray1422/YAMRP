// Code generated by MockGen. DO NOT EDIT.
// Source: ./client/agent.go
//
// Generated by this command:
//
//	mockgen --source=./client/agent.go --package client
//
// Package client is a generated GoMock package.
package client

import (
	reflect "reflect"

	webrtc "github.com/pion/webrtc/v4"
	gomock "go.uber.org/mock/gomock"
)

// MockPeerConnBuilder is a mock of PeerConnBuilder interface.
type MockPeerConnBuilder struct {
	ctrl     *gomock.Controller
	recorder *MockPeerConnBuilderMockRecorder
}

// MockPeerConnBuilderMockRecorder is the mock recorder for MockPeerConnBuilder.
type MockPeerConnBuilderMockRecorder struct {
	mock *MockPeerConnBuilder
}

// NewMockPeerConnBuilder creates a new mock instance.
func NewMockPeerConnBuilder(ctrl *gomock.Controller) *MockPeerConnBuilder {
	mock := &MockPeerConnBuilder{ctrl: ctrl}
	mock.recorder = &MockPeerConnBuilderMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockPeerConnBuilder) EXPECT() *MockPeerConnBuilderMockRecorder {
	return m.recorder
}

// NewPeerConnection mocks base method.
func (m *MockPeerConnBuilder) NewPeerConnection(config webrtc.Configuration) (peerConnAbstract, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "NewPeerConnection", config)
	ret0, _ := ret[0].(peerConnAbstract)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// NewPeerConnection indicates an expected call of NewPeerConnection.
func (mr *MockPeerConnBuilderMockRecorder) NewPeerConnection(config any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "NewPeerConnection", reflect.TypeOf((*MockPeerConnBuilder)(nil).NewPeerConnection), config)
}

// MockpeerConnAbstract is a mock of peerConnAbstract interface.
type MockpeerConnAbstract struct {
	ctrl     *gomock.Controller
	recorder *MockpeerConnAbstractMockRecorder
}

// MockpeerConnAbstractMockRecorder is the mock recorder for MockpeerConnAbstract.
type MockpeerConnAbstractMockRecorder struct {
	mock *MockpeerConnAbstract
}

// NewMockpeerConnAbstract creates a new mock instance.
func NewMockpeerConnAbstract(ctrl *gomock.Controller) *MockpeerConnAbstract {
	mock := &MockpeerConnAbstract{ctrl: ctrl}
	mock.recorder = &MockpeerConnAbstractMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockpeerConnAbstract) EXPECT() *MockpeerConnAbstractMockRecorder {
	return m.recorder
}

// AddICECandidate mocks base method.
func (m *MockpeerConnAbstract) AddICECandidate(candidate webrtc.ICECandidateInit) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "AddICECandidate", candidate)
	ret0, _ := ret[0].(error)
	return ret0
}

// AddICECandidate indicates an expected call of AddICECandidate.
func (mr *MockpeerConnAbstractMockRecorder) AddICECandidate(candidate any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "AddICECandidate", reflect.TypeOf((*MockpeerConnAbstract)(nil).AddICECandidate), candidate)
}

// AddTrack mocks base method.
func (m *MockpeerConnAbstract) AddTrack(track webrtc.TrackLocal) (*webrtc.RTPSender, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "AddTrack", track)
	ret0, _ := ret[0].(*webrtc.RTPSender)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// AddTrack indicates an expected call of AddTrack.
func (mr *MockpeerConnAbstractMockRecorder) AddTrack(track any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "AddTrack", reflect.TypeOf((*MockpeerConnAbstract)(nil).AddTrack), track)
}

// CreateAnswer mocks base method.
func (m *MockpeerConnAbstract) CreateAnswer(options *webrtc.AnswerOptions) (webrtc.SessionDescription, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CreateAnswer", options)
	ret0, _ := ret[0].(webrtc.SessionDescription)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// CreateAnswer indicates an expected call of CreateAnswer.
func (mr *MockpeerConnAbstractMockRecorder) CreateAnswer(options any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CreateAnswer", reflect.TypeOf((*MockpeerConnAbstract)(nil).CreateAnswer), options)
}

// CreateDataChannel mocks base method.
func (m *MockpeerConnAbstract) CreateDataChannel(label string, dataChannelInit *webrtc.DataChannelInit) (*webrtc.DataChannel, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CreateDataChannel", label, dataChannelInit)
	ret0, _ := ret[0].(*webrtc.DataChannel)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// CreateDataChannel indicates an expected call of CreateDataChannel.
func (mr *MockpeerConnAbstractMockRecorder) CreateDataChannel(label, dataChannelInit any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CreateDataChannel", reflect.TypeOf((*MockpeerConnAbstract)(nil).CreateDataChannel), label, dataChannelInit)
}

// CreateOffer mocks base method.
func (m *MockpeerConnAbstract) CreateOffer(options *webrtc.OfferOptions) (webrtc.SessionDescription, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CreateOffer", options)
	ret0, _ := ret[0].(webrtc.SessionDescription)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// CreateOffer indicates an expected call of CreateOffer.
func (mr *MockpeerConnAbstractMockRecorder) CreateOffer(options any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CreateOffer", reflect.TypeOf((*MockpeerConnAbstract)(nil).CreateOffer), options)
}

// OnConnectionStateChange mocks base method.
func (m *MockpeerConnAbstract) OnConnectionStateChange(arg0 func(webrtc.PeerConnectionState)) {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "OnConnectionStateChange", arg0)
}

// OnConnectionStateChange indicates an expected call of OnConnectionStateChange.
func (mr *MockpeerConnAbstractMockRecorder) OnConnectionStateChange(arg0 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "OnConnectionStateChange", reflect.TypeOf((*MockpeerConnAbstract)(nil).OnConnectionStateChange), arg0)
}

// OnDataChannel mocks base method.
func (m *MockpeerConnAbstract) OnDataChannel(arg0 func(*webrtc.DataChannel)) {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "OnDataChannel", arg0)
}

// OnDataChannel indicates an expected call of OnDataChannel.
func (mr *MockpeerConnAbstractMockRecorder) OnDataChannel(arg0 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "OnDataChannel", reflect.TypeOf((*MockpeerConnAbstract)(nil).OnDataChannel), arg0)
}

// OnICECandidate mocks base method.
func (m *MockpeerConnAbstract) OnICECandidate(arg0 func(*webrtc.ICECandidate)) {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "OnICECandidate", arg0)
}

// OnICECandidate indicates an expected call of OnICECandidate.
func (mr *MockpeerConnAbstractMockRecorder) OnICECandidate(arg0 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "OnICECandidate", reflect.TypeOf((*MockpeerConnAbstract)(nil).OnICECandidate), arg0)
}

// RemoteDescription mocks base method.
func (m *MockpeerConnAbstract) RemoteDescription() *webrtc.SessionDescription {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "RemoteDescription")
	ret0, _ := ret[0].(*webrtc.SessionDescription)
	return ret0
}

// RemoteDescription indicates an expected call of RemoteDescription.
func (mr *MockpeerConnAbstractMockRecorder) RemoteDescription() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "RemoteDescription", reflect.TypeOf((*MockpeerConnAbstract)(nil).RemoteDescription))
}

// SetLocalDescription mocks base method.
func (m *MockpeerConnAbstract) SetLocalDescription(desc webrtc.SessionDescription) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "SetLocalDescription", desc)
	ret0, _ := ret[0].(error)
	return ret0
}

// SetLocalDescription indicates an expected call of SetLocalDescription.
func (mr *MockpeerConnAbstractMockRecorder) SetLocalDescription(desc any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SetLocalDescription", reflect.TypeOf((*MockpeerConnAbstract)(nil).SetLocalDescription), desc)
}

// SetRemoteDescription mocks base method.
func (m *MockpeerConnAbstract) SetRemoteDescription(desc webrtc.SessionDescription) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "SetRemoteDescription", desc)
	ret0, _ := ret[0].(error)
	return ret0
}

// SetRemoteDescription indicates an expected call of SetRemoteDescription.
func (mr *MockpeerConnAbstractMockRecorder) SetRemoteDescription(desc any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SetRemoteDescription", reflect.TypeOf((*MockpeerConnAbstract)(nil).SetRemoteDescription), desc)
}
