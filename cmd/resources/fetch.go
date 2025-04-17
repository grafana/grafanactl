package resources

import (
	"context"

	"github.com/grafana/grafanactl/cmd/fail"
	"github.com/grafana/grafanactl/internal/config"
	"github.com/grafana/grafanactl/internal/resources"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

type fetchRequest struct {
	Config             config.NamespacedRESTConfig
	StopOnError        bool
	ExpectSingleTarget bool
}

type fetchResponse struct {
	Resources      unstructured.UnstructuredList
	IsSingleTarget bool
}

func fetchResources(ctx context.Context, opts fetchRequest, args []string) (*fetchResponse, error) {
	sels, err := resources.ParseSelectors(args)
	if err != nil {
		return nil, err
	}

	pull, err := resources.NewPuller(ctx, opts.Config)
	if err != nil {
		return nil, err
	}

	if opts.ExpectSingleTarget && !sels.IsSingleTarget() {
		return nil, fail.DetailedError{
			Summary: "Invalid resource selector",
			Details: "Expected a resource selector targeting a single resource. Example: dashboard/some-dashboard",
		}
	}

	res := fetchResponse{
		IsSingleTarget: sels.IsSingleTarget(),
	}

	req := resources.PullRequest{
		Selectors:   sels,
		StopOnError: opts.StopOnError || sels.IsSingleTarget(),
		Resources:   &res.Resources,
	}

	if err := pull.Pull(ctx, req); err != nil {
		return nil, err
	}

	return &res, nil
}
