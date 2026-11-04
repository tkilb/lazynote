package clip

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func page(t *testing.T, f *Fetcher, raw string) (Page, error) {
	t.Helper()
	u, err := ParseURL(raw)
	require.NoError(t, err)
	return f.Page(context.Background(), u)
}

func writeHTML(w http.ResponseWriter, body string) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write([]byte(body))
}

func TestPageReturnsTheHTMLAndTheFinalAddress(t *testing.T) {
	url := serve(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Empty(t, r.Header.Get("Cookie"), "no cookies are sent")
		assert.Empty(t, r.Header.Get("Authorization"), "no credentials are sent")
		writeHTML(w, "<html><title>T</title><p>hello</p></html>")
	})
	p, err := page(t, testFetcher(), url+"/doc")
	require.NoError(t, err)
	assert.Equal(t, "<html><title>T</title><p>hello</p></html>", string(p.HTML))
	assert.Equal(t, url+"/doc", p.URL.String())
	assert.False(t, p.Truncated)
}

func TestPageReportsTheAddressAfterRedirects(t *testing.T) {
	url := serve(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/start" {
			http.Redirect(w, r, "/login?next=/start", http.StatusFound)
			return
		}
		writeHTML(w, "<p>sign in</p>")
	})
	p, err := page(t, testFetcher(), url+"/start")
	require.NoError(t, err)
	assert.Equal(t, "/login", p.URL.Path, "the loader uses this to spot a login page")
}

func TestPageFailures(t *testing.T) {
	tests := []struct {
		name string
		h    http.HandlerFunc
		err  error
	}{
		{"not found", func(w http.ResponseWriter, r *http.Request) { http.NotFound(w, r) }, ErrStatus},
		{"server error", func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(500) }, ErrStatus},
		{"unauthorized", func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(401) }, ErrLogin},
		{"forbidden", func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(403) }, ErrLogin},
		{"not html", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/pdf")
			_, _ = w.Write([]byte("%PDF"))
		}, ErrNotHTML},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := page(t, testFetcher(), serve(t, tt.h))
			assert.ErrorIs(t, err, tt.err)
		})
	}
}

func TestPageCutsALargePage(t *testing.T) {
	url := serve(t, func(w http.ResponseWriter, r *http.Request) {
		writeHTML(w, strings.Repeat("x", maxPageBytes+500))
	})
	p, err := page(t, testFetcher(), url)
	require.NoError(t, err)
	assert.Len(t, p.HTML, maxPageBytes)
	assert.True(t, p.Truncated, "the caller can say the page was cut")
}

func TestPageExactlyTheLimitIsNotTruncated(t *testing.T) {
	url := serve(t, func(w http.ResponseWriter, r *http.Request) {
		writeHTML(w, strings.Repeat("x", maxPageBytes))
	})
	p, err := page(t, testFetcher(), url)
	require.NoError(t, err)
	assert.False(t, p.Truncated)
}

func TestPageRefusesLoopbackByDefault(t *testing.T) {
	hit := false
	url := serve(t, func(w http.ResponseWriter, r *http.Request) { hit = true })
	_, err := page(t, NewFetcher(), url)
	require.Error(t, err)
	assert.False(t, hit, "the request never reached the server")
}

func TestPageErrorsDoNotEchoTheURL(t *testing.T) {
	_, err := page(t, NewFetcher(), "https://secret-host.example.invalid/private/path")
	require.Error(t, err)
	assert.NotContains(t, err.Error(), "secret-host")
	assert.NotContains(t, err.Error(), "private")
}
