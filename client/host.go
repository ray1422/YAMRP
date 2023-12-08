package client

import (
	"context"
	"fmt"
	"io"

	log "github.com/sirupsen/logrus"

	"github.com/ray1422/yamrp/proto"
	"github.com/ray1422/yamrp/utils/async"
)

// Host is the host of the client.
type Host struct {
	authToken       string
	invitationToken string
	hostAPI         proto.HostClient
	answerAPI       proto.YAMRPAnswererClient
	// addr should be TODO should use dial builder
	user   UserModel
	addr   string
	agents map[string]*AgentNetConn
	hostID string
}

// HostLogin logs in the service and creates a new host.
func HostLogin(username string,
	password string,
	authAPI proto.AuthClient,
	hostAPI proto.HostClient,
	answerAPI proto.YAMRPAnswererClient,
	addr string,
) (*Host, error) {
	ctx := context.Background()
	log.Debugf("host %v login", hostAPI)
	res, err := authAPI.LoginHost(ctx, &proto.InitHostRequest{
		UserLogin: &proto.UserLogin{
			Username: username,
			Password: password,
		},
	})

	if err != nil {
		return nil, err
	}
	hostID := res.GetHostId()
	token := res.GetToken()
	invitationToken := res.GetClientSecret()
	fmt.Println("invitation token:", invitationToken)
	fmt.Println("host id:", hostID)
	fmt.Println("")
	user := &UserData{
		id:    hostID,
		token: token.GetToken(),
	}
	return newHost(hostID, token.Token, invitationToken.Token, hostAPI, answerAPI, user, addr)
}

// StartAgent starts a new agent.
func (h *Host) StartAgent() {
	agent, err := NewAgent(h.addr, "tcp", NewPeerConnBuilder(), h.user, h.answerAPI, h.hostID)
	if err != nil {
		log.Errorf("host %s failed to create new agent: %v", h.hostAPI, err)
		return
	}
	errCh := agent.Connect(nil)
	err = <-errCh
	if err != nil {
		log.Errorf("host %s failed to connect to agent: %v", h.hostAPI, err)
		return
	}
}

// NewOfferListener NewOfferListener
func (h *Host) NewOfferListener() (ret async.Future[error]) {
	ret = async.NewFuture[error]()
	go func() {
		ctx := context.Background()
		res, err := h.hostAPI.ListenNewOffer(ctx, &proto.WaitForOfferRequest{
			HostId: h.hostID,
		})
		if err != nil {
			ret.Resolve(err)
			return
		}
		ret.Resolve(err)
		log.Infof("host %s is listening new offer", h.hostAPI)
		for {
			offerRes, err := res.Recv()
			_ = offerRes
			if err != nil {
				if err == io.EOF {
					log.Infof("host %s stopped listening new offer", h.hostAPI)
					return
				}
				log.Errorf("host %s failed to receive new offer: %v", h.hostAPI, err)
				return
			}
			log.Infof("host received new offer")
			// fork a new agent
			go h.StartAgent()
			if err != nil {
				log.Errorf("host %s failed to create new agent: %v", h.hostAPI, err)
				return
			}
		}
	}()
	return
}
func newHost(hostID string,
	authToken string,
	invitationToken string,
	hostAPI proto.HostClient,
	answerAPI proto.YAMRPAnswererClient,
	user UserModel,
	addr string,
) (*Host, error) {
	h := &Host{
		authToken:       authToken,
		invitationToken: invitationToken,
		hostAPI:         hostAPI,
		answerAPI:       answerAPI,
		agents:          make(map[string]*AgentNetConn),
		user:            user,
		addr:            addr,
		hostID:          hostID,
	}
	err := async.Await(h.NewOfferListener())
	if err != nil {
		return nil, err
	}
	return h, nil

}
