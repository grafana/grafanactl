package resources

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"path/filepath"
	"strings"

	"github.com/grafana/grafana/pkg/apimachinery/utils"
	"golang.org/x/sync/errgroup"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// GroupVersionKind is a group, version, and kind,
// which can be used to identify a resource.
// Not all fields are required to be set.
// It is expected that anything that accepts a GroupVersionKind
// will handle the discovery of the resource based on the fields that are present.
type GroupVersionKind struct {
	Group   string
	Version string
	Kind    string
	// TODO: this is a quick hack to get the proper kind name without renaming everything.
	// Once we refactor discovery this can be removed.
	Name string
}

func (gvk GroupVersionKind) String() string {
	// TODO: handle empty version and group
	return fmt.Sprintf("%s.%s.%s", gvk.Kind, gvk.Version, gvk.Group)
}

// ParseGVK parses a GVK string into a DynamicGroupVersionKind.
func ParseGVK(gvk string) (GroupVersionKind, error) {
	parts := strings.SplitN(gvk, ".", 3)

	var res GroupVersionKind
	switch len(parts) {
	case 2:
		if len(parts[0]) == 0 {
			return GroupVersionKind{}, errors.New("must specify kind")
		}

		if len(parts[1]) == 0 {
			return GroupVersionKind{}, errors.New("must specify group")
		}

		res = GroupVersionKind{
			Group:   parts[1],
			Version: "", // Default version
			Kind:    parts[0],
		}
	case 3:
		if len(parts[0]) == 0 {
			return GroupVersionKind{}, errors.New("must specify kind")
		}

		if len(parts[1]) == 0 {
			return GroupVersionKind{}, errors.New("must specify version")
		}

		if len(parts[2]) == 0 {
			return GroupVersionKind{}, errors.New("must specify group")
		}

		res = GroupVersionKind{
			Group:   parts[2],
			Version: parts[1],
			Kind:    parts[0],
		}
	default:
		res = GroupVersionKind{
			Group:   "", // Default group
			Version: "", // Default version
			Kind:    parts[0],
		}
	}

	return res, nil
}

type ResourceRef string

type Resource struct {
	Raw utils.GrafanaMetaAccessor
}

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

func (r Resource) Ref() ResourceRef {
	return ResourceRef(fmt.Sprintf("%s/%s-%s", r.GroupVersionKind().String(), r.Namespace(), r.Name()))
}

func (r Resource) GroupVersionKind() GroupVersionKind {
	return GroupVersionKind{
		Group:   r.Group(),
		Version: r.Version(),
		Kind:    r.Kind(),
	}
}

func (r Resource) Namespace() string {
	return r.Raw.GetNamespace()
}

func (r Resource) Name() string {
	return r.Raw.GetName()
}

func (r Resource) Group() string {
	return r.Raw.GetGroupVersionKind().Group
}

func (r Resource) Kind() string {
	return r.Raw.GetGroupVersionKind().Kind
}

func (r Resource) Version() string {
	return r.Raw.GetGroupVersionKind().Version
}

func (r Resource) APIVersion() string {
	return r.Group() + "/" + r.Version()
}

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

type Resources struct {
	collection    map[ResourceRef]*Resource
	onChangeFuncs []func(resource *Resource)
}

func NewResources(resources ...*Resource) *Resources {
	r := &Resources{
		collection: make(map[ResourceRef]*Resource),
	}

	r.Add(resources...)

	return r
}

func (r *Resources) Add(resources ...*Resource) {
	for _, resource := range resources {
		r.collection[resource.Ref()] = resource

		for _, cb := range r.onChangeFuncs {
			cb(resource)
		}
	}
}

func (r *Resources) OnChange(callback func(resource *Resource)) {
	r.onChangeFuncs = append(r.onChangeFuncs, callback)
}

// TODO: kind + name isn't enough to unambiguously identify a resource.
func (r *Resources) Find(kind string, name string) (*Resource, bool) {
	for _, resource := range r.collection {
		if resource.Kind() == kind && resource.Name() == name {
			return resource, true
		}
	}

	return nil, false
}

func (r *Resources) Merge(resources *Resources) {
	_ = resources.ForEach(func(resource *Resource) error {
		r.Add(resource)
		return nil
	})
}

func (r *Resources) ForEach(callback func(*Resource) error) error {
	for _, resource := range r.collection {
		if err := callback(resource); err != nil {
			return err
		}
	}

	return nil
}

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

func (r *Resources) Len() int {
	return len(r.collection)
}

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
