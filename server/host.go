package server

import (
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/ray1422/yamrp/proto"
	"github.com/ray1422/yamrp/utils/async"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type hostClientStatus int

const (
	// hostClientStatusIdle is the status of the host client when it is idle.
	hostClientStatusIdle hostClientStatus = iota
	// hostClientStatusWaitingForOffer is the status of the host client when it is waiting for offer.
	hostClientStatusActive
)

// HostClientIdleTimeout is the timeout of the host client.
var HostClientIdleTimeout = 5 * time.Second

type hostClient struct {
	status hostClientStatus
	// offerSig is the signal to notify new offer. should be buffered.
	offerSig chan struct{}
	// stopListeningNewOffer is used for stopping ListenNewOffer stream. should be buffered.
	stopListeningNewOffer chan struct{}
}

// HostServerImpl is the implementation of the host server.
type HostServerImpl struct {
	// newHostCh is the channel to receive new host signal.
	newHostCh <-chan string
	// gcHostCh is the signal to check if the host is idle and should be garbage collected.
	gcHostCh chan string
	// closeHostCh is the signal to close the host. Used when the host is disconnected.
	closeHostCh chan string

	// newOfferCh is the channel to receive new offer.
	// string is the host id and hostServer will forward the signal to the host.
	newOfferCh            <-chan string
	hostClientRequestChan chan hostClientRequest

	clients map[string]*hostClient
	proto.UnimplementedHostServer
}

type hostClientRequest struct {
	ID string
	// returnChan must be buffered. nil is returned if the host is not found.
	returnChan chan *hostClient
}

var _ proto.HostServer = (*HostServerImpl)(nil)

// NewHostServer creates a new host server.
// `newHostSig` is used to receive new host. As soon as hostID is received, the host is registered.
// `offerCh` is the signal channel to receive new offer and identify the host by hostID.
//
// It should be guaranteed that offer is sent after the host registered, o.w. offer will be ignored.
func NewHostServer(newHostCh <-chan string, newOfferCh <-chan string) *HostServerImpl {
	return &HostServerImpl{
		newHostCh:             newHostCh,
		newOfferCh:            newOfferCh,
		clients:               make(map[string]*hostClient),
		gcHostCh:              make(chan string, 1024),
		closeHostCh:           make(chan string, 1024),
		hostClientRequestChan: make(chan hostClientRequest, 1024),
	}
}

// ListenNewOffer returns the channel to listen new offer.
func (hs *HostServerImpl) ListenNewOffer(req *proto.WaitForOfferRequest, stream proto.Host_ListenNewOfferServer) error {
	client := hs.client(req.GetHostId())
	if client == nil {
		return status.Errorf(codes.NotFound, "host not found")
	}
	client.status = hostClientStatusActive

	for {
		select {
		case <-client.stopListeningNewOffer:
			log.Infof("host %s stopped notifying new offer", req.GetHostId())
			return nil

		case <-client.offerSig:
			if err := stream.Send(&proto.Empty{}); err != nil {
				log.Warnf("host %s failed to send offer: %v", req.GetHostId(), err)
				return err
			}
			log.Infof("host %s notified new offer", req.GetHostId())
		}
	}

}

// Serve binds the listener and blocking serve.
func (hs *HostServerImpl) Serve() error {
	// start HostRegisterWorker
	go hs.HostLifeManager()
	// start gRPC server
	return nil
}

// helper for blocking get host client.
func (hs *HostServerImpl) client(hostID string) *hostClient {
	retChan := make(chan *hostClient, 1)
	hs.hostClientRequestChan <- hostClientRequest{
		ID:         hostID,
		returnChan: retChan,
	}
	return <-retChan
}

func (hs *HostServerImpl) clientFuture(hostID string) async.Future[*hostClient] {
	ret := async.NewFuture[*hostClient]()
	go func() {
		ret.Resolve(hs.client(hostID))
	}()
	return ret
}

// registerHost lifecycle of `hostClient` starts from here
func (hs *HostServerImpl) registerHost(hostID string) {
	hs.clients[hostID] = &hostClient{
		status: hostClientStatusIdle,
		// offerSig should be buffered to avoid blocking
		offerSig:              make(chan struct{}, 1024),
		stopListeningNewOffer: make(chan struct{}, 1),
	}
	timer := time.NewTimer(HostClientIdleTimeout)
	go func() {
		<-timer.C
		hs.gcHostCh <- hostID
	}()
}

func (hs *HostServerImpl) chkHostIdle(hostID string) {
	if hs.clients[hostID].status == hostClientStatusIdle {
		log.Infof("host %s is idle for %s, garbage collecting.", hostID, HostClientIdleTimeout.String())
		delete(hs.clients, hostID)
	}
}

func (hs *HostServerImpl) deleteHost(hostID string) {
	log.Infof("host %s is disconnected. deleting.", hostID)
	delete(hs.clients, hostID)
}
func (hs *HostServerImpl) handleHostClientRequest(hostReq hostClientRequest) {
	if client, ok := hs.clients[hostReq.ID]; ok {
		hostReq.returnChan <- client
	} else {
		hostReq.returnChan <- nil
	}
}

func (hs *HostServerImpl) handleNewOffer(hostID string) {
	log.Debugf("received new offer from %s", hostID)
	if client, ok := hs.clients[hostID]; ok {
		go func() {
			select {
			case client.offerSig <- struct{}{}:
				log.Infof("host %s notified new offer", hostID)
			case <-time.After(HostClientIdleTimeout):
				log.Warnf("host %s is idle for %s, dropping offer.", hostID,
					HostClientIdleTimeout.String())
				hs.closeHostCh <- hostID
			}
		}()
	} else {
		log.Warnf("host %s not found", hostID)
	}
}

// HostLifeManager is the worker to register new host.
func (hs *HostServerImpl) HostLifeManager() {
	// TODO graceful shutdown
	for {
		select {
		case hostID := <-hs.newHostCh:
			hs.registerHost(hostID)
		case hostID := <-hs.gcHostCh:
			hs.chkHostIdle(hostID)
		case hostID := <-hs.closeHostCh:
			hs.deleteHost(hostID)
		case hostReq := <-hs.hostClientRequestChan:
			hs.handleHostClientRequest(hostReq)
		case hostID := <-hs.newOfferCh:
			hs.handleNewOffer(hostID)
		}
	}
}
