package tart

import (
	"context"
	"errors"
	"fmt"
	"github.com/cirruslabs/cirrus-ci-agent/api"
	"github.com/cirruslabs/cirrus-cli/internal/executor/instance/persistentworker/projectdirsyncer"
	"github.com/cirruslabs/cirrus-cli/internal/executor/instance/persistentworker/remoteagent"
	"github.com/cirruslabs/cirrus-cli/internal/executor/instance/runconfig"
	"github.com/cirruslabs/cirrus-cli/internal/executor/platform"
	"github.com/cirruslabs/cirrus-cli/internal/logger"
	"github.com/getsentry/sentry-go"
	"github.com/samber/lo"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
	"golang.org/x/crypto/ssh"
	"os"
	"path"
	"strings"
)

var (
	ErrFailed     = errors.New("tart isolation failed")
	ErrSyncFailed = errors.New("failed to sync project directory")

	tracer = otel.Tracer("tart")
)

const (
	vmNamePrefix                = "cirrus-cli-"
	macOSAutomountDirectoryPath = "/Volumes/My Shared Files"
	macOSAutomountDirectoryItem = "working-dir"
)

type Tart struct {
	LaunchParameters
	logger   logger.Lightweight
	launcher Launcher
}

func New(
	image string,
	sshUser string,
	sshPassword string,
	cpu uint32,
	memory uint32,
	opts ...Option,
) (*Tart, error) {
	tart := &Tart{
		LaunchParameters: LaunchParameters{
			Image:       image,
			SSHUser:     sshUser,
			SSHPassword: sshPassword,
			CPU:         cpu,
			Memory:      memory,
		},
	}

	// Apply options
	for _, opt := range opts {
		opt(tart)
	}

	// Apply default options (to cover those that weren't specified)
	if tart.logger == nil {
		tart.logger = &logger.LightweightStub{}
	}
	if tart.launcher == nil {
		tart.launcher = &OnDemandLauncher{}
	}

	return tart, nil
}

func (tart *Tart) Run(ctx context.Context, config *runconfig.RunConfig) (err error) {
	addTartListBreadcrumb(ctx)

	if config.DirtyMode {
		tart.Volumes = append(tart.Volumes, &api.Isolation_Tart_Volume{
			Name:     macOSAutomountDirectoryItem,
			Source:   config.ProjectDir,
			ReadOnly: false,
		})
	}

	ctx, prepareInstanceSpan := tracer.Start(ctx, "prepare-instance")
	defer prepareInstanceSpan.End()

	vm, err := tart.launcher.PrepareVM(ctx, tart.LaunchParameters, config.TartOptions, config.AdditionalEnvironment, config.Logger())
	if err != nil {
		prepareInstanceSpan.SetStatus(codes.Error, err.Error())
		return err
	}
	defer func() {
		_ = vm.Release(ctx)
	}()
	prepareInstanceSpan.End()

	initializeHooks := tart.initializeHooks(config)
	terminateHooks := tart.terminateHooks(config)
	err = remoteagent.WaitForAgent(ctx, tart.logger, vm.IP, tart.SSHUser, tart.SSHPassword,
		"darwin", "arm64", config, true, initializeHooks, terminateHooks)
	if err != nil {
		addTartListBreadcrumb(ctx)
		addDHCPDLeasesBreadcrumb(ctx)

		return err
	}

	return nil
}

func (tart *Tart) WorkingDirectory(projectDir string, dirtyMode bool) string {
	if dirtyMode {
		return path.Join(macOSAutomountDirectoryPath, macOSAutomountDirectoryItem)
	}

	return platform.NewUnix().GenericWorkingDir()
}

func (tart *Tart) Close() error {
	// Cleanup volumes created by us
	for _, volume := range tart.Volumes {
		if !volume.Cleanup {
			continue
		}

		volumeIdent := volume.Name

		if volumeIdent == "" {
			volumeIdent = volume.Source
		}

		if err := os.RemoveAll(volume.Source); err != nil {
			return fmt.Errorf("%w: failed to cleanup volume %q: %v", ErrFailed, volumeIdent, err)
		}
	}

	return nil
}

func Cleanup() error {
	stdout, _, err := Cmd(context.Background(), nil, "list", "--quiet")
	if err != nil {
		return err
	}

	for _, vmName := range strings.Split(strings.TrimSpace(stdout), "\n") {
		if !strings.HasPrefix(vmName, vmNamePrefix) {
			continue
		}

		if _, _, err := Cmd(context.Background(), nil, "delete", vmName); err != nil {
			return err
		}
	}

	return nil
}

func (tart *Tart) initializeHooks(config *runconfig.RunConfig) []remoteagent.WaitForAgentHook {
	var hooks []remoteagent.WaitForAgentHook

	if config.ProjectDir != "" && !config.DirtyMode {
		hooks = append(hooks, func(ctx context.Context, sshClient *ssh.Client) error {
			syncLogger := config.Logger().Scoped("syncing working directory")
			if err := projectdirsyncer.SyncProjectDir(config.ProjectDir, sshClient); err != nil {
				syncLogger.Finish(false)
				return fmt.Errorf("%w: %v", ErrSyncFailed, err)
			}

			syncLogger.Finish(true)
			return nil
		})
	}

	if len(tart.Volumes) != 0 {
		hooks = append(hooks, func(ctx context.Context, sshClient *ssh.Client) error {
			syncLogger := config.Logger().Scoped("symlinking volume mounts")

			for _, volume := range tart.Volumes {
				if volume.Target == "" {
					continue
				}

				command := fmt.Sprintf("ln -s \"/Volumes/My Shared Files/%s\" \"%s\"",
					volume.Name, volume.Target)

				syncLogger.Infof("running command: %s", command)

				sshSess, err := sshClient.NewSession()
				if err != nil {
					return err
				}

				if err := sshSess.Run(command); err != nil {
					_ = sshSess.Close()

					return err
				}

				_ = sshSess.Close()
			}

			syncLogger.Finish(true)
			return nil
		})
	}

	return hooks
}

func (tart *Tart) terminateHooks(config *runconfig.RunConfig) []remoteagent.WaitForAgentHook {
	var hooks []remoteagent.WaitForAgentHook

	targetfulVolumes := lo.Filter(tart.Volumes, func(volume *api.Isolation_Tart_Volume, index int) bool {
		return volume.Target != ""
	})

	if len(targetfulVolumes) != 0 {
		hooks = append(hooks, func(ctx context.Context, sshClient *ssh.Client) error {
			syncLogger := config.Logger().Scoped("removing volume mount symlinks")

			for _, volume := range targetfulVolumes {
				command := fmt.Sprintf("rm -f \"%s\"", volume.Target)

				syncLogger.Infof("running command: %s", command)

				sshSess, err := sshClient.NewSession()
				if err != nil {
					return err
				}

				if err := sshSess.Run(command); err != nil {
					_ = sshSess.Close()

					return err
				}

				_ = sshSess.Close()
			}

			syncLogger.Finish(true)
			return nil
		})
	}

	return hooks
}

func addTartListBreadcrumb(ctx context.Context) {
	localHub := sentry.GetHubFromContext(ctx)
	if localHub == nil {
		return
	}

	stdout, stderr, err := Cmd(context.Background(), nil, "list", "--format=json")

	localHub.AddBreadcrumb(&sentry.Breadcrumb{
		Message: "Tart VM list",
		Data: map[string]interface{}{
			"err":    err,
			"stdout": stdout,
			"stderr": stderr,
		},
	}, nil)
}

func addDHCPDLeasesBreadcrumb(ctx context.Context) {
	localHub := sentry.GetHubFromContext(ctx)
	if localHub == nil {
		return
	}

	dhcpdLeasesBytes, err := os.ReadFile("/var/db/dhcpd_leases")

	localHub.AddBreadcrumb(&sentry.Breadcrumb{
		Message: "DHCPD server leases",
		Data: map[string]interface{}{
			"err":                  err,
			"/var/db/dhcpd_leases": string(dhcpdLeasesBytes),
		},
	}, nil)
}
