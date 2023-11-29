package server

import "github.com/ray1422/yamrp/proto"

// ┌──────┐          ┌──────────┐                              ┌────────┐
// │      ├──Login───► AuthSrv  ◄─────────LoginClient──────────┤        │
// │      │  Host    └──────────┘                              │        │
// │      │               │ notifyNewHost                      │        │
// │      │          ┌────▼─────┐                              │        │
// │      ├─ListenOn─►          │                              │        │
// │ Host │ NewOffer │ Host     │                              │ Client │
// │      ◄──────────┤ Service  │◄──Notify─────┐               │        │
// │      ◄──────────┤          │              │               │        │
// │      │          └──────────┘              │               │        │
// │      │                                    │               │        │
// │      │          ┌──────────┐        ┌─────┴─────┐         │        │
// │      ├─WaitFor──► YAMRP    ◄─Offer──┤  YAMRP    ◄─Send────┤        │
// │      │ Offer    │ Answerer │        │  Offerer  │ Offer   │        │
// │      │          │          │        │           │         │        │
// │      ├─Send─────►          ├─Answer─►           ◄─WaitFor─┤        │
// │      │ Answer   │          │        │           │ Answer  │        │
// │      │          │          │        │           │         │        │
// │      ├─Send─────►          ├─Ice..──►           ◄─WaitFor─┤        │
// │      │ Ice....  │          │        │           │ Ice.... │        │
// │      │          │          ◄─Close──┤           │         │        │
// │      │          │          │ Ice    │           │         │        │
// │      │          │          │ Chan   │           │         │        │
// └──────┘          └──────────┘        └───────────┘         └────────┘

// YAMRPServer YAMRPServer. please see the flow chart above.
type YAMRPServer interface{}

var _ YAMRPServer = (*YAMRPServerImpl)(nil)

// YAMRPServerImpl is the implementation of the server.
type YAMRPServerImpl struct {
	offererServer  proto.YAMRPOffererServer
	answererServer proto.YAMRPAnswererServer
}

// NewYAMRPServer creates a new server.
func NewYAMRPServer() YAMRPServer {
	return &YAMRPServerImpl{}
}
