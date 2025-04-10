package chacha

import (
	"fmt"
	"os"
)

type Chacha struct {
	addr       string
	cert       string
	enableTart bool

	certFile *os.File
}

func New(addr string, cert string, enableTart bool) (*Chacha, error) {
	caCertFile, err := os.CreateTemp("", "cirrus-cli-chacha-ca-cert-*")
	if err != nil {
		return nil, fmt.Errorf("failed to initialize Chacha: "+
			"failed to create a temporary cert file: %w", err)
	}
	if _, err := caCertFile.WriteString(cert); err != nil {
		return nil, fmt.Errorf("failed to initialize Chacha: "+
			"failed to write to a temporary cert file: %w", err)
	}
	if err := caCertFile.Close(); err != nil {
		return nil, fmt.Errorf("failed to initialize Chacha: "+
			"failed to close the temporary cert file: %w", err)
	}

	return &Chacha{
		addr:       addr,
		cert:       cert,
		enableTart: enableTart,
		certFile:   caCertFile,
	}, nil
}

func (chacha *Chacha) Addr() string {
	return chacha.addr
}

func (chacha *Chacha) Cert() string {
	return chacha.cert
}

func (chacha *Chacha) EnableTart() bool {
	return chacha.enableTart
}

func (chacha *Chacha) CertPath() string {
	return chacha.certFile.Name()
}

func (chacha *Chacha) AgentEnvironmentVariables() map[string]string {
	return map[string]string{
		"CIRRUS_CHACHA_ADDR": chacha.addr,
		"CIRRUS_CHACHA_CERT": chacha.cert,
	}
}

func (chacha *Chacha) Close() error {
	return os.Remove(chacha.certFile.Name())
}
