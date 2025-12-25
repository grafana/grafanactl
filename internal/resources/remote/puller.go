package remote

import (
	"context"
	"log/slog"

	"github.com/grafana/grafana-app-sdk/logging"
	"github.com/grafana/grafanactl/internal/config"
	"github.com/grafana/grafanactl/internal/logs"
	"github.com/grafana/grafanactl/internal/resources"
	"github.com/grafana/grafanactl/internal/resources/discovery"
	"github.com/grafana/grafanactl/internal/resources/dynamic"
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
		ctx context.Context, desc resources.Descriptor, names []string, opts metav1.GetOptions,
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
	client, err := dynamic.NewDefaultVersionedClient(restConfig)
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

	// Processors to apply to resources after they are pulled.
	Processors []Processor

	// Destination list for the pulled resources.
	Resources *resources.Resources

	// Whether to include resources managed by other tools.
	ExcludeManaged bool

	// Whether the operation should stop upon encountering an error.
	StopOnError bool
}

// Pull pulls resources from Grafana.
func (p *Puller) Pull(ctx context.Context, req PullRequest) error {
	filters := req.Filters

	// If no filters are provided, we need to pull all available resources.
	if filters.IsEmpty() {
		// When pulling all resources, we need to use preferred versions.
		preferred := p.registry.PreferredResources()

		filters = make(resources.Filters, 0, len(preferred))
		for _, r := range preferred {
			filters = append(filters, resources.Filter{
				Type:       resources.FilterTypeAll,
				Descriptor: r,
			})
		}
	}

	logger := logging.FromContext(ctx)
	logger.Debug("Pulling resources")

	errg, ctx := errgroup.WithContext(ctx)
	partialRes := make([][]unstructured.Unstructured, len(filters))

	for idx, filt := range filters {
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
				res, err := p.client.GetMultiple(ctx, filt.Descriptor, filt.ResourceUIDs, metav1.GetOptions{})
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

	req.Resources.Clear()
	for _, r := range partialRes {
		for _, item := range r {
			res, err := resources.FromUnstructured(&item)
			if err != nil {
				return err
			}

			// TODO: this should be replaced by a more generic mechanism,
			// e.g. label & annotation filters.
			if !res.IsManaged() && req.ExcludeManaged {
				continue
			}

			if err := p.process(res, req.Processors); err != nil {
				if req.StopOnError {
					return err
				}

				logger.Warn("Failed to process resource", logs.Err(err))
			} else {
				req.Resources.Add(res)
			}
		}
	}

	return nil
}

func (p *Puller) process(res *resources.Resource, processors []Processor) error {
	for _, processor := range processors {
		if err := processor.Process(res); err != nil {
			return err
		}
	}

	return nil
}
