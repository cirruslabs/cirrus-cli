package parser

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"github.com/cirruslabs/cirrus-ci-agent/api"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/status"
	"io/ioutil"
	"time"
)

var (
	ErrRPC = errors.New("RPC error")
)

type Parser struct {
	// RPCEndpoint specifies an alternative RPC endpoint to use. If empty, DefaultRPCEndpoint is used.
	RPCEndpoint string
}

const (
	DefaultRPCEndpoint = "grpc.cirrus-ci.com:443"
	defaultTimeout     = time.Second * 5
)

type Result struct {
	Errors []string
}

func (p *Parser) rpcEndpoint() string {
	if p.RPCEndpoint == "" {
		return DefaultRPCEndpoint
	}

	return p.RPCEndpoint
}

func (p *Parser) Parse(config string) (*Result, error) {
	// Create a context to enforce the defaultTimeout
	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()

	// Setup Cirrus CI RPC connection
	tlsCredentials := credentials.NewTLS(&tls.Config{
		MinVersion: tls.VersionTLS13,
	})
	conn, err := grpc.DialContext(ctx, p.rpcEndpoint(), grpc.WithTransportCredentials(tlsCredentials))
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrRPC, err)
	}
	defer conn.Close()

	// Send config for parsing by the Cirrus CI RPC and retrieve results
	client := api.NewCirrusCIServiceClient(conn)

	request := api.ParseConfigRequest{
		Config: config,
	}

	_, err = client.ParseConfig(ctx, &request)
	if err != nil {
		s := status.Convert(err)

		switch s.Code() {
		case codes.InvalidArgument:
			// The configuration that we sent is invalid
			return &Result{Errors: []string{s.Message()}}, nil
		default:
			// Unexpected error
			return nil, fmt.Errorf("%w: %v", ErrRPC, err)
		}
	}

	return &Result{}, nil
}

func (p *Parser) ParseFromFile(path string) (*Result, error) {
	config, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}

	return p.Parse(string(config))
}
