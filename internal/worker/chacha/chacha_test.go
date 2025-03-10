package chacha_test

import (
	"github.com/cirruslabs/cirrus-cli/internal/worker/chacha"
	"github.com/stretchr/testify/require"
	"os"
	"testing"
)

func TestChacha(t *testing.T) {
	addr := "1.2.3.4:12345"
	cert := "-----BEGIN CERTIFICATE-----\n...\n-----END CERTIFICATE-----"

	chacha, err := chacha.New(addr, cert)
	require.NoError(t, err)

	require.Equal(t, addr, chacha.Addr())

	certBytes, err := os.ReadFile(chacha.CertPath())
	require.NoError(t, err)
	require.Equal(t, cert, string(certBytes))

	require.NoError(t, chacha.Close())
}
