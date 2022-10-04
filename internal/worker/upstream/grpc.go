package upstream

import (
	"context"
	"google.golang.org/grpc"
	"time"
)

// PerRPCCredentials interface implementation.
func (upstream *Upstream) GetRequestMetadata(ctx context.Context, uri ...string) (map[string]string, error) {
	return map[string]string{
		"registration-token": upstream.registrationToken,
		"session-token":      upstream.sessionToken,
		"worker-name":        upstream.workerName,
	}, nil
}

// PerRPCCredentials interface implementation.
func (upstream *Upstream) RequireTransportSecurity() bool {
	return !upstream.rpcInsecure
}

func deadlineUnaryInterceptor(duration time.Duration) grpc.UnaryClientInterceptor {
	return func(
		ctx context.Context,
		method string,
		req,
		reply interface{},
		cc *grpc.ClientConn,
		invoker grpc.UnaryInvoker,
		opts ...grpc.CallOption,
	) error {
		ctxWithDeadline, cancel := context.WithDeadline(ctx, time.Now().Add(duration))
		defer cancel()

		return invoker(ctxWithDeadline, method, req, reply, cc, opts...)
	}
}
