package standby

import (
	"context"
	"github.com/cirruslabs/cirrus-cli/internal/executor/instance/abstract"
	"github.com/cirruslabs/cirrus-cli/internal/executor/instance/runconfig"
	"github.com/sirupsen/logrus"
	"sync/atomic"
)

type Slot struct {
	instance abstract.StandbyCapableInstance
	ctx      context.Context
	cancel   context.CancelFunc
	fqn      atomic.Pointer[string]
}

func NewSlot(ctx context.Context, instance abstract.StandbyCapableInstance) *Slot {
	subCtx, subCtxCancel := context.WithCancel(ctx)

	return &Slot{
		instance: instance,
		ctx:      subCtx,
		cancel:   subCtxCancel,
	}
}

func (slot *Slot) Instance() abstract.StandbyCapableInstance {
	return slot.instance
}

func (slot *Slot) FQN() (string, bool) {
	if fqn := slot.fqn.Load(); fqn != nil {
		return *fqn, true
	}

	return "", false
}

func (slot *Slot) Start(logger logrus.FieldLogger) {
	logger.Debugf("pulling standby instance")

	if err := slot.instance.Pull(slot.ctx, nil, nil); err != nil {
		logger.Errorf("failed to pull the standby instance: %v", err)

		return
	}

	fqn, err := slot.instance.FQN(slot.ctx)
	if err != nil {
		logger.Errorf("failed to retrieve the FQN of the standby instance: %v", err)

		return
	}

	logger.Debugf("pulled standby instance, FQN is %s", fqn)

	if _, err := slot.instance.CloneConfigureStart(slot.ctx, &runconfig.RunConfig{
		ProjectDir: "",
		TaskID:     0, // use "standby" in the future, see https://github.com/cirruslabs/cirrus-cli/issues/694
	}); err != nil {
		logger.Errorf("failed to clone, configure and start the standby instance: %v", err)

		return
	}

	logger.Debugf("standby instance was successfully started")

	slot.fqn.Store(&fqn)
}

func (slot *Slot) Close() error {
	slot.cancel()

	return slot.instance.Close()
}
