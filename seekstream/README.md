# seekstream

seekstream allows seeking on non-seekable `io.ReadCloser` source by buffering read data using memory or temp file.

```go
var source io.ReadCloser // non-seekable
var buffer seekstream.Buffer
... 
var rs io.ReadSeekCloser = seekstream.New(source, buffer) // seekable
```

## MemoryBuffer

Use `NewMemoryBuffer(size)` if total size is known and can be fit inside memory:

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

Use `NewTempFileBuffer(dir, pattern)` if total size is not known or too large to fit inside memory:

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
