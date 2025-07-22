package containerbackend

import (
	"fmt"
	"github.com/avast/retry-go/v4"
	"github.com/google/uuid"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"time"
)

type Podman struct {
	cmd *exec.Cmd

	*Docker
}

func NewPodman() (*Podman, error) {
	socketPath := filepath.Join(os.TempDir(), fmt.Sprintf("podman-%s.sock", uuid.New().String()))
	socketURI := fmt.Sprintf("unix://%s", socketPath)

	cmd := exec.Command("podman", "system", "service", "-t", "0", socketURI)

	// Prevent the signals sent to the CLI from reaching the Podman process
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setpgid: true,
	}

	if err := cmd.Start(); err != nil {
		return nil, err
	}

	err := retry.Do(func() error {
		_, err := os.Stat(socketPath)
		return err
	})
	if err != nil {
		return nil, err
	}

	docker, err := NewDocker(socketURI)
	if err != nil {
		return nil, err
	}

	podman := &Podman{
		cmd:    cmd,
		Docker: docker,
	}

	return podman, nil
}

func (backend *Podman) Close() error {
	_ = backend.Docker.Close()

	doneChan := make(chan error)

	go func() {
		doneChan <- backend.cmd.Wait()
	}()

	var interruptSent, killSent bool

	for {
		select {
		case <-time.After(time.Second):
			if !killSent {
				if err := backend.cmd.Process.Kill(); err != nil {
					return err
				}
				killSent = true
			}
		case err := <-doneChan:
			return err
		default:
			if !interruptSent {
				if err := backend.cmd.Process.Signal(syscall.SIGTERM); err != nil {
					return err
				}
				interruptSent = true
			}
		}
	}
}
