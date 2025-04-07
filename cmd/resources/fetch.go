package resources

import (
	"context"

	"github.com/grafana/grafanactl/internal/config"
	"github.com/grafana/grafanactl/internal/fail"
	"github.com/grafana/grafanactl/internal/resources"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

type fetchRequest struct {
	Config      config.NamespacedRESTConfig
	StopOnError bool
}

type fetchResponse struct {
	Resources      unstructured.UnstructuredList
	IsSingleTarget bool
}

func fetchResources(ctx context.Context, opts fetchRequest, args []string) (*fetchResponse, error) {
	sels, err := resources.ParseSelectors(args)
	if err != nil {
		return nil, parseSelectorErr(err)
	}

	pull, err := resources.NewPuller(ctx, opts.Config)
	if err != nil {
		return nil, clientInitErr(err)
	}

	res := fetchResponse{
		IsSingleTarget: sels.IsSingleTarget(),
	}

	req := resources.PullRequest{
		Selectors:   sels,
		StopOnError: opts.StopOnError,
		Resources:   &res.Resources,
	}

	if err := pull.Pull(ctx, req); err != nil {
		return nil, fail.DetailedError{
			Parent:  err,
			Summary: "Could not pull resource(s) from the API",
			Details: "One or more resources could not be pulled from the API",
			Suggestions: []string{
				"Make sure that your are passing in valid resource paths",
			},
		}
	}

	return &res, nil
}
