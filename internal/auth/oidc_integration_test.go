//go:build integration

package auth_test

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"testing"
	"time"

	"github.com/grafana/grafanactl/internal/auth"
	"github.com/grafana/grafanactl/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/oauth2"
)

// These constants match testdata/dex/config.yaml.
const (
	dexIssuerURL = "http://localhost:5556/dex"
	dexClientID  = "grafanactl-test"
	dexEmail     = "admin@example.com"
	dexPassword  = "password"
	callbackBase = "http://127.0.0.1:18085/callback"
)

func TestOIDCDiscovery(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	endpoints, err := auth.DiscoverEndpoints(ctx, dexIssuerURL)
	require.NoError(t, err)

	assert.Contains(t, endpoints.AuthorizationEndpoint, "/dex/auth")
	assert.Contains(t, endpoints.TokenEndpoint, "/dex/token")
}

func TestOIDCLoginAndRefresh(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	endpoints, err := auth.DiscoverEndpoints(ctx, dexIssuerURL)
	require.NoError(t, err)

	oauthCfg := &oauth2.Config{
		ClientID: dexClientID,
		Endpoint: oauth2.Endpoint{
			AuthURL:  endpoints.AuthorizationEndpoint,
			TokenURL: endpoints.TokenEndpoint,
		},
		RedirectURL: callbackBase,
		Scopes:      []string{"openid", "profile", "email", "offline_access"},
	}

	verifier := generateTestVerifier(t)
	challenge := s256Challenge(verifier)
	state := "test-state-12345"

	authURL := oauthCfg.AuthCodeURL(
		state,
		oauth2.SetAuthURLParam("code_challenge", challenge),
		oauth2.SetAuthURLParam("code_challenge_method", "S256"),
	)

	code, returnedState := programmaticDexLogin(t, authURL, dexEmail, dexPassword)
	require.Equal(t, state, returnedState, "state mismatch")
	require.NotEmpty(t, code)

	token, err := oauthCfg.Exchange(ctx, code,
		oauth2.SetAuthURLParam("code_verifier", verifier),
	)
	require.NoError(t, err)
	assert.NotEmpty(t, token.AccessToken)
	assert.NotEmpty(t, token.RefreshToken, "missing refresh token (offline_access scope)")
	assert.False(t, token.Expiry.IsZero())

	oidcCfg := &config.OIDCConfig{
		IssuerURL: dexIssuerURL,
		ClientID:  dexClientID,
	}
	cached := auth.NewCachedToken(token)

	assert.Equal(t, token.AccessToken, cached.AccessToken)
	assert.Equal(t, token.RefreshToken, cached.RefreshToken)
	assert.False(t, auth.TokenNeedsRefresh(cached))

	accessToken, refreshed, err := auth.EnsureValidToken(ctx, oidcCfg, cached)
	require.NoError(t, err)
	assert.False(t, refreshed)
	assert.Equal(t, token.AccessToken, accessToken)

	// Force expiry, then verify refresh works.
	cached.TokenExpiry = time.Now().Add(-1 * time.Hour).Format(time.RFC3339)
	assert.True(t, auth.TokenNeedsRefresh(cached))

	accessToken, refreshed, err = auth.EnsureValidToken(ctx, oidcCfg, cached)
	require.NoError(t, err)
	assert.True(t, refreshed)
	assert.NotEmpty(t, accessToken)
}

func TestTokenNeedsRefresh(t *testing.T) {
	tests := []struct {
		name string
		cfg  *config.CachedToken
		want bool
	}{
		{"nil", nil, true},
		{"empty", &config.CachedToken{}, true},
		{"no expiry", &config.CachedToken{AccessToken: "tok"}, false},
		{"future expiry", &config.CachedToken{AccessToken: "tok", TokenExpiry: time.Now().Add(time.Hour).Format(time.RFC3339)}, false},
		{"past expiry", &config.CachedToken{AccessToken: "tok", TokenExpiry: time.Now().Add(-time.Hour).Format(time.RFC3339)}, true},
		{"malformed expiry", &config.CachedToken{AccessToken: "tok", TokenExpiry: "bad"}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, auth.TokenNeedsRefresh(tt.cfg))
		})
	}
}

// programmaticDexLogin walks through the exact Dex password-connector login flow:
//
//  1. GET  /dex/auth?...                → 302 → /dex/auth/local?...
//  2. GET  /dex/auth/local?...          → 302 → /dex/auth/local/login?state=...
//  3. GET  /dex/auth/local/login?...    → 200   (login form)
//  4. POST /dex/auth/local/login?...    → 303 → callback?code=...&state=...
//
// it's a bit nicer than just following re-directs blindly, so if Dex ever has an issue
// we should be able to pinpoint it fast.
func programmaticDexLogin(t *testing.T, authURL, email, password string) (code, state string) {
	t.Helper()

	jar, err := cookiejar.New(nil)
	require.NoError(t, err)

	client := &http.Client{
		Jar: jar,
		CheckRedirect: func(_ *http.Request, _ []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	// Step 1: GET auth URL → 302 to connector selection.
	resp, err := client.Get(authURL)
	require.NoError(t, err)
	resp.Body.Close()
	require.Equal(t, http.StatusFound, resp.StatusCode)

	// Step 2: GET connector URL → 302 to login form.
	resp, err = client.Get(locationURL(t, resp))
	require.NoError(t, err)
	resp.Body.Close()
	require.Equal(t, http.StatusFound, resp.StatusCode)

	// Step 3: GET login form → 200.
	loginFormURL := locationURL(t, resp)
	resp, err = client.Get(loginFormURL)
	require.NoError(t, err)
	resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)

	// Step 4: POST credentials → 303 to callback with code.
	resp, err = client.PostForm(loginFormURL, url.Values{
		"login":    {email},
		"password": {password},
	})
	require.NoError(t, err)
	resp.Body.Close()
	require.Equal(t, http.StatusSeeOther, resp.StatusCode)

	callback, err := url.Parse(resp.Header.Get("Location"))
	require.NoError(t, err)

	return callback.Query().Get("code"), callback.Query().Get("state")
}

func locationURL(t *testing.T, resp *http.Response) string {
	t.Helper()
	loc := resp.Header.Get("Location")
	require.NotEmpty(t, loc)
	u, err := url.Parse(loc)
	require.NoError(t, err)
	if !u.IsAbs() {
		u = resp.Request.URL.ResolveReference(u)
	}
	return u.String()
}

func generateTestVerifier(t *testing.T) string {
	t.Helper()
	buf := make([]byte, 32)
	_, err := io.ReadFull(rand.Reader, buf)
	require.NoError(t, err)
	return base64.RawURLEncoding.EncodeToString(buf)
}

func s256Challenge(verifier string) string {
	h := sha256.Sum256([]byte(verifier))
	return base64.RawURLEncoding.EncodeToString(h[:])
}
