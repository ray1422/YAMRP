package client

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/ray1422/yamrp/proto"
	"github.com/ray1422/yamrp/proto/mock_proto"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

var testSrvAddr = "localhost:3000"
var testBindAddr = "localhost:60000"

func startTestingServer(addr string) error {
	// simple http server
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("hello world"))
		fmt.Println("request")
	})
	return http.ListenAndServe(addr, nil)

}

func createAgent(ansAPI proto.YAMRPAnswererClient) *AgentNetConn {
	agent, err := NewAgent(
		testSrvAddr,
		"tcp",
		// &fixtureAgentAllSuccessPeerConnBuilderMock,
		// use real peerConnBuilder
		&peerConnBuilderImpl{},
		&UserMock{"my_token_a", "my_id_a"},
		ansAPI,
	)
	if err != nil {
		panic(err)
	}
	return agent
}

func createListener(offAPI proto.YAMRPOffererClient) *ListenerNetConn {
	listener, err := NewListener(
		testBindAddr,
		"tcp",
		&peerConnBuilderImpl{},
		offAPI,
		nil,
		testHostID,
	)
	if err != nil {
		panic(err)
	}
	return listener
}

func CreateOfferAndAnswerAPI(t *testing.T) (proto.YAMRPOffererClient, proto.YAMRPAnswererClient) {

	ctrlOff := gomock.NewController(t)
	ctrlAns := gomock.NewController(t)

	offAPI := mock_proto.NewMockYAMRPOffererClient(ctrlOff)
	ansAPI := mock_proto.NewMockYAMRPAnswererClient(ctrlAns)

	offerChan := make(chan string, 1)
	answerChan := make(chan string, 1)
	iceCandChan := make(chan string, 1024)
	iceCandAtoOChan := make(chan string, 1024)
	// offerAPI
	offAPI.EXPECT().SendOffer(gomock.Any(), gomock.Any()).
		Do(func(ctx context.Context, req *proto.SendOfferRequest, opts ...interface{}) {
			assert.Equal(t, testHostID, req.HostId)
			go func() { offerChan <- req.Offer }()
		}).Return(&proto.OfferResponse{
		OffererId:  "My Offerer ID",
		AnswererId: "test_answer",
	}, nil).AnyTimes()

	offAPI.EXPECT().WaitForAnswer(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context,
			req *proto.WaitForAnswerRequest,
			opts ...interface{}) (*proto.AnswerResponse, error) {
			return &proto.AnswerResponse{
				Answer: <-answerChan,
			}, nil
		}).AnyTimes()

	offAPI.EXPECT().WaitForICECandidate(gomock.Any(), gomock.Any()).
		DoAndReturn(
			func(ctx context.Context,
				req *proto.WaitForICECandidateRequest,
				opts ...interface{}) (proto.YAMRPOfferer_WaitForICECandidateClient, error) {
				obj := mock_proto.NewMockYAMRPOfferer_WaitForICECandidateClient(ctrlOff)
				closed := false
				obj.EXPECT().CloseSend().Do(func() {
					closed = true
				}).AnyTimes()

				obj.EXPECT().Recv().DoAndReturn(func() (*proto.IceCandidate, error) {
					if closed {
						return nil, io.EOF
					}
					v, ok := <-iceCandChan
					if !ok {
						return nil, io.EOF
					}
					return &proto.IceCandidate{
						Candidate: v,
					}, nil

				}).AnyTimes()
				return obj, nil
			}).AnyTimes()

	offAPI.EXPECT().SendIceCandidate(gomock.Any(), gomock.Any()).
		DoAndReturn(
			func(ctx context.Context,
				opts ...interface{}) (proto.YAMRPOfferer_SendIceCandidateClient, error) {
				obj := mock_proto.NewMockYAMRPOfferer_SendIceCandidateClient(ctrlOff)
				closed := false
				obj.EXPECT().CloseSend().Do(func() {
					// close(iceCandChan)
					closed = true
				}).AnyTimes()

				obj.EXPECT().Send(gomock.Any()).Do(func(v *proto.ReplyToRequest) {
					if closed {
						return
					}
					go func() { iceCandAtoOChan <- v.Body }()
				}).Return(nil).AnyTimes()
				return obj, nil
			}).AnyTimes()

	// answerAPI
	ansAPI.EXPECT().WaitForOffer(
		gomock.Any(),
		gomock.Any(),
	).DoAndReturn(func(
		ctx context.Context,
		req *proto.WaitForOfferRequest,
		opts ...interface{}) (*proto.OfferResponse, error) {
		return &proto.OfferResponse{
			OffererId:  "My Offerer ID",
			AnswererId: "test_answer",
			Body:       <-offerChan,
		}, nil
	}).AnyTimes()

	ansAPI.EXPECT().SendAnswer(
		gomock.Any(),
		gomock.Any(),
	).Do(func(ctx context.Context, req *proto.ReplyToRequest, opts ...interface{}) {
		go func() { answerChan <- req.Body }()
	}).Return(&proto.AnswerResponse{}, nil).AnyTimes()
	streamRet := mock_proto.NewMockYAMRPAnswerer_SendIceCandidateClient(ctrlAns)
	streamRet.EXPECT().Send(
		gomock.Any(),
	).Do(func(v *proto.ReplyToRequest) {
		go func() { iceCandChan <- v.Body }()
	}).Return(nil).AnyTimes()
	ansAPI.EXPECT().SendIceCandidate(
		gomock.Any(),
	).Return(streamRet, nil).AnyTimes()

	ansAPI.EXPECT().WaitForICECandidate(
		gomock.Any(),
		gomock.Any(),
	).DoAndReturn(func(
		ctx context.Context,
		req *proto.WaitForICECandidateRequest,
		opts ...interface{}) (proto.YAMRPAnswerer_WaitForICECandidateClient, error) {
		obj := mock_proto.NewMockYAMRPAnswerer_WaitForICECandidateClient(ctrlAns)
		obj.EXPECT().Context().Return(ctx).AnyTimes()
		closed := false
		obj.EXPECT().Recv().DoAndReturn(func() (*proto.IceCandidate, error) {
			if closed {
				return nil, io.EOF
			}
			v, ok := <-iceCandAtoOChan
			if !ok {
				return nil, io.EOF
			}
			return &proto.IceCandidate{
				Candidate: v,
			}, nil
		}).AnyTimes()
		return obj, nil
	}).AnyTimes()

	return offAPI, ansAPI
}

func TestAgentListener(t *testing.T) {
	offAPI, ansAPI := CreateOfferAndAnswerAPI(t)
	agent := createAgent(ansAPI)
	listener := createListener(offAPI)
	// connect

	sig := make(chan struct{}, 2)
	// wait for connection
	go func() {
		agentCh := agent.Connect(nil)
		// wait for connection
		errAgent := <-agentCh
		assert.Nil(t, errAgent)
		sig <- struct{}{}
		t.Log("agent connected")
	}()
	go func() {
		listenerCh := listener.Connect(nil)
		errListener := <-listenerCh
		assert.Nil(t, errListener)

		t.Log("listener connected")

		// bind on localhost:60000
		err := listener.Bind()
		assert.Nil(t, err)
		// start testing server on testSrvAddr
		go func() {
			err := startTestingServer(testSrvAddr)
			assert.Nil(t, err)
		}()
		// wait 100ms
		time.Sleep(100 * time.Millisecond)
		for i := 0; i < 5; i++ {
			r, err := http.Get("http://" + testBindAddr)
			assert.Nil(t, err)
			// assert body
			buf := make([]byte, 1024)
			n, err := r.Body.Read(buf)
			if err != nil {
				assert.Equal(t, io.EOF, err)
			}
			assert.Equal(t, "hello world", string(buf[:n]))
			fmt.Println("response")
			assert.Equal(t, 200, r.StatusCode)

		}

		sig <- struct{}{}

	}()
	<-sig
	<-sig
	println("agent and listener connected")
	time.Sleep(100 * time.Millisecond)
}
