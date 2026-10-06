package imagor

import (
	"fmt"
	"strings"

	"github.com/cshum/imagor/imagorpath"
)

// PassthroughFilterName is the filter the application appends to a request that
// is eligible for source passthrough - that is, a request that names no output
// format while passthrough is enabled for some source format.
//
// The marker is appended before the path is regenerated, so it becomes part of
// the result storage key: a passthrough result and the rasterized result of the
// same request never share a key. Without it, enabling passthrough would leave
// already-cached rasterized results (and their content type) to be served for
// requests that should now stream the source untouched.
const PassthroughFilterName = "passthrough"

// PassthroughPolicy describes how a source format may be served untouched.
type PassthroughPolicy int

const (
	// PassthroughRefused formats are never served as-is. This is the default for
	// anything unrecognised, and covers formats whose bytes are active content
	// in a browser context - a PDF viewer executes embedded script, so serving
	// one inline from the proxy origin is the same problem as an unsanitized
	// SVG, without a sanitizer to fix it.
	PassthroughRefused PassthroughPolicy = iota
	// PassthroughSanitize formats are markup whose bytes must be sanitized
	// before they are returned to a browser.
	PassthroughSanitize
	// PassthroughPassive formats are not executable. Serving them with a sniffed
	// content type and nosniff is safe.
	PassthroughPassive
)

// PassthroughPolicyOf returns the passthrough policy for a source format.
// Unknown and unlisted formats are refused: a format is only ever passed
// through once it has been explicitly considered here.
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

// SVGContentSecurityPolicy is the response policy for markup responses. It
// allows inline style (SVG presentation often relies on it) and data: images,
// and nothing else: no script, no external CSS, no external image or font, so a
// document cannot reach out from the origin serving it.
//
// script-src 'none' alone is not enough - it does not restrict CSS or image
// loads, which is where an unsanitized SVG would still make requests.
const SVGContentSecurityPolicy = "default-src 'none'; style-src 'unsafe-inline'; img-src data:"

// PassthroughFormatNames returns the configuration names for the given source
// formats, for logging and diagnostics.
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

// ParsePassthroughFormats resolves configured source format names to formats.
//
// A name whose policy is PassthroughRefused is an error rather than a silent
// no-op: adding a format to the passthrough list is a decision about serving
// unprocessed bytes from the proxy origin, and it has to be made deliberately.
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
			return nil, fmt.Errorf("imagor: unknown passthrough format %q", name)
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
// untouched. The application forwards its configured passthrough formats at
// startup, so the processor and the result storage key always agree on which
// requests pass through.
type PassthroughProcessor interface {
	SetPassthroughFormats(formats []BlobType)
}

// passthroughEligible reports whether a request asks for nothing at all - no
// resize, crop, alignment, padding, flip, trim, smart crop or filter.
//
// Passthrough is limited to genuine no-op requests on purpose. A request that
// does ask for a transformation must never have it silently dropped: a crop on
// an SVG has to rasterize, because returning the untouched vector would answer
// a different question than the one asked.
//
// The test runs before auto WebP/AVIF negotiation appends its own format
// filter, so a format named here is one the client asked for, and asking for a
// format is not a no-op.
func passthroughEligible(p imagorpath.Params) bool {
	return !imagorpath.HasTransformations(p)
}
