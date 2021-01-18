package parallels

import (
	"context"
	"errors"
	"fmt"
	"github.com/avast/retry-go"
	"github.com/cirruslabs/cirrus-cli/internal/executor/instance/runconfig"
	"github.com/cirruslabs/cirrus-cli/internal/executor/platform"
	"golang.org/x/crypto/ssh"
	"net"
	"strconv"
	"strings"
)

var (
	ErrFailed = errors.New("Parallels isolation failed")
)

type Parallels struct {
	vmImage     string
	sshUser     string
	sshPassword string
	agentOS     string
}

func New(vmImage, sshUser, sshPassword, agentOS string) (*Parallels, error) {
	return &Parallels{
		vmImage:     vmImage,
		sshUser:     sshUser,
		sshPassword: sshPassword,
		agentOS:     agentOS,
	}, nil
}

func (parallels *Parallels) Run(ctx context.Context, config *runconfig.RunConfig) (err error) {
	vm, err := NewVMClonedFrom(ctx, parallels.vmImage)
	if err != nil {
		return fmt.Errorf("%w: failed to create VM cloned from %q: %v", ErrFailed, parallels.vmImage, err)
	}
	defer vm.Close()

	// Wait for the VM to start and get it's DHCP address
	var ip string

	if err := retry.Do(func() error {
		ip, err = vm.RetrieveIP(ctx)
		return err
	}); err != nil {
		return fmt.Errorf("%w: failed to retrieve VM %q IP-address: %v", ErrFailed, vm.name, err)
	}

	// Connect to the VM and upload the agent
	cli, err := ssh.Dial("tcp", ip+":22", &ssh.ClientConfig{
		HostKeyCallback: func(hostname string, remote net.Addr, key ssh.PublicKey) error {
			return nil
		},
		User: parallels.sshUser,
		Auth: []ssh.AuthMethod{
			ssh.Password(parallels.sshPassword),
		},
	})
	if err != nil {
		return fmt.Errorf("%w: failed to connect to the VM %q via SSH: %v", ErrFailed, vm.Ident(), err)
	}
	defer cli.Close()

	remoteAgentPath, err := uploadAgent(ctx, cli, parallels.agentOS, config.GetAgentVersion())
	if err != nil {
		return fmt.Errorf("%w: failed to upload agent to the VM %q via SFTP: %v",
			ErrFailed, vm.Ident(), err)
	}

	sess, err := cli.NewSession()
	if err != nil {
		return fmt.Errorf("%w: failed to open SSH session on VM %q: %v", ErrFailed, vm.Ident(), err)
	}

	stdinBuf, _ := sess.StdinPipe()

	// start a login shell so all the customization from ~/.zprofile will be picked up
	err = sess.Shell()
	if err != nil {
		return fmt.Errorf("%w: failed to start a shell on VM %q: %v", ErrFailed, vm.Ident(), err)
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
