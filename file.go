package imagor

import (
	"io/ioutil"
)

type File struct {
	path string
	buf  []byte
}

func NewFilePath(filepath string) *File {
	return &File{path: filepath}
}

func NewFileBytes(bytes []byte) *File {
	return &File{buf: bytes}
}

func (f *File) IsEmpty() bool {
	return f.path == "" && len(f.buf) == 0
}

func (f *File) HasPath() bool {
	return f.path != ""
}

func (f *File) Path() string {
	return f.path
}

func (f *File) Bytes() ([]byte, error) {
	if len(f.buf) > 0 {
		return f.buf, nil
	}
	if f.path != "" {
		return ioutil.ReadFile(f.path)
	}
	return nil, ErrNotFound
}

func IsFileEmpty(f *File) bool {
	return f == nil || f.IsEmpty()
}
