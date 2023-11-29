package server

import (
	"context"

	log "github.com/sirupsen/logrus"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/ray1422/yamrp/proto"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// AuthServerImpl is the implementation of the auth server.
type AuthServerImpl struct {
	// newHostSig should be pushed when a new host is created.
	newHostSig chan string

	proto.UnimplementedAuthServer
}

var _ proto.AuthServer = (*AuthServerImpl)(nil)

// NewAuthServer creates a new auth server.
func NewAuthServer(newHostSig chan string) *AuthServerImpl {
	return &AuthServerImpl{
		newHostSig: newHostSig,
	}
}

// Serve binds the listener and blocking serve.
func (auth *AuthServerImpl) Serve() error {
	return nil
}

// LoginHost is the implementation of the login host.
func (auth *AuthServerImpl) LoginHost(ctx context.Context, req *proto.InitHostRequest) (*proto.InitHostResponse, error) {
	// TODO auth
	if req.UserLogin.Username == "neo" {
		// allocate a host id
		hostID := uuid.New().String()
		log.Infof("creating new host %s", hostID)

		clientJWT := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
			"host_id": hostID,
			"exp":     0, // never expire
		})
		clientJWTString, err := clientJWT.SignedString([]byte("TODO_YET_ANOTHER_SECRET"))
		if err != nil {
			return nil, err
		}
		log.Debugf("sending new host signal")
		auth.newHostSig <- hostID
		log.Debugf("sent new host signal")
		return &proto.InitHostResponse{
			HostId: hostID,
			Token: &proto.AuthToken{
				// example jwt token
				// FIXME use real jwt token
				Token: "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IkpvaG4gRG9lIiwiaWF0IjoxNTE2MjM5MDIyfQ.SflKxwRJSMeKKF2QT4fwpMeJf36POk6yJV_adQssw5c",
			},
			ClientSecret: &proto.AuthToken{
				Token: clientJWTString,
			},
		}, nil
	}
	// return  GRPC_STATUS_UNAUTHENTICATED
	return nil, status.Errorf(codes.Unauthenticated, "unauthenticated")

}

// LoginClient is the implementation of the login client.
func (auth *AuthServerImpl) LoginClient(ctx context.Context, req *proto.UserLogin) (*proto.AuthToken, error) {
	// TODO auth
	if req.Username == "neo" {
		return &proto.AuthToken{
			// example jwt token
			// FIXME use real jwt token
			Token: "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6Ik5lbyIsImlhdCI6MTUxNjIzOTAyMn0.U14v-sUdACtgNwMXr26XHBvVPrLiFAqks31oLLJyVZA",
		}, nil
	}
	return nil, status.Errorf(codes.Unauthenticated, "unauthenticated")
}
