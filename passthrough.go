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
	// nosniff are enough. Nothing is configured with this policy in the first
	// release - see PassthroughSanitize and the configuration rule in
	// ParsePassthroughFormats - but a library embedder can still enable one with
	// WithPassthroughFormats, and the pass-through path supports it.
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

// passthroughFormatNameList lists the names the configuration accepts, in a
// stable order, so a rejected value can say what was expected. It follows the
// same rule as ParsePassthroughFormats.
func passthroughFormatNameList() []string {
	names := make([]string, 0, len(passthroughFormatNames))
	for name, t := range passthroughFormatNames {
		if PassthroughPolicyOf(t) == PassthroughSanitize {
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

// ParsePassthroughFormats resolves configuration names to source formats.
//
// A name is accepted only for a format that is sanitized before it is served -
// svg, in this release. The still formats are classified by PassthroughPolicyOf
// and supported by the pass-through path, but enabling one is a decision about
// serving a codec's bytes unchanged, so it is not a configuration switch yet;
// a library embedder that wants it can set it with WithPassthroughFormats.
// Everything else - including a name that only looks close to a real one - is an
// error, so a typo cannot leave the feature quietly off.
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
		if PassthroughPolicyOf(t) != PassthroughSanitize {
			return nil, fmt.Errorf(
				"imagor: passthrough format %q is not configurable: only a source that is sanitized before it is served, such as svg, can be set here", name)
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
