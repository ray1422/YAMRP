package client

import (
	"context"
	"fmt"
	"io"
	"testing"

	"github.com/pion/webrtc/v4"
	"github.com/ray1422/yamrp/proto"
	"github.com/ray1422/yamrp/proto/mock_proto"
	"github.com/stretchr/testify/assert"
	gomock "go.uber.org/mock/gomock"
)

const testHostID = "test_host_id"

var connReadySignal = make(chan struct{}, 1024)
var fixtureListenerPeerConnBuilderMock = peerConnBuilderMock{
	OnICECandidate: func(func(*webrtc.ICECandidate)) { panic(("should not be called")) },
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

	// TODO write test cases
	pcCases := []peerConnBuilderMock{
		// case 1: all success
		fixtureListenerPeerConnBuilderMock,

		// case 2: SetRemoteDescription failed
		{
			OnICECandidate: func(func(*webrtc.ICECandidate)) { panic(("should not be called")) },
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
			OnICECandidate: func(func(*webrtc.ICECandidate)) { panic(("should not be called")) },
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
}
