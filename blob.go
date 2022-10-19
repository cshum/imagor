package imagor

import (
	"bytes"
	"encoding/json"
	"github.com/cshum/imagor/fanoutreader"
	"github.com/cshum/imagor/seekstream"
	"io"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
)

type BlobType int

const maxMemorySize = int64(100 << 20) // 100MB

const (
	BlobTypeUnknown BlobType = iota
	BlobTypeEmpty
	BlobTypeMemory
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
	newReader     func() (r io.ReadCloser, size int64, err error)
	newReadSeeker func() (rs io.ReadSeekCloser, size int64, err error)
	fanout        bool
	once          sync.Once
	sniffBuf      []byte
	err           error
	size          int64
	blobType      BlobType
	filepath      string
	contentType   string
	memory        *memory

	Stat *Stat
}

// Stat blob stat attributes
type Stat struct {
	ModifiedTime time.Time
	ETag         string
	Size         int64
}

func NewBlob(newReader func() (reader io.ReadCloser, size int64, err error)) *Blob {
	return &Blob{
		fanout:    true,
		newReader: newReader,
	}
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
	blob := &Blob{
		err:      err,
		filepath: filepath,
		fanout:   true,
		newReader: func() (io.ReadCloser, int64, error) {
			if err != nil {
				return nil, 0, err
			}
			reader, err := os.Open(filepath)
			return reader, stat.Size(), err
		},
	}
	if err == nil && stat != nil {
		size := stat.Size()
		modTime := stat.ModTime()
		blob.Stat = &Stat{
			Size:         size,
			ModifiedTime: modTime,
		}
	}
	return blob
}

func NewBlobFromJsonMarshal(v any) *Blob {
	buf, err := json.Marshal(v)
	size := int64(len(buf))
	return &Blob{
		err:      err,
		blobType: BlobTypeJSON,
		fanout:   false,
		newReader: func() (io.ReadCloser, int64, error) {
			rs := bytes.NewReader(buf)
			return &readSeekNopCloser{ReadSeeker: rs}, size, err
		},
	}
}

func NewBlobFromBytes(buf []byte) *Blob {
	size := int64(len(buf))
	return &Blob{
		fanout: false,
		newReader: func() (io.ReadCloser, int64, error) {
			rs := bytes.NewReader(buf)
			return &readSeekNopCloser{ReadSeeker: rs}, size, nil
		},
	}
}

func NewBlobFromMemory(buf []byte, width, height, bands int) *Blob {
	return &Blob{memory: &memory{
		data:   buf,
		width:  width,
		height: height,
		bands:  bands,
	}}
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

var jsonPrefix = []byte(`{"`)

type readSeekNopCloser struct {
	io.ReadSeeker
}

func (readSeekNopCloser) Close() error { return nil }

type memory struct {
	data   []byte
	width  int
	height int
	bands  int
}

func newEmptyReader() (io.ReadCloser, int64, error) {
	return &readSeekNopCloser{bytes.NewReader(nil)}, 0, nil
}

func (b *Blob) init() {
	b.once.Do(func() {
		if b.err != nil {
			return
		}
		if b.newReader == nil {
			if b.memory != nil {
				b.blobType = BlobTypeMemory
			} else {
				b.blobType = BlobTypeEmpty
			}
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
		if _, ok := reader.(io.ReadSeekCloser); ok {
			// construct seeker factory if source supports seek
			newReader := b.newReader
			b.newReadSeeker = func() (io.ReadSeekCloser, int64, error) {
				r, size, err := newReader()
				return r.(io.ReadSeekCloser), size, err
			}
		}
		if b.fanout && size > 0 && size < maxMemorySize && err == nil {
			// use fan-out reader if buf size known and within memory size
			// otherwise create new readers
			fanout := fanoutreader.New(reader, int(size))
			b.newReader = func() (io.ReadCloser, int64, error) {
				return fanout.NewReader(), size, nil
			}
			reader = fanout.NewReader()
		} else {
			b.fanout = false
		}
		// sniff first 512 bytes for type sniffing
		b.sniffBuf = make([]byte, 512)
		n, err := io.ReadAtLeast(reader, b.sniffBuf, 512)
		_ = reader.Close()
		if n < 512 {
			b.sniffBuf = b.sniffBuf[:n]
		}
		if len(b.sniffBuf) == 0 {
			b.blobType = BlobTypeEmpty
		}
		if err != nil &&
			err != io.ErrUnexpectedEOF &&
			err != io.EOF {
			if b.err == nil {
				b.err = err
			}
			return
		}
		if b.blobType != BlobTypeEmpty && b.blobType != BlobTypeJSON &&
			len(b.sniffBuf) > 24 {
			if bytes.Equal(b.sniffBuf[:3], jpegHeader) {
				b.blobType = BlobTypeJPEG
			} else if bytes.Equal(b.sniffBuf[:4], pngHeader) {
				b.blobType = BlobTypePNG
			} else if bytes.Equal(b.sniffBuf[:3], gifHeader) {
				b.blobType = BlobTypeGIF
			} else if bytes.Equal(b.sniffBuf[8:12], webpHeader) {
				b.blobType = BlobTypeWEBP
			} else if bytes.Equal(b.sniffBuf[4:8], ftyp) && bytes.Equal(b.sniffBuf[8:12], avif) {
				b.blobType = BlobTypeAVIF
			} else if bytes.Equal(b.sniffBuf[4:8], ftyp) && (bytes.Equal(b.sniffBuf[8:12], heic) ||
				bytes.Equal(b.sniffBuf[8:12], mif1) ||
				bytes.Equal(b.sniffBuf[8:12], msf1)) {
				b.blobType = BlobTypeHEIF
			} else if bytes.Equal(b.sniffBuf[:4], tifII) || bytes.Equal(b.sniffBuf[:4], tifMM) {
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
				b.contentType = http.DetectContentType(b.sniffBuf)
			}
		}
		if strings.HasPrefix(b.contentType, "text/plain") {
			if bytes.Equal(b.sniffBuf[:2], jsonPrefix) {
				b.blobType = BlobTypeJSON
				b.contentType = "application/json"
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
	return b.sniffBuf
}

func (b *Blob) Size() int64 {
	b.init()
	return b.size
}

func (b *Blob) FilePath() string {
	return b.filepath
}

func (b *Blob) Memory() (data []byte, width, height, bands int, ok bool) {
	if m := b.memory; m != nil {
		data = m.data
		width = m.width
		height = m.height
		bands = m.bands
		ok = true
	}
	return
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
	return b.newReader()
}

// NewReadSeeker create read seeker if reader supports seek, or attempts to simulate seek using memory buffer
func (b *Blob) NewReadSeeker() (io.ReadSeekCloser, int64, error) {
	b.init()
	if b.newReadSeeker != nil {
		return b.newReadSeeker()
	}
	// if source not seekable, simulate seek with seek stream
	reader, size, err := b.NewReader()
	if err != nil {
		return nil, size, err
	}
	var buffer seekstream.Buffer
	if size > 0 && size < maxMemorySize {
		// in memory buffer if size is known and less then 100mb
		buffer = seekstream.NewMemoryBuffer(size)
	} else {
		// otherwise temp file buffer
		buffer, err = seekstream.NewTempFileBuffer("", "imagor-")
		if err != nil {
			return nil, size, err
		}
	}
	return seekstream.New(reader, buffer), size, err
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

func getExtension(typ BlobType) (ext string) {
	switch typ {
	case BlobTypeJPEG:
		ext = ".jpg"
	case BlobTypePNG:
		ext = ".png"
	case BlobTypeGIF:
		ext = ".gif"
	case BlobTypeWEBP:
		ext = ".webp"
	case BlobTypeAVIF:
		ext = ".avif"
	case BlobTypeHEIF:
		ext = ".heif"
	case BlobTypeTIFF:
		ext = ".tiff"
	case BlobTypeJSON:
		ext = ".json"
	}
	return
}
