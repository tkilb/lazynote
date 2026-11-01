package clip

import (
	"context"
	"errors"
	"io"
	"mime"
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"syscall"
	"time"
)

const (
	fetchTimeout   = 5 * time.Second
	maxBodyBytes   = 512 << 10
	maxRedirects   = 3
	maxHeaderBytes = 64 << 10
	userAgent      = "lazynote (note clipper)"
)

// errPrivate is why a connection was refused; it never reaches the user.
var errPrivate = errors.New("address is not public")

// Fetcher fetches page titles. Create one with NewFetcher.
type Fetcher struct {
	client *http.Client
	// allowPrivate lets a connection go to a local or private address. Only
	// the tests set it.
	allowPrivate bool
}

// NewFetcher returns a Fetcher that refuses local and private addresses at
// connection time (so a name that resolves to one, or a redirect to one, is
// refused too), sends no cookies or credentials, ignores proxy settings, and
// follows at most maxRedirects redirects within fetchTimeout.
func NewFetcher() *Fetcher {
	f := &Fetcher{}
	dialer := &net.Dialer{Timeout: fetchTimeout, Control: f.guard}
	f.client = &http.Client{
		Timeout: fetchTimeout,
		Transport: &http.Transport{
			Proxy:                  nil,
			DialContext:            dialer.DialContext,
			TLSHandshakeTimeout:    fetchTimeout,
			MaxResponseHeaderBytes: maxHeaderBytes,
			DisableKeepAlives:      true,
		},
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) > maxRedirects {
				return errors.New("too many redirects")
			}
			if _, err := ParseURL(req.URL.String()); err != nil {
				return err
			}
			if via[0].URL.Scheme == "https" && req.URL.Scheme != "https" {
				return errors.New("redirect to an unencrypted address")
			}
			req.Header.Del("Referer")
			return nil
		},
	}
	return f
}

// guard runs for every connection, after the name is resolved.
func (f *Fetcher) guard(_, address string, _ syscall.RawConn) error {
	if f.allowPrivate {
		return nil
	}
	host, _, err := net.SplitHostPort(address)
	if err != nil {
		return errPrivate
	}
	ip, err := netip.ParseAddr(host)
	if err != nil || !isPublic(ip) {
		return errPrivate
	}
	return nil
}

// globalUnicast is the only IPv6 range handed out to the public internet.
var globalUnicast = netip.MustParsePrefix("2000::/3")

// notPublic lists special-purpose ranges inside what looks public: carrier-
// grade NAT, "this network", protocol assignments, documentation, relays,
// benchmarking and reserved space (IPv4), and Teredo, documentation and 6to4
// (IPv6; 6to4 and NAT64 can carry an IPv4 address inside, so they are refused
// whole).
var notPublic = prefixes(
	"100.64.0.0/10", "0.0.0.0/8", "192.0.0.0/24", "192.0.2.0/24", "192.88.99.0/24",
	"198.18.0.0/15", "198.51.100.0/24", "203.0.113.0/24", "240.0.0.0/4",
	"2001::/32", "2001:db8::/32", "2002::/16",
)

func prefixes(list ...string) []netip.Prefix {
	out := make([]netip.Prefix, len(list))
	for i, p := range list {
		out[i] = netip.MustParsePrefix(p)
	}
	return out
}

// isPublic reports whether ip is an address on the public internet: not
// loopback, private, link-local, multicast, unspecified or in a
// special-purpose range. An IPv6 address must be global unicast (2000::/3),
// which also rules out NAT64, IPv4-compatible and site-local addresses.
func isPublic(ip netip.Addr) bool {
	ip = ip.Unmap()
	if !ip.IsValid() || ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() ||
		ip.IsLinkLocalMulticast() || ip.IsInterfaceLocalMulticast() || ip.IsMulticast() ||
		ip.IsUnspecified() {
		return false
	}
	if ip.Is6() && !globalUnicast.Contains(ip) {
		return false
	}
	for _, p := range notPublic {
		if p.Contains(ip) {
			return false
		}
	}
	return true
}

// Title fetches u and returns the title of the page. It reads at most
// maxBodyBytes. The errors are ErrFetch, ErrStatus, ErrNotHTML and
// ErrNoTitle, none of which says where the request went.
func (f *Fetcher) Title(ctx context.Context, u *url.URL) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return "", ErrBadURL
	}
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Accept", "text/html, application/xhtml+xml")
	resp, err := f.client.Do(req)
	if err != nil {
		return "", ErrFetch
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return "", ErrStatus
	}
	if !isHTML(resp.Header.Get("Content-Type")) {
		return "", ErrNotHTML
	}
	page, err := io.ReadAll(io.LimitReader(resp.Body, maxBodyBytes))
	if err != nil && len(page) == 0 {
		return "", ErrFetch
	}
	title := ExtractTitle(page)
	if title == "" {
		return "", ErrNoTitle
	}
	return title, nil
}

func isHTML(contentType string) bool {
	media, _, err := mime.ParseMediaType(contentType)
	return err == nil && (media == "text/html" || media == "application/xhtml+xml")
}
