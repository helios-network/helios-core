// Copyright Tharsis Labs Ltd.(Evmos)
// SPDX-License-Identifier:ENCL-1.0(https://github.com/evmos/evmos/blob/main/LICENSE)

// Code generated by protoc-gen-go-grpc. DO NOT EDIT.
// versions:
// - protoc-gen-go-grpc v1.3.0
// - protoc             (unknown)
// source: evmos/inflation/v1/query.proto

package inflationv1

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
	Query_Period_FullMethodName             = "/helios.inflation.v1.Query/Period"
	Query_EpochMintProvision_FullMethodName = "/helios.inflation.v1.Query/EpochMintProvision"
	Query_SkippedEpochs_FullMethodName      = "/helios.inflation.v1.Query/SkippedEpochs"
	Query_CirculatingSupply_FullMethodName  = "/helios.inflation.v1.Query/CirculatingSupply"
	Query_InflationRate_FullMethodName      = "/helios.inflation.v1.Query/InflationRate"
	Query_Params_FullMethodName             = "/helios.inflation.v1.Query/Params"
)

// QueryClient is the client API for Query service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
type QueryClient interface {
	// Period retrieves current period.
	Period(ctx context.Context, in *QueryPeriodRequest, opts ...grpc.CallOption) (*QueryPeriodResponse, error)
	// EpochMintProvision retrieves current minting epoch provision value.
	EpochMintProvision(ctx context.Context, in *QueryEpochMintProvisionRequest, opts ...grpc.CallOption) (*QueryEpochMintProvisionResponse, error)
	// SkippedEpochs retrieves the total number of skipped epochs.
	SkippedEpochs(ctx context.Context, in *QuerySkippedEpochsRequest, opts ...grpc.CallOption) (*QuerySkippedEpochsResponse, error)
	// CirculatingSupply retrieves the total number of tokens that are in
	// circulation (i.e. excluding unvested tokens).
	CirculatingSupply(ctx context.Context, in *QueryCirculatingSupplyRequest, opts ...grpc.CallOption) (*QueryCirculatingSupplyResponse, error)
	// InflationRate retrieves the inflation rate of the current period.
	InflationRate(ctx context.Context, in *QueryInflationRateRequest, opts ...grpc.CallOption) (*QueryInflationRateResponse, error)
	// Params retrieves the total set of minting parameters.
	Params(ctx context.Context, in *QueryParamsRequest, opts ...grpc.CallOption) (*QueryParamsResponse, error)
}

type queryClient struct {
	cc grpc.ClientConnInterface
}

func NewQueryClient(cc grpc.ClientConnInterface) QueryClient {
	return &queryClient{cc}
}

func (c *queryClient) Period(ctx context.Context, in *QueryPeriodRequest, opts ...grpc.CallOption) (*QueryPeriodResponse, error) {
	out := new(QueryPeriodResponse)
	err := c.cc.Invoke(ctx, Query_Period_FullMethodName, in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *queryClient) EpochMintProvision(ctx context.Context, in *QueryEpochMintProvisionRequest, opts ...grpc.CallOption) (*QueryEpochMintProvisionResponse, error) {
	out := new(QueryEpochMintProvisionResponse)
	err := c.cc.Invoke(ctx, Query_EpochMintProvision_FullMethodName, in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *queryClient) SkippedEpochs(ctx context.Context, in *QuerySkippedEpochsRequest, opts ...grpc.CallOption) (*QuerySkippedEpochsResponse, error) {
	out := new(QuerySkippedEpochsResponse)
	err := c.cc.Invoke(ctx, Query_SkippedEpochs_FullMethodName, in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *queryClient) CirculatingSupply(ctx context.Context, in *QueryCirculatingSupplyRequest, opts ...grpc.CallOption) (*QueryCirculatingSupplyResponse, error) {
	out := new(QueryCirculatingSupplyResponse)
	err := c.cc.Invoke(ctx, Query_CirculatingSupply_FullMethodName, in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *queryClient) InflationRate(ctx context.Context, in *QueryInflationRateRequest, opts ...grpc.CallOption) (*QueryInflationRateResponse, error) {
	out := new(QueryInflationRateResponse)
	err := c.cc.Invoke(ctx, Query_InflationRate_FullMethodName, in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *queryClient) Params(ctx context.Context, in *QueryParamsRequest, opts ...grpc.CallOption) (*QueryParamsResponse, error) {
	out := new(QueryParamsResponse)
	err := c.cc.Invoke(ctx, Query_Params_FullMethodName, in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// QueryServer is the server API for Query service.
// All implementations must embed UnimplementedQueryServer
// for forward compatibility
type QueryServer interface {
	// Period retrieves current period.
	Period(context.Context, *QueryPeriodRequest) (*QueryPeriodResponse, error)
	// EpochMintProvision retrieves current minting epoch provision value.
	EpochMintProvision(context.Context, *QueryEpochMintProvisionRequest) (*QueryEpochMintProvisionResponse, error)
	// SkippedEpochs retrieves the total number of skipped epochs.
	SkippedEpochs(context.Context, *QuerySkippedEpochsRequest) (*QuerySkippedEpochsResponse, error)
	// CirculatingSupply retrieves the total number of tokens that are in
	// circulation (i.e. excluding unvested tokens).
	CirculatingSupply(context.Context, *QueryCirculatingSupplyRequest) (*QueryCirculatingSupplyResponse, error)
	// InflationRate retrieves the inflation rate of the current period.
	InflationRate(context.Context, *QueryInflationRateRequest) (*QueryInflationRateResponse, error)
	// Params retrieves the total set of minting parameters.
	Params(context.Context, *QueryParamsRequest) (*QueryParamsResponse, error)
	mustEmbedUnimplementedQueryServer()
}

// UnimplementedQueryServer must be embedded to have forward compatible implementations.
type UnimplementedQueryServer struct {
}

func (UnimplementedQueryServer) Period(context.Context, *QueryPeriodRequest) (*QueryPeriodResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Period not implemented")
}
func (UnimplementedQueryServer) EpochMintProvision(context.Context, *QueryEpochMintProvisionRequest) (*QueryEpochMintProvisionResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method EpochMintProvision not implemented")
}
func (UnimplementedQueryServer) SkippedEpochs(context.Context, *QuerySkippedEpochsRequest) (*QuerySkippedEpochsResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method SkippedEpochs not implemented")
}
func (UnimplementedQueryServer) CirculatingSupply(context.Context, *QueryCirculatingSupplyRequest) (*QueryCirculatingSupplyResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method CirculatingSupply not implemented")
}
func (UnimplementedQueryServer) InflationRate(context.Context, *QueryInflationRateRequest) (*QueryInflationRateResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method InflationRate not implemented")
}
func (UnimplementedQueryServer) Params(context.Context, *QueryParamsRequest) (*QueryParamsResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Params not implemented")
}
func (UnimplementedQueryServer) mustEmbedUnimplementedQueryServer() {}

// UnsafeQueryServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to QueryServer will
// result in compilation errors.
type UnsafeQueryServer interface {
	mustEmbedUnimplementedQueryServer()
}

func RegisterQueryServer(s grpc.ServiceRegistrar, srv QueryServer) {
	s.RegisterService(&Query_ServiceDesc, srv)
}

func _Query_Period_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(QueryPeriodRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(QueryServer).Period(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: Query_Period_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(QueryServer).Period(ctx, req.(*QueryPeriodRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Query_EpochMintProvision_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(QueryEpochMintProvisionRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(QueryServer).EpochMintProvision(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: Query_EpochMintProvision_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(QueryServer).EpochMintProvision(ctx, req.(*QueryEpochMintProvisionRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Query_SkippedEpochs_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(QuerySkippedEpochsRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(QueryServer).SkippedEpochs(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: Query_SkippedEpochs_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(QueryServer).SkippedEpochs(ctx, req.(*QuerySkippedEpochsRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Query_CirculatingSupply_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(QueryCirculatingSupplyRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(QueryServer).CirculatingSupply(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: Query_CirculatingSupply_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(QueryServer).CirculatingSupply(ctx, req.(*QueryCirculatingSupplyRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Query_InflationRate_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(QueryInflationRateRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(QueryServer).InflationRate(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: Query_InflationRate_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(QueryServer).InflationRate(ctx, req.(*QueryInflationRateRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Query_Params_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(QueryParamsRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(QueryServer).Params(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: Query_Params_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(QueryServer).Params(ctx, req.(*QueryParamsRequest))
	}
	return interceptor(ctx, in, info, handler)
}

// Query_ServiceDesc is the grpc.ServiceDesc for Query service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var Query_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "helios.inflation.v1.Query",
	HandlerType: (*QueryServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "Period",
			Handler:    _Query_Period_Handler,
		},
		{
			MethodName: "EpochMintProvision",
			Handler:    _Query_EpochMintProvision_Handler,
		},
		{
			MethodName: "SkippedEpochs",
			Handler:    _Query_SkippedEpochs_Handler,
		},
		{
			MethodName: "CirculatingSupply",
			Handler:    _Query_CirculatingSupply_Handler,
		},
		{
			MethodName: "InflationRate",
			Handler:    _Query_InflationRate_Handler,
		},
		{
			MethodName: "Params",
			Handler:    _Query_Params_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "evmos/inflation/v1/query.proto",
}
