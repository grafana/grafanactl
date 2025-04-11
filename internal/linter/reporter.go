//nolint:wrapcheck
package linter

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/fatih/color"
	"github.com/olekukonko/tablewriter"
)

// Reporter formats and publishes linter reports.
type Reporter interface {
	// Publish formats and publishes a report to any appropriate target
	Publish(ctx context.Context, out io.Writer, report Report) error
}

var _ Reporter = (*CompactReporter)(nil)

// CompactReporter reports violations in a compact table.
type CompactReporter struct {
}

// Publish prints a compact report to the configured output.
func (reporter CompactReporter) Publish(_ context.Context, out io.Writer, r Report) error {
	if len(r.Violations) == 0 {
		_, err := fmt.Fprintln(out)

		return err
	}

	buffer := &strings.Builder{}
	table := tablewriter.NewWriter(buffer)

	table.SetHeader([]string{"Location", "Severity", "Rule", "Details"})
	table.SetAutoFormatHeaders(false)
	table.SetColWidth(80)
	table.SetAutoWrapText(true)

	for _, violation := range r.Violations {
		table.Append([]string{violation.Location.String(), violation.Severity, violation.Rule, violation.Details})
	}

	summary := fmt.Sprintf("%d %s linted , %d %s found.",
		r.Summary.FilesScanned, pluralize("file", r.Summary.FilesScanned),
		r.Summary.NumViolations, pluralize("violation", r.Summary.NumViolations))

	table.Render()

	_, err := fmt.Fprintln(out, strings.TrimSuffix(buffer.String(), ""), summary)

	return err
}

var _ Reporter = (*PrettyReporter)(nil)

// PrettyReporter is a Reporter for representing reports as tables.
type PrettyReporter struct {
}

// Publish prints a pretty report to the configured output.
//
//nolint:nestif
func (reporter PrettyReporter) Publish(_ context.Context, out io.Writer, r Report) error {
	table := buildPrettyViolationsTable(r.Violations)

	numsWarning, numsError := 0, 0

	for _, violation := range r.Violations {
		if violation.Severity == "warning" {
			numsWarning++
		} else if violation.Severity == "error" {
			numsError++
		}
	}

	footer := fmt.Sprintf("%d %s linted.", r.Summary.FilesScanned, pluralize("file", r.Summary.FilesScanned))

	if r.Summary.NumViolations == 0 {
		footer += " No violations found."
	} else {
		footer += fmt.Sprintf(" %d %s ", r.Summary.NumViolations, pluralize("violation", r.Summary.NumViolations))

		if numsWarning > 0 {
			footer += fmt.Sprintf("(%d %s, %d %s) found",
				numsError, pluralize("error", numsError), numsWarning, pluralize("warning", numsWarning),
			)
		} else {
			footer += "found"
		}

		if r.Summary.FilesScanned > 1 && r.Summary.FilesFailed > 0 {
			footer += fmt.Sprintf(" in %d %s.", r.Summary.FilesFailed, pluralize("file", r.Summary.FilesFailed))
		} else {
			footer += "."
		}
	}

	if r.Summary.RulesSkipped > 0 {
		footer += fmt.Sprintf(
			" %d %s skipped:\n",
			r.Summary.RulesSkipped,
			pluralize("rule", r.Summary.RulesSkipped),
		)
	}

	_, err := fmt.Fprint(out, table+footer+"\n")
	if err != nil {
		return fmt.Errorf("failed to write report: %w", err)
	}

	return err
}

func buildPrettyViolationsTable(violations []Violation) string {
	buffer := &strings.Builder{}
	table := tablewriter.NewWriter(buffer)

	table.SetNoWhiteSpace(true)
	table.SetTablePadding("\t")
	table.SetAlignment(tablewriter.ALIGN_LEFT)
	table.SetCenterSeparator("")
	table.SetColumnSeparator("")
	table.SetRowSeparator("")
	table.SetHeaderLine(false)
	table.SetBorder(false)
	table.SetAutoWrapText(false)

	numViolations := len(violations)

	// Note: it's tempting to use table.SetColumnColor here, but for whatever reason, that requires using
	// table.SetHeader as well, and we don't want a header for this format.

	yellow := color.New(color.FgYellow).SprintFunc()
	cyan := color.New(color.FgCyan).SprintFunc()
	red := color.New(color.FgRed).SprintFunc()

	for i, violation := range violations {
		description := red(violation.Description)
		if violation.Severity == "warning" {
			description = yellow(violation.Description)
		}

		table.Append([]string{yellow("Rule:"), violation.Rule})

		// if there is no support for color, then we show the level in addition
		// so that the level of the violation is still clear
		if color.NoColor {
			table.Append([]string{"Severity:", violation.Severity})
		}

		table.Append([]string{yellow("Description:"), description})
		table.Append([]string{yellow("Resource type:"), violation.ResourceType})
		table.Append([]string{yellow("Location:"), cyan(violation.Location.String())})
		table.Append([]string{yellow("Details:"), violation.Details})

		documentation := violation.DocumentationURL()
		if documentation != "" {
			table.Append([]string{yellow("Documentation:"), cyan(violation.DocumentationURL())})
		}

		if i+1 < numViolations {
			table.Append([]string{""})
		}
	}

	end := ""
	if numViolations > 0 {
		end = "\n"
	}

	table.Render()

	return buffer.String() + end
}

func pluralize(singular string, count int) string {
	if count == 1 {
		return singular
	}

	return singular + "s"
}
