package loader

// ReadClipboard reads the clipboard text through read (the system
// clipboard's read function, or a fake in tests). The text must be UTF-8, no
// larger than MaxBytes and not blank.
func ReadClipboard(read func() (string, error)) (Source, error) {
	text, err := read()
	if err != nil {
		return Source{}, ErrClipboard
	}
	text, err = checkText([]byte(text))
	if err != nil {
		return Source{}, err
	}
	return Source{Kind: Clipboard, Name: "clipboard", Text: text}, nil
}
