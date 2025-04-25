package discovery

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/grafana/grafana-app-sdk/logging"
	"github.com/grafana/grafanactl/internal/resources"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// RegistryIndex is an index with information
// about the resources & groups supported by the API server.
type RegistryIndex struct {
	lock sync.RWMutex

	shortGroups       map[string]string
	longGroups        map[string]struct{}
	preferredVersions map[string]schema.GroupVersion
	descriptors       map[schema.GroupVersion]resources.Descriptors
	singularNames     map[string][]schema.GroupKind
	pluralNames       map[string][]schema.GroupKind
	kindNames         map[string][]schema.GroupKind
}

// NewRegistryIndex creates a new empty index.
func NewRegistryIndex() RegistryIndex {
	return RegistryIndex{
		shortGroups:       make(map[string]string),
		longGroups:        make(map[string]struct{}),
		preferredVersions: make(map[string]schema.GroupVersion),
		descriptors:       make(map[schema.GroupVersion]resources.Descriptors),
		kindNames:         make(map[string][]schema.GroupKind),
		singularNames:     make(map[string][]schema.GroupKind),
		pluralNames:       make(map[string][]schema.GroupKind),
	}
}

// GetDescriptors returns all resource Descriptors from the registry index.
func (r *RegistryIndex) GetDescriptors() resources.Descriptors {
	r.lock.RLock()
	defer r.lock.RUnlock()

	res := make(resources.Descriptors, 0, len(r.descriptors))
	for _, descs := range r.descriptors {
		res = append(res, descs...)
	}

	return res
}

// GetPreferredVersions returns all preferred versions of the API groups.
func (r *RegistryIndex) GetPreferredVersions() resources.Descriptors {
	r.lock.RLock()
	defer r.lock.RUnlock()

	res := make(resources.Descriptors, 0, len(r.preferredVersions))
	for _, gv := range r.preferredVersions {
		desc, ok := r.descriptors[gv]
		if !ok {
			// TODO: this should never happen.
			// panic?
			continue
		}

		res = append(res, desc...)
	}

	return res
}

// LookupPartialGVK returns a descriptor for the provided partial GVK.
// It tries to find the most precise match based on the information in the partial GVK.
// (i.e. it will try to scope down to the exact group & version if provided and fall back to the preferred version if not).
// If no group is provided, it will return the first group that supports the resource.
func (r *RegistryIndex) LookupPartialGVK(gvk resources.PartialGVK) (resources.Descriptor, bool) {
	groupKindCandidates, ok := r.getKindCandidates(gvk.Resource)
	if !ok {
		return resources.Descriptor{}, false
	}

	desc, ok := r.filterCandidates(groupKindCandidates, gvk.Group, gvk.Version)
	if !ok {
		return resources.Descriptor{}, false
	}

	return desc, true
}

// Update updates the registry index from the provided API groups and resources.
func (r *RegistryIndex) Update(ctx context.Context, groups []*metav1.APIGroup, list []*metav1.APIResourceList) error {
	r.lock.Lock()
	defer r.lock.Unlock()

	logger := logging.FromContext(ctx)

	var (
		// API group indices
		shortGroups       = make(map[string]string)
		longGroups        = make(map[string]struct{})
		preferredVersions = make(map[string]schema.GroupVersion)
		descriptors       = make(map[schema.GroupVersion]resources.Descriptors)

		// API resource indices
		kindNames     = make(map[string][]schema.GroupKind)
		singularNames = make(map[string][]schema.GroupKind)
		pluralNames   = make(map[string][]schema.GroupKind)
	)

	// First pass: collect preferred versions and group versions.
	for _, group := range groups {
		name := group.Name
		longGroups[name] = struct{}{}

		shortName := makeShortName(name)
		shortGroups[shortName] = name

		if group.PreferredVersion.Version != "" {
			preferredVersions[name] = schema.GroupVersion{
				Group:   name,
				Version: group.PreferredVersion.Version,
			}
		}

		for _, version := range group.Versions {
			gv := schema.GroupVersion{
				Group:   name,
				Version: version.Version,
			}

			if _, ok := descriptors[gv]; ok {
				logger.Info(
					"inconsistent discovery response from the server - duplicate GroupVersion entry",
					"group", name,
					"version", version.Version,
				)

				return fmt.Errorf("duplicate group version: %s", gv.String())
			}

			descriptors[gv] = make(resources.Descriptors, 0)
		}
	}

	// Second pass: collect resources and put them into the descriptors.
	// Each descriptor will be associated with a group version.
	for _, list := range list {
		for _, res := range list.APIResources {
			gv, err := parseGroupVersion(list.GroupVersion)
			if err != nil {
				return err
			}

			// Check if the resource has a specified group,
			// if so, we'll use that group.
			if res.Group != "" {
				gv.Group = res.Group
			}

			// Same as above, but for the version.
			if res.Version != "" {
				gv.Version = res.Version
			}

			desc, ok := descriptors[gv]
			if !ok {
				logger.Info(
					"inconsistent discovery response from the server - API resource is missing from API groups",
					"group", gv.Group,
					"version", gv.Version,
					"resource", res.Name,
				)

				return fmt.Errorf(
					"unexpected response from the server - resource is not listed as supported: %s",
					gv.String(),
				)
			}

			// TODO: filter out subresources
			logger.Debug("adding resource to discovery registry",
				"group", gv.Group,
				"version", gv.Version,
				"kind", res.Kind,
				"singular", res.SingularName,
				"plural", res.Name,
			)

			descriptors[gv] = append(desc, resources.Descriptor{
				GroupVersion: gv,
				Kind:         res.Kind,
				Plural:       res.Name,
				Singular:     res.SingularName,
			})

			if _, ok := kindNames[res.Kind]; !ok {
				kindNames[res.Kind] = make([]schema.GroupKind, 0)
			}

			kindNames[res.Kind] = append(kindNames[res.Kind], schema.GroupKind{
				Group: gv.Group,
				Kind:  res.Kind,
			})

			if _, ok := singularNames[res.SingularName]; !ok {
				singularNames[res.SingularName] = make([]schema.GroupKind, 0)
			}

			singularNames[res.SingularName] = append(singularNames[res.SingularName], schema.GroupKind{
				Group: gv.Group,
				Kind:  res.Kind,
			})

			if _, ok := pluralNames[res.Name]; !ok {
				pluralNames[res.Name] = make([]schema.GroupKind, 0)
			}

			pluralNames[res.Name] = append(pluralNames[res.Name], schema.GroupKind{
				Group: gv.Group,
				Kind:  res.Kind,
			})
		}
	}

	r.shortGroups = shortGroups
	r.longGroups = longGroups
	r.preferredVersions = preferredVersions
	r.descriptors = descriptors
	r.kindNames = kindNames
	r.singularNames = singularNames
	r.pluralNames = pluralNames

	return nil
}

func (r *RegistryIndex) getKindCandidates(resource string) ([]schema.GroupKind, bool) {
	if k, ok := r.kindNames[resource]; ok {
		return k, true
	} else if k, ok := r.singularNames[resource]; ok {
		return k, true
	} else if k, ok := r.pluralNames[resource]; ok {
		return k, true
	}

	// We don't have a kind name, so we can't find the resource.
	return nil, false
}

func (r *RegistryIndex) filterCandidates(
	groupKindCandidates []schema.GroupKind, group, version string,
) (resources.Descriptor, bool) {
	var groupVersion schema.GroupVersion

	if group != "" {
		found := false

		// Check if the group was provided in the short form.
		if g, ok := r.shortGroups[group]; ok {
			groupVersion.Group = g
			found = true
		}

		// Check if the group was provided in the long form.
		if _, ok := r.longGroups[group]; ok {
			groupVersion.Group = group
			found = true
		}

		if !found {
			return resources.Descriptor{}, false
		}
	}

	// Check that the group is supported.
	var kind string
	for _, gk := range groupKindCandidates {
		// TODO: there's an issue here that if we have resource & version (but not group)
		// we end up choosing the first group that supports the resource, but this might not be what we want.
		// e.g. the lookup could be for resource = Foo, version = v1
		// And we use group = Bar but that group might not support v1
		// (whereas some other group does support v1 and has the same resource Foo)
		if groupVersion.Group == "" {
			groupVersion.Group = gk.Group
		}

		if groupVersion.Group == gk.Group {
			kind = gk.Kind
			break
		}
	}

	// If somehow we didn't find a kind, we can't proceed.
	// This could happen if the group & kind combination doesn't exist.
	if kind == "" {
		return resources.Descriptor{}, false
	}

	if version != "" {
		groupVersion.Version = version
	} else if gv, ok := r.preferredVersions[groupVersion.Group]; ok {
		groupVersion.Version = gv.Version
	} else {
		// Somehow the group is supported, but we don't have a preferred version for it.
		// TODO: panic?
		return resources.Descriptor{}, false
	}

	descs, ok := r.descriptors[groupVersion]
	if !ok {
		return resources.Descriptor{}, false
	}

	for _, desc := range descs {
		if desc.Kind == kind {
			return desc, true
		}
	}

	return resources.Descriptor{}, false
}

func makeShortName(name string) string {
	return strings.Split(name, ".")[0]
}

func parseGroupVersion(src string) (schema.GroupVersion, error) {
	parts := strings.Split(src, "/")

	if len(parts) != 2 {
		return schema.GroupVersion{}, fmt.Errorf("invalid group version: %s", src)
	}

	if parts[0] == "" {
		return schema.GroupVersion{}, fmt.Errorf("invalid group version: %s", src)
	}

	if parts[1] == "" {
		return schema.GroupVersion{}, fmt.Errorf("invalid group version: %s", src)
	}

	return schema.GroupVersion{
		Group:   parts[0],
		Version: parts[1],
	}, nil
}
