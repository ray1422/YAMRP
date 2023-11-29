package server

import (
	"context"
	"io"

	log "github.com/sirupsen/logrus"

	"github.com/ray1422/yamrp/proto"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// AnswererServerImpl AnswererServerImpl
type AnswererServerImpl struct {
	proto.UnimplementedYAMRPAnswererServer
	recvOfferCh    <-chan offering
	answerCh       chan<- AnsPacket
	iceCh          chan<- IcePacket
	recvCloseIceCh <-chan string
}

// NewAnswererServer creates a new answerer server.
func NewAnswererServer(
	recvOfferCh <-chan offering,
	answerCh chan<- AnsPacket,
	iceCh chan<- IcePacket,
	recvCloseIceCh <-chan string,
) *AnswererServerImpl {
	return &AnswererServerImpl{
		recvOfferCh:    recvOfferCh,
		answerCh:       answerCh,
		iceCh:          iceCh,
		recvCloseIceCh: recvCloseIceCh,
	}
}

// WaitForOffer waits for the offer from the offerer.
func (a *AnswererServerImpl) WaitForOffer(context.Context, *proto.WaitForOfferRequest) (*proto.OfferResponse, error) {
	// take one from the channel
	offering := <-a.recvOfferCh
	return &proto.OfferResponse{
		OffererId:  offering.offererID,
		AnswererId: offering.answererID,
		Body:       offering.offer,
	}, nil

}

// SendAnswer sends a answer to the offerer.
func (a *AnswererServerImpl) SendAnswer(ctx context.Context, req *proto.ReplyToAnswererRequest) (*proto.AnswerResponse, error) {
	pkt := AnsPacket{
		answererID: req.AnswererId,
		answer:     req.Body,
	}
	// FIXME validate the answer
	select {
	case a.answerCh <- pkt:
		return &proto.AnswerResponse{}, nil
	case <-ctx.Done():
		// deadline exceeded
		return nil, status.Errorf(codes.DeadlineExceeded, "timeout")
	}
}

// SendIceCandidate sends a ice candidate to the offerer.
func (a *AnswererServerImpl) SendIceCandidate(srv proto.YAMRPAnswerer_SendIceCandidateServer) error {
	for {
		select {
		case <-srv.Context().Done():
			log.Warnf("ice stream timeout")
			return status.Errorf(codes.DeadlineExceeded, "timeout")
		case <-func() chan struct{} {
			sig := make(chan struct{})
			go func() {
				defer func() {
					sig <- struct{}{}
				}()

				req, err := srv.Recv()
				if req != nil {
					// FIXME validate the ice candidate
					a.iceCh <- IcePacket{
						answererID:   req.AnswererId,
						iceCandidate: req.Body,
					}
				}
				if err != nil && err != io.EOF {
					log.Warnf("error when receiving ice candidate."+
						"this should be an exception since ice stream should be closed by the offerer. error: %v",
						err)
					return
				}
			}()
			return sig
		}():
			// FIXME listen to the close ice channel

		}
		// TODO
	}
}

// Serve serves the server and blocks the current goroutine.
func (a *AnswererServerImpl) Serve() error {
	return nil
}
