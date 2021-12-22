package imagor

import (
	"io/ioutil"
)

type File struct {
	Path string
	Meta *Meta
	buf  []byte
}

type Meta struct {
	Format      string `json:"format"`
	ContentType string `json:"content_type"`
	Width       int    `json:"width"`
	Height      int    `json:"height"`
	Orientation int    `json:"orientation"`
}

func NewFilePath(filepath string) *File {
	return &File{Path: filepath}
}

func NewFileBytes(bytes []byte) *File {
	return &File{buf: bytes}
}

func NewFileBytesWithMeta(bytes []byte, meta *Meta) *File {
	return &File{buf: bytes, Meta: meta}
}

func (f *File) IsEmpty() bool {
	return f.Path == "" && len(f.buf) == 0
}

func (f *File) HasPath() bool {
	return f.Path != ""
}

func (f *File) Bytes() ([]byte, error) {
	if len(f.buf) > 0 {
		return f.buf, nil
	}
	if f.Path != "" {
		return ioutil.ReadFile(f.Path)
	}
	return nil, ErrNotFound
}

func IsFileEmpty(f *File) bool {
	return f == nil || f.IsEmpty()
}
