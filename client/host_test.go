package client

import (
	"context"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/ray1422/yamrp/proto"
	"github.com/ray1422/yamrp/proto/mock_proto"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	gomock "go.uber.org/mock/gomock"
	"google.golang.org/grpc"
)

const testSecret = "secret"

func authAPIFixture(t *testing.T, ctrl *gomock.Controller) *mock_proto.MockAuthClient {
	ret := mock_proto.NewMockAuthClient(ctrl)
	// generate a valid JWT token
	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"host_id": "host_id",
		"exp":     0,
	})
	tokStr, err := tok.SignedString([]byte(testSecret))
	if err != nil {
		t.Fatal(err)
	}

	ret.EXPECT().LoginHost(gomock.Any(), gomock.Any()).Return(&proto.InitHostResponse{
		HostId: "host_id",
		// TODO should use a valid token
		Token: &proto.AuthToken{Token: `{"validJSON":true}`},
		ClientSecret: &proto.AuthToken{
			Token: tokStr,
		},
	}, nil).AnyTimes()
	return ret
}
func ListenNewOfferClientFixture(t *testing.T, ctrl *gomock.Controller) *mock_proto.MockHost_ListenNewOfferClient {
	cnt := 0
	ret := mock_proto.NewMockHost_ListenNewOfferClient(ctrl)
	ret.EXPECT().Recv().DoAndReturn(func() (*proto.Empty, error) {
		if cnt > 5 {
			return nil, io.EOF
		}
		cnt++
		time.Sleep(100 * time.Millisecond)
		return &proto.Empty{}, nil
	}).AnyTimes()

	return ret
}
func hostAPIFixture(t *testing.T, ctrl *gomock.Controller) *mock_proto.MockHostClient {
	ret := mock_proto.NewMockHostClient(ctrl)
	ret.EXPECT().ListenNewOffer(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(
			func(ctx context.Context, in *proto.WaitForOfferRequest, opts ...grpc.CallOption) (
				proto.Host_ListenNewOfferClient, error) {
				return ListenNewOfferClientFixture(t, ctrl), nil
			}).AnyTimes()
	return ret
}

func answerAPIFixture(t *testing.T, ctrl *gomock.Controller) *mock_proto.MockYAMRPAnswererClient {
	ret := mock_proto.NewMockYAMRPAnswererClient(ctrl)
	ret.EXPECT().WaitForOffer(gomock.Any(), gomock.Any()).DoAndReturn(
		func(ctx context.Context, in *proto.WaitForOfferRequest, opts ...grpc.CallOption) (
			proto.ReplyToRequest, error) {
			t.Skip("TODO")
			time.Sleep(20 * time.Second)
			panic("TODO")
		}).AnyTimes()
	return ret
}

func runTestHttpServer(addr string, handler http.Handler) *http.Server {
	server := &http.Server{
		Addr:    addr,
		Handler: handler,
	}
	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal("failed to listen and serve: %v", err)
		}
	}()
	return server
}

func TestHostLogin(t *testing.T) {
	ctrl := gomock.NewController(t)
	authAPI := authAPIFixture(t, ctrl)
	hostAPI := hostAPIFixture(t, ctrl)

	answerAPI := answerAPIFixture(t, ctrl)

	addr := "localhost:3000"
	// open a temp server on addr
	runTestHttpServer(addr, nil)

	host, err := HostLogin("username", "password", authAPI, hostAPI, answerAPI, addr)
	if !assert.NoError(t, err) {
		assert.FailNow(t, "failed to login host")
	}
	assert.NotNil(t, host)
	assert.NotEmpty(t, host.authToken)
	_, _, _ = authAPI, hostAPI, answerAPI

	// TODO make agent Mock

	// sleep 5 seconds
	time.Sleep(5 * time.Second)

}
