// Package editor builds the command that opens a note in the user's editor.
package editor

import (
	"errors"
	"os/exec"
	"strings"
)

// Default is the editor used when EDITOR is unset or blank.
const Default = "vi"

// Command returns the command that opens the file at path in the editor named
// by EDITOR in env (a list of KEY=VALUE strings, as os.Environ returns). The
// value may carry arguments, such as "code --wait". The caller sets stdio or
// hands the command to a terminal-aware runner.
func Command(path string, env []string) (*exec.Cmd, error) {
	if path == "" {
		return nil, errors.New("no file to edit")
	}
	fields := strings.Fields(envValue(env, "EDITOR"))
	if len(fields) == 0 {
		fields = []string{Default}
	}
	args := append(append([]string{}, fields[1:]...), path)
	return exec.Command(fields[0], args...), nil
}

// envValue returns the last value of key in env, as the environment does.
func envValue(env []string, key string) string {
	var val string
	for _, kv := range env {
		if v, ok := strings.CutPrefix(kv, key+"="); ok {
			val = v
		}
	}
	return val
}
