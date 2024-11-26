//go:build windows

package privdrop

import "fmt"

func Drop(username string) error {
	return fmt.Errorf("privilege dropping is not implemented on this platform")
}
