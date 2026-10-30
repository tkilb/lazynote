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
// an empty tags list) followed by body. date is YYYY-MM-DD.
func IntakeNote(name string, kind Kind, source, date, body string) ([]byte, error) {
	uid, err := NewUID()
	if err != nil {
		return nil, err
	}
	fm := NewFrontmatter()
	fm.SetString(uidKey, uid)
	fm.SetString(titleKey, noteTitle(name))
	fm.SetString(kindKey, kind.String())
	fm.SetString(sourceKey, source)
	fm.SetDate(dateKey, date)
	fm.SetStringList(tagsKey, nil)
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
