package standby

import (
	"context"
	"errors"
	"github.com/cirruslabs/cirrus-ci-agent/api"
	"github.com/cirruslabs/cirrus-cli/internal/executor/instance/abstract"
	"github.com/cirruslabs/cirrus-cli/internal/worker/resources"
	"github.com/sirupsen/logrus"
	"google.golang.org/protobuf/proto"
	"sync"
)

var ErrUnsupportedIsolation = errors.New("only Tart and Vetu isolation types are currently supported " +
	"as standby targets")

type Standby struct {
	slots  []*Slot
	logger logrus.FieldLogger

	mtx sync.Mutex
}

func New(slots []*Slot, logger logrus.FieldLogger) (*Standby, error) {
	// Assign slot indexes for better diagnostics
	for index, slot := range slots {
		slot.Index = index
	}

	return &Standby{
		slots:  slots,
		logger: logger,
	}, nil
}

func (standby *Standby) StartSlots(ctx context.Context, workerAvailableResources resources.Resources) error {
	standby.mtx.Lock()
	defer standby.mtx.Unlock()

	for _, slot := range standby.slots {
		// Re-calculate the total available resources on each iteration
		// because previous iterations might have started some slots
		totalAvailableResources := workerAvailableResources.Sub(standby.resourcesInUse())

		// Do not start the slots that are already started
		if slot.cancel != nil {
			standby.logger.Debugf("standby slot %s is already started", slot)

			continue
		}

		// Do not start the slots whose resources can't fit into the available resources
		if !totalAvailableResources.CanFit(slot.Resources) {
			standby.logger.Debugf("not starting a standby slot %s because it cannot fit "+
				"into available resources (%s)", slot, totalAvailableResources)

			continue
		}

		standby.logger.Debugf("starting standby slot %s", slot)

		instance, err := slot.GetInstance()
		if err != nil {
			return err
		}

		slotCtx, slotCtxCancel := context.WithCancel(ctx)

		slot.instance = instance
		slot.cancel = slotCtxCancel

		go slot.start(slotCtx, standby.logger)

		standby.logger.Infof("started standby slot %s", slot)
	}

	return nil
}

func (standby *Standby) FindSlot(fqn string, isolation *api.Isolation) abstract.Instance {
	standby.mtx.Lock()
	defer standby.mtx.Unlock()

	standby.logger.Debugf("finding a slot for FQN %q and isolation %s", fqn, isolation)

	// Iterate slots in descending priority
	for _, slot := range standby.slots {
		if slot.cancel == nil {
			continue
		}

		// We're only interested in slots whose spec matches ours
		if !proto.Equal(slot.instance.Isolation(), isolation) {
			continue
		}

		// Even if the isolation matches, FQN can be different due to VM image updates
		//
		// Stop that slot to cause its refresh and continue
		if slot.fqn.Load() != fqn {
			slot.stop()
		}

		standby.logger.Debugf("found a matching slot %s", slot)

		// Relinquish ownership
		slot.cancel = nil

		return slot.instance
	}

	return nil
}

func (standby *Standby) StopSlots(needResources resources.Resources) {
	standby.mtx.Lock()
	defer standby.mtx.Unlock()

	standby.logger.Debugf("stopping slots in hope of relinquishing %v", needResources)

	// Iterate slots in ascending priority
	for i := len(standby.slots) - 1; i >= 0 && needResources.HasPositiveResources(); i-- {
		slot := standby.slots[i]

		if slot.cancel == nil {
			continue
		}

		// Lack of resource overlap means there's no point in stopping this slot
		if !slot.Resources.Overlaps(needResources) {
			continue
		}

		slot.stop()

		// Subtract the resources that were freed up
		needResources = needResources.Sub(slot.Resources)
	}
}

func (standby *Standby) resourcesInUse() resources.Resources {
	result := resources.New()

	for _, slot := range standby.slots {
		if slot.cancel == nil {
			continue
		}

		result = result.Add(slot.Resources)
	}

	return result
}
