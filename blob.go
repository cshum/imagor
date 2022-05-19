package imagor

import (
	"bytes"
	"io/ioutil"
	"sync"
)

// Blob abstraction for file path, bytes data and meta attributes
type Blob struct {
	path string
	buf  []byte
	once sync.Once
	err  error

	supportsAnimation bool
	isPNG             bool
	isAVIF            bool

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

var jpegHeader = []byte("\xFF\xD8\xFF")
var gifHeader = []byte("\x47\x49\x46")
var webpHeader = []byte("\x57\x45\x42\x50")
var pngHeader = []byte("\x89\x50\x4E\x47")

// https://github.com/strukturag/libheif/blob/master/libheif/heif.cc
var ftyp = []byte("ftyp")
var avif = []byte("avif")

//var heic = []byte("heic")
//var mif1 = []byte("mif1")
//var msf1 = []byte("msf1")

func (b *Blob) readAllOnce() {
	b.once.Do(func() {
		if len(b.buf) == 0 {
			if b.path != "" {
				b.buf, b.err = ioutil.ReadFile(b.path)
			}
			if len(b.buf) == 0 && b.err == nil {
				b.buf = nil
				b.err = ErrNotFound
				return
			}
		}
		if len(b.buf) > 24 && !bytes.HasPrefix(b.buf, jpegHeader) {
			if bytes.HasPrefix(b.buf, gifHeader) || bytes.Equal(b.buf[8:12], webpHeader) {
				b.supportsAnimation = true
			} else if bytes.HasPrefix(b.buf, pngHeader) {
				b.isPNG = true
			} else if bytes.Equal(b.buf[4:8], ftyp) && bytes.Equal(b.buf[8:12], avif) {
				b.isAVIF = true
			}
		}
	})
}

func (b *Blob) IsEmpty() bool {
	b.readAllOnce()
	return b.path == "" && len(b.buf) == 0
}

func (b *Blob) SupportsAnimation() bool {
	b.readAllOnce()
	return b.supportsAnimation
}

func (b *Blob) IsPNG() bool {
	b.readAllOnce()
	return b.isPNG
}

func (b *Blob) IsAVIF() bool {
	b.readAllOnce()
	return b.isAVIF
}

func (b *Blob) ReadAll() ([]byte, error) {
	b.readAllOnce()
	return b.buf, b.err
}

func IsBlobEmpty(f *Blob) bool {
	return f == nil || f.IsEmpty()
}
