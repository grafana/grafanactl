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

// DeleteClient is a client that can delete resources from Grafana.
type DeleteClient interface {
	Delete(
		ctx context.Context, desc resources.Descriptor, name string, opts metav1.DeleteOptions,
	) error
}

// Deleter takes care of deleting resources from Grafana.
type Deleter struct {
	client   DeleteClient
	registry Registry
}

// NewDefaultDeleter creates a new Deleter.
// It uses the default namespaced dynamic client to delete resources from Grafana.
func NewDefaultDeleter(ctx context.Context, cfg config.NamespacedRESTConfig) (*Deleter, error) {
	cli, err := dynamic.NewDefaultNamespacedClient(cfg)
	if err != nil {
		return nil, err
	}

	registry, err := discovery.NewDefaultRegistry(ctx, cfg)
	if err != nil {
		return nil, err
	}

	return NewDeleter(cli, registry), nil
}

// NewDeleter creates a new Deleter.
func NewDeleter(client DeleteClient, registry Registry) *Deleter {
	return &Deleter{
		client:   client,
		registry: registry,
	}
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
	// When dry-run is enabled, skip the actual delete operation.
	// This is a client-side dry-run because the k8s.io/client-go dynamic client
	// sends DeleteOptions in the HTTP body (not as query parameters), which causes
	// the Kubernetes API server to ignore the DryRun field. By skipping the API call
	// entirely, we ensure no resources are accidentally deleted in dry-run mode.
	if dryRun {
		return nil
	}

	return deleter.client.Delete(ctx, descriptor, res.Name(), metav1.DeleteOptions{})
}

func (deleter *Deleter) supportedDescriptors() map[schema.GroupVersionKind]resources.Descriptor {
	supported := deleter.registry.SupportedResources()

	supportedDescriptors := make(map[schema.GroupVersionKind]resources.Descriptor)
	for _, sup := range supported {
		supportedDescriptors[sup.GroupVersionKind()] = sup
	}

	return supportedDescriptors
}
