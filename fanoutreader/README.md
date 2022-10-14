# fanoutreader

fanoutreader allows fan out arbitrary number of reader streams concurrently from one data source with known total size, using channel and memory buffer.

https://pkg.go.dev/github.com/cshum/imagor/fanoutreader

### Why?

There are many scenarios you are trying to fan out a reader stream to multiple writers. For example, reading from a HTTP request that writes to several cloud storages.

Normally you would first download the file into a `[]byte` buffer if it fits inside memory. You may do that with `io.ReadAll`, or better `io.ReadFull` to avoid continuous memory allocations. When the bytes are fully loaded, it is then safe to be written to multiple `io.Writer` concurrently. However, that means data need to be fully loaded before proceeding to the consumers, which is not an optimal way of using a stream pipe.

Here comes `io.TeeReader` and `io.MultiWriter` where you can mirror the reader content to a writer, or writing to several writers in a row. This is great and it works perfectly, if the writers always write at lighting speed and there is zero backpressure when consuming from the reader.

However, in the real world of network I/O, slowdown exists and it may happen at any time. If the writer cannot consume at the expected pace, it blocks, causing backpressure to the reader. This problem magnifies if `io.TeeReader` or `io.MultiWriter` are used, as the writers are sequential throughout the process. Because when any of the writer/consumer backpressure happens, it simply blocks all other writers/consumers in the stack.

So what now? Is it possible to achieve both stream pipe and concurrency? That's where fanoutreader comes handy. fanoutreader achieves both stream pipe and concurrency by leveraging memory buffer and channels. So if the data size is known and can be fit inside memory, then fanoutreader can be used.

fanoutreader is simple to use. Just wrap the `io.ReadCloser` source providing the size: 
```go
fanout := fanoutreader.New(source, size)
``` 
Then you can fan out any number of `io.ReadCloser`:
```go
reader := fanout.NewReader()
``` 
and they will simply work as expected, concurrently.

### Example

Example writing 10 files concurrently from single io.ReadCloser HTTP request. (Error handling are omitted for demo purpose only)

```go
package main

import (
	"fmt"
	"github.com/cshum/imagor/fanoutreader"
	"io"
	"net/http"
	"os"
	"strconv"
	"sync"
)

func main() {
	// http source
	resp, _ := http.DefaultClient.Get("https://raw.githubusercontent.com/cshum/imagor/master/testdata/gopher.png")
	size, _ := strconv.Atoi(resp.Header.Get("Content-Length")) // known size via Content-Length header
	fanout := fanoutreader.New(resp.Body, size) // create fan out from single reader source

	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(i int) {
			reader := fanout.NewReader() // fan out new reader
			defer reader.Close()
			file, _ := os.Create(fmt.Sprintf("gopher-%d.png", i))
			defer file.Close()
			_, _ = io.Copy(file, reader) // read/write concurrently alongside other readers
			wg.Done()
		}(i)
	}
	wg.Wait()
}
```
