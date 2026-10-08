package svg

import (
	"bytes"
	"encoding/xml"
	"io"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestSelectLanguagePicksTheMatchingBranch: a <switch> renders its first child
// whose conditions pass, so selecting a language means dropping the branches
// that do not apply and leaving the first that does.
func TestSelectLanguagePicksTheMatchingBranch(t *testing.T) {
	const doc = `<svg xmlns="http://www.w3.org/2000/svg" width="10" height="10">
<switch>
<text systemLanguage="zh">你好</text>
<text systemLanguage="en">Hello</text>
<text>Default</text>
</switch>
</svg>`
	en := selectLang(t, doc, "en")
	assert.Contains(t, en, "Hello")
	assert.NotContains(t, en, "你好")
	// The unconditional fallback stays: <switch> takes its first child whose
	// conditions pass, so what has to hold is the order, not the absence of it.
	assert.Less(t, strings.Index(en, "Hello"), strings.Index(en, "Default"))

	fr := selectLang(t, doc, "fr")
	assert.Contains(t, fr, "Default", "no branch matches, so the switch falls through")
	assert.NotContains(t, fr, "Hello")

	zh := selectLang(t, doc, "zh-CN")
	assert.Contains(t, zh, "你好", "zh-CN selects zh")
}

// TestSelectLanguageDropsTheConditionItMatched: left on the surviving node the
// renderer re-evaluates it against its own language and can hide the node.
func TestSelectLanguageDropsTheConditionItMatched(t *testing.T) {
	const doc = `<svg xmlns="http://www.w3.org/2000/svg"><text systemLanguage="en">Hello</text></svg>`
	assert.NotContains(t, selectLang(t, doc, "en"), "systemLanguage")
}

// TestSelectLanguageCommaListsAndPrefixes covers the tag shapes a request sends:
// a document may list several tags, and a request may be more or less specific.
func TestSelectLanguageCommaListsAndPrefixes(t *testing.T) {
	const doc = `<svg xmlns="http://www.w3.org/2000/svg">
<text systemLanguage="zh,zh-Hans">简体</text>
<text systemLanguage="en-US">American</text>
</svg>`
	assert.Contains(t, selectLang(t, doc, "zh-Hans"), "简体")
	assert.Contains(t, selectLang(t, doc, "en"), "American", "en selects en-US")
	assert.NotContains(t, selectLang(t, doc, "en-GB"), "American", "en-GB is not en-US")
	assert.NotContains(t, selectLang(t, doc, "fr"), "简体")
}

// TestSelectLanguageLeavesUnconditionalDocumentsAlone: a document with no
// conditions must come back with its content intact, since this pass runs
// before rendering, not instead of it.
func TestSelectLanguageLeavesUnconditionalDocumentsAlone(t *testing.T) {
	const doc = `<svg xmlns="http://www.w3.org/2000/svg" xml:lang="en" width="10" height="10">
<text x="1" y="2" font-size="3">Plain &amp; simple</text>
</svg>`
	out := selectLang(t, doc, "fr")
	assert.Contains(t, out, "Plain &amp; simple")
	assert.Contains(t, out, `xml:lang="en"`, "xml:lang is metadata, not a condition")
	assert.NotContains(t, out, "systemLanguage")
	// The boring case: an attribute is not lost on the way through. Dropping one
	// is invisible to a text assertion and shows up as a bad render.
	for _, attr := range []string{`width="10"`, `height="10"`, `x="1"`, `y="2"`, `font-size="3"`} {
		assert.Contains(t, out, attr)
	}
}

// TestSelectLanguageKeepsNamespacesAndCss: the pass runs on documents the
// sanitizer refuses, so it must preserve everything it is not asked to change -
// foreign namespace prefixes and the document's own stylesheet included.
func TestSelectLanguageKeepsNamespacesAndCss(t *testing.T) {
	const doc = `<svg xmlns="http://www.w3.org/2000/svg" xmlns:inkscape="http://www.inkscape.org/namespaces/inkscape">
<style>.a { fill: #ff0000 }</style>
<text inkscape:label="note" systemLanguage="fr">Bonjour</text>
</svg>`
	out := selectLang(t, doc, "fr")
	assert.Contains(t, out, "Bonjour")
	assert.Contains(t, out, ".a { fill: #ff0000 }", "the stylesheet survives")
	assert.Contains(t, out, "http://www.inkscape.org/namespaces/inkscape", "the declaration survives")
	assert.Contains(t, out, "inkscape:label=\"note\"", "and so does the prefixed attribute")
	assertWellFormed(t, out)
}

// TestSelectLanguageRejectsMalformedTags: the filter's value is caller input, and
// a tag that is not a tag should not reach the selection or the renderer.
func TestSelectLanguageRejectsMalformedTags(t *testing.T) {
	const doc = `<svg xmlns="http://www.w3.org/2000/svg"><text>Hello</text></svg>`
	for _, tag := range []string{"", "!!", "en_US", "en;", "-", "a b"} {
		_, err := SelectLanguage(strings.NewReader(doc), []string{tag})
		assert.Error(t, err, "tag %q", tag)
	}
	_, err := SelectLanguage(strings.NewReader(doc), []string{"en-US", "zh-Hans-CN"})
	assert.NoError(t, err)
}

// TestSelectLanguageAcceptsAnyOfTheTags pins the semantics against the renderer's
// own language option, checked side by side with rsvg-convert: the tags say what
// is acceptable, and where several branches are acceptable the document's order
// decides - so a list keeps every branch that matches any of them.
func TestSelectLanguageAcceptsAnyOfTheTags(t *testing.T) {
	const doc = `<svg xmlns="http://www.w3.org/2000/svg">
<switch>
<rect systemLanguage="en" fill="#0000ff"/>
<rect systemLanguage="zh" fill="#ff0000"/>
<rect fill="#00ff00"/>
</switch>
</svg>`
	both := selectLang(t, doc, "zh-CN", "en")
	assert.Contains(t, both, "#0000ff", "the en branch is acceptable too")
	assert.Contains(t, both, "#ff0000")
	assert.Less(t, strings.Index(both, "#0000ff"), strings.Index(both, "#ff0000"),
		"document order decides, so the renderer draws en here")

	zh := selectLang(t, doc, "zh-CN")
	assert.NotContains(t, zh, "#0000ff")
	assert.Contains(t, zh, "#ff0000")
}

func selectLang(t *testing.T, doc string, tags ...string) string {
	t.Helper()
	out, err := SelectLanguage(strings.NewReader(doc), tags)
	require.NoError(t, err)
	assertWellFormed(t, string(out))
	return string(out)
}

func assertWellFormed(t *testing.T, doc string) {
	t.Helper()
	dec := xml.NewDecoder(bytes.NewReader([]byte(doc)))
	for {
		_, err := dec.Token()
		if err == io.EOF {
			return
		}
		require.NoError(t, err, "output is not well-formed")
	}
}
