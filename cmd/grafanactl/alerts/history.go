package alerts

import (
	"errors"
	"fmt"
	"io"
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

type historyEntry struct {
	Time      string `json:"time"`
	AlertName string `json:"alertName"`
	PrevState string `json:"prevState"`
	NewState  string `json:"newState"`
	Duration  string `json:"duration,omitempty"`
}

type historyOpts struct {
	IO    cmdio.Options
	From  string
	To    string
	Limit int
}

func (opts *historyOpts) setup(flags *pflag.FlagSet) {
	flags.StringVar(&opts.From, "from", "24h", "Start time (duration like '24h', '7d', '30d' relative to now, or epoch ms)")
	flags.StringVar(&opts.To, "to", "now", "End time (duration like '1h' relative to now, 'now', or epoch ms)")
	flags.IntVar(&opts.Limit, "limit", 1000, "Maximum number of state history entries to return")
	opts.IO.RegisterCustomCodec("text", &historyTableCodec{})
	opts.IO.DefaultFormat("text")
	opts.IO.BindFlags(flags)
}

func (opts *historyOpts) Validate() error {
	return opts.IO.Validate()
}

func historyCmd(configOpts *cmdconfig.Options) *cobra.Command {
	opts := &historyOpts{}

	cmd := &cobra.Command{
		Use:   "history",
		Args:  cobra.NoArgs,
		Short: "Show alert state change history",
		Long:  "Show alert state change history from Grafana state history, displaying when alerts transitioned between states.",
		Example: `
	grafanactl alerts history
	grafanactl alerts history --from 7d
	grafanactl alerts history --from 24h --to now
	grafanactl alerts history --from 7d -o json`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if err := opts.Validate(); err != nil {
				return err
			}

			cfg, err := configOpts.LoadConfig(cmd.Context())
			if err != nil {
				return err
			}

			currentCtx := cfg.GetCurrentContext()

			from, err := parseTimeArg(opts.From, true)
			if err != nil {
				return fmt.Errorf("invalid --from value: %w", err)
			}

			to, err := parseTimeArg(opts.To, false)
			if err != nil {
				return fmt.Errorf("invalid --to value: %w", err)
			}

			history, err := fetchStateHistory(cmd.Context(), currentCtx, from/1000, to/1000)
			if err != nil {
				return fmt.Errorf("failed to fetch alert state history: %w", err)
			}

			if opts.Limit > 0 && len(history) > opts.Limit {
				history = history[:opts.Limit]
			}

			entries := stateHistoryToHistoryEntries(history)

			codec, err := opts.IO.Codec()
			if err != nil {
				return err
			}

			if err := codec.Encode(cmd.OutOrStdout(), entries); err != nil {
				return err
			}

			if opts.IO.OutputFormat == "text" {
				fmt.Fprintf(cmd.OutOrStdout(), "\nTotal: %d state change(s)\n", len(entries))
			}

			return nil
		},
	}

	opts.setup(cmd.Flags())

	return cmd
}

// parseTimeArg parses a time argument that can be:
// - "now" -> current time
// - duration string like "24h", "7d", "30d" -> relative to now (subtracted)
// - epoch milliseconds as string
func parseTimeArg(arg string, _ bool) (int64, error) {
	if arg == "now" || arg == "" {
		return time.Now().UnixMilli(), nil
	}

	if ms, err := strconv.ParseInt(arg, 10, 64); err == nil {
		return ms, nil
	}

	durStr := arg
	if strings.HasSuffix(durStr, "d") {
		days, err := strconv.Atoi(strings.TrimSuffix(durStr, "d"))
		if err != nil {
			return 0, fmt.Errorf("invalid duration %q", arg)
		}

		durStr = fmt.Sprintf("%dh", days*24)
	}

	dur, err := time.ParseDuration(durStr)
	if err != nil {
		return 0, fmt.Errorf("invalid time %q: must be 'now', a duration (24h, 7d), or epoch ms", arg)
	}

	return time.Now().Add(-dur).UnixMilli(), nil
}

func stateHistoryToHistoryEntries(history []stateHistoryEntry) []historyEntry {
	entries := make([]historyEntry, 0, len(history))

	for _, h := range history {
		entry := historyEntry{
			Time:      h.Timestamp.UTC().Format(time.RFC3339),
			AlertName: h.RuleTitle,
			PrevState: h.Previous,
			NewState:  h.Current,
		}

		entries = append(entries, entry)
	}

	return entries
}

func formatDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	}

	if d < time.Hour {
		return fmt.Sprintf("%dm%ds", int(d.Minutes()), int(d.Seconds())%60)
	}

	hours := int(d.Hours())
	minutes := int(d.Minutes()) % 60

	if hours < 24 {
		return fmt.Sprintf("%dh%dm", hours, minutes)
	}

	days := hours / 24
	hours = hours % 24

	return fmt.Sprintf("%dd%dh%dm", days, hours, minutes)
}

// historyTableCodec encodes []historyEntry as a table.
type historyTableCodec struct{}

func (c *historyTableCodec) Format() format.Format {
	return "text"
}

func (c *historyTableCodec) Encode(output io.Writer, input any) error {
	entries, ok := input.([]historyEntry)
	if !ok {
		return fmt.Errorf("expected []historyEntry, got %T", input)
	}

	out := tabwriter.NewWriter(output, 0, 4, 2, ' ', tabwriter.TabIndent|tabwriter.DiscardEmptyColumns)

	fmt.Fprintf(out, "TIME\tALERT_NAME\tPREVIOUS_STATE\tNEW_STATE\tDURATION\n")

	for _, entry := range entries {
		fmt.Fprintf(out, "%s\t%s\t%s\t%s\t%s\n",
			entry.Time,
			entry.AlertName,
			entry.PrevState,
			entry.NewState,
			entry.Duration,
		)
	}

	return out.Flush()
}

func (c *historyTableCodec) Decode(io.Reader, any) error {
	return errors.New("history table codec does not support decoding")
}
