# seekstream

seekstream provides seekable wrappers for non-seekable `io.ReadCloser` sources.

There are two main modes:

- `NewAsync(source, expected)` keeps data in chunked memory buffers and fills them in the background. This is the fast path when the total size is known and comfortably fits in memory.
- `New(source, buffer)` uses a `Buffer` implementation to build a seekable stream on demand. This is still the fallback when size is unknown or when you want temp-file-backed storage.

`imagor.Blob.NewReadSeeker()` uses the async path for small known-size sources and falls back to the buffered seekstream path for unknown or large sources.

## AsyncReadSeeker

Use `NewAsync(source, expected)` when the source size is known and you want a seekable reader without immediately spooling everything into a temp file.

```go
var source io.ReadCloser // non-seekable
var size int64           // known total size

var rs io.ReadSeekCloser = seekstream.NewAsync(source, size)
```

Example:

```go
package main

import (
	"bytes"
	"io"
	"testing"

	"github.com/cshum/imagor/seekstream"
	"github.com/stretchr/testify/assert"
)

func TestAsync(t *testing.T) {
	source := io.NopCloser(bytes.NewBuffer([]byte("0123456789")))

	rs := seekstream.NewAsync(source, 10)
	defer rs.Close()

	b := make([]byte, 4)
	_, _ = rs.Read(b)
	assert.Equal(t, "0123", string(b))

	b = make([]byte, 3)
	_, _ = rs.Seek(-2, io.SeekCurrent)
	_, _ = rs.Read(b)
	assert.Equal(t, "234", string(b))
}
```

## Buffered SeekStream

```go
var source io.ReadCloser // non-seekable
var buffer seekstream.Buffer
... 
var rs io.ReadSeekCloser = seekstream.New(source, buffer) // seekable
```

## MemoryBuffer

Use `NewMemoryBuffer(size)` with `New(source, buffer)` if total size is known and you want the original buffered seekstream implementation:

```go
package main

import (
	"github.com/cshum/imagor/seekstream"
	...
)

func Test(t *testing.T) {
	source := io.NopCloser(bytes.NewBuffer([]byte("0123456789")))

	rs := seekstream.New(source, seekstream.NewMemoryBuffer(10))
	defer rs.Close()

	b := make([]byte, 4)
	_, _ = rs.Read(b)
	assert.Equal(t, "0123", string(b))

	b = make([]byte, 3)
	_, _ = rs.Seek(-2, io.SeekCurrent)
	_, _ = rs.Read(b)
	assert.Equal(t, "234", string(b))

	b = make([]byte, 4)
	_, _ = rs.Seek(-5, io.SeekEnd)
	_, _ = rs.Read(b)
	assert.Equal(t, "5678", string(b))
}
```

## TempFileBuffer

Use `NewTempFileBuffer(dir, pattern)` with `New(source, buffer)` if total size is not known or does not fit inside memory:

```go
package main

import (
	"github.com/cshum/imagor/seekstream"
	...
)

func Test(t *testing.T) {
	source := io.NopCloser(bytes.NewBuffer([]byte("0123456789")))
	
	buffer, err := seekstream.NewTempFileBuffer("", "seekstream-") 
	assert.NoError(t, err)
	rs := seekstream.New(source, buffer)
	defer rs.Close()
	
	...
}
```
