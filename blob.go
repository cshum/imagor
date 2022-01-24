package imagor

import (
	"io/ioutil"
	"sync"
)

// Blob abstraction for file path, bytes data and meta attributes
type Blob struct {
	path string
	buf  []byte
	once sync.Once
	err  error

	Meta *Meta
}

// Meta image attributes
type Meta struct {
	Format      string `json:"format"`
	ContentType string `json:"content_type"`
	Width       int    `json:"width"`
	Height      int    `json:"height"`
	Orientation int    `json:"orientation"`
}

func NewBlobFilePath(filepath string) *Blob {
	return &Blob{path: filepath}
}

func NewBlobBytes(bytes []byte) *Blob {
	return &Blob{buf: bytes}
}

func NewBlobBytesWithMeta(bytes []byte, meta *Meta) *Blob {
	return &Blob{buf: bytes, Meta: meta}
}

func (f *Blob) IsEmpty() bool {
	return f.path == "" && len(f.buf) == 0
}

func (f *Blob) ReadAll() ([]byte, error) {
	f.once.Do(func() {
		if len(f.buf) > 0 {
			return
		}
		if f.path != "" {
			f.buf, f.err = ioutil.ReadFile(f.path)
		}
		if len(f.buf) == 0 && f.err == nil {
			f.buf = nil
			f.err = ErrNotFound
		}
	})
	return f.buf, f.err
}

func IsFileEmpty(f *Blob) bool {
	return f == nil || f.IsEmpty()
}
