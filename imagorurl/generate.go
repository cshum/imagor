package imagorurl

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"
)

const (
	HAlignLeft   = "left"
	HAlignRight  = "right"
	VAlignTop    = "top"
	VAlignBottom = "bottom"
)

func generate(p Params) string {
	var parts []string
	if p.Meta {
		parts = append(parts, "meta")
	}
	if p.Trim != "" {
		trims := []string{"trim"}
		if br := "bottom-right"; p.Trim == br {
			trims = append(trims, br)
		}
		if p.TrimTolerance > 0 {
			trims = append(trims, strconv.Itoa(p.TrimTolerance))
		}
		parts = append(parts, strings.Join(trims, ":"))
	}
	if p.CropTop > 0 || p.CropRight > 0 || p.CropLeft > 0 || p.CropBottom > 0 {
		parts = append(parts, fmt.Sprintf(
			"%dx%d:%dx%d", p.CropLeft, p.CropTop, p.CropRight, p.CropBottom))
	}
	if p.FitIn {
		parts = append(parts, "fit-in")
	}
	if p.Stretch {
		parts = append(parts, "stretch")
	}
	if p.Upscale {
		parts = append(parts, "upscale")
	}
	if p.HorizontalFlip || p.Width != 0 || p.VerticalFlip || p.Height != 0 {
		if p.Width < 0 {
			p.HorizontalFlip = !p.HorizontalFlip
			p.Width = -p.Width
		}
		if p.Height < 0 {
			p.VerticalFlip = !p.VerticalFlip
			p.Height = -p.Height
		}
		var hFlipStr, vFlipStr string
		if p.HorizontalFlip {
			hFlipStr = "-"
		}
		if p.VerticalFlip {
			vFlipStr = "-"
		}
		parts = append(parts, fmt.Sprintf(
			"%s%dx%s%d", hFlipStr, p.Width, vFlipStr, p.Height))
	}
	if p.HPadding > 0 || p.VPadding > 0 {
		parts = append(parts, fmt.Sprintf("%dx%d", p.HPadding, p.VPadding))
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

func GenerateUnsafe(p Params) string {
	return "unsafe/" + generate(p)
}

func Generate(p Params, secret string) string {
	path := generate(p)
	return Sign(path, secret) + "/" + path
}
