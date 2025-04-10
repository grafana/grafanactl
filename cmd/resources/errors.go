package resources

import (
	"errors"
	"fmt"

	"github.com/grafana/grafanactl/internal/fail"
	"github.com/grafana/grafanactl/internal/resources"
)

func clientInitErr(err error) error {
	return fail.DetailedError{
		Parent: err,
		// NB: this is somewhat generic, but since clients rely on discovery,
		// we can tell the user that if the client initialization fails,
		// it's because we cannot load the supported resources from the API.
		// (which is more or less correct anyway)
		Summary: "Could not load supported resources from the API",
		Details: "Error when trying to load supported resources from the API",
		Suggestions: []string{
			"Make sure that the API is reachable",
			"Make sure that you are using a valid authentication method (token, username/password, etc.)",
			"Make sure that your authentication credentials have the necessary permissions for listing available resources",
		},
	}
}

func parseSelectorErr(err error) error {
	invalidCommandErr := &resources.InvalidSelectorError{}
	if err != nil && errors.As(err, invalidCommandErr) {
		return fail.DetailedError{
			Parent:  err,
			Summary: "Could not parse pull command(s)",
			Details: fmt.Sprintf("Failed to parse command '%s'", invalidCommandErr.Command),
			Suggestions: []string{
				"Make sure that your are passing in valid resource paths",
			},
		}
	}

	return err
}
