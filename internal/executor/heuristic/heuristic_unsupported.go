//go:build !linux && !darwin && !windows

package heuristic

import "context"

func GetCloudBuildIP(ctx context.Context) string {
	return ""
}

func IsRunningWindowsContainers(ctx context.Context) bool {
	return false
}
