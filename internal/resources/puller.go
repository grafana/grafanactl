package resources

import (
	"context"
	"log/slog"

	"github.com/grafana/grafana-app-sdk/logging"
	"github.com/grafana/grafanactl/internal/config"
	"github.com/grafana/grafanactl/internal/logs"
	"golang.org/x/sync/errgroup"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// Puller is a command that pulls resources from Grafana.
type Puller struct {
	client *NamespacedDynamicClient
}

// NewPuller creates a new Puller.
func NewPuller(ctx context.Context, restConfig config.NamespacedRESTConfig) (*Puller, error) {
	client, err := NewDefaultNamespacedDynamicClient(ctx, restConfig)
	if err != nil {
		return nil, err
	}

	return &Puller{
		client: client,
	}, nil
}

// PullRequest is a request for pulling resources from Grafana.
type PullRequest struct {
	// Which resources to pull.
	Selectors []Selector

	// Whether the operation should stop upon encountering an error.
	StopOnError bool

	// Destination list for the pulled resources.
	Resources *unstructured.UnstructuredList
}

// Pull pulls resources from Grafana.
func (p *Puller) Pull(ctx context.Context, req PullRequest) error {
	// hack: we need to refactor this better, since there's a bunch of code duplication.
	// but we want to expose a nice minimalistic API to the user.
	if len(req.Selectors) == 0 {
		return p.pullAll(ctx, req)
	}

	logger := logging.FromContext(ctx)
	logger.Debug("Pulling resources")

	errg, ctx := errgroup.WithContext(ctx)
	partialRes := make([][]unstructured.Unstructured, len(req.Selectors))

	for idx, sel := range req.Selectors {
		errg.Go(func() error {
			switch sel.SelectorType {
			case SelectorTypeAll:
				res, err := p.client.List(ctx, sel.GroupVersionKind, metav1.ListOptions{})
				if err != nil {
					if req.StopOnError {
						return err
					}
					logger.Warn("Could not pull resources", logs.Err(err), slog.String("cmd", sel.String()))
				} else {
					partialRes[idx] = res.Items
				}
			case SelectorTypeMultiple:
				res, err := p.client.GetMultiple(ctx, sel.GroupVersionKind, sel.ResourceUIDs, metav1.ListOptions{})
				if err != nil {
					if req.StopOnError {
						return err
					}
					logger.Warn("Could not pull resources", logs.Err(err), slog.String("cmd", sel.String()))
				} else {
					partialRes[idx] = res
				}
			case SelectorTypeSingle:
				res, err := p.client.Get(ctx, sel.GroupVersionKind, sel.ResourceUIDs[0], metav1.GetOptions{})
				if err != nil {
					if req.StopOnError {
						return err
					}
					logger.Warn("Could not pull resource", logs.Err(err), slog.String("cmd", sel.String()))
				} else {
					partialRes[idx] = []unstructured.Unstructured{*res}
				}
			}
			return nil
		})
	}

	if err := errg.Wait(); err != nil {
		return err
	}

	req.Resources.Items = make([]unstructured.Unstructured, 0, len(partialRes))
	for _, r := range partialRes {
		req.Resources.Items = append(req.Resources.Items, r...)
	}

	return nil
}

func (p *Puller) pullAll(ctx context.Context, req PullRequest) error {
	logger := logging.FromContext(ctx)
	logger.Debug("Pulling all resources")

	resources, err := p.client.Resources(ctx)
	if err != nil {
		return err
	}

	errg, ctx := errgroup.WithContext(ctx)
	cmdRes := make([][]unstructured.Unstructured, len(resources))

	for i, r := range resources {
		errg.Go(func() error {
			res, err := p.client.List(ctx, r, metav1.ListOptions{})
			if err == nil {
				cmdRes[i] = res.Items
			}

			if err != nil {
				if req.StopOnError {
					return err
				}
				logger.Warn("Could not pull resources", logs.Err(err), slog.String("kind", r.String()))
			}

			return nil
		})
	}

	if err := errg.Wait(); err != nil {
		return err
	}

	req.Resources.Items = make([]unstructured.Unstructured, 0, len(cmdRes))
	for _, r := range cmdRes {
		req.Resources.Items = append(req.Resources.Items, r...)
	}

	return nil
}
