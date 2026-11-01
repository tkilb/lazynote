// Package clip checks a URL a user pastes and fetches the title of its page.
// A URL is untrusted input, so the fetch is bounded: only http and https, no
// credentials, a short timeout, a small read, and never an address on the
// local machine or a private network.
package clip

import (
	"errors"
	"net/netip"
	"net/url"
	"regexp"
	"strings"
	"unicode"
)

// maxURLLen bounds the length of a URL accepted.
const maxURLLen = 2048

// Errors never echo the URL, because they are shown in the interface.
var (
	ErrBadURL  = errors.New("that is not a web address (use http or https, with no password)")
	ErrNoTitle = errors.New("the page has no title")
	ErrNotHTML = errors.New("the address is not a web page")
	ErrStatus  = errors.New("the server did not return the page")
	ErrFetch   = errors.New("could not reach the address")
)

// ParseURL checks that raw is an absolute http or https address with a host,
// no user name or password, no spaces or control characters, and at most
// maxURLLen characters. The scheme comes back in lower case.
func ParseURL(raw string) (*url.URL, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" || len(raw) > maxURLLen || strings.IndexFunc(raw, bad) >= 0 {
		return nil, ErrBadURL
	}
	u, err := url.Parse(raw)
	if err != nil || u.Host == "" || u.Hostname() == "" || u.User != nil || u.Opaque != "" {
		return nil, ErrBadURL
	}
	switch strings.ToLower(u.Scheme) {
	case "http", "https":
		u.Scheme = strings.ToLower(u.Scheme)
		return u, nil
	}
	return nil, ErrBadURL
}

// bad reports a character a pasted address cannot hold.
func bad(r rune) bool { return unicode.IsSpace(r) || unicode.IsControl(r) }

// publicHost matches a dotted host name: letters, digits and hyphens in each
// label, no empty label, and an ending of at least two letters.
var publicHost = regexp.MustCompile(`^(?:[a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?\.)+[a-z]{2,63}$`)

// LooksLikeWebAddress reports whether text is, on its own, a web address worth
// offering as a suggestion: an address ParseURL accepts (one line, http or
// https, no password) whose host is a dotted name such as example.com or an
// IPv4 address. A bare name like "localhost" or "wiki", or anything with
// other text around it, is not. It checks the text only and says nothing
// about whether the page exists.
func LooksLikeWebAddress(text string) bool {
	u, err := ParseURL(text)
	if err != nil {
		return false
	}
	host := strings.ToLower(u.Hostname())
	if publicHost.MatchString(host) {
		return true
	}
	ip, err := netip.ParseAddr(host)
	return err == nil && ip.Is4()
}
