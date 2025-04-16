package main

import (
	"errors"
	"fmt"
	"io/fs"
	"os"

	"github.com/grafana/grafanactl/cmd/root"
	"github.com/grafana/grafanactl/internal/config"
	"github.com/grafana/grafanactl/internal/fail"
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
		convertConfigErrors, // Config-related
		convertFSErrors,     // FS-related
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
