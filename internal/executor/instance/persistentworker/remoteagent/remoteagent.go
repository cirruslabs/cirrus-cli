package remoteagent

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"github.com/avast/retry-go/v4"
	"github.com/cirruslabs/cirrus-cli/internal/executor/endpoint"
	"github.com/cirruslabs/cirrus-cli/internal/executor/instance/runconfig"
	"github.com/cirruslabs/cirrus-cli/internal/logger"
	"go.opentelemetry.io/otel"
	"golang.org/x/crypto/ssh"
	"io"
	"net"
	"strconv"
	"strings"
	"time"
)

var (
	ErrFailed = errors.New("remote agent failed")

	tracer = otel.Tracer("remoteagent")
)

type WaitForAgentHook func(ctx context.Context, sshClient *ssh.Client) error

type WaitForAgentHooks []WaitForAgentHook

func (hooks WaitForAgentHooks) Run(ctx context.Context, cli *ssh.Client) error {
	ctx, span := tracer.Start(ctx, "run-hooks")
	defer span.End()

	for _, hook := range hooks {
		if err := hook(ctx, cli); err != nil {
			return err
		}
	}

	return nil
}

func WaitForAgent(
	ctx context.Context,
	logger logger.Lightweight,
	addr string,
	sshUser string,
	sshPassword string,
	agentOS string,
	agentArchitecture string,
	config *runconfig.RunConfig,
	synchronizeTime bool,
	initializeHooks WaitForAgentHooks,
	terminateHooks WaitForAgentHooks,
	preCreatedWorkingDir string,
	env map[string]string,
) error {
	ctx, span := tracer.Start(ctx, "upload-and-wait-for-agent")
	defer span.End()

	cli, err := connectViaSSH(ctx, logger, addr, sshUser, sshPassword)
	if err != nil {
		return err
	}

	// Work around x/crypto/ssh not being context.Context-friendly (e.g. https://github.com/golang/go/issues/20288)
	monitorCtx, monitorCancel := context.WithCancel(ctx)
	go func() {
		<-monitorCtx.Done()
		_ = cli.Close()
	}()
	defer monitorCancel()

	logger.Debugf("running initialization hooks on %s...", addr)

	if err := initializeHooks.Run(ctx, cli); err != nil {
		return err
	}

	logger.Debugf("uploading agent to %s...", addr)

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

	for key, value := range env {
		_, err = stdinBuf.Write([]byte(fmt.Sprintf("export %s=%s\n", key, value)))
		if err != nil {
			return fmt.Errorf("%w: failed set env variable %s: %v", ErrFailed, key, err)
		}
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
		// Expose local RPC service to the VM via SSH port forwarding
		vmListener, err := cli.Listen("tcp", "127.0.0.1:0")
		if err != nil {
			return fmt.Errorf("%w: failed to set-up SSH port forwarding: %v", ErrFailed, err)
		}
		defer vmListener.Close()

		go forwardViaSSH(vmListener, logger, config.Endpoint.Direct())

		apiEndpoint = "http://" + vmListener.Addr().String()
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

	if preCreatedWorkingDir != "" {
		command = append(command, "-pre-created-working-dir", "\""+preCreatedWorkingDir+"\"")
	}

	// Start the agent and wait for it to terminate
	logger.Debugf("running agent on %s with arguments: %v...", addr, command)

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

	logger.Debugf("running termination hooks on %s...", addr)

	if err := terminateHooks.Run(ctx, cli); err != nil {
		return err
	}

	return nil
}

func forwardViaSSH(vmListener net.Listener, logger logger.Lightweight, endpoint string) {
	for {
		vmConn, err := vmListener.Accept()
		if err != nil {
			if errors.Is(err, io.EOF) {
				return
			}

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
			defer vmConn.Close()
			defer localConn.Close()
			_, _ = io.Copy(vmConn, localConn)
		}()
		go func() {
			defer vmConn.Close()
			defer localConn.Close()
			_, _ = io.Copy(localConn, vmConn)
		}()
	}
}

func connectViaSSH(
	ctx context.Context,
	logger logger.Lightweight,
	addr string,
	sshUser string,
	sshPassword string,
) (*ssh.Client, error) {
	ctx, span := tracer.Start(ctx, "connect-via-ssh")
	defer span.End()

	// Connect to the VM and upload the agent

	logger.Debugf("connecting via SSH to %s...", addr)

	sshClient, err := WaitForSSH(ctx, addr, sshUser, sshPassword, logger)
	if err != nil {
		return nil, err
	}

	logger.Debugf("creating new SSH client...")

	return sshClient, nil
}

func WaitForSSH(
	ctx context.Context,
	addr string,
	sshUser string,
	sshPassword string,
	logger logger.Lightweight,
) (*ssh.Client, error) {
	var sshConn ssh.Conn
	var chans <-chan ssh.NewChannel
	var reqs <-chan *ssh.Request

	if err := retry.Do(func() error {
		dialer := net.Dialer{
			Timeout: time.Second,
		}

		netConn, err := dialer.DialContext(ctx, "tcp", addr)
		if err != nil {
			logger.Debugf("failed to dial %s: %v", addr, err)

			return err
		}

		logger.Debugf("successfully dialed %s, performing SSH handshake...", addr)

		sshConfig := &ssh.ClientConfig{
			HostKeyCallback: func(hostname string, remote net.Addr, key ssh.PublicKey) error {
				return nil
			},
			User: sshUser,
			Auth: []ssh.AuthMethod{
				ssh.Password(sshPassword),
			},
			Timeout: time.Second,
		}

		sshConn, chans, reqs, err = ssh.NewClientConn(netConn, addr, sshConfig)
		if err != nil {
			err := fmt.Errorf("%w: failed to connect via SSH: %v", ErrFailed, err)

			logger.Debugf("failed to perform SSH handshake with %s: %v", addr, err)

			return err
		}

		return nil
	}, retry.Context(ctx),
		retry.Attempts(0),
		retry.DelayType(retry.FixedDelay),
		retry.Delay(time.Second),
	); err != nil {
		return nil, fmt.Errorf("%w: failed to connect via SSH: %v", ErrFailed, err)
	}

	return ssh.NewClient(sshConn, chans, reqs), nil
}
