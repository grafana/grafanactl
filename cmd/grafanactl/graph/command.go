package graph

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/grafana/grafanactl/internal/graph"
	"github.com/grafana/grafanactl/internal/query/prometheus"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

type graphOpts struct {
	File      string
	Title     string
	Width     int
	Height    int
	ChartType string
}

func (opts *graphOpts) setup(flags *pflag.FlagSet) {
	flags.StringVarP(&opts.File, "file", "f", "", "Read data from file instead of stdin")
	flags.StringVarP(&opts.Title, "title", "t", "", "Chart title")
	flags.IntVarP(&opts.Width, "width", "W", 0, "Chart width (default: terminal width)")
	flags.IntVarP(&opts.Height, "height", "H", 0, "Chart height (default: terminal height / 2)")
	flags.StringVar(&opts.ChartType, "type", "line", "Chart type: line, bar")
}

func (opts *graphOpts) Validate() error {
	if opts.ChartType != "line" && opts.ChartType != "bar" {
		return errors.New("chart type must be 'line' or 'bar'")
	}
	return nil
}

// Command returns the graph command.
func Command() *cobra.Command {
	opts := &graphOpts{}

	cmd := &cobra.Command{
		Use:   "graph",
		Short: "Render ASCII charts from query results",
		Long: `Render ASCII charts from Prometheus query results.

The graph command reads JSON query output from stdin or a file and renders
an ASCII chart in the terminal using braille characters for high resolution.

Input should be in Prometheus query response format (the JSON output from
'grafanactl query prometheus -o json').`,
		Example: `
	# Pipe query results to graph (use datasource UID from 'grafanactl datasources list')
	grafanactl query prometheus -d <datasource-uid> -e 'rate(http_requests_total[5m])' --start now-1h --end now -o json | grafanactl graph

	# Read from file
	grafanactl graph -f results.json --title "HTTP Request Rate"

	# Custom dimensions
	grafanactl graph -f results.json --width 100 --height 20`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if err := opts.Validate(); err != nil {
				return err
			}

			// Read input
			var input io.Reader
			if opts.File != "" {
				f, err := os.Open(opts.File)
				if err != nil {
					return fmt.Errorf("failed to open file: %w", err)
				}
				defer f.Close()
				input = f
			} else {
				input = cmd.InOrStdin()
			}

			data, err := io.ReadAll(input)
			if err != nil {
				return fmt.Errorf("failed to read input: %w", err)
			}

			// Parse as Prometheus response
			var resp prometheus.QueryResponse
			if err := json.Unmarshal(data, &resp); err != nil {
				return fmt.Errorf("failed to parse JSON: %w (expected Prometheus query response format)", err)
			}

			// Convert to chart data
			chartData, err := graph.FromPrometheusResponse(&resp)
			if err != nil {
				return fmt.Errorf("failed to convert data: %w", err)
			}

			// Get chart options
			chartOpts := graph.DefaultChartOptions()
			if opts.Width > 0 {
				chartOpts.Width = opts.Width
			}
			if opts.Height > 0 {
				chartOpts.Height = opts.Height
			}
			chartOpts.Title = opts.Title

			// Render chart based on type
			switch opts.ChartType {
			case "bar":
				return graph.RenderBarChart(cmd.OutOrStdout(), chartData, chartOpts)
			default:
				return graph.RenderLineChart(cmd.OutOrStdout(), chartData, chartOpts)
			}
		},
	}

	opts.setup(cmd.Flags())
	return cmd
}
