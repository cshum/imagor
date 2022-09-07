package imagorpath

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestHasher(t *testing.T) {
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
