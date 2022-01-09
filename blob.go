package imagor

import (
	"io/ioutil"
	"sync"
)

// Blob abstraction for file path, bytes data and meta attributes
type Blob struct {
	FilePath string
	Meta     *Meta
	buf      []byte

	rw sync.RWMutex
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
	return &Blob{FilePath: filepath}
}

func NewBlobBytes(bytes []byte) *Blob {
	return &Blob{buf: bytes}
}

func NewBlobBytesWithMeta(bytes []byte, meta *Meta) *Blob {
	return &Blob{buf: bytes, Meta: meta}
}

func (f *Blob) IsEmpty() bool {
	return f.FilePath == "" && len(f.buf) == 0
}

func (f *Blob) HasFilePath() bool {
	return f.FilePath != ""
}

func (f *Blob) setBuf(buf []byte) {
	f.rw.Lock()
	f.buf = buf
	f.rw.Unlock()
}

func (f *Blob) getBuf() []byte {
	f.rw.RLock()
	defer f.rw.RUnlock()
	return f.buf
}

func (f *Blob) ReadAll() ([]byte, error) {
	buf := f.getBuf()
	if len(buf) > 0 {
		return buf, nil
	}
	if f.FilePath != "" {
		buf, err := ioutil.ReadFile(f.FilePath)
		if err != nil {
			return buf, err
		}
		f.setBuf(buf)
		return buf, nil
	}
	return nil, ErrNotFound
}

func IsFileEmpty(f *Blob) bool {
	return f == nil || f.IsEmpty()
}
