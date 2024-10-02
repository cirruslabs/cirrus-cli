package upstream

import (
	"context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"time"
)

// PerRPCCredentials interface implementation.
func (upstream *Upstream) GetRequestMetadata(ctx context.Context, uri ...string) (map[string]string, error) {
	result := map[string]string{
		"registration-token": upstream.registrationToken,
		"session-token":      upstream.sessionToken,
		"worker-name":        upstream.workerName,
	}
	// inherit any outgoing metadata
	md, ok := metadata.FromOutgoingContext(ctx)
	if ok {
		for key, values := range md {
			if len(values) > 0 {
				result[key] = values[0]
			}
		}
	}
	return result, nil
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
