package imagor

import (
	"bytes"
	"io/ioutil"
	"net/http"
	"sync"
	"time"
)

type BlobType int

const (
	BlobTypeUnknown BlobType = iota
	BlobTypeEmpty
	BlobTypeJPEG
	BlobTypePNG
	BlobTypeGIF
	BlobTypeWEBP
	BlobTypeAVIF
	BlobTypeTIFF
)

// Blob abstraction for file path, bytes data and meta attributes
type Blob struct {
	path  string
	buf   []byte
	once  sync.Once
	once2 sync.Once
	err   error

	blobType    BlobType
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
	Pages       int    `json:"pages"`
}

func NewBlobFromPath(filepath string) *Blob {
	return &Blob{path: filepath, blobType: BlobTypeUnknown}
}

func NewBlobFromBytes(bytes []byte) *Blob {
	return &Blob{buf: bytes, blobType: BlobTypeUnknown}
}

func NewEmptyBlob() *Blob {
	return &Blob{buf: []byte{}, blobType: BlobTypeEmpty}
}

var jpegHeader = []byte("\xFF\xD8\xFF")
var gifHeader = []byte("\x47\x49\x46")
var webpHeader = []byte("\x57\x45\x42\x50")
var pngHeader = []byte("\x89\x50\x4E\x47")

// https://github.com/strukturag/libheif/blob/master/libheif/heif.cc
var ftyp = []byte("ftyp")
var avif = []byte("avif")

var tifII = []byte("\x49\x49\x2A\x00")
var tifMM = []byte("\x4D\x4D\x00\x2A")

func (b *Blob) readAllOnce() {
	b.once.Do(func() {
		if b.blobType == BlobTypeEmpty {
			return
		}
		if len(b.buf) == 0 {
			if b.path != "" {
				b.buf, b.err = ioutil.ReadFile(b.path)
			}
			if len(b.buf) == 0 && b.err == nil {
				b.blobType = BlobTypeEmpty
				return
			}
		}
		if len(b.buf) > 24 {
			if bytes.Equal(b.buf[:3], jpegHeader) {
				b.blobType = BlobTypeJPEG
			} else if bytes.Equal(b.buf[:4], pngHeader) {
				b.blobType = BlobTypePNG
			} else if bytes.Equal(b.buf[:3], gifHeader) {
				b.blobType = BlobTypeGIF
			} else if bytes.Equal(b.buf[8:12], webpHeader) {
				b.blobType = BlobTypeWEBP
			} else if bytes.Equal(b.buf[4:8], ftyp) && bytes.Equal(b.buf[8:12], avif) {
				b.blobType = BlobTypeAVIF
			} else if bytes.Equal(b.buf[:4], tifII) || bytes.Equal(b.buf[:4], tifMM) {
				b.blobType = BlobTypeTIFF
			}
		}
	})
}

func (b *Blob) IsEmpty() bool {
	b.readAllOnce()
	return b.blobType == BlobTypeEmpty
}

func (b *Blob) SupportsAnimation() bool {
	b.readAllOnce()
	return b.blobType == BlobTypeGIF || b.blobType == BlobTypeWEBP
}

func (b *Blob) BytesType() BlobType {
	b.readAllOnce()
	return b.blobType
}

func (b *Blob) ContentType() string {
	if b.Meta != nil && b.Meta.ContentType != "" {
		return b.Meta.ContentType
	}
	b.readAllOnce()
	b.once2.Do(func() {
		b.contentType = "application/octet-stream"
		switch b.BytesType() {
		case BlobTypeJPEG:
			b.contentType = "image/jpeg"
		case BlobTypePNG:
			b.contentType = "image/png"
		case BlobTypeGIF:
			b.contentType = "image/gif"
		case BlobTypeWEBP:
			b.contentType = "image/webp"
		case BlobTypeAVIF:
			b.contentType = "image/avif"
		case BlobTypeTIFF:
			b.contentType = "image/tiff"
		default:
			b.contentType = http.DetectContentType(b.buf)
		}
	})
	return b.contentType
}

func (b *Blob) ReadAll() ([]byte, error) {
	b.readAllOnce()
	return b.buf, b.err
}

func (b *Blob) Err() error {
	b.readAllOnce()
	return b.err
}

func isEmpty(f *Blob) bool {
	return f == nil || f.IsEmpty()
}
