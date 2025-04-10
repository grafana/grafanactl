package resources

import (
	"errors"
	"fmt"
	"strings"
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
