package store

import (
	"bytes"
	"regexp"
	"strings"
	"time"
)

const (
	templateDay     = "{{date}}"
	templateWeekday = "{{weekday}}"
)

// TemplatedNote makes the contents of a note for day from template, the text
// of a template file, to be saved under name. The template's properties are
// kept, except that the note gets a uid of its own (the template's would
// otherwise exist twice), a title equal to name, the date of day, and source
// when the template has none. The body has its date filled in (see
// expandDay). A template whose properties block cannot be parsed yields
// ErrFrontmatterInvalid.
func TemplatedNote(template []byte, name, source string, day time.Time) ([]byte, error) {
	bom := bytes.HasPrefix(template, utf8BOM)
	raw, body, ok := splitFrontmatter(template)
	fm := NewFrontmatter()
	if ok {
		var err error
		if fm, err = ParseFrontmatter(raw); err != nil {
			return nil, ErrFrontmatterInvalid
		}
	}
	sample := ""
	if v := fm.Get(dateKey); v != nil {
		if d, ok := DateOnly(v.Value); ok {
			sample = d
		}
	}
	uid, err := NewUID()
	if err != nil {
		return nil, err
	}
	fm.SetString(uidKey, uid)
	fm.SetString(titleKey, noteTitle(name))
	fm.SetDate(dateKey, day.Format(dateLayout))
	if fm.blank(sourceKey) {
		fm.SetString(sourceKey, source)
	}
	return assembleNote(fm, []byte(expandDay(string(body), sample, day)), bom)
}

// expandDay fills the day into a template's body. A body that has {{date}}
// (2026-11-04) or {{weekday}} (Wednesday) gets those replaced and nothing
// else. Otherwise the template is an example of one day, the one in its date
// property, and that date is replaced wherever it appears, along with the
// weekday written right after it, such as "(Monday)". With neither, the body
// is returned as it is.
func expandDay(body, sample string, day time.Time) string {
	date, weekday := day.Format(dateLayout), day.Weekday().String()
	if strings.Contains(body, templateDay) || strings.Contains(body, templateWeekday) {
		return strings.NewReplacer(templateDay, date, templateWeekday, weekday).Replace(body)
	}
	sampleDay, err := time.Parse(dateLayout, sample)
	if err != nil {
		return body
	}
	next := regexp.MustCompile(regexp.QuoteMeta(sample) + `([\s,(]*)` + sampleDay.Weekday().String() + `\b`)
	body = next.ReplaceAllString(body, date+"${1}"+weekday)
	return strings.ReplaceAll(body, sample, date)
}
