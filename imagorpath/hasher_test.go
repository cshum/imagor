package imagorpath

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestHasher(t *testing.T) {
	assert.Equal(t, "aa/f4c61ddcc5e8a2dabede0f3b482cd9aea9434d", DigestStorageHasher.Hash("hello"))
	p := Params{
		FitIn: true, Width: 16, Height: 17, Image: "foobar",
	}
	assert.Equal(t, "d5/c2804e5d81c475bee50f731db17ee613f43262", DigestResultStorageHasher.HashResult(p))
	p.Path = GeneratePath(p)
	assert.Equal(t, "d5/c2804e5d81c475bee50f731db17ee613f43262", DigestResultStorageHasher.HashResult(p))
	p = Params{
		FitIn: true, Width: 16, Height: 17, Image: "foobar",
	}
	assert.Equal(t, "foobar.d5c2804e5d81c475bee5", SuffixResultStorageHasher.HashResult(p))
	p.Path = GeneratePath(p)
	assert.Equal(t, "foobar.d5c2804e5d81c475bee5", SuffixResultStorageHasher.HashResult(p))
	p = Params{
		Smart: true, Width: 17, Height: 19, Image: "example.com/foobar",
	}
	assert.Equal(t, "example.com/foobar.ddd349e092cda6d9c729", SuffixResultStorageHasher.HashResult(p))
	p = Params{
		VAlign: "top", Width: 166, Height: 169, Image: "foobar.jpg",
	}
	assert.Equal(t, "foobar.45d8ebb31bd4ed80c26e.jpg", SuffixResultStorageHasher.HashResult(p))
	p.Path = GeneratePath(p)
	assert.Equal(t, "foobar.45d8ebb31bd4ed80c26e.jpg", SuffixResultStorageHasher.HashResult(p))
}
