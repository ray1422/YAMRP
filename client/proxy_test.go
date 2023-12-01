package client

import (
	"fmt"
	"io"
	"sync"
	"testing"
	"time"

	"github.com/pion/webrtc/v4"
	"github.com/stretchr/testify/assert"
	gomock "go.uber.org/mock/gomock"
)

// MockConn is the mock of net.Conn.

var connCloseMutex = sync.Mutex{}
var connClosedCnt = 0

func TestProxy(t *testing.T) {
	// given
	readySig := make(chan struct{}, 1)
	doneSig := make(chan struct{}, 1)
	ctrl := gomock.NewController(t)

	// test remote server sending `incomingData` (i.e. "hello", "world", "!") to the proxy
	// and the user replies with `outgoingData` (i.e. "hua", "?")

	incomingData := [][]byte{
		[]byte("hello"),
		[]byte("world"),
		[]byte("!"),
	}
	outgoingData := [][]byte{
		[]byte("hua"),
		[]byte("?"),
		[]byte("1"),
		[]byte("2"),
		[]byte("3"),
	}

	recv := [][]byte{}
	send := [][]byte{}
	// setup Proxy
	conn := NewMockConn(ctrl)
	outgoingCnt := 0
	conn.EXPECT().Read(gomock.Any()).DoAndReturn(func(p []byte) (n int, err error) {
		if outgoingCnt >= len(outgoingData) {
			time.Sleep(100 * time.Millisecond)
			return 0, io.EOF
		}
		copy(p, outgoingData[outgoingCnt])
		fmt.Println("replying:", string(outgoingData[outgoingCnt]))
		outgoingCnt++
		return len(outgoingData[outgoingCnt-1]), nil
	}).AnyTimes()

	conn.EXPECT().Write(gomock.Any()).DoAndReturn(func(p []byte) (n int, err error) {
		recv = append(recv, append([]byte{}, p...))
		return len(p), nil
	}).AnyTimes()

	// close should called by listener, not proxy, so we don't mock it.
	// conn.EXPECT().Close().Do(func() {
	// 	connCloseMutex.Lock()
	// 	defer connCloseMutex.Unlock()
	// 	connClosedCnt++
	// }).Return(nil).AnyTimes()

	dataChannel := NewMockDataChannelAbstract(ctrl)

	dataChannel.EXPECT().Send(gomock.Any()).DoAndReturn(func(p []byte) error {
		// copy and write to recv
		send = append(send, append([]byte{}, p...))
		fmt.Println("sent:", string(p))
		return nil
	}).AnyTimes()

	dataChannel.EXPECT().OnMessage(gomock.Any()).DoAndReturn(func(f func(v webrtc.DataChannelMessage)) {
		// delay for a while, and send the data to the channel
		go func() {
			<-readySig
			for _, v := range incomingData {
				f(webrtc.DataChannelMessage{
					Data: v,
				})
				time.Sleep(100 * time.Millisecond)
			}
			doneSig <- struct{}{}
		}()
	}).Times(1)

	// on close should be handled by listener, so we don't mock it.

	// dataChannel.EXPECT().Close().Do(func() {
	// 	t.Log("data channel closed")
	// }).Return(nil).Times(1)
	outCnt := 0
	conn.EXPECT().Read(gomock.Any()).DoAndReturn(func(p []byte) (n int, err error) {
		if outCnt >= len(outgoingData) {
			t.Log("reach EOF")
			return 0, io.EOF
		}
		copy(p, outgoingData[outCnt])
		outCnt++
		return len(outgoingData[outCnt]), nil
	}).AnyTimes()

	// when
	eolSig := make(chan error, 1)
	lp := NewProxy("test-proxy-id", conn, dataChannel, eolSig)
	readySig <- struct{}{}

	<-doneSig

	// then
	// time.Sleep(100 * time.Millisecond)

	for i := 0; i < len(incomingData); i++ {
		assert.Equal(t, incomingData[i], recv[i])
	}
	// trim empty data for recv and send

	assert.Equal(t, incomingData, recv)
	assert.Equal(t, outgoingData, send)

	// test EoL
	// when
	// read reads io.EOF
	// then
	assert.Equal(t, io.EOF, <-eolSig)
	err := lp.Close()

	// time.Sleep(3000 * time.Millisecond)

	assert.NoError(t, err)

}
