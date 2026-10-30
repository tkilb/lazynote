package store

import (
	"crypto/rand"
	"fmt"
)

const (
	uidBytes       = 16
	uidVersionMask = 0x0f
	uidVersion4    = 0x40
	uidVariantMask = 0x3f
	uidVariantRFC  = 0x80
)

// NewUID returns a random UUIDv4, the stable identity of a note.
func NewUID() (string, error) {
	var b [uidBytes]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", ErrUnreadable
	}
	b[6] = b[6]&uidVersionMask | uidVersion4
	b[8] = b[8]&uidVariantMask | uidVariantRFC
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:]), nil
}

// BlankNote returns the contents of a new empty note: frontmatter holding
// only a fresh uid. The kind stays unset until the user chooses one.
func BlankNote() ([]byte, error) {
	uid, err := NewUID()
	if err != nil {
		return nil, err
	}
	return []byte("---\nuid: " + uid + "\n---\n"), nil
}
