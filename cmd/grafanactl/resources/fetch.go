package resources

import (
	"context"

	"github.com/grafana/grafanactl/cmd/grafanactl/fail"
	"github.com/grafana/grafanactl/internal/config"
	"github.com/grafana/grafanactl/internal/resources"
	"github.com/grafana/grafanactl/internal/resources/discovery"
	"github.com/grafana/grafanactl/internal/resources/remote"
)

type fetchRequest struct {
	Config             config.NamespacedRESTConfig
	StopOnError        bool
	ExcludeManaged     bool
	ExpectSingleTarget bool
	Processors         []remote.Processor
}

type fetchResponse struct {
	Resources      resources.Resources
	IsSingleTarget bool
}

func fetchResources(ctx context.Context, opts fetchRequest, args []string) (*fetchResponse, error) {
	sels, err := resources.ParseSelectors(args)
	if err != nil {
		return nil, err
	}

	if opts.ExpectSingleTarget && !sels.IsSingleTarget() {
		return nil, fail.DetailedError{
			Summary: "Invalid resource selector",
			Details: "Expected a resource selector targeting a single resource. Example: dashboard/some-dashboard",
		}
	}

	reg, err := discovery.NewDefaultRegistry(ctx, opts.Config)
	if err != nil {
		return nil, err
	}

	filters, err := reg.MakeFilters(discovery.MakeFiltersOptions{
		Selectors:            sels,
		PreferredVersionOnly: true,
	})
	if err != nil {
		return nil, err
	}

	pull, err := remote.NewDefaultPuller(ctx, opts.Config)
	if err != nil {
		return nil, err
	}

	res := fetchResponse{
		IsSingleTarget: sels.IsSingleTarget(),
	}

	req := remote.PullRequest{
		Filters:        filters,
		Resources:      &res.Resources,
		Processors:     opts.Processors,
		ExcludeManaged: opts.ExcludeManaged,
		StopOnError:    opts.StopOnError || sels.IsSingleTarget(),
	}

	if err := pull.Pull(ctx, req); err != nil {
		return nil, err
	}

	return &res, nil
}
