package alerts

import (
	"errors"
	"fmt"
	"io"
	"sort"
	"strconv"
	"strings"
	"text/tabwriter"
	"time"

	cmdconfig "github.com/grafana/grafanactl/cmd/grafanactl/config"
	cmdio "github.com/grafana/grafanactl/cmd/grafanactl/io"
	"github.com/grafana/grafanactl/internal/format"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// NoiseEntry represents the noise analysis result for a single alert.
type NoiseEntry struct {
	AlertName      string `json:"alertName"`
	UID            string `json:"uid,omitempty"`
	FireCount      int    `json:"fireCount"`
	ResolveCount   int    `json:"resolveCount"`
	AvgDuration    string `json:"avgDuration"`
	Classification string `json:"classification"`
}

type noiseReportOpts struct {
	IO        cmdio.Options
	Period    string
	Threshold int
}

func (opts *noiseReportOpts) setup(flags *pflag.FlagSet) {
	flags.StringVar(&opts.Period, "period", "7d", "Time period to analyze (e.g., '24h', '7d', '30d')")
	flags.IntVar(&opts.Threshold, "threshold", 5, "Fire count above which an alert is classified as 'noisy'")
	opts.IO.RegisterCustomCodec("text", &noiseReportTableCodec{})
	opts.IO.DefaultFormat("text")
	opts.IO.BindFlags(flags)
}

func (opts *noiseReportOpts) Validate() error {
	if opts.Threshold < 1 {
		return errors.New("--threshold must be at least 1")
	}

	return opts.IO.Validate()
}

func noiseReportCmd(configOpts *cmdconfig.Options) *cobra.Command {
	opts := &noiseReportOpts{}

	cmd := &cobra.Command{
		Use:   "noise-report",
		Args:  cobra.NoArgs,
		Short: "Analyze alert noise patterns",
		Long:  "Analyze alert firing patterns over a time period to identify noisy alerts that fire frequently vs meaningful alerts.",
		Example: `
	grafanactl alerts noise-report
	grafanactl alerts noise-report --period 7d --threshold 5
	grafanactl alerts noise-report --period 30d -o json`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if err := opts.Validate(); err != nil {
				return err
			}

			cfg, err := configOpts.LoadConfig(cmd.Context())
			if err != nil {
				return err
			}

			currentCtx := cfg.GetCurrentContext()

			duration, err := parsePeriod(opts.Period)
			if err != nil {
				return err
			}

			now := time.Now()
			to := now.UnixMilli()
			from := now.Add(-duration).UnixMilli()

			stateEntries, err := fetchStateHistory(cmd.Context(), currentCtx, from/1000, to/1000)
			if err != nil {
				return fmt.Errorf("failed to fetch state history: %w", err)
			}

			entries := analyzeNoise(stateEntries, opts.Threshold)

			codec, err := opts.IO.Codec()
			if err != nil {
				return err
			}

			if err := codec.Encode(cmd.OutOrStdout(), entries); err != nil {
				return err
			}

			if opts.IO.OutputFormat == "text" {
				noisy := 0
				for _, e := range entries {
					if e.Classification == "noisy" {
						noisy++
					}
				}

				fmt.Fprintf(cmd.OutOrStdout(), "\n%d noisy, %d meaningful, %d total\n", noisy, len(entries)-noisy, len(entries))
			}

			return nil
		},
	}

	opts.setup(cmd.Flags())

	return cmd
}

func analyzeNoise(entries []stateHistoryEntry, threshold int) []NoiseEntry {
	type alertStats struct {
		uid           string
		fireCount     int
		resolveCount  int
		totalFiringMs int64
		firingPeriods int
	}

	statsByName := make(map[string]*alertStats)
	entriesByRule := make(map[string][]stateHistoryEntry)

	for _, entry := range entries {
		stats, ok := statsByName[entry.RuleTitle]
		if !ok {
			stats = &alertStats{}
			statsByName[entry.RuleTitle] = stats
		}

		if stats.uid == "" && entry.RuleUID != "" {
			stats.uid = entry.RuleUID
		}

		switch strings.ToLower(entry.Current) {
		case "alerting", "firing":
			stats.fireCount++
		case "ok", "normal":
			stats.resolveCount++
		}

		entriesByRule[entry.RuleTitle] = append(entriesByRule[entry.RuleTitle], entry)
	}

	// Compute durations from fire→resolve transitions.
	for name, ruleEntries := range entriesByRule {
		sort.Slice(ruleEntries, func(i, j int) bool {
			return ruleEntries[i].Timestamp.Before(ruleEntries[j].Timestamp)
		})

		stats := statsByName[name]

		var lastFireTime *time.Time

		for _, entry := range ruleEntries {
			switch strings.ToLower(entry.Current) {
			case "alerting", "firing":
				ts := entry.Timestamp
				lastFireTime = &ts
			case "ok", "normal":
				if lastFireTime != nil {
					duration := entry.Timestamp.Sub(*lastFireTime)
					stats.totalFiringMs += duration.Milliseconds()
					stats.firingPeriods++
					lastFireTime = nil
				}
			}
		}
	}

	results := make([]NoiseEntry, 0, len(statsByName))

	for name, stats := range statsByName {
		avgDur := ""
		if stats.firingPeriods > 0 {
			avg := time.Duration(stats.totalFiringMs/int64(stats.firingPeriods)) * time.Millisecond
			avgDur = avg.Truncate(time.Second).String()
		}

		classification := "meaningful"
		if stats.fireCount > threshold {
			classification = "noisy"
		}

		results = append(results, NoiseEntry{
			AlertName:      name,
			UID:            stats.uid,
			FireCount:      stats.fireCount,
			ResolveCount:   stats.resolveCount,
			AvgDuration:    avgDur,
			Classification: classification,
		})
	}

	sort.Slice(results, func(i, j int) bool {
		return results[i].FireCount > results[j].FireCount
	})

	return results
}

func parsePeriod(period string) (time.Duration, error) {
	if strings.HasSuffix(period, "d") {
		days, err := strconv.Atoi(strings.TrimSuffix(period, "d"))
		if err != nil {
			return 0, fmt.Errorf("invalid period %q", period)
		}

		return time.Duration(days) * 24 * time.Hour, nil
	}

	d, err := time.ParseDuration(period)
	if err != nil {
		return 0, fmt.Errorf("invalid period %q: %w", period, err)
	}

	return d, nil
}

// noiseReportTableCodec encodes []NoiseEntry as a table.
type noiseReportTableCodec struct{}

func (c *noiseReportTableCodec) Format() format.Format {
	return "text"
}

func (c *noiseReportTableCodec) Encode(output io.Writer, input any) error {
	entries, ok := input.([]NoiseEntry)
	if !ok {
		return fmt.Errorf("expected []NoiseEntry, got %T", input)
	}

	out := tabwriter.NewWriter(output, 0, 4, 2, ' ', tabwriter.TabIndent|tabwriter.DiscardEmptyColumns)

	fmt.Fprintf(out, "ALERT_NAME\tUID\tFIRES\tRESOLVES\tAVG_DURATION\tCLASSIFICATION\n")

	for _, entry := range entries {
		fmt.Fprintf(out, "%s\t%s\t%d\t%d\t%s\t%s\n",
			entry.AlertName,
			entry.UID,
			entry.FireCount,
			entry.ResolveCount,
			entry.AvgDuration,
			entry.Classification,
		)
	}

	return out.Flush()
}

func (c *noiseReportTableCodec) Decode(io.Reader, any) error {
	return errors.New("noise report table codec does not support decoding")
}
