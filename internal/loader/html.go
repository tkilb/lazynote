package loader

import (
	"bytes"
	"strings"
	"unicode"

	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

// HTMLToText reads an HTML page into its title and its readable text as
// light Markdown: headings as # lines, list items as "- " lines, table rows
// as "a | b" lines and paragraphs separated by a blank line. Scripts, styles,
// hidden elements, forms controls, navigation, footers and asides are left
// out, and when the page has a main element (or exactly one article) only
// that is read. Nothing in the page is run or fetched. It never fails: bad
// HTML gives whatever text could be read.
func HTMLToText(data []byte) (title, text string) {
	doc, err := html.Parse(bytes.NewReader(data))
	if err != nil {
		return "", ""
	}
	if t := findFirst(doc, atom.Title); t != nil {
		title = strings.Join(strings.Fields(plainText(t)), " ")
	}
	var w writer
	w.walk(contentRoot(doc), false)
	return title, strings.TrimSpace(w.b.String())
}

// contentRoot picks the part of the page to read.
func contentRoot(doc *html.Node) *html.Node {
	if m := findFirst(doc, atom.Main); m != nil {
		return m
	}
	if articles := findAll(doc, atom.Article); len(articles) == 1 {
		return articles[0]
	}
	if b := findFirst(doc, atom.Body); b != nil {
		return b
	}
	return doc
}

func findFirst(n *html.Node, a atom.Atom) *html.Node {
	if n.Type == html.ElementNode && n.DataAtom == a {
		return n
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if f := findFirst(c, a); f != nil {
			return f
		}
	}
	return nil
}

func findAll(n *html.Node, a atom.Atom) []*html.Node {
	var out []*html.Node
	if n.Type == html.ElementNode && n.DataAtom == a {
		out = append(out, n)
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		out = append(out, findAll(c, a)...)
	}
	return out
}

// plainText joins all the text under n.
func plainText(n *html.Node) string {
	if n.Type == html.TextNode {
		return n.Data
	}
	var b strings.Builder
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		b.WriteString(plainText(c))
	}
	return b.String()
}

// skipped elements are never read.
var skipped = map[atom.Atom]bool{
	atom.Script: true, atom.Style: true, atom.Noscript: true, atom.Template: true,
	atom.Svg: true, atom.Iframe: true, atom.Head: true, atom.Title: true,
	atom.Input: true, atom.Button: true, atom.Select: true, atom.Textarea: true,
	atom.Nav: true, atom.Footer: true, atom.Aside: true,
}

// paragraphs are set apart by a blank line; lines by a line break.
var (
	paragraphs = map[atom.Atom]bool{
		atom.P: true, atom.Ul: true, atom.Ol: true, atom.Table: true, atom.Blockquote: true,
		atom.Pre: true, atom.Section: true, atom.Article: true, atom.Main: true,
		atom.Figure: true, atom.Dl: true,
	}
	lines = map[atom.Atom]bool{
		atom.Div: true, atom.Tr: true, atom.Li: true, atom.Dt: true, atom.Dd: true,
		atom.Br: true, atom.Header: true, atom.Form: true, atom.Hr: true,
	}
	headings = map[atom.Atom]int{atom.H1: 1, atom.H2: 2, atom.H3: 3, atom.H4: 4, atom.H5: 5, atom.H6: 6}
)

// writer builds the text. pending is the line breaks owed before the next
// text, and space is an inline space owed.
type writer struct {
	b       strings.Builder
	pending int
	space   bool
}

func (w *writer) breaks(n int) {
	w.pending = max(w.pending, n)
	w.space = false
}

// emit writes s after any owed breaks or space.
func (w *writer) emit(s string) {
	if w.b.Len() > 0 {
		switch {
		case w.pending > 0:
			w.b.WriteString(strings.Repeat("\n", w.pending))
		case w.space:
			w.b.WriteByte(' ')
		}
	}
	w.pending, w.space = 0, false
	w.b.WriteString(s)
}

// marker writes a prefix such as "-" or "##" that a space then follows.
func (w *writer) marker(s string) {
	w.emit(s)
	w.space = true
}

// inline writes running text, collapsing white space to single spaces.
func (w *writer) inline(raw string) {
	words := strings.Fields(raw)
	if len(words) == 0 {
		if raw != "" {
			w.space = true
		}
		return
	}
	if first := []rune(raw)[0]; unicode.IsSpace(first) {
		w.space = true
	}
	w.emit(strings.Join(words, " "))
	runes := []rune(raw)
	if unicode.IsSpace(runes[len(runes)-1]) {
		w.space = true
	}
}

func (w *writer) walk(n *html.Node, inPre bool) {
	switch n.Type {
	case html.TextNode:
		switch {
		case inPre:
			if s := strings.Trim(n.Data, "\n"); s != "" {
				w.emit(s)
			}
		default:
			w.inline(n.Data)
		}
		return
	case html.ElementNode:
		if skipped[n.DataAtom] || hidden(n) {
			return
		}
	case html.DocumentNode:
	default:
		return
	}
	w.enter(n)
	pre := inPre || n.DataAtom == atom.Pre
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		w.walk(c, pre)
	}
	w.leave(n)
}

func (w *writer) enter(n *html.Node) {
	a := n.DataAtom
	switch {
	case headings[a] > 0:
		w.breaks(2)
		w.marker(strings.Repeat("#", headings[a]))
	case a == atom.Li:
		w.breaks(1)
		w.marker("-")
	case a == atom.Td || a == atom.Th:
		if prev := previousElement(n); prev != nil {
			w.b.WriteString(" |")
			w.pending, w.space = 0, true
		}
	case paragraphs[a]:
		w.breaks(2)
	case lines[a]:
		w.breaks(1)
	}
}

func (w *writer) leave(n *html.Node) {
	a := n.DataAtom
	switch {
	case headings[a] > 0 || paragraphs[a]:
		w.breaks(2)
	case lines[a]:
		w.breaks(1)
	}
}

func previousElement(n *html.Node) *html.Node {
	for p := n.PrevSibling; p != nil; p = p.PrevSibling {
		if p.Type == html.ElementNode {
			return p
		}
	}
	return nil
}

// hidden reports an element marked hidden, which a reader never sees.
func hidden(n *html.Node) bool {
	for _, a := range n.Attr {
		if a.Key == "hidden" || (a.Key == "aria-hidden" && strings.EqualFold(a.Val, "true")) {
			return true
		}
	}
	return false
}
