//go:build !windows

package privdrop

import (
	"errors"
	"fmt"
	userpkg "os/user"
	"strconv"
	"syscall"
)

type Chown struct {
	UID int
	GID int
}

var ErrFailed = errors.New("failed to initialize privilege dropping")

var (
	SysProcAttr *syscall.SysProcAttr
	ChownTo     *Chown
)

func Initialize(username string) error {
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

	uid, err := strconv.Atoi(user.Uid)
	if err != nil {
		return fmt.Errorf("%w: failed to parse %s's user ID %q: %v",
			ErrFailed, user.Name, user.Uid, err)
	}

	SysProcAttr = &syscall.SysProcAttr{
		Credential: &syscall.Credential{
			Uid: uint32(uid),
			Gid: uint32(gid),
		},
	}
	ChownTo = &Chown{
		UID: uid,
		GID: gid,
	}

	return nil
}
