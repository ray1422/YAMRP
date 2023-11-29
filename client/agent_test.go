package client

import (
	"errors"
	"testing"

	"github.com/pion/webrtc/v4"
	"github.com/ray1422/yamrp/proto"
	"github.com/ray1422/yamrp/proto/mock_proto"
	"github.com/stretchr/testify/assert"

	"go.uber.org/mock/gomock"
)

type UserMock struct {
	token string
	id    string
}

func (u *UserMock) Token() string {
	return u.token
}
func (u *UserMock) ID() string {
	return u.id
}

type peerConnBuilderMock struct {
	OnICECandidate          func(func(*webrtc.ICECandidate))
	AddICECandidate         func(candidate webrtc.ICECandidateInit) error
	SetRemoteDescription    func(desc webrtc.SessionDescription) error
	SetLocalDescription     func(desc webrtc.SessionDescription) error
	CreateOffer             func(options *webrtc.OfferOptions) (webrtc.SessionDescription, error)
	CreateAnswer            func(options *webrtc.AnswerOptions) (webrtc.SessionDescription, error)
	CreateDataChannel       func(label string, dataChannelInit *webrtc.DataChannelInit) (*webrtc.DataChannel, error)
	OnConnectionStateChange func(func(webrtc.PeerConnectionState))
	OnDataChannel           func(func(*webrtc.DataChannel))
}

func (p peerConnBuilderMock) NewPeerConnection(config webrtc.Configuration) (peerConnAbstract, error) {
	ctrl := gomock.NewController(nil)
	obj := NewMockpeerConnAbstract(ctrl)

	obj.EXPECT().OnICECandidate(
		gomock.Any(),
	).DoAndReturn(p.OnICECandidate).AnyTimes()

	obj.EXPECT().AddICECandidate(
		gomock.Any(),
	).DoAndReturn(p.AddICECandidate).AnyTimes()

	obj.EXPECT().SetRemoteDescription(
		gomock.Any(),
	).DoAndReturn(p.SetRemoteDescription).AnyTimes()

	obj.EXPECT().SetLocalDescription(
		gomock.Any(),
	).DoAndReturn(p.SetLocalDescription).AnyTimes()

	obj.EXPECT().CreateOffer(
		gomock.Any(),
	).DoAndReturn(p.CreateOffer).AnyTimes()

	obj.EXPECT().CreateAnswer(
		gomock.Any(),
	).DoAndReturn(p.CreateAnswer).AnyTimes()
	obj.EXPECT().CreateDataChannel(
		gomock.Any(),
		gomock.Any(),
	).DoAndReturn(p.CreateDataChannel).AnyTimes()

	obj.EXPECT().OnConnectionStateChange(
		gomock.Any(),
	).DoAndReturn(p.OnConnectionStateChange).AnyTimes()

	obj.EXPECT().OnDataChannel(
		gomock.Any(),
	).DoAndReturn(p.OnDataChannel).AnyTimes()

	return obj, nil
}

var fixtureAgentAllSuccessPeerConnBuilderMock = peerConnBuilderMock{
	OnICECandidate: func(func(*webrtc.ICECandidate)) {},
	SetRemoteDescription: func(desc webrtc.SessionDescription) error {
		return nil
	},
	SetLocalDescription: func(desc webrtc.SessionDescription) error {
		return nil
	},
	CreateAnswer: func(options *webrtc.AnswerOptions) (webrtc.SessionDescription, error) {
		return webrtc.SessionDescription{}, nil
	},
	CreateDataChannel: func(label string, dataChannelInit *webrtc.DataChannelInit) (*webrtc.DataChannel, error) {
		return nil, nil
	},
	OnDataChannel: func(func(*webrtc.DataChannel)) {},
}

func TestNewAgentConnect(t *testing.T) {
	errorsIsNil := []bool{true, false, false, false}
	peerConnBuilderMockTestCases := []peerConnBuilderMock{
		// case 1: all success
		fixtureAgentAllSuccessPeerConnBuilderMock,
		// case 2: fail on SetRemoteDescription
		{
			OnICECandidate: func(func(*webrtc.ICECandidate)) {},
			SetRemoteDescription: func(desc webrtc.SessionDescription) error {
				t.Log("test failing on SetRemoteDescription")
				return errors.New("fail")
			},
			SetLocalDescription: func(desc webrtc.SessionDescription) error {
				return nil
			},
			CreateAnswer: func(options *webrtc.AnswerOptions) (webrtc.SessionDescription, error) {
				return webrtc.SessionDescription{}, nil
			},
			OnDataChannel: func(func(*webrtc.DataChannel)) {},
		},
		// case 3: fail on SetLocalDescription
		{
			OnICECandidate: func(func(*webrtc.ICECandidate)) {},
			SetRemoteDescription: func(desc webrtc.SessionDescription) error {
				return nil
			},
			SetLocalDescription: func(desc webrtc.SessionDescription) error {
				t.Log("test fail on SetLocalDescription")
				return errors.New("fail")
			},
			CreateAnswer: func(options *webrtc.AnswerOptions) (webrtc.SessionDescription, error) {
				return webrtc.SessionDescription{}, nil
			},
			OnDataChannel: func(func(*webrtc.DataChannel)) {},
		},
		// case 4: fail on CreateAnswer
		{
			OnICECandidate: func(func(*webrtc.ICECandidate)) {},
			SetRemoteDescription: func(desc webrtc.SessionDescription) error {
				return nil
			},
			SetLocalDescription: func(desc webrtc.SessionDescription) error {
				return nil
			},
			CreateAnswer: func(options *webrtc.AnswerOptions) (webrtc.SessionDescription, error) {
				t.Log("test fail on CreateAnswer")
				return webrtc.SessionDescription{}, errors.New("fail")
			},
			OnDataChannel: func(func(*webrtc.DataChannel)) {},
		},
	}
	for caseIdx, webRTCCase := range peerConnBuilderMockTestCases {
		ctrl := gomock.NewController(t)
		ansAPI := mock_proto.NewMockYAMRPAnswererClient(ctrl)
		streamRet := mock_proto.NewMockYAMRPAnswerer_SendIceCandidateClient(ctrl)
		streamRet.EXPECT().Send(
			gomock.Any(),
		).Return(nil).AnyTimes()

		ansAPI.EXPECT().WaitForOffer(
			gomock.Any(),
			gomock.Any(),
		).Return(&proto.OfferResponse{
			OffererId:  "My Offerer ID",
			AnswererId: "test_answer",
			Body:       "{}",
		}, nil).AnyTimes()

		ansAPI.EXPECT().SendAnswer(
			gomock.Any(),
			gomock.Any(),
		).Return(&proto.AnswerResponse{
			Answer: "{}",
		}, nil).AnyTimes()

		ansAPI.EXPECT().SendIceCandidate(
			gomock.Any(),
		).Return(streamRet, nil).AnyTimes()

		agent, err := NewAgent(
			"localhost:60000",
			"tcp",
			&webRTCCase,
			&UserMock{"my_token_a", "my_id_a"},
			ansAPI,
		)
		if !assert.Nil(t, err) {
			t.Fatalf("err is not nil: %v", err)
		}

		if !assert.NotNil(t, agent) {
			t.Fatalf("agent is nil")
		}

		// test default config
		errCh := agent.Connect(nil)
		err = <-errCh
		t.Logf("case %d: err: %v", caseIdx, err)
		if errorsIsNil[caseIdx] {
			assert.Nil(t, err)
		} else {
			assert.NotNil(t, err)
		}
	}
}
