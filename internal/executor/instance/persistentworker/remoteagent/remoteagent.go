package remoteagent

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"github.com/avast/retry-go"
	"github.com/cirruslabs/cirrus-cli/internal/executor/endpoint"
	"github.com/cirruslabs/cirrus-cli/internal/executor/instance/runconfig"
	"github.com/cirruslabs/cirrus-cli/internal/logger"
	"golang.org/x/crypto/ssh"
	"io"
	"net"
	"strconv"
	"strings"
	"time"
)

var ErrFailed = errors.New("remote agent failed")

func WaitForAgent(
	ctx context.Context,
	logger logger.Lightweight,
	ip string,
	sshUser string,
	sshPassword string,
	agentOS string,
	agentArchitecture string,
	config *runconfig.RunConfig,
	synchronizeTime bool,
) error {
	// Connect to the VM and upload the agent
	var netConn net.Conn
	var err error

	addr := ip + ":22"

	if err := retry.Do(func() error {
		dialer := net.Dialer{}

		netConn, err = dialer.DialContext(ctx, "tcp", addr)

		return err
	}, retry.Context(ctx)); err != nil {
		return fmt.Errorf("%w: failed to connect via SSH: %v", ErrFailed, err)
	}

	sshConfig := &ssh.ClientConfig{
		HostKeyCallback: func(hostname string, remote net.Addr, key ssh.PublicKey) error {
			return nil
		},
		User: sshUser,
		Auth: []ssh.AuthMethod{
			ssh.Password(sshPassword),
		},
	}

	sshConn, chans, reqs, err := ssh.NewClientConn(netConn, addr, sshConfig)
	if err != nil {
		return fmt.Errorf("%w: failed to connect via SSH: %v", ErrFailed, err)
	}

	cli := ssh.NewClient(sshConn, chans, reqs)

	// Work around x/crypto/ssh not being context.Context-friendly (e.g. https://github.com/golang/go/issues/20288)
	monitorCtx, monitorCancel := context.WithCancel(ctx)
	go func() {
		<-monitorCtx.Done()
		_ = cli.Close()
	}()
	defer monitorCancel()

	remoteAgentPath, err := uploadAgent(ctx, cli, agentOS, config.GetAgentVersion(), agentArchitecture)
	if err != nil {
		return fmt.Errorf("%w: failed to upload agent via SFTP: %v",
			ErrFailed, err)
	}

	sess, err := cli.NewSession()
	if err != nil {
		return fmt.Errorf("%w: failed to open SSH session: %v", ErrFailed, err)
	}

	// Log output from the virtual machine
	stdout, err := sess.StdoutPipe()
	if err != nil {
		return fmt.Errorf("%w: while opening stdout pipe: %v", ErrFailed, err)
	}
	stderr, err := sess.StderrPipe()
	if err != nil {
		return fmt.Errorf("%w: while opening stderr pipe: %v", ErrFailed, err)
	}
	go func() {
		output := io.MultiReader(stdout, stderr)

		scanner := bufio.NewScanner(output)

		for scanner.Scan() {
			logger.Debugf("VM: %s", scanner.Text())
		}
	}()

	stdinBuf, err := sess.StdinPipe()
	if err != nil {
		return fmt.Errorf("%w: while opening stdin pipe: %v", ErrFailed, err)
	}

	// start a login shell so all the customization from ~/.zprofile will be picked up
	err = sess.Shell()
	if err != nil {
		return fmt.Errorf("%w: failed to start a shell: %v", ErrFailed, err)
	}

	// Synchronize time for suspended VMs
	if synchronizeTime {
		_, err = stdinBuf.Write([]byte(TimeSyncCommand(time.Now().UTC())))
		if err != nil {
			return fmt.Errorf("%w: failed to sync time: %v", ErrFailed, err)
		}
	}

	var apiEndpoint string

	switch config.Endpoint.(type) {
	case *endpoint.Local:
		apiEndpoint, err = forwardViaSSH(cli, logger, config.Endpoint.Direct())
		if err != nil {
			return err
		}
	default:
		apiEndpoint = config.Endpoint.Direct()
	}

	command := []string{
		remoteAgentPath,
		"-api-endpoint",
		apiEndpoint,
		"-server-token",
		"\"" + config.ServerSecret + "\"",
		"-client-token",
		"\"" + config.ClientSecret + "\"",
		"-task-id",
		strconv.FormatInt(config.TaskID, 10),
	}

	// Start the agent and wait for it to terminate
	_, err = stdinBuf.Write([]byte(strings.Join(command, " ") + "\nexit\n"))
	if err != nil {
		return fmt.Errorf("%w: failed to start agent: %v", ErrFailed, err)
	}
	err = sess.Wait()
	if err != nil {
		// Work around x/crypto/ssh not being context.Context-friendly (e.g. https://github.com/golang/go/issues/20288)
		if err := monitorCtx.Err(); err != nil {
			return err
		}

		return fmt.Errorf("%w: failed to run agent: %v", ErrFailed, err)
	}

	return nil
}

func forwardViaSSH(cli *ssh.Client, logger logger.Lightweight, endpoint string) (string, error) {
	// Expose local RPC service to the VM via SSH port forwarding
	vmListener, err := cli.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return "", fmt.Errorf("%w: failed to set-up SSH port forwarding: %v", ErrFailed, err)
	}

	go func() {
		for {
			vmConn, err := vmListener.Accept()
			if err != nil {
				logger.Debugf("failed to accept connection from the VM on forwarded port: %v", err)
				continue
			}

			// Convert endpoint address to host
			host := strings.TrimPrefix(endpoint, "http://")

			localConn, err := net.Dial("tcp", host)
			if err != nil {
				logger.Debugf("failed to connect to the RPC service on %s: %v", endpoint, err)
				continue
			}

			go func() {
				_, _ = io.Copy(vmConn, localConn)
			}()
			go func() {
				_, _ = io.Copy(localConn, vmConn)
			}()
		}
	}()

	return "http://" + vmListener.Addr().String(), nil
}
