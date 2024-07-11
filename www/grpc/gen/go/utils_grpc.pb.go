// Code generated by protoc-gen-go-grpc. DO NOT EDIT.
// versions:
// - protoc-gen-go-grpc v1.4.0
// - protoc             (unknown)
// source: utils.proto

package pactus

import (
	context "context"
	grpc "google.golang.org/grpc"
	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
)

// This is a compile-time assertion to ensure that this generated file
// is compatible with the grpc package it is being compiled against.
// Requires gRPC-Go v1.62.0 or later.
const _ = grpc.SupportPackageIsVersion8

const (
	Utils_SignMessageWithPrivateKey_FullMethodName = "/pactus.Utils/SignMessageWithPrivateKey"
	Utils_VerifyMessage_FullMethodName             = "/pactus.Utils/VerifyMessage"
)

// UtilsClient is the client API for Utils service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
//
// Util service defines various RPC methods for interacting with
// Utils.
type UtilsClient interface {
	// SignMessageWithPrivateKey sign message with provided private key
	SignMessageWithPrivateKey(ctx context.Context, in *SignMessageWithPrivateKeyRequest, opts ...grpc.CallOption) (*SignMessageWithPrivateKeyResponse, error)
	// VerifyMessage verify signature with public key and message
	VerifyMessage(ctx context.Context, in *VerifyMessageRequest, opts ...grpc.CallOption) (*VerifyMessageResponse, error)
}

type utilsClient struct {
	cc grpc.ClientConnInterface
}

func NewUtilsClient(cc grpc.ClientConnInterface) UtilsClient {
	return &utilsClient{cc}
}

func (c *utilsClient) SignMessageWithPrivateKey(ctx context.Context, in *SignMessageWithPrivateKeyRequest, opts ...grpc.CallOption) (*SignMessageWithPrivateKeyResponse, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(SignMessageWithPrivateKeyResponse)
	err := c.cc.Invoke(ctx, Utils_SignMessageWithPrivateKey_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *utilsClient) VerifyMessage(ctx context.Context, in *VerifyMessageRequest, opts ...grpc.CallOption) (*VerifyMessageResponse, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(VerifyMessageResponse)
	err := c.cc.Invoke(ctx, Utils_VerifyMessage_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// UtilsServer is the server API for Utils service.
// All implementations should embed UnimplementedUtilsServer
// for forward compatibility
//
// Util service defines various RPC methods for interacting with
// Utils.
type UtilsServer interface {
	// SignMessageWithPrivateKey sign message with provided private key
	SignMessageWithPrivateKey(context.Context, *SignMessageWithPrivateKeyRequest) (*SignMessageWithPrivateKeyResponse, error)
	// VerifyMessage verify signature with public key and message
	VerifyMessage(context.Context, *VerifyMessageRequest) (*VerifyMessageResponse, error)
}

// UnimplementedUtilsServer should be embedded to have forward compatible implementations.
type UnimplementedUtilsServer struct {
}

func (UnimplementedUtilsServer) SignMessageWithPrivateKey(context.Context, *SignMessageWithPrivateKeyRequest) (*SignMessageWithPrivateKeyResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method SignMessageWithPrivateKey not implemented")
}
func (UnimplementedUtilsServer) VerifyMessage(context.Context, *VerifyMessageRequest) (*VerifyMessageResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method VerifyMessage not implemented")
}

// UnsafeUtilsServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to UtilsServer will
// result in compilation errors.
type UnsafeUtilsServer interface {
	mustEmbedUnimplementedUtilsServer()
}

func RegisterUtilsServer(s grpc.ServiceRegistrar, srv UtilsServer) {
	s.RegisterService(&Utils_ServiceDesc, srv)
}

func _Utils_SignMessageWithPrivateKey_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(SignMessageWithPrivateKeyRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(UtilsServer).SignMessageWithPrivateKey(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: Utils_SignMessageWithPrivateKey_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(UtilsServer).SignMessageWithPrivateKey(ctx, req.(*SignMessageWithPrivateKeyRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Utils_VerifyMessage_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(VerifyMessageRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(UtilsServer).VerifyMessage(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: Utils_VerifyMessage_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(UtilsServer).VerifyMessage(ctx, req.(*VerifyMessageRequest))
	}
	return interceptor(ctx, in, info, handler)
}

// Utils_ServiceDesc is the grpc.ServiceDesc for Utils service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var Utils_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "pactus.Utils",
	HandlerType: (*UtilsServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "SignMessageWithPrivateKey",
			Handler:    _Utils_SignMessageWithPrivateKey_Handler,
		},
		{
			MethodName: "VerifyMessage",
			Handler:    _Utils_VerifyMessage_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "utils.proto",
}
