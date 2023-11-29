package server

import (
	"context"
	"testing"
	"time"

	"github.com/ray1422/yamrp/proto"
	"github.com/ray1422/yamrp/utils/mock"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func offererServerFixture(t *testing.T) (
	notifyNewOfferCh chan string,
	offerCh chan offering,
	answerCh chan AnsPacket,
	iceCh chan IcePacket,
	closeIceCh chan string,
	lis *mock.Listener,
	offServer *OffererServer,
	client proto.YAMRPOffererClient,
) {
	// given
	notifyNewOfferCh = make(chan string)
	offerCh = make(chan offering)
	answerCh = make(chan AnsPacket, 1024)
	iceCh = make(chan IcePacket, 1024)
	closeIceCh = make(chan string)
	lis = mock.NewMockListener()
	offServer = NewOffererServer(notifyNewOfferCh, offerCh, answerCh, iceCh, closeIceCh)
	s := grpc.NewServer()
	proto.RegisterYAMRPOffererServer(s, offServer)
	go offServer.Serve()
	go s.Serve(lis)

	// wait 100ms
	time.Sleep(100 * time.Millisecond)

	conn, err := grpc.Dial("pipe", grpc.WithContextDialer(lis.DialContext), grpc.WithTransportCredentials(insecure.NewCredentials()))
	assert.NoError(t, err)
	client = proto.NewYAMRPOffererClient(conn)

	// return all variables
	return notifyNewOfferCh, offerCh, answerCh, iceCh, closeIceCh, lis, offServer, client
}

func TestOffererWhenOffer(t *testing.T) {

	// test that when receive a offer, it should send a answer to the offer
	// channel, as well as notify the host server that a new answer is created.
	// and after the connection is closed, it should notify the offerer server
	// that ice channel is closed.

	// given
	waitingForIceCandidateAndAnswerTimeout = 1 * time.Second
	notifyNewOfferCh, offerCh, recvAnswerCh, iceCh, closeIceCh, lis, offServer, client := offererServerFixture(t)
	_, _, _, _, _, _, _, _ = notifyNewOfferCh, offerCh, recvAnswerCh, iceCh, closeIceCh, lis, offServer, client
	// when
	// send a offer
	res := &proto.OfferResponse{}
	sig := make(chan *struct{}, 1)
	go func() {
		res, _ = client.SendOffer(context.TODO(), &proto.SendOfferRequest{
			HostId: "test_host",
			Token: &proto.AuthToken{
				// FIXME: use real token
				Token: "FIXME",
			},
		})
		sig <- nil
	}()

	// then
	recvOffer := <-offerCh
	newOfferID := <-notifyNewOfferCh
	<-sig
	time.Sleep(100 * time.Millisecond)
	t.Log("waiting for offer")
	assert.Equal(t, "test_host", newOfferID)
	assert.Equal(t, res.AnswererId, recvOffer.answererID)
	assert.NotEmpty(t, recvOffer.answererID)
	assert.NotEmpty(t, recvOffer.offererID)

	t.Log("offer sent, answererID: ", recvOffer.answererID)

	// when

	// send a answer
	go func() {
		time.Sleep(100 * time.Millisecond)
		recvAnswerCh <- AnsPacket{
			answererID: res.AnswererId,
			answer:     `{"valid_JSON": "true"}`,
		}
	}()

	// then
	ret, err := client.WaitForAnswer(context.TODO(), &proto.WaitForAnswerRequest{
		AnswererId: res.AnswererId,
		Token: &proto.AuthToken{
			Token: "FIXME",
		},
	})
	assert.NoError(t, err)
	assert.Equal(t, `{"valid_JSON": "true"}`, ret.Answer)

	// when

	// when ice candidate stream is closed, it should notify the answerer server that
	// the ice channel is closed.

	stream, err := client.WaitForICECandidate(context.TODO(), &proto.WaitForICECandidateRequest{
		AnswererId: res.AnswererId,
		Token: &proto.AuthToken{
			Token: "FIXME",
		},
	})

	assert.NoError(t, err)
	doneFlag := make(chan bool, 1)
	go func() {
		for {
			select {
			case iceCh <- IcePacket{
				answererID:   res.AnswererId,
				iceCandidate: `{"valid_JSON": "true"}`,
			}:
				time.Sleep(100 * time.Millisecond)
			case <-closeIceCh:
				doneFlag <- true
				return
			}
		}
	}()
	stream.Recv()

	// then
	<-doneFlag

}

func TestSendOfferButNotTakeFromChan(t *testing.T) {
	// given
	sendOfferTimeout = 1 * time.Second
	notifyNewOfferCh, offerCh, recvAnswerCh, iceCh, closeIceCh, lis, offServer, client := offererServerFixture(t)
	_, _, _, _, _, _, _ = notifyNewOfferCh, offerCh, iceCh, closeIceCh, lis, offServer, client
	_, _ = recvAnswerCh, iceCh
	// when
	// send a offer
	// res := &proto.OfferResponse{}
	sig := make(chan *struct{}, 1)
	go func() {
		_, err := client.SendOffer(context.TODO(), &proto.SendOfferRequest{
			HostId: "test_host",
			Token: &proto.AuthToken{
				Token: "FIXME",
			},
		})
		// don't take from the offer channel
		// should get timeout error
		assert.Error(t, err)
		t.Log(err)
		sig <- nil
	}()
	<-sig
}

func TestSendOfferButNotAttachIce(t *testing.T) {
	// given
	sendOfferTimeout = 100 * time.Second
	waitingForIceCandidateAndAnswerTimeout = 100 * time.Millisecond
	notifyNewOfferCh, offerCh, recvAnswerCh, iceCh, closeIceCh, lis, offServer, client := offererServerFixture(t)
	_, _, _, _, _, _, _ = notifyNewOfferCh, offerCh, iceCh, closeIceCh, lis, offServer, client
	_, _ = recvAnswerCh, iceCh
	// when
	// send a offer
	// res := &proto.OfferResponse{}
	sig := make(chan *struct{}, 1)
	go func() {
		_, err := client.SendOffer(context.TODO(), &proto.SendOfferRequest{
			HostId: "test_host",
			Token: &proto.AuthToken{
				Token: "FIXME",
			},
		})
		assert.NoError(t, err)
		// don't attach ice candidate
		time.Sleep(1 * time.Second)
		sig <- nil
	}()
	go func() {
		offer := <-offerCh
		assert.NotEmpty(t, offer.answererID)
		sig <- nil
	}()

	go func() {
		<-notifyNewOfferCh
		sig <- nil
	}()

	<-sig
	<-sig
	<-sig
	offServer.RLock()
	assert.Empty(t, offServer.AnsID2IceChan)
	offServer.RUnlock()
}
