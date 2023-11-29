package server

import (
	"context"
	"io"
	"testing"
	"time"

	"github.com/ray1422/yamrp/proto"
	"github.com/ray1422/yamrp/utils/mock"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func answerServerFixture(t *testing.T) (
	offerCh chan offering,
	answerCh chan AnsPacket,
	iceCh chan IcePacket,
	recvCloseIceCh chan string,
	lis *mock.Listener,
	ansServer *AnswererServerImpl,
	client proto.YAMRPAnswererClient,
) {
	offerCh = make(chan offering)
	answerCh = make(chan AnsPacket, 1024)
	iceCh = make(chan IcePacket, 1024)
	recvCloseIceCh = make(chan string)
	lis = mock.NewMockListener()

	ansServer = NewAnswererServer(offerCh, answerCh, iceCh, recvCloseIceCh)
	go ansServer.Serve()
	s := grpc.NewServer()
	proto.RegisterYAMRPAnswererServer(s, ansServer)
	go s.Serve(lis)
	conn, err := grpc.Dial("pipe", grpc.WithContextDialer(lis.DialContext), grpc.WithTransportCredentials(insecure.NewCredentials()))
	assert.NoError(t, err)
	client = proto.NewYAMRPAnswererClient(conn)
	ansServer = NewAnswererServer(
		offerCh,
		answerCh,
		iceCh,
		recvCloseIceCh,
	)
	return
}

func TestAnswerServer(t *testing.T) {
	// given
	offerCh, AnswerCh, iceCh, recvCloseIceCh, lis, ansServer, client := answerServerFixture(t)
	_, _, _, _, _, _, _ = offerCh, AnswerCh, iceCh, recvCloseIceCh, lis, ansServer, client
	// when
	// send a offer
	res := &proto.OfferResponse{}
	sig := make(chan *struct{}, 1)
	go func() {
		var err error
		res, err = client.WaitForOffer(context.Background(), &proto.WaitForOfferRequest{
			HostId: "host_id",
		})
		if err != nil {
			panic(err)
		}
		sig <- &struct{}{}
	}()

	offerCh <- offering{
		offererID:  "offerer_id",
		answererID: "answerer_id",
		offer:      `{"offer": "valid JSON"}`,
	}

	// then
	<-sig
	assert.Equal(t, "offerer_id", res.OffererId)

	// when
	// sending answer
	sig = make(chan *struct{}, 1)
	go func() {
		_, err := client.SendAnswer(context.Background(), &proto.ReplyToAnswererRequest{
			AnswererId: "answerer_id",
			Body:       `{"answer": "valid JSON"}`,
		})
		if err != nil {
			panic(err)
		}
		sig <- &struct{}{}
	}()

	// then
	assert.NotEmpty(t, <-AnswerCh)
	<-sig

	// when
	// sending ice candidate

	sig = make(chan *struct{}, 1)
	go func() {
		defer func() {
			sig <- &struct{}{}
		}()
		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		defer cancel()
		cli, err := client.SendIceCandidate(ctx)
		assert.NoError(t, err)
		for {
			err := cli.Send(&proto.ReplyToAnswererRequest{
				AnswererId: "answerer_id",
				Body:       `{"ice": "valid JSON"}`,
			})
			if err == io.EOF {
				break
			} else if err != nil {
				t.Log("error when sending ice candidate", err)
				break
			}
		}

	}()

	// sending two ice candidates
	iceCh <- IcePacket{
		answererID:   "answerer_id",
		iceCandidate: `{"ice": "valid JSON 1"}`,
	}
	iceCh <- IcePacket{
		answererID:   "answerer_id",
		iceCandidate: `{"ice": "valid JSON 2"}`,
	}

	// TODO
	// recvCloseIceCh <- "answerer_id"

	sig <- &struct{}{}

}
