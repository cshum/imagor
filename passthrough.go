package imagor

import (
	"bytes"
	"errors"
	"fmt"
	"net/http"
	"sort"
	"strings"

	"github.com/cshum/imagor/imagorpath"
	"github.com/cshum/imagor/svg"
	"go.uber.org/zap"
)

// PassthroughFilterName is the internal marker the application appends to a
// request that asks for no transformation, when passthrough is enabled for its
// source format. It lands in the result storage key, so a passthrough result and
// a rasterized result of the same request never share a cache entry. It is not a
// client-facing filter and nothing documents it.
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
	// PassthroughPassive formats are not executable: a sniffed content type and
	// nosniff are enough. No configured format uses it - only markup that is
	// sanitized before it is served can be set - but a library embedder may.
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

// SVGContentSecurityPolicy is the policy for markup responses: inline style and
// data: images, nothing else, so CSS and image loads cannot reach out from the
// origin serving the document either. sandbox is the backstop for a document
// served with sanitization off.
const SVGContentSecurityPolicy = "default-src 'none'; style-src 'unsafe-inline'; img-src data:; sandbox"

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

// ParsePassthroughFormats resolves configured format names to formats. A name is
// accepted once PassthroughPolicyOf has considered it; anything else is an error
// rather than a silent no-op, since serving unprocessed bytes has to be
// deliberate.
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

// servePassthrough returns the source blob as it should be served when the
// request asks for nothing and its source format may pass through. handled is
// false when the request should be processed normally, which is also the outcome
// when sanitization cannot be completed: nothing unsanitized is served.
//
// Serving is not rendering. The document is handed over as it came, sanitized,
// so the resolution and image bomb limits - which bound what imagor renders -
// are not measured here, the same as raw().
func (app *Imagor) servePassthrough(p imagorpath.Params, blob *Blob) (out *Blob, handled bool, err error) {
	if blob == nil || blob.IsEmpty() || p.Meta {
		return nil, false, nil
	}
	blobType := blob.BlobType()
	explicit := explicitSVGFormat(p)
	if !explicit {
		// Only a marked request for a configured format is served untouched. The
		// marker is appended by the application; a client can write it too, which
		// the transformations check below answers.
		if len(app.PassthroughFormats) == 0 || !imagorpath.HasFilter(p, PassthroughFilterName) ||
			!app.passthroughEnabled(blobType) {
			return nil, false, nil
		}
	}
	// A format filter may come from content negotiation rather than the client,
	// so it is not on its own an operation. The marker is internal bookkeeping
	// and never one.
	transformations := imagorpath.HasTransformations(p, passthroughIgnoreFilters...)

	if explicit {
		// format(svg) asks for the vector itself. That is an explicit request
		// rather than a change of default, so the sanitizer makes it safe to
		// honour without passthrough being enabled for svg generally. The
		// exception is a deployment that turned sanitization off: there it would
		// serve the source untouched, so it needs the opt-in instead of being a
		// way around it.
		switch {
		case blobType != BlobTypeSVG:
			return nil, false, NewError(
				"format(svg) is only supported for svg sources", http.StatusBadRequest)
		case transformations:
			return nil, false, NewError(
				"format(svg) cannot be combined with transformations", http.StatusBadRequest)
		case !app.SanitizeSVG && !app.passthroughEnabled(blobType):
			return nil, false, NewError(
				"format(svg) requires svg passthrough to be enabled when svg sanitization is off",
				http.StatusBadRequest)
		}
	} else if transformations {
		// The marker is only appended to no-op requests, but a crafted path can
		// carry it next to an operation. Honour the operation. A client-named
		// format is not covered by this, since content negotiation reuses that
		// filter name and the two cannot be told apart.
		if app.Debug {
			app.Logger.Warn("passthrough-marker-ignored", zap.Any("params", p))
		}
		return nil, false, nil
	}

	policy := PassthroughPolicyOf(blobType)
	if policy == PassthroughRefused {
		return nil, false, nil
	}
	if policy == PassthroughSanitize && app.SanitizeSVG {
		sanitized, sanitizeErr := app.sanitizeSVG(blob)
		if sanitizeErr != nil {
			if explicit {
				// The request named the vector: fail rather than answer with a
				// raster under an svg path.
				return nil, false, sanitizeErr
			}
			if app.Debug {
				app.Logger.Warn("svg-sanitize-fallback", zap.Error(sanitizeErr))
			}
			return nil, false, nil
		}
		sanitized.SetContentType(SVGContentType)
		if app.Debug {
			app.Logger.Debug("passthrough", zap.String("format", "svg"), zap.Bool("sanitize", true))
		}
		return sanitized, true, nil
	}

	// A still format, or svg with sanitization off: the source is streamed.
	if blobType == BlobTypeSVG && blob.ContentType() == "" {
		blob.SetContentType(SVGContentType)
	}
	if app.Debug {
		app.Logger.Debug("passthrough",
			zap.String("format", fmt.Sprintf("%v", blobType)), zap.Bool("sanitize", false))
	}
	return blob, true, nil
}

// passthroughIgnoreFilters are filters that do not make a request an operation:
// content negotiation names one of them itself, and the marker is internal.
var passthroughIgnoreFilters = []string{
	"format", "fallback_format", "autojpg", PassthroughFilterName,
}

// explicitSVGFormat reports whether the request names svg as its output format.
func explicitSVGFormat(p imagorpath.Params) bool {
	for _, f := range p.Filters {
		if f.Name == "format" && strings.EqualFold(strings.TrimSpace(f.Args), "svg") {
			return true
		}
	}
	return false
}

// passthroughEnabled reports whether the source format may be served untouched.
func (app *Imagor) passthroughEnabled(t BlobType) bool {
	for _, f := range app.PassthroughFormats {
		if f == t {
			return true
		}
	}
	return false
}

// maxSanitizeBytes caps the size of an SVG buffered for sanitization; beyond it
// the document rasterizes instead of being read into memory unbounded. A
// variable so tests can exercise the limit.
var maxSanitizeBytes = 32 << 20

// sanitizeSVG reads the source document and returns a sanitized copy.
func (app *Imagor) sanitizeSVG(blob *Blob) (*Blob, error) {
	data, err := blob.ReadAll()
	if err != nil {
		return nil, WrapError(err)
	}
	if len(data) > maxSanitizeBytes {
		return nil, NewError(fmt.Sprintf(
			"svg source of %d bytes exceeds the sanitize limit of %d bytes",
			len(data), maxSanitizeBytes), http.StatusUnprocessableEntity)
	}
	sanitized, err := svg.Sanitize(bytes.NewReader(data))
	if err != nil {
		if errors.Is(err, svg.ErrInvalidSVG) || errors.Is(err, svg.ErrRenderingChanged) {
			// The document cannot be served as a vector, which is something the
			// caller can act on, so it answers like the other refusals.
			return nil, NewError(err.Error(), http.StatusBadRequest)
		}
		return nil, WrapError(err)
	}
	return NewBlobFromBytes(sanitized), nil
}

// passthroughEligible reports whether a request asks for nothing at all: a
// transformation must never be dropped, since the untouched source would answer
// a different question. It runs before auto WebP/AVIF appends its own format
// filter, so a format here is one the client asked for, and that is not a no-op.
func passthroughEligible(p imagorpath.Params) bool {
	return !imagorpath.HasTransformations(p)
}
