package imagor

import (
	"io/ioutil"
)

// Blob abstraction for file path, bytes data and meta attributes
type Blob struct {
	FilePath string
	Meta     *Meta
	buf      []byte
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

func (f *Blob) ReadAll() ([]byte, error) {
	if len(f.buf) > 0 {
		return f.buf, nil
	}
	if f.FilePath != "" {
		return ioutil.ReadFile(f.FilePath)
	}
	return nil, ErrNotFound
}

func IsFileEmpty(f *Blob) bool {
	return f == nil || f.IsEmpty()
}
