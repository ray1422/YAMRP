package main

import (
	"os"
	"os/signal"

	"github.com/ray1422/yamrp/client"
	"github.com/ray1422/yamrp/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func startHost(addr, username, password string) {
	conn, err := grpc.Dial(
		grpcURL,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		panic(err)
	}
	authAPI := proto.NewAuthClient(conn)
	hostAPI := proto.NewHostClient(conn)
	answerAPI := proto.NewYAMRPAnswererClient(conn)

	host, err := client.HostLogin(username, password, authAPI, hostAPI, answerAPI, addr)
	if err != nil {
		panic(err)
	}
	_ = host
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	select {
	case <-c:
		err := shutdown()
		if err != nil {
			panic(err)
		}
	}
}
