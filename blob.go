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

const maxBodySize = int64(100 << 20) // 100MB

const (
	BlobTypeUnknown BlobType = iota
	BlobTypeEmpty
	BlobTypeJPEG
	BlobTypePNG
	BlobTypeGIF
	BlobTypeWEBP
	BlobTypeAVIF
	BlobTypeHEIF
	BlobTypeTIFF
)

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

type Blob struct {
	newReader  func() (r io.ReadCloser, size int64, err error)
	peekReader *peekReaderCloser
	once       sync.Once
	onceReader sync.Once
	buf        []byte
	err        error
	size       int64

	blobType    BlobType
	contentType string

	Meta *Meta
}

func NewBlob(newReader func() (reader io.ReadCloser, size int64, err error)) *Blob {
	return &Blob{newReader: newReader}
}

func NewBlobFromFile(filepath string, checks ...func(stats os.FileInfo) error) *Blob {
	return NewBlob(func() (io.ReadCloser, int64, error) {
		stats, err := os.Stat(filepath)
		if err != nil {
			if os.IsNotExist(err) {
				err = ErrNotFound
			}
			return nil, 0, err
		}
		for _, check := range checks {
			if err := check(stats); err != nil {
				return nil, stats.Size(), err
			}
		}
		reader, err := os.Open(filepath)
		return reader, stats.Size(), err
	})
}

func NewBlobFromBytes(buf []byte) *Blob {
	size := int64(len(buf))
	return NewBlob(func() (io.ReadCloser, int64, error) {
		return io.NopCloser(bytes.NewReader(buf)), size, nil
	})
}

func NewEmptyBlob() *Blob {
	return &Blob{}
}

var jpegHeader = []byte("\xFF\xD8\xFF")
var gifHeader = []byte("\x47\x49\x46")
var webpHeader = []byte("\x57\x45\x42\x50")
var pngHeader = []byte("\x89\x50\x4E\x47")

// https://github.com/strukturag/libheif/blob/master/libheif/heif.cc
var ftyp = []byte("ftyp")
var heic = []byte("heic")
var mif1 = []byte("mif1")
var msf1 = []byte("msf1")
var avif = []byte("avif")

var tifII = []byte("\x49\x49\x2A\x00")
var tifMM = []byte("\x4D\x4D\x00\x2A")

type peekReaderCloser struct {
	*bufio.Reader
	io.Closer
}

func newEmptyReader() (io.ReadCloser, int64, error) {
	return io.NopCloser(bytes.NewReader(nil)), 0, nil
}

func (b *Blob) init() {
	b.once.Do(func() {
		b.blobType = BlobTypeUnknown
		b.contentType = "application/octet-stream"

		if b.newReader == nil {
			b.blobType = BlobTypeEmpty
			b.newReader = newEmptyReader
			return
		}
		reader, size, err := b.newReader()
		if err != nil {
			b.err = err
		}
		if reader == nil {
			return
		}
		b.size = size
		if reader != nil && size > 0 && size < maxBodySize && err == nil {
			// use fan-out reader if buf size < 10mb
			// otherwise create new readers
			newReader := FanoutReader(reader, int(size))
			b.newReader = func() (io.ReadCloser, int64, error) {
				return newReader(), size, nil
			}
			reader = newReader()
		}
		b.peekReader = &peekReaderCloser{
			Reader: bufio.NewReader(reader),
			Closer: reader,
		}
		// peek first 512 bytes for type sniffing
		b.buf, err = b.peekReader.Peek(512)
		if len(b.buf) == 0 {
			b.blobType = BlobTypeEmpty
		}
		if err != nil && err != bufio.ErrBufferFull && err != io.EOF {
			if b.err == nil {
				b.err = err
			}
			return
		}
		if b.blobType != BlobTypeEmpty && len(b.buf) > 24 {
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
			} else if bytes.Equal(b.buf[4:8], ftyp) && (bytes.Equal(b.buf[8:12], heic) ||
				bytes.Equal(b.buf[8:12], mif1) ||
				bytes.Equal(b.buf[8:12], msf1)) {
				b.blobType = BlobTypeHEIF
			} else if bytes.Equal(b.buf[:4], tifII) || bytes.Equal(b.buf[:4], tifMM) {
				b.blobType = BlobTypeTIFF
			}
		}
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
		case BlobTypeHEIF:
			b.contentType = "image/heif"
		case BlobTypeTIFF:
			b.contentType = "image/tiff"
		default:
			b.contentType = http.DetectContentType(b.buf)
		}
	})
}

func (b *Blob) IsEmpty() bool {
	b.init()
	return b.blobType == BlobTypeEmpty
}

func (b *Blob) SupportsAnimation() bool {
	b.init()
	return b.blobType == BlobTypeGIF || b.blobType == BlobTypeWEBP
}

func (b *Blob) BlobType() BlobType {
	b.init()
	return b.blobType
}

func (b *Blob) Sniff() []byte {
	b.init()
	return b.buf
}

func (b *Blob) ContentType() string {
	if b.Meta != nil && b.Meta.ContentType != "" {
		return b.Meta.ContentType
	}
	b.init()
	return b.contentType
}

func (b *Blob) NewReader() (reader io.ReadCloser, size int64, err error) {
	b.init()
	b.onceReader.Do(func() {
		if b.err != nil {
			err = b.err
		}
		if b.peekReader != nil {
			reader = b.peekReader
			size = b.size
		}
	})
	if reader == nil && err == nil {
		reader, size, err = b.newReader()
	}
	return
}

func (b *Blob) ReadAll() ([]byte, error) {
	b.init()
	if b.blobType == BlobTypeEmpty {
		return nil, b.err
	}
	reader, _, err := b.NewReader()
	if reader != nil {
		defer func() {
			_ = reader.Close()
		}()
		buf, err2 := io.ReadAll(reader)
		if err != nil {
			return buf, err
		}
		return buf, err2
	}
	return nil, err
}

func (b *Blob) Err() error {
	b.init()
	return b.err
}

func isBlobEmpty(blob *Blob) bool {
	return blob == nil || blob.IsEmpty()
}

func checkBlob(blob *Blob, err error) (*Blob, error) {
	if blob != nil && err == nil {
		err = blob.Err()
	}
	return blob, err
}
