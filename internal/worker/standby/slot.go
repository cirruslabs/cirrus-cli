package standby

import (
	"context"
	"fmt"
	"github.com/cirruslabs/cirrus-ci-agent/api"
	"github.com/cirruslabs/cirrus-cli/internal/executor/instance/abstract"
	"github.com/cirruslabs/cirrus-cli/internal/executor/instance/persistentworker/isolation/tart"
	"github.com/cirruslabs/cirrus-cli/internal/executor/instance/persistentworker/isolation/vetu"
	"github.com/cirruslabs/cirrus-cli/internal/executor/instance/runconfig"
	"github.com/cirruslabs/cirrus-cli/internal/worker/resources"
	"github.com/cirruslabs/cirrus-cli/pkg/parser/instance/isolation"
	"github.com/cirruslabs/cirrus-cli/pkg/parser/node"
	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"
	"sync/atomic"
)

type Slot struct {
	Index       int
	Isolation   *api.Isolation
	GetInstance func() (abstract.StandbyCapableInstance, error)
	Resources   resources.Resources `yaml:"resources"`

	instance abstract.StandbyCapableInstance
	cancel   context.CancelFunc
	logger   logrus.FieldLogger

	fqn atomic.Value
}

func (slot *Slot) UnmarshalYAML(value *yaml.Node) error {
	// Parse isolation
	isolation := isolation.NewIsolation(nil, nil)

	node, err := node.NewFromNodeWithMergeExemptions(yaml.Node{
		Kind: yaml.DocumentNode,
		Content: []*yaml.Node{
			value,
		},
	}, nil)
	if err != nil {
		return err
	}

	if err := isolation.Parse(node, nil); err != nil {
		return err
	}

	slot.Isolation = isolation.Proto()

	switch iso := isolation.Proto().Type.(type) {
	case *api.Isolation_Tart_:
		slot.GetInstance = func() (abstract.StandbyCapableInstance, error) {
			return tart.NewFromIsolation(iso)
		}
	case *api.Isolation_Vetu_:
		slot.GetInstance = func() (abstract.StandbyCapableInstance, error) {
			return vetu.NewFromIsolation(iso)
		}
	default:
		return fmt.Errorf("%w, got %T", ErrUnsupportedIsolation, iso)
	}

	// Parse resources
	resources := resources.New()

	for i := 0; i < len(value.Content); i += 2 {
		key := value.Content[i]
		value := value.Content[i+1]

		if key.Value != "resources" {
			continue
		}

		if err := value.Decode(resources); err != nil {
			return err
		}
	}

	slot.Resources = resources

	return nil
}

func (slot *Slot) String() string {
	return fmt.Sprintf("#%d (%v, %s)", slot.Index, slot.Isolation, slot.Resources)
}

func (slot *Slot) start(ctx context.Context, logger logrus.FieldLogger) {
	logger.Debugf("pulling instance with spec %#v", slot.instance.Isolation())

	if err := slot.instance.Pull(ctx, nil, nil); err != nil {
		slot.logger.Errorf("failed to pull instance for slot %s: %v", slot, err)

		return
	}

	fqn, err := slot.instance.FQN(ctx)
	if err != nil {
		slot.logger.Errorf("failed to retrieve the FQN of the instance for slot %s: %v", slot, err)

		return
	}

	logger.Debugf("instance with spec %#v pulled, FQN is %s", slot.instance.Isolation(), fqn)

	slot.fqn.Store(fqn)

	if _, err := slot.instance.CloneConfigureStart(ctx, &runconfig.RunConfig{
		ProjectDir: "",
		TaskID:     fmt.Sprintf("standby-%d", slot.Index),
	}); err != nil {
		slot.logger.Errorf("failed to clone, configure and start instance for slot %s: %v", slot, err)

		return
	}

	<-ctx.Done()
}

func (slot *Slot) stop() {
	slot.cancel()
	slot.cancel = nil
}
