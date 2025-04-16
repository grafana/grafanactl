package main

import (
	"errors"
	"fmt"
	"io/fs"
	"net/url"
	"os"

	"github.com/grafana/grafanactl/cmd/root"
	"github.com/grafana/grafanactl/internal/config"
	"github.com/grafana/grafanactl/internal/fail"
	"github.com/grafana/grafanactl/internal/resources"
	k8sapi "k8s.io/apimachinery/pkg/api/errors"
)

var version = "SNAPSHOT"

func main() {
	handleError(root.Command(version).Execute())
}

func handleError(err error) {
	if err == nil {
		return
	}

	exitCode := 1
	detailedErr := errorToDetailedError(err)

	fmt.Fprintln(os.Stderr, detailedErr.Error())

	if detailedErr.ExitCode != nil {
		exitCode = *detailedErr.ExitCode
	}

	os.Exit(exitCode)
}

func errorToDetailedError(err error) *fail.DetailedError {
	var converted bool
	detailedErr := &fail.DetailedError{}
	if errors.As(err, detailedErr) {
		return detailedErr
	}

	// Try to convert the error for common error categories
	errorConverters := []func(err error) (*fail.DetailedError, bool){
		convertConfigErrors,    // Config-related
		convertFSErrors,        // FS-related
		convertResourcesErrors, // Resources-related
		convertNetworkErrors,   // Network-related errors
		convertAPIErrors,       // API-related errors
	}

	for _, converter := range errorConverters {
		detailedErr, converted = converter(err)
		if converted {
			return detailedErr
		}
	}

	return &fail.DetailedError{
		Summary: "Unexpected error",
		Parent:  err,
	}
}

func convertConfigErrors(err error) (*fail.DetailedError, bool) {
	validationErr := config.ValidationError{}
	if errors.As(err, &validationErr) {
		message := fmt.Sprintf("Invalid configuration found in '%s':\n%s", validationErr.File, validationErr.Message)
		if validationErr.AnnotatedSource != "" {
			message += "\n\n" + validationErr.AnnotatedSource
		}

		return &fail.DetailedError{
			Summary: "Invalid configuration",
			Details: message,
			Suggestions: append([]string{
				"Review your configuration: grafanactl config view",
			}, validationErr.Suggestions...),
		}, true
	}

	unmarshalErr := config.UnmarshalError{}
	if errors.As(err, &unmarshalErr) {
		return &fail.DetailedError{
			Summary: "Could not parse configuration",
			Details: fmt.Sprintf("Invalid configuration found in '%s'.", unmarshalErr.File),
			Parent:  unmarshalErr.Err,
		}, true
	}

	if errors.Is(err, config.ErrContextNotFound) {
		return &fail.DetailedError{
			Summary: "Invalid configuration",
			Parent:  err,
			Suggestions: []string{
				"Check for typos in the context name",
				"Review your configuration: grafanactl config view",
			},
		}, true
	}

	return nil, false
}

func convertNetworkErrors(err error) (*fail.DetailedError, bool) {
	urlErr := &url.Error{}
	if errors.As(err, &urlErr) {
		return &fail.DetailedError{
			Parent:  err,
			Summary: "Network error",
			Suggestions: []string{
				"Make sure that the API is reachable",
				"Make sure that the configured target server is correct",
			},
		}, true
	}

	return nil, false
}

func convertAPIErrors(err error) (*fail.DetailedError, bool) {
	statusErr := &k8sapi.StatusError{}
	if !errors.As(err, &statusErr) {
		return nil, false
	}

	reason := k8sapi.ReasonForError(statusErr)
	code := statusErr.Status().Code

	switch {
	case k8sapi.IsUnauthorized(statusErr),
		k8sapi.IsForbidden(statusErr):
		return &fail.DetailedError{
			Parent:  err,
			Summary: fmt.Sprintf("%s - code %d", reason, code),
			Suggestions: []string{
				"Make sure that the configured credentials are correct",
				"Make sure that the configured credentials have enough permissions",
			},
		}, true
	case k8sapi.IsNotFound(statusErr):
		return &fail.DetailedError{
			Parent:  err,
			Summary: fmt.Sprintf("Resource not found - code %d", code),
			Suggestions: []string{
				"Make sure that your are passing in valid resource selectors",
			},
		}, true
	}

	return &fail.DetailedError{
		Parent:  err,
		Summary: fmt.Sprintf("API error: %s - code %d", reason, code),
	}, true
}

func convertResourcesErrors(err error) (*fail.DetailedError, bool) {
	invalidCommandErr := &resources.InvalidSelectorError{}
	if err != nil && errors.As(err, invalidCommandErr) {
		return &fail.DetailedError{
			Parent:  err,
			Summary: "Could not parse resource(s) selector",
			Details: fmt.Sprintf("Failed to parse command '%s'", invalidCommandErr.Command),
			Suggestions: []string{
				"Make sure that your are passing in valid resource selectors",
			},
		}, true
	}

	return nil, false
}

func convertFSErrors(err error) (*fail.DetailedError, bool) {
	pathErr := &fs.PathError{}

	if errors.Is(err, os.ErrNotExist) && errors.As(err, &pathErr) {
		return &fail.DetailedError{
			Summary: "File not found",
			Details: fmt.Sprintf("could not read '%s'", pathErr.Path),
			Parent:  err,
			Suggestions: []string{
				"Check for typos in the command's arguments",
			},
		}, true
	}

	if errors.Is(err, os.ErrPermission) && errors.As(err, &pathErr) {
		return &fail.DetailedError{
			Summary: "Permission denied",
			Parent:  err,
			Suggestions: []string{
				"Review the permissions on the file",
			},
		}, true
	}

	return nil, false
}
