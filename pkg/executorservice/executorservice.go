package executorservice

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"github.com/cirruslabs/cirrus-ci-agent/api"
	grpcretry "github.com/grpc-ecosystem/go-grpc-middleware/retry"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"time"
)

var (
	ErrRPC = errors.New("RPC error")
)

type ExecutorService struct{}

const (
	DefaultRPCEndpoint = "grpc.cirrus-ci.com:443"
	defaultTimeout     = time.Second * 15
	defaultRetries     = 3
)

func New() *ExecutorService {
	return &ExecutorService{}
}

func (p *ExecutorService) SupportedInstances() (*api.AdditionalInstancesInfo, error) {
	// Create a context to enforce the defaultTimeout
	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()

	// Setup Cirrus CI RPC connection
	tlsCredentials := credentials.NewTLS(&tls.Config{
		MinVersion: tls.VersionTLS13,
	})
	conn, err := grpc.DialContext(
		ctx,
		DefaultRPCEndpoint,
		grpc.WithBlock(),
		grpc.WithTransportCredentials(tlsCredentials),
		grpc.WithUnaryInterceptor(
			grpcretry.UnaryClientInterceptor(
				grpcretry.WithMax(defaultRetries),
			),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("%w: failed to dial: %v", ErrRPC, err)
	}
	defer conn.Close()

	client := api.NewCirrusRemoteExecutorServiceClient(conn)

	response, err := client.Capabilities(ctx, &api.CapabilitiesRequest{})
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrRPC, err)
	}

	return response.SupportedInstances, nil
}
