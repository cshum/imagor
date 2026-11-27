package filestorage

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/cshum/imagor"
	"github.com/cshum/imagor/imagorpath"
)

var dotFileRegex = regexp.MustCompile("/\\.")

// FileStorage File Storage implements imagor.Storage interface
type FileStorage struct {
	BaseDir         string
	PathPrefix      string
	Blacklists      []*regexp.Regexp
	MkdirPermission os.FileMode
	WritePermission os.FileMode
	SaveErrIfExists bool
	SafeChars       string
	Expiration      time.Duration

	safeChars imagorpath.SafeChars
}

// New creates FileStorage
func New(baseDir string, options ...Option) *FileStorage {
	s := &FileStorage{
		BaseDir:         baseDir,
		PathPrefix:      "/",
		Blacklists:      []*regexp.Regexp{dotFileRegex},
		MkdirPermission: 0755,
		WritePermission: 0666,
	}
	for _, option := range options {
		option(s)
	}
	s.safeChars = imagorpath.NewSafeChars(s.SafeChars)
	return s
}

// Path transforms and validates image key for storage path
func (s *FileStorage) Path(image string) (string, bool) {
	image = "/" + imagorpath.Normalize(image, s.safeChars)
	for _, blacklist := range s.Blacklists {
		if blacklist.MatchString(image) {
			return "", false
		}
	}
	if !strings.HasPrefix(image, s.PathPrefix) {
		return "", false
	}
	return confineJoin(s.BaseDir, strings.TrimPrefix(image, s.PathPrefix))
}

// confineJoin joins the image key with the base directory, rejecting keys
// whose joined path would resolve outside the base directory, so that
// storage paths cannot escape the base directory even when the blacklist
// configuration does not block dot segments or backslash separators.
func confineJoin(baseDir, key string) (string, bool) {
	p := filepath.Join(baseDir, key)
	if !withinBase(baseDir, p) {
		return "", false
	}
	return p, true
}

// withinBase reports whether path p stays inside baseDir, lexically,
// after both are resolved by filepath join semantics.
func withinBase(baseDir, p string) bool {
	rel, err := filepath.Rel(filepath.Clean(baseDir), p)
	if err != nil {
		return false
	}
	return rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}

// Get implements imagor.Storage interface
func (s *FileStorage) Get(_ *http.Request, image string) (*imagor.Blob, error) {
	image, ok := s.Path(image)
	if !ok {
		return nil, imagor.ErrInvalid
	}
	blob := imagor.NewBlobFromFile(image, func(stat os.FileInfo) error {
		if s.Expiration > 0 && time.Now().Sub(stat.ModTime()) > s.Expiration {
			return imagor.ErrExpired
		}
		return nil
	})
	return blob, blob.Err()
}

// Put implements imagor.Storage interface
func (s *FileStorage) Put(_ context.Context, image string, blob *imagor.Blob) (err error) {
	image, ok := s.Path(image)
	if !ok {
		return imagor.ErrInvalid
	}
	dir := filepath.Dir(image)
	if err = os.MkdirAll(dir, s.MkdirPermission); err != nil {
		return
	}
	reader, _, err := blob.NewReader()
	if err != nil {
		return err
	}
	defer func() {
		_ = reader.Close()
	}()
	// Written beside the target and renamed, so a reader never sees a partial file.
	perm, replacing := s.WritePermission, false
	if st, statErr := os.Lstat(image); statErr == nil && st.Mode().IsRegular() {
		perm, replacing = st.Mode().Perm(), true
	}
	w, err := createTemp(dir, perm)
	if err != nil {
		return err
	}
	tmp := w.Name()
	if replacing {
		// The umask shapes a new file's mode; an overwrite keeps the file's own.
		if err = os.Chmod(tmp, perm); err != nil {
			return
		}
	}
	defer func() {
		_ = w.Close()
		if err != nil {
			_ = os.Remove(tmp)
		}
	}()
	if _, err = io.Copy(w, reader); err != nil {
		return
	}
	if err = w.Sync(); err != nil {
		return
	}
	if err = w.Close(); err != nil {
		return
	}
	if s.SaveErrIfExists {
		err = linkIntoPlace(tmp, image)
		return
	}
	return renameFile(tmp, image)
}

// createTemp creates a dot-prefixed temp file in dir, which the storage
// blacklist already ignores.
func createTemp(dir string, perm os.FileMode) (*os.File, error) {
	for i := 0; i < 100; i++ {
		var rnd [9]byte
		if _, err := rand.Read(rnd[:]); err != nil {
			return nil, err
		}
		name := filepath.Join(dir, ".imagor-tmp-"+base64.RawURLEncoding.EncodeToString(rnd[:]))
		f, err := os.OpenFile(name, os.O_WRONLY|os.O_CREATE|os.O_EXCL, perm)
		if err == nil {
			return f, nil
		}
		if !errors.Is(err, fs.ErrExist) {
			return nil, err
		}
	}
	return nil, fmt.Errorf("filestorage: cannot create a temporary file in %s", dir)
}

// linkIntoPlace adds the target only if it has none, by link; the fallback is a
// check and a rename, which is not exclusive.
func linkIntoPlace(tmp, image string) error {
	err := os.Link(tmp, image)
	if err == nil {
		_ = os.Remove(tmp)
		return nil
	}
	if errors.Is(err, fs.ErrExist) {
		return err
	}
	if _, statErr := os.Stat(image); statErr == nil {
		return fs.ErrExist
	}
	return renameFile(tmp, image)
}

// Delete implements imagor.Storage interface
func (s *FileStorage) Delete(_ context.Context, image string) error {
	image, ok := s.Path(image)
	if !ok {
		return imagor.ErrInvalid
	}
	return os.Remove(image)
}

// Stat implements imagor.Storage interface
func (s *FileStorage) Stat(_ context.Context, image string) (stat *imagor.Stat, err error) {
	image, ok := s.Path(image)
	if !ok {
		return nil, imagor.ErrInvalid
	}
	osStat, err := os.Stat(image)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, imagor.ErrNotFound
		}
		return nil, err
	}
	size := osStat.Size()
	modTime := osStat.ModTime()
	return &imagor.Stat{
		Size:         size,
		ModifiedTime: modTime,
	}, nil
}
