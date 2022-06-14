package imagor

import (
	"bufio"
	"bytes"
	"io"
	"net/http"
	"os"
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

type bufioReadCloser struct {
	*bufio.Reader
	io.Closer
}

// Blob abstraction for file path, bytes data and meta attributes
type Blob struct {
	newReader  func() (io.ReadCloser, error)
	peekReader *bufioReadCloser
	once       sync.Once
	once2      sync.Once
	err        error

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
	return &Blob{newReader: func() (io.ReadCloser, error) {
		return os.Open(filepath)
	}, blobType: BlobTypeUnknown}
}

func NewBlobFromBytes(buf []byte) *Blob {
	return &Blob{newReader: func() (io.ReadCloser, error) {
		return io.NopCloser(bytes.NewReader(buf)), nil
	}, blobType: BlobTypeUnknown}
}

func NewBlobFromReadCloser(newReader func() (io.ReadCloser, error)) *Blob {
	return &Blob{newReader: newReader, blobType: BlobTypeUnknown}
}

func NewEmptyBlob() *Blob {
	return &Blob{blobType: BlobTypeEmpty}
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

func (b *Blob) peekOnce() {
	b.once.Do(func() {
		if b.blobType == BlobTypeEmpty || b.newReader == nil {
			b.blobType = BlobTypeEmpty
			return
		}
		reader, err := b.newReader()
		if err != nil {
			b.err = err
			return
		}
		b.peekReader = &bufioReadCloser{bufio.NewReader(reader), reader}
		buf := make([]byte, 0, 512)
		if buf, err = b.peekReader.Peek(512); err != nil && err != bufio.ErrBufferFull && err != io.EOF {
			b.err = err
			return
		}
		if len(buf) == 0 && b.err == nil {
			b.blobType = BlobTypeEmpty
			return
		}
		if len(buf) > 24 {
			if bytes.Equal(buf[:3], jpegHeader) {
				b.blobType = BlobTypeJPEG
			} else if bytes.Equal(buf[:4], pngHeader) {
				b.blobType = BlobTypePNG
			} else if bytes.Equal(buf[:3], gifHeader) {
				b.blobType = BlobTypeGIF
			} else if bytes.Equal(buf[8:12], webpHeader) {
				b.blobType = BlobTypeWEBP
			} else if bytes.Equal(buf[4:8], ftyp) && bytes.Equal(buf[8:12], avif) {
				b.blobType = BlobTypeAVIF
			} else if bytes.Equal(buf[:4], tifII) || bytes.Equal(buf[:4], tifMM) {
				b.blobType = BlobTypeTIFF
			}
			b.contentType = "application/octet-stream"
			switch b.blobType {
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
				b.contentType = http.DetectContentType(buf)
			}
		}
	})
}

func (b *Blob) IsEmpty() bool {
	b.peekOnce()
	return b.blobType == BlobTypeEmpty
}

func (b *Blob) SupportsAnimation() bool {
	b.peekOnce()
	return b.blobType == BlobTypeGIF || b.blobType == BlobTypeWEBP
}

func (b *Blob) BlobType() BlobType {
	b.peekOnce()
	return b.blobType
}

func (b *Blob) ContentType() string {
	if b.Meta != nil && b.Meta.ContentType != "" {
		return b.Meta.ContentType
	}
	b.peekOnce()
	return b.contentType
}

func (b *Blob) NewReader() (reader io.ReadCloser, err error) {
	b.once2.Do(func() {
		b.peekOnce()
		if b.err != nil {
			err = b.err
			return
		}
		if b.peekReader != nil {
			reader = b.peekReader
		}
	})
	if reader != nil {
		return
	}
	return b.newReader()
}

func (b *Blob) ReadAll() ([]byte, error) {
	b.peekOnce()
	if b.blobType == BlobTypeEmpty || b.err != nil {
		return nil, b.err
	}
	reader, err := b.NewReader()
	if err != nil {
		return nil, err
	}
	defer reader.Close()
	return io.ReadAll(reader)
}

func (b *Blob) Err() error {
	b.peekOnce()
	return b.err
}

func isEmpty(f *Blob) bool {
	return f == nil || f.IsEmpty()
}
