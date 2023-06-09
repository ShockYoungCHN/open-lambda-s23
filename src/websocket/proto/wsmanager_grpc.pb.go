// Code generated by protoc-gen-go-grpc. DO NOT EDIT.
// versions:
// - protoc-gen-go-grpc v1.3.0
// - protoc             v3.11.2
// source: proto/wsmanager.proto

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

const (
	WsManager_PostToConnection_FullMethodName = "/proto.WsManager/PostToConnection"
)

// WsManagerClient is the client API for WsManager service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
type WsManagerClient interface {
	PostToConnection(ctx context.Context, in *PostToConnectionRequest, opts ...grpc.CallOption) (*PostToConnectionResponse, error)
}

type wsManagerClient struct {
	cc grpc.ClientConnInterface
}

func NewWsManagerClient(cc grpc.ClientConnInterface) WsManagerClient {
	return &wsManagerClient{cc}
}

func (c *wsManagerClient) PostToConnection(ctx context.Context, in *PostToConnectionRequest, opts ...grpc.CallOption) (*PostToConnectionResponse, error) {
	out := new(PostToConnectionResponse)
	err := c.cc.Invoke(ctx, WsManager_PostToConnection_FullMethodName, in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// WsManagerServer is the server API for WsManager service.
// All implementations must embed UnimplementedWsManagerServer
// for forward compatibility
type WsManagerServer interface {
	PostToConnection(context.Context, *PostToConnectionRequest) (*PostToConnectionResponse, error)
	mustEmbedUnimplementedWsManagerServer()
}

// UnimplementedWsManagerServer must be embedded to have forward compatible implementations.
type UnimplementedWsManagerServer struct {
}

func (UnimplementedWsManagerServer) PostToConnection(context.Context, *PostToConnectionRequest) (*PostToConnectionResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method PostToConnection not implemented")
}
func (UnimplementedWsManagerServer) mustEmbedUnimplementedWsManagerServer() {}

// UnsafeWsManagerServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to WsManagerServer will
// result in compilation errors.
type UnsafeWsManagerServer interface {
	mustEmbedUnimplementedWsManagerServer()
}

func RegisterWsManagerServer(s grpc.ServiceRegistrar, srv WsManagerServer) {
	s.RegisterService(&WsManager_ServiceDesc, srv)
}

func _WsManager_PostToConnection_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(PostToConnectionRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(WsManagerServer).PostToConnection(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: WsManager_PostToConnection_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(WsManagerServer).PostToConnection(ctx, req.(*PostToConnectionRequest))
	}
	return interceptor(ctx, in, info, handler)
}

// WsManager_ServiceDesc is the grpc.ServiceDesc for WsManager service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var WsManager_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "proto.WsManager",
	HandlerType: (*WsManagerServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "PostToConnection",
			Handler:    _WsManager_PostToConnection_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "proto/wsmanager.proto",
}
