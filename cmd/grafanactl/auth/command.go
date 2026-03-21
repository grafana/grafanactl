package auth

import (
	"fmt"

	cmdconfig "github.com/grafana/grafanactl/cmd/grafanactl/config"
	"github.com/grafana/grafanactl/cmd/grafanactl/io"
	"github.com/grafana/grafanactl/internal/auth"
	"github.com/grafana/grafanactl/internal/config"
	"github.com/spf13/cobra"
)

// Command returns the auth command group.
func Command() *cobra.Command {
	configOpts := &cmdconfig.Options{}

	cmd := &cobra.Command{
		Use:   "auth",
		Short: "Authenticate with a Grafana instance",
		Long:  "Authenticate with a Grafana instance using OpenID Connect (OIDC).",
	}

	configOpts.BindFlags(cmd.PersistentFlags())

	cmd.AddCommand(loginCmd(configOpts))
	cmd.AddCommand(statusCmd(configOpts))

	return cmd
}

func loginCmd(configOpts *cmdconfig.Options) *cobra.Command {
	var callbackPort int

	cmd := &cobra.Command{
		Use:   "login",
		Args:  cobra.NoArgs,
		Short: "Log in using OIDC",
		Long: `Log in to Grafana using OpenID Connect (OIDC) with the Authorization Code + PKCE flow.

This opens your browser for authentication with the configured OIDC provider.
The resulting tokens are cached separately from the configuration file.

Before running this command, configure OIDC settings for the context:

  grafanactl config set contexts.<name>.grafana.oidc.issuer-url https://your-idp.example.com
  grafanactl config set contexts.<name>.grafana.oidc.client-id your-client-id`,
		Example: "\n\tgrafanactl auth login\n\tgrafanactl auth login --context production\n\tgrafanactl auth login --callback-port 8085",
		RunE: func(cmd *cobra.Command, _ []string) error {
			cfg, err := configOpts.LoadConfigTolerant(cmd.Context())
			if err != nil {
				return err
			}

			gCtx := cfg.GetCurrentContext()
			if gCtx == nil {
				return fmt.Errorf("no current context set")
			}

			if gCtx.Grafana == nil {
				gCtx.Grafana = &config.GrafanaConfig{}
			}

			if gCtx.Grafana.OIDC == nil {
				return fmt.Errorf("OIDC is not configured for context %q\n\nConfigure it with:\n  grafanactl config set contexts.%[1]s.grafana.oidc.issuer-url <issuer-url>\n  grafanactl config set contexts.%[1]s.grafana.oidc.client-id <client-id>", gCtx.Name)
			}

			if !gCtx.Grafana.OIDC.IsConfigured() {
				return fmt.Errorf("OIDC issuer-url and client-id are required for context %q", gCtx.Name)
			}

			stdout := cmd.OutOrStdout()
			io.Info(stdout, "Opening browser for OIDC login to %s...", gCtx.Grafana.OIDC.IssuerURL)

			token, err := auth.Login(cmd.Context(), gCtx.Grafana.OIDC, auth.LoginOptions{
				CallbackPort: callbackPort,
			})
			if err != nil {
				return fmt.Errorf("OIDC login failed: %w", err)
			}

			// Store tokens in the cache file, not the main config.
			cache, _ := config.LoadTokenCache(cmd.Context())
			cache.Set(cfg.CurrentContext, auth.NewCachedToken(token))

			if err := config.WriteTokenCache(cmd.Context(), cache); err != nil {
				return fmt.Errorf("saving tokens to cache: %w", err)
			}

			io.Success(stdout, "Successfully authenticated via OIDC")

			if token.Expiry.IsZero() {
				io.Info(stdout, "Token has no expiry")
			} else {
				io.Info(stdout, "Token expires at %s", token.Expiry.Format("2006-01-02 15:04:05"))
			}

			return nil
		},
	}

	cmd.Flags().IntVar(&callbackPort, "callback-port", 0, "Fixed port for the local OIDC callback server (0 = random)")

	return cmd
}

func statusCmd(configOpts *cmdconfig.Options) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "status",
		Args:    cobra.NoArgs,
		Short:   "Show OIDC authentication status",
		Example: "\n\tgrafanactl auth status",
		RunE: func(cmd *cobra.Command, _ []string) error {
			cfg, err := configOpts.LoadConfigTolerant(cmd.Context())
			if err != nil {
				return err
			}

			gCtx := cfg.GetCurrentContext()
			if gCtx == nil {
				return fmt.Errorf("no current context set")
			}

			stdout := cmd.OutOrStdout()

			if gCtx.Grafana == nil || gCtx.Grafana.OIDC == nil || !gCtx.Grafana.OIDC.IsConfigured() {
				io.Warning(stdout, "OIDC is not configured for context %q", gCtx.Name)
				return nil
			}

			io.Info(stdout, "OIDC provider: %s", gCtx.Grafana.OIDC.IssuerURL)
			io.Info(stdout, "Client ID: %s", gCtx.Grafana.OIDC.ClientID)

			cache, _ := config.LoadTokenCache(cmd.Context())
			cached := cache.Get(cfg.CurrentContext)

			if cached == nil || cached.AccessToken == "" {
				io.Warning(stdout, "Not authenticated. Run 'grafanactl auth login' to log in.")
				return nil
			}

			if auth.TokenNeedsRefresh(cached) {
				io.Warning(stdout, "Token expired")
				if cached.RefreshToken != "" {
					io.Info(stdout, "Refresh token available — token will be refreshed on next command")
				} else {
					io.Info(stdout, "No refresh token — run 'grafanactl auth login' to re-authenticate")
				}
			} else {
				io.Success(stdout, "Authenticated")
				if cached.TokenExpiry != "" {
					io.Info(stdout, "Token expires: %s", cached.TokenExpiry)
				}
			}

			return nil
		},
	}

	return cmd
}
