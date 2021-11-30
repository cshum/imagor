package filestore

import (
	"context"
	"github.com/cshum/imagor"
	"io"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"
)

type fileStore struct {
	Root       string
	BasePath   string
	Blacklists []*regexp.Regexp
}

func New(root string, options ...Option) *fileStore {
	s := &fileStore{
		Root:       root,
		BasePath:   "/",
		Blacklists: []*regexp.Regexp{regexp.MustCompile("/\\.")},
	}
	for _, option := range options {
		option(s)
	}
	return s
}

func (s *fileStore) Path(image string) (string, bool) {
	image = "/" + strings.TrimPrefix(path.Clean(
		strings.ReplaceAll(image, ":/", "%3A"),
	), "/")
	for _, blacklist := range s.Blacklists {
		if blacklist.MatchString(image) {
			return "", false
		}
	}
	baseURI := "/" + strings.Trim(s.BasePath, "/")
	if baseURI != "/" {
		baseURI += "/"
	}
	if !strings.HasPrefix(image, baseURI) {
		return "", false
	}
	return filepath.Join(s.Root, strings.TrimPrefix(image, baseURI)), true
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
	if _, err = os.Stat(s.Root); err != nil {
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
