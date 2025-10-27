package remote_test

import (
	"context"
	"sync"
	"testing"

	"github.com/grafana/grafanactl/internal/resources"
	"github.com/grafana/grafanactl/internal/resources/remote"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func TestDeleter_Delete_DryRunSkipsAPICall(t *testing.T) {
	tests := []struct {
		name                string
		dryRun              bool
		expectDeleteAPICAll bool
		expectedDeleted     int
		expectedFailed      int
	}{
		{
			name:                "dry-run enabled skips actual delete",
			dryRun:              true,
			expectDeleteAPICAll: false,
			expectedDeleted:     2,
			expectedFailed:      0,
		},
		{
			name:                "dry-run disabled calls delete API",
			dryRun:              false,
			expectDeleteAPICAll: true,
			expectedDeleted:     2,
			expectedFailed:      0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := require.New(t)

			// Create test resources
			testResources := resources.NewResources(
				createFolderResource("folder-1", "v1"),
				createDashboardResource("dashboard-1"),
			)

			// Mock client that tracks delete calls
			mockClient := &mockDeleteClient{
				deleteCalls: []string{},
				mu:          sync.Mutex{},
			}

			// Mock registry that supports all test resources
			mockRegistry := &mockRegistry{
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

			deleter := remote.NewDeleter(mockClient, mockRegistry)

			// Delete resources
			summary, err := deleter.Delete(t.Context(), remote.DeleteRequest{
				Resources:      testResources,
				MaxConcurrency: 2,
				DryRun:         tt.dryRun,
			})

			req.NoError(err)
			req.Equal(tt.expectedDeleted, summary.DeletedCount)
			req.Equal(tt.expectedFailed, summary.FailedCount)

			// Verify delete API calls
			if tt.expectDeleteAPICAll {
				req.Len(mockClient.deleteCalls, 2, "Should have made 2 delete API calls")
			} else {
				req.Empty(mockClient.deleteCalls, "Should not have made any delete API calls in dry-run mode")
			}
		})
	}
}

func TestDeleter_Delete_EmptyResources(t *testing.T) {
	req := require.New(t)

	testResources := resources.NewResources()

	mockClient := &mockDeleteClient{
		deleteCalls: []string{},
		mu:          sync.Mutex{},
	}

	mockRegistry := &mockRegistry{
		supportedResources: []resources.Descriptor{},
	}

	deleter := remote.NewDeleter(mockClient, mockRegistry)

	summary, err := deleter.Delete(t.Context(), remote.DeleteRequest{
		Resources:      testResources,
		MaxConcurrency: 2,
		DryRun:         false,
	})

	req.NoError(err)
	req.Equal(0, summary.DeletedCount)
	req.Equal(0, summary.FailedCount)
	req.Empty(mockClient.deleteCalls)
}

func TestDeleter_Delete_UnsupportedResource(t *testing.T) {
	req := require.New(t)

	// Create a resource with an unsupported GVK
	testResources := resources.NewResources(
		createFolderResource("folder-1", "v1"),
	)

	mockClient := &mockDeleteClient{
		deleteCalls: []string{},
		mu:          sync.Mutex{},
	}

	// Registry that doesn't support folders
	mockRegistry := &mockRegistry{
		supportedResources: []resources.Descriptor{},
	}

	deleter := remote.NewDeleter(mockClient, mockRegistry)

	summary, err := deleter.Delete(t.Context(), remote.DeleteRequest{
		Resources:      testResources,
		MaxConcurrency: 2,
		DryRun:         false,
		StopOnError:    false, // Don't stop on error
	})

	req.NoError(err)
	req.Equal(0, summary.DeletedCount, "Should not delete unsupported resources")
	req.Equal(0, summary.FailedCount, "Should skip unsupported resources, not fail")
	req.Empty(mockClient.deleteCalls)
}

// Mock implementations

type mockDeleteClient struct {
	deleteCalls []string
	mu          sync.Mutex
}

func (m *mockDeleteClient) Delete(
	_ context.Context, desc resources.Descriptor, name string, _ metav1.DeleteOptions,
) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.deleteCalls = append(m.deleteCalls, desc.Kind+"/"+name)
	return nil
}

type mockRegistry struct {
	supportedResources []resources.Descriptor
}

func (m *mockRegistry) SupportedResources() resources.Descriptors {
	return m.supportedResources
}
