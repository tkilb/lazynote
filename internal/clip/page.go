package clip

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/url"
)

// maxPageBytes bounds how much of a page Page reads.
const maxPageBytes = 2 << 20

// ErrLogin reports a page that answered 401 or 403: it needs a login.
var ErrLogin = errors.New("the page needs a login")

// Page is a fetched web page.
type Page struct {
	// URL is where the page was finally read from, after any redirects.
	URL *url.URL
	// HTML is the start of the page, at most maxPageBytes.
	HTML []byte
	// Truncated reports that the page was longer than HTML.
	Truncated bool
}

// Page fetches u under the same rules as Title (public addresses only, no
// cookies or credentials, a short timeout, few redirects) and returns the
// start of the page. The errors are ErrFetch, ErrStatus, ErrLogin (for 401
// and 403) and ErrNotHTML, none of which says where the request went.
func (f *Fetcher) Page(ctx context.Context, u *url.URL) (Page, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return Page{}, ErrBadURL
	}
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Accept", "text/html, application/xhtml+xml")
	resp, err := f.client.Do(req)
	if err != nil {
		return Page{}, ErrFetch
	}
	defer resp.Body.Close()
	switch {
	case resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden:
		return Page{}, ErrLogin
	case resp.StatusCode < 200 || resp.StatusCode > 299:
		return Page{}, ErrStatus
	case !isHTML(resp.Header.Get("Content-Type")):
		return Page{}, ErrNotHTML
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxPageBytes+1))
	if err != nil && len(body) == 0 {
		return Page{}, ErrFetch
	}
	truncated := len(body) > maxPageBytes
	if truncated {
		body = body[:maxPageBytes]
	}
	return Page{URL: resp.Request.URL, HTML: body, Truncated: truncated}, nil
}
