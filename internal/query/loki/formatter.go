package loki

import (
	"fmt"
	"io"
	"sort"
	"strconv"
	"strings"
	"text/tabwriter"
	"time"
)

func FormatQueryTable(w io.Writer, resp *QueryResponse) error {
	if len(resp.Data.Result) == 0 {
		fmt.Fprintln(w, "No data")
		return nil
	}

	tw := tabwriter.NewWriter(w, 0, 4, 2, ' ', 0)

	labelNames := collectStreamLabelNames(resp.Data.Result)

	header := make([]string, 0, len(labelNames)+2)
	for _, name := range labelNames {
		header = append(header, strings.ToUpper(name))
	}
	header = append(header, "TIMESTAMP", "LINE")
	fmt.Fprintln(tw, strings.Join(header, "\t"))

	for _, stream := range resp.Data.Result {
		for _, value := range stream.Values {
			row := make([]string, 0, len(labelNames)+2)
			for _, name := range labelNames {
				if val, ok := stream.Stream[name]; ok {
					row = append(row, val)
				} else {
					row = append(row, "")
				}
			}

			if len(value) >= 2 {
				ts := parseNanoTimestamp(value[0])
				line := value[1]
				if len(line) > 100 {
					line = line[:97] + "..."
				}
				row = append(row, ts, line)
			}
			fmt.Fprintln(tw, strings.Join(row, "\t"))
		}
	}

	return tw.Flush()
}

func FormatLabelsTable(w io.Writer, resp *LabelsResponse) error {
	tw := tabwriter.NewWriter(w, 0, 4, 2, ' ', 0)
	fmt.Fprintln(tw, "LABEL")
	for _, label := range resp.Data {
		fmt.Fprintln(tw, label)
	}
	return tw.Flush()
}

func FormatSeriesTable(w io.Writer, resp *SeriesResponse) error {
	if len(resp.Data) == 0 {
		fmt.Fprintln(w, "No series found")
		return nil
	}

	labelNames := collectLabelNames(resp.Data)

	tw := tabwriter.NewWriter(w, 0, 4, 2, ' ', 0)

	header := make([]string, 0, len(labelNames))
	for _, name := range labelNames {
		header = append(header, strings.ToUpper(name))
	}
	fmt.Fprintln(tw, strings.Join(header, "\t"))

	for _, series := range resp.Data {
		row := make([]string, 0, len(labelNames))
		for _, name := range labelNames {
			if val, ok := series[name]; ok {
				row = append(row, val)
			} else {
				row = append(row, "")
			}
		}
		fmt.Fprintln(tw, strings.Join(row, "\t"))
	}

	return tw.Flush()
}

func collectStreamLabelNames(streams []StreamEntry) []string {
	nameSet := make(map[string]struct{})
	for _, stream := range streams {
		for name := range stream.Stream {
			nameSet[name] = struct{}{}
		}
	}

	names := make([]string, 0, len(nameSet))
	for name := range nameSet {
		names = append(names, name)
	}
	sort.Strings(names)

	return names
}

func collectLabelNames(series []map[string]string) []string {
	nameSet := make(map[string]struct{})
	for _, s := range series {
		for name := range s {
			nameSet[name] = struct{}{}
		}
	}

	names := make([]string, 0, len(nameSet))
	for name := range nameSet {
		names = append(names, name)
	}
	sort.Strings(names)

	return names
}

func parseNanoTimestamp(tsStr string) string {
	nanos, err := strconv.ParseInt(tsStr, 10, 64)
	if err != nil {
		return tsStr
	}
	t := time.Unix(0, nanos)
	return t.Format(time.RFC3339Nano)
}
