package server

import (
	"context"
	"fmt"
	"io"
	"testing"

	log "github.com/sirupsen/logrus"

	"github.com/golang-jwt/jwt/v5"
	"github.com/ray1422/yamrp/proto"
	"github.com/ray1422/yamrp/utils/mock"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

const testingBindingAddr = ":5555"

func TestLoginHost(t *testing.T) {
	ch := make(chan string)
	lis := mock.NewMockListener()

	// start a testing server
	authServer := NewAuthServer(ch)
	go func() {
		s := grpc.NewServer()
		proto.RegisterAuthServer(s, authServer)
		go s.Serve(lis)
	}()

	// grpc client
	conn, err := grpc.Dial("pipe",
		grpc.WithContextDialer(lis.DialContext), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	defer conn.Close()
	c := proto.NewAuthClient(conn)
	// FIXME the real server is not implemented yet, so testing with dummy data
	retCh := make(chan string)
	go func() {
		r, err := c.LoginHost(context.TODO(), &proto.InitHostRequest{
			UserLogin: &proto.UserLogin{
				Username: "neo",
			},
		})
		if err != io.EOF {
			assert.NoError(t, err)
		}
		clientSecret := r.GetClientSecret()
		assert.NotEmpty(t, clientSecret)

		// decode the client secret
		token, err := jwt.ParseWithClaims(clientSecret.GetToken(), &jwt.MapClaims{}, func(token *jwt.Token) (interface{}, error) {
			return []byte("TODO_YET_ANOTHER_SECRET"), nil
		})
		if v, ok := token.Claims.(*jwt.MapClaims); assert.True(t, ok) {
			fmt.Println("host_id:", (*v)["host_id"])
			assert.NotEmpty(t, (*v)["host_id"])
		} else {
			assert.Fail(t, "cannot parse the client secret")
		}
		fmt.Println("token:", token)
		assert.NoError(t, err)
		assert.NotNil(t, token)

		assert.NotNil(t, r)
		retCh <- r.GetHostId()
	}()
	// wait for the new host signal
	hostID := <-ch
	assert.NotNil(t, hostID)
	assert.Equal(t, hostID, <-retCh)
	fmt.Println("hostID:", hostID)

}
