// +build !windows

package platform

func Auto() Platform {
	return NewUnix()
}
