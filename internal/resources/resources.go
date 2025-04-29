package resources

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"path/filepath"
	"slices"
	"strings"

	"github.com/grafana/grafana/pkg/apimachinery/utils"
	"golang.org/x/sync/errgroup"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// ResourceRef is a unique identifier for a resource.
type ResourceRef string

// Resource is a resource in the Grafana API.
type Resource struct {
	Raw utils.GrafanaMetaAccessor
}

// MustFromObject creates a new Resource from an object.
// If the object is not a valid Grafana resource, it will panic.
func MustFromObject(obj map[string]any) *Resource {
	return MustFromUnstructured(&unstructured.Unstructured{Object: obj})
}

// MustFromUnstructured creates a new Resource from an unstructured object.
// If the object is not a valid Grafana resource, it will panic.
func MustFromUnstructured(obj *unstructured.Unstructured) *Resource {
	r, err := FromUnstructured(obj)
	if err != nil {
		panic(err)
	}
	return r
}

// FromUnstructured creates a new Resource from an unstructured object.
func FromUnstructured(obj *unstructured.Unstructured) (*Resource, error) {
	metaAccessor, err := utils.MetaAccessor(obj)
	if err != nil {
		return nil, err
	}
	return &Resource{Raw: metaAccessor}, nil
}

// ToUnstructured converts the resource to an unstructured object.
func (r Resource) ToUnstructured() (*unstructured.Unstructured, error) {
	runtimeObj, ok := r.Raw.GetRuntimeObject()
	if !ok {
		return nil, errors.New("failed converting resource to runtime object")
	}
	unstructuredObj, ok := runtimeObj.(*unstructured.Unstructured)
	if !ok {
		return nil, errors.New("failed converting resource to unstructured object")
	}

	return unstructuredObj, nil
}

// Ref returns a unique identifier for the resource.
func (r Resource) Ref() ResourceRef {
	return ResourceRef(
		fmt.Sprintf("%s/%s-%s", r.GroupVersionKind().String(), r.Namespace(), r.Name()),
	)
}

// GroupVersionKind returns the GroupVersionKind of the resource.
func (r Resource) GroupVersionKind() schema.GroupVersionKind {
	return r.Raw.GetGroupVersionKind()
}

// Namespace returns the namespace of the resource.
func (r Resource) Namespace() string {
	return r.Raw.GetNamespace()
}

// Name returns the name of the resource.
func (r Resource) Name() string {
	return r.Raw.GetName()
}

// Group returns the group of the resource.
func (r Resource) Group() string {
	return r.Raw.GetGroupVersionKind().Group
}

// Kind returns the kind of the resource.
func (r Resource) Kind() string {
	return r.Raw.GetGroupVersionKind().Kind
}

// Version returns the version of the resource.
func (r Resource) Version() string {
	return r.Raw.GetGroupVersionKind().Version
}

// APIVersion returns the API version of the resource.
func (r Resource) APIVersion() string {
	return r.Group() + "/" + r.Version()
}

// SourcePath returns the source path of the resource.
func (r Resource) SourcePath() string {
	properties, _ := r.Raw.GetSourceProperties()
	if properties.Path == "" {
		return ""
	}

	u, err := url.Parse(properties.Path)
	if err != nil {
		return ""
	}

	return filepath.Join(u.Host, u.Path)
}

// SourceFormat returns the source format of the resource.
func (r Resource) SourceFormat() string {
	properties, _ := r.Raw.GetSourceProperties()
	if properties.Path == "" {
		return ""
	}

	u, err := url.Parse(properties.Path)
	if err != nil {
		return ""
	}

	return u.Scheme
}

// Resources is a collection of resources.
type Resources struct {
	collection    map[ResourceRef]*Resource
	onChangeFuncs []func(resource *Resource)
}

// NewResources creates a new Resources collection.
func NewResources(resources ...*Resource) *Resources {
	r := MakeResources(len(resources))
	r.Add(resources...)
	return r
}

// MakeResources makes a new empty Resources collection of the given size.
func MakeResources(size int) *Resources {
	return &Resources{
		collection: make(map[ResourceRef]*Resource, size),
	}
}

// NewResourcesFromUnstructured creates a new Resources collection from an unstructured list.
func NewResourcesFromUnstructured(resources unstructured.UnstructuredList) (*Resources, error) {
	if len(resources.Items) == 0 {
		return NewResources(), nil
	}

	list := make([]*Resource, 0, len(resources.Items))
	for i := range resources.Items {
		r, err := FromUnstructured(&resources.Items[i])
		if err != nil {
			return nil, err
		}

		list = append(list, r)
	}

	return NewResources(list...), nil
}

// Clear removes all resources from the collection by resetting the underlying map.
// The new map will have the same capacity as the old one.
func (r *Resources) Clear() {
	r.collection = make(map[ResourceRef]*Resource, len(r.collection))
}

// Add adds resources to the collection.
func (r *Resources) Add(resources ...*Resource) {
	for _, resource := range resources {
		r.collection[resource.Ref()] = resource

		for _, cb := range r.onChangeFuncs {
			cb(resource)
		}
	}
}

// OnChange adds a callback that will be called when a resource is added to the collection.
func (r *Resources) OnChange(callback func(resource *Resource)) {
	r.onChangeFuncs = append(r.onChangeFuncs, callback)
}

// Find finds a resource by kind and name.
// TODO: kind + name isn't enough to unambiguously identify a resource.
func (r *Resources) Find(kind string, name string) (*Resource, bool) {
	for _, resource := range r.collection {
		if resource.Kind() == kind && resource.Name() == name {
			return resource, true
		}
	}

	return nil, false
}

// Merge merges another resources collection into the current one.
func (r *Resources) Merge(resources *Resources) {
	_ = resources.ForEach(func(resource *Resource) error {
		r.Add(resource)
		return nil
	})
}

// ForEach iterates over all resources in the collection and calls the callback for each resource.
func (r *Resources) ForEach(callback func(*Resource) error) error {
	for _, resource := range r.collection {
		if err := callback(resource); err != nil {
			return err
		}
	}

	return nil
}

// ForEachConcurrently iterates over all resources in the collection and calls the callback for each resource.
// The callback is called concurrently, up to maxInflight at a time.
func (r *Resources) ForEachConcurrently(
	ctx context.Context, maxInflight int, callback func(context.Context, *Resource) error,
) error {
	g, ctx := errgroup.WithContext(ctx)
	g.SetLimit(maxInflight)

	for _, resource := range r.collection {
		g.Go(func() error {
			return callback(ctx, resource)
		})
	}

	return g.Wait()
}

// Len returns the number of resources in the collection.
func (r *Resources) Len() int {
	return len(r.collection)
}

// AsList returns a list of resources from the collection.
func (r *Resources) AsList() []*Resource {
	if r.collection == nil {
		return nil
	}

	list := make([]*Resource, 0, r.Len())
	for _, resource := range r.collection {
		list = append(list, resource)
	}

	return list
}

// GroupByKind groups resources by kind.
func (r *Resources) GroupByKind() map[string]*Resources {
	resourceByKind := map[string]*Resources{}
	_ = r.ForEach(func(resource *Resource) error {
		if _, ok := resourceByKind[resource.Kind()]; !ok {
			resourceByKind[resource.Kind()] = NewResources()
		}

		resourceByKind[resource.Kind()].Add(resource)
		return nil
	})

	return resourceByKind
}

// ToUnstructuredList converts the resources to an unstructured list.
func (r *Resources) ToUnstructuredList() unstructured.UnstructuredList {
	res := unstructured.UnstructuredList{
		Items: make([]unstructured.Unstructured, 0, r.Len()),
	}

	if err := r.ForEach(func(r *Resource) error {
		obj, err := r.ToUnstructured()
		if err != nil {
			return err
		}
		res.Items = append(res.Items, *obj)
		return nil
	}); err != nil {
		return unstructured.UnstructuredList{}
	}

	return res
}

// SortUnstructured sorts a list of unstructured objects by group, version, kind, and name.
func SortUnstructured(items []unstructured.Unstructured) {
	slices.SortStableFunc(items, func(a, b unstructured.Unstructured) int {
		gva := a.GroupVersionKind()
		gvb := b.GroupVersionKind()

		res := strings.Compare(gva.Group, gvb.Group)
		if res != 0 {
			return res
		}

		res = strings.Compare(gva.Version, gvb.Version)
		if res != 0 {
			return res
		}

		res = strings.Compare(gva.Kind, gvb.Kind)
		if res != 0 {
			return res
		}

		return strings.Compare(a.GetName(), b.GetName())
	})
}
