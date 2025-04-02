package config

import (
	"errors"
	"fmt"
	"os"
	"path"

	"github.com/grafana/grafanactl/internal/fail"
)

var ErrContextNotFound = errors.New("context not found")

type ValidationError struct {
	// Path refers to the location of the error in the configuration file.
	// It is expressed as a YAMLPath, as described in https://pkg.go.dev/github.com/goccy/go-yaml#PathString
	Path        string
	Message     string
	Suggestions []string
}

func (e ValidationError) Error() string {
	return e.Message
}

func ContextNotFound(name string) error {
	return fail.DetailedError{
		Summary: "Context \"" + name + "\" does not exist",
		Parent:  ErrContextNotFound,
		Suggestions: []string{
			"Check for typos in the context name",
			fmt.Sprintf("Review your configuration: %s config view", path.Base(os.Args[0])),
		},
	}
}

func InvalidConfiguration(file string, details string, annotatedSource string) fail.DetailedError {
	message := fmt.Sprintf("Invalid configuration found in '%s':\n%s", file, details)
	if annotatedSource != "" {
		message += "\n\n" + annotatedSource
	}

	return fail.DetailedError{
		Summary: "Invalid configuration",
		Details: message,
		Suggestions: []string{
			fmt.Sprintf("Review your configuration: %s config view", path.Base(os.Args[0])),
		},
	}
}

func UnmarshalError(file string, parent error) error {
	return fail.DetailedError{
		Summary: "Invalid configuration",
		Details: fmt.Sprintf("Invalid configuration found in '%s'.", file),
		Parent:  parent,
	}
}
