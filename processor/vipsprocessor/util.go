package vipsprocessor

import (
	"encoding/base64"
	"net/url"
	"strconv"
	"strings"

	"github.com/cshum/vipsgen/vips"
)

// linearRGB applies linear RGB transformation to an image
// Automatically handles alpha channel if present
func linearRGB(img *vips.Image, a, b []float64) error {
	if img.HasAlpha() {
		a = append(a, 1)
		b = append(b, 0)
	}
	return img.Linear(a, b, nil)
}

// isAnimated checks if an image is animated (multi-page)
func isAnimated(img *vips.Image) bool {
	return img.Height() > img.PageHeight()
}

// decodeTextArg decodes a text/label content argument. It first attempts
// URL query-unescape, then if the result has a "b64:" prefix it base64url-
// decodes the payload (RFC 4648 §5, no padding), allowing arbitrary unicode
// to be passed without URL encoding issues.
func decodeTextArg(s string) string {
	if a, e := url.QueryUnescape(s); e == nil {
		s = a
	}
	if strings.HasPrefix(s, "b64:") {
		if decoded, e := base64.RawURLEncoding.DecodeString(s[4:]); e == nil {
			s = string(decoded)
		}
	}
	return s
}

// parseFontArg decodes a Pango font string from a filter argument.
// After URL query-unescaping, hyphens are replaced with spaces so callers
// can write e.g. "sans-bold-24" or "monospace-18" without percent-encoding.
func parseFontArg(s string) string {
	if a, e := url.QueryUnescape(s); e == nil {
		s = a
	}
	return strings.ReplaceAll(s, "-", " ")
}

// parseTextWidth resolves the wrap-width argument against the canvas width.
// Supports the same conventions as overlay x/y positions:
//
//	0 or ""           → 0 (unconstrained, Pango wraps only on explicit newlines)
//	plain int         → pixel count
//	Np  (e.g. 80p)   → N% of canvas width
//	0.N (e.g. 0.8)   → fraction of canvas width
//	f or full         → full canvas width
//	f-N / full-N      → canvas width minus N pixels
func parseTextWidth(arg string, canvasWidth int) int {
	if arg == "" {
		return 0
	}
	// full / f / full-N / f-N  (reuses fullDimRegex from overlay.go)
	if m := fullDimRegex.FindStringSubmatch(arg); m != nil {
		offset := 0
		if m[2] != "" {
			offset, _ = strconv.Atoi(m[2]) // already negative e.g. "-20"
		}
		return canvasWidth + offset
	}
	// percentage: e.g. 80p
	if strings.HasSuffix(arg, "p") {
		val, _ := strconv.Atoi(strings.TrimSuffix(arg, "p"))
		return val * canvasWidth / 100
	}
	// float fraction: e.g. 0.8
	if strings.HasPrefix(strings.TrimPrefix(arg, "-"), "0.") {
		frac, _ := strconv.ParseFloat(arg, 64)
		return int(frac * float64(canvasWidth))
	}
	// plain integer pixels
	v, _ := strconv.Atoi(arg)
	return v
}

// getAngle converts an integer angle to vips.Angle enum
func getAngle(angle int) vips.Angle {
	switch angle {
	case 90:
		return vips.AngleD270
	case 180:
		return vips.AngleD180
	case 270:
		return vips.AngleD90
	default:
		return vips.AngleD0
	}
}
