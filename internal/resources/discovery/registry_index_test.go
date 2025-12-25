package discovery_test

import (
	"testing"

	"github.com/grafana/grafanactl/internal/resources"
	"github.com/grafana/grafanactl/internal/resources/discovery"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func TestRegistryIndex_GetDescriptors(t *testing.T) {
	tests := []struct {
		name      string
		discovery func() ([]*metav1.APIGroup, []*metav1.APIResourceList)
		want      resources.Descriptors
	}{
		{
			name:      "empty registry returns empty descriptors",
			discovery: getEmptyDiscovery,
			want:      resources.Descriptors{},
		},
		{
			name:      "registry with resources returns descriptors",
			discovery: getSingleVersionDiscovery,
			want: resources.Descriptors{
				{
					Kind:     "Dashboard",
					Plural:   "dashboards",
					Singular: "dashboard",
					GroupVersion: schema.GroupVersion{
						Group:   "dashboard.grafana.app",
						Version: "v1",
					},
				},
				{
					Kind:     "Folder",
					Plural:   "folders",
					Singular: "folder",
					GroupVersion: schema.GroupVersion{
						Group:   "folder.grafana.app",
						Version: "v1",
					},
				},
			},
		},
		{
			name:      "registry with multiple versions returns all descriptors",
			discovery: getMultipleVersionsDiscovery,
			want: resources.Descriptors{
				{
					Kind:     "Dashboard",
					Plural:   "dashboards",
					Singular: "dashboard",
					GroupVersion: schema.GroupVersion{
						Group:   "dashboard.grafana.app",
						Version: "v1",
					},
				},
				{
					Kind:     "Dashboard",
					Plural:   "dashboards",
					Singular: "dashboard",
					GroupVersion: schema.GroupVersion{
						Group:   "dashboard.grafana.app",
						Version: "v2",
					},
				},
			},
		},
		{
			name:      "registry with mixed versions returns all descriptors",
			discovery: getMixedVersionsDiscovery,
			want: resources.Descriptors{
				{
					Kind:     "Dashboard",
					Plural:   "dashboards",
					Singular: "dashboard",
					GroupVersion: schema.GroupVersion{
						Group:   "dashboard.grafana.app",
						Version: "v1",
					},
				},
				{
					Kind:     "Dashboard",
					Plural:   "dashboards",
					Singular: "dashboard",
					GroupVersion: schema.GroupVersion{
						Group:   "dashboard.grafana.app",
						Version: "v2",
					},
				},
				{
					Kind:     "Folder",
					Plural:   "folders",
					Singular: "folder",
					GroupVersion: schema.GroupVersion{
						Group:   "folder.grafana.app",
						Version: "v1",
					},
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			registryIndex := discovery.NewRegistryIndex()
			groups, resources := test.discovery()
			require.NoError(t, registryIndex.Update(t.Context(), groups, resources))
			assert.ElementsMatch(t, test.want, registryIndex.GetDescriptors())
		})
	}
}

func TestRegistryIndex_GetPreferredVersions(t *testing.T) {
	tests := []struct {
		name      string
		discovery func() ([]*metav1.APIGroup, []*metav1.APIResourceList)
		want      resources.Descriptors
	}{
		{
			name:      "empty registry returns empty descriptors",
			discovery: getEmptyDiscovery,
			want:      resources.Descriptors{},
		},
		{
			name:      "registry with single version returns that version",
			discovery: getSingleVersionDiscovery,
			want: resources.Descriptors{
				{
					Kind:     "Dashboard",
					Plural:   "dashboards",
					Singular: "dashboard",
					GroupVersion: schema.GroupVersion{
						Group:   "dashboard.grafana.app",
						Version: "v1",
					},
				},
				{
					Kind:     "Folder",
					Plural:   "folders",
					Singular: "folder",
					GroupVersion: schema.GroupVersion{
						Group:   "folder.grafana.app",
						Version: "v1",
					},
				},
			},
		},
		{
			name:      "registry with multiple versions returns preferred versions",
			discovery: getMultipleVersionsDiscovery,
			want: resources.Descriptors{
				{
					Kind:     "Dashboard",
					Plural:   "dashboards",
					Singular: "dashboard",
					GroupVersion: schema.GroupVersion{
						Group:   "dashboard.grafana.app",
						Version: "v2",
					},
				},
			},
		},
		{
			name:      "registry with mixed versions returns preferred versions",
			discovery: getMixedVersionsDiscovery,
			want: resources.Descriptors{
				{
					Kind:     "Dashboard",
					Plural:   "dashboards",
					Singular: "dashboard",
					GroupVersion: schema.GroupVersion{
						Group:   "dashboard.grafana.app",
						Version: "v2",
					},
				},
				{
					Kind:     "Folder",
					Plural:   "folders",
					Singular: "folder",
					GroupVersion: schema.GroupVersion{
						Group:   "folder.grafana.app",
						Version: "v1",
					},
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			registryIndex := discovery.NewRegistryIndex()
			groups, resources := test.discovery()
			require.NoError(t, registryIndex.Update(t.Context(), groups, resources))
			assert.ElementsMatch(t, test.want, registryIndex.GetPreferredVersions())
		})
	}
}

func TestRegistryIndex_LookupPartialGVK(t *testing.T) {
	tests := []struct {
		name      string
		discovery func() ([]*metav1.APIGroup, []*metav1.APIResourceList)
		gvk       resources.PartialGVK
		want      resources.Descriptor
		wantOK    bool
	}{
		{
			name:      "lookup with empty GVK returns not found",
			discovery: getSingleVersionDiscovery,
			gvk:       resources.PartialGVK{},
			want:      resources.Descriptor{},
			wantOK:    false,
		},
		{
			name:      "lookup with non-existent resource returns not found",
			discovery: getSingleVersionDiscovery,
			gvk: resources.PartialGVK{
				Resource: "nonexistent",
			},
			want:   resources.Descriptor{},
			wantOK: false,
		},
		{
			name:      "lookup with existing resource returns descriptor",
			discovery: getSingleVersionDiscovery,
			gvk: resources.PartialGVK{
				Resource: "dashboards",
			},
			want: resources.Descriptor{
				Kind:     "Dashboard",
				Plural:   "dashboards",
				Singular: "dashboard",
				GroupVersion: schema.GroupVersion{
					Group:   "dashboard.grafana.app",
					Version: "v1",
				},
			},
			wantOK: true,
		},
		{
			name:      "lookup with non-existent group returns not found",
			discovery: getSingleVersionDiscovery,
			gvk: resources.PartialGVK{
				Resource: "dashboards",
				Group:    "v1alpha1",
			},
			want:   resources.Descriptor{},
			wantOK: false,
		},
		{
			name:      "lookup with resource and group returns descriptor",
			discovery: getSingleVersionDiscovery,
			gvk: resources.PartialGVK{
				Resource: "folders",
				Group:    "folder.grafana.app",
			},
			want: resources.Descriptor{
				Kind:     "Folder",
				Plural:   "folders",
				Singular: "folder",
				GroupVersion: schema.GroupVersion{
					Group:   "folder.grafana.app",
					Version: "v1",
				},
			},
			wantOK: true,
		},
		{
			name:      "lookup with resource, group and version returns descriptor",
			discovery: getMultipleVersionsDiscovery,
			gvk: resources.PartialGVK{
				Resource: "dashboards",
				Group:    "dashboard.grafana.app",
				Version:  "v1",
			},
			want: resources.Descriptor{
				Kind:     "Dashboard",
				Plural:   "dashboards",
				Singular: "dashboard",
				GroupVersion: schema.GroupVersion{
					Group:   "dashboard.grafana.app",
					Version: "v1",
				},
			},
			wantOK: true,
		},
		{
			name:      "lookup with resource and group returns preferred version",
			discovery: getMixedVersionsDiscovery,
			gvk: resources.PartialGVK{
				Resource: "dashboards",
				Group:    "dashboard.grafana.app",
			},
			want: resources.Descriptor{
				Kind:     "Dashboard",
				Plural:   "dashboards",
				Singular: "dashboard",
				GroupVersion: schema.GroupVersion{
					Group:   "dashboard.grafana.app",
					Version: "v2",
				},
			},
			wantOK: true,
		},
		{
			name:      "lookup with resource, group and specific version returns that version",
			discovery: getMixedVersionsDiscovery,
			gvk: resources.PartialGVK{
				Resource: "dashboards",
				Group:    "dashboard.grafana.app",
				Version:  "v1",
			},
			want: resources.Descriptor{
				Kind:     "Dashboard",
				Plural:   "dashboards",
				Singular: "dashboard",
				GroupVersion: schema.GroupVersion{
					Group:   "dashboard.grafana.app",
					Version: "v1",
				},
			},
			wantOK: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			registryIndex := discovery.NewRegistryIndex()
			groups, resources := test.discovery()
			require.NoError(t, registryIndex.Update(t.Context(), groups, resources))

			actual, ok := registryIndex.LookupPartialGVK(test.gvk)
			assert.Equal(t, test.want, actual)
			assert.Equal(t, test.wantOK, ok)
		})
	}
}

func TestRegistryIndex_LookupAllVersionsForPartialGVK(t *testing.T) {
	tests := []struct {
		name      string
		discovery func() ([]*metav1.APIGroup, []*metav1.APIResourceList)
		gvk       resources.PartialGVK
		want      resources.Descriptors
		wantOK    bool
	}{
		{
			name:      "lookup with empty GVK returns not found",
			discovery: getSingleVersionDiscovery,
			gvk:       resources.PartialGVK{},
			want:      nil,
			wantOK:    false,
		},
		{
			name:      "lookup with non-existent resource returns not found",
			discovery: getSingleVersionDiscovery,
			gvk: resources.PartialGVK{
				Resource: "nonexistent",
			},
			want:   nil,
			wantOK: false,
		},
		{
			name:      "lookup with existing resource returns all versions",
			discovery: getMultipleVersionsDiscovery,
			gvk: resources.PartialGVK{
				Resource: "dashboards",
			},
			want: resources.Descriptors{
				{
					Kind:     "Dashboard",
					Plural:   "dashboards",
					Singular: "dashboard",
					GroupVersion: schema.GroupVersion{
						Group:   "dashboard.grafana.app",
						Version: "v1",
					},
				},
				{
					Kind:     "Dashboard",
					Plural:   "dashboards",
					Singular: "dashboard",
					GroupVersion: schema.GroupVersion{
						Group:   "dashboard.grafana.app",
						Version: "v2",
					},
				},
			},
			wantOK: true,
		},
		{
			name:      "lookup with resource and group returns all versions for that group",
			discovery: getMixedVersionsDiscovery,
			gvk: resources.PartialGVK{
				Resource: "dashboards",
				Group:    "dashboard.grafana.app",
			},
			want: resources.Descriptors{
				{
					Kind:     "Dashboard",
					Plural:   "dashboards",
					Singular: "dashboard",
					GroupVersion: schema.GroupVersion{
						Group:   "dashboard.grafana.app",
						Version: "v1",
					},
				},
				{
					Kind:     "Dashboard",
					Plural:   "dashboards",
					Singular: "dashboard",
					GroupVersion: schema.GroupVersion{
						Group:   "dashboard.grafana.app",
						Version: "v2",
					},
				},
			},
			wantOK: true,
		},
		{
			name:      "lookup with resource, group and version returns only that version",
			discovery: getMixedVersionsDiscovery,
			gvk: resources.PartialGVK{
				Resource: "dashboards",
				Group:    "dashboard.grafana.app",
				Version:  "v1",
			},
			want: resources.Descriptors{
				{
					Kind:     "Dashboard",
					Plural:   "dashboards",
					Singular: "dashboard",
					GroupVersion: schema.GroupVersion{
						Group:   "dashboard.grafana.app",
						Version: "v1",
					},
				},
			},
			wantOK: true,
		},
		{
			name:      "lookup with single version resource returns that version",
			discovery: getSingleVersionDiscovery,
			gvk: resources.PartialGVK{
				Resource: "dashboards",
			},
			want: resources.Descriptors{
				{
					Kind:     "Dashboard",
					Plural:   "dashboards",
					Singular: "dashboard",
					GroupVersion: schema.GroupVersion{
						Group:   "dashboard.grafana.app",
						Version: "v1",
					},
				},
			},
			wantOK: true,
		},
		{
			name:      "lookup with mixed discovery returns all versions from all groups",
			discovery: getMixedVersionsDiscovery,
			gvk: resources.PartialGVK{
				Resource: "dashboards",
			},
			want: resources.Descriptors{
				{
					Kind:     "Dashboard",
					Plural:   "dashboards",
					Singular: "dashboard",
					GroupVersion: schema.GroupVersion{
						Group:   "dashboard.grafana.app",
						Version: "v1",
					},
				},
				{
					Kind:     "Dashboard",
					Plural:   "dashboards",
					Singular: "dashboard",
					GroupVersion: schema.GroupVersion{
						Group:   "dashboard.grafana.app",
						Version: "v2",
					},
				},
			},
			wantOK: true,
		},
		{
			name:      "lookup with kind instead of resource returns all versions",
			discovery: getMultipleVersionsDiscovery,
			gvk: resources.PartialGVK{
				Resource: "Dashboard",
			},
			want: resources.Descriptors{
				{
					Kind:     "Dashboard",
					Plural:   "dashboards",
					Singular: "dashboard",
					GroupVersion: schema.GroupVersion{
						Group:   "dashboard.grafana.app",
						Version: "v1",
					},
				},
				{
					Kind:     "Dashboard",
					Plural:   "dashboards",
					Singular: "dashboard",
					GroupVersion: schema.GroupVersion{
						Group:   "dashboard.grafana.app",
						Version: "v2",
					},
				},
			},
			wantOK: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			registryIndex := discovery.NewRegistryIndex()
			groups, resources := test.discovery()
			require.NoError(t, registryIndex.Update(t.Context(), groups, resources))

			actual, ok := registryIndex.LookupAllVersionsForPartialGVK(test.gvk)
			assert.ElementsMatch(t, test.want, actual)
			assert.Equal(t, test.wantOK, ok)
		})
	}
}

func getEmptyDiscovery() ([]*metav1.APIGroup, []*metav1.APIResourceList) {
	return []*metav1.APIGroup{}, []*metav1.APIResourceList{}
}

func getSingleVersionDiscovery() ([]*metav1.APIGroup, []*metav1.APIResourceList) {
	groups := []*metav1.APIGroup{
		{
			Name: "dashboard.grafana.app",
			Versions: []metav1.GroupVersionForDiscovery{
				{
					GroupVersion: "dashboard.grafana.app/v1",
					Version:      "v1",
				},
			},
			PreferredVersion: metav1.GroupVersionForDiscovery{
				GroupVersion: "dashboard.grafana.app/v1",
				Version:      "v1",
			},
		},
		{
			Name: "folder.grafana.app",
			Versions: []metav1.GroupVersionForDiscovery{
				{
					GroupVersion: "folder.grafana.app/v1",
					Version:      "v1",
				},
			},
			PreferredVersion: metav1.GroupVersionForDiscovery{
				GroupVersion: "folder.grafana.app/v1",
				Version:      "v1",
			},
		},
	}

	resources := []*metav1.APIResourceList{
		{
			GroupVersion: "dashboard.grafana.app/v1",
			APIResources: []metav1.APIResource{
				{
					Name:         "dashboards",
					SingularName: "dashboard",
					Kind:         "Dashboard",
				},
			},
		},
		{
			GroupVersion: "folder.grafana.app/v1",
			APIResources: []metav1.APIResource{
				{
					Name:         "folders",
					SingularName: "folder",
					Kind:         "Folder",
				},
			},
		},
	}

	return groups, resources
}

func getMultipleVersionsDiscovery() ([]*metav1.APIGroup, []*metav1.APIResourceList) {
	groups := []*metav1.APIGroup{
		{
			Name: "dashboard.grafana.app",
			Versions: []metav1.GroupVersionForDiscovery{
				{
					GroupVersion: "dashboard.grafana.app/v1",
					Version:      "v1",
				},
				{
					GroupVersion: "dashboard.grafana.app/v2",
					Version:      "v2",
				},
			},
			PreferredVersion: metav1.GroupVersionForDiscovery{
				GroupVersion: "dashboard.grafana.app/v2",
				Version:      "v2",
			},
		},
	}

	resources := []*metav1.APIResourceList{
		{
			GroupVersion: "dashboard.grafana.app/v1",
			APIResources: []metav1.APIResource{
				{
					Name:         "dashboards",
					SingularName: "dashboard",
					Kind:         "Dashboard",
				},
			},
		},
		{
			GroupVersion: "dashboard.grafana.app/v2",
			APIResources: []metav1.APIResource{
				{
					Name:         "dashboards",
					SingularName: "dashboard",
					Kind:         "Dashboard",
				},
			},
		},
	}

	return groups, resources
}

func getMixedVersionsDiscovery() ([]*metav1.APIGroup, []*metav1.APIResourceList) {
	groups := []*metav1.APIGroup{
		{
			Name: "dashboard.grafana.app",
			Versions: []metav1.GroupVersionForDiscovery{
				{
					GroupVersion: "dashboard.grafana.app/v1",
					Version:      "v1",
				},
				{
					GroupVersion: "dashboard.grafana.app/v2",
					Version:      "v2",
				},
			},
			PreferredVersion: metav1.GroupVersionForDiscovery{
				GroupVersion: "dashboard.grafana.app/v2",
				Version:      "v2",
			},
		},
		{
			Name: "folder.grafana.app",
			Versions: []metav1.GroupVersionForDiscovery{
				{
					GroupVersion: "folder.grafana.app/v1",
					Version:      "v1",
				},
			},
			PreferredVersion: metav1.GroupVersionForDiscovery{
				GroupVersion: "folder.grafana.app/v1",
				Version:      "v1",
			},
		},
	}

	resources := []*metav1.APIResourceList{
		{
			GroupVersion: "dashboard.grafana.app/v1",
			APIResources: []metav1.APIResource{
				{
					Name:         "dashboards",
					SingularName: "dashboard",
					Kind:         "Dashboard",
					Namespaced:   true,
				},
			},
		},
		{
			GroupVersion: "dashboard.grafana.app/v2",
			APIResources: []metav1.APIResource{
				{
					Name:         "dashboards",
					SingularName: "dashboard",
					Kind:         "Dashboard",
					Namespaced:   true,
				},
			},
		},
		{
			GroupVersion: "folder.grafana.app/v1",
			APIResources: []metav1.APIResource{
				{
					Name:         "folders",
					SingularName: "folder",
					Kind:         "Folder",
					Namespaced:   true,
				},
			},
		},
	}

	return groups, resources
}
