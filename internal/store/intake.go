package store

import "bytes"

const (
	sourceKey = "source"
	dateKey   = "date"
	tagsKey   = "tags"
	// SourceImport is the source of a note brought in from a file.
	SourceImport = "import"
)

// IntakeNote returns the contents of a new intake note named name: the
// properties every intake note carries (uid, title, kind, source, date and
// a tags list, empty when there are none) followed by body. date is
// YYYY-MM-DD. An unset kind writes no kind property.
func IntakeNote(name string, kind Kind, source, date, body string, tags []string) ([]byte, error) {
	return intakeNote(name, kind, source, date, body, tags, nil)
}

// ExtractedNote is IntakeNote for a note made from lines of another note:
// it has no tags and, when related is not empty, a related property holding
// those wikilinks, such as [[Origin]].
func ExtractedNote(name string, kind Kind, source, date, body string, related []string) ([]byte, error) {
	return intakeNote(name, kind, source, date, body, nil, related)
}

func intakeNote(name string, kind Kind, source, date, body string, tags, related []string) ([]byte, error) {
	uid, err := NewUID()
	if err != nil {
		return nil, err
	}
	fm := NewFrontmatter()
	fm.SetString(uidKey, uid)
	fm.SetString(titleKey, noteTitle(name))
	if kind != KindUnset {
		fm.SetString(kindKey, kind.String())
	}
	fm.SetString(sourceKey, source)
	fm.SetDate(dateKey, date)
	fm.SetStringList(tagsKey, tags)
	if len(related) > 0 {
		fm.SetStringList("related", related)
	}
	return assembleNote(fm, []byte(body), false)
}

// ImportedNote returns data, the contents of a markdown note from outside
// the notebook, ready to be saved under name: its properties are kept, with
// source (import) and date added when missing, a fresh uid (the original
// would otherwise exist twice) and the title set to name. The kind is left
// as it is. A properties block that cannot be parsed yields
// ErrFrontmatterInvalid and the caller should save data unchanged.
func ImportedNote(data []byte, name, date string) ([]byte, error) {
	bom := bytes.HasPrefix(data, utf8BOM)
	raw, body, ok := splitFrontmatter(data)
	fm := NewFrontmatter()
	if ok {
		var err error
		if fm, err = ParseFrontmatter(raw); err != nil {
			return nil, ErrFrontmatterInvalid
		}
	}
	uid, err := NewUID()
	if err != nil {
		return nil, err
	}
	if fm.blank(sourceKey) {
		fm.SetString(sourceKey, SourceImport)
	}
	if fm.blank(dateKey) {
		fm.SetDate(dateKey, date)
	}
	fm.SetString(uidKey, uid)
	fm.SetString(titleKey, noteTitle(name))
	return assembleNote(fm, body, bom)
}
