package templates

import (
	"regexp"
	"time"
)

const (
	dateLayout = "2006-01-02"
	title      = "title"
)

// placeholder matches the three forms Render fills, written exactly.
var placeholder = regexp.MustCompile(`\{\{(date|weekday|title)\}\}`)

// Vars are the values of the placeholders. A blank Title leaves {{title}} as
// written.
type Vars struct {
	Day   time.Time
	Title string
}

// Render replaces {{date}} (YYYY-MM-DD), {{weekday}} (Monday) and {{title}}
// in body. Any other {{word}}, and any spelling with spaces, stays as
// written.
func Render(body string, v Vars) string {
	return placeholder.ReplaceAllStringFunc(body, func(m string) string {
		switch m[2 : len(m)-2] {
		case "date":
			return v.Day.Format(dateLayout)
		case "weekday":
			return v.Day.Weekday().String()
		case title:
			if v.Title != "" {
				return v.Title
			}
		}
		return m
	})
}
