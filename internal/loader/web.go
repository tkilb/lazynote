package loader

import (
	"context"
	"errors"
	"net/url"
	"regexp"
	"strings"

	"github.com/tkilb/para/internal/clip"
)

const (
	// minReadableRunes is the least text a page must give to be worth importing.
	minReadableRunes = 40
	// loginMaxRunes is how short a page must be for a password box or a
	// "sign in" title to mean it is a login page and not an article.
	loginMaxRunes = 2000
)

// PageFetcher fetches a page. *clip.Fetcher is one.
type PageFetcher interface {
	Page(ctx context.Context, u *url.URL) (clip.Page, error)
}

// ReadURL fetches the page at raw and reads it into a Source. raw must be an
// http or https address (see clip.ParseURL). A page that needs a login gives
// ErrLoginRequired and one with no text gives ErrNoReadableText; the other
// errors are the fetcher's.
func ReadURL(ctx context.Context, f PageFetcher, raw string) (Source, error) {
	u, err := clip.ParseURL(raw)
	if err != nil {
		return Source{}, err
	}
	p, err := f.Page(ctx, u)
	if errors.Is(err, clip.ErrLogin) {
		return Source{}, ErrLoginRequired
	}
	if err != nil {
		return Source{}, err
	}
	return FromPage(u, p)
}

// FromPage reads a fetched page into a Source. requested is the address the
// user gave, which is what the Source records; the page may have come from
// elsewhere after redirects. The text starts with the page title as a heading.
func FromPage(requested *url.URL, p clip.Page) (Source, error) {
	title, text := HTMLToText(p.HTML)
	if looksLikeLogin(requested, p, title, text) {
		return Source{}, ErrLoginRequired
	}
	if len([]rune(text)) < minReadableRunes {
		return Source{}, ErrNoReadableText
	}
	name := title
	if name == "" {
		name = strings.TrimPrefix(strings.ToLower(requested.Hostname()), "www.")
	} else {
		text = "# " + title + "\n\n" + text
	}
	return Source{Kind: Web, Name: name, Origin: requested.String(), Text: text, Truncated: p.Truncated}, nil
}

var (
	passwordInput = regexp.MustCompile(`(?i)<input\b[^>]*\btype\s*=\s*["']?password`)
	loginTitle    = regexp.MustCompile(`(?i)^\s*(log[\s-]?in|sign[\s-]?in|single sign[\s-]?on|sso)\b`)

	loginSegments = map[string]bool{
		"login": true, "log-in": true, "signin": true, "sign-in": true, "sso": true,
		"saml": true, "oauth": true, "oauth2": true, "authorize": true, "auth": true,
		"cas": true, "adfs": true,
	}
	loginHostPrefixes = []string{"login.", "signin.", "sso.", "auth.", "accounts."}
	loginHostSuffixes = []string{".okta.com", ".oktapreview.com", ".auth0.com", ".onelogin.com", ".pingone.com", "id.atlassian.com"}
)

// looksLikeLogin reports a page that is a login wall and not the page asked
// for: a redirect to a login address, or a short page with a password box or a
// "sign in" title.
func looksLikeLogin(requested *url.URL, p clip.Page, title, text string) bool {
	if p.URL != nil && (p.URL.Host != requested.Host || p.URL.Path != requested.Path) &&
		(loginHost(p.URL.Hostname()) || loginPath(p.URL.Path)) {
		return true
	}
	if len([]rune(text)) >= loginMaxRunes {
		return false
	}
	return passwordInput.Match(p.HTML) || loginTitle.MatchString(title)
}

func loginHost(host string) bool {
	host = strings.ToLower(host)
	for _, p := range loginHostPrefixes {
		if strings.HasPrefix(host, p) {
			return true
		}
	}
	for _, s := range loginHostSuffixes {
		if strings.HasSuffix(host, s) || host == strings.TrimPrefix(s, ".") {
			return true
		}
	}
	return false
}

func loginPath(path string) bool {
	for _, seg := range strings.Split(strings.ToLower(path), "/") {
		if loginSegments[seg] {
			return true
		}
	}
	return false
}
