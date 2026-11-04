package digest

import (
	_ "embed"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// MaxPromptBytes is the largest prompt file read. The prompt is passed as a
// command-line argument, so it must stay well under the system limit.
const MaxPromptBytes = 64 << 10

var (
	// ErrEmptyPrompt reports a prompt file with nothing but white space.
	ErrEmptyPrompt = errors.New("the prompt file is empty")
	// ErrPromptTooLarge reports a prompt file over MaxPromptBytes.
	ErrPromptTooLarge = errors.New("the prompt file is too large")
)

// DefaultPrompt and DefaultMergePrompt are written to the user's config
// directory on first use. The user owns the files afterwards.
var (
	//go:embed prompts/import-prompt.md
	DefaultPrompt string
	//go:embed prompts/import-merge-prompt.md
	DefaultMergePrompt string
)

// EnsurePrompts writes the default main prompt to prompt and the default
// merge prompt to merge, each only when nothing exists at that path. It
// creates missing directories and never overwrites or follows an existing
// file or link, so the user's edits are safe.
func EnsurePrompts(prompt, merge string) error {
	if err := ensureFile(prompt, DefaultPrompt); err != nil {
		return err
	}
	return ensureFile(merge, DefaultMergePrompt)
}

func ensureFile(path, text string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create the prompt directory: %w", err)
	}
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if errors.Is(err, os.ErrExist) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("write the default prompt: %w", err)
	}
	if _, err := f.WriteString(text); err != nil {
		_ = f.Close()
		return fmt.Errorf("write the default prompt: %w", err)
	}
	return f.Close()
}

// ReadPrompt reads a prompt file whole. It returns ErrEmptyPrompt for a file
// of only white space and ErrPromptTooLarge above MaxPromptBytes.
func ReadPrompt(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", fmt.Errorf("read the prompt: %w", err)
	}
	defer f.Close()
	data, err := io.ReadAll(io.LimitReader(f, MaxPromptBytes+1))
	if err != nil {
		return "", fmt.Errorf("read the prompt: %w", err)
	}
	if len(data) > MaxPromptBytes {
		return "", ErrPromptTooLarge
	}
	if strings.TrimSpace(string(data)) == "" {
		return "", ErrEmptyPrompt
	}
	return string(data), nil
}
