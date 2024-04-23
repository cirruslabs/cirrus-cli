package terminalwrapper

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestTrustedSecretIsLargeEnough(t *testing.T) {
	const trustedSecretHexadecimalStringLength = 64

	trustedSecret, err := generateTrustedSecret()
	if err != nil {
		t.Fatal(err)
	}

	assert.Len(t, trustedSecret, trustedSecretHexadecimalStringLength)
}
