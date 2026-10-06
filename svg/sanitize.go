// Package svg provides a deny-by-default sanitizer for SVG documents.
//
// It re-serializes an SVG from its XML tokens, keeping an allowlist of elements
// and attributes. Everything else is dropped: <script>, <foreignObject>,
// <style>, animation, DOCTYPE, processing instructions, comments, and every
// reference that is not a same-document fragment. Unknown names are dropped
// rather than inspected, so the allowlist is the defence, not a denylist of
// known-bad constructs. Parse failures are errors - the caller must not fall
// back to serving the original bytes.
//
// Serving an upstream SVG hands the browser executable markup that runs with
// the origin serving it, which is why this exists.
//
// Output is normalized: an XML declaration and a standard namespace set, so
// whitespace, self-closing tags, comments and namespace prefixes do not
// survive. Content is otherwise preserved, including attribute name case
// (viewBox), text and url(#id) references.
//
// A source declaring ISO-8859-1 is converted to UTF-8 rather than refused, since
// libvips renders those documents and the mapping is exact. Any other declared
// charset is refused: the caller rasterizes instead.
package svg

import (
	"bytes"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"regexp"
	"strings"
	"unicode/utf8"
)

// ErrInvalidSVG reports input that is not a usable SVG document.
var ErrInvalidSVG = errors.New("svg: invalid svg document")

// maxDepth limits element nesting so a hostile document cannot exhaust the stack
// during serialization.
const maxDepth = 256

const (
	svgNamespace   = "http://www.w3.org/2000/svg"
	xlinkNamespace = "http://www.w3.org/1999/xlink"
	xmlNamespace   = "http://www.w3.org/XML/1998/namespace"
)

// allowedElements is the set of elements that survive. An element that is not
// here is dropped with its entire subtree.
//
// Deliberately absent: script, foreignObject, style, iframe, form, metadata,
// animation (animate, set, animateTransform, animateMotion), feImage, and every
// non-SVG element.
var allowedElements = map[string]struct{}{
	// structure
	"svg": {}, "g": {}, "defs": {}, "symbol": {}, "use": {}, "switch": {},
	"title": {}, "desc": {},
	// a container, kept for its children: real documents wrap a whole graphic in
	// <a> when a link was authored, and dropping the element would drop the
	// graphic. Its href is subject to the same reference rule as any other, so
	// only a same-document link survives.
	"a": {},
	// shapes
	"path": {}, "rect": {}, "circle": {}, "ellipse": {}, "line": {},
	"polyline": {}, "polygon": {},
	// text
	"text": {}, "tspan": {}, "textPath": {},
	// embedded raster (href restricted to data: URIs)
	"image": {},
	// paint servers and geometry references
	"linearGradient": {}, "radialGradient": {}, "stop": {}, "pattern": {},
	"clipPath": {}, "mask": {}, "marker": {},
	// filters
	"filter": {}, "feGaussianBlur": {}, "feBlend": {}, "feColorMatrix": {},
	"feComponentTransfer": {}, "feFuncA": {}, "feFuncB": {}, "feFuncG": {},
	"feFuncR": {}, "feComposite": {}, "feConvolveMatrix": {},
	"feDiffuseLighting": {}, "feDisplacementMap": {}, "feFlood": {},
	"feMerge": {}, "feMergeNode": {}, "feMorphology": {}, "feOffset": {},
	"feSpecularLighting": {}, "feTile": {}, "feTurbulence": {},
	"feDistantLight": {}, "fePointLight": {}, "feSpotLight": {},
	"feDropShadow": {},
}

// allowedAttrs is the set of attributes that survive, matched on the local name;
// namespace-qualified attributes are handled by attrName. Event handlers (on*)
// and anything else not listed never survive - the allowlist is the whole
// defence for attributes.
var allowedAttrs = map[string]struct{}{
	// core and styling
	"id": {}, "class": {}, "style": {}, "transform": {}, "opacity": {}, "display": {},
	"visibility": {}, "overflow": {}, "isolation": {}, "mix-blend-mode": {},
	"vector-effect": {}, "paint-order": {},
	"clip-path": {}, "clip-rule": {}, "mask": {}, "filter": {},
	"marker-start": {}, "marker-mid": {}, "marker-end": {},
	"color": {}, "color-interpolation": {}, "color-interpolation-filters": {},
	"shape-rendering": {}, "text-rendering": {}, "image-rendering": {},
	// XML namespace: whitespace handling and language selection, never a
	// reference.
	"xml:space": {}, "xml:lang": {},
	// Other presentation attributes real documents use, none of which can carry
	// a reference: the radial gradient focal point, and the SVG 1.1
	// background/pointer attributes.
	"fx": {}, "fy": {}, "fr": {}, "enable-background": {},
	"pointer-events": {}, "cursor": {},
	// fill and stroke
	"fill": {}, "fill-opacity": {}, "fill-rule": {},
	"stroke": {}, "stroke-opacity": {}, "stroke-width": {},
	"stroke-linecap": {}, "stroke-linejoin": {}, "stroke-miterlimit": {},
	"stroke-dasharray": {}, "stroke-dashoffset": {},
	// text
	"font-family": {}, "font-size": {}, "font-size-adjust": {}, "font-stretch": {},
	"font-style": {}, "font-variant": {}, "font-weight": {},
	"letter-spacing": {}, "word-spacing": {}, "text-anchor": {},
	"text-decoration": {}, "dominant-baseline": {}, "alignment-baseline": {},
	"baseline-shift": {}, "direction": {}, "unicode-bidi": {}, "kerning": {},
	"writing-mode": {}, "glyph-orientation-horizontal": {},
	"glyph-orientation-vertical": {},
	// geometry
	"x": {}, "y": {}, "width": {}, "height": {}, "rx": {}, "ry": {},
	"cx": {}, "cy": {}, "r": {}, "x1": {}, "y1": {}, "x2": {}, "y2": {},
	"points": {}, "d": {}, "pathLength": {}, "dx": {}, "dy": {}, "rotate": {},
	"textLength": {}, "lengthAdjust": {}, "startOffset": {}, "method": {},
	"spacing": {}, "side": {},
	// viewport, gradients, patterns, markers, filters
	"viewBox": {}, "preserveAspectRatio": {},
	"gradientUnits": {}, "gradientTransform": {}, "spreadMethod": {},
	"offset": {}, "stop-color": {}, "stop-opacity": {},
	"patternUnits": {}, "patternContentUnits": {}, "patternTransform": {},
	"maskUnits": {}, "maskContentUnits": {}, "clipPathUnits": {},
	"markerUnits": {}, "markerWidth": {}, "markerHeight": {},
	"refX": {}, "refY": {}, "orient": {}, "filterUnits": {}, "primitiveUnits": {},
	"result": {}, "in": {}, "in2": {}, "stdDeviation": {}, "mode": {},
	"values": {}, "type": {}, "tableValues": {}, "slope": {}, "intercept": {},
	"amplitude": {}, "exponent": {}, "operator": {}, "k1": {}, "k2": {},
	"k3": {}, "k4": {}, "order": {}, "kernelMatrix": {}, "divisor": {},
	"bias": {}, "targetX": {}, "targetY": {}, "edgeMode": {},
	"kernelUnitLength": {}, "azimuth": {}, "elevation": {}, "pointsAtX": {},
	"pointsAtY": {}, "pointsAtZ": {}, "specularExponent": {},
	"limitingConeAngle": {}, "surfaceScale": {}, "diffuseConstant": {},
	"scale": {}, "xChannelSelector": {}, "yChannelSelector": {},
	"flood-color": {}, "flood-opacity": {}, "lighting-color": {},
	// conditional processing
	"systemLanguage": {}, "requiredFeatures": {}, "requiredExtensions": {},
}

// hrefAttrs are attributes carrying a reference. Values are restricted to
// same-document fragments; <image> is additionally allowed data: URIs, in
// imageHrefValue.
var hrefAttrs = map[string]struct{}{"href": {}}

// urlAttrs carry a paint server or geometry reference that may be written as
// url(...); their values are checked so url() only points inside the document.
var urlAttrs = map[string]struct{}{
	"fill": {}, "stroke": {}, "filter": {}, "clip-path": {}, "mask": {},
	"marker-start": {}, "marker-mid": {}, "marker-end": {},
}

// styleAttrRe matches constructs in a style attribute that can reach outside
// the document. url(#id) is fine and is handled by urlAttrValue.
var (
	styleForbiddenRe = regexp.MustCompile(`(?i)@import|expression\s*\(|javascript:|vbscript:|behavior\s*:|\\|<!--|-->|</`)
	styleURLRe       = regexp.MustCompile(`(?i)url\s*\(\s*['"]?([^'")]*)`)
	anyURLRe         = regexp.MustCompile(`(?i)url\s*\(`)
)

// Sanitize reads an SVG document and returns a sanitized copy.
//
// It returns ErrInvalidSVG (wrapped) when the input is not an SVG document or
// cannot be parsed. Callers must treat any error as "do not serve this to a
// browser".
func Sanitize(r io.Reader) ([]byte, error) {
	doc, err := parse(r)
	if err != nil {
		return nil, err
	}
	return doc.serialize()
}

type document struct {
	tokens []xml.Token
	// end maps a StartElement token index to its matching EndElement index.
	end  []int
	root int
}

func parse(r io.Reader) (*document, error) {
	dec := xml.NewDecoder(r)
	dec.CharsetReader = charsetReader
	var (
		tokens []xml.Token
		end    []int
		stack  []int
	)
	for {
		tok, err := dec.Token()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("%w: %v", ErrInvalidSVG, err)
		}
		idx := len(tokens)
		// xml.Decoder reuses its internal buffer for CharData, so the slice is
		// only valid until the next Token() call - copy it before storing.
		if cd, ok := tok.(xml.CharData); ok {
			tok = xml.CharData(append([]byte(nil), cd...))
		}
		tokens = append(tokens, tok)
		end = append(end, -1)
		switch tok.(type) {
		case xml.StartElement:
			if len(stack) >= maxDepth {
				return nil, fmt.Errorf("%w: exceeds max element depth %d", ErrInvalidSVG, maxDepth)
			}
			// A document that carries its own CSS would be served without it, and
			// the result would look different from the document that was authored.
			// Refusing lets the caller rasterize it instead, which renders what the
			// author drew.
			if t := tok.(xml.StartElement); t.Name.Local == "style" {
				return nil, fmt.Errorf("%w: document styling is not supported", ErrInvalidSVG)
			}
			stack = append(stack, idx)
		case xml.EndElement:
			if len(stack) == 0 {
				return nil, fmt.Errorf("%w: unexpected end element", ErrInvalidSVG)
			}
			open := stack[len(stack)-1]
			stack = stack[:len(stack)-1]
			end[open] = idx
		}
	}
	if len(stack) != 0 {
		return nil, fmt.Errorf("%w: unclosed element", ErrInvalidSVG)
	}

	doc := &document{tokens: tokens, end: end, root: -1}
	// Locate the root <svg> element, tolerating whitespace, comments and
	// directives (DOCTYPE) before it.
	for i, tok := range tokens {
		switch t := tok.(type) {
		case xml.StartElement:
			if t.Name.Local != "svg" {
				return nil, fmt.Errorf("%w: root element is <%s>", ErrInvalidSVG, t.Name.Local)
			}
			doc.root = i
			// Only ignorable tokens may follow the root element.
			for j := doc.end[i] + 1; j < len(tokens); j++ {
				switch tt := tokens[j].(type) {
				case xml.CharData:
					if len(bytes.TrimSpace(tt)) > 0 {
						return nil, fmt.Errorf("%w: content after root element", ErrInvalidSVG)
					}
				case xml.Comment, xml.ProcInst, xml.Directive:
				default:
					return nil, fmt.Errorf("%w: content after root element", ErrInvalidSVG)
				}
			}
			return doc, nil
		case xml.CharData:
			if len(bytes.TrimSpace(t)) > 0 {
				return nil, fmt.Errorf("%w: text before root element", ErrInvalidSVG)
			}
		}
	}
	return nil, fmt.Errorf("%w: no root element", ErrInvalidSVG)
}

func (d *document) serialize() ([]byte, error) {
	var b bytes.Buffer
	b.WriteString(`<?xml version="1.0" encoding="UTF-8"?>`)
	root := d.tokens[d.root].(xml.StartElement) //nolint:errcheck // index set in parse
	b.WriteString("<svg")
	// The sanitized document always carries its own namespace declarations.
	b.WriteString(` xmlns="` + svgNamespace + `"`)
	b.WriteString(` xmlns:xlink="` + xlinkNamespace + `"`)
	d.writeAttrs(&b, root.Attr, false)
	rootEnd := d.end[d.root]
	if rootEnd == d.root+1 {
		b.WriteString("/>")
		return b.Bytes(), nil
	}
	b.WriteString(">")
	if err := d.writeChildren(&b, d.root+1, rootEnd); err != nil {
		return nil, err
	}
	b.WriteString("</svg>")
	return b.Bytes(), nil
}

func (d *document) writeChildren(b *bytes.Buffer, from, to int) error {
	for i := from; i < to; i++ {
		switch t := d.tokens[i].(type) {
		case xml.CharData:
			b.WriteString(escapeText(string(t)))
		case xml.StartElement:
			elEnd := d.end[i]
			if elEnd < 0 || elEnd > to {
				return fmt.Errorf("%w: malformed element nesting", ErrInvalidSVG)
			}
			if _, ok := allowedElements[t.Name.Local]; !ok {
				i = elEnd // drop the element and its subtree
				continue
			}
			name := t.Name.Local
			isImage := name == "image"
			b.WriteString("<" + name)
			d.writeAttrs(b, t.Attr, isImage)
			if elEnd == i+1 {
				b.WriteString("/>")
			} else {
				b.WriteString(">")
				if err := d.writeChildren(b, i+1, elEnd); err != nil {
					return err
				}
				b.WriteString("</" + name + ">")
			}
			i = elEnd
		default:
			// comments, processing instructions and directives are dropped
		}
	}
	return nil
}

func (d *document) writeAttrs(b *bytes.Buffer, attrs []xml.Attr, isImage bool) {
	for _, attr := range attrs {
		name, ok := attrName(attr.Name)
		if !ok {
			continue
		}
		if _, ok := allowedAttrs[name]; !ok && !isHrefAttr(name) {
			continue
		}
		value, ok := sanitizeAttrValue(name, attr.Value, isImage)
		if !ok {
			continue
		}
		b.WriteString(` ` + name + `="` + escapeAttr(value) + `"`)
	}
}

// attrName maps an XML attribute name to a serializable SVG name, returning
// false for attributes that must be dropped: foreign namespaces and namespace
// declarations.
func attrName(n xml.Name) (string, bool) {
	switch n.Space {
	case "":
		if n.Local == "" {
			return "", false
		}
		return n.Local, true
	case xmlNamespace:
		return "xml:" + n.Local, true
	case xlinkNamespace:
		// Serialized as xlink:href; the xlink prefix is declared on the root.
		if _, ok := hrefAttrs[n.Local]; !ok {
			return "", false
		}
		return "xlink:" + n.Local, true
	default:
		// Namespace declarations (Space == "xmlns") and any foreign namespace.
		return "", false
	}
}

func isHrefAttr(name string) bool {
	_, ok := hrefAttrs[strings.TrimPrefix(name, "xlink:")]
	return ok && (name == "href" || strings.HasPrefix(name, "xlink:"))
}

func sanitizeAttrValue(name, value string, isImage bool) (string, bool) {
	if isHrefAttr(name) {
		if isImage {
			return imageHrefValue(value)
		}
		return hrefValue(value)
	}
	if name == "style" {
		return styleValue(value)
	}
	if _, ok := urlAttrs[name]; ok {
		return urlAttrValue(value)
	}
	return value, true
}

// hrefValue accepts only same-document fragment references. A relative or
// absolute reference would be fetched from whatever origin serves the SVG - the
// proxy itself - so it counts as external.
func hrefValue(value string) (string, bool) {
	v := strings.TrimSpace(decodeEntities(value))
	if !strings.HasPrefix(v, "#") || len(v) == 1 {
		return "", false
	}
	if strings.ContainsAny(v, " 	\r\n\"'<>\\") {
		return "", false
	}
	return v, true
}

// imageHrefValue is hrefValue plus raster data: URIs, since <image> embeds
// raster data.
func imageHrefValue(value string) (string, bool) {
	v := strings.TrimSpace(decodeEntities(value))
	if isRasterDataURI(v) && !strings.ContainsAny(v, "<>\"'\\") {
		return v, true
	}
	return hrefValue(v)
}

// urlAttrValue allows a bare CSS value, or url(#fragment). Any other url()
// target - http:, data:, a relative path - is an external reference.
func urlAttrValue(value string) (string, bool) {
	v := strings.TrimSpace(decodeEntities(value))
	if !anyURLRe.MatchString(v) {
		if _, ok := hasScheme(v); ok {
			return "", false
		}
		if strings.ContainsAny(v, "/\\") {
			return "", false
		}
		return v, true
	}
	for _, m := range styleURLRe.FindAllStringSubmatch(v, -1) {
		ref := strings.TrimSpace(m[1])
		if !strings.HasPrefix(ref, "#") || len(ref) == 1 {
			return "", false
		}
	}
	if strings.ContainsAny(v, "\\") || strings.Contains(strings.ToLower(v), "@import") {
		return "", false
	}
	return v, true
}

// charsetReader converts a declared charset to UTF-8 for the XML parser.
//
// ISO-8859-1 is supported because its mapping is exact - every byte is the code
// point of the same value - and because libvips renders such documents, so
// refusing them would mean refusing to pass through a document the rasterizer
// handles. Anything else is refused, and the caller falls back to rasterizing.
func charsetReader(charset string, input io.Reader) (io.Reader, error) {
	switch strings.ToLower(strings.TrimSpace(charset)) {
	case "utf-8", "us-ascii", "ascii":
		return input, nil
	case "iso-8859-1", "latin1", "latin-1":
		data, err := io.ReadAll(input)
		if err != nil {
			return nil, err
		}
		var b strings.Builder
		b.Grow(len(data))
		for _, c := range data {
			b.WriteRune(rune(c))
		}
		return strings.NewReader(b.String()), nil
	default:
		return nil, fmt.Errorf("%w: unsupported charset %q", ErrInvalidSVG, charset)
	}
}

// imageRasterTypes are the types an <image> may embed as a data: URI. An
// embedded SVG is a document this allowlist cannot inspect, so it is not
// allowed through.
var imageRasterTypes = []string{"png", "jpeg", "jpg", "gif", "webp", "avif", "bmp", "tiff"}

// isRasterDataURI reports whether value is a data: URI of a raster image type.
func isRasterDataURI(value string) bool {
	lower := strings.ToLower(value)
	rest, ok := strings.CutPrefix(lower, "data:image/")
	if !ok {
		return false
	}
	for _, t := range imageRasterTypes {
		if after, ok := strings.CutPrefix(rest, t); ok && after != "" {
			switch after[0] {
			case ';', ',':
				return true
			}
		}
	}
	return false
}

// styleValue keeps a style attribute only when it cannot reach outside the
// document: no @import, no expression(), no escape, and every url() pointing at
// a same-document fragment. CSS declarations look like URI schemes
// ("fill:red"), so only the url() targets are examined.
func styleValue(value string) (string, bool) {
	v := strings.TrimSpace(decodeEntities(value))
	if styleForbiddenRe.MatchString(v) {
		return "", false
	}
	for _, m := range styleURLRe.FindAllStringSubmatch(v, -1) {
		ref := strings.TrimSpace(m[1])
		if !strings.HasPrefix(ref, "#") || len(ref) == 1 {
			return "", false
		}
	}
	return v, true
}

// schemeRe matches a leading URI scheme such as "javascript:" or "https:".
var schemeRe = regexp.MustCompile(`^([a-zA-Z][a-zA-Z0-9+.\-]*):`)

// hasScheme reports whether s starts with a URI scheme. Whitespace and control
// characters inside the scheme are normalized away first, so "java\nscript:" is
// still detected.
func hasScheme(s string) (string, bool) {
	cleaned := strings.Map(func(r rune) rune {
		if r <= ' ' || r == 0x7f {
			return -1
		}
		return r
	}, strings.ToLower(s))
	m := schemeRe.FindStringSubmatch(cleaned)
	if m == nil {
		return "", false
	}
	return m[1], true
}

// decodeEntities resolves the XML predefined entities and numeric character
// references before a value is inspected, so that "&#x6a;avascript:" is seen.
func decodeEntities(s string) string {
	if !strings.Contains(s, "&") {
		return s
	}
	var b strings.Builder
	b.Grow(len(s))
	for i := 0; i < len(s); {
		c := s[i]
		if c != '&' {
			b.WriteByte(c)
			i++
			continue
		}
		semi := strings.IndexByte(s[i:], ';')
		if semi < 0 || semi > 12 {
			b.WriteByte(c)
			i++
			continue
		}
		entity := s[i+1 : i+semi]
		i += semi + 1
		switch entity {
		case "amp":
			b.WriteByte('&')
		case "lt":
			b.WriteByte('<')
		case "gt":
			b.WriteByte('>')
		case "quot":
			b.WriteByte('"')
		case "apos":
			b.WriteByte('\'')
		default:
			if r, ok := decodeNumericEntity(entity); ok {
				b.WriteRune(r)
			}
			// Unknown entities are dropped; encoding/xml would have failed
			// on them anyway unless they are predefined.
		}
	}
	return b.String()
}

func decodeNumericEntity(entity string) (rune, bool) {
	if len(entity) < 2 || entity[0] != '#' {
		return 0, false
	}
	digits := entity[1:]
	base := 10
	if digits[0] == 'x' || digits[0] == 'X' {
		digits, base = digits[1:], 16
	}
	var v int64
	for i := 0; i < len(digits); i++ {
		var d int64
		switch c := digits[i]; {
		case c >= '0' && c <= '9':
			d = int64(c - '0')
		case base == 16 && c >= 'a' && c <= 'f':
			d = int64(c-'a') + 10
		case base == 16 && c >= 'A' && c <= 'F':
			d = int64(c-'A') + 10
		default:
			return 0, false
		}
		v = v*int64(base) + d
		if v > utf8.MaxRune {
			return 0, false
		}
	}
	// A character reference outside the XML character range is dropped rather
	// than substituted: a document asking for a control character or a surrogate
	// is not a document to pass through with edits.
	if !isValidXMLChar(rune(v)) {
		return 0, false
	}
	return rune(v), true
}

// escapeText escapes the characters that are special in XML content. Nothing
// else needs dropping: a value reaches the serializer either from the XML
// parser or from decodeEntities, and both enforce the XML character range.
func escapeText(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		switch {
		case r == '&':
			b.WriteString("&amp;")
		case r == '<':
			b.WriteString("&lt;")
		case r == '>':
			b.WriteString("&gt;")
		default:
			b.WriteRune(r)
		}
	}
	return b.String()
}

// escapeAttr escapes the characters that are special in an attribute value.
func escapeAttr(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		switch {
		case r == '&':
			b.WriteString("&amp;")
		case r == '<':
			b.WriteString("&lt;")
		case r == '"':
			b.WriteString("&quot;")
		case r == '	':
			b.WriteString("&#9;")
		case r == '\n':
			b.WriteString("&#10;")
		case r == '\r':
			b.WriteString("&#13;")
		default:
			b.WriteRune(r)
		}
	}
	return b.String()
}

// isValidXMLChar reports whether a rune may appear in XML 1.0 content. It is
// enforced in two places: the XML parser applies it while reading, and
// decodeNumericEntity applies it to a character reference it resolves, which the
// parser never saw.
func isValidXMLChar(r rune) bool {
	return r == 0x9 || r == 0xA || r == 0xD ||
		(r >= 0x20 && r <= 0xD7FF) ||
		(r >= 0xE000 && r <= 0xFFFD) ||
		(r >= 0x10000 && r <= 0x10FFFF)
}
