package linter

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	cmdio "github.com/grafana/grafanactl/cmd/grafanactl/io"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

var (
	resourceTypeRegex = regexp.MustCompile(`^[a-z]+$`)
	nameRegex         = regexp.MustCompile(`^[a-z_]+[a-z0-9_\-]*$`)
)

type newRuleOpts struct {
	output string
}

func (opts *newRuleOpts) setup(flags *pflag.FlagSet) {
	flags.StringVarP(&opts.output, "output", "o", "", "Output directory")
}

func (opts *newRuleOpts) Validate(args []string) error {
	if args[0] == "" {
		return errors.New("resource-type is required for rule")
	}

	if !resourceTypeRegex.MatchString(args[0]) {
		return errors.New("resource-type must be a single word, using lowercase letters only")
	}

	if args[1] == "" {
		return errors.New("name is required for rule")
	}

	if !nameRegex.MatchString(args[1]) {
		return errors.New("name must consist only of lowercase letters, numbers, underscores and dashes")
	}

	return nil
}

func newCmd() *cobra.Command {
	opts := newRuleOpts{}

	cmd := &cobra.Command{
		Use:   "new resource-type name",
		Short: "Creates a new resource linter",
		Long:  "Creates a new resource linter.",
		Args:  cobra.ExactArgs(2),
		Example: `
	# Creates a new dashboard linter in the current directory:

	grafanactl experimental linter new dashboard test-linter

	# Creates a new dashboard linter in another directory:

	grafanactl experimental linter new dashboard test-linter -o custom-rules
`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := opts.Validate(args); err != nil {
				return err
			}

			return scaffoldCustomRule(cmd.OutOrStdout(), opts, args[0], args[1])
		},
	}

	opts.setup(cmd.Flags())

	return cmd
}

func scaffoldCustomRule(stdout io.Writer, opts newRuleOpts, resourceType string, name string) error {
	ruleDir := filepath.Join(
		opts.output, "rules", "custom", "grafanactl", "rules", resourceType, name,
	)

	if err := os.MkdirAll(ruleDir, 0o770); err != nil {
		return err
	}

	ruleFileContent := strings.ReplaceAll(customRuleTemplate, "{{.ResourceType}}", resourceType)
	ruleFileContent = strings.ReplaceAll(ruleFileContent, "{{.Name}}", name)

	ruleFileName := strings.ToLower(strings.ReplaceAll(name, "-", "_")) + ".rego"

	if err := os.WriteFile(filepath.Join(ruleDir, ruleFileName), []byte(ruleFileContent), 0o600); err != nil {
		return err
	}

	cmdio.Success(stdout, "Rule written in %s", ruleDir)

	return nil
}

const customRuleTemplate = `# METADATA
# description: Describe the rule here.
# custom:
#  severity: warning
package custom.grafanactl.rules.{{.ResourceType}}["{{.Name}}"]

import data.grafanactl.result
import data.grafanactl.utils

# Dashboard v1
report contains violation if {
	utils.resource_is_dashboard_v1(input)

	input.spec.timezone != "utc"

	violation := result.fail(rego.metadata.chain(), "details")
}

# Dashboard v2
report contains violation if {
	utils.resource_is_dashboard_v2(input)

	input.spec.timeSettings.timezone != "utc"

	violation := result.fail(rego.metadata.chain(), "details")
}
`
