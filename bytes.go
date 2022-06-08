package imagor

import (
	"bytes"
	"io/ioutil"
	"net/http"
	"sync"
	"time"
)

type BytesType int

const (
	BytesTypeUnknown BytesType = iota
	BytesTypeEmpty
	BytesTypeJPEG
	BytesTypePNG
	BytesTypeGIF
	BytesTypeWEBP
	BytesTypeAVIF
)

// Bytes abstraction for file path, bytes data and meta attributes
type Bytes struct {
	path  string
	buf   []byte
	once  sync.Once
	once2 sync.Once
	err   error

	bytesType   BytesType
	contentType string

	Meta *Meta
}

// Stat image attributes
type Stat struct {
	ModifiedTime time.Time
	Size         int64
}

// Meta image attributes
type Meta struct {
	Format      string `json:"format"`
	ContentType string `json:"content_type"`
	Width       int    `json:"width"`
	Height      int    `json:"height"`
	Orientation int    `json:"orientation"`
}

func NewBytesFilePath(filepath string) *Bytes {
	return &Bytes{path: filepath, bytesType: BytesTypeUnknown}
}

func NewBytes(bytes []byte) *Bytes {
	return &Bytes{buf: bytes, bytesType: BytesTypeUnknown}
}

func NewEmptyBytes() *Bytes {
	return &Bytes{buf: []byte{}, bytesType: BytesTypeEmpty}
}

var jpegHeader = []byte("\xFF\xD8\xFF")
var gifHeader = []byte("\x47\x49\x46")
var webpHeader = []byte("\x57\x45\x42\x50")
var pngHeader = []byte("\x89\x50\x4E\x47")

// https://github.com/strukturag/libheif/blob/master/libheif/heif.cc
var ftyp = []byte("ftyp")
var avif = []byte("avif")

func (b *Bytes) readAllOnce() {
	b.once.Do(func() {
		if b.bytesType == BytesTypeEmpty {
			return
		}
		if len(b.buf) == 0 {
			if b.path != "" {
				b.buf, b.err = ioutil.ReadFile(b.path)
			}
			if len(b.buf) == 0 && b.err == nil {
				b.bytesType = BytesTypeEmpty
				return
			}
		}
		if len(b.buf) > 24 {
			if bytes.Equal(b.buf[:3], jpegHeader) {
				b.bytesType = BytesTypeJPEG
			} else if bytes.Equal(b.buf[:4], pngHeader) {
				b.bytesType = BytesTypePNG
			} else if bytes.Equal(b.buf[:3], gifHeader) {
				b.bytesType = BytesTypeGIF
			} else if bytes.Equal(b.buf[8:12], webpHeader) {
				b.bytesType = BytesTypeWEBP
			} else if bytes.Equal(b.buf[4:8], ftyp) && bytes.Equal(b.buf[8:12], avif) {
				b.bytesType = BytesTypeAVIF
			}
		}
	})
}

func (b *Bytes) IsEmpty() bool {
	b.readAllOnce()
	return b.bytesType == BytesTypeEmpty
}

func (b *Bytes) SupportsAnimation() bool {
	b.readAllOnce()
	return b.bytesType == BytesTypeGIF || b.bytesType == BytesTypeWEBP
}

func (b *Bytes) BytesType() BytesType {
	b.readAllOnce()
	return b.bytesType
}

func (b *Bytes) ContentType() string {
	if b.Meta != nil && b.Meta.ContentType != "" {
		return b.Meta.ContentType
	}
	b.readAllOnce()
	b.once2.Do(func() {
		b.contentType = "application/octet-stream"
		switch b.BytesType() {
		case BytesTypeJPEG:
			b.contentType = "image/jpeg"
		case BytesTypePNG:
			b.contentType = "image/png"
		case BytesTypeGIF:
			b.contentType = "image/gif"
		case BytesTypeWEBP:
			b.contentType = "image/webp"
		case BytesTypeAVIF:
			b.contentType = "image/avif"
		default:
			b.contentType = http.DetectContentType(b.buf)
		}
	})
	return b.contentType
}

func (b *Bytes) ReadAll() ([]byte, error) {
	b.readAllOnce()
	return b.buf, b.err
}

func (b *Bytes) Err() error {
	b.readAllOnce()
	return b.err
}

func isEmpty(f *Bytes) bool {
	return f == nil || f.IsEmpty()
}
