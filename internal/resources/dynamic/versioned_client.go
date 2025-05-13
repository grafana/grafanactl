package dynamic

import (
	"context"

	"github.com/grafana/grafana/pkg/apimachinery/utils"
	"github.com/grafana/grafanactl/internal/config"
	"github.com/grafana/grafanactl/internal/resources"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// VersionedClient is a dynamic client that supports automatic version switching.
// It will automatically switch to the correct version of the resource based on the stored version.
type VersionedClient struct {
	*NamespacedClient
}

// NewDefaultVersionedClient creates a new versioned client using the default namespaced client.
func NewDefaultVersionedClient(cfg config.NamespacedRESTConfig) (*VersionedClient, error) {
	client, err := NewDefaultNamespacedClient(cfg)
	if err != nil {
		return nil, err
	}

	return WrapNamespacedClient(client), nil
}

// WrapNamespacedClient wraps a namespaced client in a versioned client.
func WrapNamespacedClient(client *NamespacedClient) *VersionedClient {
	return &VersionedClient{
		NamespacedClient: client,
	}
}

// List lists resources from the server.
// It will automatically re-fetch resources which need to be fetched using the stored version.
func (c *VersionedClient) List(
	ctx context.Context, desc resources.Descriptor, opts metav1.ListOptions,
) (*unstructured.UnstructuredList, error) {
	list, err := c.NamespacedClient.List(ctx, desc, opts)
	if err != nil {
		return nil, err
	}

	res, err := c.getMultipleCorrectVersions(ctx, desc, list.Items, opts)
	if err != nil {
		return nil, err
	}

	// Replace the items in the original list with the new ones and return it.
	list.Items = res
	return list, nil
}

// GetMultiple gets multiple resources from the server.
// It will automatically re-fetch resources which need to be fetched using the stored version.
func (c *VersionedClient) GetMultiple(
	ctx context.Context, desc resources.Descriptor, names []string, opts metav1.ListOptions,
) ([]unstructured.Unstructured, error) {
	objs, err := c.NamespacedClient.GetMultiple(ctx, desc, names, opts)
	if err != nil {
		return nil, err
	}

	return c.getMultipleCorrectVersions(ctx, desc, objs, opts)
}

// Get gets a resource from the server.
func (c *VersionedClient) Get(
	ctx context.Context, desc resources.Descriptor, name string, opts metav1.GetOptions,
) (*unstructured.Unstructured, error) {
	obj, err := c.NamespacedClient.Get(ctx, desc, name, opts)
	if err != nil {
		return nil, err
	}

	storedVersion, err := getStoredVersion(obj)
	if err != nil {
		return nil, err
	}

	if storedVersion == "" {
		return obj, nil
	}

	newdesc := desc
	newdesc.GroupVersion.Version = storedVersion

	return c.NamespacedClient.Get(ctx, newdesc, name, opts)
}

func (c *VersionedClient) getMultipleCorrectVersions(
	ctx context.Context, desc resources.Descriptor, src []unstructured.Unstructured, opts metav1.ListOptions,
) ([]unstructured.Unstructured, error) {
	res := make([]unstructured.Unstructured, 0, len(src))
	versioned := make(map[resources.Descriptor][]string)

	// Iterate over all objects and check if they need to be re-fetched using the stored version.
	// Group the objects by the new descriptor.
	for _, obj := range src {
		storedVersion, err := getStoredVersion(&obj)
		if err != nil {
			return nil, err
		}

		if storedVersion == "" {
			res = append(res, obj)
			continue
		}

		newdesc := desc
		newdesc.GroupVersion.Version = storedVersion
		if _, ok := versioned[newdesc]; !ok {
			versioned[newdesc] = make([]string, 0, len(src))
		}
		versioned[newdesc] = append(versioned[newdesc], obj.GetName())
	}

	// If there are no versioned objects, we can return the original list.
	if len(versioned) == 0 {
		return src, nil
	}

	// Iterate over all descriptors we need to re-fetch,
	// fetch them using GetMultiple and append the results to the result list.
	for desc, names := range versioned {
		objs, err := c.GetMultiple(ctx, desc, names, opts)
		if err != nil {
			return nil, err
		}

		res = append(res, objs...)
	}

	return res, nil
}

func getStoredVersion(obj *unstructured.Unstructured) (string, error) {
	acc, err := utils.MetaAccessor(obj)
	if err != nil {
		return "", err
	}

	stat, err := acc.GetStatus()
	if err != nil {
		return "", nil
	}

	statm, ok := stat.(map[string]any)
	if !ok {
		return "", nil
	}

	conv, ok := statm["conversion"]
	if !ok {
		return "", nil
	}

	convm, ok := conv.(map[string]any)
	if !ok {
		return "", nil
	}

	v, ok := convm["storedVersion"]
	if !ok {
		return "", nil
	}

	storedVersion, ok := v.(string)
	if !ok {
		return "", nil
	}

	return storedVersion, nil
}
