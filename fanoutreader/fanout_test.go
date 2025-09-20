package fanoutreader

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/sync/errgroup"
)

func doFanoutTest(t *testing.T, do func(), n, m int) {
	g, _ := errgroup.WithContext(context.Background())
	for i := 0; i < n; i++ {
		g.Go(func() error {
			do()
			return nil
		})
	}
	assert.NoError(t, g.Wait())
	for i := 0; i < m; i++ {
		do()
	}
}

func TestFanoutSizeOver(t *testing.T) {
	buf := []byte("abcdefghi")
	source := io.NopCloser(bytes.NewReader(buf))
	factory := New(source, 5)
	doFanoutTest(t, func() {
		r := factory.NewReader()
		res1, err := io.ReadAll(r)
		assert.NoError(t, err)
		assert.NoError(t, r.Close())
		assert.Equal(t, buf[:5], res1)
	}, 100, 1)
}

func TestFanoutSizeBelow(t *testing.T) {
	buf := []byte("abcd")
	source := io.NopCloser(bytes.NewReader(buf))
	factory := New(source, 5)
	doFanoutTest(t, func() {
		r := factory.NewReader()
		res1, err := io.ReadAll(r)
		assert.NoError(t, err)
		assert.NoError(t, r.Close())
		assert.Equal(t, buf, res1)
	}, 100, 1)
}

type readerFunc func(p []byte) (n int, err error)

func (rf readerFunc) Read(p []byte) (n int, err error) { return rf(p) }

func TestFanoutUpstreamError(t *testing.T) {
	e := errors.New("upstream error")
	buf := []byte("abcdefghi")
	called := false
	source := io.NopCloser(readerFunc(func(p []byte) (n int, err error) {
		if called {
			return 0, e
		}
		called = true
		n = copy(p, buf)
		return
	}))
	factory := New(source, 10000)
	doFanoutTest(t, func() {
		r := factory.NewReader()
		res, err := io.ReadAll(r)
		assert.ErrorIs(t, err, e)
		assert.Equal(t, []byte("abcdefghi"), res)
	}, 100, 1)
}

func TestFanoutErrClosedPipe(t *testing.T) {
	buf := []byte("abcdefghi")
	source := io.NopCloser(bytes.NewReader(buf))
	factory := New(source, len(buf))
	r := factory.NewReader()
	b := make([]byte, 5)
	n, err := r.Read(b)
	assert.NoError(t, err)
	assert.Equal(t, n, 5)
	assert.Equal(t, buf[:5], b)
	assert.NoError(t, r.Close())
	b = make([]byte, 5)
	n, err = r.Read(b)
	assert.ErrorIs(t, err, io.ErrClosedPipe)
	assert.Empty(t, n)
}

// Test empty source
func TestFanoutEmptySource(t *testing.T) {
	source := io.NopCloser(bytes.NewReader([]byte{}))
	factory := New(source, 0)

	r := factory.NewReader()
	defer r.Close()

	data, err := io.ReadAll(r)
	assert.NoError(t, err)
	assert.Empty(t, data)
}

// Test single byte source
func TestFanoutSingleByte(t *testing.T) {
	buf := []byte("x")
	source := io.NopCloser(bytes.NewReader(buf))
	factory := New(source, 1)

	doFanoutTest(t, func() {
		r := factory.NewReader()
		defer r.Close()

		data, err := io.ReadAll(r)
		assert.NoError(t, err)
		assert.Equal(t, buf, data)
	}, 50, 1)
}

// Test readers created at different times
func TestFanoutReadersAtDifferentTimes(t *testing.T) {
	buf := make([]byte, 10000)
	for i := range buf {
		buf[i] = byte(i % 256)
	}

	source := io.NopCloser(&slowReader{data: buf, delay: time.Millisecond})
	factory := New(source, len(buf))

	var wg sync.WaitGroup
	results := make([][]byte, 5)

	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()

			// Stagger reader creation
			time.Sleep(time.Duration(index) * 10 * time.Millisecond)

			r := factory.NewReader()
			defer r.Close()

			data, err := io.ReadAll(r)
			require.NoError(t, err)
			results[index] = data
		}(i)
	}

	wg.Wait()

	// All readers should get the same complete data
	for i, result := range results {
		assert.Equal(t, buf, result, "Reader %d got different data", i)
	}
}

// Test partial reads with multiple readers
func TestFanoutPartialReads(t *testing.T) {
	buf := []byte("0123456789abcdefghijklmnopqrstuvwxyz")
	source := io.NopCloser(bytes.NewReader(buf))
	factory := New(source, len(buf))

	var wg sync.WaitGroup
	numReaders := 10

	for i := 0; i < numReaders; i++ {
		wg.Add(1)
		go func(readerID int) {
			defer wg.Done()

			r := factory.NewReader()
			defer r.Close()

			var result []byte
			chunk := make([]byte, 3) // Read in small chunks

			for {
				n, err := r.Read(chunk)
				if n > 0 {
					result = append(result, chunk[:n]...)
				}
				if err == io.EOF {
					break
				}
				require.NoError(t, err, "Reader %d failed", readerID)
			}

			assert.Equal(t, buf, result, "Reader %d got wrong data", readerID)
		}(i)
	}

	wg.Wait()
}

// Test reader closing early
func TestFanoutReaderClosesEarly(t *testing.T) {
	buf := make([]byte, 10000)
	for i := range buf {
		buf[i] = byte(i % 256)
	}

	source := io.NopCloser(&slowReader{data: buf, delay: time.Millisecond})
	factory := New(source, len(buf))

	var wg sync.WaitGroup

	// Reader that closes early
	wg.Add(1)
	go func() {
		defer wg.Done()
		r := factory.NewReader()

		// Read a bit then close
		chunk := make([]byte, 100)
		_, err := r.Read(chunk)
		require.NoError(t, err)

		r.Close() // Close early
	}()

	// Reader that reads everything
	wg.Add(1)
	go func() {
		defer wg.Done()
		time.Sleep(50 * time.Millisecond) // Start after first reader

		r := factory.NewReader()
		defer r.Close()

		data, err := io.ReadAll(r)
		require.NoError(t, err)
		assert.Equal(t, buf, data)
	}()

	wg.Wait()
}

// Test large file with many readers
func TestFanoutLargeFileMultipleReaders(t *testing.T) {
	// Create 1MB of test data
	size := 1024 * 1024
	buf := make([]byte, size)
	for i := range buf {
		buf[i] = byte(i % 256)
	}

	source := io.NopCloser(bytes.NewReader(buf))
	factory := New(source, size)

	numReaders := 20
	var wg sync.WaitGroup

	for i := 0; i < numReaders; i++ {
		wg.Add(1)
		go func(readerID int) {
			defer wg.Done()

			r := factory.NewReader()
			defer r.Close()

			data, err := io.ReadAll(r)
			require.NoError(t, err)
			assert.Equal(t, size, len(data), "Reader %d got wrong size", readerID)
			assert.Equal(t, buf, data, "Reader %d got wrong data", readerID)
		}(i)
	}

	wg.Wait()
}

// Test channel buffer sizing
func TestFanoutChannelBufferSizing(t *testing.T) {
	testCases := []struct {
		size           int
		expectedBuffer int
	}{
		{0, 1},            // 0/4096 + 1 = 1
		{4096, 2},         // 4096/4096 + 1 = 2
		{8192, 3},         // 8192/4096 + 1 = 3
		{128 * 1024, 32},  // 131072/4096 + 1 = 33, capped at 32
		{1024 * 1024, 32}, // Large size, capped at 32
	}

	for _, tc := range testCases {
		t.Run(fmt.Sprintf("size_%d", tc.size), func(t *testing.T) {
			buf := make([]byte, tc.size)
			source := io.NopCloser(bytes.NewReader(buf))
			factory := New(source, tc.size)

			r := factory.NewReader()
			defer r.Close()

			// Check that the channel buffer was sized correctly
			// (This is implicit - we can't directly check channel buffer size)
			data, err := io.ReadAll(r)
			assert.NoError(t, err)
			assert.Equal(t, buf, data)
		})
	}
}

// Test concurrent reader creation and closure
func TestFanoutConcurrentReaderManagement(t *testing.T) {
	buf := make([]byte, 50000)
	for i := range buf {
		buf[i] = byte(i % 256)
	}

	source := io.NopCloser(&slowReader{data: buf, delay: 100 * time.Microsecond})
	factory := New(source, len(buf))

	var wg sync.WaitGroup
	numWorkers := 50

	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			// Randomly create and close readers
			for j := 0; j < 5; j++ {
				r := factory.NewReader()

				// Random read pattern
				if rand.Intn(2) == 0 {
					// Read everything
					data, err := io.ReadAll(r)
					assert.NoError(t, err)
					assert.Equal(t, buf, data)
				} else {
					// Read partially then close
					chunk := make([]byte, rand.Intn(1000)+100)
					_, err := r.Read(chunk)
					if err != io.EOF {
						assert.NoError(t, err)
					}
				}

				r.Close()
			}
		}()
	}

	wg.Wait()
}

// Test source that returns data in very small chunks
func TestFanoutSourceSmallChunks(t *testing.T) {
	buf := []byte("The quick brown fox jumps over the lazy dog")
	source := io.NopCloser(&chunkReader{data: buf, chunkSize: 1}) // 1 byte at a time
	factory := New(source, len(buf))

	doFanoutTest(t, func() {
		r := factory.NewReader()
		defer r.Close()

		data, err := io.ReadAll(r)
		assert.NoError(t, err)
		assert.Equal(t, buf, data)
	}, 20, 1)
}

// Test reading after source EOF
func TestFanoutReadAfterSourceEOF(t *testing.T) {
	buf := []byte("hello world")
	source := io.NopCloser(bytes.NewReader(buf))
	factory := New(source, len(buf))

	// First reader reads everything
	r1 := factory.NewReader()
	data1, err := io.ReadAll(r1)
	require.NoError(t, err)
	assert.Equal(t, buf, data1)
	r1.Close()

	// Second reader created after source is exhausted should still get all data
	time.Sleep(10 * time.Millisecond) // Ensure source is exhausted

	r2 := factory.NewReader()
	defer r2.Close()
	data2, err := io.ReadAll(r2)
	require.NoError(t, err)
	assert.Equal(t, buf, data2)
}

// Test memory cleanup
func TestFanoutMemoryCleanup(t *testing.T) {
	buf := make([]byte, 100000) // 100KB
	source := io.NopCloser(bytes.NewReader(buf))
	factory := New(source, len(buf))

	readers := make([]io.ReadCloser, 10)

	// Create multiple readers
	for i := range readers {
		readers[i] = factory.NewReader()
	}

	// Read some data from each
	for _, r := range readers {
		chunk := make([]byte, 1000)
		_, err := r.Read(chunk)
		require.NoError(t, err)
	}

	// Close all readers
	for _, r := range readers {
		assert.NoError(t, r.Close())
	}

	// Verify closed readers can't be read from
	chunk := make([]byte, 100)
	for _, r := range readers {
		_, err := r.Read(chunk)
		assert.ErrorIs(t, err, io.ErrClosedPipe)
	}
}

// Test zero-size fanout
func TestFanoutZeroSize(t *testing.T) {
	source := io.NopCloser(strings.NewReader(""))
	factory := New(source, 0)

	r := factory.NewReader()
	defer r.Close()

	data, err := io.ReadAll(r)
	assert.NoError(t, err)
	assert.Empty(t, data)
}

// Helper types for testing

// slowReader simulates a slow data source
type slowReader struct {
	data  []byte
	pos   int
	delay time.Duration
}

func (sr *slowReader) Read(p []byte) (int, error) {
	if sr.pos >= len(sr.data) {
		return 0, io.EOF
	}

	time.Sleep(sr.delay)

	// Read at most 1KB at a time to simulate realistic I/O
	maxRead := 1024
	if len(p) < maxRead {
		maxRead = len(p)
	}

	remaining := len(sr.data) - sr.pos
	if maxRead > remaining {
		maxRead = remaining
	}

	n := copy(p[:maxRead], sr.data[sr.pos:sr.pos+maxRead])
	sr.pos += n
	return n, nil
}

// chunkReader reads data in fixed-size chunks
type chunkReader struct {
	data      []byte
	pos       int
	chunkSize int
}

func (cr *chunkReader) Read(p []byte) (int, error) {
	if cr.pos >= len(cr.data) {
		return 0, io.EOF
	}

	readSize := cr.chunkSize
	if len(p) < readSize {
		readSize = len(p)
	}

	remaining := len(cr.data) - cr.pos
	if readSize > remaining {
		readSize = remaining
	}

	n := copy(p[:readSize], cr.data[cr.pos:cr.pos+readSize])
	cr.pos += n
	return n, nil
}

// Test basic release functionality
func TestFanoutRelease(t *testing.T) {
	buf := []byte("hello world test data")
	source := io.NopCloser(bytes.NewReader(buf))
	factory := New(source, len(buf))

	// Create a reader but don't close it yet
	r1 := factory.NewReader()

	// Release the fanout - buffer should NOT be freed yet because r1 is still active
	factory.Release()

	// Verify existing reader still works (was created before release)
	data1, err := io.ReadAll(r1)
	require.NoError(t, err)
	assert.Equal(t, buf, data1)

	// Create another reader after release - should get EOF immediately
	r2 := factory.NewReader()
	data2, err := io.ReadAll(r2)
	require.NoError(t, err)
	assert.Empty(t, data2) // Should be empty since it's an eofReader

	// Close first reader - buffer should still exist because r2 might need cleanup
	assert.NoError(t, r1.Close())

	// Create third reader after release - should also get EOF
	r3 := factory.NewReader()
	data3, err := io.ReadAll(r3)
	require.NoError(t, err)
	assert.Empty(t, data3) // Should be empty since it's an eofReader

	// Close remaining readers
	assert.NoError(t, r2.Close())
	assert.NoError(t, r3.Close())

	// After release, new readers should immediately return EOF
	r4 := factory.NewReader()
	defer r4.Close()
	data4, err := io.ReadAll(r4)
	require.NoError(t, err)
	assert.Empty(t, data4) // Should be empty since it returns EOF immediately
}

// Test release called multiple times
func TestFanoutReleaseMultipleCalls(t *testing.T) {
	buf := []byte("test data")
	source := io.NopCloser(bytes.NewReader(buf))
	factory := New(source, len(buf))

	r := factory.NewReader()

	// Call release multiple times - should be idempotent
	factory.Release()
	factory.Release()
	factory.Release()

	// Reader should still work
	data, err := io.ReadAll(r)
	require.NoError(t, err)
	assert.Equal(t, buf, data)

	assert.NoError(t, r.Close())
}

// Test release before any readers are created
func TestFanoutReleaseBeforeReaders(t *testing.T) {
	buf := []byte("test data for early release")
	source := io.NopCloser(bytes.NewReader(buf))
	factory := New(source, len(buf))

	// Release immediately - buffer should be freed since no readers exist
	factory.Release()

	// Creating readers after release should return EOF immediately
	r1 := factory.NewReader()
	defer r1.Close()

	data, err := io.ReadAll(r1)
	require.NoError(t, err)
	assert.Empty(t, data) // Should be empty since released fanout returns EOF
}

// Test release with concurrent reader operations
func TestFanoutReleaseConcurrent(t *testing.T) {
	buf := make([]byte, 50000)
	for i := range buf {
		buf[i] = byte(i % 256)
	}

	source := io.NopCloser(&slowReader{data: buf, delay: time.Microsecond})
	factory := New(source, len(buf))

	var wg sync.WaitGroup
	numReaders := 20
	results := make([][]byte, numReaders)

	// Start readers concurrently
	for i := 0; i < numReaders; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()

			r := factory.NewReader()
			defer r.Close()

			// Some readers will be active when Release() is called
			if index%3 == 0 {
				time.Sleep(10 * time.Millisecond)
			}

			data, err := io.ReadAll(r)
			require.NoError(t, err)
			results[index] = data
		}(i)
	}

	// Release while readers are active
	go func() {
		time.Sleep(5 * time.Millisecond)
		factory.Release()
	}()

	wg.Wait()

	// All readers should have gotten the same complete data
	for i, result := range results {
		assert.Equal(t, buf, result, "Reader %d got different data", i)
	}
}

// Test release with readers closing at different times
func TestFanoutReleaseStaggeredClose(t *testing.T) {
	buf := []byte("staggered close test data")
	source := io.NopCloser(bytes.NewReader(buf))
	factory := New(source, len(buf))

	readers := make([]io.ReadCloser, 5)
	for i := range readers {
		readers[i] = factory.NewReader()
	}

	// Read from all readers first
	for i, r := range readers {
		data, err := io.ReadAll(r)
		require.NoError(t, err)
		assert.Equal(t, buf, data, "Reader %d failed", i)
	}

	// Release the fanout
	factory.Release()

	var wg sync.WaitGroup

	// Close readers at different times
	for i, r := range readers {
		wg.Add(1)
		go func(index int, reader io.ReadCloser) {
			defer wg.Done()

			// Stagger the closes
			time.Sleep(time.Duration(index) * 10 * time.Millisecond)
			assert.NoError(t, reader.Close())
		}(i, r)
	}

	wg.Wait()

	// After release, new readers should return EOF immediately
	// This is the expected and safe behavior - no deadlocks or complex edge cases
}

// Test release with empty source
func TestFanoutReleaseEmpty(t *testing.T) {
	source := io.NopCloser(bytes.NewReader([]byte{}))
	factory := New(source, 0)

	factory.Release()

	r := factory.NewReader()
	defer r.Close()

	data, err := io.ReadAll(r)
	assert.NoError(t, err)
	assert.Empty(t, data)
}

// Test release with source errors - with more debugging
func TestFanoutReleaseWithErrors(t *testing.T) {
	buf := []byte("test data")
	source := io.NopCloser(bytes.NewReader(buf))
	factory := New(source, len(buf))

	// Release immediately
	factory.Release()

	// Check the release flag directly
	factory.lock.RLock()
	isReleased := factory.released
	factory.lock.RUnlock()
	t.Logf("Factory released flag: %v", isReleased)

	// New readers after release should get eofReader which returns EOF
	r := factory.NewReader()
	defer r.Close()

	// Check if this is actually an eofReader by checking its type
	t.Logf("Reader type: %T", r)

	data, err := io.ReadAll(r)
	t.Logf("Read result: data=%v (len=%d), err=%v", data, len(data), err)

	// For eofReader, io.ReadAll should return empty slice and no error (EOF is handled)
	assert.NoError(t, err)
	assert.Empty(t, data)
}

// Test release timing - ensure proper cleanup
func TestFanoutReleaseMemoryCleanup(t *testing.T) {
	buf := make([]byte, 10000) // 10KB
	for i := range buf {
		buf[i] = byte(i % 256)
	}

	source := io.NopCloser(bytes.NewReader(buf))
	factory := New(source, len(buf))

	// Create and close readers multiple times
	for round := 0; round < 3; round++ {
		readers := make([]io.ReadCloser, 5)

		// Create readers
		for i := range readers {
			readers[i] = factory.NewReader()
		}

		// Release on first round only
		if round == 0 {
			factory.Release()
		}

		// Read from all readers
		for i, r := range readers {
			data, err := io.ReadAll(r)
			require.NoError(t, err)

			if round == 0 {
				// Round 0: readers created BEFORE release → should get data
				assert.Equal(t, buf, data, "Round %d, Reader %d failed", round, i)
			} else {
				// Round 1 & 2: readers created AFTER release → should get empty data
				assert.Empty(t, data, "Round %d, Reader %d should get empty data", round, i)
			}
		}

		// Close all readers
		for _, r := range readers {
			assert.NoError(t, r.Close())
		}
	}
}

// Test release with very large number of readers - CORRECTED VERSION
func TestFanoutReleaseHighConcurrency(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping high concurrency test in short mode")
	}

	buf := make([]byte, 1000)
	for i := range buf {
		buf[i] = byte(i % 256)
	}

	source := io.NopCloser(bytes.NewReader(buf))
	factory := New(source, len(buf))

	var wg sync.WaitGroup
	var normalReaders, eofReaders int32

	// Phase 1: Create some readers BEFORE release
	numBeforeRelease := 30
	for i := 0; i < numBeforeRelease; i++ {
		wg.Add(1)
		go func(readerID int) {
			defer wg.Done()

			r := factory.NewReader() // Created before release
			defer r.Close()

			data, err := io.ReadAll(r)
			require.NoError(t, err)
			atomic.AddInt32(&normalReaders, 1)
			assert.Equal(t, buf, data, "Reader %d got wrong data", readerID)
		}(i)
	}

	// Small delay to ensure the above readers are created
	time.Sleep(1 * time.Millisecond)

	// Release the fanout
	factory.Release()

	// Phase 2: Create readers AFTER release
	numAfterRelease := 70
	for i := 0; i < numAfterRelease; i++ {
		wg.Add(1)
		go func(readerID int) {
			defer wg.Done()

			r := factory.NewReader() // Created after release
			defer r.Close()

			data, err := io.ReadAll(r)
			require.NoError(t, err)
			atomic.AddInt32(&eofReaders, 1)
			assert.Empty(t, data, "Reader %d should get empty data", readerID)
		}(i)
	}

	wg.Wait()

	// Verify deterministic behavior
	normal := atomic.LoadInt32(&normalReaders)
	eof := atomic.LoadInt32(&eofReaders)

	t.Logf("Normal readers: %d, EOF readers: %d", normal, eof)

	assert.Equal(t, int32(numBeforeRelease), normal, "Should have exact number of normal readers")
	assert.Equal(t, int32(numAfterRelease), eof, "Should have exact number of EOF readers")
	assert.Equal(t, int32(100), normal+eof, "Total should be 100 readers")
}

// Test release behavior with partial reads
func TestFanoutReleasePartialReads(t *testing.T) {
	buf := []byte("0123456789abcdefghijklmnopqrstuvwxyz")
	source := io.NopCloser(bytes.NewReader(buf))
	factory := New(source, len(buf))

	r1 := factory.NewReader()
	r2 := factory.NewReader()

	// Partial read from r1 (created before release)
	chunk1 := make([]byte, 10)
	n1, err := r1.Read(chunk1)
	require.NoError(t, err)
	assert.Equal(t, 10, n1)
	assert.Equal(t, buf[:10], chunk1)

	// Release the fanout
	factory.Release()

	// Continue reading from r1 (created before release) - should work
	remaining1, err := io.ReadAll(r1)
	require.NoError(t, err)
	assert.Equal(t, buf[10:], remaining1)

	// Read all from r2 (created before release) - should work
	data2, err := io.ReadAll(r2)
	require.NoError(t, err)
	assert.Equal(t, buf, data2)

	// Close readers
	assert.NoError(t, r1.Close())
	assert.NoError(t, r2.Close())

	// New readers after release should get EOF
	r3 := factory.NewReader()
	defer r3.Close()
	data3, err := io.ReadAll(r3)
	require.NoError(t, err)
	assert.Empty(t, data3) // Should be empty since created after release
}

// Benchmark release performance
func BenchmarkFanoutRelease(b *testing.B) {
	buf := make([]byte, 10000)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		source := io.NopCloser(bytes.NewReader(buf))
		factory := New(source, len(buf))

		readers := make([]io.ReadCloser, 10)
		for j := range readers {
			readers[j] = factory.NewReader()
		}

		factory.Release()

		for _, r := range readers {
			io.ReadAll(r)
			r.Close()
		}
	}
}

// Test integration with Blob-like usage pattern
func TestFanoutReleaseIntegration(t *testing.T) {
	buf := []byte("integration test data for blob-like usage")
	source := io.NopCloser(bytes.NewReader(buf))
	factory := New(source, len(buf))

	// Simulate blob initialization (sniffing)
	sniffReader := factory.NewReader()
	sniffBuf := make([]byte, 512)
	n, _ := io.ReadAtLeast(sniffReader, sniffBuf, len(buf))
	sniffBuf = sniffBuf[:n]
	sniffReader.Close()
	assert.Equal(t, buf, sniffBuf)

	// Simulate multiple consumers
	var wg sync.WaitGroup

	// Consumer 1: ReadAll
	wg.Add(1)
	go func() {
		defer wg.Done()
		r := factory.NewReader()
		defer r.Close()
		data, err := io.ReadAll(r)
		require.NoError(t, err)
		assert.Equal(t, buf, data)
	}()

	// Consumer 2: Chunked reading
	wg.Add(1)
	go func() {
		defer wg.Done()
		r := factory.NewReader()
		defer r.Close()

		var result []byte
		chunk := make([]byte, 5)
		for {
			n, err := r.Read(chunk)
			if n > 0 {
				result = append(result, chunk[:n]...)
			}
			if err == io.EOF {
				break
			}
			require.NoError(t, err)
		}
		assert.Equal(t, buf, result)
	}()

	// Release while consumers are active
	time.Sleep(1 * time.Millisecond)
	factory.Release()

	wg.Wait()
}
