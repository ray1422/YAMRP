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
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
)

func TestHostServer(t *testing.T) {
	// given
	// use un-buffered chan to ensure the order
	newHostCh := make(chan string)
	offerCh := make(chan string)
	hostServer := NewHostServer(newHostCh, offerCh)
	lis := mock.NewMockListener()
	go hostServer.Serve()
	grpcServer := grpc.NewServer()
	proto.RegisterHostServer(grpcServer, hostServer)
	go grpcServer.Serve(lis)

	conn, err := grpc.Dial("pipe", grpc.WithContextDialer(
		lis.DialContext), grpc.WithTransportCredentials(insecure.NewCredentials()))
	assert.NoError(t, err)
	client := proto.NewHostClient(conn)

	// when

	// then
	nilClient := hostServer.client("test_host")
	assert.Nil(t, nilClient)

	// when

	newHostCh <- "test_host"
	t.Log("sent new host signal")

	res, err := client.ListenNewOffer(context.Background(), &proto.WaitForOfferRequest{
		HostId: "test_host",
	})
	// then
	assert.NoError(t, err)
	assert.NotNil(t, res)
	offerCh <- "test_host"
	_, err = res.Recv()
	assert.Nil(t, err)

	// then
	c := hostServer.client("test_host")
	assert.Equal(t, hostClientStatusActive, c.status)
	assert.NotNil(t, client)

}

func TestHostTimeout(t *testing.T) {
	// given
	HostClientIdleTimeout = 1 * time.Millisecond

	newHostCh := make(chan string)
	offerCh := make(chan string)
	hostServer := NewHostServer(newHostCh, offerCh)

	lis := mock.NewMockListener()
	s := grpc.NewServer()
	proto.RegisterHostServer(s, hostServer)
	s.Serve(lis)
	err := hostServer.Serve()
	assert.NoError(t, err)

	// wait 100ms
	time.Sleep(100 * time.Millisecond)

	newHostCh <- "test_host"
	time.Sleep(HostClientIdleTimeout * 30)
	// then
	c := hostServer.client("test_host")
	assert.Nil(t, c)

}

func TestHostServerListenOnInvalidID(t *testing.T) {
	// given
	// use un-buffered chan to ensure the order
	newHostCh := make(chan string)
	offerCh := make(chan string)
	hostServer := NewHostServer(newHostCh, offerCh)
	lis := mock.NewMockListener()
	s := grpc.NewServer()
	proto.RegisterHostServer(s, hostServer)
	s.Serve(lis)
	err := hostServer.Serve()
	assert.NoError(t, err)
	// wait 100ms
	time.Sleep(100 * time.Millisecond)

	conn, err := grpc.Dial("pipe", grpc.WithContextDialer(lis.DialContext),
		grpc.WithTransportCredentials(insecure.NewCredentials()))
	assert.NoError(t, err)
	client := proto.NewHostClient(conn)

	// when

	// then
	nilClient := hostServer.client("test_host")
	assert.Nil(t, nilClient)

	// when

	newHostCh <- "test_host"
	res, err := client.ListenNewOffer(context.Background(), &proto.WaitForOfferRequest{
		HostId: "INVALID_HOST_ID",
	})
	_, err = res.Recv()
	// then
	assert.NotNil(t, err)
	if assert.NotEqual(t, err, io.EOF) {
		e, _ := status.FromError(err)
		assert.Equal(t, e.Code(), codes.NotFound)
	} else {
		t.Fatal("should be NotFound error but got EOF")
	}

}
