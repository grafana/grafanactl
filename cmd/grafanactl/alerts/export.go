package alerts

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"

	cmdconfig "github.com/grafana/grafanactl/cmd/grafanactl/config"
	"github.com/grafana/grafanactl/internal/httputils"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

type exportOpts struct {
	Format    string
	RuleUID   string
	FolderUID string
	Group     string
}

func (opts *exportOpts) setup(flags *pflag.FlagSet) {
	flags.StringVar(&opts.Format, "format", "yaml", "Export format: json, yaml, or hcl")
	flags.StringVar(&opts.RuleUID, "rule-uid", "", "Export a single alert rule by UID")
	flags.StringVar(&opts.FolderUID, "folder-uid", "", "Export alert rules from a specific folder")
	flags.StringVar(&opts.Group, "group", "", "Export alert rules from a specific group (requires --folder-uid)")
}

func (opts *exportOpts) Validate() error {
	switch opts.Format {
	case "json", "yaml", "hcl":
		// valid
	default:
		return fmt.Errorf("unsupported export format %q: must be json, yaml, or hcl", opts.Format)
	}

	if opts.Group != "" && opts.FolderUID == "" {
		return errors.New("--group requires --folder-uid to be specified")
	}

	if opts.RuleUID != "" && (opts.FolderUID != "" || opts.Group != "") {
		return errors.New("--rule-uid cannot be combined with --folder-uid or --group")
	}

	return nil
}

func exportCmd(configOpts *cmdconfig.Options) *cobra.Command {
	opts := &exportOpts{}

	cmd := &cobra.Command{
		Use:   "export",
		Args:  cobra.NoArgs,
		Short: "Export alert rules in provisioning format",
		Long:  "Export alert rules from Grafana in JSON, YAML, or HCL format for provisioning workflows.",
		Example: `
	grafanactl alerts export
	grafanactl alerts export --format hcl
	grafanactl alerts export --format json --rule-uid my-alert
	grafanactl alerts export --folder-uid my-folder
	grafanactl alerts export --folder-uid my-folder --group my-group`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if err := opts.Validate(); err != nil {
				return err
			}

			cfg, err := configOpts.LoadConfig(cmd.Context())
			if err != nil {
				return err
			}

			currentCtx := cfg.GetCurrentContext()

			exportURL, err := url.Parse(currentCtx.Grafana.Server)
			if err != nil {
				return fmt.Errorf("invalid server URL: %w", err)
			}

			exportURL.Path += "/api/v1/provisioning/alert-rules/export"

			q := exportURL.Query()
			q.Set("format", opts.Format)

			if opts.RuleUID != "" {
				q.Set("ruleUid", opts.RuleUID)
			}

			if opts.FolderUID != "" {
				q.Set("folderUid", opts.FolderUID)
			}

			if opts.Group != "" {
				q.Set("group", opts.Group)
			}

			exportURL.RawQuery = q.Encode()

			req, err := http.NewRequestWithContext(cmd.Context(), http.MethodGet, exportURL.String(), nil)
			if err != nil {
				return err
			}

			if currentCtx.Grafana.APIToken != "" {
				req.Header.Set("Authorization", "Bearer "+currentCtx.Grafana.APIToken)
			} else if currentCtx.Grafana.User != "" && currentCtx.Grafana.Password != "" {
				req.SetBasicAuth(currentCtx.Grafana.User, currentCtx.Grafana.Password)
			}

			if currentCtx.Grafana.OrgID != 0 {
				req.Header.Set("X-Grafana-Org-Id", strconv.FormatInt(currentCtx.Grafana.OrgID, 10))
			}

			httpClient, err := httputils.NewHTTPClient(currentCtx)
			if err != nil {
				return err
			}

			resp, err := httpClient.Do(req)
			if err != nil {
				return fmt.Errorf("export request failed: %w", err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				body, _ := io.ReadAll(resp.Body)
				return fmt.Errorf("export failed with status %d: %s", resp.StatusCode, string(body))
			}

			_, err = io.Copy(cmd.OutOrStdout(), resp.Body)

			return err
		},
	}

	opts.setup(cmd.Flags())

	return cmd
}
