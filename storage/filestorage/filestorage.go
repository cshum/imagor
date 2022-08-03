package filestorage

import (
	"context"
	"encoding/json"
	"github.com/cshum/imagor"
	"github.com/cshum/imagor/imagorpath"
	"io"
	"io/ioutil"
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
	path := "/" + imagorpath.Normalize(image, s.safeChars)
	for _, blacklist := range s.Blacklists {
		if blacklist.MatchString(path) {
			return "", false
		}
	}
	if !strings.HasPrefix(path, s.PathPrefix) {
		return "", false
	}
	return filepath.Join(s.BaseDir, strings.TrimPrefix(path, s.PathPrefix)), true
}

func (s *FileStorage) Get(_ *http.Request, image string) (*imagor.Blob, error) {
	path, ok := s.Path(image)
	if !ok {
		return nil, imagor.ErrInvalid
	}
	stats, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, imagor.ErrNotFound
		}
		return nil, err
	}
	if s.Expiration > 0 && time.Now().Sub(stats.ModTime()) > s.Expiration {
		return nil, imagor.ErrExpired
	}
	return imagor.NewBlob(func() (io.ReadCloser, int64, error) {
		r, err := os.Open(path)
		return r, stats.Size(), err
	}), nil
}

func (s *FileStorage) Put(_ context.Context, image string, blob *imagor.Blob) (err error) {
	path, ok := s.Path(image)
	if !ok {
		return imagor.ErrInvalid
	}
	if err = os.MkdirAll(filepath.Dir(path), s.MkdirPermission); err != nil {
		return
	}
	reader, _, err := blob.NewReader()
	if err != nil {
		return err
	}
	defer func() {
		_ = reader.Close()
	}()
	flag := os.O_RDWR | os.O_CREATE | os.O_TRUNC
	if s.SaveErrIfExists {
		flag = os.O_RDWR | os.O_CREATE | os.O_EXCL
	}
	w, err := os.OpenFile(path, flag, s.WritePermission)
	if err != nil {
		return
	}
	defer func() {
		_ = w.Close()
	}()
	if _, err = io.Copy(w, reader); err != nil {
		return
	}
	if blob.Meta != nil {
		if buf, _ := json.Marshal(blob.Meta); len(buf) > 0 {
			w, err := os.OpenFile(path+".meta.json", flag, s.WritePermission)
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

func (s *FileStorage) Delete(_ context.Context, image string) error {
	path, ok := s.Path(image)
	if !ok {
		return imagor.ErrInvalid
	}
	return os.Remove(path)
}

func (s *FileStorage) Stat(_ context.Context, image string) (stat *imagor.Stat, err error) {
	path, ok := s.Path(image)
	if !ok {
		return nil, imagor.ErrInvalid
	}
	stats, err := os.Stat(path)
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

func (s *FileStorage) Meta(_ context.Context, image string) (*imagor.Meta, error) {
	path, ok := s.Path(image)
	if !ok {
		return nil, imagor.ErrInvalid
	}
	key := path + ".meta.json"

	if s.Expiration > 0 {
		stats, err := os.Stat(key)
		if err != nil {
			if os.IsNotExist(err) {
				return nil, imagor.ErrNotFound
			}
			return nil, err
		}
		if time.Now().Sub(stats.ModTime()) > s.Expiration {
			return nil, imagor.ErrExpired
		}
	}
	buf, err := ioutil.ReadFile(key)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, imagor.ErrNotFound
		}
		return nil, err
	}
	meta := &imagor.Meta{}
	if err := json.Unmarshal(buf, meta); err != nil {
		return nil, err
	}
	return meta, nil
}
