//go:build !windows

package filestorage

import "os"

func renameFile(oldpath, newpath string) error {
	return os.Rename(oldpath, newpath)
}
