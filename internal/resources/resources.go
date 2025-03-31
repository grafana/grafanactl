package resources

import (
	"fmt"
	"strings"
)

type PullCommandKind int

const (
	PullCommandTypeUnknown PullCommandKind = iota
	PullCommandTypeAll
	PullCommandTypeMultiple
	PullCommandTypeSingle
)

type PullCommand struct {
	Kind PullCommandKind
	GVK  DynamicGroupVersionKind
	UIDs []string
}

type DynamicGroupVersionKind struct {
	Group   string
	Version string
	Kind    string
}

func (gvk DynamicGroupVersionKind) String() string {
	// TODO: handle empty version and group
	return fmt.Sprintf("%s.%s.%s", gvk.Kind, gvk.Version, gvk.Group)
}

// ParsePullCommands parses a list of pull commands.
func ParsePullCommands(cmds []string) ([]PullCommand, error) {
	res := make([]PullCommand, len(cmds))

	for i, cmd := range cmds {
		if err := ParsePullCommand(cmd, &res[i]); err != nil {
			return nil, fmt.Errorf("failed to parse pull command `%s`: %w", cmd, err)
		}
	}

	return res, nil
}

// ParsePullCommand parses a pull command string into a PullCommand.
func ParsePullCommand(cmd string, dst *PullCommand) error {
	parts := strings.Split(cmd, "/")

	if len(parts) == 0 {
		return fmt.Errorf("missing resource type")
	}

	// grafanactl resources pull dashboards
	if len(parts) == 1 {
		gvk, err := ParseGVK(parts[0])
		if err != nil {
			return err
		}

		dst.Kind = PullCommandTypeAll
		dst.GVK = gvk
	}

	// grafanactl resources pull dashboards/foo
	// grafanactl resources pull dashboards/foo,bar
	if len(parts) == 2 {
		if parts[1] == "" {
			return fmt.Errorf("missing resource UID(s)")
		}

		gvk, err := ParseGVK(parts[0])
		if err != nil {
			return err
		}

		uids, err := parseUIDs(parts[1])
		if err != nil {
			return err
		}

		dst.GVK = gvk
		dst.UIDs = uids
		if len(dst.UIDs) > 1 {
			dst.Kind = PullCommandTypeMultiple
		} else {
			dst.Kind = PullCommandTypeSingle
		}
	}

	return nil
}

func parseUIDs(uids string) ([]string, error) {
	if uids == "" {
		return nil, fmt.Errorf("missing resource UID(s)")
	}

	res := strings.Split(uids, ",")
	for _, uid := range res {
		if uid == "" {
			return nil, fmt.Errorf("missing resource UID")
		}
	}

	return res, nil
}

// ParseGVK parses a GVK string into a schema.GroupVersionKind.
//
// The format is one of the following:
// kind.version.group
// kind.group
// kind
func ParseGVK(gvk string) (DynamicGroupVersionKind, error) {
	parts := strings.SplitN(gvk, ".", 3)

	var res DynamicGroupVersionKind
	switch len(parts) {
	case 2:
		if len(parts[0]) == 0 {
			return DynamicGroupVersionKind{}, fmt.Errorf("must specify kind")
		}

		if len(parts[1]) == 0 {
			return DynamicGroupVersionKind{}, fmt.Errorf("must specify group")
		}

		res = DynamicGroupVersionKind{
			Group:   parts[1],
			Version: "", // Default version
			Kind:    parts[0],
		}
	case 3:
		if len(parts[0]) == 0 {
			return DynamicGroupVersionKind{}, fmt.Errorf("must specify kind")
		}

		if len(parts[1]) == 0 {
			return DynamicGroupVersionKind{}, fmt.Errorf("must specify version")
		}

		if len(parts[2]) == 0 {
			return DynamicGroupVersionKind{}, fmt.Errorf("must specify group")
		}

		res = DynamicGroupVersionKind{
			Group:   parts[2],
			Version: parts[1],
			Kind:    parts[0],
		}
	default:
		res = DynamicGroupVersionKind{
			Group:   "", // Default group
			Version: "", // Default version
			Kind:    parts[0],
		}
	}

	return res, nil
}

// TODO: move this to a separate file
