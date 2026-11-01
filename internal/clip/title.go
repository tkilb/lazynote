package clip

import (
	"html"
	"regexp"
	"strings"
	"unicode"

	"github.com/tkilb/lazynote/internal/textsafe"
)

var titleTag = regexp.MustCompile(`(?is)<title(?:\s[^>]*)?>(.*?)</title\s*>`)

// ExtractTitle returns the text of the first <title> element in page, with
// entities decoded, control characters removed and white space collapsed, or
// "" when there is none. A page that is not valid UTF-8 loses the bad bytes.
func ExtractTitle(page []byte) string {
	m := titleTag.FindSubmatch(page)
	if m == nil {
		return ""
	}
	text := strings.ToValidUTF8(string(m[1]), "")
	text = html.UnescapeString(text)
	text = textsafe.SingleLine(text)
	text = strings.Map(dropFormat, text)
	return strings.Join(strings.Fields(text), " ")
}

// dropFormat removes invisible format characters (bidirectional overrides,
// zero-width marks) that could make a title read as something it is not.
func dropFormat(r rune) rune {
	if unicode.Is(unicode.Cf, r) {
		return -1
	}
	return r
}
