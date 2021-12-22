package filestore

import (
	"context"
	"github.com/cshum/imagor"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"
)

var dotFileRegex = regexp.MustCompile("/\\.")

type FileStore struct {
	BaseDir         string
	PathPrefix      string
	Blacklists      []*regexp.Regexp
	MkdirPermission os.FileMode
	WritePermission os.FileMode
}

func New(baseDir string, options ...Option) *FileStore {
	s := &FileStore{
		BaseDir:         baseDir,
		PathPrefix:      "/",
		Blacklists:      []*regexp.Regexp{dotFileRegex},
		MkdirPermission: 0755,
		WritePermission: 0666,
	}
	for _, option := range options {
		option(s)
	}
	return s
}

func (s *FileStore) Path(image string) (string, bool) {
	image = "/" + strings.TrimPrefix(path.Clean(
		strings.ReplaceAll(image, ":/", "%3A"),
	), "/")
	for _, blacklist := range s.Blacklists {
		if blacklist.MatchString(image) {
			return "", false
		}
	}
	if !strings.HasPrefix(image, s.PathPrefix) {
		return "", false
	}
	return filepath.Join(s.BaseDir, strings.TrimPrefix(image, s.PathPrefix)), true
}

func (s *FileStore) Load(_ *http.Request, image string) (*imagor.File, error) {
	image, ok := s.Path(image)
	if !ok {
		return nil, imagor.ErrPass
	}
	_, err := os.Stat(image)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, imagor.ErrNotFound
		}
		return nil, err
	}
	return imagor.NewFilePath(image), nil
}

func (s *FileStore) Save(_ context.Context, image string, file *imagor.File) (err error) {
	image, ok := s.Path(image)
	if !ok {
		return imagor.ErrPass
	}
	if imagor.IsFileEmpty(file) {
		return imagor.ErrNotFound
	}
	if err = os.MkdirAll(filepath.Dir(image), s.MkdirPermission); err != nil {
		return
	}
	buf, err := file.Bytes()
	if err != nil {
		return err
	}
	w, err := os.OpenFile(image, os.O_RDWR|os.O_CREATE|os.O_TRUNC, s.WritePermission)
	if err != nil {
		return
	}
	defer w.Close()
	if _, err = w.Write(buf); err != nil {
		return
	}
	return
}
