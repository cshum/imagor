// Package svg provides a deny-by-default sanitizer for SVG documents.
//
// It re-serializes an SVG from its XML tokens, keeping an allowlist of elements
// and attributes and dropping everything else, so an unknown name is dropped
// rather than inspected. Parse failures are errors: the caller must not fall
// back to serving the original bytes.
//
// Serving an upstream SVG hands the browser executable markup that runs with
// the origin serving it, which is why this exists.
//
// Output is normalized - an XML declaration and a standard namespace set - so
// whitespace, comments and namespace prefixes do not survive, while attribute
// name case (viewBox), text and url(#id) references do. An ISO-8859-1 source is
// converted to UTF-8; any other declared charset is refused.
package svg

import (
	"bufio"
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

// ErrRenderingChanged reports a readable document that cannot be served as
// authored: its own CSS, an SVG font definition, or a reference outside the
// document would have to be removed. The caller rasterizes it instead, which
// renders what the author drew.
var ErrRenderingChanged = errors.New("svg: serving this document would change how it looks")

// resourceElements load something, so an href on one of them that has to be
// removed means the document would render differently. An href on <a> is
// navigation rather than content, and is dropped without refusing the document.
var resourceElements = map[string]struct{}{
	"image": {}, "feImage": {}, "use": {}, "tref": {}, "altGlyph": {},
	"linearGradient": {}, "radialGradient": {}, "pattern": {}, "filter": {},
	// A font whose data lives in another file: removing it leaves the text to a
	// fallback font.
	"font-face-uri": {},
}

// fontElements declare SVG fonts; only font-face-uri is a resource reference,
// since it points at font data outside the document.
var fontElements = map[string]struct{}{
	"font-face-uri": {},
}

// unservableReason reports why this element cannot be served as authored, or an
// empty string when it can. The checks reuse the same attribute rules the
// serializer applies, so a value that would be dropped there is recognised here.
func unservableReason(el xml.StartElement) string {
	local := el.Name.Local
	if local == "style" {
		return "the document's own CSS"
	}
	if _, ok := fontElements[local]; ok {
		return "font data outside the document"
	}
	_, isResource := resourceElements[local]
	for _, attr := range el.Attr {
		name, ok := attrName(attr.Name)
		if !ok {
			continue
		}
		if _, kept := sanitizeAttrValue(name, attr.Value, local); kept {
			continue
		}
		if isResource && isHrefAttr(name) {
			return "a reference to a resource outside the document"
		}
		if _, ok := urlAttrs[name]; ok && strings.Contains(attr.Value, "url(") {
			return "a reference to a resource outside the document"
		}
		if name == "style" {
			return "an inline style declaration"
		}
	}
	return ""
}

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
// animation (animate, set, animateTransform, animateMotion) and every non-SVG
// element.
var allowedElements = map[string]struct{}{
	// structure
	"svg": {}, "g": {}, "defs": {}, "symbol": {}, "use": {}, "switch": {},
	"title": {}, "desc": {},
	// a container, kept for its children: a document that wraps its whole graphic
	// in a link would lose the graphic with the element. An absolute http(s) href
	// survives on it, see linkHrefValue.
	"a": {},
	// shapes
	"path": {}, "rect": {}, "circle": {}, "ellipse": {}, "line": {},
	"polyline": {}, "polygon": {},
	// text
	"text": {}, "tspan": {}, "textPath": {}, "tref": {},
	// embedded raster (href restricted to data: URIs)
	"image": {},
	// SVG fonts: declarations and glyph outlines are inert and keep the
	// font-family set a document was authored with. font-face-uri is not kept.
	"font": {}, "font-face": {}, "font-face-src": {}, "font-face-name": {},
	"glyph": {}, "missing-glyph": {}, "hkern": {}, "vkern": {},
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
	"feDropShadow": {}, "feImage": {},
}

// allowedAttrs is the set of attributes that survive, matched on the local name;
// namespace-qualified attributes are handled by attrName. Event handlers (on*)
// and anything else not listed never survive.
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
	// glyph metrics and kerning, all inert values
	"units-per-em": {}, "ascent": {}, "descent": {}, "alphabetic": {},
	"hanging": {}, "ideographic": {}, "mathematical": {}, "cap-height": {},
	"x-height": {}, "accent-height": {}, "underline-position": {},
	"underline-thickness": {}, "strikethrough-position": {},
	"strikethrough-thickness": {}, "panose-1": {}, "unicode": {},
	"unicode-range": {}, "glyph-name": {}, "horiz-adv-x": {}, "vert-adv-y": {},
	"horiz-origin-x": {}, "horiz-origin-y": {}, "vert-origin-x": {},
	"vert-origin-y": {}, "arabic-form": {}, "stemv": {}, "stemh": {},
	"widths": {}, "bbox": {}, "g1": {}, "g2": {}, "u1": {}, "u2": {}, "k": {},
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
	// filter parameters a document sets on a primitive: without them the filter
	// is applied with default values and the result differs from the document
	// that was authored.
	"specularConstant": {}, "radius": {}, "preserveAlpha": {},
	"baseFrequency": {}, "numOctaves": {}, "seed": {}, "stitchTiles": {},
	"z": {}, "bottomLeftOrigin": {},
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
	doc, err := parse(skipBOM(r))
	if err != nil {
		return nil, err
	}
	return doc.serialize()
}

var utf8BOM = []byte{0xEF, 0xBB, 0xBF}

// skipBOM drops a leading byte order mark: some tools write one, and it is not
// document content.
func skipBOM(r io.Reader) io.Reader {
	br := bufio.NewReader(r)
	if b, err := br.Peek(3); err == nil && bytes.Equal(b, utf8BOM) {
		_, _ = br.Discard(3)
	}
	return br
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
			// A document that would have to be restyled, refonted or stripped of a
			// resource reference is refused: the caller rasterizes it, which renders
			// what the author drew, rather than serving something that looks different.
			if reason := unservableReason(tok.(xml.StartElement)); reason != "" {
				return nil, fmt.Errorf("%w: %s", ErrRenderingChanged, reason)
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
	d.writeAttrs(&b, "svg", root.Attr)
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
			b.WriteString("<" + name)
			d.writeAttrs(b, name, t.Attr)
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

func (d *document) writeAttrs(b *bytes.Buffer, el string, attrs []xml.Attr) {
	for _, attr := range attrs {
		name, ok := attrName(attr.Name)
		if !ok {
			continue
		}
		if _, ok := allowedAttrs[name]; !ok && !isHrefAttr(name) {
			continue
		}
		value, ok := sanitizeAttrValue(name, attr.Value, el)
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

func sanitizeAttrValue(name, value, el string) (string, bool) {
	if isHrefAttr(name) {
		switch el {
		case "image", "feImage":
			return imageHrefValue(value)
		case "a":
			// A link navigates on a click rather than fetching, so an absolute
			// http(s) target is kept: it is the author's own reference.
			return linkHrefValue(value)
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

// linkHrefValue keeps a navigation target on <a>: an absolute http(s) reference
// or a same-document fragment. A relative one is dropped, since the document is
// served from imagor's address rather than the one it was authored at, and every
// other scheme is dropped because javascript: and data: execute.
func linkHrefValue(value string) (string, bool) {
	v := strings.TrimSpace(decodeEntities(value))
	if strings.ContainsAny(v, " \t\r\n\"'<>\\") {
		return "", false
	}
	if strings.HasPrefix(v, "#") {
		return hrefValue(v)
	}
	if strings.HasPrefix(v, "//") {
		// Protocol-relative, so still an absolute reference to another host.
		return v, true
	}
	if scheme, ok := hasScheme(v); ok {
		switch scheme {
		case "http", "https":
			return v, true
		}
	}
	return "", false
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
