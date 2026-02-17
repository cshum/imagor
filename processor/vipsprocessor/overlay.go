package vipsprocessor

import (
	"strconv"
	"strings"

	"github.com/cshum/imagor/imagorpath"
	"github.com/cshum/vipsgen/vips"
)

// blendModeMap maps blend mode names to vips.BlendMode constants
var blendModeMap = map[string]vips.BlendMode{
	"normal":      vips.BlendModeOver,
	"multiply":    vips.BlendModeMultiply,
	"color-burn":  vips.BlendModeColourBurn,
	"darken":      vips.BlendModeDarken,
	"screen":      vips.BlendModeScreen,
	"color-dodge": vips.BlendModeColourDodge,
	"lighten":     vips.BlendModeLighten,
	"add":         vips.BlendModeAdd,
	"overlay":     vips.BlendModeOverlay,
	"soft-light":  vips.BlendModeSoftLight,
	"hard-light":  vips.BlendModeHardLight,
	"difference":  vips.BlendModeDifference,
	"exclusion":   vips.BlendModeExclusion,
	"mask":        vips.BlendModeDestIn,
	"mask-out":    vips.BlendModeDestOut,
}

// parseOverlayPosition parses position argument and returns position value and repeat count
func parseOverlayPosition(arg string, canvasSize, overlaySize int, hAlign, vAlign string) (pos int, repeat int) {
	repeat = 1
	if arg == "" {
		return 0, 1
	}

	// Check for alignment keyword with negative offset (e.g., left-20, l-20, right-30, r-30, top-20, t-20, bottom-20, b-20)
	if strings.HasPrefix(arg, "left-") || strings.HasPrefix(arg, "l-") {
		offset, _ := strconv.Atoi(strings.TrimPrefix(strings.TrimPrefix(arg, "left-"), "l-"))
		return -offset, 1
	} else if strings.HasPrefix(arg, "right-") || strings.HasPrefix(arg, "r-") {
		offset, _ := strconv.Atoi(strings.TrimPrefix(strings.TrimPrefix(arg, "right-"), "r-"))
		return canvasSize - overlaySize + offset, 1
	} else if strings.HasPrefix(arg, "top-") || strings.HasPrefix(arg, "t-") {
		offset, _ := strconv.Atoi(strings.TrimPrefix(strings.TrimPrefix(arg, "top-"), "t-"))
		return -offset, 1
	} else if strings.HasPrefix(arg, "bottom-") || strings.HasPrefix(arg, "b-") {
		offset, _ := strconv.Atoi(strings.TrimPrefix(strings.TrimPrefix(arg, "bottom-"), "b-"))
		return canvasSize - overlaySize + offset, 1
	}

	if arg == "center" {
		return (canvasSize - overlaySize) / 2, 1
	} else if arg == hAlign || arg == vAlign {
		if arg == imagorpath.HAlignRight || arg == imagorpath.VAlignBottom {
			return canvasSize - overlaySize, 1
		}
		return 0, 1
	} else if arg == "repeat" {
		return 0, canvasSize/overlaySize + 1
	} else if strings.HasPrefix(strings.TrimPrefix(arg, "-"), "0.") {
		pec, _ := strconv.ParseFloat(arg, 64)
		return int(pec * float64(canvasSize)), 1
	} else if strings.HasSuffix(arg, "p") {
		val, _ := strconv.Atoi(strings.TrimSuffix(arg, "p"))
		return val * canvasSize / 100, 1
	}

	pos, _ = strconv.Atoi(arg)
	return pos, 1
}

// compositeOverlay transforms and composites overlay image onto the base image
// Handles color space, alpha channel, positioning, repeat patterns, cropping, and animation frames
// Returns early without compositing if overlay is completely outside canvas bounds
func compositeOverlay(img *vips.Image, overlay *vips.Image, xArg, yArg string, alpha float64, blendMode vips.BlendMode) error {
	// Ensure overlay has proper color space and alpha
	if overlay.Bands() < 3 {
		if err := overlay.Colourspace(vips.InterpretationSrgb, nil); err != nil {
			return err
		}
	}
	if !overlay.HasAlpha() {
		if err := overlay.Addalpha(); err != nil {
			return err
		}
	}

	// Apply alpha if provided
	if alpha > 0 {
		alphaMultiplier := 1 - alpha/100
		if alphaMultiplier != 1 {
			if err := overlay.Linear([]float64{1, 1, 1, alphaMultiplier}, []float64{0, 0, 0, 0}, nil); err != nil {
				return err
			}
		}
	}

	// Parse position
	overlayWidth := overlay.Width()
	overlayHeight := overlay.PageHeight()

	x, across := parseOverlayPosition(xArg, img.Width(), overlayWidth, imagorpath.HAlignLeft, imagorpath.HAlignRight)
	y, down := parseOverlayPosition(yArg, img.PageHeight(), overlayHeight, imagorpath.VAlignTop, imagorpath.VAlignBottom)

	// Apply negative adjustment for plain numeric values only (not prefixed keywords)
	if x < 0 && xArg != "center" &&
		!strings.HasPrefix(xArg, "left-") && !strings.HasPrefix(xArg, "l-") &&
		!strings.HasPrefix(xArg, "right-") && !strings.HasPrefix(xArg, "r-") {
		x += img.Width() - overlayWidth
	}
	if y < 0 && yArg != "center" &&
		!strings.HasPrefix(yArg, "top-") && !strings.HasPrefix(yArg, "t-") &&
		!strings.HasPrefix(yArg, "bottom-") && !strings.HasPrefix(yArg, "b-") {
		y += img.PageHeight() - overlayHeight
	}

	// Handle repeat pattern
	if across*down > 1 {
		if err := overlay.EmbedMultiPage(0, 0, across*overlayWidth, down*overlayHeight,
			&vips.EmbedMultiPageOptions{Extend: vips.ExtendRepeat}); err != nil {
			return err
		}
		// Update dimensions after repeat
		overlayWidth = overlay.Width()
		overlayHeight = overlay.PageHeight()
	}

	// Check if overlay is completely outside canvas bounds
	// Skip compositing if there's no intersection with the canvas
	if x >= img.Width() || y >= img.PageHeight() ||
		x+overlayWidth <= 0 || y+overlayHeight <= 0 {
		// Overlay is completely outside canvas bounds, skip it
		return nil
	}

	// Position overlay on canvas
	// Crop overlay to only the visible portion within canvas bounds
	visibleLeft := 0
	visibleTop := 0
	visibleWidth := overlayWidth
	visibleHeight := overlayHeight
	embedX := x
	embedY := y

	// Handle overlay extending beyond right/bottom edges
	if x+overlayWidth > img.Width() {
		visibleWidth = img.Width() - x
	}
	if y+overlayHeight > img.PageHeight() {
		visibleHeight = img.PageHeight() - y
	}

	// Handle overlay starting before left/top edges (negative positions)
	if x < 0 {
		visibleLeft = -x
		visibleWidth = overlayWidth + x // reduce width
		embedX = 0
	}
	if y < 0 {
		visibleTop = -y
		visibleHeight = overlayHeight + y // reduce height
		embedY = 0
	}

	// Crop overlay to visible portion if needed
	if visibleLeft > 0 || visibleTop > 0 ||
		visibleWidth < overlayWidth || visibleHeight < overlayHeight {
		if visibleWidth > 0 && visibleHeight > 0 {
			if err := overlay.ExtractAreaMultiPage(
				visibleLeft, visibleTop, visibleWidth, visibleHeight,
			); err != nil {
				return err
			}
		} else {
			// Overlay is completely outside canvas bounds, skip it
			return nil
		}
	}

	// Embed the cropped overlay at adjusted position
	if err := overlay.EmbedMultiPage(
		embedX, embedY, img.Width(), img.PageHeight(), nil,
	); err != nil {
		return err
	}

	// Handle animation frames
	overlayN := overlay.Height() / overlay.PageHeight()
	if n := img.Height() / img.PageHeight(); n > overlayN {
		cnt := n / overlayN
		if n%overlayN > 0 {
			cnt++
		}
		if err := overlay.Replicate(1, cnt); err != nil {
			return err
		}
	}

	// Composite overlay onto image with specified blend mode
	return img.Composite2(overlay, blendMode, nil)
}

// getBlendMode returns the vips.BlendMode for a given mode string
// Defaults to BlendModeOver (normal) if mode is empty or invalid
func getBlendMode(mode string) vips.BlendMode {
	if mode == "" {
		return vips.BlendModeOver
	}
	if blendMode, ok := blendModeMap[strings.ToLower(mode)]; ok {
		return blendMode
	}
	// Default to normal if invalid mode
	return vips.BlendModeOver
}
