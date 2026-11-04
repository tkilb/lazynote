package body

import (
	"github.com/charmbracelet/glamour/ansi"
	"github.com/charmbracelet/glamour/styles"
)

// The gruvbox-material dark palette (the "hard" variant the author's Neovim
// uses), as hex colors.
const (
	gbFg     = "#d4be98"
	gbRed    = "#ea6962"
	gbOrange = "#e78a4e"
	gbYellow = "#d8a657"
	gbGreen  = "#a9b665"
	gbAqua   = "#89b482"
	gbBlue   = "#7daea3"
	gbPurple = "#d3869b"
	gbGrey0  = "#7c6f64"
	gbGrey1  = "#928374"
	gbBgCode = "#1d2021"
)

func strPtr(s string) *string { return &s }
func boolPtr(b bool) *bool    { return &b }

// heading is a heading level in color: bold, with the given prefix.
func heading(prefix, color string) ansi.StyleBlock {
	return ansi.StyleBlock{StylePrimitive: ansi.StylePrimitive{
		Prefix: prefix, Color: strPtr(color), Bold: boolPtr(true),
	}}
}

// gruvboxStyle is glamour's dark style recolored with gruvbox-material, with
// no margins. Headings go red, orange, yellow, green, blue and purple by
// level, inline code is aqua, links blue over a grey address, and code blocks
// use chroma's gruvbox theme.
func gruvboxStyle() ansi.StyleConfig {
	s := styles.DarkStyleConfig
	none := uint(0)

	s.Document.Margin = &none
	s.Document.BlockPrefix, s.Document.BlockSuffix = "", ""
	s.Document.Color = strPtr(gbFg)

	s.Heading = ansi.StyleBlock{StylePrimitive: ansi.StylePrimitive{BlockSuffix: "\n", Bold: boolPtr(true)}}
	s.H1 = heading("# ", gbRed)
	s.H2 = heading("## ", gbOrange)
	s.H3 = heading("### ", gbYellow)
	s.H4 = heading("#### ", gbGreen)
	s.H5 = heading("##### ", gbBlue)
	s.H6 = heading("###### ", gbPurple)

	s.BlockQuote.Color = strPtr(gbGrey1)
	s.BlockQuote.Italic = boolPtr(true)
	s.HorizontalRule.Color = strPtr(gbGrey0)
	s.Strikethrough.Color = strPtr(gbGrey1)

	s.Link = ansi.StylePrimitive{Color: strPtr(gbGrey1), Underline: boolPtr(true)}
	s.LinkText = ansi.StylePrimitive{Color: strPtr(gbBlue), Bold: boolPtr(true)}
	s.Image = ansi.StylePrimitive{Color: strPtr(gbPurple), Underline: boolPtr(true)}
	s.ImageText = ansi.StylePrimitive{Color: strPtr(gbGrey1), Format: "Image: {{.text}} →"}

	s.Code = ansi.StyleBlock{StylePrimitive: ansi.StylePrimitive{
		Prefix: " ", Suffix: " ", Color: strPtr(gbAqua), BackgroundColor: strPtr(gbBgCode),
	}}
	s.CodeBlock.Color = strPtr(gbFg)
	s.CodeBlock.Theme, s.CodeBlock.Chroma = "gruvbox", nil
	return s
}
