package resources

import (
	"fmt"

	"k8s.io/apimachinery/pkg/runtime/schema"
)

// Descriptors is a list of descriptors.
type Descriptors []Descriptor

// Descriptor describes a resource.
type Descriptor struct {
	GroupVersion schema.GroupVersion
	Kind         string
	Singular     string
	Plural       string
}

func (d Descriptor) String() string {
	return fmt.Sprintf("%s.%s.%s", d.Plural, d.GroupVersion.Version, d.GroupVersion.Group)
}

// GroupVersionKind returns the GroupVersionKind for the descriptor.
func (d Descriptor) GroupVersionKind() schema.GroupVersionKind {
	return schema.GroupVersionKind{
		Group:   d.GroupVersion.Group,
		Version: d.GroupVersion.Version,
		Kind:    d.Kind,
	}
}

// GroupVersionResource returns the GroupVersionResource for the descriptor.
func (d Descriptor) GroupVersionResource() schema.GroupVersionResource {
	return schema.GroupVersionResource{
		Group:    d.GroupVersion.Group,
		Version:  d.GroupVersion.Version,
		Resource: d.Plural,
	}
}

// Matches returns true if the descriptor matches the given GroupVersionKind.
func (d Descriptor) Matches(gvk schema.GroupVersionKind) bool {
	return d.GroupVersion == gvk.GroupVersion() &&
		d.Kind == gvk.Kind
}
