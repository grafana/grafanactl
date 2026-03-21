package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/grafana/grafana-app-sdk/logging"
	"github.com/grafana/grafanactl/internal/config"
	"golang.org/x/oauth2"
)

// DefaultScopes is the default set of OIDC scopes to request.
const DefaultScopes = "openid profile email"

const (
	callbackPath      = "/callback"
	tokenExpiryBuffer = 30 * time.Second
)

// OIDCEndpoints holds the discovered OIDC provider endpoints.
type OIDCEndpoints struct {
	AuthorizationEndpoint string `json:"authorization_endpoint"`
	TokenEndpoint         string `json:"token_endpoint"`
}

// DiscoverEndpoints fetches the OIDC provider's endpoints from .well-known/openid-configuration.
func DiscoverEndpoints(ctx context.Context, issuerURL string) (*OIDCEndpoints, error) {
	discoveryURL := strings.TrimRight(issuerURL, "/") + "/.well-known/openid-configuration"

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, discoveryURL, nil)
	if err != nil {
		return nil, fmt.Errorf("creating discovery request: %w", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetching OIDC discovery document: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("OIDC discovery returned status %d", resp.StatusCode)
	}

	var endpoints OIDCEndpoints
	if err := json.NewDecoder(resp.Body).Decode(&endpoints); err != nil {
		return nil, fmt.Errorf("decoding OIDC discovery document: %w", err)
	}

	if endpoints.AuthorizationEndpoint == "" || endpoints.TokenEndpoint == "" {
		return nil, fmt.Errorf("OIDC discovery document missing required endpoints")
	}

	return &endpoints, nil
}

// LoginOptions configures the OIDC login flow.
type LoginOptions struct {
	// CallbackPort sets a fixed port for the local callback server.
	// If 0, a random available port is used.
	CallbackPort int
}

// Login performs the OIDC Authorization Code + PKCE flow.
// It starts a local HTTP server, opens the browser for authentication,
// waits for the callback, exchanges the code for tokens, and returns the token.
func Login(ctx context.Context, oidcCfg *config.OIDCConfig, opts LoginOptions) (*oauth2.Token, error) {
	endpoints, err := DiscoverEndpoints(ctx, oidcCfg.IssuerURL)
	if err != nil {
		return nil, err
	}

	listenAddr := fmt.Sprintf("127.0.0.1:%d", opts.CallbackPort)
	listener, err := net.Listen("tcp", listenAddr)
	if err != nil {
		return nil, fmt.Errorf("starting local listener on %s: %w", listenAddr, err)
	}
	defer listener.Close()

	port := listener.Addr().(*net.TCPAddr).Port
	redirectURL := fmt.Sprintf("http://127.0.0.1:%d%s", port, callbackPath)

	scopes := strings.Fields(oidcCfg.Scopes)
	if len(scopes) == 0 {
		scopes = strings.Fields(DefaultScopes)
	}

	oauthCfg := &oauth2.Config{
		ClientID:     oidcCfg.ClientID,
		ClientSecret: oidcCfg.ClientSecret,
		Endpoint: oauth2.Endpoint{
			AuthURL:  endpoints.AuthorizationEndpoint,
			TokenURL: endpoints.TokenEndpoint,
		},
		RedirectURL: redirectURL,
		Scopes:      scopes,
	}

	// Generate PKCE verifier and state.
	verifier, err := generateVerifier()
	if err != nil {
		return nil, fmt.Errorf("generating PKCE verifier: %w", err)
	}

	state, err := generateState()
	if err != nil {
		return nil, fmt.Errorf("generating state: %w", err)
	}

	challenge := s256Challenge(verifier)

	authURL := oauthCfg.AuthCodeURL(
		state,
		oauth2.SetAuthURLParam("code_challenge", challenge),
		oauth2.SetAuthURLParam("code_challenge_method", "S256"),
	)

	// Channel to receive the authorization code.
	type callbackResult struct {
		code string
		err  error
	}
	resultCh := make(chan callbackResult, 1)

	mux := http.NewServeMux()
	mux.HandleFunc(callbackPath, func(w http.ResponseWriter, r *http.Request) {
		if errMsg := r.URL.Query().Get("error"); errMsg != "" {
			desc := r.URL.Query().Get("error_description")
			resultCh <- callbackResult{err: fmt.Errorf("OIDC error: %s: %s", errMsg, desc)}
			fmt.Fprintf(w, "<html><body><h1>Authentication failed</h1><p>%s: %s</p><p>You can close this window.</p></body></html>", errMsg, desc)
			return
		}

		code := r.URL.Query().Get("code")
		returnedState := r.URL.Query().Get("state")

		if returnedState != state {
			resultCh <- callbackResult{err: fmt.Errorf("state mismatch: possible CSRF attack")}
			fmt.Fprint(w, "<html><body><h1>Authentication failed</h1><p>State mismatch.</p></body></html>")
			return
		}

		resultCh <- callbackResult{code: code}
		fmt.Fprint(w, "<html><body><h1>Authentication successful</h1><p>You can close this window and return to the terminal.</p></body></html>")
	})

	server := &http.Server{Handler: mux}
	go func() {
		if serveErr := server.Serve(listener); serveErr != nil && serveErr != http.ErrServerClosed {
			resultCh <- callbackResult{err: fmt.Errorf("callback server: %w", serveErr)}
		}
	}()
	defer server.Shutdown(ctx) //nolint:errcheck

	// Open browser.
	logging.FromContext(ctx).Debug("Opening browser for OIDC login", slog.String("url", authURL))
	if err := openBrowser(authURL); err != nil {
		return nil, fmt.Errorf("opening browser: %w\n\nPlease open this URL manually:\n%s", err, authURL)
	}

	// Wait for callback or context cancellation.
	select {
	case result := <-resultCh:
		if result.err != nil {
			return nil, result.err
		}

		// Exchange authorization code for tokens.
		token, err := oauthCfg.Exchange(ctx, result.code,
			oauth2.SetAuthURLParam("code_verifier", verifier),
		)
		if err != nil {
			return nil, fmt.Errorf("exchanging authorization code: %w", err)
		}

		return token, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

// RefreshAccessToken refreshes the OIDC access token using the given refresh token.
func RefreshAccessToken(ctx context.Context, oidcCfg *config.OIDCConfig, refreshToken string) (*oauth2.Token, error) {
	endpoints, err := DiscoverEndpoints(ctx, oidcCfg.IssuerURL)
	if err != nil {
		return nil, err
	}

	oauthCfg := &oauth2.Config{
		ClientID:     oidcCfg.ClientID,
		ClientSecret: oidcCfg.ClientSecret,
		Endpoint: oauth2.Endpoint{
			AuthURL:  endpoints.AuthorizationEndpoint,
			TokenURL: endpoints.TokenEndpoint,
		},
	}

	tokenSource := oauthCfg.TokenSource(ctx, &oauth2.Token{
		RefreshToken: refreshToken,
	})

	return tokenSource.Token()
}

// TokenNeedsRefresh returns true if the cached token is expired or about to expire.
func TokenNeedsRefresh(cached *config.CachedToken) bool {
	if cached == nil || cached.AccessToken == "" {
		return true
	}

	if cached.TokenExpiry == "" {
		// No expiry info — assume token is still valid.
		return false
	}

	expiry, err := time.Parse(time.RFC3339, cached.TokenExpiry)
	if err != nil {
		return true
	}

	return time.Now().Add(tokenExpiryBuffer).After(expiry)
}

// NewCachedToken creates a CachedToken from an oauth2.Token.
func NewCachedToken(token *oauth2.Token) *config.CachedToken {
	cached := &config.CachedToken{
		AccessToken:  token.AccessToken,
		RefreshToken: token.RefreshToken,
	}

	if !token.Expiry.IsZero() {
		cached.TokenExpiry = token.Expiry.Format(time.RFC3339)
	}

	return cached
}

// EnsureValidToken checks if the cached token is valid and refreshes it if needed.
// Returns the current access token, whether it was refreshed, and any error.
func EnsureValidToken(ctx context.Context, oidcCfg *config.OIDCConfig, cached *config.CachedToken) (string, bool, error) {
	if cached == nil || cached.AccessToken == "" {
		return "", false, fmt.Errorf("no OIDC tokens found, run 'grafanactl auth login' to authenticate")
	}

	if !TokenNeedsRefresh(cached) {
		return cached.AccessToken, false, nil
	}

	if cached.RefreshToken == "" {
		return "", false, fmt.Errorf("OIDC access token expired and no refresh token available, run 'grafanactl auth login' to re-authenticate")
	}

	logging.FromContext(ctx).Debug("OIDC access token expired, refreshing")

	token, err := RefreshAccessToken(ctx, oidcCfg, cached.RefreshToken)
	if err != nil {
		return "", false, fmt.Errorf("refreshing OIDC token: %w\n\nRun 'grafanactl auth login' to re-authenticate", err)
	}

	refreshed := NewCachedToken(token)
	cached.AccessToken = refreshed.AccessToken
	cached.RefreshToken = refreshed.RefreshToken
	cached.TokenExpiry = refreshed.TokenExpiry

	return cached.AccessToken, true, nil
}

func generateVerifier() (string, error) {
	buf := make([]byte, 32)
	if _, err := io.ReadFull(rand.Reader, buf); err != nil {
		return "", err
	}

	return base64.RawURLEncoding.EncodeToString(buf), nil
}

func s256Challenge(verifier string) string {
	h := sha256.Sum256([]byte(verifier))
	return base64.RawURLEncoding.EncodeToString(h[:])
}

func generateState() (string, error) {
	buf := make([]byte, 16)
	if _, err := io.ReadFull(rand.Reader, buf); err != nil {
		return "", err
	}

	return base64.RawURLEncoding.EncodeToString(buf), nil
}

func openBrowser(rawURL string) error {
	var cmd *exec.Cmd

	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", rawURL)
	case "linux":
		cmd = exec.Command("xdg-open", rawURL)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", rawURL)
	default:
		return fmt.Errorf("unsupported platform %s", runtime.GOOS)
	}

	return cmd.Start()
}

