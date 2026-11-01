package clip

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseURL(t *testing.T) {
	tests := []struct {
		in   string
		want string
		ok   bool
	}{
		{"https://example.com/a?b=1#c", "https://example.com/a?b=1#c", true},
		{"  http://example.com  ", "http://example.com", true},
		{"HTTPS://Example.com/x", "https://Example.com/x", true},
		{"ftp://example.com/file", "", false},
		{"file:///etc/passwd", "", false},
		{"javascript:alert(1)", "", false},
		{"example.com", "", false},
		{"https://", "", false},
		{"https://user:pw@example.com/", "", false},
		{"https://example.com/\x00", "", false},
		{"https://example.com/ with space", "", false},
		{"", "", false},
		{"https://example.com/" + strings.Repeat("a", maxURLLen), "", false},
	}
	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			u, err := ParseURL(tt.in)
			if !tt.ok {
				require.Error(t, err)
				assert.NotContains(t, err.Error(), "pw", "an error never echoes the input")
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, u.String())
		})
	}
}

func TestExtractTitle(t *testing.T) {
	tests := []struct {
		name, html, want string
	}{
		{"plain", "<html><head><title>Hello</title></head></html>", "Hello"},
		{"upper case and attributes", "<TITLE lang=en>Hello</TITLE>", "Hello"},
		{"entities", "<title>Tom &amp; Jerry &#8211; 1&nbsp;2</title>", "Tom & Jerry – 1 2"},
		{"white space collapsed", "<title>\n  A\n\t B  </title>", "A B"},
		{"first one wins", "<title>One</title><title>Two</title>", "One"},
		{"none", "<html><body>hi</body></html>", ""},
		{"empty", "<title>   </title>", ""},
		{"unclosed", "<title>Never closed", ""},
		{"control characters removed", "<title>A\x1b[31mB</title>", "A[31mB"},
		{"invalid utf8", "<title>caf\xe9</title>", "caf"},
		{"bidi and zero width removed", "<title>a\u202eb\u200bc</title>", "abc"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ExtractTitle([]byte(tt.html))
			assert.Equal(t, tt.want, strings.TrimSpace(strings.ReplaceAll(got, "�", "")))
			assert.NotContains(t, got, "\x1b")
		})
	}
}

func TestIsPublic(t *testing.T) {
	tests := map[string]bool{
		"93.184.216.34":      true,
		"2606:2800:220:1::1": true,
		"127.0.0.1":          false,
		"::1":                false,
		"10.1.2.3":           false,
		"172.16.0.1":         false,
		"192.168.1.1":        false,
		"169.254.169.254":    false,
		"100.64.0.1":         false,
		"0.0.0.0":            false,
		"::":                 false,
		"224.0.0.1":          false,
		"fe80::1":            false,
		"fc00::1":            false,
		"::ffff:127.0.0.1":   false,
		"::ffff:10.0.0.1":    false,
		"64:ff9b::7f00:1":    false,
		"64:ff9b:1::1":       false,
		"2002:7f00:1::1":     false,
		"::127.0.0.1":        false,
		"fec0::1":            false,
		"2001::1":            false,
		"2001:db8::1":        false,
		"0.1.2.3":            false,
		"192.0.0.1":          false,
		"192.0.2.1":          false,
		"192.88.99.1":        false,
		"198.18.0.1":         false,
		"198.51.100.1":       false,
		"203.0.113.1":        false,
		"240.0.0.1":          false,
		"255.255.255.255":    false,
	}
	for in, want := range tests {
		t.Run(in, func(t *testing.T) {
			assert.Equal(t, want, isPublic(netip.MustParseAddr(in)))
		})
	}
}

// testFetcher reaches loopback, which the real one never does.
func testFetcher() *Fetcher {
	f := NewFetcher()
	f.allowPrivate = true
	return f
}

func serve(t *testing.T, h http.HandlerFunc) string {
	t.Helper()
	srv := httptest.NewServer(h)
	t.Cleanup(srv.Close)
	return srv.URL
}

func fetch(t *testing.T, f *Fetcher, raw string) (string, error) {
	t.Helper()
	u, err := ParseURL(raw)
	require.NoError(t, err)
	return f.Title(context.Background(), u)
}

func TestFetchTitle(t *testing.T) {
	url := serve(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Empty(t, r.Header.Get("Cookie"))
		assert.Empty(t, r.Header.Get("Authorization"))
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte("<html><title>Page Title</title>"))
	})
	got, err := fetch(t, testFetcher(), url)
	require.NoError(t, err)
	assert.Equal(t, "Page Title", got)
}

func TestFetchRefusesLoopbackByDefault(t *testing.T) {
	hit := false
	url := serve(t, func(w http.ResponseWriter, r *http.Request) { hit = true })
	_, err := fetch(t, NewFetcher(), url)
	require.Error(t, err)
	assert.False(t, hit, "the request never reached the server")
}

func TestFetchFailures(t *testing.T) {
	tests := []struct {
		name string
		h    http.HandlerFunc
		err  error
	}{
		{"not found", func(w http.ResponseWriter, r *http.Request) { http.NotFound(w, r) }, ErrStatus},
		{"not html", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/pdf")
			_, _ = w.Write([]byte("<title>x</title>"))
		}, ErrNotHTML},
		{"no title", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "text/html")
			_, _ = w.Write([]byte("<p>hi</p>"))
		}, ErrNoTitle},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := fetch(t, testFetcher(), serve(t, tt.h))
			assert.ErrorIs(t, err, tt.err)
		})
	}
}

func TestFetchReadsOnlyTheStart(t *testing.T) {
	url := serve(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write([]byte(strings.Repeat("x", maxBodyBytes+10) + "<title>late</title>"))
	})
	_, err := fetch(t, testFetcher(), url)
	assert.ErrorIs(t, err, ErrNoTitle, "a title past the limit is not read")
}

func TestFetchTimesOut(t *testing.T) {
	url := serve(t, func(w http.ResponseWriter, r *http.Request) {
		select {
		case <-r.Context().Done():
		case <-time.After(2 * time.Second):
		}
	})
	f := testFetcher()
	f.client.Timeout = 100 * time.Millisecond
	start := time.Now()
	_, err := fetch(t, f, url)
	require.Error(t, err)
	assert.Less(t, time.Since(start), time.Second)
}

func TestFetchFollowsFewRedirects(t *testing.T) {
	var base string
	base = serve(t, func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/final":
			w.Header().Set("Content-Type", "text/html")
			_, _ = w.Write([]byte("<title>Landed</title>"))
		case "/loop":
			http.Redirect(w, r, base+"/loop", http.StatusFound)
		default:
			http.Redirect(w, r, base+"/final", http.StatusFound)
		}
	})
	got, err := fetch(t, testFetcher(), base+"/start")
	require.NoError(t, err)
	assert.Equal(t, "Landed", got)

	_, err = fetch(t, testFetcher(), base+"/loop")
	assert.Error(t, err, "a redirect loop is cut off")
}

func TestFetchErrorsDoNotEchoTheURL(t *testing.T) {
	url := serve(t, func(w http.ResponseWriter, r *http.Request) { http.NotFound(w, r) })
	_, err := fetch(t, testFetcher(), url+"/secret-path")
	require.Error(t, err)
	assert.NotContains(t, err.Error(), "secret-path")
	assert.NotContains(t, err.Error(), "127.0.0.1")
}

func TestFetchSendsNoRefererOnARedirect(t *testing.T) {
	var sawReferer bool
	var base string
	base = serve(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/final" {
			sawReferer = r.Header.Get("Referer") != ""
			w.Header().Set("Content-Type", "text/html")
			_, _ = w.Write([]byte("<title>ok</title>"))
			return
		}
		http.Redirect(w, r, base+"/final", http.StatusFound)
	})
	_, err := fetch(t, testFetcher(), base+"/start")
	require.NoError(t, err)
	assert.False(t, sawReferer, "the original address is not passed on")
}

func TestLooksLikeWebAddress(t *testing.T) {
	tests := []struct {
		in   string
		want bool
	}{
		{"https://example.com", true},
		{"http://example.com/a/b?x=1&y=2#frag", true},
		{"  https://sub.example.co.uk/path  ", true},
		{"https://wiki.example.net:8443/pages/1234/Some+Title", true},
		{"HTTPS://Example.COM/x", true},
		{"https://93.184.216.34/x", true},
		{"https://my-site.example.org", true},
		{"https://example", false},                     // no dot: not a public host
		{"https://localhost:3000/x", false},            // local
		{"https://$$$.com", false},                     // not a host name
		{"https://exa mple.com", false},                // a space
		{"https://example.com/two\nlines", false},      // more than one line
		{"see https://example.com for details", false}, // text around it
		{"https://example.com and more", false},
		{"ftp://example.com/file", false},
		{"mailto:me@example.com", false},
		{"example.com", false}, // no scheme
		{"https://", false},
		{"https://.com", false},
		{"https://example..com", false},
		{"https://example.c", false}, // a one-letter ending
		{"https://example.123", false},
		{"https://user:pw@example.com/", false},
		{"", false},
		{"https://example.com/" + strings.Repeat("a", maxURLLen), false},
	}
	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			assert.Equal(t, tt.want, LooksLikeWebAddress(tt.in))
		})
	}
}
