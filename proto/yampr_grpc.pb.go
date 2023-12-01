// Code generated by protoc-gen-go-grpc. DO NOT EDIT.
// versions:
// - protoc-gen-go-grpc v1.2.0
// - protoc             v4.24.3
// source: yampr.proto

package proto

import (
	context "context"
	grpc "google.golang.org/grpc"
	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
)

// This is a compile-time assertion to ensure that this generated file
// is compatible with the grpc package it is being compiled against.
// Requires gRPC-Go v1.32.0 or later.
const _ = grpc.SupportPackageIsVersion7

// YAMRPAnswererClient is the client API for YAMRPAnswerer service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
type YAMRPAnswererClient interface {
	WaitForOffer(ctx context.Context, in *WaitForOfferRequest, opts ...grpc.CallOption) (*OfferResponse, error)
	SendAnswer(ctx context.Context, in *ReplyToRequest, opts ...grpc.CallOption) (*AnswerResponse, error)
	SendIceCandidate(ctx context.Context, opts ...grpc.CallOption) (YAMRPAnswerer_SendIceCandidateClient, error)
	WaitForICECandidate(ctx context.Context, in *WaitForICECandidateRequest, opts ...grpc.CallOption) (YAMRPAnswerer_WaitForICECandidateClient, error)
}

type yAMRPAnswererClient struct {
	cc grpc.ClientConnInterface
}

func NewYAMRPAnswererClient(cc grpc.ClientConnInterface) YAMRPAnswererClient {
	return &yAMRPAnswererClient{cc}
}

func (c *yAMRPAnswererClient) WaitForOffer(ctx context.Context, in *WaitForOfferRequest, opts ...grpc.CallOption) (*OfferResponse, error) {
	out := new(OfferResponse)
	err := c.cc.Invoke(ctx, "/YAMRPAnswerer/WaitForOffer", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *yAMRPAnswererClient) SendAnswer(ctx context.Context, in *ReplyToRequest, opts ...grpc.CallOption) (*AnswerResponse, error) {
	out := new(AnswerResponse)
	err := c.cc.Invoke(ctx, "/YAMRPAnswerer/SendAnswer", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *yAMRPAnswererClient) SendIceCandidate(ctx context.Context, opts ...grpc.CallOption) (YAMRPAnswerer_SendIceCandidateClient, error) {
	stream, err := c.cc.NewStream(ctx, &YAMRPAnswerer_ServiceDesc.Streams[0], "/YAMRPAnswerer/SendIceCandidate", opts...)
	if err != nil {
		return nil, err
	}
	x := &yAMRPAnswererSendIceCandidateClient{stream}
	return x, nil
}

type YAMRPAnswerer_SendIceCandidateClient interface {
	Send(*ReplyToRequest) error
	CloseAndRecv() (*SendIceCandidateResponse, error)
	grpc.ClientStream
}

type yAMRPAnswererSendIceCandidateClient struct {
	grpc.ClientStream
}

func (x *yAMRPAnswererSendIceCandidateClient) Send(m *ReplyToRequest) error {
	return x.ClientStream.SendMsg(m)
}

func (x *yAMRPAnswererSendIceCandidateClient) CloseAndRecv() (*SendIceCandidateResponse, error) {
	if err := x.ClientStream.CloseSend(); err != nil {
		return nil, err
	}
	m := new(SendIceCandidateResponse)
	if err := x.ClientStream.RecvMsg(m); err != nil {
		return nil, err
	}
	return m, nil
}

func (c *yAMRPAnswererClient) WaitForICECandidate(ctx context.Context, in *WaitForICECandidateRequest, opts ...grpc.CallOption) (YAMRPAnswerer_WaitForICECandidateClient, error) {
	stream, err := c.cc.NewStream(ctx, &YAMRPAnswerer_ServiceDesc.Streams[1], "/YAMRPAnswerer/WaitForICECandidate", opts...)
	if err != nil {
		return nil, err
	}
	x := &yAMRPAnswererWaitForICECandidateClient{stream}
	if err := x.ClientStream.SendMsg(in); err != nil {
		return nil, err
	}
	if err := x.ClientStream.CloseSend(); err != nil {
		return nil, err
	}
	return x, nil
}

type YAMRPAnswerer_WaitForICECandidateClient interface {
	Recv() (*IceCandidate, error)
	grpc.ClientStream
}

type yAMRPAnswererWaitForICECandidateClient struct {
	grpc.ClientStream
}

func (x *yAMRPAnswererWaitForICECandidateClient) Recv() (*IceCandidate, error) {
	m := new(IceCandidate)
	if err := x.ClientStream.RecvMsg(m); err != nil {
		return nil, err
	}
	return m, nil
}

// YAMRPAnswererServer is the server API for YAMRPAnswerer service.
// All implementations must embed UnimplementedYAMRPAnswererServer
// for forward compatibility
type YAMRPAnswererServer interface {
	WaitForOffer(context.Context, *WaitForOfferRequest) (*OfferResponse, error)
	SendAnswer(context.Context, *ReplyToRequest) (*AnswerResponse, error)
	SendIceCandidate(YAMRPAnswerer_SendIceCandidateServer) error
	WaitForICECandidate(*WaitForICECandidateRequest, YAMRPAnswerer_WaitForICECandidateServer) error
	mustEmbedUnimplementedYAMRPAnswererServer()
}

// UnimplementedYAMRPAnswererServer must be embedded to have forward compatible implementations.
type UnimplementedYAMRPAnswererServer struct {
}

func (UnimplementedYAMRPAnswererServer) WaitForOffer(context.Context, *WaitForOfferRequest) (*OfferResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method WaitForOffer not implemented")
}
func (UnimplementedYAMRPAnswererServer) SendAnswer(context.Context, *ReplyToRequest) (*AnswerResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method SendAnswer not implemented")
}
func (UnimplementedYAMRPAnswererServer) SendIceCandidate(YAMRPAnswerer_SendIceCandidateServer) error {
	return status.Errorf(codes.Unimplemented, "method SendIceCandidate not implemented")
}
func (UnimplementedYAMRPAnswererServer) WaitForICECandidate(*WaitForICECandidateRequest, YAMRPAnswerer_WaitForICECandidateServer) error {
	return status.Errorf(codes.Unimplemented, "method WaitForICECandidate not implemented")
}
func (UnimplementedYAMRPAnswererServer) mustEmbedUnimplementedYAMRPAnswererServer() {}

// UnsafeYAMRPAnswererServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to YAMRPAnswererServer will
// result in compilation errors.
type UnsafeYAMRPAnswererServer interface {
	mustEmbedUnimplementedYAMRPAnswererServer()
}

func RegisterYAMRPAnswererServer(s grpc.ServiceRegistrar, srv YAMRPAnswererServer) {
	s.RegisterService(&YAMRPAnswerer_ServiceDesc, srv)
}

func _YAMRPAnswerer_WaitForOffer_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(WaitForOfferRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(YAMRPAnswererServer).WaitForOffer(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/YAMRPAnswerer/WaitForOffer",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(YAMRPAnswererServer).WaitForOffer(ctx, req.(*WaitForOfferRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _YAMRPAnswerer_SendAnswer_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(ReplyToRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(YAMRPAnswererServer).SendAnswer(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/YAMRPAnswerer/SendAnswer",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(YAMRPAnswererServer).SendAnswer(ctx, req.(*ReplyToRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _YAMRPAnswerer_SendIceCandidate_Handler(srv interface{}, stream grpc.ServerStream) error {
	return srv.(YAMRPAnswererServer).SendIceCandidate(&yAMRPAnswererSendIceCandidateServer{stream})
}

type YAMRPAnswerer_SendIceCandidateServer interface {
	SendAndClose(*SendIceCandidateResponse) error
	Recv() (*ReplyToRequest, error)
	grpc.ServerStream
}

type yAMRPAnswererSendIceCandidateServer struct {
	grpc.ServerStream
}

func (x *yAMRPAnswererSendIceCandidateServer) SendAndClose(m *SendIceCandidateResponse) error {
	return x.ServerStream.SendMsg(m)
}

func (x *yAMRPAnswererSendIceCandidateServer) Recv() (*ReplyToRequest, error) {
	m := new(ReplyToRequest)
	if err := x.ServerStream.RecvMsg(m); err != nil {
		return nil, err
	}
	return m, nil
}

func _YAMRPAnswerer_WaitForICECandidate_Handler(srv interface{}, stream grpc.ServerStream) error {
	m := new(WaitForICECandidateRequest)
	if err := stream.RecvMsg(m); err != nil {
		return err
	}
	return srv.(YAMRPAnswererServer).WaitForICECandidate(m, &yAMRPAnswererWaitForICECandidateServer{stream})
}

type YAMRPAnswerer_WaitForICECandidateServer interface {
	Send(*IceCandidate) error
	grpc.ServerStream
}

type yAMRPAnswererWaitForICECandidateServer struct {
	grpc.ServerStream
}

func (x *yAMRPAnswererWaitForICECandidateServer) Send(m *IceCandidate) error {
	return x.ServerStream.SendMsg(m)
}

// YAMRPAnswerer_ServiceDesc is the grpc.ServiceDesc for YAMRPAnswerer service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var YAMRPAnswerer_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "YAMRPAnswerer",
	HandlerType: (*YAMRPAnswererServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "WaitForOffer",
			Handler:    _YAMRPAnswerer_WaitForOffer_Handler,
		},
		{
			MethodName: "SendAnswer",
			Handler:    _YAMRPAnswerer_SendAnswer_Handler,
		},
	},
	Streams: []grpc.StreamDesc{
		{
			StreamName:    "SendIceCandidate",
			Handler:       _YAMRPAnswerer_SendIceCandidate_Handler,
			ClientStreams: true,
		},
		{
			StreamName:    "WaitForICECandidate",
			Handler:       _YAMRPAnswerer_WaitForICECandidate_Handler,
			ServerStreams: true,
		},
	},
	Metadata: "yampr.proto",
}

// YAMRPOffererClient is the client API for YAMRPOfferer service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
type YAMRPOffererClient interface {
	SendOffer(ctx context.Context, in *SendOfferRequest, opts ...grpc.CallOption) (*OfferResponse, error)
	WaitForAnswer(ctx context.Context, in *WaitForAnswerRequest, opts ...grpc.CallOption) (*AnswerResponse, error)
	SendIceCandidate(ctx context.Context, opts ...grpc.CallOption) (YAMRPOfferer_SendIceCandidateClient, error)
	WaitForICECandidate(ctx context.Context, in *WaitForICECandidateRequest, opts ...grpc.CallOption) (YAMRPOfferer_WaitForICECandidateClient, error)
}

type yAMRPOffererClient struct {
	cc grpc.ClientConnInterface
}

func NewYAMRPOffererClient(cc grpc.ClientConnInterface) YAMRPOffererClient {
	return &yAMRPOffererClient{cc}
}

func (c *yAMRPOffererClient) SendOffer(ctx context.Context, in *SendOfferRequest, opts ...grpc.CallOption) (*OfferResponse, error) {
	out := new(OfferResponse)
	err := c.cc.Invoke(ctx, "/YAMRPOfferer/SendOffer", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *yAMRPOffererClient) WaitForAnswer(ctx context.Context, in *WaitForAnswerRequest, opts ...grpc.CallOption) (*AnswerResponse, error) {
	out := new(AnswerResponse)
	err := c.cc.Invoke(ctx, "/YAMRPOfferer/WaitForAnswer", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *yAMRPOffererClient) SendIceCandidate(ctx context.Context, opts ...grpc.CallOption) (YAMRPOfferer_SendIceCandidateClient, error) {
	stream, err := c.cc.NewStream(ctx, &YAMRPOfferer_ServiceDesc.Streams[0], "/YAMRPOfferer/SendIceCandidate", opts...)
	if err != nil {
		return nil, err
	}
	x := &yAMRPOffererSendIceCandidateClient{stream}
	return x, nil
}

type YAMRPOfferer_SendIceCandidateClient interface {
	Send(*ReplyToRequest) error
	CloseAndRecv() (*SendIceCandidateResponse, error)
	grpc.ClientStream
}

type yAMRPOffererSendIceCandidateClient struct {
	grpc.ClientStream
}

func (x *yAMRPOffererSendIceCandidateClient) Send(m *ReplyToRequest) error {
	return x.ClientStream.SendMsg(m)
}

func (x *yAMRPOffererSendIceCandidateClient) CloseAndRecv() (*SendIceCandidateResponse, error) {
	if err := x.ClientStream.CloseSend(); err != nil {
		return nil, err
	}
	m := new(SendIceCandidateResponse)
	if err := x.ClientStream.RecvMsg(m); err != nil {
		return nil, err
	}
	return m, nil
}

func (c *yAMRPOffererClient) WaitForICECandidate(ctx context.Context, in *WaitForICECandidateRequest, opts ...grpc.CallOption) (YAMRPOfferer_WaitForICECandidateClient, error) {
	stream, err := c.cc.NewStream(ctx, &YAMRPOfferer_ServiceDesc.Streams[1], "/YAMRPOfferer/WaitForICECandidate", opts...)
	if err != nil {
		return nil, err
	}
	x := &yAMRPOffererWaitForICECandidateClient{stream}
	if err := x.ClientStream.SendMsg(in); err != nil {
		return nil, err
	}
	if err := x.ClientStream.CloseSend(); err != nil {
		return nil, err
	}
	return x, nil
}

type YAMRPOfferer_WaitForICECandidateClient interface {
	Recv() (*IceCandidate, error)
	grpc.ClientStream
}

type yAMRPOffererWaitForICECandidateClient struct {
	grpc.ClientStream
}

func (x *yAMRPOffererWaitForICECandidateClient) Recv() (*IceCandidate, error) {
	m := new(IceCandidate)
	if err := x.ClientStream.RecvMsg(m); err != nil {
		return nil, err
	}
	return m, nil
}

// YAMRPOffererServer is the server API for YAMRPOfferer service.
// All implementations must embed UnimplementedYAMRPOffererServer
// for forward compatibility
type YAMRPOffererServer interface {
	SendOffer(context.Context, *SendOfferRequest) (*OfferResponse, error)
	WaitForAnswer(context.Context, *WaitForAnswerRequest) (*AnswerResponse, error)
	SendIceCandidate(YAMRPOfferer_SendIceCandidateServer) error
	WaitForICECandidate(*WaitForICECandidateRequest, YAMRPOfferer_WaitForICECandidateServer) error
	mustEmbedUnimplementedYAMRPOffererServer()
}

// UnimplementedYAMRPOffererServer must be embedded to have forward compatible implementations.
type UnimplementedYAMRPOffererServer struct {
}

func (UnimplementedYAMRPOffererServer) SendOffer(context.Context, *SendOfferRequest) (*OfferResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method SendOffer not implemented")
}
func (UnimplementedYAMRPOffererServer) WaitForAnswer(context.Context, *WaitForAnswerRequest) (*AnswerResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method WaitForAnswer not implemented")
}
func (UnimplementedYAMRPOffererServer) SendIceCandidate(YAMRPOfferer_SendIceCandidateServer) error {
	return status.Errorf(codes.Unimplemented, "method SendIceCandidate not implemented")
}
func (UnimplementedYAMRPOffererServer) WaitForICECandidate(*WaitForICECandidateRequest, YAMRPOfferer_WaitForICECandidateServer) error {
	return status.Errorf(codes.Unimplemented, "method WaitForICECandidate not implemented")
}
func (UnimplementedYAMRPOffererServer) mustEmbedUnimplementedYAMRPOffererServer() {}

// UnsafeYAMRPOffererServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to YAMRPOffererServer will
// result in compilation errors.
type UnsafeYAMRPOffererServer interface {
	mustEmbedUnimplementedYAMRPOffererServer()
}

func RegisterYAMRPOffererServer(s grpc.ServiceRegistrar, srv YAMRPOffererServer) {
	s.RegisterService(&YAMRPOfferer_ServiceDesc, srv)
}

func _YAMRPOfferer_SendOffer_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(SendOfferRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(YAMRPOffererServer).SendOffer(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/YAMRPOfferer/SendOffer",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(YAMRPOffererServer).SendOffer(ctx, req.(*SendOfferRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _YAMRPOfferer_WaitForAnswer_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(WaitForAnswerRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(YAMRPOffererServer).WaitForAnswer(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/YAMRPOfferer/WaitForAnswer",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(YAMRPOffererServer).WaitForAnswer(ctx, req.(*WaitForAnswerRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _YAMRPOfferer_SendIceCandidate_Handler(srv interface{}, stream grpc.ServerStream) error {
	return srv.(YAMRPOffererServer).SendIceCandidate(&yAMRPOffererSendIceCandidateServer{stream})
}

type YAMRPOfferer_SendIceCandidateServer interface {
	SendAndClose(*SendIceCandidateResponse) error
	Recv() (*ReplyToRequest, error)
	grpc.ServerStream
}

type yAMRPOffererSendIceCandidateServer struct {
	grpc.ServerStream
}

func (x *yAMRPOffererSendIceCandidateServer) SendAndClose(m *SendIceCandidateResponse) error {
	return x.ServerStream.SendMsg(m)
}

func (x *yAMRPOffererSendIceCandidateServer) Recv() (*ReplyToRequest, error) {
	m := new(ReplyToRequest)
	if err := x.ServerStream.RecvMsg(m); err != nil {
		return nil, err
	}
	return m, nil
}

func _YAMRPOfferer_WaitForICECandidate_Handler(srv interface{}, stream grpc.ServerStream) error {
	m := new(WaitForICECandidateRequest)
	if err := stream.RecvMsg(m); err != nil {
		return err
	}
	return srv.(YAMRPOffererServer).WaitForICECandidate(m, &yAMRPOffererWaitForICECandidateServer{stream})
}

type YAMRPOfferer_WaitForICECandidateServer interface {
	Send(*IceCandidate) error
	grpc.ServerStream
}

type yAMRPOffererWaitForICECandidateServer struct {
	grpc.ServerStream
}

func (x *yAMRPOffererWaitForICECandidateServer) Send(m *IceCandidate) error {
	return x.ServerStream.SendMsg(m)
}

// YAMRPOfferer_ServiceDesc is the grpc.ServiceDesc for YAMRPOfferer service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var YAMRPOfferer_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "YAMRPOfferer",
	HandlerType: (*YAMRPOffererServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "SendOffer",
			Handler:    _YAMRPOfferer_SendOffer_Handler,
		},
		{
			MethodName: "WaitForAnswer",
			Handler:    _YAMRPOfferer_WaitForAnswer_Handler,
		},
	},
	Streams: []grpc.StreamDesc{
		{
			StreamName:    "SendIceCandidate",
			Handler:       _YAMRPOfferer_SendIceCandidate_Handler,
			ClientStreams: true,
		},
		{
			StreamName:    "WaitForICECandidate",
			Handler:       _YAMRPOfferer_WaitForICECandidate_Handler,
			ServerStreams: true,
		},
	},
	Metadata: "yampr.proto",
}

// AuthClient is the client API for Auth service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
type AuthClient interface {
	LoginHost(ctx context.Context, in *InitHostRequest, opts ...grpc.CallOption) (*InitHostResponse, error)
	LoginClient(ctx context.Context, in *UserLogin, opts ...grpc.CallOption) (*AuthToken, error)
}

type authClient struct {
	cc grpc.ClientConnInterface
}

func NewAuthClient(cc grpc.ClientConnInterface) AuthClient {
	return &authClient{cc}
}

func (c *authClient) LoginHost(ctx context.Context, in *InitHostRequest, opts ...grpc.CallOption) (*InitHostResponse, error) {
	out := new(InitHostResponse)
	err := c.cc.Invoke(ctx, "/Auth/LoginHost", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *authClient) LoginClient(ctx context.Context, in *UserLogin, opts ...grpc.CallOption) (*AuthToken, error) {
	out := new(AuthToken)
	err := c.cc.Invoke(ctx, "/Auth/LoginClient", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// AuthServer is the server API for Auth service.
// All implementations must embed UnimplementedAuthServer
// for forward compatibility
type AuthServer interface {
	LoginHost(context.Context, *InitHostRequest) (*InitHostResponse, error)
	LoginClient(context.Context, *UserLogin) (*AuthToken, error)
	mustEmbedUnimplementedAuthServer()
}

// UnimplementedAuthServer must be embedded to have forward compatible implementations.
type UnimplementedAuthServer struct {
}

func (UnimplementedAuthServer) LoginHost(context.Context, *InitHostRequest) (*InitHostResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method LoginHost not implemented")
}
func (UnimplementedAuthServer) LoginClient(context.Context, *UserLogin) (*AuthToken, error) {
	return nil, status.Errorf(codes.Unimplemented, "method LoginClient not implemented")
}
func (UnimplementedAuthServer) mustEmbedUnimplementedAuthServer() {}

// UnsafeAuthServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to AuthServer will
// result in compilation errors.
type UnsafeAuthServer interface {
	mustEmbedUnimplementedAuthServer()
}

func RegisterAuthServer(s grpc.ServiceRegistrar, srv AuthServer) {
	s.RegisterService(&Auth_ServiceDesc, srv)
}

func _Auth_LoginHost_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(InitHostRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(AuthServer).LoginHost(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/Auth/LoginHost",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(AuthServer).LoginHost(ctx, req.(*InitHostRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Auth_LoginClient_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(UserLogin)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(AuthServer).LoginClient(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/Auth/LoginClient",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(AuthServer).LoginClient(ctx, req.(*UserLogin))
	}
	return interceptor(ctx, in, info, handler)
}

// Auth_ServiceDesc is the grpc.ServiceDesc for Auth service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var Auth_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "Auth",
	HandlerType: (*AuthServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "LoginHost",
			Handler:    _Auth_LoginHost_Handler,
		},
		{
			MethodName: "LoginClient",
			Handler:    _Auth_LoginClient_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "yampr.proto",
}

// HostClient is the client API for Host service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
type HostClient interface {
	ListenNewOffer(ctx context.Context, in *WaitForOfferRequest, opts ...grpc.CallOption) (Host_ListenNewOfferClient, error)
}

type hostClient struct {
	cc grpc.ClientConnInterface
}

func NewHostClient(cc grpc.ClientConnInterface) HostClient {
	return &hostClient{cc}
}

func (c *hostClient) ListenNewOffer(ctx context.Context, in *WaitForOfferRequest, opts ...grpc.CallOption) (Host_ListenNewOfferClient, error) {
	stream, err := c.cc.NewStream(ctx, &Host_ServiceDesc.Streams[0], "/Host/ListenNewOffer", opts...)
	if err != nil {
		return nil, err
	}
	x := &hostListenNewOfferClient{stream}
	if err := x.ClientStream.SendMsg(in); err != nil {
		return nil, err
	}
	if err := x.ClientStream.CloseSend(); err != nil {
		return nil, err
	}
	return x, nil
}

type Host_ListenNewOfferClient interface {
	Recv() (*Empty, error)
	grpc.ClientStream
}

type hostListenNewOfferClient struct {
	grpc.ClientStream
}

func (x *hostListenNewOfferClient) Recv() (*Empty, error) {
	m := new(Empty)
	if err := x.ClientStream.RecvMsg(m); err != nil {
		return nil, err
	}
	return m, nil
}

// HostServer is the server API for Host service.
// All implementations must embed UnimplementedHostServer
// for forward compatibility
type HostServer interface {
	ListenNewOffer(*WaitForOfferRequest, Host_ListenNewOfferServer) error
	mustEmbedUnimplementedHostServer()
}

// UnimplementedHostServer must be embedded to have forward compatible implementations.
type UnimplementedHostServer struct {
}

func (UnimplementedHostServer) ListenNewOffer(*WaitForOfferRequest, Host_ListenNewOfferServer) error {
	return status.Errorf(codes.Unimplemented, "method ListenNewOffer not implemented")
}
func (UnimplementedHostServer) mustEmbedUnimplementedHostServer() {}

// UnsafeHostServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to HostServer will
// result in compilation errors.
type UnsafeHostServer interface {
	mustEmbedUnimplementedHostServer()
}

func RegisterHostServer(s grpc.ServiceRegistrar, srv HostServer) {
	s.RegisterService(&Host_ServiceDesc, srv)
}

func _Host_ListenNewOffer_Handler(srv interface{}, stream grpc.ServerStream) error {
	m := new(WaitForOfferRequest)
	if err := stream.RecvMsg(m); err != nil {
		return err
	}
	return srv.(HostServer).ListenNewOffer(m, &hostListenNewOfferServer{stream})
}

type Host_ListenNewOfferServer interface {
	Send(*Empty) error
	grpc.ServerStream
}

type hostListenNewOfferServer struct {
	grpc.ServerStream
}

func (x *hostListenNewOfferServer) Send(m *Empty) error {
	return x.ServerStream.SendMsg(m)
}

// Host_ServiceDesc is the grpc.ServiceDesc for Host service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var Host_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "Host",
	HandlerType: (*HostServer)(nil),
	Methods:     []grpc.MethodDesc{},
	Streams: []grpc.StreamDesc{
		{
			StreamName:    "ListenNewOffer",
			Handler:       _Host_ListenNewOffer_Handler,
			ServerStreams: true,
		},
	},
	Metadata: "yampr.proto",
}
