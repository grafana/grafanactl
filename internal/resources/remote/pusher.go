package remote

import (
	"context"
	"fmt"
	"sync"

	"github.com/grafana/grafana-app-sdk/logging"
	"github.com/grafana/grafanactl/internal/config"
	"github.com/grafana/grafanactl/internal/logs"
	"github.com/grafana/grafanactl/internal/resources"
	"github.com/grafana/grafanactl/internal/resources/discovery"
	"github.com/grafana/grafanactl/internal/resources/dynamic"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// PushRegistry is a registry of resources that can be pushed to Grafana.
type PushRegistry interface {
	SupportedResources() resources.Descriptors
}

// PushClient is a client that can push resources to Grafana.
type PushClient interface {
	Create(
		ctx context.Context, desc resources.Descriptor, obj *unstructured.Unstructured, opts metav1.CreateOptions,
	) (*unstructured.Unstructured, error)

	Update(
		ctx context.Context, desc resources.Descriptor, obj *unstructured.Unstructured, opts metav1.UpdateOptions,
	) (*unstructured.Unstructured, error)

	Get(
		ctx context.Context, desc resources.Descriptor, name string, opts metav1.GetOptions,
	) (*unstructured.Unstructured, error)
}

// Pusher takes care of pushing resources to Grafana API.
type Pusher struct {
	client   PushClient
	registry PushRegistry
}

// NewDefaultPusher creates a new Pusher.
// It uses the default namespaced dynamic client to push resources to Grafana.
func NewDefaultPusher(ctx context.Context, cfg config.NamespacedRESTConfig) (*Pusher, error) {
	client, err := dynamic.NewDefaultNamespacedClient(cfg)
	if err != nil {
		return nil, err
	}

	registry, err := discovery.NewDefaultRegistry(ctx, cfg)
	if err != nil {
		return nil, err
	}

	return NewPusher(client, registry), nil
}

// NewPusher creates a new Pusher.
func NewPusher(client PushClient, registry PushRegistry) *Pusher {
	return &Pusher{
		client:   client,
		registry: registry,
	}
}

// PushRequest is a request for pushing resources to Grafana.
type PushRequest struct {
	// A list of resources to push.
	Resources *resources.Resources

	// Processors to apply to resources before pushing them.
	Processors []Processor

	// The maximum number of concurrent pushes.
	MaxConcurrency int

	// Whether the operation should stop upon encountering an error.
	StopOnError bool

	// If set to true, the pusher will use the server-side dry-run feature to simulate the push operations.
	// This will not actually create or update any resources,
	// but will ensure the requests are valid and perform server-side validations.
	DryRun bool

	// Disable log emission for push failures. Callers will have to rely on the PushSummary
	// returned by the Push() function to explore and report failures.
	NoPushFailureLog bool

	// Whether to include resources managed by other tools.
	IncludeManaged bool
}

type PushFailure struct {
	Resource *resources.Resource
	Error    error
}

type PushSummary struct {
	PushedCount int
	FailedCount int
	Failures    []PushFailure
	mu          sync.Mutex
}

func (summary *PushSummary) recordFailure(resource *resources.Resource, err error) {
	summary.mu.Lock()
	defer summary.mu.Unlock()

	summary.FailedCount++
	summary.Failures = append(summary.Failures, PushFailure{
		Resource: resource,
		Error:    err,
	})
}

// Push pushes resources to Grafana.
func (p *Pusher) Push(ctx context.Context, request PushRequest) (*PushSummary, error) {
	summary := &PushSummary{}
	supported := p.supportedDescriptors()

	if request.MaxConcurrency < 1 {
		request.MaxConcurrency = 1
	}

	err := request.Resources.ForEachConcurrently(
		ctx, request.MaxConcurrency, func(ctx context.Context, res *resources.Resource) error {
			name := res.Name()
			gvk := res.GroupVersionKind()

			logger := logging.FromContext(ctx).With(
				"gvk", gvk,
				"name", name,
				"dryRun", request.DryRun,
			)

			desc, ok := supported[gvk]
			if !ok {
				err := fmt.Errorf("resource not supported by the API: %s/%s", gvk, name)
				summary.recordFailure(res, err)

				if request.StopOnError {
					return err
				}

				if !request.NoPushFailureLog {
					logger.Warn("Skipping resource not supported by the API")
				}
				return nil
			}

			for _, processor := range request.Processors {
				if err := processor.Process(res); err != nil {
					summary.recordFailure(res, err)

					if request.StopOnError {
						return err
					}

					if !request.NoPushFailureLog {
						logger.Warn("Failed to process resource", logs.Err(err))
					}

					return nil
				}
			}

			if !res.IsManaged() && !request.IncludeManaged {
				logger.Info(fmt.Sprintf("Skipping resource managed by %s", res.GetManagerKind()))
				return nil
			}

			if err := p.upsertResource(ctx, desc, name, res, request.DryRun, logger); err != nil {
				summary.recordFailure(res, err)

				if request.StopOnError {
					return err
				}

				if !request.NoPushFailureLog {
					logger.Warn("Failed to push resource", logs.Err(err))
				}
				return nil
			}

			logger.Info("Resource pushed")
			summary.PushedCount++
			return nil
		},
	)

	return summary, err
}

func (p *Pusher) upsertResource(
	ctx context.Context, desc resources.Descriptor, name string, src *resources.Resource, dryRun bool, log logging.Logger,
) error {
	var dryRunOpts []string
	if dryRun {
		dryRunOpts = []string{"All"}
	}

	// Check if the resource already exists.
	_, err := p.client.Get(ctx, desc, name, metav1.GetOptions{})
	if err == nil {
		obj := src.ToUnstructured()

		// Otherwise, update the resource.
		// TODO: double-check if we need to do some resource version shenanigans here.
		// (most likely yes)
		// Something like – take existing resource, replace the annotations, labels, spec, etc.
		// and then push it back.
		if _, err := p.client.Update(ctx, desc, &obj, metav1.UpdateOptions{
			DryRun: dryRunOpts,
		}); err != nil {
			return err
		}

		log.Info("Resource updated")
		return nil
	}

	// If the resource does not exist, create it.
	if apierrors.IsNotFound(err) {
		obj := src.ToUnstructured()
		if _, err := p.client.Create(ctx, desc, &obj, metav1.CreateOptions{
			DryRun: dryRunOpts,
		}); err != nil {
			return err
		}

		log.Info("Resource created")
		return nil
	}

	// Some unknown error occurred, return it.
	return err
}

func (p *Pusher) supportedDescriptors() map[schema.GroupVersionKind]resources.Descriptor {
	supported := p.registry.SupportedResources()

	supportedDescriptors := make(map[schema.GroupVersionKind]resources.Descriptor)
	for _, sup := range supported {
		supportedDescriptors[sup.GroupVersionKind()] = sup
	}

	return supportedDescriptors
}
