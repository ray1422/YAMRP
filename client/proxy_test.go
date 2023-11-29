package client

import (
	"fmt"
	"testing"
	"time"

	"github.com/pion/webrtc/v4"
	"github.com/stretchr/testify/assert"
	gomock "go.uber.org/mock/gomock"
)

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
			return 0, nil
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

	conn.EXPECT().Close().Return(nil).AnyTimes()

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

	dataChannel.EXPECT().Close().Return(nil).Times(1)
	outCnt := 0
	conn.EXPECT().Read(gomock.Any()).DoAndReturn(func(p []byte) (n int, err error) {
		if outCnt >= len(outgoingData) {
			return 0, nil
		}
		copy(p, outgoingData[outCnt])
		outCnt++
		return len(outgoingData[outCnt]), nil
	}).AnyTimes()

	// when

	lp := NewProxy("test-proxy-id", conn, dataChannel)
	readySig <- struct{}{}
	// wait 5 seconds for the data to be sent
	<-doneSig

	// then
	err := lp.Close()
	time.Sleep(300 * time.Millisecond)
	assert.NoError(t, err)

	for i := 0; i < len(incomingData); i++ {
		assert.Equal(t, incomingData[i], recv[i])
	}
	// trim empty data for recv and send

	assert.Equal(t, incomingData, recv)
	assert.Equal(t, outgoingData, send)
}
