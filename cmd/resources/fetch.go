package resources

import (
	"context"
	"errors"
	"fmt"

	"github.com/grafana/grafanactl/internal/config"
	"github.com/grafana/grafanactl/internal/fail"
	"github.com/grafana/grafanactl/internal/resources"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

type fetchOpts struct {
	Config      config.Config
	StopOnError bool
}

type fetchResult struct {
	Resources      unstructured.UnstructuredList
	IsSingleTarget bool
}

func fetchResources(
	ctx context.Context, opts fetchOpts, args []string,
) (*fetchResult, error) {
	// Looks like contextcheck is being confused here.
	// Probably thinks that `GetCurrentContext()` related to `context.Context`.
	//nolint:contextcheck
	pull, err := resources.NewPuller(*opts.Config.GetCurrentContext())
	if err != nil {
		// TODO: is this error actually related to what `resources.NewPuller()` does?
		return nil, fail.DetailedError{
			Parent:  err,
			Summary: "Could not parse pull command(s)",
			Details: "One or more of the provided resource paths are invalid",
			Suggestions: []string{
				"Make sure that your are passing in valid resource paths",
			},
		}
	}

	var (
		res              *unstructured.UnstructuredList
		singlePullTarget bool
		perr             error
	)
	if len(args) == 0 {
		res, perr = pull.PullAll(ctx)
	} else {
		invalidCommandErr := &resources.InvalidCommandError{}
		cmds, err := resources.ParsePullCommands(args)
		if err != nil && errors.As(err, invalidCommandErr) {
			return nil, fail.DetailedError{
				Parent:  err,
				Summary: "Could not parse pull command(s)",
				Details: fmt.Sprintf("Failed to parse command '%s'", invalidCommandErr.Command),
				Suggestions: []string{
					"Make sure that your are passing in valid resource paths",
				},
			}
		} else if err != nil {
			return nil, err
		}

		singlePullTarget = cmds.HasSingleTarget()
		res, perr = pull.Pull(ctx, resources.PullerRequest{
			Commands:    cmds,
			StopOnError: opts.StopOnError,
		})
	}

	if perr != nil {
		return nil, fail.DetailedError{
			Parent:  perr,
			Summary: "Could not pull resource(s) from the API",
			Details: "One or more resources could not be pulled from the API",
			Suggestions: []string{
				"Make sure that your are passing in valid resource paths",
			},
		}
	}

	return &fetchResult{
		Resources:      *res,
		IsSingleTarget: singlePullTarget,
	}, nil
}
