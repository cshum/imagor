package imagorpath

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestHasher(t *testing.T) {
	p := Parse("fit-in/16x17/foobar")
	assert.Equal(t, "d5/c2/804e5d81c475bee50f731db17ee613f43262", DigestResultStorageHasher.HashResult(p))
	p.Path = ""
	assert.Equal(t, "d5/c2/804e5d81c475bee50f731db17ee613f43262", DigestResultStorageHasher.HashResult(p))
	p = Parse("fit-in/16x17/foobar")
	assert.Equal(t, "foobar.d5c2804e5d81c475bee5", SuffixResultStorageHasher.HashResult(p))
	assert.Equal(t, "foobar.d5c2804e5d81c475bee5_16x17", SizeSuffixResultStorageHasher.HashResult(p))
	p.Path = ""
	assert.Equal(t, "foobar.d5c2804e5d81c475bee5", SuffixResultStorageHasher.HashResult(p))
	p = Parse("17x19/smart/example.com/foobar")
	assert.Equal(t, "example.com/foobar.ddd349e092cda6d9c729", SuffixResultStorageHasher.HashResult(p))
	p = Parse("smart/example.com/foobar")
	assert.Equal(t, "example.com/foobar.afa3503c0d76bc49eccd", SizeSuffixResultStorageHasher.HashResult(p))
	assert.Equal(t, "example.com/foobar.afa3503c0d76bc49eccd", SuffixResultStorageHasher.HashResult(p))
	p = Parse("166x169/top/foobar.jpg")
	assert.Equal(t, "foobar.45d8ebb31bd4ed80c26e.jpg", SuffixResultStorageHasher.HashResult(p))
	assert.Equal(t, "foobar.45d8ebb31bd4ed80c26e_166x169.jpg", SizeSuffixResultStorageHasher.HashResult(p))
	p.Path = ""
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
