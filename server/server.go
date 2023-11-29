package server

import (
	"net"

	"github.com/ray1422/yamrp/proto"
	"google.golang.org/grpc"
)

// Serve starts the server and blocks the current goroutine.
func Serve(lis net.Listener, opts ...grpc.ServerOption) {

	newHostSig := make(chan string)
	notifyNewOfferCh := make(chan string)
	offerCh := make(chan offering)
	answerCh := make(chan AnsPacket, 1024)
	iceToAnsCh := make(chan IcePacket, 1024)
	iceToOfferCh := make(chan IcePacket, 1024)
	closeIceToAnsCh := make(chan string)

	authServer := NewAuthServer(newHostSig)
	hostServer := NewHostServer(newHostSig, notifyNewOfferCh)
	offererServer := NewOffererServer(notifyNewOfferCh, offerCh, answerCh, iceToAnsCh, closeIceToAnsCh)
	answererServer := NewAnswererServer(offerCh, answerCh, iceToAnsCh, iceToOfferCh,
		closeIceToAnsCh)

	hostServer.Serve()
	authServer.Serve()
	offererServer.Serve()
	answererServer.Serve()

	s := grpc.NewServer(opts...)

	proto.RegisterHostServer(s, hostServer)
	proto.RegisterYAMRPOffererServer(s, offererServer)
	proto.RegisterYAMRPAnswererServer(s, answererServer)
	proto.RegisterAuthServer(s, authServer)
	s.Serve(lis)
	// select {}
}
