package filestore

import (
	"context"
	"github.com/cshum/imagor"
	"io"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"
)

type fileStore struct {
	Root    string
	BaseURI string
	once    sync.Once
}

func New(root string, options ...Option) *fileStore {
	s := &fileStore{
		Root:    root,
		BaseURI: "/",
	}
	for _, options := range options {
		options(s)
	}
	return s
}

func (s *fileStore) Path(image string) (string, bool) {
	image = "/" + strings.TrimPrefix(path.Clean(image), "/")
	if !strings.HasPrefix(image, s.BaseURI) {
		return "", false
	}
	return filepath.Join(s.Root, strings.TrimPrefix(image, s.BaseURI)), true
}

func (s *fileStore) Load(_ *http.Request, image string) ([]byte, error) {
	image, ok := s.Path(image)
	if !ok {
		return nil, imagor.ErrPass
	}
	r, err := os.Open(image)
	if os.IsNotExist(err) {
		return nil, imagor.ErrPass
	}
	return io.ReadAll(r)
}

func (s *fileStore) Store(_ context.Context, image string, buf []byte) (err error) {
	s.once.Do(func() {
		_, err = os.Stat(s.Root)
	})
	if err != nil {
		return
	}
	image, ok := s.Path(image)
	if !ok {
		return imagor.ErrPass
	}
	if err = os.MkdirAll(filepath.Dir(image), 0755); err != nil {
		return
	}
	w, err := os.Create(image)
	if err != nil {
		return
	}
	defer w.Close()
	if _, err = w.Write(buf); err != nil {
		return
	}
	return
}
