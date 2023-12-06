package main

import (
	"fmt"
	"testing"
	"time"
)

func TestEnd2EndClientMain(t *testing.T) {
	// if not debug mode skip this test
	go func() {
		grpcURL = "arm.ray-fish.me:48762"
		addr, username, password, hostToken :=
			"localhost:3000",
			"test",
			"test",
			"9fb95ab0-9554-4ffe-8788-301334d0ac3c"
		fmt.Println("please input the token of the host: ")
		// fmt.Scanln(&hostToken)
		startClient(addr, username, password, hostToken)
	}()
	_, ok := t.Deadline()
	if ok {

		time.Sleep(1 * time.Second)
	} else {
		select {}
	}
	t.Skip()
}
