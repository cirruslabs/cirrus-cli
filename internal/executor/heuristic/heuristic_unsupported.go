// +build !linux,!darwin,!windows

package heuristic

import "context"

func GetCloudBuildIP(ctx context.Context) string {
	return ""
}
