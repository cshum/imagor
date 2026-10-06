package imagor

import (
	"fmt"
	"sort"
	"strings"

	"github.com/cshum/imagor/imagorpath"
)

// PassthroughFilterName is the internal marker the application appends to a
// request that asks for no transformation, when passthrough is enabled for its
// source format. It is part of the result storage key, so a passthrough result
// and a rasterized result of the same request never share a cache entry.
//
// It is not a client-facing filter: it has no implementation of its own, and
// nothing in the documentation lists it.
const PassthroughFilterName = "passthrough"

// PassthroughPolicy describes how a source format may be served untouched.
type PassthroughPolicy int

const (
	// PassthroughRefused formats are never served as-is - the default for
	// anything unrecognised. PDF belongs here: a PDF viewer runs embedded
	// script, the problem an unsanitized SVG poses, with no sanitizer to answer
	// it.
	PassthroughRefused PassthroughPolicy = iota
	// PassthroughSanitize formats are markup that must be sanitized before it
	// reaches a browser.
	PassthroughSanitize
	// PassthroughPassive formats are not executable. A sniffed content type and
	// nosniff are enough.
	PassthroughPassive
)

// PassthroughPolicyOf returns the policy for a source format. Unknown formats
// are refused: a format passes through only once it has been considered here.
func PassthroughPolicyOf(t BlobType) PassthroughPolicy {
	switch t {
	case BlobTypeSVG:
		return PassthroughSanitize
	case BlobTypePNG, BlobTypeJPEG, BlobTypeGIF, BlobTypeWEBP,
		BlobTypeAVIF, BlobTypeHEIF, BlobTypeTIFF, BlobTypeBMP,
		BlobTypeJXL, BlobTypeJP2:
		return PassthroughPassive
	default:
		return PassthroughRefused
	}
}

// passthroughFormatNames maps configuration names to source formats.
var passthroughFormatNames = map[string]BlobType{
	"svg": BlobTypeSVG, "png": BlobTypePNG, "jpeg": BlobTypeJPEG,
	"jpg": BlobTypeJPEG, "gif": BlobTypeGIF, "webp": BlobTypeWEBP,
	"avif": BlobTypeAVIF, "heif": BlobTypeHEIF, "tiff": BlobTypeTIFF,
	"bmp": BlobTypeBMP, "jxl": BlobTypeJXL, "jp2": BlobTypeJP2,
	"pdf": BlobTypePDF,
}

// SVGContentType is the content type of an SVG response.
const SVGContentType = "image/svg+xml"

// SVGContentSecurityPolicy is the response policy for markup responses: inline
// style and data: images, nothing else. script-src 'none' alone would not
// restrict CSS or image loads, which is how an SVG still reaches out from the
// origin serving it.
const SVGContentSecurityPolicy = "default-src 'none'; style-src 'unsafe-inline'; img-src data:"

// passthroughFormatNameList lists the configuration names that may be enabled,
// in a stable order, so a rejected value can say what was expected.
func passthroughFormatNameList() []string {
	names := make([]string, 0, len(passthroughFormatNames))
	for name, t := range passthroughFormatNames {
		if PassthroughPolicyOf(t) != PassthroughRefused {
			names = append(names, name)
		}
	}
	sort.Strings(names)
	return names
}

// PassthroughFormatNames returns configuration names for the given formats.
func PassthroughFormatNames(formats []BlobType) []string {
	var names []string
	for name, t := range passthroughFormatNames {
		for _, f := range formats {
			if f == t {
				names = append(names, name)
				break
			}
		}
	}
	return names
}

// ParsePassthroughFormats resolves configured format names to formats. A name
// that is refused or unknown is an error rather than a silent no-op: adding one
// is a decision about serving unprocessed bytes, and it has to be deliberate.
func ParsePassthroughFormats(names []string) ([]BlobType, error) {
	var formats []BlobType
	var seen = map[BlobType]struct{}{}
	for _, raw := range names {
		name := strings.ToLower(strings.TrimSpace(raw))
		if name == "" {
			continue
		}
		t, ok := passthroughFormatNames[name]
		if !ok {
			return nil, fmt.Errorf("imagor: unknown passthrough format %q, expected one of %s",
				name, strings.Join(passthroughFormatNameList(), ", "))
		}
		if PassthroughPolicyOf(t) == PassthroughRefused {
			return nil, fmt.Errorf(
				"imagor: passthrough format %q is refused: its bytes are active content in a browser", name)
		}
		if _, ok := seen[t]; ok {
			continue
		}
		seen[t] = struct{}{}
		formats = append(formats, t)
	}
	return formats, nil
}

// PassthroughProcessor is implemented by processors that can serve a source blob
// untouched. The application forwards its formats at startup, so the processor
// and the result storage key agree on which requests pass through.
type PassthroughProcessor interface {
	SetPassthroughFormats(formats []BlobType)
}

// passthroughEligible reports whether a request asks for nothing at all.
//
// Only no-op requests qualify: a request that asks for a transformation must
// never have it dropped, since the untouched source would answer a different
// question. It runs before auto WebP/AVIF appends its own format filter, so a
// format here is one the client asked for, and asking for a format is not a
// no-op.
func passthroughEligible(p imagorpath.Params) bool {
	return !imagorpath.HasTransformations(p)
}
