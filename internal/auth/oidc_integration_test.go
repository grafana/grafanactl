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
	"strings"
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

	// Programmatically log in to Dex (no browser needed).
	code, returnedState := programmaticDexLogin(t, authURL, dexEmail, dexPassword)
	require.Equal(t, state, returnedState, "state mismatch")
	require.NotEmpty(t, code)

	// Exchange the code for tokens.
	token, err := oauthCfg.Exchange(ctx, code,
		oauth2.SetAuthURLParam("code_verifier", verifier),
	)
	require.NoError(t, err)
	assert.NotEmpty(t, token.AccessToken)
	assert.NotEmpty(t, token.RefreshToken, "missing refresh token (offline_access scope)")
	assert.False(t, token.Expiry.IsZero())

	// Store and verify round-trip through CachedToken.
	oidcCfg := &config.OIDCConfig{
		IssuerURL: dexIssuerURL,
		ClientID:  dexClientID,
	}
	cached := auth.NewCachedToken(token)

	assert.Equal(t, token.AccessToken, cached.AccessToken)
	assert.Equal(t, token.RefreshToken, cached.RefreshToken)
	assert.False(t, auth.TokenNeedsRefresh(cached))

	// EnsureValidToken should return the token without refreshing.
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

// programmaticDexLogin walks through Dex's login form via HTTP requests,
// returning the authorization code and state from the callback redirect.
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

	// Follow redirects from auth URL until we reach the login form (200).
	loginURL := authURL
	for range 10 {
		resp, err := client.Get(loginURL)
		require.NoError(t, err)
		resp.Body.Close()
		if resp.StatusCode == http.StatusOK {
			break
		}
		require.True(t, resp.StatusCode == http.StatusFound || resp.StatusCode == http.StatusSeeOther,
			"unexpected status %d at %s", resp.StatusCode, loginURL)
		loginURL = resolveRedirect(t, resp).String()
	}

	// POST credentials.
	resp, err := client.PostForm(loginURL, url.Values{
		"login":    {email},
		"password": {password},
	})
	require.NoError(t, err)
	resp.Body.Close()

	// Follow redirects until we hit the callback URL.
	for range 10 {
		require.True(t, resp.StatusCode == http.StatusFound || resp.StatusCode == http.StatusSeeOther,
			"expected redirect, got %d", resp.StatusCode)
		target := resolveRedirect(t, resp)
		if strings.HasPrefix(target.String(), callbackBase) {
			return target.Query().Get("code"), target.Query().Get("state")
		}
		resp, err = client.Get(target.String())
		require.NoError(t, err)
		resp.Body.Close()
	}

	t.Fatal("did not reach callback URL")
	return "", ""
}

func resolveRedirect(t *testing.T, resp *http.Response) *url.URL {
	t.Helper()
	location := resp.Header.Get("Location")
	require.NotEmpty(t, location)
	u, err := url.Parse(location)
	require.NoError(t, err)
	if !u.IsAbs() {
		u = resp.Request.URL.ResolveReference(u)
	}
	return u
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
