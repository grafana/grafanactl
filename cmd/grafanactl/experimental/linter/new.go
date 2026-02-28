package linter

import (
	"errors"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

var (
	resourceTypeRegex = regexp.MustCompile(`^[a-z]+$`)
	nameRegex         = regexp.MustCompile(`^[a-z_]+[a-z0-9_\-]*$`)
)

type newRuleOpts struct {
	resourceType string
	name         string
	output       string
}

func (opts *newRuleOpts) setup(flags *pflag.FlagSet) {
	flags.StringVarP(&opts.resourceType, "resource-type", "t", "", "Resource type targeted by the rule.")
	flags.StringVarP(&opts.name, "name", "n", "", "Rule name.")
	flags.StringVarP(&opts.output, "output", "o", "", "Output directory")
}

func (opts *newRuleOpts) Validate() error {
	if opts.resourceType == "" {
		return errors.New("resource-type is required for rule")
	}

	if !resourceTypeRegex.MatchString(opts.resourceType) {
		return errors.New("resource-type must be a single word, using lowercase letters only")
	}

	if opts.name == "" {
		return errors.New("name is required for rule")
	}

	if !nameRegex.MatchString(opts.name) {
		return errors.New("name must consist only of lowercase letters, numbers, underscores and dashes")
	}

	return nil
}

func newCmd() *cobra.Command {
	opts := newRuleOpts{}

	cmd := &cobra.Command{
		Use: "new [-t resource-type] [-n name]",
		RunE: func(_ *cobra.Command, _ []string) error {
			if err := opts.Validate(); err != nil {
				return err
			}

			if opts.output == "" {
				workingDir, err := os.Getwd()
				if err != nil {
					opts.output = workingDir
				}
			}

			return scaffoldCustomRule(opts)
		},
	}

	opts.setup(cmd.Flags())

	return cmd
}

func scaffoldCustomRule(opts newRuleOpts) error {
	rulesDir := filepath.Join(
		opts.output, "rules", "custom", "grafanactl", "rules", opts.resourceType, opts.name,
	)

	if err := os.MkdirAll(rulesDir, 0o770); err != nil {
		return err
	}

	ruleFileContent := strings.ReplaceAll(customRuleTemplate, "{{.ResourceType}}", opts.resourceType)
	ruleFileContent = strings.ReplaceAll(ruleFileContent, "{{.Name}}", opts.name)

	ruleFileName := strings.ToLower(strings.ReplaceAll(opts.name, "-", "_")) + ".rego"

	if err := os.WriteFile(filepath.Join(rulesDir, ruleFileName), []byte(ruleFileContent), 0o600); err != nil {
		return err
	}

	return nil
}

const customRuleTemplate = `# METADATA
# description: Describe the rule here.
# custom:
#  severity: warning
package custom.grafanactl.rules.{{.ResourceType}}["{{.Name}}"]

import data.grafanactl.result
import data.grafanactl.utils

report contains violation if {
	utils.resource_is_dashboard_v1alpha1(input)

	input.spec.editable != false

	violation := result.fail(rego.metadata.chain(), "details")
}
`
