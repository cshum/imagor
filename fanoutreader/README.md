# fanoutreader

fanoutreader allows fan-out an arbitrary number of reader streams concurrently from one data source with known total size, using channel and memory buffer.

```go
package main

import (
	"context"
	"fmt"
	"github.com/cshum/imagor/fanoutreader"
	"github.com/stretchr/testify/require"
	"golang.org/x/sync/errgroup"
	"io"
	"net/http"
	"os"
	"strconv"
	"testing"
)

func Test(t *testing.T) {
	// http source with known size via Content-Length header
	resp, err := http.DefaultClient.Get("https://raw.githubusercontent.com/cshum/imagor/master/testdata/gopher.png")
	require.NoError(t, err)
	size, _ := strconv.Atoi(resp.Header.Get("Content-Length"))

	fanout := fanoutreader.New(resp.Body, size) // create fanout from single reader source

	g, _ := errgroup.WithContext(context.Background())
	for i := 0; i < 10; i++ {
		func(i int) {
			g.Go(func() error {
				reader := fanout.NewReader() // spawn new reader concurrently
				defer reader.Close()
				file, err := os.Create(fmt.Sprintf("gopher-%d.png", i))
				if err != nil {
					return err
				}
				defer file.Close()
				_, err := io.Copy(file, reader)
				return err
			})
		}(i)
	}
	require.NoError(t, g.Wait())
}

```
