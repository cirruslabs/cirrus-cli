package remoteagent

import (
	"fmt"
	"time"
)

func TimeSyncCommand(t time.Time) string {
	return fmt.Sprintf("sudo date -u %s\n", t.Format("010215042006"))
}
