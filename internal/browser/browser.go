// Package browser opens a web address in the default browser by handing it
// to the system's opener. The address is passed as a single argument, never
// through a shell.
package browser

import (
	"errors"
	"os/exec"
	"runtime"

	"github.com/tkilb/para/internal/clip"
)

// Errors never echo the address.
var (
	ErrUnsupported = errors.New("opening a web address is not supported on this system")
	ErrOpen        = errors.New("could not start the browser (needs open on macOS or xdg-open on Linux)")
)

// Open opens raw, which must be an http or https address, in the default
// browser. It returns once the opener has started, not when it exits.
func Open(raw string) error {
	name, args, err := command(runtime.GOOS, raw)
	if err != nil {
		return err
	}
	cmd := exec.Command(name, args...)
	if err := cmd.Start(); err != nil {
		return ErrOpen
	}
	go func() { _ = cmd.Wait() }() // reap the opener
	return nil
}

// command returns the opener for goos and its arguments. Only web addresses
// that clip.ParseURL accepts get through, so an address can never be taken
// for an option or a local file.
func command(goos, raw string) (string, []string, error) {
	u, err := clip.ParseURL(raw)
	if err != nil {
		return "", nil, err
	}
	switch goos {
	case "darwin":
		return "open", []string{u.String()}, nil
	case "linux", "freebsd", "openbsd", "netbsd":
		return "xdg-open", []string{u.String()}, nil
	}
	return "", nil, ErrUnsupported
}
