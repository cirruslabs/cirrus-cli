package projectdirsyncer

import (
	"github.com/cirruslabs/cirrus-cli/internal/executor/platform"
	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
	"io"
	"os"
	"path/filepath"
)

func SyncProjectDir(dir string, sshClient *ssh.Client) error {
	sftpClient, err := sftp.NewClient(sshClient)
	if err != nil {
		return err
	}
	defer sftpClient.Close()

	return filepath.Walk(dir, func(path string, fileInfo os.FileInfo, err error) error {
		// Handle possible error that occurred when reading this directory entry information
		if err != nil {
			return err
		}

		relativePath, err := filepath.Rel(dir, path)
		if err != nil {
			return err
		}
		remotePath := sftp.Join(platform.NewUnix().GenericWorkingDir(), relativePath)

		if fileInfo.Mode().IsDir() {
			return sftpClient.MkdirAll(remotePath)
		} else if fileInfo.Mode().IsRegular() {
			localFile, err := os.Open(path)
			if err != nil {
				return err
			}
			defer localFile.Close()

			remoteFile, err := sftpClient.Create(remotePath)
			if err != nil {
				return err
			}
			defer remoteFile.Close()

			if _, err := io.Copy(remoteFile, localFile); err != nil {
				return err
			}
		}

		return nil
	})
}
