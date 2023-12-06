package main

import (
	"testing"
	"time"
)

func TestEnd2EndHostMain(t *testing.T) {
	// if not debug mode skip this test
	go func() {
		grpcURL = "arm.ray-fish.me:48762"
		addr := "localhost:22"
		startHost(addr, "neo", "neo")
		select {}
	}()
	// wait forever if no timeout, else skip in `timeout` seconds
	_, ok := t.Deadline()
	if ok {
		time.Sleep(1 * time.Second)
	} else {
		select {}
	}

	t.SkipNow()

}
