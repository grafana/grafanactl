package alerts

import (
	"fmt"

	cmdconfig "github.com/grafana/grafanactl/cmd/grafanactl/config"
	cmdio "github.com/grafana/grafanactl/cmd/grafanactl/io"
	"github.com/grafana/grafanactl/internal/grafana"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

type getOpts struct {
	IO cmdio.Options
}

func (opts *getOpts) setup(flags *pflag.FlagSet) {
	opts.IO.BindFlags(flags)
}

func (opts *getOpts) Validate() error {
	return opts.IO.Validate()
}

func getCmd(configOpts *cmdconfig.Options) *cobra.Command {
	opts := &getOpts{}

	cmd := &cobra.Command{
		Use:   "get UID",
		Args:  cobra.ExactArgs(1),
		Short: "Get a single alert rule by UID",
		Long:  "Get a single alert rule from Grafana by its UID via the Provisioning API.",
		Example: `
	grafanactl alerts get my-alert-uid
	grafanactl alerts get my-alert-uid -o yaml`,
		RunE: func(cmd *cobra.Command, args []string) error {
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

			resp, err := gClient.Provisioning.GetAlertRule(args[0])
			if err != nil {
				return fmt.Errorf("failed to get alert rule %q: %w", args[0], err)
			}

			codec, err := opts.IO.Codec()
			if err != nil {
				return err
			}

			return codec.Encode(cmd.OutOrStdout(), resp.Payload)
		},
	}

	opts.setup(cmd.Flags())

	return cmd
}
