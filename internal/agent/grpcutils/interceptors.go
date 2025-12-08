package grpcutils

import (
	"context"

	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

func UnaryMetadataInterceptor(md metadata.MD) grpc.UnaryClientInterceptor {
	return func(
		ctx context.Context,
		method string,
		req, reply interface{},
		cc *grpc.ClientConn,
		invoker grpc.UnaryInvoker,
		opts ...grpc.CallOption,
	) error {
		ctx = metadata.NewOutgoingContext(ctx, md)

		return invoker(ctx, method, req, reply, cc, opts...)
	}
}

func StreamMetadataInterceptor(md metadata.MD) grpc.StreamClientInterceptor {
	return func(
		ctx context.Context,
		desc *grpc.StreamDesc,
		cc *grpc.ClientConn,
		method string,
		streamer grpc.Streamer,
		opts ...grpc.CallOption,
	) (grpc.ClientStream, error) {
		ctx = metadata.NewOutgoingContext(ctx, md)

		return streamer(ctx, desc, cc, method, opts...)
	}
}
