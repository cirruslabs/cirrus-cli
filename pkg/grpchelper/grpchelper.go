package grpchelper

import (
	"crypto/tls"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	insecurepkg "google.golang.org/grpc/credentials/insecure"
	"strings"
)

func TransportSettings(apiEndpoint string) (string, bool) {
	// HTTP is always insecure
	if strings.HasPrefix(apiEndpoint, "http://") {
		return strings.TrimPrefix(apiEndpoint, "http://"), true
	}

	// Unix domain sockets are always insecure
	if strings.HasPrefix(apiEndpoint, "unix:") {
		return apiEndpoint, true
	}

	// HTTPS and other cases are always secure
	return strings.TrimPrefix(apiEndpoint, "https://"), false
}

func TransportSettingsAsDialOption(apiEndpoint string) (string, grpc.DialOption) {
	target, insecure := TransportSettings(apiEndpoint)
	if insecure {
		return target, grpc.WithTransportCredentials(insecurepkg.NewCredentials())
	}

	// Use embedded root certificates because the agent can be executed in a distroless container
	// and don't check for error, since then the default certificates from the host will be used
	tlsCredentials := credentials.NewTLS(&tls.Config{
		MinVersion: tls.VersionTLS13,
	})

	return target, grpc.WithTransportCredentials(tlsCredentials)
}
