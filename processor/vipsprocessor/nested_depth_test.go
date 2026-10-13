package vipsprocessor

import (
	"context"
	"fmt"
	"net/url"
	"testing"

	"github.com/cshum/imagor"
	"github.com/cshum/imagor/imagorpath"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// nestImagePath builds a path nesting image() `depth` levels deep, offsetting
// each level so every depth renders differently.
func nestImagePath(depth int) string {
	path := "100x100/color:red"
	for i := 1; i <= depth; i++ {
		path = fmt.Sprintf("400x400/filters:image(%s,%d,%d)/color:green",
			url.QueryEscape(path), i*5, i*5)
	}
	return path
}

// A path: source re-enters Do, so it is bounded by MaxNestedDepth. The image()
// filter nests for grouping instead: it resolves a nested path through
// imagorpath.Parse and loadFilterImage without calling Do, so a group of nested
// overlays is not the runaway recursion that bound guards against. It is
// routinely used deeper than the bound.
//
// This test fails if the depth counter is ever moved somewhere image() passes
// through, which would break legitimate deep groupings.
func TestImageFilterNestingIsNotBoundedByMaxNestedDepth(t *testing.T) {
	v := NewProcessor()
	load := func(image string) (*imagor.Blob, error) {
		return imagor.NewBlobFromFile("../../testdata/gopher.png"), nil
	}

	depth := imagor.MaxNestedDepth + 4

	params := imagorpath.Parse(nestImagePath(depth))
	require.Len(t, params.Filters, 1)
	require.Equal(t, "image", params.Filters[0].Name)

	out, err := v.Process(context.Background(),
		imagor.NewBlobFromBytes([]byte("seed")), params, load)
	require.NoError(t, err,
		"image() nested %d deep must render, not hit MaxNestedDepth", depth)
	require.NotNil(t, out)

	deep, err := out.ReadAll()
	require.NoError(t, err)
	assert.NotEmpty(t, deep)

	// The nesting must actually compose. Render one level shallower and require
	// different bytes: identical output would mean the nesting collapsed, and the
	// assertion above would then pass for the wrong reason.
	shallowerOut, err := v.Process(context.Background(),
		imagor.NewBlobFromBytes([]byte("seed")),
		imagorpath.Parse(nestImagePath(depth-1)), load)
	require.NoError(t, err)
	shallower, err := shallowerOut.ReadAll()
	require.NoError(t, err)

	assert.NotEqual(t, fmt.Sprintf("%x", deep), fmt.Sprintf("%x", shallower),
		"nesting %d vs %d levels must render differently, or image() is not composing",
		depth, depth-1)
}
