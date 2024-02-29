package standby

import (
	"context"
	"github.com/cirruslabs/cirrus-ci-agent/api"
	"github.com/cirruslabs/cirrus-cli/internal/executor/instance/abstract"
	"github.com/sirupsen/logrus"
	"google.golang.org/protobuf/proto"
	"sync"
)

type Standby struct {
	Isolation   *api.Isolation
	GetInstance func() (abstract.StandbyCapableInstance, error)

	slot *Slot
	mtx  sync.Mutex
}

func (standby *Standby) TryStart(ctx context.Context, logger logrus.FieldLogger) error {
	standby.mtx.Lock()
	defer standby.mtx.Unlock()

	if standby.slot != nil {
		logger.Debugf("not starting the standby instance since it's already started")

		return nil
	}

	instance, err := standby.GetInstance()
	if err != nil {
		return err
	}

	logger.Debugf("starting standby instance with isolation %s", instance.Isolation())

	standby.slot = NewSlot(ctx, instance)
	go standby.slot.Start(logger)

	return nil
}

func (standby *Standby) Find(
	ctx context.Context,
	userInst abstract.Instance,
	logger logrus.FieldLogger,
) (abstract.Instance, error) {
	standby.mtx.Lock()
	defer standby.mtx.Unlock()

	// Do nothing if standby instance is not started
	if standby.slot == nil {
		return userInst, nil
	}

	// Do nothing if standby instance has not resolved its FQN yet
	standbyFQN, ok := standby.slot.FQN()
	if !ok {
		return userInst, nil
	}

	// Do nothing if the user wants to run a standby-incompatible instance
	userInstance, ok := userInst.(abstract.StandbyCapableInstance)
	if !ok {
		return userInst, nil
	}

	// Pull the user instance and figure out its FQN
	if err := userInstance.Pull(ctx, nil, nil); err != nil {
		return nil, err
	}

	userInstanceFQN, err := userInstance.FQN(ctx)
	if err != nil {
		return nil, err
	}

	logger.Debugf("checking if our standby instance matches the requested FQN %q and isolation %s",
		userInstanceFQN, userInstance.Isolation())

	if standbyFQN != userInstanceFQN || !proto.Equal(standby.Isolation, userInstance.Isolation()) {
		logger.Debugf("terminating standby instance due to no FQN + isolation match")

		if err := standby.slot.Close(); err != nil {
			logger.Errorf("failed to close the standby instance: %v", err)
		}
		standby.slot = nil

		return userInst, nil
	}

	logger.Debugf("standby instance FQN + isolation match, relinquishing our ownership and returning it")

	instance := standby.slot.Instance()
	standby.slot = nil

	return instance, nil
}
