package resources

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os/exec"
	"runtime"

	"github.com/grafana/grafana-app-sdk/logging"
	cmdconfig "github.com/grafana/grafanactl/cmd/config"
	cmdio "github.com/grafana/grafanactl/cmd/io"
	"github.com/grafana/grafanactl/internal/logs"
	"github.com/grafana/grafanactl/internal/resources"
	"github.com/grafana/grafanactl/internal/server"
	"github.com/grafana/grafanactl/internal/server/livereload"
	serverresources "github.com/grafana/grafanactl/internal/server/resources"
	"github.com/grafana/grafanactl/internal/server/watch"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

type serveOpts struct {
	Address      string
	Port         int
	WatchPaths   []string
	Script       string
	ScriptFormat string
}

func (opts *serveOpts) setup(flags *pflag.FlagSet) {
	flags.StringVar(&opts.Address, "address", "0.0.0.0", "Address to bind")
	flags.IntVar(&opts.Port, "port", 8080, "Port on which the server will listen")
	flags.StringArrayVarP(&opts.WatchPaths, "watch", "w", nil, "Paths to watch for changes")
	flags.StringVarP(&opts.Script, "script", "S", "", "Script")
	flags.StringVarP(&opts.ScriptFormat, "script-format", "f", "yaml", "Format of the data returned by the script")
}

func (opts *serveOpts) Validate(args []string) error {
	if len(args) == 0 && opts.Script == "" {
		// TODO: better error msg
		return errors.New("must specify path to resources or script")
	}

	return nil
}

func serveCmd(configOpts *cmdconfig.Options) *cobra.Command {
	opts := &serveOpts{}

	cmd := &cobra.Command{
		Use:   "serve",
		Args:  cobra.MaximumNArgs(1), // TODO: arbitrary list of paths to resources
		Short: "Serve Grafana resources locally",
		Long: `Serve Grafana resources locally.

Note on NFS/SMB support for watch:

fsnotify requires support from underlying OS to work. The current NFS and SMB protocols does not provide network level support for file notifications.

TODO: more information.
`,
		Example: fmt.Sprintf(`
TODO
  %[1]s resources serve
`, binaryName),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := configOpts.LoadConfig(cmd.Context())
			if err != nil {
				return err
			}

			logger := logging.FromContext(cmd.Context())
			parsedResources := resources.NewResources()
			parser := serverresources.DefaultParser(cmd.Context(), serverresources.ParserConfig{
				ContinueOnError: true,
			})

			if len(args) != 0 {
				if err := parser.ParseInto(parsedResources, args[0]); err != nil {
					return err
				}
			}

			parseFromScript := func() error {
				output, err := executeWatchScript(cmd.Context(), opts.Script)
				if err != nil {
					return err
				}

				if err = parser.ParseBytesInto(parsedResources, output, opts.ScriptFormat); err != nil {
					logger.Warn("Could not parse script output", logs.Err(err))
					return err
				}

				return nil
			}

			if opts.Script != "" {
				if err := parseFromScript(); err != nil {
					return err
				}
			}

			if len(opts.WatchPaths) > 0 {
				livereload.Initialize()

				parsedResources.OnChange(func(resource *resources.Resource) {
					logger.Debug("Resource changed in memory", slog.String("resource", string(resource.Ref())))
					livereload.ReloadResource(resource)
				})

				// By default, react to changes by parsing changed files
				onInputChange := func(file string) {
					if err = parser.ParseInto(parsedResources, file); err != nil {
						logger.Warn("Could not parse file", slog.String("file", file), logs.Err(err))
					}
				}

				// If a script is given, run the script on change
				if opts.Script != "" {
					onInputChange = func(_ string) {
						_ = parseFromScript()
					}
				}

				watcher, err := watch.NewWatcher(cmd.Context(), onInputChange)
				if err != nil {
					return err
				}

				if err := watcher.Add(opts.WatchPaths...); err != nil {
					return err
				}

				// Start listening for events.
				go watcher.Watch()
			}

			serverCfg := server.Config{
				ListenAddr: opts.Address,
				Port:       opts.Port,
				NoColor:    cmd.Flags().Lookup("no-color").Value.String() == "true",
			}
			resourceServer := server.New(serverCfg, cfg.GetCurrentContext(), parsedResources)

			logger.Debug(fmt.Sprintf("Listening on %s:%d", opts.Address, opts.Port))
			cmdio.Info(cmd.OutOrStdout(), "Server will be available on http://localhost:%d/", opts.Port)

			return resourceServer.Start(cmd.Context())
		},
	}

	opts.setup(cmd.Flags())

	return cmd
}

func executeWatchScript(ctx context.Context, command string) ([]byte, error) {
	logger := logging.FromContext(ctx).With(slog.String("component", "script"), slog.String("command", command))

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	logger.Debug("executing script")

	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		// TODO: test on windows
		cmd = exec.Command("cmd", "/c", command)
	} else {
		cmd = exec.Command("sh", "-c", command)
	}

	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	if stderr.Len() > 0 {
		logger.Error("script stderr", slog.String("output", stderr.String()))
	}
	if err != nil {
		return nil, fmt.Errorf("script failed: %w", err)
	}

	return stdout.Bytes(), nil
}
