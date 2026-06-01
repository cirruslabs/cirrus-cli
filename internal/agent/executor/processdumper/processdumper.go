//go:build !(openbsd || netbsd)

package processdumper

import (
	"fmt"
	"log/slog"
	"sort"

	"github.com/mitchellh/go-ps"
	gopsutilprocess "github.com/shirou/gopsutil/v3/process"
)

func Dump() {
	processes, err := ps.Processes()
	if err != nil {
		slog.Warn("Failed to retrieve processes to diagnose the time out", "err", err)
	} else {
		fmt.Println("Dumping process list to diagnose the time out")
		fmt.Println("PID\tPPID\tExe or cmdline")

		sort.Slice(processes, func(left, right int) bool {
			return processes[left].Pid() < processes[right].Pid()
		})

		for _, process := range processes {
			fmt.Printf("%d\t%d\t%s\n", process.Pid(), process.PPid(), processExeOrCmdline(process))
		}
	}
}

func processExeOrCmdline(process ps.Process) string {
	gopsutilProcess, err := gopsutilprocess.NewProcess(int32(process.Pid()))
	if err != nil {
		// Fall back to just the comm value
		return process.Executable()
	}

	cmdline, err := gopsutilProcess.Cmdline()
	if err != nil {
		// Fall back to just the comm value
		return process.Executable()
	}

	return cmdline
}
