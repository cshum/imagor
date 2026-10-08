package svg

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"regexp"
	"strings"
)

// ErrInvalidLanguageTag reports a tag that is not a language tag: caller input,
// so a request error rather than a source to fall back on.
var ErrInvalidLanguageTag = fmt.Errorf("%w: invalid language tag", ErrInvalidSVG)

// languageCond is the conditional this pass resolves. requiredFeatures and
// requiredExtensions select the same way, and are left to the renderer.
const languageCond = "systemLanguage"

// tagRe is a language tag as BCP47 writes it in an Accept-Language: a primary
// subtag and any number of subtags. The renderer errors on a malformed one.
var tagRe = regexp.MustCompile(`^[A-Za-z]{1,8}(-[A-Za-z0-9]{1,8})*$`)

// SelectLanguage returns the document with its language conditionals resolved for
// the accepted tags. A branch matching none of them is dropped with its subtree,
// and the condition is removed from one that matches: left on the node, the
// renderer evaluates it again and passes over the branch chosen here.
//
// This runs on the rendering path, so the sanitizer's policy does not apply and
// everything else survives - foreign declarations and the document's own CSS
// included.
func SelectLanguage(r io.Reader, tags []string) ([]byte, error) {
	langs, err := parseLanguageTags(tags)
	if err != nil {
		return nil, err
	}
	// Not serving, so the sanitizer's refusals do not apply: a document carrying
	// its own CSS or a reference outside itself is still a document to render.
	doc, err := parse(r, false)
	if err != nil {
		return nil, err
	}

	w := &langWriter{
		doc:   doc,
		langs: langs,
		// The decoder hands attributes over as namespace URLs, so a prefix has to
		// be tracked to write a name back. One map is enough: a prefix is
		// declared before it is used.
		prefix: map[string]string{
			"":             "", // an attribute with no namespace has no prefix
			svgNamespace:   "",
			xmlNamespace:   "xml",
			xlinkNamespace: "xlink",
		},
	}

	var b bytes.Buffer
	b.WriteString(`<?xml version="1.0" encoding="UTF-8"?>`)
	root := doc.tokens[doc.root].(xml.StartElement) //nolint:errcheck // index set in parse
	b.WriteString("<svg")
	// The document's own declarations for these two are skipped below, so exactly
	// one of each is written.
	b.WriteString(` xmlns="` + svgNamespace + `"`)
	b.WriteString(` xmlns:xlink="` + xlinkNamespace + `"`)
	w.writeAttrs(&b, root.Attr)
	rootEnd := doc.end[doc.root]
	if rootEnd == doc.root+1 {
		b.WriteString("/>")
		return b.Bytes(), nil
	}
	b.WriteString(">")
	w.writeChildren(&b, doc.root+1, rootEnd)
	b.WriteString("</svg>")
	return b.Bytes(), nil
}

type langWriter struct {
	doc    *document
	langs  []string
	prefix map[string]string // namespace URL -> prefix
}

func (w *langWriter) writeChildren(b *bytes.Buffer, from, to int) {
	for i := from; i < to; i++ {
		switch t := w.doc.tokens[i].(type) {
		case xml.CharData:
			// Escaping holds in every context: the parser restores it, so a
			// stylesheet or a text node keeps what it holds.
			b.WriteString(escapeText(string(t)))
		case xml.StartElement:
			elEnd := w.doc.end[i]
			if elEnd < 0 || elEnd > to {
				return
			}
			if !w.matches(t.Attr) {
				i = elEnd // drop the element and its subtree
				continue
			}
			name := t.Name.Local
			b.WriteString("<" + name)
			w.writeAttrs(b, t.Attr)
			if elEnd == i+1 {
				b.WriteString("/>")
			} else {
				b.WriteString(">")
				w.writeChildren(b, i+1, elEnd)
				b.WriteString("</" + name + ">")
			}
			i = elEnd
		default:
			// comments, processing instructions and directives: dropped
		}
	}
}

// writeAttrs writes every attribute with its namespace prefix, minus the language
// condition. Declarations are written as they are met, or the document will not
// parse.
func (w *langWriter) writeAttrs(b *bytes.Buffer, attrs []xml.Attr) {
	for _, attr := range attrs {
		switch {
		case attr.Name.Space == "xmlns": // xmlns:foo="..."
			w.prefix[attr.Value] = attr.Name.Local
			b.WriteString(" " + attr.Name.Local + `="` + escapeAttr(attr.Value) + `"`)
			continue
		case attr.Name.Space == "" && attr.Name.Local == "xmlns":
			// The default namespace, already written on the document element.
			continue
		case attr.Name.Local == languageCond && attr.Name.Space == "":
			continue
		}
		name, declared := w.prefix[attr.Name.Space]
		if !declared {
			// No declaration for this namespace, so the name cannot be written
			// back as it was. Not reachable from a parsed document.
			continue
		}
		if name != "" {
			name += ":"
		}
		b.WriteString(" " + name + attr.Name.Local + `="` + escapeAttr(attr.Value) + `"`)
	}
}

// matches reports whether an element's language condition passes.
func (w *langWriter) matches(attrs []xml.Attr) bool {
	for _, attr := range attrs {
		if attr.Name.Local != languageCond || attr.Name.Space != "" {
			continue
		}
		for _, want := range strings.Split(attr.Value, ",") {
			want = strings.ToLower(strings.TrimSpace(want))
			for _, have := range w.langs {
				if tagMatches(have, want) {
					return true
				}
			}
		}
		return false
	}
	return true
}

// tagMatches is prefix matching in both directions: "en" takes "en-US", and
// "zh-Hans-CN" takes "zh". The spec's rule is one direction; a request means both.
func tagMatches(have, want string) bool {
	return have != "" && want != "" && (have == want ||
		strings.HasPrefix(have, want+"-") || strings.HasPrefix(want, have+"-"))
}

func parseLanguageTags(tags []string) ([]string, error) {
	out := make([]string, 0, len(tags))
	for _, tag := range tags {
		tag = strings.TrimSpace(tag)
		if !tagRe.MatchString(tag) {
			return nil, fmt.Errorf("%w: %q", ErrInvalidLanguageTag, tag)
		}
		out = append(out, strings.ToLower(tag))
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("%w: no tag requested", ErrInvalidLanguageTag)
	}
	return out, nil
}
