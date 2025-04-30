package remote

import (
	"context"
	"fmt"

	"github.com/grafana/grafana-app-sdk/logging"
	"github.com/grafana/grafanactl/internal/config"
	"github.com/grafana/grafanactl/internal/logs"
	"github.com/grafana/grafanactl/internal/resources"
	"github.com/grafana/grafanactl/internal/resources/discovery"
	"github.com/grafana/grafanactl/internal/resources/dynamic"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// Deleter takes care of deleting resources from Grafana.
type Deleter struct {
	client   *dynamic.NamespacedClient
	registry PushRegistry
}

// NewDeleter creates a new Deleter.
func NewDeleter(ctx context.Context, cfg config.NamespacedRESTConfig) (*Deleter, error) {
	cli, err := dynamic.NewDefaultNamespacedClient(cfg)
	if err != nil {
		return nil, err
	}

	registry, err := discovery.NewDefaultRegistry(ctx, cfg)
	if err != nil {
		return nil, err
	}

	return &Deleter{
		client:   cli,
		registry: registry,
	}, nil
}

// DeleteRequest is a request for deleting resources from Grafana.
type DeleteRequest struct {
	// A list of resources to delete.
	Resources *resources.Resources

	// The maximum number of concurrent pushes.
	MaxConcurrency int

	// Whether the operation should stop upon encountering an error.
	StopOnError bool

	// If set to true, the deleter will simulate the delete operations.
	DryRun bool
}

type DeleteSummary struct {
	DeletedCount int
	FailedCount  int
}

func (deleter *Deleter) Delete(ctx context.Context, request DeleteRequest) (DeleteSummary, error) {
	summary := DeleteSummary{}
	supported := deleter.supportedDescriptors()

	if request.MaxConcurrency < 1 {
		request.MaxConcurrency = 1
	}

	err := request.Resources.ForEachConcurrently(ctx, request.MaxConcurrency,
		func(ctx context.Context, res *resources.Resource) error {
			name := res.Name()
			gvk := res.GroupVersionKind()

			logger := logging.FromContext(ctx).With(
				"gvk", gvk,
				"name", name,
			)

			if _, ok := supported[gvk]; !ok {
				if request.StopOnError {
					return fmt.Errorf("resource not supported by the API: %s/%s", gvk, name)
				}

				logger.Warn("Skipping resource not supported by the API")
				return nil
			}

			desc, ok := supported[gvk]
			if !ok {
				if request.StopOnError {
					return fmt.Errorf("resource not supported by the API: %s/%s", gvk, name)
				}

				logger.Warn("Skipping resource not supported by the API")
				return nil
			}

			if err := deleter.deleteResource(ctx, desc, res, request.DryRun); err != nil {
				summary.FailedCount++
				if request.StopOnError {
					return err
				}

				logger.Warn("Failed to delete resource", logs.Err(err))
				return nil
			}

			summary.DeletedCount++
			logger.Info("Resource deleted")
			return nil
		},
	)
	if err != nil {
		return summary, err
	}

	return summary, nil
}

func (deleter *Deleter) deleteResource(ctx context.Context, descriptor resources.Descriptor, res *resources.Resource, dryRun bool) error {
	var dryRunOpts []string
	if dryRun {
		dryRunOpts = []string{"All"}
	}

	return deleter.client.Delete(ctx, descriptor, res.Name(), metav1.DeleteOptions{
		DryRun: dryRunOpts,
	})
}

func (deleter *Deleter) supportedDescriptors() map[schema.GroupVersionKind]resources.Descriptor {
	supported := deleter.registry.SupportedResources()

	supportedDescriptors := make(map[schema.GroupVersionKind]resources.Descriptor)
	for _, sup := range supported {
		supportedDescriptors[sup.GroupVersionKind()] = sup
	}

	return supportedDescriptors
}
