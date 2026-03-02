package query

import (
	"errors"
	"fmt"
	"io"
	"text/tabwriter"
	"time"

	cmdconfig "github.com/grafana/grafanactl/cmd/grafanactl/config"
	cmdio "github.com/grafana/grafanactl/cmd/grafanactl/io"
	"github.com/grafana/grafanactl/internal/format"
	"github.com/grafana/grafanactl/internal/query/prometheus"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

type prometheusOpts struct {
	IO          cmdio.Options
	Datasource  string
	Query       string
	Start       string
	End         string
	Step        string
}

func (opts *prometheusOpts) setup(flags *pflag.FlagSet) {
	opts.IO.RegisterCustomCodec("table", &prometheusTableCodec{})
	opts.IO.DefaultFormat("table")
	opts.IO.BindFlags(flags)

	flags.StringVarP(&opts.Datasource, "datasource", "d", "", "Datasource UID (required unless default-prometheus-datasource is configured)")
	flags.StringVarP(&opts.Query, "expr", "e", "", "PromQL query expression (required)")
	flags.StringVar(&opts.Start, "start", "", "Start time (RFC3339, Unix timestamp, or relative like 'now-1h')")
	flags.StringVar(&opts.End, "end", "", "End time (RFC3339, Unix timestamp, or relative like 'now')")
	flags.StringVar(&opts.Step, "step", "", "Query step (e.g., '15s', '1m')")
}

func (opts *prometheusOpts) Validate() error {
	if err := opts.IO.Validate(); err != nil {
		return err
	}

	if opts.Query == "" {
		return errors.New("query expression is required (use -e or --expr)")
	}

	return nil
}

func prometheusCmd(configOpts *cmdconfig.Options) *cobra.Command {
	opts := &prometheusOpts{}

	cmd := &cobra.Command{
		Use:   "prometheus",
		Short: "Execute Prometheus queries",
		Long:  "Execute PromQL queries against a Prometheus datasource via Grafana.",
		Example: `
	# First, find your datasource UID
	grafanactl datasources list

	# Instant query (use the UID from datasources list, not the name)
	grafanactl query prometheus -d <datasource-uid> -e 'up{job="grafana"}'

	# Range query
	grafanactl query prometheus -d <datasource-uid> -e 'rate(http_requests_total[5m])' --start now-1h --end now

	# Range query with step
	grafanactl query prometheus -d <datasource-uid> -e 'rate(http_requests_total[5m])' --start now-1h --end now --step 1m

	# Output as JSON
	grafanactl query prometheus -d <datasource-uid> -e 'up' -o json`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if err := opts.Validate(); err != nil {
				return err
			}

			ctx := cmd.Context()

			cfg, err := configOpts.LoadRESTConfig(ctx)
			if err != nil {
				return err
			}

			// Resolve datasource
			datasourceUID := opts.Datasource
			if datasourceUID == "" {
				fullCfg, err := configOpts.LoadConfig(ctx)
				if err != nil {
					return err
				}
				datasourceUID = fullCfg.GetCurrentContext().DefaultPrometheusDatasource
			}
			if datasourceUID == "" {
				return errors.New("datasource UID is required: use -d flag or set default-prometheus-datasource in config")
			}

			client, err := prometheus.NewClient(cfg)
			if err != nil {
				return fmt.Errorf("failed to create client: %w", err)
			}

			now := time.Now()
			start, err := ParseTime(opts.Start, now)
			if err != nil {
				return fmt.Errorf("invalid start time: %w", err)
			}

			end, err := ParseTime(opts.End, now)
			if err != nil {
				return fmt.Errorf("invalid end time: %w", err)
			}

			step, err := ParseDuration(opts.Step)
			if err != nil {
				return fmt.Errorf("invalid step: %w", err)
			}

			req := prometheus.QueryRequest{
				Query: opts.Query,
				Start: start,
				End:   end,
				Step:  step,
			}

			resp, err := client.Query(ctx, datasourceUID, req)
			if err != nil {
				return fmt.Errorf("query failed: %w", err)
			}

			codec, err := opts.IO.Codec()
			if err != nil {
				return err
			}

			if opts.IO.OutputFormat == "table" {
				return prometheus.FormatTable(cmd.OutOrStdout(), resp)
			}

			return codec.Encode(cmd.OutOrStdout(), resp)
		},
	}

	opts.setup(cmd.Flags())

	// Add subcommands
	cmd.AddCommand(labelsCmd(configOpts))
	cmd.AddCommand(targetsCmd(configOpts))

	return cmd
}

type prometheusTableCodec struct{}

func (c *prometheusTableCodec) Format() format.Format {
	return "table"
}

func (c *prometheusTableCodec) Encode(w io.Writer, data any) error {
	resp, ok := data.(*prometheus.QueryResponse)
	if !ok {
		return fmt.Errorf("invalid data type for prometheus table codec")
	}

	return prometheus.FormatTable(w, resp)
}

func (c *prometheusTableCodec) Decode(io.Reader, any) error {
	return errors.New("prometheus table codec does not support decoding")
}

type labelsOpts struct {
	IO         cmdio.Options
	Datasource string
	Label      string
	Metric     string
}

func (opts *labelsOpts) setup(flags *pflag.FlagSet) {
	opts.IO.RegisterCustomCodec("table", &labelsTableCodec{})
	opts.IO.DefaultFormat("table")
	opts.IO.BindFlags(flags)

	flags.StringVarP(&opts.Datasource, "datasource", "d", "", "Datasource UID (required unless default-prometheus-datasource is configured)")
	flags.StringVarP(&opts.Label, "label", "l", "", "Get values for this label (omit to list all labels)")
	flags.StringVarP(&opts.Metric, "metric", "m", "", "Get metadata for this metric (use with 'metadata' command)")
}

func (opts *labelsOpts) Validate() error {
	return opts.IO.Validate()
}

func labelsCmd(configOpts *cmdconfig.Options) *cobra.Command {
	opts := &labelsOpts{}

	cmd := &cobra.Command{
		Use:   "labels",
		Short: "List labels or label values",
		Long:  "List all labels or get values for a specific label from a Prometheus datasource.",
		Example: `
	# List all labels (use datasource UID, not name)
	grafanactl query prometheus labels -d <datasource-uid>

	# Get values for a specific label
	grafanactl query prometheus labels -d <datasource-uid> --label job

	# Output as JSON
	grafanactl query prometheus labels -d <datasource-uid> -o json`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if err := opts.Validate(); err != nil {
				return err
			}

			ctx := cmd.Context()

			cfg, err := configOpts.LoadRESTConfig(ctx)
			if err != nil {
				return err
			}

			// Resolve datasource
			datasourceUID := opts.Datasource
			if datasourceUID == "" {
				fullCfg, err := configOpts.LoadConfig(ctx)
				if err != nil {
					return err
				}
				datasourceUID = fullCfg.GetCurrentContext().DefaultPrometheusDatasource
			}
			if datasourceUID == "" {
				return errors.New("datasource UID is required: use -d flag or set default-prometheus-datasource in config")
			}

			client, err := prometheus.NewClient(cfg)
			if err != nil {
				return fmt.Errorf("failed to create client: %w", err)
			}

			codec, err := opts.IO.Codec()
			if err != nil {
				return err
			}

			if opts.Label != "" {
				resp, err := client.LabelValues(ctx, datasourceUID, opts.Label)
				if err != nil {
					return fmt.Errorf("failed to get label values: %w", err)
				}

				if opts.IO.OutputFormat == "table" {
					return prometheus.FormatLabelsTable(cmd.OutOrStdout(), resp)
				}
				return codec.Encode(cmd.OutOrStdout(), resp)
			}

			resp, err := client.Labels(ctx, datasourceUID)
			if err != nil {
				return fmt.Errorf("failed to get labels: %w", err)
			}

			if opts.IO.OutputFormat == "table" {
				return prometheus.FormatLabelsTable(cmd.OutOrStdout(), resp)
			}
			return codec.Encode(cmd.OutOrStdout(), resp)
		},
	}

	opts.setup(cmd.Flags())

	// Add metadata subcommand
	cmd.AddCommand(metadataCmd(configOpts))

	return cmd
}

type labelsTableCodec struct{}

func (c *labelsTableCodec) Format() format.Format {
	return "table"
}

func (c *labelsTableCodec) Encode(w io.Writer, data any) error {
	resp, ok := data.(*prometheus.LabelsResponse)
	if !ok {
		return fmt.Errorf("invalid data type for labels table codec")
	}

	return prometheus.FormatLabelsTable(w, resp)
}

func (c *labelsTableCodec) Decode(io.Reader, any) error {
	return errors.New("labels table codec does not support decoding")
}

type metadataOpts struct {
	IO         cmdio.Options
	Datasource string
	Metric     string
}

func (opts *metadataOpts) setup(flags *pflag.FlagSet) {
	opts.IO.RegisterCustomCodec("table", &metadataTableCodec{})
	opts.IO.DefaultFormat("table")
	opts.IO.BindFlags(flags)

	flags.StringVarP(&opts.Datasource, "datasource", "d", "", "Datasource UID (required unless default-prometheus-datasource is configured)")
	flags.StringVarP(&opts.Metric, "metric", "m", "", "Filter by metric name")
}

func (opts *metadataOpts) Validate() error {
	return opts.IO.Validate()
}

func metadataCmd(configOpts *cmdconfig.Options) *cobra.Command {
	opts := &metadataOpts{}

	cmd := &cobra.Command{
		Use:   "metadata",
		Short: "Get metric metadata",
		Long:  "Get metadata (type, help text) for metrics from a Prometheus datasource.",
		Example: `
	# Get all metric metadata (use datasource UID, not name)
	grafanactl query prometheus labels metadata -d <datasource-uid>

	# Get metadata for a specific metric
	grafanactl query prometheus labels metadata -d <datasource-uid> --metric http_requests_total

	# Output as JSON
	grafanactl query prometheus labels metadata -d <datasource-uid> -o json`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if err := opts.Validate(); err != nil {
				return err
			}

			ctx := cmd.Context()

			cfg, err := configOpts.LoadRESTConfig(ctx)
			if err != nil {
				return err
			}

			// Resolve datasource
			datasourceUID := opts.Datasource
			if datasourceUID == "" {
				fullCfg, err := configOpts.LoadConfig(ctx)
				if err != nil {
					return err
				}
				datasourceUID = fullCfg.GetCurrentContext().DefaultPrometheusDatasource
			}
			if datasourceUID == "" {
				return errors.New("datasource UID is required: use -d flag or set default-prometheus-datasource in config")
			}

			client, err := prometheus.NewClient(cfg)
			if err != nil {
				return fmt.Errorf("failed to create client: %w", err)
			}

			resp, err := client.Metadata(ctx, datasourceUID, opts.Metric)
			if err != nil {
				return fmt.Errorf("failed to get metadata: %w", err)
			}

			codec, err := opts.IO.Codec()
			if err != nil {
				return err
			}

			if opts.IO.OutputFormat == "table" {
				return prometheus.FormatMetadataTable(cmd.OutOrStdout(), resp)
			}
			return codec.Encode(cmd.OutOrStdout(), resp)
		},
	}

	opts.setup(cmd.Flags())
	return cmd
}

type metadataTableCodec struct{}

func (c *metadataTableCodec) Format() format.Format {
	return "table"
}

func (c *metadataTableCodec) Encode(w io.Writer, data any) error {
	resp, ok := data.(*prometheus.MetadataResponse)
	if !ok {
		return fmt.Errorf("invalid data type for metadata table codec")
	}

	tw := tabwriter.NewWriter(w, 0, 4, 2, ' ', 0)
	fmt.Fprintln(tw, "METRIC\tTYPE\tHELP")

	for metric, entries := range resp.Data {
		for _, entry := range entries {
			fmt.Fprintf(tw, "%s\t%s\t%s\n", metric, entry.Type, entry.Help)
		}
	}

	return tw.Flush()
}

func (c *metadataTableCodec) Decode(io.Reader, any) error {
	return errors.New("metadata table codec does not support decoding")
}

type targetsOpts struct {
	IO         cmdio.Options
	Datasource string
	State      string
}

func (opts *targetsOpts) setup(flags *pflag.FlagSet) {
	opts.IO.RegisterCustomCodec("table", &targetsTableCodec{})
	opts.IO.DefaultFormat("table")
	opts.IO.BindFlags(flags)

	flags.StringVarP(&opts.Datasource, "datasource", "d", "", "Datasource UID (required unless default-prometheus-datasource is configured)")
	flags.StringVar(&opts.State, "state", "", "Filter by target state: active, dropped, any (default: active)")
}

func (opts *targetsOpts) Validate() error {
	if opts.State != "" && opts.State != "active" && opts.State != "dropped" && opts.State != "any" {
		return errors.New("state must be 'active', 'dropped', or 'any'")
	}
	return opts.IO.Validate()
}

func targetsCmd(configOpts *cmdconfig.Options) *cobra.Command {
	opts := &targetsOpts{}

	cmd := &cobra.Command{
		Use:   "targets",
		Short: "List scrape targets",
		Long:  "List scrape targets from a Prometheus datasource.",
		Example: `
	# List active targets (use datasource UID, not name)
	grafanactl query prometheus targets -d <datasource-uid>

	# List dropped targets
	grafanactl query prometheus targets -d <datasource-uid> --state dropped

	# List all targets
	grafanactl query prometheus targets -d <datasource-uid> --state any

	# Output as JSON
	grafanactl query prometheus targets -d <datasource-uid> -o json`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if err := opts.Validate(); err != nil {
				return err
			}

			ctx := cmd.Context()

			cfg, err := configOpts.LoadRESTConfig(ctx)
			if err != nil {
				return err
			}

			// Resolve datasource
			datasourceUID := opts.Datasource
			if datasourceUID == "" {
				fullCfg, err := configOpts.LoadConfig(ctx)
				if err != nil {
					return err
				}
				datasourceUID = fullCfg.GetCurrentContext().DefaultPrometheusDatasource
			}
			if datasourceUID == "" {
				return errors.New("datasource UID is required: use -d flag or set default-prometheus-datasource in config")
			}

			client, err := prometheus.NewClient(cfg)
			if err != nil {
				return fmt.Errorf("failed to create client: %w", err)
			}

			resp, err := client.Targets(ctx, datasourceUID, opts.State)
			if err != nil {
				return fmt.Errorf("failed to get targets: %w", err)
			}

			codec, err := opts.IO.Codec()
			if err != nil {
				return err
			}

			if opts.IO.OutputFormat == "table" {
				return prometheus.FormatTargetsTable(cmd.OutOrStdout(), resp)
			}
			return codec.Encode(cmd.OutOrStdout(), resp)
		},
	}

	opts.setup(cmd.Flags())
	return cmd
}

type targetsTableCodec struct{}

func (c *targetsTableCodec) Format() format.Format {
	return "table"
}

func (c *targetsTableCodec) Encode(w io.Writer, data any) error {
	resp, ok := data.(*prometheus.TargetsResponse)
	if !ok {
		return fmt.Errorf("invalid data type for targets table codec")
	}

	return prometheus.FormatTargetsTable(w, resp)
}

func (c *targetsTableCodec) Decode(io.Reader, any) error {
	return errors.New("targets table codec does not support decoding")
}
