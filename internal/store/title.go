package store

import (
	"bytes"
	"errors"
)

const (
	titleKey  = "title"
	uidKey    = "uid"
	fmOpening = "---\n"
)

// ErrFrontmatterInvalid reports a properties block that cannot be edited
// safely, so the note is left exactly as it is.
var ErrFrontmatterInvalid = errors.New("frontmatter is not valid")

// assembleNote joins a properties block and a body into note bytes, with the
// properties made standard (see Frontmatter.Standardize).
func assembleNote(fm *Frontmatter, body []byte, bom bool) ([]byte, error) {
	if err := fm.Standardize(); err != nil {
		return nil, err
	}
	block, err := fm.Marshal()
	if err != nil {
		return nil, err
	}
	var out bytes.Buffer
	if bom {
		out.Write(utf8BOM)
	}
	out.WriteString(fmOpening)
	out.Write(block)
	out.WriteString(fmDelimiter + "\n")
	out.Write(body)
	return out.Bytes(), nil
}

// SetTitle returns data with its title property set to title, leaving every
// other property, their order, comments and the body as they are (the block
// itself is re-encoded, which may normalize its formatting). A note with no
// properties block gets one holding only the title. Data whose title
// already equals title is returned unchanged, byte for byte. A properties
// block that cannot be parsed yields ErrFrontmatterInvalid and the note must
// be left alone.
func SetTitle(data []byte, title string) ([]byte, error) {
	bom := bytes.HasPrefix(data, utf8BOM)
	raw, body, ok := splitFrontmatter(data)

	fm := NewFrontmatter()
	if ok {
		var err error
		if fm, err = ParseFrontmatter(raw); err != nil {
			return nil, ErrFrontmatterInvalid
		}
		if v := fm.Get(titleKey); v != nil && v.Value == title && v.Tag == "!!str" {
			return data, nil
		}
	}
	fm.SetString(titleKey, title)
	return assembleNote(fm, body, bom)
}

// RenewIdentity returns data with a fresh uid and its title property set to
// title, leaving the other properties and the body as they are. A note with
// no properties block gets one holding both. A properties block that cannot
// be parsed yields ErrFrontmatterInvalid and the note must be left alone.
func RenewIdentity(data []byte, title string) ([]byte, error) {
	uid, err := NewUID()
	if err != nil {
		return nil, err
	}
	bom := bytes.HasPrefix(data, utf8BOM)
	raw, body, ok := splitFrontmatter(data)

	fm := NewFrontmatter()
	if ok {
		if fm, err = ParseFrontmatter(raw); err != nil {
			return nil, ErrFrontmatterInvalid
		}
	}
	fm.SetString(uidKey, uid)
	fm.SetString(titleKey, title)
	return assembleNote(fm, body, bom)
}
