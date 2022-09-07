package imagorpath

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestHasher(t *testing.T) {
	assert.Equal(t, "45/d8/ebb31bd4ed80c26e7c2572957ac3eb99d3db", DigestStorageHasher.Hash("foobar.jpg"))
	assert.Equal(t, "45/d8/ebb31bd4ed80c26e7c2572957ac3eb99d3db", SuffixResultStorageHasher.HashResult(Parse("fit-in/16x17/foobar.jpg")))
	assert.Equal(t, "aa/f4/c61ddcc5e8a2dabede0f3b482cd9aea9434d", DigestStorageHasher.Hash("path/to/foobar.jpg"))
	p := Params{
		FitIn: true, Width: 16, Height: 17, Image: "foobar",
	}
	fmt.Println(GeneratePath(p))
	assert.Equal(t, "d5/c2/804e5d81c475bee50f731db17ee613f43262", DigestResultStorageHasher.HashResult(p))
	p.Path = GeneratePath(p)
	assert.Equal(t, "d5/c2/804e5d81c475bee50f731db17ee613f43262", DigestResultStorageHasher.HashResult(p))
	p = Params{
		FitIn: true, Width: 16, Height: 17, Image: "foobar",
	}
	fmt.Println(GeneratePath(p))
	assert.Equal(t, "foobar.d5c2804e5d81c475bee5", SuffixResultStorageHasher.HashResult(p))
	p.Path = GeneratePath(p)
	assert.Equal(t, "foobar.d5c2804e5d81c475bee5", SuffixResultStorageHasher.HashResult(p))
	p = Params{
		Smart: true, Width: 17, Height: 19, Image: "example.com/foobar",
	}
	fmt.Println(GeneratePath(p))
	assert.Equal(t, "example.com/foobar.ddd349e092cda6d9c729", SuffixResultStorageHasher.HashResult(p))
	p = Params{
		VAlign: "top", Width: 166, Height: 169, Image: "foobar.jpg",
	}
	fmt.Println(GeneratePath(p))
	assert.Equal(t, "foobar.45d8ebb31bd4ed80c26e.jpg", SuffixResultStorageHasher.HashResult(p))
	p.Path = GeneratePath(p)
	assert.Equal(t, "foobar.45d8ebb31bd4ed80c26e.jpg", SuffixResultStorageHasher.HashResult(p))
}

func TestSuffixResultStorageHasher(t *testing.T) {
	p := Params{
		Smart: true, Width: 17, Height: 19, Image: "example.com/foobar.jpg",
		Filters: []Filter{{"format", "webp"}},
	}
	fmt.Println(GeneratePath(p))
	assert.Equal(t, "example.com/foobar.8aade9060badfcb289f9.webp", SuffixResultStorageHasher.HashResult(p))

	p = Params{
		Meta:  true,
		Smart: true, Width: 17, Height: 19, Image: "example.com/foobar.jpg",
	}
	fmt.Println(GeneratePath(p))
	assert.Equal(t, "example.com/foobar.d72ff6ef20ba41fa570c.json", SuffixResultStorageHasher.HashResult(p))

	p = Params{
		Meta:  true,
		Smart: true, Width: 17, Height: 19, Image: "example.com/foobar.jpg",
		Filters: []Filter{{"format", "webp"}},
	}
	fmt.Println(GeneratePath(p))
	assert.Equal(t, "example.com/foobar.c80ab0faf85b35a140a8.json", SuffixResultStorageHasher.HashResult(p))
}
