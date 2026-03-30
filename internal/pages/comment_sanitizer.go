package pages

import "github.com/sym01/htmlsanitizer"

// commentAllowList is a restrictive HTML allowlist for user comments.
// Only basic text formatting and links are permitted — no structural,
// media, or layout tags that could disrupt the page.
var commentAllowList = &htmlsanitizer.AllowList{
	Tags: []*htmlsanitizer.Tag{
		{Name: "p"},
		{Name: "br"},
		{Name: "b"},
		{Name: "i"},
		{Name: "em"},
		{Name: "strong"},
		{Name: "u"},
		{Name: "s"},
		{Name: "code"},
		{Name: "pre"},
		{Name: "blockquote"},
		{Name: "ul"},
		{Name: "ol"},
		{Name: "li"},
		{Name: "a", Attr: []string{"rel", "target"}, URLAttr: []string{"href"}},
	},
	// No global class/id attributes — prevents CSS injection and ID pollution.
	GlobalAttr: []string{},
	NonHTMLTags: []*htmlsanitizer.Tag{
		{Name: "script"},
		{Name: "style"},
		{Name: "object"},
	},
}

// newCommentSanitizer returns an HTML sanitizer configured with a restrictive
// allowlist suitable for user comments.
func newCommentSanitizer() *htmlsanitizer.HTMLSanitizer {
	return &htmlsanitizer.HTMLSanitizer{
		AllowList: commentAllowList.Clone(),
	}
}
