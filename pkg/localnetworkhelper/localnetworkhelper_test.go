package localnetworkhelper_test

import (
	"bytes"
	"context"
	cryptrand "crypto/rand"
	"github.com/cirruslabs/cirrus-cli/pkg/localnetworkhelper"
	"github.com/stretchr/testify/require"
	"io"
	"net"
	"os"
	"testing"
)

func TestMain(m *testing.M) {
	if len(os.Args) == 2 && os.Args[1] == localnetworkhelper.CommandName {
		_ = localnetworkhelper.Serve(context.Background(), 3)

		return
	}

	m.Run()
}

func TestLocalNetworkHelper(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start the local network helper process and connect to it
	require.NoError(t, localnetworkhelper.StartAndConnect(ctx))

	// Verify that the SSH server in the local network helper process works
	// by establishing a regular TCP connection through that SSH server
	// to our fixture TCP server and reading a chunk of bytes from it
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)

	// Send d ata
	sent := make([]byte, 1*1024*1024)
	_, err = cryptrand.Read(sent)
	require.NoError(t, err)

	go func() {
		conn, err := lis.Accept()
		require.NoError(t, err)

		_, err = io.Copy(conn, bytes.NewReader(sent))
		require.NoError(t, err)

		require.NoError(t, conn.Close())
	}()

	// Receive data
	conn, err := localnetworkhelper.SSHClient.DialContext(ctx, "tcp", lis.Addr().String())
	require.NoError(t, err)

	received, err := io.ReadAll(conn)
	require.NoError(t, err)
	require.EqualValues(t, sent, received)
}
