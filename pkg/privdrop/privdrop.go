//go:build !windows

package privdrop

import (
	"errors"
	"fmt"
	"golang.org/x/sys/unix"
	"os"
	userpkg "os/user"
	"strconv"
)

var ErrFailed = errors.New("failed to drop privileges")

func Drop(username string) error {
	user, err := userpkg.Lookup(username)
	if err != nil {
		return fmt.Errorf("%w: failed to lookup username %q: %v",
			ErrFailed, username, err)
	}

	gid, err := strconv.Atoi(user.Gid)
	if err != nil {
		return fmt.Errorf("%w: failed to parse %s's group ID %q: %v",
			ErrFailed, user.Name, user.Gid, err)
	}

	if err := unix.Setregid(gid, gid); err != nil {
		return fmt.Errorf("%w: failed to set real and effective group ID to %d: %v",
			ErrFailed, gid, err)
	}

	if err := os.Setenv("HOME", user.HomeDir); err != nil {
		return fmt.Errorf("%w: failed to set new HOME to %q: %v",
			ErrFailed, user.HomeDir, err)
	}

	uid, err := strconv.Atoi(user.Uid)
	if err != nil {
		return fmt.Errorf("%w: failed to parse %s's user ID %q: %v",
			ErrFailed, user.Name, user.Uid, err)
	}

	if err := unix.Setreuid(uid, uid); err != nil {
		return fmt.Errorf("%w: failed to set real and effective user ID to %d: %v",
			ErrFailed, uid, err)
	}

	return nil
}
