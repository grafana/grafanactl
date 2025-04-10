package resources

import (
	"slices"
	"strings"
)

const (
	// FilterTypeUnknown is the default fallback value for a selector.
	FilterTypeUnknown FilterType = iota

	// FilterTypeAll is the selector is to select all resources of a type.
	FilterTypeAll

	// FilterTypeMultiple is the selector is to select multiple resources by UID.
	FilterTypeMultiple

	// FilterTypeSingle is the selector is to select a single resource by UID.
	FilterTypeSingle
)

// FilterType is the type of a resource selector.
// It identifies whether the selector needs to select all resources of a type,
// select a single resource by UID, or select multiple resources by UID.
type FilterType int

func (t FilterType) String() string {
	switch t {
	case FilterTypeAll:
		return "GetAll"
	case FilterTypeMultiple:
		return "GetMultiple"
	case FilterTypeSingle:
		return "GetSingle"
	default:
		return "Unknown"
	}
}

// Filter is a filter to filter resources from Grafana.
// Unlike Selector, filters use the Descriptor to identify the resource type,
// which fully defines the target API resource.
type Filter struct {
	Type         FilterType
	Descriptor   Descriptor
	ResourceUIDs []string
}

func (f Filter) String() string {
	var sb strings.Builder

	sb.WriteString(f.Type.String())
	sb.WriteString(":")
	sb.WriteString(f.Descriptor.String())

	if len(f.ResourceUIDs) > 0 {
		sb.WriteString("/")
		sb.WriteString(strings.Join(f.ResourceUIDs, ","))
	}

	return sb.String()
}

// Matches returns true if the filter matches the resource.
func (f Filter) Matches(res Resource) bool {
	if !f.Descriptor.Matches(res.GroupVersionKind()) {
		return false
	}

	switch f.Type {
	case FilterTypeAll:
		return true
	case FilterTypeMultiple, FilterTypeSingle:
		return slices.Contains(f.ResourceUIDs, res.Name())
	}

	return false
}

// Filters is a list of filters.
type Filters []Filter

// IsEmpty returns true if the filters are empty.
func (f Filters) IsEmpty() bool {
	return len(f) == 0
}

// Matches returns true if any of the filters matches the resource.
func (f Filters) Matches(res Resource) bool {
	// Empty filters match all resources.
	if f.IsEmpty() {
		return true
	}

	for _, filter := range f {
		if filter.Matches(res) {
			return true
		}
	}

	return false
}
