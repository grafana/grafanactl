package dynamic

import (
	"context"
	"slices"

	"github.com/grafana/grafanactl/internal/config"
	"github.com/grafana/grafanactl/internal/resources"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/dynamic"
)

// NamespacedClient is a dynamic client with a namespace and a discovery registry.
type NamespacedClient struct {
	namespace string
	client    dynamic.Interface
}

// NewDefaultNamespacedClient creates a new namespaced dynamic client using the default discovery registry.
func NewDefaultNamespacedClient(cfg config.NamespacedRESTConfig) (*NamespacedClient, error) {
	client, err := dynamic.NewForConfig(&cfg.Config)
	if err != nil {
		return nil, err
	}

	return NewNamespacedClient(cfg.Namespace, client), nil
}

// NewNamespacedClient creates a new namespaced dynamic client.
func NewNamespacedClient(namespace string, client dynamic.Interface) *NamespacedClient {
	return &NamespacedClient{
		client:    client,
		namespace: namespace,
	}
}

// List lists resources from the server.
func (c *NamespacedClient) List(
	ctx context.Context, desc resources.Descriptor, opts metav1.ListOptions,
) (*unstructured.UnstructuredList, error) {
	return c.client.Resource(desc.GroupVersionResource()).Namespace(c.namespace).List(ctx, opts)
}

// GetMultiple gets multiple resources from the server.
//
// Kubernetes does not support getting multiple resources by name,
// so instead we list all resources and filter on the client side.
//
// Ideally we'd like to use field selectors,
// but Kubernetes does not support set-based operators in field selectors (only in labels).
func (c *NamespacedClient) GetMultiple(
	ctx context.Context, desc resources.Descriptor, names []string, opts metav1.ListOptions,
) ([]unstructured.Unstructured, error) {
	res, err := c.client.Resource(desc.GroupVersionResource()).Namespace(c.namespace).List(ctx, opts)
	if err != nil {
		return nil, err
	}

	filtered := make([]unstructured.Unstructured, 0, len(res.Items))
	for _, item := range res.Items {
		// TODO: worth using a map index for this?
		// (on small lists the performance difference is negligible)
		if slices.Contains(names, item.GetName()) {
			filtered = append(filtered, item)
		}
	}

	return filtered, nil
}

// Get gets a resource from the server.
func (c *NamespacedClient) Get(
	ctx context.Context, desc resources.Descriptor, name string, opts metav1.GetOptions,
) (*unstructured.Unstructured, error) {
	return c.client.Resource(desc.GroupVersionResource()).Namespace(c.namespace).Get(ctx, name, opts)
}

// Create creates a resource on the server.
func (c *NamespacedClient) Create(
	ctx context.Context, desc resources.Descriptor, obj *unstructured.Unstructured, opts metav1.CreateOptions,
) (*unstructured.Unstructured, error) {
	return c.client.Resource(desc.GroupVersionResource()).Namespace(c.namespace).Create(ctx, obj, opts)
}

// Update updates a resource on the server.
func (c *NamespacedClient) Update(
	ctx context.Context, desc resources.Descriptor, obj *unstructured.Unstructured, opts metav1.UpdateOptions,
) (*unstructured.Unstructured, error) {
	return c.client.Resource(desc.GroupVersionResource()).Namespace(c.namespace).Update(ctx, obj, opts)
}

// Delete deletes a resource on the server.
func (c *NamespacedClient) Delete(
	ctx context.Context, desc resources.Descriptor, name string, opts metav1.DeleteOptions,
) error {
	return c.client.Resource(desc.GroupVersionResource()).Namespace(c.namespace).Delete(ctx, name, opts)
}

// Apply applies a resource on the server.
func (c *NamespacedClient) Apply(
	ctx context.Context, desc resources.Descriptor, name string, obj *unstructured.Unstructured, opts metav1.ApplyOptions,
) (*unstructured.Unstructured, error) {
	return c.client.Resource(desc.GroupVersionResource()).Namespace(c.namespace).Apply(ctx, name, obj, opts)
}
