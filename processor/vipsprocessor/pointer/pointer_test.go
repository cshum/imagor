package pointer

import (
	"sync"
	"testing"
	"unsafe"

	"github.com/stretchr/testify/assert"
)

func TestPointer(t *testing.T) {
	t.Cleanup(Clear)
	//	assert := makeAssert(t)
	assert.Len(t, store, 0)
	assert.Len(t, free, 0)
	assert.Len(t, blocks, 0)
	mutex.Lock()
	assert.Equal(t, unsafe.Pointer(nil), Save(nil))
	assert.Nil(t, Restore(nil))
	Unref(nil)
	mutex.Unlock()
	i1 := Save("foo")
	i2 := Save("bar")
	i3 := Save("baz")
	assert.Len(t, store, 3)
	assert.Len(t, free, blockSize-3)
	assert.Len(t, blocks, 1)
	var x interface{}
	x = Restore(i1)
	assert.NotNil(t, x)
	if s, ok := x.(string); ok {
		assert.Equal(t, "foo", s)
	} else {
		t.Fail()
	}
	x = Restore(unsafe.Pointer(&x))
	assert.Nil(t, x)
	x = Restore(i3)
	assert.NotNil(t, x)
	if s, ok := x.(string); ok {
		assert.Equal(t, "baz", s)
	} else {
		t.Fail()
	}
	Unref(i3)
	x = Restore(i3)
	assert.Nil(t, x)
	Unref(i2)
	Unref(i1)
	assert.Len(t, store, 0)
	assert.Len(t, free, blockSize)
	assert.Len(t, blocks, 1)
	i3 = Save("baz")
	assert.Len(t, store, 1)
	assert.Len(t, free, blockSize-1)
	Clear()
	assert.Len(t, store, 0)
	assert.Len(t, free, 0)
	assert.Len(t, blocks, 0)
}

func TestPointerIndexing(t *testing.T) {
	t.Cleanup(Clear)
	assert.Len(t, store, 0)
	assert.Len(t, free, 0)
	assert.Len(t, blocks, 0)

	i1 := Save("foo")
	i2 := Save("bar")
	_ = Save("baz")
	_ = Save("wibble")
	_ = Save("wabble")
	assert.Len(t, store, 5)
	assert.Len(t, free, blockSize-5)

	// Check that when we remove the first items inserted into the map there are
	// no subsequent issues
	Unref(i1)
	Unref(i2)
	assert.Len(t, free, blockSize-3)
	_ = Save("flim")
	ilast := Save("flam")
	assert.Len(t, store, 5)
	assert.Len(t, free, blockSize-5)

	x := Restore(ilast)
	assert.NotNil(t, x)
	if s, ok := x.(string); ok {
		assert.Equal(t, "flam", s)
	}
}

func TestCallbacksData(t *testing.T) {
	t.Cleanup(Clear)
	assert.Len(t, store, 0)
	assert.Len(t, free, 0)
	assert.Len(t, blocks, 0)

	// insert a plain function
	i1 := Save(func(v int) int { return v + 1 })

	// insert a type "containing" a function, note that it doesn't
	// actually have a callable function. Users of the type must
	// check that themselves
	type flup struct {
		Stuff int
		Junk  func(int, int) error
	}
	i2 := Save(flup{
		Stuff: 55,
	})

	// did we get a function back
	x1 := Restore(i1)
	if assert.NotNil(t, x1) {
		if f, ok := x1.(func(v int) int); ok {
			assert.Equal(t, 2, f(1))
		} else {
			t.Fatalf("conversion failed")
		}
	}

	// did we get our data structure back
	x2 := Restore(i2)
	if assert.NotNil(t, x2) {
		if d, ok := x2.(flup); ok {
			assert.Equal(t, 55, d.Stuff)
			assert.Nil(t, d.Junk)
		} else {
			t.Fatalf("conversion failed")
		}
	}
}

func BenchmarkPointer(t *testing.B) {
	t.Cleanup(Clear)
	assert.Len(t, store, 0)
	assert.Len(t, free, 0)
	assert.Len(t, blocks, 0)
	const workers = 1000
	var wg sync.WaitGroup
	f := func() {
		defer wg.Done()
		var x interface{}
		var i1, i2, i3 unsafe.Pointer
		for i := 0; i < t.N/workers; i++ {
			i1 = Save("foo")
			i2 = Save("bar")
			i3 = Save("baz")
			x = Restore(i1)
			assert.NotNil(t, x)
			if s, ok := x.(string); ok {
				assert.Equal(t, "foo", s)
			} else {
				t.Fail()
			}
			x = Restore(i3)
			assert.NotNil(t, x)
			if s, ok := x.(string); ok {
				assert.Equal(t, "baz", s)
			} else {
				t.Fail()
			}
			Unref(i3)
			Unref(i2)
			Unref(i1)
		}
	}
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go f()
	}
	wg.Wait()
	assert.Len(t, store, 0)
	assert.Len(t, free, len(blocks)*blockSize)
}
