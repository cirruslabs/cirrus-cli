package parallels

import (
	"context"
	"github.com/cirruslabs/cirrus-cli/internal/executor/agent"
	"github.com/cirruslabs/cirrus-cli/internal/executor/platform"
	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
	"io"
	"os"
	"path"
)

func uploadAgent(ctx context.Context, cli *ssh.Client, agentOS, agentVersion string) (string, error) {
	sftpCli, err := sftp.NewClient(cli)
	if err != nil {
		return "", err
	}
	defer sftpCli.Close()

	// Ensure working directory exists
	if err := sftpCli.MkdirAll(platform.NewUnix().CirrusDir()); err != nil {
		return "", err
	}

	// Open agent's binary locally
	localAgentPath, err := agent.RetrieveBinary(ctx, agentVersion, agentOS, "amd64")
	if err != nil {
		return "", err
	}
	localAgentFile, err := os.Open(localAgentPath)
	if err != nil {
		return "", err
	}

	// Create agent's binary remotely
	remoteAgentPath := path.Join(platform.NewUnix().CirrusDir(), "cirrus-ci-agent")
	remoteAgentFile, err := sftpCli.Create(remoteAgentPath)
	if err != nil {
		return "", err
	}

	// Copy
	if _, err := io.Copy(remoteAgentFile, localAgentFile); err != nil {
		return "", err
	}

	// Close and flush
	if err := remoteAgentFile.Close(); err != nil {
		return "", err
	}
	if err := localAgentFile.Close(); err != nil {
		return "", err
	}

	// Agent binary should be executable
	if err := sftpCli.Chmod(remoteAgentPath, 0700); err != nil {
		return "", err
	}

	return remoteAgentPath, nil
}
