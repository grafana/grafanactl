package discovery

import (
	"context"
	"slices"
	"strings"
	"time"

	"github.com/grafana/grafanactl/internal/config"
	"github.com/grafana/grafanactl/internal/resources"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/discovery"
)

// ignoredResourceGroups is a list of resource groups that are supported by Grafana API.
// But are not supposed to be used by the clients just yet.
// (or in case of some groups they are read-only by design)
//
//nolint:gochecknoglobals
var ignoredResourceGroups = []string{
	"apiregistration.k8s.io",
	"featuretoggle.grafana.app",
	"service.grafana.app",
	"userstorage.grafana.app",
	// TODO: check with alerting folks if this should be ignored or not
	"notifications.alerting.grafana.app",
	"iam.grafana.app",
}

// Client is a client that can be used to discover resources.
type Client interface {
	ServerGroupsAndResources() ([]*metav1.APIGroup, []*metav1.APIResourceList, error)
}

// Registry is a registry of resources and their preferred versions.
type Registry struct {
	client          Client
	index           RegistryIndex
	refreshInterval time.Duration
}

// NewDefaultRegistry creates a new discovery registry using the default discovery client.
func NewDefaultRegistry(
	ctx context.Context, cfg config.NamespacedRESTConfig, refreshInterval time.Duration,
) (*Registry, error) {
	client, err := discovery.NewDiscoveryClientForConfig(&cfg.Config)
	if err != nil {
		return nil, err
	}

	return NewRegistry(ctx, client, refreshInterval)
}

// NewRegistry creates a new discovery registry.
//
// The registry will be populated with the resources and their preferred versions
// by calling the server's preferred resources endpoint.
//
// The registry will perform the discovery upon initialization.
func NewRegistry(ctx context.Context, client Client, refreshInterval time.Duration) (*Registry, error) {
	reg := &Registry{
		client:          client,
		index:           NewRegistryIndex(),
		refreshInterval: refreshInterval,
	}

	// TODO: implement refresh loop
	// if refreshInterval > 0 {
	// }

	// Perform initial discovery.
	// TODO: should this be optional?
	if err := reg.Discover(ctx); err != nil {
		return reg, err
	}

	return reg, nil
}

// MakeFilters creates a set of filters for the given selectors.
func (r *Registry) MakeFilters(selectors resources.Selectors) (resources.Filters, error) {
	filters := make(resources.Filters, len(selectors))

	for i, selector := range selectors {
		desc, ok := r.index.LookupPartialGVK(selector.GroupVersionKind)
		if !ok {
			return nil, resources.InvalidSelectorError{Command: selector.String(), Err: "the server does not support this resource"}
		}

		filters[i].Type = selector.Type
		filters[i].ResourceUIDs = selector.ResourceUIDs
		filters[i].Descriptor = desc
	}

	return filters, nil
}

// PreferredResources returns all resources with their preferred versions.
func (r *Registry) PreferredResources() resources.Descriptors {
	return r.index.GetPreferredVersions()
}

// SupportedResources returns all resources supported by the server.
func (r *Registry) SupportedResources() resources.Descriptors {
	return r.index.GetDescriptors()
}

// Discover discovers the resources and their preferred versions from the server,
// and stores them in the registry.
func (r *Registry) Discover(ctx context.Context) error {
	apiGroups, apiResources, err := r.client.ServerGroupsAndResources()
	if err != nil {
		return err
	}

	// Filter out ignored resource groups.
	apiGroups, apiResources, err = FilterDiscoveryResults(ignoredResourceGroups, apiGroups, apiResources)
	if err != nil {
		return err
	}

	return r.index.Update(ctx, apiGroups, apiResources)
}

// FilterDiscoveryResults filters the discovery results to exclude ignored resource groups.
func FilterDiscoveryResults(
	ignored []string, apiGroups []*metav1.APIGroup, apiResources []*metav1.APIResourceList,
) ([]*metav1.APIGroup, []*metav1.APIResourceList, error) {
	filteredGroups := make([]*metav1.APIGroup, 0, len(apiGroups))
	filteredResources := make([]*metav1.APIResourceList, 0, len(apiResources))

	for _, group := range apiGroups {
		if slices.Contains(ignored, group.Name) {
			continue
		}

		filteredGroups = append(filteredGroups, group)
	}

	for _, resource := range apiResources {
		gv, err := parseGroupVersion(resource.GroupVersion)
		if err != nil {
			return nil, nil, err
		}

		if slices.Contains(ignored, gv.Group) {
			continue
		}

		filteredAPIResources := make([]metav1.APIResource, 0, len(resource.APIResources))
		for _, r := range resource.APIResources {
			if !r.Namespaced {
				continue
			}

			// TODO (@radiohead): this excludes subresources, but we should check if that's what we want.
			if strings.Contains(r.Name, "/") {
				continue
			}

			filteredAPIResources = append(filteredAPIResources, r)
		}
		resource.APIResources = filteredAPIResources

		filteredResources = append(filteredResources, resource)
	}

	return filteredGroups, filteredResources, nil
}
