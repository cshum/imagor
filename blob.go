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

func (b *Blob) readAllOnce() {
	b.once.Do(func() {
		if len(b.buf) > 0 {
			return
		}
		if b.path != "" {
			b.buf, b.err = ioutil.ReadFile(b.path)
		}
		if len(b.buf) == 0 && b.err == nil {
			b.buf = nil
			b.err = ErrNotFound
			return
		}
	})
}

func (b *Blob) IsEmpty() bool {
	return b.path == "" && len(b.buf) == 0
}

func (b *Blob) ReadAll() ([]byte, error) {
	b.readAllOnce()
	return b.buf, b.err
}

func IsBlobEmpty(f *Blob) bool {
	return f == nil || f.IsEmpty()
}
