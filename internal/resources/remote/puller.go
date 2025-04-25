package remote

import (
	"context"
	"log/slog"

	"github.com/grafana/grafana-app-sdk/logging"
	"github.com/grafana/grafanactl/internal/config"
	"github.com/grafana/grafanactl/internal/logs"
	"github.com/grafana/grafanactl/internal/resources"
	"github.com/grafana/grafanactl/internal/resources/client"
	"github.com/grafana/grafanactl/internal/resources/discovery"
	"golang.org/x/sync/errgroup"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// PullClient is a client that can pull resources from Grafana.
type PullClient interface {
	Get(
		ctx context.Context, desc resources.Descriptor, name string, opts metav1.GetOptions,
	) (*unstructured.Unstructured, error)

	GetMultiple(
		ctx context.Context, desc resources.Descriptor, names []string, opts metav1.ListOptions,
	) ([]unstructured.Unstructured, error)

	List(
		ctx context.Context, desc resources.Descriptor, opts metav1.ListOptions,
	) (*unstructured.UnstructuredList, error)
}

// PullRegistry is a registry of resources that can be pulled from Grafana.
type PullRegistry interface {
	PreferredResources() resources.Descriptors
}

// Puller is a command that pulls resources from Grafana.
type Puller struct {
	client   PullClient
	registry PullRegistry
}

// NewDefaultPuller creates a new Puller.
func NewDefaultPuller(ctx context.Context, restConfig config.NamespacedRESTConfig) (*Puller, error) {
	client, err := client.NewDefaultNamespacedDynamicClient(restConfig)
	if err != nil {
		return nil, err
	}

	registry, err := discovery.NewDefaultRegistry(ctx, restConfig)
	if err != nil {
		return nil, err
	}

	return NewPuller(client, registry), nil
}

// NewPuller creates a new Puller.
func NewPuller(client PullClient, registry PullRegistry) *Puller {
	return &Puller{
		client:   client,
		registry: registry,
	}
}

// PullRequest is a request for pulling resources from Grafana.
type PullRequest struct {
	// Which resources to pull.
	Filters resources.Filters

	// Whether the operation should stop upon encountering an error.
	StopOnError bool

	// Destination list for the pulled resources.
	Resources *unstructured.UnstructuredList
}

// Pull pulls resources from Grafana.
func (p *Puller) Pull(ctx context.Context, req PullRequest) error {
	// hack: we need to refactor this better, since there's a bunch of code duplication.
	// but we want to expose a nice minimalistic API to the user.
	if req.Filters.IsEmpty() {
		return p.pullAll(ctx, req)
	}

	logger := logging.FromContext(ctx)
	logger.Debug("Pulling resources")

	errg, ctx := errgroup.WithContext(ctx)
	partialRes := make([][]unstructured.Unstructured, len(req.Filters))

	for idx, filt := range req.Filters {
		errg.Go(func() error {
			switch filt.Type {
			case resources.FilterTypeAll:
				res, err := p.client.List(ctx, filt.Descriptor, metav1.ListOptions{})
				if err != nil {
					if req.StopOnError {
						return err
					}
					logger.Warn("Could not pull resources", logs.Err(err), slog.String("cmd", filt.String()))
				} else {
					partialRes[idx] = res.Items
				}
			case resources.FilterTypeMultiple:
				res, err := p.client.GetMultiple(ctx, filt.Descriptor, filt.ResourceUIDs, metav1.ListOptions{})
				if err != nil {
					if req.StopOnError {
						return err
					}
					logger.Warn("Could not pull resources", logs.Err(err), slog.String("cmd", filt.String()))
				} else {
					partialRes[idx] = res
				}
			case resources.FilterTypeSingle:
				res, err := p.client.Get(ctx, filt.Descriptor, filt.ResourceUIDs[0], metav1.GetOptions{})
				if err != nil {
					if req.StopOnError {
						return err
					}
					logger.Warn("Could not pull resource", logs.Err(err), slog.String("cmd", filt.String()))
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

	resources := p.registry.PreferredResources()

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
