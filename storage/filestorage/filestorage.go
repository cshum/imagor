package filestorage

import (
	"context"
	"encoding/json"
	"github.com/cshum/imagor"
	"github.com/cshum/imagor/imagorpath"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

var dotFileRegex = regexp.MustCompile("/\\.")

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
	return filepath.Join(s.BaseDir, strings.TrimPrefix(image, s.PathPrefix)), true
}

func (s *FileStorage) Get(_ *http.Request, image string) (*imagor.Bytes, error) {
	image, ok := s.Path(image)
	if !ok {
		return nil, imagor.ErrPass
	}
	stats, err := os.Stat(image)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, imagor.ErrNotFound
		}
		return nil, err
	}
	if s.Expiration > 0 && time.Now().Sub(stats.ModTime()) > s.Expiration {
		return nil, imagor.ErrExpired
	}
	return imagor.NewBytesFilePath(image), nil
}

func (s *FileStorage) Put(_ context.Context, image string, blob *imagor.Bytes) (err error) {
	image, ok := s.Path(image)
	if !ok {
		return imagor.ErrPass
	}
	if err = os.MkdirAll(filepath.Dir(image), s.MkdirPermission); err != nil {
		return
	}
	buf, err := blob.ReadAll()
	if err != nil {
		return err
	}
	flag := os.O_RDWR | os.O_CREATE | os.O_TRUNC
	if s.SaveErrIfExists {
		flag = os.O_RDWR | os.O_CREATE | os.O_EXCL
	}
	w, err := os.OpenFile(image, flag, s.WritePermission)
	if err != nil {
		return
	}
	defer w.Close()
	if _, err = w.Write(buf); err != nil {
		return
	}
	if blob.Meta != nil {
		if buf, _ := json.Marshal(blob.Meta); len(buf) > 0 {
			w, err := os.OpenFile(image+".meta.json", flag, s.WritePermission)
			if err != nil {
				return err
			}
			defer w.Close()
			if _, err = w.Write(buf); err != nil {
				return err
			}
		}
	}
	return
}

func (s *FileStorage) Stat(_ context.Context, image string) (stat *imagor.Stat, err error) {
	image, ok := s.Path(image)
	if !ok {
		return nil, imagor.ErrPass
	}
	stats, err := os.Stat(image)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, imagor.ErrNotFound
		}
		return nil, err
	}
	return &imagor.Stat{
		Size:         stats.Size(),
		ModifiedTime: stats.ModTime(),
	}, nil
}
