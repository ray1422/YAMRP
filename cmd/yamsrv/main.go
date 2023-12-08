package main

import (
	"fmt"
	"net"
	"os"

	"github.com/ray1422/yamrp/server"
	log "github.com/sirupsen/logrus"
)

var grpcURL = "0.0.0.0:6666"

func main() {
	log.SetLevel(log.DebugLevel)
	if os.Getenv("GRPC_URL") != "" {
		grpcURL = os.Getenv("GRPC_URL")
	} else {
		fmt.Println("GRPC_URL is not set, use default value")
	}
	fmt.Println("Listen on:", grpcURL)
	lis, err := net.Listen("tcp", grpcURL)

	if err != nil {
		panic(err)
	}

	fmt.Printf("start server on %s\n", grpcURL)
	server.Serve(lis)

}
