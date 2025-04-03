package resources

import (
	"errors"
	"fmt"
	"strings"
)

// PullCommandKind is the kind of pull command.
// It identifies whether the command needs to list all resources of a type,
// get a single resource by UID, or get multiple resources by UID.
type PullCommandKind int

const (
	PullCommandTypeUnknown PullCommandKind = iota
	PullCommandTypeAll
	PullCommandTypeMultiple
	PullCommandTypeSingle
)

// DynamicGroupVersionKind is a group, version, and kind,
// which can be used to identify a resource.
// Not all fields are required to be set.
// It is expected that anything that accepts a DynamicGroupVersionKind
// will handle the discovery of the resource based on the fields that are present.
type DynamicGroupVersionKind struct {
	Group   string
	Version string
	Kind    string
}

func (gvk DynamicGroupVersionKind) String() string {
	// TODO: handle empty version and group
	return fmt.Sprintf("%s.%s.%s", gvk.Kind, gvk.Version, gvk.Group)
}

// PullCommand is a command to pull a resource from Grafana.
type PullCommand struct {
	Kind PullCommandKind
	GVK  DynamicGroupVersionKind
	UIDs []string
}

type InvalidCommandError struct {
	Command string
	Err     string
}

func (e InvalidCommandError) Error() string {
	return fmt.Sprintf("invalid command '%s': %s", e.Command, e.Err)
}

// ParsePullCommands parses a list of pull commands.
func ParsePullCommands(cmds []string) ([]PullCommand, error) {
	res := make([]PullCommand, len(cmds))

	for i, cmd := range cmds {
		if err := ParsePullCommand(cmd, &res[i]); err != nil {
			return nil, err
		}
	}

	return res, nil
}

// ParsePullCommand parses a pull command string into a PullCommand.
func ParsePullCommand(cmd string, dst *PullCommand) error {
	parts := strings.Split(cmd, "/")

	if len(parts) == 0 {
		return InvalidCommandError{Command: cmd, Err: "missing resource type"}
	}

	// grafanactl resources pull dashboards
	if len(parts) == 1 {
		gvk, err := ParseGVK(parts[0])
		if err != nil {
			return InvalidCommandError{Command: cmd, Err: err.Error()}
		}

		dst.Kind = PullCommandTypeAll
		dst.GVK = gvk
	}

	if len(parts) == 2 { //nolint:nestif
		if parts[1] == "" {
			return InvalidCommandError{Command: cmd, Err: "missing resource UID(s)"}
		}

		gvk, err := ParseGVK(parts[0])
		if err != nil {
			return InvalidCommandError{Command: cmd, Err: err.Error()}
		}

		uids, err := parseUIDs(parts[1])
		if err != nil {
			return InvalidCommandError{Command: cmd, Err: err.Error()}
		}

		dst.GVK = gvk
		dst.UIDs = uids
		if len(dst.UIDs) > 1 {
			dst.Kind = PullCommandTypeMultiple
		} else {
			dst.Kind = PullCommandTypeSingle
		}
	}

	// TODO: what if len(parts) > 2?
	// Shouldn't there be an error if there are more parts than expected?

	return nil
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

// ParseGVK parses a GVK string into a DynamicGroupVersionKind.
func ParseGVK(gvk string) (DynamicGroupVersionKind, error) {
	parts := strings.SplitN(gvk, ".", 3)

	var res DynamicGroupVersionKind
	switch len(parts) {
	case 2:
		if len(parts[0]) == 0 {
			return DynamicGroupVersionKind{}, errors.New("must specify kind")
		}

		if len(parts[1]) == 0 {
			return DynamicGroupVersionKind{}, errors.New("must specify group")
		}

		res = DynamicGroupVersionKind{
			Group:   parts[1],
			Version: "", // Default version
			Kind:    parts[0],
		}
	case 3:
		if len(parts[0]) == 0 {
			return DynamicGroupVersionKind{}, errors.New("must specify kind")
		}

		if len(parts[1]) == 0 {
			return DynamicGroupVersionKind{}, errors.New("must specify version")
		}

		if len(parts[2]) == 0 {
			return DynamicGroupVersionKind{}, errors.New("must specify group")
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
