package terminalwrapper

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"github.com/avast/retry-go/v4"
	"github.com/cirruslabs/cirrus-cli/internal/agent/client"
	"github.com/cirruslabs/cirrus-cli/pkg/api"
	"github.com/cirruslabs/terminal/pkg/host"
	"github.com/cirruslabs/terminal/pkg/host/session"
	"time"
)

type Wrapper struct {
	ctx                context.Context
	taskIdentification *api.TaskIdentification
	operationChan      chan Operation
	terminalHost       *host.TerminalHost
	expirationWindow   time.Duration
}

func New(
	ctx context.Context,
	taskIdentification *api.TaskIdentification,
	serverAddress string,
	expirationWindow time.Duration,
	shellEnv []string,
) *Wrapper {
	wrapper := &Wrapper{
		ctx:                ctx,
		taskIdentification: taskIdentification,
		operationChan:      make(chan Operation, 4096),
		expirationWindow:   expirationWindow,
	}

	// A trusted secret that grants ability to spawn shells on the terminal host we start below
	trustedSecret, err := generateTrustedSecret()
	if err != nil {
		wrapper.operationChan <- &LogOperation{Message: fmt.Sprintf("Unable to generate a trusted secret needed to"+
			" initialize a terminal host: %v", err)}
		return wrapper
	}

	// A callback that will be called once the terminal host connects and registers on the terminal server
	locatorCallback := func(locator string) error {
		_, err := client.CirrusClient.ReportTerminalAttached(ctx, &api.ReportTerminalAttachedRequest{
			TaskIdentification: taskIdentification,
			Locator:            locator,
			TrustedSecret:      trustedSecret,
		})
		if err != nil {
			return err
		}

		_, err = client.CirrusClient.ReportTerminalLifecycle(wrapper.ctx, &api.ReportTerminalLifecycleRequest{
			TaskIdentification: wrapper.taskIdentification,
			Lifecycle: &api.ReportTerminalLifecycleRequest_Started_{
				Started: &api.ReportTerminalLifecycleRequest_Started{},
			},
		})
		if err != nil {
			wrapper.operationChan <- &LogOperation{
				Message: fmt.Sprintf("Failed to send lifecycle notification (started): %v", err),
			}
		}

		return err
	}

	terminalHostOpts := []host.Option{
		host.WithTrustedSecret(trustedSecret),
		host.WithLocatorCallback(locatorCallback),
		host.WithShellEnv(shellEnv),
	}

	if serverAddress != "" {
		terminalHostOpts = append(terminalHostOpts, host.WithServerAddress(serverAddress))
	}

	wrapper.terminalHost, err = host.New(terminalHostOpts...)
	if err != nil {
		wrapper.operationChan <- &LogOperation{Message: fmt.Sprintf("Failed to initialize a terminal host: %v", err)}
		return wrapper
	}

	go func() {
		_ = retry.Do(
			func() error {
				subCtx, cancel := context.WithCancel(ctx)
				defer cancel()

				return wrapper.terminalHost.Run(subCtx)
			},
			retry.OnRetry(func(n uint, err error) {
				wrapper.operationChan <- &LogOperation{Message: fmt.Sprintf("Terminal host failed: %v", err)}
			}),
			retry.Context(ctx),
			retry.Delay(5*time.Second), retry.MaxDelay(5*time.Second),
			retry.Attempts(0), retry.LastErrorOnly(true),
		)
	}()

	return wrapper
}

func (wrapper *Wrapper) Wait() chan Operation {
	go func() {
		// Might happen when we fail to initialize the terminal host
		if wrapper.terminalHost == nil {
			wrapper.operationChan <- &ExitOperation{Success: false}

			return
		}

		// Wait for the terminal to connect, exit on ctx cancellation/deadline
		if !wrapper.waitForConnection() {
			return
		}

		// Wait for the terminal to be inactive for a period of wrapper.expirationWindow.Seconds() seconds
		for {
			lastActivityBeforeWait := max(wrapper.terminalHost.LastRegistration(),
				wrapper.terminalHost.LastActivity())

			// Notify the user that the countdown has started
			message := fmt.Sprintf("Waiting for the terminal session to be inactive for at least %.1f seconds...",
				wrapper.expirationWindow.Seconds())
			wrapper.operationChan <- &LogOperation{Message: message}

			// Notify the server that the countdown has started
			_, err := client.CirrusClient.ReportTerminalLifecycle(wrapper.ctx, &api.ReportTerminalLifecycleRequest{
				TaskIdentification: wrapper.taskIdentification,
				Lifecycle: &api.ReportTerminalLifecycleRequest_Expiring_{
					Expiring: &api.ReportTerminalLifecycleRequest_Expiring{},
				},
			})
			if err != nil {
				wrapper.operationChan <- &LogOperation{
					Message: fmt.Sprintf("Failed to send lifecycle notification (expiring): %v", err),
				}
			}

			select {
			case <-time.After(wrapper.expirationWindow):
				numActiveSessions := wrapper.terminalHost.NumSessionsFunc(func(session *session.Session) bool {
					return session.LastActivity().After(lastActivityBeforeWait)
				})

				if numActiveSessions == 0 {
					wrapper.operationChan <- &ExitOperation{Success: true}

					return
				}

				message := fmt.Sprintf("Waited %.1f seconds, but there are still %d terminal sessions open "+
					"and %d of them are active.", wrapper.expirationWindow.Seconds(), wrapper.terminalHost.NumSessions(),
					numActiveSessions)
				wrapper.operationChan <- &LogOperation{Message: message}

				continue
			case <-wrapper.ctx.Done():
				wrapper.operationChan <- &ExitOperation{Success: false}

				return
			}
		}
	}()

	return wrapper.operationChan
}

func (wrapper *Wrapper) waitForConnection() bool {
	wrapper.operationChan <- &LogOperation{
		Message: "Waiting for the terminal server connection to be established...",
	}

	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			defaultTime := time.Time{}
			if wrapper.terminalHost.LastConnection() != defaultTime {
				return true
			}
		case <-wrapper.ctx.Done():
			wrapper.operationChan <- &ExitOperation{Success: true}
			return false
		}
	}
}

func generateTrustedSecret() (string, error) {
	buf := make([]byte, 32)

	_, err := rand.Read(buf)
	if err != nil {
		return "", err
	}

	return hex.EncodeToString(buf), nil
}

func max(times ...time.Time) time.Time {
	var result time.Time

	for _, time := range times {
		if time.After(result) {
			result = time
		}
	}

	return result
}
