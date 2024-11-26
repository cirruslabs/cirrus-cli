package localnetworkhelper

import (
	"context"
	"crypto/ed25519"
	cryptorand "crypto/rand"
	"errors"
	"fmt"
	"golang.org/x/crypto/ssh"
	"golang.org/x/sys/unix"
	"io"
	"net"
	"os"
	"os/exec"
	"time"
)

const (
	CommandName = "local-network-helper"

	// "ssh -J" uses channels of type "direct-tcpip", which are documented
	// in the RFC 4254 (7.2. TCP/IP Forwarding Channels)[1].
	//
	// [1]: https://datatracker.ietf.org/doc/html/rfc4254#section-7.2
	channelTypeDirectTCPIP = "direct-tcpip"
)

var SSHClient *ssh.Client

func StartAndConnect(ctx context.Context) error {
	// Create a socketpair(2) for communicating with the helper process
	socketPair, err := unix.Socketpair(unix.AF_UNIX, unix.SOCK_STREAM, 0)
	if err != nil {
		return err
	}

	// Convert file descriptor numbers to *os.File's
	ourFile := os.NewFile(uintptr(socketPair[0]), "")
	helperFile := os.NewFile(uintptr(socketPair[1]), "")

	// Launch our executable as a child process
	//
	// We're specifying the CommandName argument,
	// so that the child will jump to Serve()
	// and will wait for us.
	executable, err := os.Executable()
	if err != nil {
		return err
	}

	go func() {
		cmd := exec.CommandContext(ctx, executable, CommandName)

		cmd.ExtraFiles = []*os.File{
			helperFile,
		}

		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr

		cmd.WaitDelay = time.Second

		if err := cmd.Run(); !errors.Is(err, context.Canceled) {
			panic(err)
		}
	}()

	// Convert *os.File to net.Conn
	ourConn, err := net.FileConn(ourFile)
	if err != nil {
		return err
	}

	// Check helper connectivity, should be near-instant
	_ = ourConn.SetDeadline(time.Now().Add(1 * time.Second))

	c, chans, reqs, err := ssh.NewClientConn(ourConn, "127.0.0.1:22", &ssh.ClientConfig{
		//nolint:gosec // it's safe to ignore the host key here as we're communicating over IPC
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	})
	if err != nil {
		return err
	}

	SSHClient = ssh.NewClient(c, chans, reqs)

	// Now that the helper process is started,
	// we can close our end of the socketpair(2)
	if err := helperFile.Close(); err != nil {
		return err
	}

	return nil
}

func Serve(ctx context.Context, fd int) error {
	// Convert file descriptor number to *os.File
	file := os.NewFile(uintptr(fd), "")

	// Convert *os.File to net.Conn
	conn, err := net.FileConn(file)
	if err != nil {
		return err
	}

	// Generate a host key to be used by the SSH server
	_, privateKey, err := ed25519.GenerateKey(cryptorand.Reader)
	if err != nil {
		return err
	}

	signer, err := ssh.NewSignerFromKey(privateKey)
	if err != nil {
		return err
	}

	// Configure the SSH server
	serverConfig := &ssh.ServerConfig{
		NoClientAuth: true,
	}
	serverConfig.AddHostKey(signer)

	// Start the SSH server, immediately serving the connection
	sshConn, newChannelCh, requestCh, err := ssh.NewServerConn(conn, serverConfig)
	if err != nil {
		return err
	}
	defer func() {
		_ = sshConn.Close()
	}()

	for {
		select {
		case newChannel, ok := <-newChannelCh:
			if !ok {
				return nil
			}

			switch newChannel.ChannelType() {
			case channelTypeDirectTCPIP:
				go handleDirectTCPIP(newChannel)
			default:
				message := fmt.Sprintf("unsupported channel type requested: %q", newChannel.ChannelType())

				if err := newChannel.Reject(ssh.UnknownChannelType, message); err != nil {
					return err
				}
			}
		case request, ok := <-requestCh:
			if !ok {
				return nil
			}

			if err := request.Reply(false, nil); err != nil {
				return err
			}
		}
	}
}

func handleDirectTCPIP(newChannel ssh.NewChannel) {
	// Unmarshal the payload to determine to which VM the user wants to connect to
	//
	// This direct TCP/IP channel's payload is documented
	// in the RFC 4254 (7.2. TCP/IP Forwarding Channels)[1].
	//
	// [1]: https://datatracker.ietf.org/doc/html/rfc4254#section-7.2
	payload := struct {
		HostToConnect       string
		PortToConnect       uint32
		OriginatorIPAddress string
		OriginatorPort      uint32
	}{}

	if err := ssh.Unmarshal(newChannel.ExtraData(), &payload); err != nil {
		message := fmt.Sprintf("failed to unmarshal payload: %v", err)

		_ = newChannel.Reject(ssh.ConnectionFailed, message)
	}

	targetConn, err := net.Dial("tcp", fmt.Sprintf("%s:%d", payload.HostToConnect, payload.PortToConnect))
	if err != nil {
		panic(err)
	}

	newChan, requestCh, err := newChannel.Accept()
	if err != nil {
		return
	}

	go func() {
		defer func() {
			_ = targetConn.Close()
			_ = newChan.Close()
		}()

		_, _ = io.Copy(targetConn, newChan)
	}()

	go func() {
		defer func() {
			_ = targetConn.Close()
			_ = newChan.Close()
		}()

		_, _ = io.Copy(newChan, targetConn)
	}()

	for request := range requestCh {
		_ = request.Reply(false, nil)
	}
}
