package rpcparser_test

import (
	"errors"
	"github.com/cirruslabs/cirrus-ci-agent/api"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"log"
	"net"
	"path/filepath"
	"testing"

	"github.com/cirruslabs/cirrus-cli/pkg/rpcparser"
	"github.com/stretchr/testify/assert"
)

var validCases = []string{
	"example-android.yml",
	"example-flutter-web.yml",
	"example-mysql.yml",
	"example-rust.yml",
}

var invalidCases = []string{
	"invalid-empty.yml",
}

func absolutize(file string) string {
	return filepath.Join("testdata", file)
}

func TestValidConfigs(t *testing.T) {
	for _, validCase := range validCases {
		file := validCase
		t.Run(file, func(t *testing.T) {
			p := rpcparser.Parser{}
			_, err := p.ParseFromFile(absolutize(file))
			require.NoError(t, err)
		})
	}
}

func TestInvalidConfigs(t *testing.T) {
	for _, invalidCase := range invalidCases {
		file := invalidCase
		t.Run(file, func(t *testing.T) {
			p := rpcparser.Parser{}
			_, err := p.ParseFromFile(absolutize(file))
			require.Error(t, err)
		})
	}
}

// TestErrTransport ensures that network-related errors result in ErrRPC.
func TestErrRPC(t *testing.T) {
	p := rpcparser.Parser{RPCEndpoint: "api.invalid:443"}
	result, err := p.Parse("a: b")

	assert.Nil(t, result)
	assert.True(t, errors.Is(err, rpcparser.ErrRPC))
}

// TestErrInternal ensures that RPC errors other than grpc.codes.InvalidArgument result in ErrRPC.
func TestErrInternal(t *testing.T) {
	// Create a gRPC server that returns grpc.codes.Unimplemented to all of it's method calls
	listener, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		t.Fatal(err)
	}

	server := grpc.NewServer()
	api.RegisterCirrusCIServiceServer(server, &api.UnimplementedCirrusCIServiceServer{})
	go func() {
		if err := server.Serve(listener); err != nil {
			log.Fatal(err)
		}
	}()
	defer server.Stop()

	p := rpcparser.Parser{RPCEndpoint: listener.Addr().String()}
	result, err := p.Parse("a: b")

	assert.Nil(t, result)
	assert.True(t, errors.Is(err, rpcparser.ErrRPC))
}
