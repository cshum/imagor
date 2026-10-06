package vipsprocessor

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/cshum/imagor"
	"github.com/cshum/imagor/imagorpath"
	"github.com/cshum/imagor/svg"
	"github.com/cshum/vipsgen/vips"
	"go.uber.org/zap"
)

// maxSanitizeBytes caps the size of an SVG buffered for sanitization; beyond it
// the request rasterizes instead of reading an unbounded document into memory. A
// variable so tests can exercise the limit.
var maxSanitizeBytes = 32 << 20

// passthroughEnabled reports whether the source format may pass through.
func (v *Processor) passthroughEnabled(t imagor.BlobType) bool {
	_, ok := v.passthroughFormats[t]
	return ok
}

// SetPassthroughFormats implements imagor.PassthroughProcessor: the application
// calls it at startup with the configured formats.
func (v *Processor) SetPassthroughFormats(formats []imagor.BlobType) {
	if len(formats) == 0 {
		v.passthroughFormats = nil
		return
	}
	m := make(map[imagor.BlobType]struct{}, len(formats))
	for _, f := range formats {
		m[f] = struct{}{}
	}
	v.passthroughFormats = m
}

// passthroughHeaders are the headers needed to return source markup to a
// browser safely. The application enforces them for any markup response too,
// since a result cache hit replays only the bytes.
func passthroughHeaders() http.Header {
	return http.Header{
		"Content-Security-Policy": {imagor.SVGContentSecurityPolicy},
		"X-Content-Type-Options":  {"nosniff"},
	}
}

// explicitSVGFormat reports whether the request names svg as its output format.
func explicitSVGFormat(p imagorpath.Params) bool {
	for _, f := range p.Filters {
		if f.Name == "format" {
			return f.Args == "svg"
		}
	}
	return false
}

// passthroughBlob serves the source blob untouched when the request is eligible
// and the source format may pass through. handled is false when the request
// should be processed normally, which is also the outcome when sanitization
// cannot be completed: rasterizing never serves unsanitized markup.
func (v *Processor) passthroughBlob(
	ctx context.Context, blob *imagor.Blob, p imagorpath.Params,
) (out *imagor.Blob, handled bool, err error) {
	if blob == nil || blob.IsEmpty() || p.Meta {
		return nil, false, nil
	}
	blobType := blob.BlobType()
	marked := imagorpath.HasFilter(p, imagor.PassthroughFilterName)
	explicit := explicitSVGFormat(p)
	// A format filter may come from content negotiation rather than the client,
	// so it is not on its own an operation. The marker is this code's own
	// bookkeeping, and never one.
	transformations := imagorpath.HasTransformations(p,
		"format", "fallback_format", "autojpg", imagor.PassthroughFilterName)

	if explicit {
		// format(svg) asks for the vector itself. That is an explicit request
		// rather than a change of default, so the sanitizer makes it safe to
		// honour without passthrough being enabled for SVG generally. The
		// exception is a deployment that turned sanitization off: there it would
		// serve upstream markup untouched, so it needs the opt-in instead of
		// being a way around it.
		switch {
		case blobType != imagor.BlobTypeSVG:
			return nil, false, imagor.NewError(
				"format(svg) is only supported for svg sources", http.StatusBadRequest)
		case transformations:
			return nil, false, imagor.NewError(
				"format(svg) cannot be combined with transformations", http.StatusBadRequest)
		case !v.SanitizeSVG && !v.passthroughEnabled(blobType):
			return nil, false, imagor.NewError(
				"format(svg) requires svg passthrough to be enabled when svg sanitization is off",
				http.StatusBadRequest)
		}
	} else if !marked || !v.passthroughEnabled(blobType) {
		return nil, false, nil
	} else if transformations {
		// The marker is only appended to no-op requests, but a crafted path can
		// carry it next to an operation. Honour the operation. A client-named
		// format is not covered by this, since content negotiation reuses that
		// filter name and the two cannot be told apart.
		if v.Debug {
			v.Logger.Warn("passthrough-marker-ignored", zap.Any("params", p))
		}
		return nil, false, nil
	}

	// A format the operator did not enable is processed normally.
	policy := imagor.PassthroughPolicyOf(blobType)
	if policy == imagor.PassthroughRefused {
		return nil, false, nil
	}

	// The resolution guard runs here too, so passthrough is not a way around the
	// image bomb limits.
	if err = v.checkPassthroughResolution(ctx, blob); err != nil {
		return nil, false, err
	}

	if policy == imagor.PassthroughSanitize && v.SanitizeSVG {
		sanitized, sanitizeErr := v.sanitizeSVG(blob)
		if sanitizeErr != nil {
			if explicit {
				// The request named the vector: fail rather than answer with a
				// raster under an svg path. A document the sanitizer refuses is
				// reported as a bad request, like the cases above.
				return nil, false, sanitizeErr
			}
			if v.Debug {
				v.Logger.Warn("svg-sanitize-fallback", zap.Error(sanitizeErr))
			}
			return nil, false, nil
		}
		if v.Debug {
			v.Logger.Debug("passthrough", zap.String("format", "svg"), zap.Bool("sanitize", true))
		}
		return sanitized, true, nil
	}

	// Passive formats, or SVG with sanitization disabled: stream the source.
	blob.Header = passthroughHeaders()
	if blobType == imagor.BlobTypeSVG && blob.ContentType() == "" {
		blob.SetContentType(imagor.SVGContentType)
	}
	if v.Debug {
		v.Logger.Debug("passthrough",
			zap.String("format", fmt.Sprintf("%v", blobType)), zap.Bool("sanitize", false))
	}
	return blob, true, nil
}

// sanitizeSVG reads the source document and returns a sanitized SVG blob with
// the passthrough response headers attached.
func (v *Processor) sanitizeSVG(blob *imagor.Blob) (*imagor.Blob, error) {
	data, err := blob.ReadAll()
	if err != nil {
		return nil, WrapErr(err)
	}
	if len(data) > maxSanitizeBytes {
		return nil, imagor.NewError(
			fmt.Sprintf("svg source of %d bytes exceeds the sanitize limit of %d bytes",
				len(data), maxSanitizeBytes), http.StatusUnprocessableEntity)
	}
	sanitized, err := svg.Sanitize(bytes.NewReader(data))
	if err != nil {
		if errors.Is(err, svg.ErrInvalidSVG) || errors.Is(err, svg.ErrRenderingChanged) {
			// The document cannot be served as a vector, which is something the
			// caller can act on - another source, or no format(svg) - so it
			// answers like the other refusals rather than as an internal error.
			return nil, imagor.NewError(err.Error(), http.StatusBadRequest)
		}
		return nil, WrapErr(err)
	}
	out := imagor.NewBlobFromBytes(sanitized)
	out.SetContentType(imagor.SVGContentType)
	out.Header = passthroughHeaders()
	return out, nil
}

// checkPassthroughResolution enforces the same dimension limits as the
// processed path, using a header-only load that does not decode pixels.
func (v *Processor) checkPassthroughResolution(ctx context.Context, blob *imagor.Blob) error {
	options := &vips.LoadOptions{}
	options.Unlimited = v.Unlimited && unlimitedSupportedByLoader(blob)
	img, err := v.newImageFromBlob(ctx, blob, options)
	if err != nil {
		return WrapErr(err)
	}
	if _, err = v.CheckResolution(img, nil); err != nil {
		return err
	}
	img.Close()
	return nil
}
