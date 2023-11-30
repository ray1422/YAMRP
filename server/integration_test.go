package server

import (
	"net"
	"net/http"
	"testing"
	"time"

	"github.com/ray1422/yamrp/client"
	"github.com/ray1422/yamrp/proto"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type userMock struct {
	token string
	id    string
}

func (u *userMock) Token() string {
	return u.token
}

func (u *userMock) ID() string {
	return u.id
}

func serverFixtureExposeHost(t *testing.T) (
	notifiedNewOfferCh chan string,
) {
	lis, err := net.Listen("tcp", "localhost:3222")
	assert.NoError(t, err)
	notifiedNewOfferCh = make(chan string, 1024)
	offerCh := make(chan offering)
	answerCh := make(chan AnsPacket, 1024)
	iceToAnsCh := make(chan IcePacket, 1024)

	iceToOfferCh := make(chan IcePacket, 1024)
	closeIceToAnsCh := make(chan string)
	// authServer := NewAuthServer(newHostSig)
	// hostServer = NewHostServer(newHostCh, newOfferCh)
	offererServer := NewOffererServer(notifiedNewOfferCh, offerCh, answerCh, iceToAnsCh, iceToOfferCh,
		closeIceToAnsCh)
	answererServer := NewAnswererServer(offerCh, answerCh, iceToAnsCh, iceToOfferCh, closeIceToAnsCh)

	// authServer.Serve()
	// hostServer.Serve()
	offererServer.Serve()
	answererServer.Serve()

	s := grpc.NewServer()

	// proto.RegisterHostServer(s, hostServer)
	proto.RegisterYAMRPOffererServer(s, offererServer)
	proto.RegisterYAMRPAnswererServer(s, answererServer)
	// proto.RegisterAuthServer(s, authServer)
	go s.Serve(lis)
	return
}
func startTestingServer(addr string) error {
	// simple http server
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("hello world"))
	})
	return http.ListenAndServe(addr, nil)

}
func TestClientServerWithoutListeningNewOffer(t *testing.T) {

	// mock newOfferCh, test only YAMRPAnswerer and YAMRPOfferer with real client

	// lis := mock.NewMockListener()

	notifiedNewOfferCh := serverFixtureExposeHost(t)
	// go startTestingServer("localhost:3000")
	_ = notifiedNewOfferCh
	// notifiedNewOfferCh <- "host_id"
	go func() {
		t.Log("receive host_id", <-notifiedNewOfferCh)
	}()
	t.Log("host_id is notified")

	// wait 100ms
	time.Sleep(100 * time.Millisecond)

	ccOff, err := grpc.Dial("localhost:3222",
		grpc.WithTransportCredentials(insecure.NewCredentials()))
	assert.NoError(t, err)
	offAPI := proto.NewYAMRPOffererClient(ccOff)

	ccAns, err := grpc.Dial("localhost:3222",
		grpc.WithTransportCredentials(insecure.NewCredentials()))
	assert.NoError(t, err)
	ansAPI := proto.NewYAMRPAnswererClient(ccAns)

	// wait 1s
	time.Sleep(1 * time.Second)

	listener, err := client.NewListener(
		"localhost:60000", "tcp",
		client.NewPeerConnBuilder(),
		offAPI, nil,
		"host_id")

	assert.NoError(t, err)
	assert.NotNil(t, listener)

	agent, err := client.NewAgent(
		"localhost:3000", "tcp",
		client.NewPeerConnBuilder(),
		&userMock{"my_token_a", "my_id_a"},
		ansAPI,
	)
	assert.NoError(t, err)
	assert.NotNil(t, agent)

	go func() {
		listener.Bind()
		errCh := listener.Connect(nil)
		t.Log("listener is connecting")
		err := <-errCh
		t.Log("listener is connected")
		assert.NoError(t, err)

	}()

	go func() {
		errCh := agent.Connect(nil)
		t.Log("agent is connecting")
		err := <-errCh
		t.Log("agent is connected")
		assert.NoError(t, err)
	}()
	time.Sleep(30 * time.Second)
}
