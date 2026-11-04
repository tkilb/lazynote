package loader

import (
	"context"
	"errors"
	"net/url"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/tkilb/lazynote/internal/clip"
)

func mustURL(t *testing.T, raw string) *url.URL {
	t.Helper()
	u, err := url.Parse(raw)
	require.NoError(t, err)
	return u
}

func pageOf(t *testing.T, finalURL, body string) clip.Page {
	t.Helper()
	return clip.Page{URL: mustURL(t, finalURL), HTML: []byte(body)}
}

const article = `<html><head><title>Pricing Rules</title></head><body><main><h1>Pricing</h1><p>Three tiers exist: basic, plus and pro. Renewals lock the price for a year, and upgrades are prorated.</p></main></body></html>`

func TestFromPage(t *testing.T) {
	req := mustURL(t, "https://wiki.example.com/pricing")
	got, err := FromPage(req, pageOf(t, "https://wiki.example.com/pricing", article))
	require.NoError(t, err)
	assert.Equal(t, Web, got.Kind)
	assert.Equal(t, "Pricing Rules", got.Name, "named after the page title")
	assert.Equal(t, "https://wiki.example.com/pricing", got.Origin)
	assert.True(t, strings.HasPrefix(got.Text, "# Pricing Rules\n\n# Pricing\n"), "the title heads the text")
	assert.Contains(t, got.Text, "Three tiers exist")
	assert.False(t, got.Truncated)
}

func TestFromPageNameFallsBackToTheHost(t *testing.T) {
	body := `<html><body><p>` + strings.Repeat("Plenty of readable words here. ", 5) + `</p></body></html>`
	got, err := FromPage(mustURL(t, "https://www.wiki.example.com/x"), pageOf(t, "https://www.wiki.example.com/x", body))
	require.NoError(t, err)
	assert.Equal(t, "wiki.example.com", got.Name, "no title: the host, without www.")
	assert.False(t, strings.HasPrefix(got.Text, "# "), "no title, no heading")
}

func TestFromPageRecordsTheRequestedAddressNotTheFinalOne(t *testing.T) {
	req := mustURL(t, "https://short.example.com/a")
	p := pageOf(t, "https://long.example.com/final?token=SECRET", article)
	got, err := FromPage(req, p)
	require.NoError(t, err)
	assert.Equal(t, "https://short.example.com/a", got.Origin)
}

func TestFromPagePassesTruncationOn(t *testing.T) {
	p := pageOf(t, "https://wiki.example.com/p", article)
	p.Truncated = true
	got, err := FromPage(mustURL(t, "https://wiki.example.com/p"), p)
	require.NoError(t, err)
	assert.True(t, got.Truncated)
}

func TestFromPageNoReadableText(t *testing.T) {
	for name, body := range map[string]string{
		"javascript only": `<html><head><title>App</title></head><body><div id="root"></div><script>render()</script></body></html>`,
		"tiny":            `<html><body><p>Hi</p></body></html>`,
		"empty":           ``,
	} {
		t.Run(name, func(t *testing.T) {
			_, err := FromPage(mustURL(t, "https://app.example.com/"), pageOf(t, "https://app.example.com/", body))
			assert.ErrorIs(t, err, ErrNoReadableText)
		})
	}
}

func TestFromPageSpotsALoginPage(t *testing.T) {
	loginForm := `<html><head><title>Sign in</title></head><body><form><input type="text" name="u"><input type="password" name="p"><button>Go</button></form></body></html>`
	long := strings.Repeat("A real article about passwords and how they are stored. ", 60)
	tests := []struct {
		name      string
		requested string
		final     string
		body      string
		wantLogin bool
	}{
		{"redirected to a login path", "https://wiki.example.com/p", "https://wiki.example.com/login?next=/p", loginForm, true},
		{"redirected to signin", "https://x.example.com/p", "https://x.example.com/signin", `<p>` + long + `</p>`, true},
		{"redirected to an identity provider", "https://x.example.com/p", "https://login.microsoftonline.com/tenant/oauth2/authorize", `<p>` + long + `</p>`, true},
		{"redirected to okta", "https://x.example.com/p", "https://corp.okta.com/app/x", `<p>` + long + `</p>`, true},
		{"redirected to saml", "https://x.example.com/p", "https://x.example.com/saml/sso", `<p>` + long + `</p>`, true},
		{"short page with a password box", "https://x.example.com/p", "https://x.example.com/p", loginForm, true},
		{"short page titled log in", "https://x.example.com/p", "https://x.example.com/p", `<html><head><title>Log in to Acme</title></head><body><p>Use your company account to continue to the portal.</p></body></html>`, true},
		{"long article with a password box", "https://x.example.com/p", "https://x.example.com/p", `<html><head><title>Passwords</title></head><body><p>` + long + `</p><input type="password"></body></html>`, false},
		{"redirect to an ordinary page", "https://x.example.com/old", "https://x.example.com/new", `<p>` + long + `</p>`, false},
		{"a page that is about logging in", "https://x.example.com/login-guide", "https://x.example.com/login-guide", `<html><head><title>Login guide</title></head><body><p>` + long + `</p></body></html>`, false},
		{"short page about sign in with no form", "https://x.example.com/p", "https://x.example.com/p", `<html><head><title>About sign in rules</title></head><body><p>Short but fine and readable text for the page lives here.</p></body></html>`, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := FromPage(mustURL(t, tt.requested), pageOf(t, tt.final, tt.body))
			if tt.wantLogin {
				assert.ErrorIs(t, err, ErrLoginRequired)
				return
			}
			assert.NoError(t, err)
		})
	}
}

type fakePages struct {
	page clip.Page
	err  error
	got  []string
}

func (f *fakePages) Page(_ context.Context, u *url.URL) (clip.Page, error) {
	f.got = append(f.got, u.String())
	return f.page, f.err
}

func TestReadURL(t *testing.T) {
	f := &fakePages{page: pageOf(t, "https://wiki.example.com/pricing", article)}
	got, err := ReadURL(context.Background(), f, "  https://wiki.example.com/pricing ")
	require.NoError(t, err)
	assert.Equal(t, []string{"https://wiki.example.com/pricing"}, f.got)
	assert.Equal(t, "Pricing Rules", got.Name)
	assert.Equal(t, "https://wiki.example.com/pricing", got.Origin)
}

func TestReadURLRefusesABadAddressWithoutFetching(t *testing.T) {
	for _, raw := range []string{"https://", "https://user:pw@example.com/", "ftp://example.com", "https://exa mple.com"} {
		f := &fakePages{}
		_, err := ReadURL(context.Background(), f, raw)
		assert.ErrorIs(t, err, clip.ErrBadURL, raw)
		assert.Empty(t, f.got, "nothing is fetched")
	}
}

func TestReadURLMapsFetchErrors(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want error
	}{
		{"login status", clip.ErrLogin, ErrLoginRequired},
		{"server status", clip.ErrStatus, clip.ErrStatus},
		{"unreachable", clip.ErrFetch, clip.ErrFetch},
		{"not a page", clip.ErrNotHTML, clip.ErrNotHTML},
		{"cancelled", context.Canceled, context.Canceled},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ReadURL(context.Background(), &fakePages{err: tt.err}, "https://wiki.example.com/x")
			assert.ErrorIs(t, err, tt.want)
		})
	}
}

func TestLoginErrorTellsTheUserWhatToDo(t *testing.T) {
	msg := ErrLoginRequired.Error()
	assert.Contains(t, msg, "login")
	assert.Contains(t, msg, "paste")
	assert.Contains(t, msg, "file")
	assert.False(t, errors.Is(ErrLoginRequired, clip.ErrLogin), "a separate user-facing error")
}
