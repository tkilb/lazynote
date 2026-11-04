package loader

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestHTMLToText(t *testing.T) {
	tests := []struct {
		name, in, wantTitle, wantText string
	}{
		{"title and paragraphs", "<html><head><title>My Page</title></head><body><p>One.</p><p>Two.</p></body></html>",
			"My Page", "One.\n\nTwo."},
		{"inline tags join the text", "<p>Hello <b>big</b> <i>wide</i> world</p>", "", "Hello big wide world"},
		{"headings", "<h1>Top</h1><p>a</p><h2>Sub</h2><h3>Deeper</h3>", "", "# Top\n\na\n\n## Sub\n\n### Deeper"},
		{"lists", "<ul><li>one</li><li>two <b>x</b></li></ul>", "", "- one\n- two x"},
		{"line breaks", "<p>a<br>b<br/>c</p>", "", "a\nb\nc"},
		{"scripts and styles are dropped", "<p>keep</p><script>var x='SECRET'</script><style>p{color:red}</style><noscript>enable js</noscript><template><p>tpl</p></template>", "", "keep"},
		{"hidden elements are dropped", `<p>keep</p><div hidden>no</div><p aria-hidden="true">no</p>`, "", "keep"},
		{"entities are decoded", "<p>R&amp;D &lt;b&gt; caf&eacute; a&nbsp;b</p>", "", "R&D <b> café a b"},
		{"white space collapses", "<p>  a \n\t b   c </p>", "", "a b c"},
		{"preformatted text keeps its lines", "<pre>line 1\n  line 2</pre>", "", "line 1\n  line 2"},
		{"table rows", "<table><tr><th>Tier</th><th>Price</th></tr><tr><td>Basic</td><td>$5</td></tr></table>", "", "Tier | Price\nBasic | $5"},
		{"navigation and footer go when there is no main", "<nav>Home About</nav><p>body</p><footer>(c) corp</footer><aside>ad</aside>", "", "body"},
		{"main wins over the rest", "<div>outer</div><main><p>inside</p></main><div>after</div>", "", "inside"},
		{"article wins when there is no main", "<div>outer</div><article><p>post</p></article>", "", "post"},
		{"empty page", "<html><body></body></html>", "", ""},
		{"unclosed tags do not break it", "<p>one<p>two <b>bold", "", "one\n\ntwo bold"},
		{"title is trimmed and flattened", "<title>\n  A\n  B </title><p>x</p>", "A B", "x"},
		{"no more than one blank line", "<p>a</p><p></p><p></p><p>b</p><br><br><br><p>c</p>", "", "a\n\nb\n\nc"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			title, text := HTMLToText([]byte(tt.in))
			assert.Equal(t, tt.wantTitle, title)
			assert.Equal(t, tt.wantText, text)
		})
	}
}

func TestHTMLToTextSurvivesGarbage(t *testing.T) {
	for _, in := range []string{"", "<<<>>>", "\x00\x01<p>", "<p " + strings.Repeat("a=b ", 1000), strings.Repeat("<div>", 5000) + "x"} {
		assert.NotPanics(t, func() { HTMLToText([]byte(in)) })
	}
}
