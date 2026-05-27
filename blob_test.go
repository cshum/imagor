package imagor

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"sync/atomic"
	"testing"

	"github.com/cshum/imagor/seekstream"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type nopReadCloser struct {
	io.Reader
}

func (n nopReadCloser) Close() error { return nil }

func doTestBlobReaders(t *testing.T, b *Blob, buf []byte) {
	r, size, err := b.NewReader()
	assert.NotNil(t, r)
	assert.NotEmpty(t, size)
	assert.NoError(t, err)

	buf2, err := io.ReadAll(r)
	require.NoError(t, err)
	assert.NotEmpty(t, buf2)
	assert.Equal(t, buf, buf2, "bytes not equal")

	rs, size, err := b.NewReadSeeker()
	assert.NotNil(t, rs)
	assert.NotEmpty(t, size)
	assert.NoError(t, err)
	defer rs.Close()

	buf3, err := io.ReadAll(rs)
	require.NoError(t, err)
	assert.NotEmpty(t, buf3)
	assert.Equal(t, buf, buf3, "bytes not equal")

	for i := 0; i < 3; i++ {
		_, err = rs.Seek(0, io.SeekStart)
		require.NoError(t, err)
		buf4, err := io.ReadAll(rs)
		require.NoError(t, err)
		assert.NotEmpty(t, buf4)
		assert.Equal(t, buf, buf4, "bytes not equal")
	}
}

func TestBlobTypes(t *testing.T) {
	tests := []struct {
		name              string
		path              string
		contentType       string
		extension         string
		bytesType         BlobType
		supportsAnimation bool
	}{
		{
			name:        "jpeg",
			path:        "demo1.jpg",
			contentType: "image/jpeg",
			extension:   ".jpg",
			bytesType:   BlobTypeJPEG,
		},
		{
			name:        "png",
			path:        "gopher.png",
			contentType: "image/png",
			extension:   ".png",
			bytesType:   BlobTypePNG,
		},
		{
			name:        "tiff",
			path:        "gopher.tiff",
			contentType: "image/tiff",
			extension:   ".tiff",
			bytesType:   BlobTypeTIFF,
		},
		{
			name:              "gif",
			path:              "dancing-banana.gif",
			contentType:       "image/gif",
			extension:         ".gif",
			bytesType:         BlobTypeGIF,
			supportsAnimation: true,
		},
		{
			name:              "webp",
			path:              "demo3.webp",
			contentType:       "image/webp",
			extension:         ".webp",
			bytesType:         BlobTypeWEBP,
			supportsAnimation: true,
		},
		{
			name:        "jxl",
			path:        "jxl-isobmff.jxl",
			contentType: "image/jxl",
			extension:   ".jxl",
			bytesType:   BlobTypeJXL,
		},
		{
			name:        "avif",
			path:        "gopher-front.avif",
			contentType: "image/avif",
			extension:   ".avif",
			bytesType:   BlobTypeAVIF,
		},
		{
			name:        "heif",
			path:        "gopher-front.heif",
			contentType: "image/heif",
			extension:   ".heif",
			bytesType:   BlobTypeHEIF,
		},
		{
			name:        "jp2",
			path:        "gopher.jp2",
			contentType: "image/jp2",
			extension:   ".jp2",
			bytesType:   BlobTypeJP2,
		},
		{
			name:        "pdf",
			path:        "sample.pdf",
			contentType: "application/pdf",
			extension:   ".pdf",
			bytesType:   BlobTypePDF,
		},
		{
			name:        "bmp",
			path:        "bmp_24.bmp",
			contentType: "image/bmp",
			extension:   ".bmp",
			bytesType:   BlobTypeBMP,
		},
		{
			name:        "svg",
			path:        "test.svg",
			contentType: "image/svg+xml",
			extension:   ".svg",
			bytesType:   BlobTypeSVG,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filepath := "testdata/" + tt.path
			b := NewBlobFromFile(filepath, func(info os.FileInfo) error {
				// noop
				return nil
			})
			assert.Equal(t, tt.supportsAnimation, b.SupportsAnimation())
			assert.Equal(t, tt.contentType, b.ContentType())
			assert.Equal(t, filepath, b.FilePath())
			assert.Equal(t, tt.bytesType, b.BlobType())
			assert.Equal(t, tt.extension, getExtension(b.BlobType()))
			assert.False(t, b.IsEmpty())
			assert.NotEmpty(t, b.Sniff())
			assert.NotEmpty(t, b.Size())
			require.NoError(t, b.Err())

			buf, err := b.ReadAll()
			require.NoError(t, err)

			doTestBlobReaders(t, b, buf)

			b = NewBlobFromBytes(buf)
			assert.Equal(t, tt.supportsAnimation, b.SupportsAnimation())
			assert.Equal(t, tt.contentType, b.ContentType())
			assert.Equal(t, tt.bytesType, b.BlobType())
			assert.False(t, b.IsEmpty())
			assert.NotEmpty(t, b.Sniff())
			assert.NotEmpty(t, b.Size())
			require.NoError(t, b.Err())

			doTestBlobReaders(t, b, buf)

			b = NewBlob(func() (reader io.ReadCloser, size int64, err error) {
				return io.NopCloser(bytes.NewReader(buf)), int64(len(buf)), nil
			})
			assert.Equal(t, tt.supportsAnimation, b.SupportsAnimation())
			assert.Equal(t, tt.contentType, b.ContentType())
			assert.Equal(t, tt.bytesType, b.BlobType())
			assert.False(t, b.IsEmpty())
			assert.NotEmpty(t, b.Sniff())
			assert.NotEmpty(t, b.Size())
			require.NoError(t, b.Err())

			doTestBlobReaders(t, b, buf)

			b = NewBlob(func() (reader io.ReadCloser, size int64, err error) {
				// unknown size to force discard fanout
				return io.NopCloser(bytes.NewReader(buf)), 0, nil
			})
			assert.Equal(t, tt.supportsAnimation, b.SupportsAnimation())
			assert.Equal(t, tt.contentType, b.ContentType())
			assert.Equal(t, tt.bytesType, b.BlobType())
			assert.False(t, b.IsEmpty())
			assert.NotEmpty(t, b.Sniff())
			assert.Empty(t, b.Size())
			require.NoError(t, b.Err())

			r, size, err := b.NewReader()
			assert.NotNil(t, r)
			assert.Empty(t, size)
			assert.NoError(t, err)

			buf2, err := io.ReadAll(r)
			require.NoError(t, err)
			assert.NotEmpty(t, buf2)
			assert.Equal(t, buf, buf2, "bytes not equal")

			rs, size, err := b.NewReadSeeker()
			require.NoError(t, err)
			defer rs.Close()
			buf3, err := io.ReadAll(rs)
			require.NoError(t, err)
			assert.NotEmpty(t, buf3)
			assert.Equal(t, buf, buf3, "bytes not equal")

			for i := 0; i < 3; i++ {
				_, err = rs.Seek(0, io.SeekStart)
				require.NoError(t, err)
				buf4, err := io.ReadAll(rs)
				require.NoError(t, err)
				assert.NotEmpty(t, buf4)
				assert.Equal(t, buf, buf4, "bytes not equal")
			}
		})
	}
}

func TestNewEmptyBlob(t *testing.T) {
	b := NewBlobFromBytes([]byte{})
	assert.Empty(t, b.Sniff())
	assert.True(t, b.IsEmpty())
	assert.Equal(t, BlobTypeEmpty, b.BlobType())

	buf, err := b.ReadAll()
	assert.NoError(t, err)
	assert.Empty(t, buf)

	b = NewEmptyBlob()
	assert.Equal(t, BlobTypeEmpty, b.BlobType())
	assert.True(t, b.IsEmpty())
	assert.Empty(t, b.Sniff())
	assert.Empty(t, b.Size())

	buf, err = b.ReadAll()
	assert.NoError(t, err)
	assert.Empty(t, buf)

	r, size, err := b.NewReader()
	assert.NoError(t, err)
	assert.Empty(t, size)
	assert.Empty(t, b.Size())

	buf, err = io.ReadAll(r)
	assert.NoError(t, err)
	assert.Empty(t, buf)

	f, err := os.CreateTemp("", "tmpfile-")
	require.NoError(t, err)
	defer f.Close()
	defer os.Remove(f.Name())
	fmt.Println(f.Name())
	b = NewBlobFromFile(f.Name())
	assert.Equal(t, BlobTypeEmpty, b.BlobType())
	assert.True(t, b.IsEmpty())
	assert.Empty(t, b.Sniff())
	assert.Empty(t, b.Size())

	buf, err = b.ReadAll()
	assert.NoError(t, err)
	assert.Empty(t, buf)
}

func TestBlobNewReadSeeker_PrefersNativeSeeker(t *testing.T) {
	b := NewBlob(func() (io.ReadCloser, int64, error) {
		return &readSeekNopCloser{ReadSeeker: bytes.NewReader([]byte("012345"))}, 6, nil
	})

	rs, size, err := b.NewReadSeeker()
	require.NoError(t, err)
	defer rs.Close()

	assert.Equal(t, int64(6), size)
	_, ok := rs.(*seekstream.AsyncReadSeeker)
	assert.False(t, ok)
	_, ok = rs.(*seekstream.SeekStream)
	assert.False(t, ok)
}

func TestBlobNewReadSeeker_UsesAsyncForSmallKnownSize(t *testing.T) {
	b := NewBlob(func() (io.ReadCloser, int64, error) {
		return nopReadCloser{Reader: bytes.NewReader([]byte("012345"))}, 6, nil
	})

	rs, size, err := b.NewReadSeeker()
	require.NoError(t, err)
	defer rs.Close()

	assert.Equal(t, int64(6), size)
	_, ok := rs.(*seekstream.AsyncReadSeeker)
	assert.True(t, ok)
	}

func TestBlobNewReadSeeker_UsesSeekStreamForUnknownOrLargeSize(t *testing.T) {
	t.Run("unknown size", func(t *testing.T) {
		b := NewBlob(func() (io.ReadCloser, int64, error) {
			return nopReadCloser{Reader: bytes.NewReader([]byte("012345"))}, 0, nil
		})

		rs, size, err := b.NewReadSeeker()
		require.NoError(t, err)
		defer rs.Close()

		assert.Equal(t, int64(0), size)
		_, ok := rs.(*seekstream.SeekStream)
		assert.True(t, ok)
	})

	t.Run("large size", func(t *testing.T) {
		b := NewBlob(func() (io.ReadCloser, int64, error) {
			return nopReadCloser{Reader: bytes.NewReader([]byte("012345"))}, maxMemorySize, nil
		})

		rs, size, err := b.NewReadSeeker()
		require.NoError(t, err)
		defer rs.Close()

		assert.Equal(t, maxMemorySize, size)
		_, ok := rs.(*seekstream.SeekStream)
		assert.True(t, ok)
	})
}

func TestBlobFanoutNewReadSeeker_UsesAsyncOnSharedNonSeekableSource(t *testing.T) {
	payload := bytes.Repeat([]byte("fanout-async-"), 64)
	var sourceCalls atomic.Int32
	b := NewBlob(func() (io.ReadCloser, int64, error) {
		sourceCalls.Add(1)
		return nopReadCloser{Reader: bytes.NewReader(payload)}, int64(len(payload)), nil
	})
	b.setFanout(true)

	r, size, err := b.NewReader()
	require.NoError(t, err)
	defer r.Close()
	assert.Equal(t, int64(len(payload)), size)

	head := make([]byte, 17)
	n, err := io.ReadFull(r, head)
	require.NoError(t, err)
	assert.Equal(t, payload[:n], head[:n])

	rs, size, err := b.NewReadSeeker()
	require.NoError(t, err)
	defer rs.Close()
	assert.Equal(t, int64(len(payload)), size)
	_, ok := rs.(*seekstream.AsyncReadSeeker)
	assert.True(t, ok)
	assert.Equal(t, int32(1), sourceCalls.Load())

	chunk := make([]byte, 23)
	n, err = rs.Read(chunk)
	require.NoError(t, err)
	assert.Equal(t, payload[:n], chunk[:n])

	_, err = rs.Seek(0, io.SeekStart)
	require.NoError(t, err)
	all, err := io.ReadAll(rs)
	require.NoError(t, err)
	assert.Equal(t, payload, all)

	remainder, err := io.ReadAll(r)
	require.NoError(t, err)
	assert.Equal(t, payload[len(head):], remainder)
	assert.Equal(t, int32(1), sourceCalls.Load())
}

func TestBlobFanoutNewReadSeeker_DefersSeekableCloneUntilSeek(t *testing.T) {
	payload := bytes.Repeat([]byte("fanout-hybrid-"), 64)
	var sourceCalls atomic.Int32
	b := NewBlob(func() (io.ReadCloser, int64, error) {
		sourceCalls.Add(1)
		return &readSeekNopCloser{ReadSeeker: bytes.NewReader(payload)}, int64(len(payload)), nil
	})
	b.setFanout(true)

	rs, size, err := b.NewReadSeeker()
	require.NoError(t, err)
	defer rs.Close()
	assert.Equal(t, int64(len(payload)), size)

	hybrid, ok := rs.(*hybridReadSeeker)
	require.True(t, ok)
	assert.Nil(t, hybrid.seeker)
	assert.Equal(t, int32(1), sourceCalls.Load())

	buf := make([]byte, 19)
	n, err := rs.Read(buf)
	require.NoError(t, err)
	assert.Equal(t, payload[:n], buf[:n])
	assert.Nil(t, hybrid.seeker)
	assert.Equal(t, int32(1), sourceCalls.Load())

	_, err = rs.Seek(0, io.SeekStart)
	require.NoError(t, err)
	require.NotNil(t, hybrid.seeker)
	assert.Equal(t, int32(2), sourceCalls.Load())

	all, err := io.ReadAll(rs)
	require.NoError(t, err)
	assert.Equal(t, payload, all)
}

func TestNewBlobFromMemory(t *testing.T) {
	b := NewEmptyBlob()
	data, width, height, bands, ok := b.Memory()
	assert.Empty(t, data)
	assert.Empty(t, width)
	assert.Empty(t, height)
	assert.Empty(t, bands)
	assert.False(t, ok)
	assert.True(t, b.IsEmpty())
	b = NewBlobFromMemory([]byte{167, 169}, 2, 1, 1)
	assert.Equal(t, BlobTypeMemory, b.BlobType())
	assert.False(t, b.IsEmpty())
	data, width, height, bands, ok = b.Memory()
	assert.Equal(t, []byte{167, 169}, data)
	assert.Equal(t, 2, width)
	assert.Equal(t, 1, height)
	assert.Equal(t, 1, bands)
	assert.True(t, ok)
}

func TestNewJsonMarshalBlob(t *testing.T) {
	b := NewBlobFromJsonMarshal(map[string]string{
		"foo": "bar",
	})
	assert.Equal(t, BlobTypeJSON, b.BlobType())
	assert.Equal(t, "application/json", b.ContentType())
	assert.Equal(t, `{"foo":"bar"}`, string(b.Sniff()))
	buf, _ := b.ReadAll()
	assert.Equal(t, `{"foo":"bar"}`, string(buf))
}

func TestBlobOverrideContentType(t *testing.T) {
	b := NewBlobFromFile("testdata/demo1.jpg")
	b.SetContentType("foo/bar")
	assert.Equal(t, BlobTypeJPEG, b.BlobType())
	assert.Equal(t, "foo/bar", b.ContentType())
}

func TestBlobJsonBytes(t *testing.T) {
	b := NewBlobFromBytes([]byte(`{"foo": "bar"}`))
	assert.Equal(t, BlobTypeJSON, b.BlobType())
	assert.Equal(t, "application/json", b.ContentType())
	assert.Equal(t, ".json", getExtension(b.BlobType()))
}

type readerFunc func(p []byte) (n int, err error)

func (rf readerFunc) Read(p []byte) (n int, err error) { return rf(p) }

func TestBlobCreateError(t *testing.T) {
	e := errors.New("some error")
	b := NewBlob(func() (reader io.ReadCloser, size int64, err error) {
		return nil, 0, e
	})
	assert.Equal(t, e, b.Err())
	buf, err := b.ReadAll()
	assert.Empty(t, buf)
	assert.Equal(t, e, err)
}

func TestBlobTypeRAWFormats(t *testing.T) {
	tests := []struct {
		name        string
		header      []byte
		blobType    BlobType
		contentType string
		extension   string
	}{
		{
			name:        "fuji_raf",
			header:      []byte("FUJIFILMCCD-RAW\x00\x00\x00\x00\x00\x00\x00\x00\x00"),
			blobType:    BlobTypeRAF,
			contentType: "image/x-fuji-raf",
			extension:   ".raf",
		},
		{
			name:        "olympus_orf_le",
			header:      []byte("\x49\x49\x52\x4F\x08\x00\x00\x00\x10\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00"),
			blobType:    BlobTypeORF,
			contentType: "image/x-olympus-orf",
			extension:   ".orf",
		},
		{
			name:        "olympus_orf_be",
			header:      []byte("\x4D\x4D\x4F\x52\x00\x08\x00\x00\x00\x10\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00"),
			blobType:    BlobTypeORF,
			contentType: "image/x-olympus-orf",
			extension:   ".orf",
		},
		{
			name:        "panasonic_rw2",
			header:      []byte("\x49\x49\x55\x00\x08\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00"),
			blobType:    BlobTypeRW2,
			contentType: "image/x-panasonic-rw2",
			extension:   ".rw2",
		},
		{
			name:        "sigma_x3f",
			header:      []byte("\x46\x4F\x56\x62\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00"),
			blobType:    BlobTypeX3F,
			contentType: "image/x-sigma-x3f",
			extension:   ".x3f",
		},
		{
			name: "canon_cr3",
			// ftyp at [4:8], crx  at [8:12]
			header:      []byte("\x00\x00\x00\x18\x66\x74\x79\x70\x63\x72\x78\x20\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00"),
			blobType:    BlobTypeCR3,
			contentType: "image/x-canon-cr3",
			extension:   ".cr3",
		},
		{
			// Canon CR2: TIFF header + "CR" at [8:10] — unique to CR2.
			// Gets its own BlobTypeCR2 so it bypasses dcrawload_source (which crashes on CR2)
			// and goes straight to the normal TIFF loader. IsRaw() still returns true.
			name:        "canon_cr2",
			header:      []byte("\x49\x49\x2A\x00\x08\x00\x00\x00\x43\x52\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00"),
			blobType:    BlobTypeCR2,
			contentType: "image/x-canon-cr2",
			extension:   ".cr2",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Pad to 512 bytes so sniffing works
			buf := make([]byte, 512)
			copy(buf, tt.header)

			b := NewBlobFromBytes(buf)
			assert.Equal(t, tt.blobType, b.BlobType(), "expected %v for %s", tt.blobType, tt.name)
			assert.Equal(t, tt.contentType, b.ContentType())
			assert.Equal(t, tt.extension, getExtension(b.BlobType()))
			assert.True(t, b.IsRaw(), "IsRaw() must be true for %s", tt.name)
			assert.False(t, b.IsEmpty())
			assert.False(t, b.SupportsAnimation())
		})
	}
}

func TestBlobReaderError(t *testing.T) {
	e := errors.New("some error")
	buf, err := os.ReadFile("testdata/demo1.jpg")
	require.NoError(t, err)
	var called int
	b := NewBlob(func() (reader io.ReadCloser, size int64, err error) {
		return io.NopCloser(readerFunc(func(p []byte) (n int, err error) {
			if called > 4 {
				return 0, e
			}
			called++
			if len(p) > 100 {
				p = p[:100]
			}
			n = copy(p, buf)
			buf = buf[n:]
			return
		})), int64(len(buf)), nil
	})
	assert.Equal(t, e, b.Err())
	assert.Equal(t, 500, len(b.Sniff()))
	buf, err = b.ReadAll()
	assert.Equal(t, 500, len(buf))
	assert.Equal(t, e, err)
}

func TestBlobUnknownSizeReusesSniffedReaderForFirstRead(t *testing.T) {
	buf, err := os.ReadFile("testdata/demo1.jpg")
	require.NoError(t, err)

	var calls int
	b := NewBlob(func() (reader io.ReadCloser, size int64, err error) {
		calls++
		return io.NopCloser(bytes.NewReader(buf)), 0, nil
	})

	assert.Equal(t, BlobTypeJPEG, b.BlobType())
	assert.Equal(t, 1, calls, "sniff should open the source once")

	r, size, err := b.NewReader()
	require.NoError(t, err)
	defer r.Close()
	assert.Zero(t, size)

	data, err := io.ReadAll(r)
	require.NoError(t, err)
	assert.Equal(t, buf, data)
	assert.Equal(t, 1, calls, "first reader should reuse the sniffed source")

	r2, size, err := b.NewReader()
	require.NoError(t, err)
	defer r2.Close()
	assert.Zero(t, size)

	data2, err := io.ReadAll(r2)
	require.NoError(t, err)
	assert.Equal(t, buf, data2)
	assert.Equal(t, 2, calls, "subsequent readers still reopen the source")
}

func TestBlobKnownSizeFanoutStillAvoidsReopenAfterSniff(t *testing.T) {
	buf, err := os.ReadFile("testdata/demo1.jpg")
	require.NoError(t, err)

	var calls int
	b := NewBlob(func() (reader io.ReadCloser, size int64, err error) {
		calls++
		return io.NopCloser(bytes.NewReader(buf)), int64(len(buf)), nil
	})

	assert.Equal(t, BlobTypeJPEG, b.BlobType())
	assert.Equal(t, 1, calls, "sniff should open the source once")

	r1, size, err := b.NewReader()
	require.NoError(t, err)
	defer r1.Close()
	assert.Equal(t, int64(len(buf)), size)

	r2, size, err := b.NewReader()
	require.NoError(t, err)
	defer r2.Close()
	assert.Equal(t, int64(len(buf)), size)

	data1, err := io.ReadAll(r1)
	require.NoError(t, err)
	data2, err := io.ReadAll(r2)
	require.NoError(t, err)
	assert.Equal(t, buf, data1)
	assert.Equal(t, buf, data2)
	assert.Equal(t, 1, calls, "fanout path should keep sharing the original source")
}

func TestBlobKnownSizeNonFanoutPreservesSeekableFirstReader(t *testing.T) {
	buf, err := os.ReadFile("testdata/demo1.jpg")
	require.NoError(t, err)

	b := NewBlobFromBytes(buf)

	assert.Equal(t, BlobTypeJPEG, b.BlobType())

	r, size, err := b.NewReader()
	require.NoError(t, err)
	defer r.Close()
	assert.Equal(t, int64(len(buf)), size)

	seeker, ok := r.(io.Seeker)
	require.True(t, ok, "first reader should remain seekable")

	_, err = seeker.Seek(0, io.SeekStart)
	require.NoError(t, err)

	data, err := io.ReadAll(r)
	require.NoError(t, err)
	assert.Equal(t, buf, data)
}
