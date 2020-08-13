package build

import (
	"github.com/cirruslabs/cirrus-ci-agent/api"
	"github.com/cirruslabs/cirrus-cli/internal/executor/build/taskstatus"
	"sync"
)

type Command struct {
	status taskstatus.Status

	// Original Protocol Buffers structure for reference
	ProtoCommand *api.Command

	// A mutex to guarantee safe accesses from both the main loop and gRPC server handlers
	Mutex sync.RWMutex
}

func (command *Command) Status() taskstatus.Status {
	command.Mutex.RLock()
	defer command.Mutex.RUnlock()

	return command.status
}

func (command *Command) SetStatus(status taskstatus.Status) {
	command.Mutex.Lock()
	defer command.Mutex.Unlock()

	command.status = status
}
