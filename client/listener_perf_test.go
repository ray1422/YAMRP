package client

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/pion/webrtc/v4"
	log "github.com/sirupsen/logrus"
	"go.uber.org/mock/gomock"
)

func setupPeerConnPair() (pc1, pc2 peerConnAbstract) {
	readySig := make(chan struct{}, 2)
	peerA, err := peerConnBuilderImpl{}.NewPeerConnection(DefaultWebRTCConfig)
	if err != nil {
		panic("failed to create peerA")
	}
	peerB, err := peerConnBuilderImpl{}.NewPeerConnection(DefaultWebRTCConfig)
	if err != nil {
		panic("failed to create peerB")
	}
	peerA.OnConnectionStateChange(func(s webrtc.PeerConnectionState) {
		fmt.Println("peerA state", s)
		if s == webrtc.PeerConnectionStateConnected {
			readySig <- struct{}{}
		}
	})
	peerB.OnConnectionStateChange(func(s webrtc.PeerConnectionState) {
		fmt.Println("peerB state", s)
		if s == webrtc.PeerConnectionStateConnected {
			readySig <- struct{}{}
		}
	})

	peerA.OnICECandidate(func(c *webrtc.ICECandidate) {
		if c == nil {
			return
		}
		if err := peerB.AddICECandidate(c.ToJSON()); err != nil {
			panic(err)
		}
	})

	peerB.OnICECandidate(func(c *webrtc.ICECandidate) {
		if c == nil {
			return
		}
		if err := peerA.AddICECandidate(c.ToJSON()); err != nil {
			panic(err)
		}
	})

	_, err = peerA.CreateDataChannel("dummy", nil)
	if err != nil {
		panic("failed to create data channel: " + err.Error())
	}
	offer, err := peerA.CreateOffer(nil)
	if err != nil {
		panic(err)
	}
	if err := peerA.SetLocalDescription(offer); err != nil {
		panic(err)
	}
	if err := peerB.SetRemoteDescription(offer); err != nil {
		panic(err)
	}

	answer, err := peerB.CreateAnswer(nil)
	if err != nil {
		panic(err)
	}
	if err := peerB.SetLocalDescription(answer); err != nil {
		panic(err)
	}
	if err := peerA.SetRemoteDescription(answer); err != nil {
		panic(err)
	}

	<-readySig
	<-readySig

	return peerA, peerB
}

func BenchmarkOpenProxy(b *testing.B) {
	// given
	// open b.N connections, listener should open N proxies and close them
	// connection is closed by client

	log.SetLevel(log.WarnLevel)
	ctrl := gomock.NewController(b)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	lf := newListenerFixture(ctrl, nil, ctx)
	doneSig := make(chan struct{}, 10240)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		go func(i int) {
			conn, err := lf.dialer("pipe", "localhost")
			if err != nil {
				panic(err)
			}
			sigs := lf.dataChannelCtrl()
			sigs.onOpen <- struct{}{}
			sigs.onMsg <- webrtc.DataChannelMessage{Data: []byte("hello")}
			conn.Write([]byte("world"))
			if string(<-sigs.sentMsg) != "world" {
				b.Fail()
			}
			if i%2 == 0 {
				sigs.onClose <- struct{}{}
			} else {
				conn.Close()
			}
			doneSig <- struct{}{}
		}(i)
	}
	for i := 0; i < b.N; i++ {
		<-doneSig
	}
	for len(lf.listener.proxies) != 0 {
		time.Sleep(1 * time.Millisecond)
		// b.Log("waiting for proxies to close")
	}
	b.Log("done")
}

func BenchmarkOpenProxyWithRealPeerConn(b *testing.B) {
	peer, peerLis := setupPeerConnPair()
	log.SetLevel(log.WarnLevel)
	ctrl := gomock.NewController(b)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	lf := newListenerFixture(ctrl, peerLis, ctx)
	doneSig := make(chan struct{}, 10240)
	failSig := make(chan struct{}, 10240)
	b.ResetTimer()
	// when new connection open, read hello, send world, close
	peer.OnDataChannel(func(dc *webrtc.DataChannel) {
		dc.OnMessage(func(msg webrtc.DataChannelMessage) {
			go func() {
				if string(msg.Data) != "hello" {
					panic("unexpected message:" + string(msg.Data))
				}
				err := dc.Send([]byte("world"))
				if err != nil {
					panic(err)
				}
				doneSig <- struct{}{}
			}()
		})

	})
	for i := 0; i < b.N; i++ {
		go func(i int) {
			defer func() {
				r := recover()
				if r != nil {
					b.Log("panic:", r)
					failSig <- struct{}{}
				}
			}()
			conn, err := lf.dialer("pipe", "localhost")
			if err != nil {
				panic(err)
			}
			conn.Write([]byte("hello"))
			buf := make([]byte, 1024)
			n, err := conn.Read(buf)
			if n == 0 && err != nil {
				panic("failed to read message from connection: " + err.Error())
			}
			if string(buf[:n]) != "world" {
				panic("unexpected message")
			}
			conn.Close()

			doneSig <- struct{}{}
		}(i)
	}
	doneCnt, failedCnt := 0, 0
	for i := 0; i < b.N; i++ {
		select {
		case <-doneSig:
			doneCnt++
		case <-failSig:
			failedCnt++
		}
	}
	for len(lf.listener.proxies) != 0 {
		time.Sleep(1 * time.Millisecond)
		// b.Log("waiting for proxies to close")
	}
	b.Logf("done: %d, failed: %d", doneCnt, failedCnt)
}

func BenchmarkProxyBandWidthWithRealConn(b *testing.B) {
	pktSize := 32 * 1024
	readySig := make(chan struct{}, 1)
	peer, peerLis := setupPeerConnPair()
	log.SetLevel(log.WarnLevel)
	ctrl := gomock.NewController(b)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	lf := newListenerFixture(ctrl, peerLis, ctx)

	b.ResetTimer()

	// in the real case, peerConn should be created by the listener
	// but here we create it manually
	// so we just wait until it ready

	// when new connection open, read hello, send world, close
	peer.OnDataChannel(func(dc *webrtc.DataChannel) {
		wait := make(chan struct{})
		dc.OnOpen(func() {
			wait <- struct{}{}
		})
		go func() {

			<-wait
			b.Log("data channel opened")
			buf := make([]byte, pktSize)
			for i := 0; i < len(buf); i++ {
				buf[i] = byte(i % 256)
			}
			go func() {
				<-readySig
				b.Log("start sending")
				for i := 0; i < b.N; i++ {
					// write b.N KB
					err := dc.Send(buf)
					if err != nil {
						panic("failed to write: " + err.Error())
					}
				}
				b.Log("done sending")
			}()
		}()
	})
	doneCnt, failedCnt := 0, 0

	time.Sleep(1 * time.Second)
	conn, err := lf.dialer("pipe", "localhost")
	if err != nil {
		panic(err)
	}

	buf := make([]byte, pktSize)
	readySig <- struct{}{}
	b.ResetTimer()
	b.Log("start receiving")
	for i := 0; i < b.N; i++ {
		// write b.N KB
		// fmt.Println("reading", i)
		n, err := conn.Read(buf)
		if err != nil || n != len(buf) {
			panic("failed to read")
		}
		// fmt.Println("read", n, "bytes")
	}

	b.Logf("done: %d, failed: %d", doneCnt, failedCnt)

}
