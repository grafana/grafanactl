package resources

import (
	"errors"
	"fmt"
	"strings"
)

// SelectorType is the type of a resource selector.
// It identifies whether the selector needs to select all resources of a type,
// select a single resource by UID, or select multiple resources by UID.
type SelectorType int

const (
	// SelectorTypeUnknown is the default fallback value for a selector.
	SelectorTypeUnknown SelectorType = iota

	// SelectorTypeAll is the selector is to select all resources of a type.
	SelectorTypeAll

	// SelectorTypeMultiple is the selector is to select multiple resources by UID.
	SelectorTypeMultiple

	// SelectorTypeSingle is the selector is to select a single resource by UID.
	SelectorTypeSingle
)

// Selectors is a list of resource selectors.
type Selectors []Selector

// IsSingleTarget returns true if the selector is to get a single resource.
func (s Selectors) IsSingleTarget() bool {
	if len(s) != 1 {
		return false
	}

	return s[0].SelectorType == SelectorTypeSingle
}

// Selector is a selector to select a resource from Grafana.
type Selector struct {
	SelectorType     SelectorType
	GroupVersionKind GroupVersionKind
	ResourceUIDs     []string
}

func (sel Selector) String() string {
	cmd := sel.GroupVersionKind.String()
	if len(sel.ResourceUIDs) > 0 {
		cmd += "/" + strings.Join(sel.ResourceUIDs, ",")
	}

	return cmd
}

// InvalidSelectorError is an error that occurs when a command is invalid.
type InvalidSelectorError struct {
	Command string
	Err     string
}

func (e InvalidSelectorError) Error() string {
	return fmt.Sprintf("invalid command '%s': %s", e.Command, e.Err)
}

// ParseSelectors parses a list of resource selector strings into a list of Selectors.
func ParseSelectors(sels []string) (Selectors, error) {
	if len(sels) == 0 {
		return []Selector{}, nil
	}

	res := make([]Selector, len(sels))

	for i, cmd := range sels {
		if err := ParseResourceSelector(cmd, &res[i]); err != nil {
			return nil, err
		}
	}

	return res, nil
}

// ParseResourceSelector parses a resource selector string into a Selector.
func ParseResourceSelector(sel string, dst *Selector) error {
	parts := strings.Split(sel, "/")

	switch len(parts) {
	case 0:
		return InvalidSelectorError{Command: sel, Err: "missing resource type"}
	case 1:
		gvk, err := ParseGVK(parts[0])
		if err != nil {
			return InvalidSelectorError{Command: sel, Err: err.Error()}
		}

		dst.SelectorType = SelectorTypeAll
		dst.GroupVersionKind = gvk

		return nil
	case 2:
		if parts[1] == "" {
			return InvalidSelectorError{Command: sel, Err: "missing resource UID(s)"}
		}

		gvk, err := ParseGVK(parts[0])
		if err != nil {
			return InvalidSelectorError{Command: sel, Err: err.Error()}
		}

		uids, err := parseUIDs(parts[1])
		if err != nil {
			return InvalidSelectorError{Command: sel, Err: err.Error()}
		}

		dst.GroupVersionKind = gvk
		dst.ResourceUIDs = uids
		if len(dst.ResourceUIDs) > 1 {
			dst.SelectorType = SelectorTypeMultiple
		} else {
			dst.SelectorType = SelectorTypeSingle
		}

		return nil
	}

	return InvalidSelectorError{Command: sel, Err: fmt.Sprintf("invalid command '%s'", parts)}
}

func parseUIDs(uids string) ([]string, error) {
	if uids == "" {
		return nil, errors.New("missing resource UID(s)")
	}

	res := strings.Split(uids, ",")
	for _, uid := range res {
		if uid == "" {
			return nil, errors.New("missing resource UID")
		}
	}

	return res, nil
}
