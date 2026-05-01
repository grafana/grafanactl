package alerts

import (
	"errors"
	"fmt"
	"strings"

	"github.com/grafana/grafana-openapi-client-go/models"
	cmdconfig "github.com/grafana/grafanactl/cmd/grafanactl/config"
	cmdio "github.com/grafana/grafanactl/cmd/grafanactl/io"
	"github.com/grafana/grafanactl/internal/grafana"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

type searchOpts struct {
	IO   cmdio.Options
	Name string
}

func (opts *searchOpts) setup(flags *pflag.FlagSet) {
	flags.StringVar(&opts.Name, "name", "", "Search pattern to match against alert rule titles (case-insensitive substring match)")
	opts.IO.RegisterCustomCodec("text", &alertTableCodec{})
	opts.IO.DefaultFormat("text")
	opts.IO.BindFlags(flags)
}

func (opts *searchOpts) Validate() error {
	if opts.Name == "" {
		return errors.New("--name flag is required")
	}

	return opts.IO.Validate()
}

func searchCmd(configOpts *cmdconfig.Options) *cobra.Command {
	opts := &searchOpts{}

	cmd := &cobra.Command{
		Use:   "search",
		Args:  cobra.NoArgs,
		Short: "Search alert rules by name",
		Long:  "Search alert rules from Grafana by matching against their titles using a case-insensitive substring match.",
		Example: `
	grafanactl alerts search --name "cpu"
	grafanactl alerts search --name "disk" -o json`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if err := opts.Validate(); err != nil {
				return err
			}

			cfg, err := configOpts.LoadConfig(cmd.Context())
			if err != nil {
				return err
			}

			currentCtx := cfg.GetCurrentContext()

			gClient, err := grafana.ClientFromContext(currentCtx)
			if err != nil {
				return err
			}

			resp, err := gClient.Provisioning.GetAlertRules()
			if err != nil {
				return fmt.Errorf("failed to list alert rules: %w", err)
			}

			pattern := strings.ToLower(opts.Name)
			var matched models.ProvisionedAlertRules

			for _, rule := range resp.Payload {
				title := strings.ToLower(derefStr(rule.Title))
				if strings.Contains(title, pattern) {
					matched = append(matched, rule)
				}
			}

			codec, err := opts.IO.Codec()
			if err != nil {
				return err
			}

			if opts.IO.OutputFormat != "text" {
				return codec.Encode(cmd.OutOrStdout(), matched)
			}

			if err := codec.Encode(cmd.OutOrStdout(), matched); err != nil {
				return err
			}

			fmt.Fprintf(cmd.OutOrStdout(), "\nTotal: %d alert rule(s)\n", len(matched))

			return nil
		},
	}

	opts.setup(cmd.Flags())

	return cmd
}
