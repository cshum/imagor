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

var dotFileRegex = regexp.MustCompile("/\\.")

type fileStore struct {
	BaseDir    string
	BaseURI    string
	Blacklists []*regexp.Regexp
}

func New(baseDir string, options ...Option) *fileStore {
	s := &fileStore{
		BaseDir:    baseDir,
		BaseURI:    "/",
		Blacklists: []*regexp.Regexp{dotFileRegex},
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
	if !strings.HasPrefix(image, s.BaseURI) {
		return "", false
	}
	return filepath.Join(s.BaseDir, strings.TrimPrefix(image, s.BaseURI)), true
}

func (s *fileStore) Load(_ *http.Request, image string) ([]byte, error) {
	image, ok := s.Path(image)
	if !ok {
		return nil, imagor.ErrPass
	}
	r, err := os.Open(image)
	if os.IsNotExist(err) {
		return nil, imagor.ErrNotFound
	}
	return io.ReadAll(r)
}

func (s *fileStore) Save(_ context.Context, image string, buf []byte) (err error) {
	if _, err = os.Stat(s.BaseDir); err != nil {
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
