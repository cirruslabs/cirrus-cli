//go:build windows

package privdrop

import "fmt"

func Initialize(username string) error {
	return fmt.Errorf("privilege dropping is not supported on this platform")
}
