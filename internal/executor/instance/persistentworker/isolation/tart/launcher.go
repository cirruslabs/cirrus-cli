package tart

import (
	"context"
	"fmt"
	"github.com/cirruslabs/cirrus-ci-agent/api"
	"github.com/cirruslabs/cirrus-cli/internal/executor/instance/abstract"
	"github.com/cirruslabs/cirrus-cli/internal/executor/options"
	"github.com/cirruslabs/echelon"
	"github.com/getsentry/sentry-go"
	"github.com/google/uuid"
	"os"
	"time"
)

type LaunchParameters struct {
	Image       string `yaml:"image"`
	SSHUser     string `yaml:"ssh_user"`
	SSHPassword string `yaml:"ssh_password"`
	CPU         uint32 `yaml:"cpu"`
	Memory      uint32 `yaml:"memory"`
	DiskSize    uint32 `yaml:"disk_size"`
	Softnet     bool   `yaml:"softnet"`
	Display     string `yaml:"display"`
	// dont't support volumes for standby instances
	Volumes []*api.Isolation_Tart_Volume
}

type Launcher interface {
	PrepareVM(
		ctx context.Context,
		tartParameters LaunchParameters,
		tartOptions options.TartOptions,
		additionalEnvironment map[string]string,
		logger *echelon.Logger,
	) (*LaunchedVM, error)
}

type LaunchedVM struct {
	IP      string
	Release func(context.Context) error
}

type OnDemandLauncher struct {
}

func (l *OnDemandLauncher) PrepareVM(
	ctx context.Context,
	tartParameters LaunchParameters,
	tartOptions options.TartOptions,
	additionalEnvironment map[string]string,
	logger *echelon.Logger,
) (*LaunchedVM, error) {
	if localHub := sentry.GetHubFromContext(ctx); localHub != nil {
		localHub.ConfigureScope(func(scope *sentry.Scope) {
			scope.SetExtra("Softnet enabled", tartParameters.Softnet)
		})
	}

	tmpVMName := vmNamePrefix + uuid.NewString()
	vm, err := NewVMClonedFrom(ctx,
		tartParameters.Image, tmpVMName,
		tartOptions.LazyPull,
		additionalEnvironment,
		logger,
	)
	if err != nil {
		return nil, fmt.Errorf("%w: failed to create VM cloned from %q: %v", ErrFailed, tartParameters.Image, err)
	}

	if err := vm.Configure(ctx, tartParameters.CPU, tartParameters.Memory,
		tartParameters.DiskSize, tartParameters.Display, logger); err != nil {
		return nil, fmt.Errorf("%w: failed to configure VM %q: %v", ErrFailed, vm.Ident(), err)
	}

	// Convert volumes to directory mounts
	var directoryMounts []directoryMount
	for _, volume := range tartParameters.Volumes {
		if volume.Name == "" {
			volume.Name = uuid.NewString()
		}

		_, err = os.Stat(volume.Source)
		if err != nil {
			if os.IsNotExist(err) {
				if err := os.Mkdir(volume.Source, 0755); err != nil {
					return nil, fmt.Errorf("%w: volume source %q doesn't exist, failed to pre-create it: %v",
						ErrFailed, volume.Source, err)
				}

				volume.Cleanup = true
			} else {
				return nil, fmt.Errorf("%w: volume source %q cannot be accessed: %v",
					ErrFailed, volume.Source, err)
			}
		}

		directoryMounts = append(directoryMounts, directoryMount{
			Name:     volume.Name,
			Path:     volume.Source,
			ReadOnly: volume.ReadOnly,
		})
	}

	// Start the VM (asynchronously)
	vm.Start(ctx, tartParameters.Softnet, directoryMounts)

	// Wait for the VM to start and get it's DHCP address
	var ip string
	bootLogger := logger.Scoped("boot virtual machine")

	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case err := <-vm.ErrChan():
			return nil, err
		default:
			time.Sleep(time.Second)
		}

		ip, err = vm.RetrieveIP(ctx)
		if err != nil {
			logger.Debugf("failed to retrieve VM %s IP: %v\n", vm.Ident(), err)
			continue
		}

		break
	}

	logger.Debugf("IP %s retrieved from VM %s, running agent...", ip, vm.Ident())

	bootLogger.Errorf("VM was assigned with %s IP", ip)
	bootLogger.Finish(true)

	addDHCPDLeasesBreadcrumb(ctx)

	return &LaunchedVM{
		IP: ip,
		Release: func(ctx context.Context) error {
			if localHub := sentry.GetHubFromContext(ctx); localHub != nil {
				localHub.AddBreadcrumb(&sentry.Breadcrumb{
					Message: fmt.Sprintf("stopping and deleting the VM %s", vm.ident),
				}, nil)
			}

			return vm.Close()
		},
	}, nil
}

type StandByLauncher struct {
	parameters LaunchParameters
	vm         *VM
}

func NewStandByLauncher(parameters LaunchParameters) *StandByLauncher {
	return &StandByLauncher{
		parameters: parameters,
	}

}

func (l *StandByLauncher) PrepareVM(
	ctx context.Context,
	tartParameters LaunchParameters,
	tartOptions options.TartOptions,
	additionalEnvironment map[string]string,
	logger *echelon.Logger,
) (*LaunchedVM, error) {
	return nil, nil
}

func (l *StandByLauncher) Name() string {
	return "TartStandByLauncher"
}

func (l *StandByLauncher) BeforePoll(ctx context.Context, request *api.PollRequest) error {
	return nil
}

func (l *StandByLauncher) BeforeRunInstance(ctx context.Context, inst abstract.Instance) error {
	return nil
}
