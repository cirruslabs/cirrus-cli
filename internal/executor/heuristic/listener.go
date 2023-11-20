package heuristic

import (
	"context"
	"fmt"
	"github.com/google/uuid"
	"net"
	"os"
	"path/filepath"
	"runtime"
)

const networkUnix = "unix"

type Listener struct {
	listener net.Listener
}

func NewListener(ctx context.Context, address string, virtualMachine bool) (*Listener, error) {
	network := "tcp"
	var socketDir string

	// Work around host.docker.internal missing on Linux
	//
	// See the following tickets:
	// * https://github.com/docker/for-linux/issues/264
	// * https://github.com/moby/moby/pull/40007
	if runtime.GOOS == "linux" {
		if cloudBuildIP := GetCloudBuildIP(ctx); cloudBuildIP != "" {
			network = "tcp"
			address = fmt.Sprintf("%s:0", cloudBuildIP)
		} else if !virtualMachine {
			socketDir = fmt.Sprintf("/tmp/cli-%s", uuid.New().String())
		}
	} else if runtime.GOOS == "windows" && IsRunningWindowsContainers(ctx) {
		socketDir = fmt.Sprintf("C:\\Windows\\Temp\\cli-%s", uuid.New().String())
	}

	if socketDir != "" {
		if err := os.Mkdir(socketDir, 0700); err != nil {
			return nil, err
		}

		network = networkUnix
		address = filepath.Join(socketDir, "cli.sock")
	}

	lis, err := net.Listen(network, address)
	if err != nil {
		return nil, err
	}

	return &Listener{
		listener: lis,
	}, nil
}

func (lis *Listener) Accept() (net.Conn, error) {
	return lis.listener.Accept()
}

func (lis *Listener) Close() error {
	err := lis.listener.Close()

	if lis.listener.Addr().Network() == networkUnix {
		_ = os.Remove(lis.listener.Addr().String())
	}

	return err
}

func (lis *Listener) Addr() net.Addr {
	return lis.listener.Addr()
}

// ContainerEndpoint returns listener address suitable for use in agent's "-api-endpoint" flag
// when running inside of a container.
func (lis *Listener) ContainerEndpoint() string {
	if lis.listener.Addr().Network() == networkUnix {
		return "unix:" + lis.listener.Addr().String()
	}

	// There's no host.docker.internal on Linux
	if runtime.GOOS == "linux" {
		return fmt.Sprintf("http://%s", lis.listener.Addr().String())
	}

	port := lis.listener.Addr().(*net.TCPAddr).Port

	return fmt.Sprintf("http://host.docker.internal:%d", port)
}

// DirectEndpoint returns listener address suitable for use in agent's "-api-endpoint" flag
// when running on the host.
func (lis *Listener) DirectEndpoint() string {
	if lis.listener.Addr().Network() == networkUnix {
		return "unix://" + lis.listener.Addr().String()
	}

	return "http://" + lis.listener.Addr().String()
}
