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

		err := remoteCommand(sshClient, syncLogger, fmt.Sprintf("mkdir %q && mount_virtiofs %q %q",
			macOSAutomountDirectoryPath, tag, macOSAutomountDirectoryPath))
		if err != nil {
			syncLogger.Finish(false)

			return err
		}

		syncLogger.Finish(true)

		return nil
	}
}

func unmountWorkingDirectoryHook(logger *echelon.Logger) remoteagent.WaitForAgentHook {
	return func(ctx context.Context, sshClient *ssh.Client) error {
		syncLogger := logger.Scoped("unmounting the working directory")

		err := remoteCommand(sshClient, syncLogger, fmt.Sprintf("umount %q",
			macOSAutomountDirectoryPath))
		if err == nil {
			syncLogger.Finish(true)

			return nil
		}

		err = remoteCommand(sshClient, syncLogger, fmt.Sprintf("diskutil unmount %q",
			macOSAutomountDirectoryPath))
		if err == nil {
			syncLogger.Finish(true)

			return nil
		}

		syncLogger.Finish(false)

		return err
	}
}

func remoteCommand(sshClient *ssh.Client, syncLogger *echelon.Logger, command string) error {
	syncLogger.Infof("running command: %s", command)

	sshSess, err := sshClient.NewSession()
	if err != nil {
		return err
	}
	defer sshSess.Close()

	var stdout, stderr bytes.Buffer
	sshSess.Stdout = &stdout
	sshSess.Stderr = &stderr

	if err = sshSess.Run(command); err != nil {
		syncLogger.Errorf("%s", firstNonEmptyLine(stderr.String(), stdout.String()))

		return err
	}

	return nil
}
