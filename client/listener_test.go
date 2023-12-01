package client

import (
	"context"
	"fmt"
	"io"
	"testing"
	"time"

	"github.com/pion/webrtc/v4"
	"github.com/ray1422/yamrp/proto"
	"github.com/ray1422/yamrp/proto/mock_proto"
	"github.com/ray1422/yamrp/utils/mock"
	"github.com/stretchr/testify/assert"
	gomock "go.uber.org/mock/gomock"
)

func onIceCandidateMock(f func(*webrtc.ICECandidate)) {
	ticker := time.NewTicker(100 * time.Millisecond)
	go func() {
		// a fake candidate
		candidate := &webrtc.ICECandidate{
			Address:        "127.0.0.1",
			Port:           3333,
			RelatedAddress: "127.0.0.1",
			Protocol:       webrtc.ICEProtocolUDP,
			RelatedPort:    3333,
			Typ:            webrtc.ICECandidateTypeRelay,
		}

		for cnt := 0; cnt < 10; cnt++ {
			<-ticker.C
			f(candidate)
		}
	}()
}

const testHostID = "test_host_id"

var connReadySignal = make(chan struct{}, 1024)
var fixtureListenerPeerConnBuilderMock = peerConnBuilderMock{
	OnICECandidate: onIceCandidateMock,
	AddICECandidate: func(candidate webrtc.ICECandidateInit) error {
		// should be called
		return nil
	},
	SetRemoteDescription: func(desc webrtc.SessionDescription) error { return nil },
	SetLocalDescription:  func(desc webrtc.SessionDescription) error { return nil },
	CreateOffer: func(options *webrtc.OfferOptions) (webrtc.SessionDescription, error) {
		return webrtc.SessionDescription{}, nil
	},
	CreateAnswer: func(options *webrtc.AnswerOptions) (webrtc.SessionDescription, error) {
		panic(("not implemented"))
	},
	CreateDataChannel: func(label string, dataChannelInit *webrtc.DataChannelInit) (*webrtc.DataChannel, error) {
		return &webrtc.DataChannel{}, nil
	},
	OnConnectionStateChange: func(f func(webrtc.PeerConnectionState)) {
		// should be called
		// TODO impl should be call
		fmt.Println("callback set")
		go func() {
			<-connReadySignal
			f(webrtc.PeerConnectionStateConnected)

		}()
	},
}

func TestNewListenerConnect(t *testing.T) {
	ctrl := gomock.NewController(t)
	offAPI := mock_proto.NewMockYAMRPOffererClient(ctrl)
	offAPI.EXPECT().SendOffer(gomock.Any(), gomock.Any()).Do(func(ctx context.Context, req *proto.SendOfferRequest, opts ...interface{}) {
		assert.Equal(t, testHostID, req.HostId)
	}).Return(&proto.OfferResponse{
		OffererId:  "My Offerer ID",
		AnswererId: "test_answer",
	}, io.EOF).AnyTimes()

	offAPI.EXPECT().WaitForAnswer(gomock.Any(), gomock.Any()).
		Return(&proto.AnswerResponse{
			Answer: "test_answer",
		}, nil).
		AnyTimes()

	offAPI.EXPECT().WaitForICECandidate(gomock.Any(), gomock.Any()).
		DoAndReturn(
			func(ctx context.Context,
				req *proto.WaitForICECandidateRequest,
				opts ...interface{}) (proto.YAMRPOfferer_WaitForICECandidateClient, error) {
				// test for sending 5 times ice candidates
				cnt := 0
				obj := mock_proto.NewMockYAMRPOfferer_WaitForICECandidateClient(ctrl)
				obj.EXPECT().Context().Return(context.Background()).AnyTimes()
				obj.EXPECT().Recv().DoAndReturn(func() (*proto.IceCandidate, error) {
					// fmt.Printf("recv called %d times\n", cnt)
					if cnt == 6 {
						return nil, io.EOF
					} else if cnt == 5 {
						connReadySignal <- struct{}{}
						// trigger callback function
					}
					cnt++
					return &proto.IceCandidate{
						Candidate: "test_candidate",
					}, nil
				}).AnyTimes()
				return obj, nil
			}).AnyTimes()
	offAPI.EXPECT().SendIceCandidate(gomock.Any(), gomock.Any()).
		DoAndReturn(
			func(ctx context.Context,
				opts ...interface{}) (proto.YAMRPOfferer_SendIceCandidateClient, error) {
				// test for sending 5 times ice candidates
				cnt := 0
				obj := mock_proto.NewMockYAMRPOfferer_SendIceCandidateClient(ctrl)
				obj.EXPECT().Context().Return(context.Background()).AnyTimes()
				obj.EXPECT().Send(gomock.Any()).DoAndReturn(func(req *proto.ReplyToRequest) error {
					// fmt.Printf("send called %d times\n", cnt)
					if cnt == 6 {
						return io.EOF
					}
					cnt++
					return nil
				}).AnyTimes()
				return obj, nil
			}).AnyTimes()

	// TODO write test cases
	pcCases := []peerConnBuilderMock{
		// case 1: all success
		fixtureListenerPeerConnBuilderMock,

		// case 2: SetRemoteDescription failed
		{
			OnICECandidate: onIceCandidateMock,
			AddICECandidate: func(candidate webrtc.ICECandidateInit) error {
				// should be called
				return nil
			},
			SetRemoteDescription: func(desc webrtc.SessionDescription) error { return fmt.Errorf("failed") },
			SetLocalDescription:  func(desc webrtc.SessionDescription) error { return nil },
			CreateOffer: func(options *webrtc.OfferOptions) (webrtc.SessionDescription, error) {
				return webrtc.SessionDescription{}, nil
			},
			CreateAnswer: func(options *webrtc.AnswerOptions) (webrtc.SessionDescription, error) {
				panic(("not implemented"))
			},
			CreateDataChannel: fixtureListenerPeerConnBuilderMock.CreateDataChannel,
			OnConnectionStateChange: func(f func(webrtc.PeerConnectionState)) {
				// should be called
				// TODO impl should be call
				fmt.Println("callback set")
				go func() {
					<-connReadySignal
					fmt.Println("resolved callback")
					f(webrtc.PeerConnectionStateConnected)

				}()
			},
		},
		// case 3: SetLocalDescription failed
		{
			OnICECandidate: onIceCandidateMock,
			AddICECandidate: func(candidate webrtc.ICECandidateInit) error {
				// should be called
				return nil
			},
			SetRemoteDescription: func(desc webrtc.SessionDescription) error { return nil },
			SetLocalDescription:  func(desc webrtc.SessionDescription) error { return fmt.Errorf("failed") },
			CreateOffer: func(options *webrtc.OfferOptions) (webrtc.SessionDescription, error) {
				return webrtc.SessionDescription{}, nil
			},
			CreateAnswer: func(options *webrtc.AnswerOptions) (webrtc.SessionDescription, error) {
				panic(("not implemented"))
			},
			CreateDataChannel: fixtureListenerPeerConnBuilderMock.CreateDataChannel,
			OnConnectionStateChange: func(f func(webrtc.PeerConnectionState)) {
				// doesn't matter in this case
			},
		},
	}
	shouldFail := []bool{false, true, true}
	for i, pcBuilder := range pcCases {
		fmt.Printf("test case %d\n", i)
		listener, err := NewListener("localhost:8080", "tcp", pcBuilder, offAPI, nil, testHostID)
		assert.Nil(t, err)
		assert.NotNil(t, listener)
		result := listener.Connect(nil)

		if shouldFail[i] {
			assert.NotNil(t, <-result)
		} else {
			assert.Nil(t, <-result)
		}
		// assert.Nil(t, <-result)
	}
	// nop
}

func ProxyLifecyclePeerConnForFixture(t *testing.T, ctrl *gomock.Controller) *MockpeerConnAbstract {
	peerConnMock := NewMockpeerConnAbstract(ctrl)
	peerConnMock.EXPECT().OnICECandidate(gomock.Any()).DoAndReturn(func(f func(*webrtc.ICECandidate)) {
		onIceCandidateMock(f)
	}).AnyTimes()
	peerConnMock.EXPECT().AddICECandidate(gomock.Any()).DoAndReturn(func(candidate webrtc.ICECandidateInit) error {
		// should be called
		return nil
	}).AnyTimes()
	peerConnMock.EXPECT().SetRemoteDescription(gomock.Any()).DoAndReturn(func(desc webrtc.SessionDescription) error {
		return nil
	}).AnyTimes()
	peerConnMock.EXPECT().SetLocalDescription(gomock.Any()).DoAndReturn(func(desc webrtc.SessionDescription) error {
		return nil
	}).AnyTimes()
	peerConnMock.EXPECT().CreateOffer(gomock.Any()).DoAndReturn(func(options *webrtc.OfferOptions) (webrtc.SessionDescription, error) {
		return webrtc.SessionDescription{}, nil
	}).AnyTimes()
	peerConnMock.EXPECT().CreateAnswer(gomock.Any()).DoAndReturn(func(options *webrtc.AnswerOptions) (webrtc.SessionDescription, error) {
		panic(("not implemented"))
	}).AnyTimes()
	peerConnMock.EXPECT().CreateDataChannel(gomock.Any(), gomock.Any()).DoAndReturn(func(label string, dataChannelInit *webrtc.DataChannelInit) (*webrtc.DataChannel, error) {
		return &webrtc.DataChannel{}, nil
	}).Times(1)
	peerConnMock.EXPECT().OnConnectionStateChange(gomock.Any()).DoAndReturn(func(f func(webrtc.PeerConnectionState)) {
		// should be called
		// TODO impl should be call
		fmt.Println("callback set")
		go func() {
			<-connReadySignal
			fmt.Println("resolved callback")
			f(webrtc.PeerConnectionStateConnected)

		}()
	}).AnyTimes()
	return peerConnMock
}
func TestStartProxyLifecycle(t *testing.T) {
	// given
	ctrl := gomock.NewController(t)
	peerConnMock := ProxyLifecyclePeerConnForFixture(t, ctrl)
	lis := mock.NewMockListener()
	listener := &ListenerNetConn{
		listener: lis,
		hostID:   "test_host_id",
		peerConn: peerConnMock,
	}
	err := listener.Bind()
	assert.NoError(t, err)

	// when
	lis.Dial("pipe", "test_addr")

}
