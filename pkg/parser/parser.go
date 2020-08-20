package parser

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"github.com/certifi/gocertifi"
	"github.com/cirruslabs/cirrus-ci-agent/api"
	"github.com/cirruslabs/cirrus-cli/internal/executor/instance"
	grpcretry "github.com/grpc-ecosystem/go-grpc-middleware/retry"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/status"
	"io/ioutil"
	"path/filepath"
	"sync"
	"time"
)

var (
	ErrRPC           = errors.New("RPC error")
	ErrFilesContents = errors.New("failed to retrieve files contents")
)

type Parser struct {
	// RPCEndpoint specifies an alternative RPC endpoint to use. If empty, DefaultRPCEndpoint is used.
	RPCEndpoint string

	// Environment to take into account when expanding variables.
	Environment map[string]string

	// Paths and contents of the files that might influence the parser.
	FilesContents map[string]string
}

const (
	DefaultRPCEndpoint = "grpc.cirrus-ci.com:443"
	defaultTimeout     = time.Second * 5
	defaultRetries     = 3
)

var clientsMutex sync.Mutex
var clientsCache = make(map[string]api.CirrusCIServiceClient)

type Result struct {
	Errors []string
	Tasks  []*api.Task
}

func (p *Parser) rpcEndpoint() string {
	if p.RPCEndpoint == "" {
		return DefaultRPCEndpoint
	}

	return p.RPCEndpoint
}

func getRPCClient(ctx context.Context, endpoint string) (api.CirrusCIServiceClient, error) {
	clientsMutex.Lock()
	defer clientsMutex.Unlock()
	if cachedClient, ok := clientsCache[endpoint]; ok {
		return cachedClient, nil
	}

	// Setup Cirrus CI RPC connection
	certPool, _ := gocertifi.CACerts()
	tlsCredentials := credentials.NewTLS(&tls.Config{
		MinVersion: tls.VersionTLS13,
		RootCAs:    certPool,
	})
	conn, err := grpc.DialContext(
		ctx,
		endpoint,
		grpc.WithBlock(),
		grpc.WithTransportCredentials(tlsCredentials),
		grpc.WithUnaryInterceptor(
			grpcretry.UnaryClientInterceptor(
				grpcretry.WithMax(defaultRetries),
			),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("%w: failed to dial with '%v'", ErrRPC, err)
	}

	// Send config for parsing by the Cirrus CI RPC and retrieve results
	client := api.NewCirrusCIServiceClient(conn)
	clientsCache[endpoint] = client
	return client, nil
}

func (p *Parser) Parse(config string) (*Result, error) {
	// Create a context to enforce the defaultTimeout
	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()
	client, err := getRPCClient(ctx, p.rpcEndpoint())
	if err != nil {
		return nil, err
	}

	if p.Environment == nil {
		p.Environment = make(map[string]string)
	}

	if p.FilesContents == nil {
		p.FilesContents = make(map[string]string)
	}

	request := api.ParseConfigRequest{
		Config:        config,
		Environment:   p.Environment,
		FilesContents: p.FilesContents,
	}

	response, err := client.ParseConfig(ctx, &request)
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

	return &Result{Tasks: response.Tasks}, nil
}

func (p *Parser) ParseFromFile(path string) (*Result, error) {
	config, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}

	result, err := p.Parse(string(config))
	if err != nil || len(result.Errors) != 0 {
		return result, err
	}

	// Get the contents of files that might influence the parser results
	//
	// For example, when using Dockerfile as CI environment feature[1], the unique hash of the container
	// image is calculated from the file specified in the "dockerfile" field.
	//
	// [1]: https://cirrus-ci.org/guide/docker-builder-vm/#dockerfile-as-a-ci-environment
	filesContents := make(map[string]string)
	for _, task := range result.Tasks {
		inst, err := instance.NewFromProto(task.Instance, []*api.Command{})
		if err != nil {
			continue
		}
		prebuilt, ok := inst.(*instance.PrebuiltInstance)
		if !ok {
			continue
		}
		contents, err := ioutil.ReadFile(filepath.Join(filepath.Dir(path), prebuilt.Dockerfile))
		if err != nil {
			return nil, fmt.Errorf("%w: %v", ErrFilesContents, err)
		}
		filesContents[prebuilt.Dockerfile] = string(contents)
	}

	// Short-circuit if we've found no special files
	if len(filesContents) == 0 {
		return result, nil
	}

	// Parse again with the file contents supplied
	p.FilesContents = filesContents
	return p.Parse(string(config))
}
