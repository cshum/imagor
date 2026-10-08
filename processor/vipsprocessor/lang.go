package vipsprocessor

import (
	"bytes"
	"strings"

	"github.com/cshum/imagor"
	"github.com/cshum/imagor/imagorpath"
	"github.com/cshum/imagor/svg"
)

// langFilterName selects a language in a multi-language SVG. The rewrite has to
// happen before the source is loaded, because it decides what the renderer
// draws - so it is not in the filter table, whose filters run on the image.
const langFilterName = "lang"

// langTags returns the accepted language tags, and whether the request asks for
// a language at all. The tags are a set: which branch renders is the document's
// order to decide, the same way the renderer's own language option behaves.
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
