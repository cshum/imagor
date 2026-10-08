package vipsprocessor

import (
	"bytes"
	"strings"

	"github.com/cshum/imagor"
	"github.com/cshum/imagor/imagorpath"
	"github.com/cshum/imagor/svg"
)

// langFilterName selects a language in a multi-language SVG. The rewrite runs
// before the source is loaded, so it is not in the filter table, whose filters
// run on the image.
const langFilterName = "lang"

// langTags returns the accepted language tags, and whether the request asks for a
// language at all. The tags are a set, as in the renderer's own language option:
// the document's order decides which branch renders.
func langTags(p imagorpath.Params) ([]string, bool) {
	for _, f := range p.Filters {
		if f.Name != langFilterName {
			continue
		}
		var tags []string
		for _, tag := range strings.Split(f.Args, ",") {
			if tag = strings.TrimSpace(tag); tag != "" {
				tags = append(tags, tag)
			}
		}
		return tags, true
	}
	return nil, false
}

// selectLanguage returns the source with its language conditionals resolved for
// the requested tags.
func selectLanguage(blob *imagor.Blob, tags []string) (*imagor.Blob, error) {
	data, err := blob.ReadAll()
	if err != nil {
		return nil, err
	}
	out, err := svg.SelectLanguage(bytes.NewReader(data), tags)
	if err != nil {
		return nil, err
	}
	return imagor.NewBlobFromBytes(out), nil
}
