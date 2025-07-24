package remote_test

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"

	"github.com/grafana/grafanactl/internal/resources"
	"github.com/grafana/grafanactl/internal/resources/remote"
	"github.com/stretchr/testify/require"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func TestPusher_Push_FoldersFirst(t *testing.T) {
	req := require.New(t)

	// Create test resources: 2 folders and 2 dashboards
	testResources := createTestResources()

	// Mock client that records the order of operations
	mockClient := &mockPushClient{
		operations: []string{},
		mu:         sync.Mutex{},
	}

	// Mock registry that supports all test resources
	mockRegistry := &mockPushRegistry{
		supportedResources: []resources.Descriptor{
			{
				GroupVersion: schema.GroupVersion{Group: "folder.grafana.app", Version: "v1"},
				Kind:         "Folder",
				Singular:     "folder",
				Plural:       "folders",
			},
			{
				GroupVersion: schema.GroupVersion{Group: "dashboard.grafana.app", Version: "v1"},
				Kind:         "Dashboard",
				Singular:     "dashboard",
				Plural:       "dashboards",
			},
		},
	}

	pusher := remote.NewPusher(mockClient, mockRegistry)

	// Push resources
	summary, err := pusher.Push(t.Context(), remote.PushRequest{
		Resources:      testResources,
		MaxConcurrency: 2,
		IncludeManaged: true, // Include managed resources
	})

	req.NoError(err)
	req.Equal(4, summary.PushedCount)
	req.Equal(0, summary.FailedCount)

	// Verify that all folders were pushed before all dashboards
	req.Len(mockClient.operations, 4)

	// Extract resource names and kinds from operations
	var folderOps, dashboardOps []string
	for _, op := range mockClient.operations {
		if contains(op, "folder-1") || contains(op, "folder-2") {
			folderOps = append(folderOps, op)
		} else if contains(op, "dashboard-1") || contains(op, "dashboard-2") {
			dashboardOps = append(dashboardOps, op)
		}
	}

	req.Len(folderOps, 2, "Should have 2 folder operations")
	req.Len(dashboardOps, 2, "Should have 2 dashboard operations")

	// Find the index of the last folder operation and first dashboard operation
	lastFolderIndex := -1
	firstDashboardIndex := len(mockClient.operations)

	for i, op := range mockClient.operations {
		if contains(op, "folder-1") || contains(op, "folder-2") {
			if i > lastFolderIndex {
				lastFolderIndex = i
			}
		}
		if (contains(op, "dashboard-1") || contains(op, "dashboard-2")) && i < firstDashboardIndex {
			firstDashboardIndex = i
		}
	}

	req.Less(lastFolderIndex, firstDashboardIndex,
		"All folders should be pushed before any dashboard. Last folder at index %d, first dashboard at index %d",
		lastFolderIndex, firstDashboardIndex)
}

func TestPusher_Push_OnlyFolders(t *testing.T) {
	req := require.New(t)

	// Create resources with only folders
	testResources := resources.NewResources(
		createFolderResource("folder-1", "v1"),
		createFolderResource("folder-2", "v0alpha1"),
	)

	mockClient := &mockPushClient{
		operations: []string{},
		mu:         sync.Mutex{},
	}

	mockRegistry := &mockPushRegistry{
		supportedResources: []resources.Descriptor{
			{
				GroupVersion: schema.GroupVersion{Group: "folder.grafana.app", Version: "v1"},
				Kind:         "Folder",
				Singular:     "folder",
				Plural:       "folders",
			},
			{
				GroupVersion: schema.GroupVersion{Group: "folder.grafana.app", Version: "v0alpha1"},
				Kind:         "Folder",
				Singular:     "folder",
				Plural:       "folders",
			},
		},
	}

	pusher := remote.NewPusher(mockClient, mockRegistry)

	summary, err := pusher.Push(t.Context(), remote.PushRequest{
		Resources:      testResources,
		MaxConcurrency: 2,
		IncludeManaged: true, // Include managed resources
	})

	req.NoError(err)
	req.Equal(2, summary.PushedCount)
	req.Equal(0, summary.FailedCount)
	req.Len(mockClient.operations, 2)
}

func TestPusher_Push_OnlyDashboards(t *testing.T) {
	req := require.New(t)

	// Create resources with only dashboards
	testResources := resources.NewResources(
		createDashboardResource("dashboard-1"),
		createDashboardResource("dashboard-2"),
	)

	mockClient := &mockPushClient{
		operations: []string{},
		mu:         sync.Mutex{},
	}

	mockRegistry := &mockPushRegistry{
		supportedResources: []resources.Descriptor{
			{
				GroupVersion: schema.GroupVersion{Group: "dashboard.grafana.app", Version: "v1"},
				Kind:         "Dashboard",
				Singular:     "dashboard",
				Plural:       "dashboards",
			},
		},
	}

	pusher := remote.NewPusher(mockClient, mockRegistry)

	summary, err := pusher.Push(t.Context(), remote.PushRequest{
		Resources:      testResources,
		MaxConcurrency: 2,
		IncludeManaged: true, // Include managed resources
	})

	req.NoError(err)
	req.Equal(2, summary.PushedCount)
	req.Equal(0, summary.FailedCount)
	req.Len(mockClient.operations, 2)
}

func TestPusher_Push_EmptyResources(t *testing.T) {
	req := require.New(t)

	testResources := resources.NewResources()

	mockClient := &mockPushClient{
		operations: []string{},
		mu:         sync.Mutex{},
	}

	mockRegistry := &mockPushRegistry{
		supportedResources: []resources.Descriptor{},
	}

	pusher := remote.NewPusher(mockClient, mockRegistry)

	summary, err := pusher.Push(t.Context(), remote.PushRequest{
		Resources:      testResources,
		MaxConcurrency: 2,
	})

	req.NoError(err)
	req.Equal(0, summary.PushedCount)
	req.Equal(0, summary.FailedCount)
	req.Empty(mockClient.operations)
}

func TestPusher_Push_FolderCreationError(t *testing.T) {
	req := require.New(t)

	testResources := createTestResources()

	mockClient := &mockPushClient{
		operations:   []string{},
		mu:           sync.Mutex{},
		shouldFail:   map[string]bool{"folder-1": true},
		failureError: errors.New("folder creation failed"),
	}

	mockRegistry := &mockPushRegistry{
		supportedResources: []resources.Descriptor{
			{
				GroupVersion: schema.GroupVersion{Group: "folder.grafana.app", Version: "v1"},
				Kind:         "Folder",
				Singular:     "folder",
				Plural:       "folders",
			},
			{
				GroupVersion: schema.GroupVersion{Group: "dashboard.grafana.app", Version: "v1"},
				Kind:         "Dashboard",
				Singular:     "dashboard",
				Plural:       "dashboards",
			},
		},
	}

	pusher := remote.NewPusher(mockClient, mockRegistry)

	summary, err := pusher.Push(t.Context(), remote.PushRequest{
		Resources:      testResources,
		MaxConcurrency: 2,
		StopOnError:    false,
		IncludeManaged: true, // Include managed resources
	})

	req.NoError(err)
	req.Equal(3, summary.PushedCount) // 1 folder + 2 dashboards succeeded
	req.Equal(1, summary.FailedCount) // 1 folder failed
	req.Len(summary.Failures, 1)
	req.Equal("folder-1", summary.Failures[0].Resource.Name())
}

// Helper functions

func createTestResources() *resources.Resources {
	return resources.NewResources(
		createFolderResource("folder-1", "v1"),
		createFolderResource("folder-2", "v1"),
		createDashboardResource("dashboard-1"),
		createDashboardResource("dashboard-2"),
	)
}

func createFolderResource(name, version string) *resources.Resource {
	return resources.MustFromObject(map[string]any{
		"apiVersion": "folder.grafana.app/" + version,
		"kind":       "Folder",
		"metadata": map[string]any{
			"name":      name,
			"namespace": "default",
		},
		"spec": map[string]any{
			"title": "Test Folder " + name,
		},
	}, resources.SourceInfo{})
}

func createDashboardResource(name string) *resources.Resource {
	return resources.MustFromObject(map[string]any{
		"apiVersion": "dashboard.grafana.app/v1",
		"kind":       "Dashboard",
		"metadata": map[string]any{
			"name":      name,
			"namespace": "default",
		},
		"spec": map[string]any{
			"title": "Test Dashboard " + name,
		},
	}, resources.SourceInfo{})
}

func contains(s, substr string) bool {
	return strings.Contains(s, substr)
}

// Mock implementations

type mockPushClient struct {
	operations   []string
	mu           sync.Mutex
	shouldFail   map[string]bool
	failureError error
}

func (m *mockPushClient) Create(
	_ context.Context, _ resources.Descriptor, obj *unstructured.Unstructured, _ metav1.CreateOptions,
) (*unstructured.Unstructured, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	name := obj.GetName()
	m.operations = append(m.operations, "create-"+name)

	if m.shouldFail != nil && m.shouldFail[name] {
		return nil, m.failureError
	}

	return obj, nil
}

func (m *mockPushClient) Update(
	_ context.Context, _ resources.Descriptor, obj *unstructured.Unstructured, _ metav1.UpdateOptions,
) (*unstructured.Unstructured, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	name := obj.GetName()
	m.operations = append(m.operations, "update-"+name)

	if m.shouldFail != nil && m.shouldFail[name] {
		return nil, m.failureError
	}

	return obj, nil
}

func (m *mockPushClient) Get(
	_ context.Context, desc resources.Descriptor, name string, _ metav1.GetOptions,
) (*unstructured.Unstructured, error) {
	// Simulate resource not found to trigger Create operations
	return nil, apierrors.NewNotFound(desc.GroupVersionResource().GroupResource(), name)
}

type mockPushRegistry struct {
	supportedResources []resources.Descriptor
}

func (m *mockPushRegistry) SupportedResources() resources.Descriptors {
	return m.supportedResources
}
