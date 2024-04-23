package executor

import (
	"context"
	"errors"
	"fmt"
	"github.com/cirruslabs/cirrus-cli/internal/agent/environment"
	"github.com/cirruslabs/cirrus-cli/internal/agent/executor/piper"
	"github.com/cirruslabs/cirrus-cli/internal/agent/executor/processdumper"
	"io"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"syscall"
	"time"
)

type ShellOutputHandler func(bytes []byte) (int, error)

type ShellOutputWriter struct {
	io.Writer
	handler ShellOutputHandler
}

var TimeOutError = errors.New("timed out")

func (writer ShellOutputWriter) Write(bytes []byte) (int, error) {
	return writer.handler(bytes)
}

// return true if executed successful
func ShellCommandsAndWait(
	ctx context.Context,
	scripts []string,
	custom_env *environment.Environment,
	handler ShellOutputHandler,
	shouldKillProcesses bool,
) (*exec.Cmd, error) {
	sc, err := NewShellCommands(ctx, scripts, custom_env, handler)
	if err != nil {
		return nil, err
	}

	cmd := sc.cmd

	done := make(chan error)
	go func() {
		// give time to flush logs
		time.Sleep(1 * time.Second)
		done <- cmd.Wait()
	}()
	go func() {
		for {
			time.Sleep(10 * time.Second)
			if cmd.ProcessState != nil && cmd.ProcessState.Exited() {
				done <- nil
			}
		}
	}()

	select {
	case <-ctx.Done():
		handler([]byte("\nTimed out!"))

		processdumper.Dump()

		if err = sc.kill(); err != nil {
			handler([]byte(fmt.Sprintf("\nFailed to kill a timed out shell session: %s", err)))
		}

		return cmd, TimeOutError
	case <-done:
		var forcePiperClosure bool

		if shouldKillProcesses {
			_ = sc.kill()
		} else {
			forcePiperClosure = true
		}

		if err := sc.piper.Close(ctx, forcePiperClosure); err != nil {
			handler([]byte(fmt.Sprintf("\nShell session I/O error: %s", err)))
		}

		if ws, ok := cmd.ProcessState.Sys().(syscall.WaitStatus); ok {
			if ws.Signaled() {
				message := fmt.Sprintf("\nSignaled to exit (%v)!", ws.Signal())
				handler([]byte(message))
			}
			exitStatus := ws.ExitStatus()
			if exitStatus > 1 {
				handler([]byte(fmt.Sprintf("\nExit status: %d", exitStatus)))
			}
		} else {
			log.Printf("Failed to get wait status: %v", cmd.ProcessState.Sys())
		}
		return cmd, nil
	}
}

func NewShellCommands(
	ctx context.Context,
	scripts []string,
	custom_env *environment.Environment,
	handler ShellOutputHandler,
) (*ShellCommands, error) {
	var cmd *exec.Cmd
	var scriptFile *os.File
	var err error

	cmd, scriptFile, err = createCmd(scripts, custom_env)

	sc := &ShellCommands{cmd: cmd}

	if scriptFile != nil {
		sigs := make(chan os.Signal, 1)
		signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
		go func() {
			<-sigs
			os.Remove(scriptFile.Name())
		}()
	}

	if err != nil {
		message := fmt.Sprintf("Error creating command-line script: %s", err)
		handler([]byte(message))
		return nil, errors.New(message)
	}

	env := os.Environ()
	if custom_env != nil {
		for k, v := range custom_env.Items() {
			env = append(env, fmt.Sprintf("%s=%s", k, v))
		}

		if _, environmentAlreadyHasShell := os.LookupEnv("SHELL"); environmentAlreadyHasShell {
			_, userSpecifiedShell := custom_env.Lookup("SHELL")
			if shellOverride, userSpecifiedCustomShell := custom_env.Lookup("CIRRUS_SHELL"); userSpecifiedCustomShell && !userSpecifiedShell {
				env = append(env, fmt.Sprintf("SHELL=%s", shellOverride))
			}
		}
	}

	cmd.Env = env
	if custom_env != nil {
		if workingDir, ok := custom_env.Lookup("CIRRUS_WORKING_DIR"); ok {
			EnsureFolderExists(workingDir)
			cmd.Dir = workingDir
		}
	}

	var writer io.Writer

	writer = ShellOutputWriter{
		handler: handler,
	}

	if custom_env != nil {
		if _, ok := custom_env.Lookup("CIRRUS_AGENT_EXPOSE_SCRIPTS_OUTPUTS"); ok {
			writer = io.MultiWriter(os.Stdout, writer)
		}
	}

	// Work around https://github.com/golang/go/issues/23019 by creating a pipe
	// and passing *os.File to exec.Cmd's Stderr and Stdout fields, which results
	// in skipping of exec.Cmd.Start()'s internal io.Copy() logic that might block
	// when the Shell started by us shares it's stderr/stdout file descriptor with
	// other processes that run in the background
	sc.piper, err = piper.New(writer)
	if err != nil {
		return nil, err
	}

	cmd.Stderr = sc.piper.FileProxy()
	cmd.Stdout = sc.piper.FileProxy()

	if err := sc.beforeStart(custom_env); err != nil {
		return nil, err
	}

	err = cmd.Start()
	if err != nil {
		if err := sc.piper.Close(ctx, true); err != nil {
			_, _ = fmt.Fprintf(writer, "Shell session I/O error: %s", err)
		}

		message := fmt.Sprintf("Error starting command: %s", err)
		handler([]byte(message))
		return nil, errors.New(message)
	}

	sc.afterStart()

	// At this point the shell has successfully started and inherited
	// the proxy file descriptor. We can release our own descriptor now.
	if err := sc.piper.FileProxy().Close(); err != nil {
		_, _ = fmt.Fprintf(writer, "Shell session I/O error: %s", err)
	}

	return sc, nil
}
