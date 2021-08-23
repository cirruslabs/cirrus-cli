//go:build windows
// +build windows

package platform

func Auto() Platform {
	return NewWindows("")
}
