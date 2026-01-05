package tart

import (
	"bytes"
	"context"
	"fmt"

	"github.com/cirruslabs/cirrus-cli/internal/executor/instance/persistentworker/remoteagent"
	"github.com/cirruslabs/echelon"
	"golang.org/x/crypto/ssh"
)

func mountWorkingDirectoryHook(tag string, logger *echelon.Logger) remoteagent.WaitForAgentHook {
	return func(ctx context.Context, sshClient *ssh.Client) error {
		syncLogger := logger.Scoped("mounting the working directory")

		command := fmt.Sprintf("mkdir %q && mount_virtiofs %q %q",
			macOSAutomountDirectoryPath, tag, macOSAutomountDirectoryPath)

		syncLogger.Infof("running command: %s", command)

		sshSess, err := sshClient.NewSession()
		if err != nil {
			syncLogger.Finish(false)
			return err
		}

		if err := sshSess.Run(command); err != nil {
			_ = sshSess.Close()

			syncLogger.Finish(false)
			return err
		}

		_ = sshSess.Close()

		syncLogger.Finish(true)
		return nil
	}
}

func unmountWorkingDirectoryHook(logger *echelon.Logger) remoteagent.WaitForAgentHook {
	return func(ctx context.Context, sshClient *ssh.Client) error {
		syncLogger := logger.Scoped("unmounting the working directory")

		command := fmt.Sprintf("umount %q", macOSAutomountDirectoryPath)

		syncLogger.Infof("running command: %s", command)

		sshSess, err := sshClient.NewSession()
		if err != nil {
			syncLogger.Finish(false)
			return err
		}

		var stdout, stderr bytes.Buffer
		sshSess.Stdout = &stdout
		sshSess.Stderr = &stderr

		if err := sshSess.Run(command); err != nil {
			_ = sshSess.Close()

			syncLogger.Errorf("%s", firstNonEmptyLine(stderr.String(), stdout.String()))

			syncLogger.Finish(false)
			return err
		}

		_ = sshSess.Close()

		syncLogger.Finish(true)
		return nil
	}
}
