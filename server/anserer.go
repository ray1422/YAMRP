package server

import (
	"context"
	"io"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/ray1422/yamrp/proto"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// AnswererServerImpl AnswererServerImpl
type AnswererServerImpl struct {
	proto.UnimplementedYAMRPAnswererServer
	recvOfferCh         <-chan offering
	answerCh            chan<- AnsPacket
	iceToOfferCh        chan<- IcePacket
	iceToAnsCh          <-chan IcePacket
	recvCloseIceToAnsCh <-chan string

	// FIXME make ice candidate forwarding a delicate service
	// AnsID2iceToAnsChan chan is for waitForIceCandidate
	// chan itself is thread safe, but the map is not. should lock before get or set.
	sync.RWMutex
	AnsID2iceToAnsChan map[string]chan string
}

// NewAnswererServer creates a new answerer server.
func NewAnswererServer(
	recvOfferCh <-chan offering,
	answerCh chan<- AnsPacket,
	iceToAnswerCh <-chan IcePacket,
	iceToOfferCh chan<- IcePacket,
	recvCloseIceToAnsCh <-chan string,
) *AnswererServerImpl {
	return &AnswererServerImpl{
		recvOfferCh:         recvOfferCh,
		answerCh:            answerCh,
		iceToOfferCh:        iceToOfferCh,
		iceToAnsCh:          iceToAnswerCh,
		recvCloseIceToAnsCh: recvCloseIceToAnsCh,
		AnsID2iceToAnsChan:  make(map[string]chan string),
	}
}

// WaitForOffer waits for the offer from the offerer.
func (a *AnswererServerImpl) WaitForOffer(context.Context, *proto.WaitForOfferRequest) (*proto.OfferResponse, error) {
	// take one from the channel
	offering := <-a.recvOfferCh

	// FIXME refactor
	a.Lock()
	a.AnsID2iceToAnsChan[offering.answererID] = make(chan string, 1024)
	a.Unlock()

	go func() {
		time.Sleep(10 * time.Second)
		a.Lock()
		delete(a.AnsID2iceToAnsChan, offering.answererID)
		a.Unlock()
	}()

	return &proto.OfferResponse{
		OffererId:  offering.offererID,
		AnswererId: offering.answererID,
		Body:       offering.offer,
	}, nil

}

// SendAnswer sends a answer to the offerer.
func (a *AnswererServerImpl) SendAnswer(ctx context.Context, req *proto.ReplyToRequest) (*proto.AnswerResponse, error) {
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
					a.iceToOfferCh <- IcePacket{
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

// WaitForICECandidate waits for the ice candidate from the offerer.
func (a *AnswererServerImpl) WaitForICECandidate(req *proto.WaitForICECandidateRequest,
	srv proto.YAMRPAnswerer_WaitForICECandidateServer) error {
	// FIXME auth
	a.RLock()
	ch, ok := a.AnsID2iceToAnsChan[req.AnswererId]
	a.RUnlock()
	if !ok {
		log.Warnf("ice chan not found for answerID %s", req.AnswererId)
		return status.Errorf(codes.NotFound, "answerer not found")
	}
	defer func() {
		a.Lock()
		delete(a.AnsID2iceToAnsChan, req.AnswererId)
		a.Unlock()
	}()

	ctx, cancel := context.WithTimeout(context.Background(), waitingForIceCandidateAndAnswerTimeout)
	defer cancel()
	for {
		select {
		case <-ctx.Done():
			log.Debugf("timeout, stop forwarding ice candidate for answerer %s", req.AnswererId)
			return nil
		case ice := <-ch:
			log.Debugf("forwarding ice candidate to answerer %s", req.AnswererId)
			err := srv.Send(&proto.IceCandidate{
				Candidate: ice,
			})
			if err == io.EOF {
				log.Debugf("ice stream closed for answerer %s", req.AnswererId)
				return nil
			}
			if err != nil {
				return err
			}
		}
	}
}

// Serve serves the server and blocks the current goroutine.
func (a *AnswererServerImpl) Serve() error {
	go a.iceCandidateRouter()
	return nil
}

func (a *AnswererServerImpl) iceCandidateRouter() {
	for {
		select {
		case ice := <-a.iceToAnsCh:
			a.RLock()
			ch, ok := a.AnsID2iceToAnsChan[ice.answererID]
			a.RUnlock()
			if !ok {
				log.Warnf("ice chan not found for answerID %s", ice.answererID)
				continue
			}
			ch <- ice.iceCandidate

		case <-a.recvCloseIceToAnsCh:
			// FIXME
		}
	}
}
