package server

import (
	"context"
	"io"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/google/uuid"
	"github.com/ray1422/yamrp/proto"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// offering
type offering struct {
	// offererID
	//
	hostID string
	// answerID
	answererID string
	offer      string
}

// IcePacket IcePacket
type IcePacket struct {
	answererID   string
	iceCandidate string
}

// AnsPacket AnsPacket
type AnsPacket struct {
	answererID string
	answer     string
}

// OffererServer OffererServer
type OffererServer struct {
	notifyNewOfferCh chan<- string
	offerCh          chan<- offering
	recvAnswerCh     <-chan AnsPacket
	iceToOfferCh     <-chan IcePacket
	iceToAnswerCh    chan<- IcePacket
	closeIceToAnsCh  chan<- string
	// chan itself is thread safe, but the map is not. should lock before get or set.
	//
	// the life cycle of the channel is:
	//
	// 1. created when the offer is created
	//
	// 2. deleted when the ice channel is closed or timeout
	// FIXME should be refactored someday.
	AnsID2iceToAnsChan map[string]chan string

	// chan itself is thread safe, but the map is not. should lock before get or set.
	//
	// the life cycle of the channel is:
	//
	// 1. created when the offer is created
	//
	// 2. deleted when the answer is received or timeout
	// FIXME should be refactored someday.
	AnsID2AnsChan map[string]chan string

	// FIXME use a ring as monotonic queue to manage the ice channel.
	sync.RWMutex

	proto.UnimplementedYAMRPOffererServer
}

var waitingForIceCandidateAndAnswerTimeout = time.Second * 10
var sendOfferTimeout = time.Second * 10
var _ proto.YAMRPOffererServer = (*OffererServer)(nil)

// NewOffererServer creates a new answerer server.
//
// notifyNewHostCh used to notify HostServer that a new host is created.
//
// offerCh used to notify OffererServer that a new offer is created.
//
// iceToAnsCh is receiving the ice candidate from the answerer.
//
// closeIceToAnsCh used to notify OffererServer that the ice channel is closed.
func NewOffererServer(
	notifyNewOfferCh chan<- string,
	offerCh chan<- offering,
	recvAnswerCh <-chan AnsPacket,
	iceToAnsCh chan<- IcePacket,
	iceToOfferCh <-chan IcePacket,

	// closeIceToAnsCh used to notify other that the ice channel is closed. should be write in this server.
	closeIceToAnsCh chan string,
) *OffererServer {
	return &OffererServer{
		notifyNewOfferCh:   notifyNewOfferCh,
		offerCh:            offerCh,
		recvAnswerCh:       recvAnswerCh,
		iceToAnswerCh:      iceToAnsCh,
		iceToOfferCh:       iceToOfferCh,
		closeIceToAnsCh:    closeIceToAnsCh,
		AnsID2iceToAnsChan: make(map[string]chan string),
		AnsID2AnsChan:      make(map[string]chan string),
	}
}

// SendOffer sends a offer to the answerer.
func (o *OffererServer) SendOffer(ctx2 context.Context, req *proto.SendOfferRequest) (*proto.OfferResponse, error) {
	// FIXME auth
	log.Debugf("sending offer to %s", req.HostId)
	ctx, cancel := context.WithTimeout(ctx2, sendOfferTimeout)
	defer cancel()
	// FIXME validate the offer

	// allocate a new id for answerer
	ansID := uuid.New().String()
	// allocate a offerer id for the answerer
	offID := uuid.New().String()
	// lock self and write map

	o.Lock()
	o.AnsID2iceToAnsChan[ansID] = make(chan string)
	o.AnsID2AnsChan[ansID] = make(chan string)
	o.Unlock()
	log.Debugf("created ice channel for %s", ansID)

	go func() {

		// delete a non-existing key is no-op,
		// and key is unique, so it's safe to delete the
		// key even if the key is already deleted.

		// however this implementation is ugly
		// and should be refactored someday.
		time.Sleep(waitingForIceCandidateAndAnswerTimeout)
		log.Debugf("timeout cleaning up ice channel for %s", ansID)
		log.Debugf("timeout cleaning up answer channel for %s", ansID)
		o.Lock()
		delete(o.AnsID2iceToAnsChan, ansID)
		delete(o.AnsID2AnsChan, ansID)
		o.Unlock()
	}()

	// notify the host server that a new offer is created.
	select {
	case o.notifyNewOfferCh <- req.HostId:
		log.Debugf("notified new offer for host %s", req.HostId)
	case <-ctx.Done():
		log.Warnf("timeout when notifying new offer")
		return nil, status.Errorf(codes.DeadlineExceeded, "timeout")
	}

	select {
	case o.offerCh <- offering{
		hostID:     req.GetHostId(),
		answererID: ansID,
		offer:      req.GetOffer(),
	}:
	case <-ctx.Done():
		log.Warnf("timeout when sending offer")
		return nil, status.Errorf(codes.DeadlineExceeded, "timeout")
	}

	return &proto.OfferResponse{
		OffererId:  offID,
		AnswererId: ansID,
	}, nil
}

// WaitForAnswer waits for the answer from the answerer.
func (o *OffererServer) WaitForAnswer(ctx context.Context, req *proto.WaitForAnswerRequest) (*proto.AnswerResponse, error) {
	// FIXME auth
	o.RLock()
	ch, ok := o.AnsID2AnsChan[req.AnswererId]
	o.RUnlock()
	if !ok {
		return nil, status.Errorf(codes.NotFound, "answerer not found")
	}
	select {
	case answer := <-ch:
		return &proto.AnswerResponse{
			Answer: answer,
		}, nil
	case <-ctx.Done():
		return nil, status.Errorf(codes.DeadlineExceeded, "timeout")
	}
}

// WaitForICECandidate waits for the ice candidate from the answerer.
func (o *OffererServer) WaitForICECandidate(req *proto.WaitForICECandidateRequest, srv proto.YAMRPOfferer_WaitForICECandidateServer) error {
	// client closes the stream when it connected to the answerer.
	// or timeout also closes the stream.
	// FIXME auth

	o.RLock()
	ch, ok := o.AnsID2iceToAnsChan[req.AnswererId]
	o.RUnlock()
	if !ok {
		log.Warnf("ice chan not found for answerID %s", req.AnswererId)
		return status.Errorf(codes.NotFound, "answerer not found")
	}
	// map should be eventually deleted even if the client isn't attached.
	// just setup a timer and delete the map when timeout.

	ctx, cancel := context.WithTimeout(context.Background(), waitingForIceCandidateAndAnswerTimeout)
	defer cancel()
	defer func() {
		o.closeIceToAnsCh <- req.AnswererId
		o.Lock()
		delete(o.AnsID2iceToAnsChan, req.AnswererId)
		o.Unlock()
	}()
	for {
		select {
		case <-ctx.Done():
			log.Debugf("timeout stop forwarding ice candidate")
			return nil
		case ice := <-ch:
			log.Debugf("forwarding ice candidate %v", ice)
			err := srv.Send(&proto.IceCandidate{
				Candidate: ice,
			})
			if err == io.EOF {
				// TODO write a lifecycle management for the ice channel

				return nil
			}
			if err != nil {
				return err
			}
		}
	}
}

// SendIceCandidate sends the ice candidate to the answerer.
func (o *OffererServer) SendIceCandidate(req proto.YAMRPOfferer_SendIceCandidateServer) error {
	// FIXME auth

	for {
		select {
		case <-req.Context().Done():
			log.Warnf("ice stream timeout")
			return status.Errorf(codes.DeadlineExceeded, "timeout")
		case <-func() chan struct{} {
			sig := make(chan struct{})
			go func() {
				defer func() {
					sig <- struct{}{}
				}()

				req, err := req.Recv()
				if req != nil {
					// FIXME validate the ice candidate

					log.Debugf("forwarding ice candidate to answerer %s", req.AnswererId)
					o.iceToAnswerCh <- IcePacket{
						answererID:   req.AnswererId,
						iceCandidate: req.Body,
					}
					log.Debugf("ice candidate forwarded to answerer %s", req.AnswererId)
				}
				if err != nil && err != io.EOF {
					log.Warnf("error when receiving ice candidate: %v", err)
					return
				}
			}()
			return sig
		}():
			// FIXME listen to the close ice channel

		}
	}

}

// Serve blocks the current goroutine and serves the answerer server.
func (o *OffererServer) Serve() error {
	go o.iceCandidateRouter()
	go o.answerRouter()
	return nil

}

// TODO FEAT: use a ring as monotonic queue to manage the ice channel.

func (o *OffererServer) iceCandidateRouter() {
	for {
		select {
		case icePacket := <-o.iceToOfferCh:
			log.Debugf("ice packet received: %s", icePacket)
			o.RLock()
			ch, ok := o.AnsID2iceToAnsChan[icePacket.answererID]
			o.RUnlock()
			if !ok {
				log.Warnf("ice channel %s not found: %v", icePacket.answererID, icePacket)
				continue
			}
			log.Debugf("forwarding ice candidate %s", icePacket)
			ch <- icePacket.iceCandidate
			log.Debugf("ice candidate forwarded %s", icePacket)
		}
	}
}

func (o *OffererServer) answerRouter() {
	for {
		select {
		case ansPacket := <-o.recvAnswerCh:
			log.Debugf("answer packet received: %v", ansPacket)
			o.RLock()
			ch, ok := o.AnsID2AnsChan[ansPacket.answererID]
			o.RUnlock()
			if !ok {
				log.Warn("answer channel not found", ansPacket)
				continue
			}
			log.Debugf("forwarding answer %s", ansPacket)
			ch <- ansPacket.answer
			log.Debugf("answer forwarded %s", ansPacket)
		}
	}
}
