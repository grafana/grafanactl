package alerts

import (
	"errors"
	"fmt"
	"io"
	"strings"
	"text/tabwriter"

	"github.com/grafana/grafana-openapi-client-go/models"
	cmdconfig "github.com/grafana/grafanactl/cmd/grafanactl/config"
	cmdio "github.com/grafana/grafanactl/cmd/grafanactl/io"
	"github.com/grafana/grafanactl/internal/format"
	"github.com/grafana/grafanactl/internal/grafana"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// AlertListEntry represents a merged alert rule with both provisioning and runtime state.
type AlertListEntry struct {
	Title     string
	UID       string
	FolderUID string
	RuleGroup string
	Status    string // active/paused (from provisioning API)
	State     string // firing/pending/inactive/error/unknown (from Prometheus API)
}

type listOpts struct {
	IO    cmdio.Options
	State string
}

func (opts *listOpts) setup(flags *pflag.FlagSet) {
	flags.StringVar(&opts.State, "state", "", "Filter by runtime state (firing, pending, inactive, error)")
	opts.IO.RegisterCustomCodec("text", &alertListTableCodec{})
	opts.IO.DefaultFormat("text")
	opts.IO.BindFlags(flags)
}

func (opts *listOpts) Validate() error {
	if opts.State != "" {
		valid := map[string]bool{"firing": true, "pending": true, "inactive": true, "error": true}
		if !valid[strings.ToLower(opts.State)] {
			return fmt.Errorf("invalid --state value %q: must be one of firing, pending, inactive, error", opts.State)
		}
	}

	return opts.IO.Validate()
}

func listCmd(configOpts *cmdconfig.Options) *cobra.Command {
	opts := &listOpts{}

	cmd := &cobra.Command{
		Use:   "list",
		Args:  cobra.NoArgs,
		Short: "List all alert rules",
		Long:  "List all alert rules from Grafana via the Provisioning API, enriched with runtime state from the Prometheus-compatible rules API.",
		Example: `
	grafanactl alerts list
	grafanactl alerts list --state firing
	grafanactl alerts list -o json
	grafanactl alerts list -o yaml`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if err := opts.Validate(); err != nil {
				return err
			}

			cfg, err := configOpts.LoadConfig(cmd.Context())
			if err != nil {
				return err
			}

			currentCtx := cfg.GetCurrentContext()

			gClient, err := grafana.ClientFromContext(currentCtx)
			if err != nil {
				return err
			}

			resp, err := gClient.Provisioning.GetAlertRules()
			if err != nil {
				return fmt.Errorf("failed to list alert rules: %w", err)
			}

			rules := resp.Payload

			// For non-text output, return raw provisioning data (backward compatible).
			codec, err := opts.IO.Codec()
			if err != nil {
				return err
			}

			if opts.IO.OutputFormat != "text" {
				return codec.Encode(cmd.OutOrStdout(), rules)
			}

			// Fetch runtime state (best-effort, don't fail if unavailable).
			runtimeStateByTitle := make(map[string]string)

			rulesResp, err := fetchRulesFromPrometheusAPI(cmd.Context(), currentCtx)
			if err == nil {
				for _, group := range rulesResp.Data.RuleGroups {
					for _, rule := range group.Rules {
						runtimeStateByTitle[rule.Name] = rule.State
					}
				}
			}

			// Merge provisioning rules with runtime state.
			entries := mergeProvisioningWithRuntime(rules, runtimeStateByTitle)

			// Apply --state filter if specified.
			if opts.State != "" {
				entries = filterByState(entries, strings.ToLower(opts.State))
			}

			if err := codec.Encode(cmd.OutOrStdout(), entries); err != nil {
				return err
			}

			fmt.Fprintf(cmd.OutOrStdout(), "\nTotal: %d alert rule(s)\n", len(entries))

			return nil
		},
	}

	opts.setup(cmd.Flags())

	return cmd
}

func mergeProvisioningWithRuntime(rules models.ProvisionedAlertRules, runtimeStateByTitle map[string]string) []AlertListEntry {
	entries := make([]AlertListEntry, 0, len(rules))

	for _, rule := range rules {
		status := "active"
		if rule.IsPaused {
			status = "paused"
		}

		title := derefStr(rule.Title)

		state := "unknown"
		if s, ok := runtimeStateByTitle[title]; ok {
			state = s
		}

		entries = append(entries, AlertListEntry{
			Title:     title,
			UID:       rule.UID,
			FolderUID: derefStr(rule.FolderUID),
			RuleGroup: derefStr(rule.RuleGroup),
			Status:    status,
			State:     state,
		})
	}

	return entries
}

func filterByState(entries []AlertListEntry, state string) []AlertListEntry {
	filtered := make([]AlertListEntry, 0, len(entries))

	for _, e := range entries {
		if strings.EqualFold(e.State, state) {
			filtered = append(filtered, e)
		}
	}

	return filtered
}

// alertListTableCodec encodes []AlertListEntry as a table with a STATE column.
type alertListTableCodec struct{}

func (c *alertListTableCodec) Format() format.Format {
	return "text"
}

func (c *alertListTableCodec) Encode(output io.Writer, input any) error {
	entries, ok := input.([]AlertListEntry)
	if !ok {
		return fmt.Errorf("expected []AlertListEntry, got %T", input)
	}

	out := tabwriter.NewWriter(output, 0, 4, 2, ' ', tabwriter.TabIndent|tabwriter.DiscardEmptyColumns)

	fmt.Fprintf(out, "TITLE\tUID\tFOLDER\tGROUP\tSTATUS\tSTATE\n")

	for _, entry := range entries {
		fmt.Fprintf(out, "%s\t%s\t%s\t%s\t%s\t%s\n",
			entry.Title,
			entry.UID,
			entry.FolderUID,
			entry.RuleGroup,
			entry.Status,
			entry.State,
		)
	}

	return out.Flush()
}

func (c *alertListTableCodec) Decode(io.Reader, any) error {
	return errors.New("alert list table codec does not support decoding")
}

// alertTableCodec encodes models.ProvisionedAlertRules as a table (used by search command).
type alertTableCodec struct{}

func (c *alertTableCodec) Format() format.Format {
	return "text"
}

func (c *alertTableCodec) Encode(output io.Writer, input any) error {
	rules, ok := input.(models.ProvisionedAlertRules)
	if !ok {
		return fmt.Errorf("expected models.ProvisionedAlertRules, got %T", input)
	}

	out := tabwriter.NewWriter(output, 0, 4, 2, ' ', tabwriter.TabIndent|tabwriter.DiscardEmptyColumns)

	fmt.Fprintf(out, "TITLE\tUID\tFOLDER\tGROUP\tSTATUS\n")

	for _, rule := range rules {
		status := "active"
		if rule.IsPaused {
			status = "paused"
		}

		fmt.Fprintf(out, "%s\t%s\t%s\t%s\t%s\n",
			derefStr(rule.Title),
			rule.UID,
			derefStr(rule.FolderUID),
			derefStr(rule.RuleGroup),
			status,
		)
	}

	return out.Flush()
}

func (c *alertTableCodec) Decode(io.Reader, any) error {
	return errors.New("alert table codec does not support decoding")
}

func derefStr(s *string) string {
	if s == nil {
		return ""
	}

	return *s
}
