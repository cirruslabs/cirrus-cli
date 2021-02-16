package parallels

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"github.com/avast/retry-go"
	"github.com/cirruslabs/cirrus-cli/internal/executor/instance/runconfig"
	"github.com/cirruslabs/cirrus-cli/internal/executor/platform"
	"github.com/cirruslabs/cirrus-cli/internal/logger"
	"golang.org/x/crypto/ssh"
	"io"
	"net"
	"strconv"
	"strings"
	"time"
)

var (
	ErrFailed = errors.New("Parallels isolation failed")
)

type Parallels struct {
	logger      logger.Lightweight
	vmImage     string
	sshUser     string
	sshPassword string
	agentOS     string
}

func New(vmImage, sshUser, sshPassword, agentOS string, opts ...Option) (*Parallels, error) {
	parallels := &Parallels{
		vmImage:     vmImage,
		sshUser:     sshUser,
		sshPassword: sshPassword,
		agentOS:     agentOS,
	}

	// Apply options
	for _, opt := range opts {
		opt(parallels)
	}

	// Apply default options (to cover those that weren't specified)
	if parallels.logger == nil {
		parallels.logger = &logger.LightweightStub{}
	}

	return parallels, nil
}

func (parallels *Parallels) Run(ctx context.Context, config *runconfig.RunConfig) (err error) {
	vm, err := NewVMClonedFrom(ctx, parallels.vmImage)
	if err != nil {
		return fmt.Errorf("%w: failed to create VM cloned from %q: %v", ErrFailed, parallels.vmImage, err)
	}
	defer vm.Close()

	if err := vm.Start(ctx); err != nil {
		return fmt.Errorf("%w: failed to start VM %q: %v", ErrFailed, vm.Ident(), err)
	}

	// Wait for the VM to start and get it's DHCP address
	var ip string

	if err := retry.Do(func() error {
		ip, err = vm.RetrieveIP(ctx)
		return err
	}, retry.Context(ctx), retry.RetryIf(func(err error) bool {
		return errors.Is(err, ErrDHCPSnoopFailed)
	})); err != nil {
		return fmt.Errorf("%w: failed to retrieve VM %q IP-address: %v", ErrFailed, vm.name, err)
	}

	// Connect to the VM and upload the agent
	var netConn net.Conn
	addr := ip + ":22"

	if err := retry.Do(func() error {
		dialer := net.Dialer{}

		netConn, err = dialer.DialContext(ctx, "tcp", addr)

		return err
	}, retry.Context(ctx)); err != nil {
		return fmt.Errorf("%w: failed to connect to the VM %q on SSH port: %v", ErrFailed, vm.Ident(), err)
	}

	sshConfig := &ssh.ClientConfig{
		HostKeyCallback: func(hostname string, remote net.Addr, key ssh.PublicKey) error {
			return nil
		},
		User: parallels.sshUser,
		Auth: []ssh.AuthMethod{
			ssh.Password(parallels.sshPassword),
		},
	}

	sshConn, chans, reqs, err := ssh.NewClientConn(netConn, addr, sshConfig)
	if err != nil {
		return fmt.Errorf("%w: failed to connect to the VM %q via SSH: %v", ErrFailed, vm.Ident(), err)
	}

	cli := ssh.NewClient(sshConn, chans, reqs)

	// Work around x/crypto/ssh not being context.Context-friendly (e.g. https://github.com/golang/go/issues/20288)
	monitorCtx, monitorCancel := context.WithCancel(ctx)
	go func() {
		<-monitorCtx.Done()
		_ = cli.Close()
	}()
	defer monitorCancel()

	remoteAgentPath, err := uploadAgent(ctx, cli, parallels.agentOS, config.GetAgentVersion())
	if err != nil {
		return fmt.Errorf("%w: failed to upload agent to the VM %q via SFTP: %v",
			ErrFailed, vm.Ident(), err)
	}

	sess, err := cli.NewSession()
	if err != nil {
		return fmt.Errorf("%w: failed to open SSH session on VM %q: %v", ErrFailed, vm.Ident(), err)
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
			parallels.logger.Debugf("VM: %s", scanner.Text())
		}
	}()

	stdinBuf, err := sess.StdinPipe()
	if err != nil {
		return fmt.Errorf("%w: while opening stdin pipe: %v", ErrFailed, err)
	}

	// start a login shell so all the customization from ~/.zprofile will be picked up
	err = sess.Shell()
	if err != nil {
		return fmt.Errorf("%w: failed to start a shell on VM %q: %v", ErrFailed, vm.Ident(), err)
	}

	// Synchronize time for suspended VMs
	if vm.ClonedFromSuspended() {
		_, err = stdinBuf.Write([]byte(TimeSyncCommand(time.Now().UTC())))
		if err != nil {
			return fmt.Errorf("%w: failed to sync time on VM %q: %v", ErrFailed, vm.Ident(), err)
		}
	}

	command := []string{
		remoteAgentPath,
		"-api-endpoint",
		config.DirectEndpoint,
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
		return fmt.Errorf("%w: failed to start agent on VM %q: %v", ErrFailed, vm.Ident(), err)
	}
	err = sess.Wait()
	if err != nil {
		return fmt.Errorf("%w: failed to run agent on VM %q: %v", ErrFailed, vm.Ident(), err)
	}

	return nil
}

func (parallels *Parallels) WorkingDirectory(projectDir string, dirtyMode bool) string {
	return platform.NewUnix().WorkingVolumeMountpoint() + platform.WorkingVolumeWorkingDir
}

func TimeSyncCommand(t time.Time) string {
	return fmt.Sprintf("sudo date -u %s\n", t.Format("010215042006"))
}
