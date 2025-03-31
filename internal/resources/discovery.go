package resources

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"sync"

	"github.com/grafana/grafanactl/internal/config"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/discovery"
)

// Kind is a kind of resource.
type Kind struct {
	Plural   string
	Singular string
	Group    string
}

// Group is a group of resources.
type Group struct {
	Short    string
	Long     string
	Versions []string
}

// DiscoveryRegistry is a registry of resources and their preferred versions.
type DiscoveryRegistry struct {
	client    discovery.DiscoveryInterface
	preferred map[schema.GroupResource]schema.GroupVersionResource
	kinds     []Kind
	groups    []Group
	mu        sync.RWMutex
}

// NewDefaultDiscoveryRegistry creates a new discovery registry using the default discovery client.
func NewDefaultDiscoveryRegistry(cfg config.NamespacedRESTConfig) (*DiscoveryRegistry, error) {
	client, err := discovery.NewDiscoveryClientForConfig(&cfg.Config)
	if err != nil {
		return nil, err
	}

	return NewDiscoveryRegistry(client)
}

// NewDiscoveryRegistry creates a new discovery registry.
//
// The registry will be populated with the resources and their preferred versions
// by calling the server's preferred resources endpoint.
//
// The registry will perform the discovery upon initialization.
func NewDiscoveryRegistry(client discovery.DiscoveryInterface) (*DiscoveryRegistry, error) {
	reg := &DiscoveryRegistry{
		client:    client,
		preferred: make(map[schema.GroupResource]schema.GroupVersionResource),
	}

	if err := reg.Discover(context.Background()); err != nil {
		return nil, err
	}

	return reg, nil
}

// GetPreferred returns the preferred version of a resource.
//
// If forceUpdate is true or the resource is not found in the registry,
// the registry will perform a new discovery.
//
// If the resource is not found in the registry, an error will be returned.
func (r *DiscoveryRegistry) GetPreferred(
	ctx context.Context, gvk DynamicGroupVersionKind, forceUpdate bool,
) (schema.GroupVersionResource, error) {
	res, err := r.lookup(gvk)
	if forceUpdate || err != nil {
		if err := r.Discover(ctx); err != nil {
			return schema.GroupVersionResource{}, err
		}

		res, err = r.lookup(gvk)
		if err != nil {
			return schema.GroupVersionResource{}, err
		}
	}

	return res, nil
}

// Resources returns all resources in the Grafana API.
func (r *DiscoveryRegistry) Resources(
	ctx context.Context, forceUpdate bool,
) ([]DynamicGroupVersionKind, error) {
	if forceUpdate || len(r.groups) == 0 {
		if err := r.Discover(ctx); err != nil {
			return nil, err
		}
	}

	res := make([]DynamicGroupVersionKind, 0, len(r.groups))
	for _, g := range r.groups {
		for _, v := range g.Versions {
			for _, k := range r.kinds {
				if k.Group == g.Long {
					res = append(res, DynamicGroupVersionKind{
						Group:   g.Long,
						Version: v,
						Kind:    k.Singular,
					})
				}
			}
		}
	}

	return res, nil
}

// Discover discovers the resources and their preferred versions from the server,
// and stores them in the registry.
func (r *DiscoveryRegistry) Discover(context.Context) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	preferred, err := r.client.ServerPreferredResources()
	if err != nil {
		return err
	}

	for _, pref := range preferred {
		groupVersion := strings.Split(pref.GroupVersion, "/")
		if len(groupVersion) != 2 {
			return fmt.Errorf("invalid group version: %s", pref.GroupVersion)
		}

		// APIResources can contain empty values for group and version,
		// which indicate that the group & version values should be taken from the APIResourceList.
		group := groupVersion[0]
		version := groupVersion[1]

		for _, res := range pref.APIResources {
			// Check if the resource has a specified version,
			// if so, we'll use that version.
			if res.Version != "" {
				version = res.Version
			}

			// Same as above, but for the group.
			if res.Group != "" {
				group = res.Group
			}

			short := strings.SplitN(group, ".", 2)[0]
			r.groups = append(r.groups, Group{
				Short: short,
				Long:  group,
				// TODO: add support for multiple versions
				Versions: []string{version},
			})

			r.kinds = append(r.kinds, Kind{
				Plural:   strings.ToLower(res.Name),
				Singular: strings.ToLower(res.SingularName),
				Group:    group,
			})

			r.preferred[schema.GroupResource{
				Group:    group,
				Resource: strings.ToLower(res.Name),
			}] = schema.GroupVersionResource{
				Group:    group,
				Version:  version,
				Resource: res.Name,
			}
		}
	}

	return nil
}

func (r *DiscoveryRegistry) lookup(gvk DynamicGroupVersionKind) (schema.GroupVersionResource, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return LookupGVR(gvk, LookupOptions{
		Groups:    r.groups,
		Kinds:     r.kinds,
		Preferred: r.preferred,
	})
}

type LookupOptions struct {
	Groups    []Group
	Kinds     []Kind
	Preferred map[schema.GroupResource]schema.GroupVersionResource
}

// LookupGVR looks up a GVR for a given DynamicGroupVersionKind.
func LookupGVR(gvk DynamicGroupVersionKind, opts LookupOptions) (schema.GroupVersionResource, error) {
	var res schema.GroupVersionResource

	// If the group was specified, it takes precedence,
	// because there can be multiple kinds with the same name in different groups.
	// This is a less ambiguous lookup path.
	if gvk.Group != "" { //nolint:nestif
		for _, group := range opts.Groups {
			if gvk.Group == group.Short || gvk.Group == group.Long {
				res.Group = group.Long

				// If the version was specified, we need to make sure it's supported.
				if gvk.Version != "" {
					if !slices.Contains(group.Versions, gvk.Version) {
						return schema.GroupVersionResource{}, fmt.Errorf(
							"the server does not support version '%s' of API group '%s'",
							gvk.Version, gvk.Group,
						)
					}

					res.Version = gvk.Version
				}

				break
			}
		}

		// If we still don't have a group, we can't find the resource.
		if res.Group == "" {
			return schema.GroupVersionResource{}, fmt.Errorf(
				"the server does not support API group '%s'", gvk.Group,
			)
		}
	}

	// Then we need to find the canonical kind name.
	// If at this point we still don't have a group,
	// we'll use the group of the first kind we find.
	for _, kind := range opts.Kinds {
		if kind.Singular == gvk.Kind || kind.Plural == gvk.Kind {
			// If we already have a group, we need to make sure it matches.
			if res.Group != "" && res.Group != kind.Group {
				continue
			}

			// Otherwise we pick the first kind we found and use its group.
			res.Resource = kind.Plural
			res.Group = kind.Group
			break
		}
	}

	// At this point we should have a group and a kind.
	if res.Resource == "" {
		return schema.GroupVersionResource{}, fmt.Errorf(
			"the server does not support API resource '%s/%s'", gvk.Group, gvk.Kind,
		)
	}

	// Lastly, if the version was not specified, we'll use the preferred version instead.
	if res.Version == "" {
		pref, ok := opts.Preferred[schema.GroupResource{
			Group:    res.Group,
			Resource: res.Resource,
		}]

		if !ok {
			return schema.GroupVersionResource{}, fmt.Errorf(
				"the server does not support API resource '%s/%s/%s'", res.Group, res.Version, res.Resource,
			)
		}

		res.Version = pref.Version
	}

	return res, nil
}

// TODO:
// index shapes:
// kind -> [groups]
// shortGroup -> [longGroup]
// longGroup -> [groupVersion] (plus a group/EmptyVersion for the default preferred version if none is specified)
// groupVersion -> [kinds]
