package main

import (
	"fmt"
	"os"
	"os/signal"

	log "github.com/sirupsen/logrus"

	"github.com/ray1422/yamrp/client"
	"github.com/ray1422/yamrp/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func startClient(addr, username, password, token string) {
	fmt.Printf("start client on %s\n", addr)
	conn, err := grpc.Dial(
		grpcURL,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		panic(err)
	}
	defer conn.Close()
	offerAPI := proto.NewYAMRPOffererClient(conn)
	authAPI := proto.NewAuthClient(conn)
	lis, err := client.NewListener(
		addr,
		"tcp",
		client.NewPeerConnBuilder(),
		offerAPI,
		authAPI,
		token,
	)

	if err != nil {
		panic(err)
	}
	listenerCh := lis.Connect(nil)
	err = <-listenerCh
	if err != nil {
		log.Fatal("failed to connect to signaling server:", err)
	}
	err = lis.Bind()
	if err != nil {
		panic(err)
	}
	// listen ctrl+c signal
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)

	for {
		select {
		case <-c:
			err := shutdown()
			if err != nil {
				panic(err)
			}
			return
		}
	}
}

func shutdown() error {
	fmt.Println("received shutdown signal, shutting down...")
	fmt.Println("goodbye!")
	return nil
}
