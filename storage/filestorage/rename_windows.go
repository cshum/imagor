//go:build windows

package filestorage

import (
	"errors"
	"syscall"
	"time"
)

// Errno 32 is not exported by syscall; it is what MoveFileEx returns when a
// reader has the target open without sharing delete.
const errSharingViolation = syscall.Errno(32)

// renameFile replaces the target, retrying while a reader holds it open.
func renameFile(oldpath, newpath string) error {
	var err error
	for i := 0; i < 10; i++ {
		if err = syscall.Rename(oldpath, newpath); err == nil {
			return nil
		}
		if !errors.Is(err, errSharingViolation) && !errors.Is(err, syscall.ERROR_ACCESS_DENIED) {
			return err
		}
		time.Sleep(time.Duration(i+1) * 10 * time.Millisecond)
	}
	return err
}
