package resources

import (
	"context"
	"fmt"

	"github.com/grafana/grafana-app-sdk/logging"
	"github.com/grafana/grafanactl/internal/config"
	"github.com/grafana/grafanactl/internal/logs"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Pusher takes care of pushing resources to Grafana API.
type Pusher struct {
	client *NamespacedDynamicClient
}

// NewPusher creates a new Pusher.
func NewPusher(ctx context.Context, cfg config.NamespacedRESTConfig) (*Pusher, error) {
	client, err := NewDefaultNamespacedDynamicClient(ctx, cfg)
	if err != nil {
		return nil, err
	}

	return &Pusher{
		client: client,
	}, nil
}

// PushRequest is a request for pushing resources to Grafana.
type PushRequest struct {
	// A list of selector filters to apply to the resources before pushing them.
	Selectors []Selector

	// A list of resources to push.
	Resources *Resources

	// The maximum number of concurrent pushes.
	MaxConcurrency int

	// Whether the operation should stop upon encountering an error.
	StopOnError bool

	// Whether to overwrite existing resources.
	// If false, the resource will be skipped if it already exists and is a newer version,
	// compared to the resource passed in the request.
	OverwriteExisting bool

	// If set to true, the pusher will use the server-side dry-run feature to simulate the push operations.
	// This will not actually create or update any resources,
	// but will ensure the requests are valid and perform server-side validations.
	DryRun bool
}

// Push pushes resources to Grafana.
func (p *Pusher) Push(ctx context.Context, request PushRequest) error {
	supported, err := p.supportedGVKs(ctx)
	if err != nil {
		return err
	}

	if request.MaxConcurrency < 1 {
		request.MaxConcurrency = 1
	}

	return request.Resources.ForEachConcurrently(ctx, request.MaxConcurrency,
		func(ctx context.Context, res *Resource) error {
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

			if err := p.upsertResource(
				ctx, gvk, name, res, request.OverwriteExisting, request.DryRun, logger); err != nil {
				if request.StopOnError {
					return err
				}

				logger.Warn("Failed to push resource", logs.Err(err))
				return nil
			}

			logger.Info("Resource pushed")
			return nil
		},
	)
}

func (p *Pusher) upsertResource(
	ctx context.Context,
	gvk GroupVersionKind,
	name string,
	src *Resource,
	overwrite bool,
	dryRun bool,
	logger logging.Logger,
) error {
	var dryRunOpts []string
	if dryRun {
		dryRunOpts = []string{"All"}
	}

	// Check if the resource already exists.
	dst, err := p.client.Get(ctx, gvk, name, metav1.GetOptions{})
	//nolint:nestif
	if err == nil {
		// If the resource already exists, check if it is a newer version.
		// If it is, and overwrite is not set, skip the resource.
		if dst.GetResourceVersion() != src.Raw.GetResourceVersion() {
			if !overwrite {
				return fmt.Errorf(
					"resource `%s/%s` already exists with a different resource version: %s",
					gvk, name, dst.GetResourceVersion(),
				)
			}

			// Force the resource version to be the same as the destination resource version.
			// This effectively means that we will overwrite the resource in the API with local data.
			// This will lead to data loss but that's what we want since the overwrite flag is set to true.
			src.Raw.SetResourceVersion(dst.GetResourceVersion())
		}

		unstructuredObj, err := src.ToUnstructured()
		if err != nil {
			return err
		}

		// Otherwise, update the resource.
		// TODO: double-check if we need to do some resource version shenanigans here.
		// (most likely yes)
		// Something like – take existing resource, replace the annotations, labels, spec, etc.
		// and then push it back.
		if _, err := p.client.Update(ctx, gvk, unstructuredObj, metav1.UpdateOptions{
			DryRun: dryRunOpts,
		}); err != nil {
			return err
		}

		logger.Info("Resource updated")
		return nil
	}

	// If the resource does not exist, create it.
	if apierrors.IsNotFound(err) {
		unstructuredObj, err := src.ToUnstructured()
		if err != nil {
			return err
		}

		if _, err := p.client.Create(ctx, gvk, unstructuredObj, metav1.CreateOptions{
			DryRun: dryRunOpts,
		}); err != nil {
			return err
		}

		logger.Info("Resource created")
		return nil
	}

	// Some unknown error occurred, return it.
	return err
}

func (p *Pusher) supportedGVKs(ctx context.Context) (map[GroupVersionKind]struct{}, error) {
	supported, err := p.client.Resources(ctx)
	if err != nil {
		return nil, err
	}

	supportedGVKs := make(map[GroupVersionKind]struct{})
	for _, sup := range supported {
		supportedGVKs[GroupVersionKind{
			Group:   sup.Group,
			Version: sup.Version,
			// NB: this is deliberate, because the kind on disk is the actual kind,
			// but what we stored in the registry under `Name` is the singular form of the kind.
			Kind: sup.Name,
		}] = struct{}{}
	}

	return supportedGVKs, nil
}
