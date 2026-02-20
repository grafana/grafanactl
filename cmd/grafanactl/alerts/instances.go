package alerts

import (
	"errors"
	"fmt"
	"io"
	"sort"
	"strings"
	"text/tabwriter"

	cmdconfig "github.com/grafana/grafanactl/cmd/grafanactl/config"
	cmdio "github.com/grafana/grafanactl/cmd/grafanactl/io"
	"github.com/grafana/grafanactl/internal/format"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

type instancesOpts struct {
	IO    cmdio.Options
	State string
}

func (opts *instancesOpts) setup(flags *pflag.FlagSet) {
	flags.StringVar(&opts.State, "state", "", "Filter by instance state (firing, pending)")
	opts.IO.RegisterCustomCodec("text", &instanceTableCodec{})
	opts.IO.DefaultFormat("text")
	opts.IO.BindFlags(flags)
}

func (opts *instancesOpts) Validate() error {
	if opts.State != "" && opts.State != "firing" && opts.State != "pending" {
		return fmt.Errorf("invalid state filter %q: must be one of: firing, pending", opts.State)
	}

	return opts.IO.Validate()
}

// instanceEntry represents a single alert instance for display.
type instanceEntry struct {
	RuleName string `json:"ruleName"`
	State    string `json:"state"`
	ActiveAt string `json:"activeAt"`
	Labels   string `json:"labels"`
	Value    string `json:"value"`
}

func instancesCmd(configOpts *cmdconfig.Options) *cobra.Command {
	opts := &instancesOpts{}

	cmd := &cobra.Command{
		Use:   "instances",
		Args:  cobra.NoArgs,
		Short: "List currently firing alert instances",
		Long:  "List currently firing alert instances from Grafana with their labels and values.",
		Example: `
	grafanactl alerts instances
	grafanactl alerts instances --state firing
	grafanactl alerts instances -o json`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if err := opts.Validate(); err != nil {
				return err
			}

			cfg, err := configOpts.LoadConfig(cmd.Context())
			if err != nil {
				return err
			}

			currentCtx := cfg.GetCurrentContext()

			rulesResp, err := fetchRulesFromPrometheusAPI(cmd.Context(), currentCtx)
			if err != nil {
				return fmt.Errorf("failed to fetch alert instances: %w", err)
			}

			var entries []instanceEntry
			for _, group := range rulesResp.Data.RuleGroups {
				for _, rule := range group.Rules {
					for _, alert := range rule.Alerts {
						if opts.State != "" && alert.State != opts.State {
							continue
						}

						activeAt := ""
						if alert.ActiveAt != nil {
							activeAt = alert.ActiveAt.Format("2006-01-02 15:04:05")
						}

						entries = append(entries, instanceEntry{
							RuleName: rule.Name,
							State:    alert.State,
							ActiveAt: activeAt,
							Labels:   formatLabels(alert.Labels),
							Value:    alert.Value,
						})
					}
				}
			}

			codec, err := opts.IO.Codec()
			if err != nil {
				return err
			}

			if err := codec.Encode(cmd.OutOrStdout(), entries); err != nil {
				return err
			}

			if opts.IO.OutputFormat == "text" {
				fmt.Fprintf(cmd.OutOrStdout(), "\nTotal: %d instance(s)\n", len(entries))
			}

			return nil
		},
	}

	opts.setup(cmd.Flags())

	return cmd
}

func formatLabels(labels map[string]string) string {
	if len(labels) == 0 {
		return ""
	}

	keys := make([]string, 0, len(labels))
	for k := range labels {
		keys = append(keys, k)
	}

	sort.Strings(keys)

	pairs := make([]string, 0, len(labels))
	for _, k := range keys {
		pairs = append(pairs, k+"="+labels[k])
	}

	return strings.Join(pairs, ",")
}

type instanceTableCodec struct{}

func (c *instanceTableCodec) Format() format.Format {
	return "text"
}

func (c *instanceTableCodec) Encode(output io.Writer, input any) error {
	entries, ok := input.([]instanceEntry)
	if !ok {
		return fmt.Errorf("expected []instanceEntry, got %T", input)
	}

	out := tabwriter.NewWriter(output, 0, 4, 2, ' ', tabwriter.TabIndent|tabwriter.DiscardEmptyColumns)

	fmt.Fprintf(out, "RULE\tSTATE\tACTIVE_SINCE\tLABELS\tVALUE\n")

	for _, e := range entries {
		fmt.Fprintf(out, "%s\t%s\t%s\t%s\t%s\n", e.RuleName, e.State, e.ActiveAt, e.Labels, e.Value)
	}

	return out.Flush()
}

func (c *instanceTableCodec) Decode(io.Reader, any) error {
	return errors.New("instance table codec does not support decoding")
}
