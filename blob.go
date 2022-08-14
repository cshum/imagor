package imagor

import (
	"bufio"
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"sync"
)

type BlobType int

const maxBodySize = int64(32 << 20) // 32MB

const (
	BlobTypeUnknown BlobType = iota
	BlobTypeEmpty
	BlobTypeJSON
	BlobTypeJPEG
	BlobTypePNG
	BlobTypeGIF
	BlobTypeWEBP
	BlobTypeAVIF
	BlobTypeHEIF
	BlobTypeTIFF
)

type Blob struct {
	newReader  func() (r io.ReadCloser, size int64, err error)
	peekReader *peekReaderCloser
	once       sync.Once
	onceReader sync.Once
	buf        []byte
	err        error
	size       int64

	blobType    BlobType
	filepath    string
	contentType string
}

func NewBlob(newReader func() (reader io.ReadCloser, size int64, err error)) *Blob {
	return &Blob{newReader: newReader}
}

func NewBlobFromFile(filepath string, checks ...func(os.FileInfo) error) *Blob {
	stat, err := os.Stat(filepath)
	if os.IsNotExist(err) {
		err = ErrNotFound
	}
	if err == nil {
		for _, check := range checks {
			if err = check(stat); err != nil {
				break
			}
		}
	}
	return &Blob{
		err:      err,
		filepath: filepath,
		newReader: func() (io.ReadCloser, int64, error) {
			if err != nil {
				return nil, 0, err
			}
			reader, err := os.Open(filepath)
			return reader, stat.Size(), err
		},
	}
}

func NewBlobFromJsonMarshal(v any) *Blob {
	buf, err := json.Marshal(v)
	size := int64(len(buf))
	return &Blob{
		err:      err,
		blobType: BlobTypeJSON,
		newReader: func() (io.ReadCloser, int64, error) {
			return io.NopCloser(bytes.NewReader(buf)), size, err
		},
	}
}

func NewBlobFromBytes(buf []byte) *Blob {
	size := int64(len(buf))
	return NewBlob(func() (io.ReadCloser, int64, error) {
		return io.NopCloser(bytes.NewReader(buf)), size, nil
	})
}

func NewBlobFromReader(reader io.ReadCloser) *Blob {
	newReader := FanoutReader(reader, 0)
	return NewBlob(func() (io.ReadCloser, int64, error) {
		return newReader(), 0, nil
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
		if b.err != nil {
			return
		}
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
		if b.blobType != BlobTypeEmpty && b.blobType != BlobTypeJSON &&
			len(b.buf) > 24 {
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
		if b.contentType == "" {
			switch b.blobType {
			case BlobTypeJSON:
				b.contentType = "application/json"
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

func (b *Blob) Size() int64 {
	b.init()
	return b.size
}

func (b *Blob) FilePath() string {
	return b.filepath
}

func (b *Blob) SetContentType(contentType string) {
	b.contentType = contentType
}

func (b *Blob) ContentType() string {
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
