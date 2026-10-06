package vipsprocessor

import (
	"bytes"
	"context"
	"fmt"
	"net/http"

	"github.com/cshum/imagor"
	"github.com/cshum/imagor/imagorpath"
	"github.com/cshum/imagor/svg"
	"github.com/cshum/vipsgen/vips"
	"go.uber.org/zap"
)

// maxSanitizeBytes caps the size of an SVG that is buffered for sanitization.
// Beyond it the request falls back to rasterizing, which is the behaviour
// without passthrough, rather than reading an unbounded document into memory.
// It is a variable so tests can exercise the limit.
var maxSanitizeBytes = 32 << 20

// passthroughEnabled reports whether the source format may pass through.
func (v *Processor) passthroughEnabled(t imagor.BlobType) bool {
	_, ok := v.passthroughFormats[t]
	return ok
}

// svgContentType is the content type of a passthrough SVG response.
const svgContentType = "image/svg+xml"

// SetPassthroughFormats implements imagor.PassthroughProcessor. It is called by
// the application at startup with the configured passthrough formats.
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

// passthroughHeaders are the response headers required to return source markup
// to a browser safely. The application also enforces these for any markup
// response, since a result cache hit replays only the bytes.
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
// and the source format has a passthrough policy.
//
// It returns handled == false (and no error) when the request should be
// processed normally, which is also the outcome when sanitization cannot be
// completed: falling back to rasterizing is the pre-passthrough behaviour and
// never serves unsanitized markup.
func (v *Processor) passthroughBlob(
	ctx context.Context, blob *imagor.Blob, p imagorpath.Params,
) (out *imagor.Blob, handled bool, err error) {
	if blob == nil || blob.IsEmpty() || p.Meta {
		return nil, false, nil
	}
	blobType := blob.BlobType()
	marked := imagorpath.HasFilter(p, imagor.PassthroughFilterName)
	explicit := explicitSVGFormat(p)
	// A format filter may have been injected by content negotiation rather than
	// written by the client, so it is not on its own a request for an operation.
	// The marker is this code's own bookkeeping and never one either.
	transformations := imagorpath.HasTransformations(p,
		"format", "fallback_format", "autojpg", imagor.PassthroughFilterName)

	if explicit {
		// format(svg) asks for the vector itself. Either the source is an SVG and
		// passthrough is enabled for it, or the request has to be refused:
		// answering with another format under the requested name is the silent
		// lie this filter used to tell.
		switch {
		case blobType != imagor.BlobTypeSVG:
			return nil, false, imagor.NewError(
				"format(svg) is only supported for svg sources", http.StatusBadRequest)
		case !v.passthroughEnabled(blobType):
			return nil, false, imagor.NewError(
				"format(svg) requires svg passthrough to be enabled", http.StatusBadRequest)
		case transformations:
			// The vector cannot also be resized, cropped or filtered here, and
			// quietly serving it untouched would drop the rest of the request.
			return nil, false, imagor.NewError(
				"format(svg) cannot be combined with transformations", http.StatusBadRequest)
		}
	} else if !marked || !v.passthroughEnabled(blobType) {
		return nil, false, nil
	} else if transformations {
		// The marker is only appended to no-op requests, but a crafted path can
		// carry it alongside an operation. Honour the operation, not the marker.
		if v.Debug {
			v.Logger.Warn("passthrough-marker-ignored", zap.Any("params", p))
		}
		return nil, false, nil
	}

	// A format the operator did not enable, or one that has not been considered
	// at all, is processed normally rather than served.
	policy := imagor.PassthroughPolicyOf(blobType)
	if policy == imagor.PassthroughRefused {
		return nil, false, nil
	}

	// The resolution guard runs on the passthrough path too, so serving the
	// source is not a way around the image bomb limits.
	if err = v.checkPassthroughResolution(ctx, blob); err != nil {
		return nil, false, err
	}

	if policy == imagor.PassthroughSanitize && v.SanitizeSVG {
		sanitized, sanitizeErr := v.sanitizeSVG(blob)
		if sanitizeErr != nil {
			if explicit {
				// The request asked for the vector specifically: failing loudly
				// beats answering with a raster under an svg path.
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
		blob.SetContentType(svgContentType)
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
		return nil, WrapErr(err)
	}
	out := imagor.NewBlobFromBytes(sanitized)
	out.SetContentType(svgContentType)
	out.Header = passthroughHeaders()
	return out, nil
}

// checkPassthroughResolution enforces the same dimension limits as the
// processed path, using a header-only load that does not decode pixels.
func (v *Processor) checkPassthroughResolution(ctx context.Context, blob *imagor.Blob) error {
	if blob.BlobType() == imagor.BlobTypeMemory {
		// Raw pixel blobs are produced by the processor itself and carry their
		// own dimensions; nothing to guard.
		return nil
	}
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
