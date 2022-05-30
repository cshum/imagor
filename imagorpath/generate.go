package imagorpath

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"
)

// GeneratePath generate Imagor path by Params struct
func GeneratePath(p Params) string {
	var parts []string
	if p.Meta {
		parts = append(parts, "meta")
	}
	if p.Trim || (p.TrimBy == TrimByTopLeft || p.TrimBy == TrimByBottomRight) {
		trims := []string{"trim"}
		if p.TrimBy == TrimByBottomRight {
			trims = append(trims, "bottom-right")
		}
		if p.TrimTolerance > 0 {
			trims = append(trims, strconv.Itoa(p.TrimTolerance))
		}
		parts = append(parts, strings.Join(trims, ":"))
	}
	if p.CropTop > 0 || p.CropRight > 0 || p.CropLeft > 0 || p.CropBottom > 0 {
		parts = append(parts, fmt.Sprintf(
			"%sx%s:%sx%s",
			strconv.FormatFloat(p.CropLeft, 'f', -1, 64),
			strconv.FormatFloat(p.CropTop, 'f', -1, 64),
			strconv.FormatFloat(p.CropRight, 'f', -1, 64),
			strconv.FormatFloat(p.CropBottom, 'f', -1, 64)))
	}
	if p.FitIn {
		parts = append(parts, "fit-in")
	}
	if p.Stretch {
		parts = append(parts, "stretch")
	}
	if p.HFlip || p.Width != 0 || p.VFlip || p.Height != 0 ||
		p.PaddingLeft > 0 || p.PaddingTop > 0 {
		if p.Width < 0 {
			p.HFlip = !p.HFlip
			p.Width = -p.Width
		}
		if p.Height < 0 {
			p.VFlip = !p.VFlip
			p.Height = -p.Height
		}
		var hFlipStr, vFlipStr string
		if p.HFlip {
			hFlipStr = "-"
		}
		if p.VFlip {
			vFlipStr = "-"
		}
		parts = append(parts, fmt.Sprintf(
			"%s%dx%s%d", hFlipStr, p.Width, vFlipStr, p.Height))
	}
	if p.PaddingLeft > 0 || p.PaddingTop > 0 || p.PaddingRight > 0 || p.PaddingBottom > 0 {
		if p.PaddingLeft == p.PaddingRight && p.PaddingTop == p.PaddingBottom {
			parts = append(parts, fmt.Sprintf("%dx%d", p.PaddingLeft, p.PaddingTop))
		} else {
			parts = append(parts, fmt.Sprintf(
				"%dx%d:%dx%d",
				p.PaddingLeft, p.PaddingTop,
				p.PaddingRight, p.PaddingBottom))
		}
	}
	if p.HAlign == HAlignLeft || p.HAlign == HAlignRight {
		parts = append(parts, p.HAlign)
	}
	if p.VAlign == VAlignTop || p.VAlign == VAlignBottom {
		parts = append(parts, p.VAlign)
	}
	if p.Smart {
		parts = append(parts, "smart")
	}
	if len(p.Filters) > 0 {
		var filters []string
		for _, f := range p.Filters {
			filters = append(filters, fmt.Sprintf("%s(%s)", f.Name, f.Args))
		}
		parts = append(parts, "filters:"+strings.Join(filters, ":"))
	}
	if strings.Contains(p.Image, "?") {
		p.Image = url.QueryEscape(p.Image)
	}
	parts = append(parts, p.Image)
	return strings.Join(parts, "/")
}

// GenerateUnsafe generate unsafe Imagor endpoint by Params struct
func GenerateUnsafe(p Params) string {
	return Generate(p, nil)
}

// Generate Imagor endpoint with signature by Params struct with signer
func Generate(p Params, signer Signer) string {
	imgPath := GeneratePath(p)
	if signer != nil {
		return signer.Sign(imgPath) + "/" + imgPath
	} else {
		return "unsafe/" + imgPath
	}
}
