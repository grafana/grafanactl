package graph

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/grafana/grafanactl/internal/graph"
	"github.com/grafana/grafanactl/internal/query/loki"
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
	TextOnly  bool
}

func (opts *graphOpts) setup(flags *pflag.FlagSet) {
	flags.StringVarP(&opts.File, "file", "f", "", "Read data from file instead of stdin")
	flags.StringVarP(&opts.Title, "title", "t", "", "Chart title")
	flags.IntVarP(&opts.Width, "width", "W", 0, "Chart width (default: terminal width)")
	flags.IntVarP(&opts.Height, "height", "H", 0, "Chart height (default: terminal height / 2)")
	flags.StringVar(&opts.ChartType, "type", "line", "Chart type: line, bar")
	flags.BoolVar(&opts.TextOnly, "text", false, "Output text instead of ASCII chart")
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
		Long: `Render ASCII charts from query results.

The graph command reads JSON query output from stdin or a file and renders
an ASCII chart in the terminal using braille characters for high resolution.

Input should be in Prometheus or Loki query response format (the JSON output from
'grafanactl query -o json'). The format is automatically detected.`,
		Example: `
	# Pipe Prometheus query results to graph (use datasource UID from 'grafanactl datasources list')
	grafanactl query -d <datasource-uid> -e 'rate(http_requests_total[5m])' --start now-1h --end now -o json | grafanactl graph

	# Pipe Loki metric query results to graph
	grafanactl query -d <loki-uid> -t loki -e 'sum(rate({job="varlogs"}[5m]))' --start now-1h --end now --step 1m -o json | grafanactl graph

	# Read from file
	grafanactl graph -f results.json --title "HTTP Request Rate"

	# Render as bar chart
	grafanactl graph -f results.json --type bar

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

			// Detect format by checking resultType
			var intermediate map[string]any
			if err := json.Unmarshal(data, &intermediate); err != nil {
				return fmt.Errorf("failed to parse JSON: %w", err)
			}

			chartData, err := parseQueryResponse(data, intermediate)
			if err != nil {
				return err
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
			chartOpts.TextOnly = opts.TextOnly

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

func parseQueryResponse(data []byte, intermediate map[string]any) (*graph.ChartData, error) {
	dataMap, ok := intermediate["data"].(map[string]any)
	if !ok {
		return nil, errors.New("invalid query response format: missing data.resultType field")
	}

	resultType, _ := dataMap["resultType"].(string)

	switch resultType {
	case "streams":
		// Loki response
		var resp loki.QueryResponse
		if err := json.Unmarshal(data, &resp); err != nil {
			return nil, fmt.Errorf("failed to parse Loki response: %w", err)
		}
		chartData, err := graph.FromLokiResponse(&resp)
		if err != nil {
			return nil, fmt.Errorf("failed to convert Loki data: %w", err)
		}
		return chartData, nil
	case "matrix", "vector":
		// Prometheus response
		var resp prometheus.QueryResponse
		if err := json.Unmarshal(data, &resp); err != nil {
			return nil, fmt.Errorf("failed to parse Prometheus response: %w", err)
		}
		chartData, err := graph.FromPrometheusResponse(&resp)
		if err != nil {
			return nil, fmt.Errorf("failed to convert Prometheus data: %w", err)
		}
		return chartData, nil
	default:
		return nil, fmt.Errorf("unsupported resultType: %s (expected 'streams', 'matrix', or 'vector')", resultType)
	}
}
