package config

import (
	"errors"
	"fmt"
)

var ErrContextNotFound = errors.New("context not found")

type ValidationError struct {
	// File holds the path to the configuration file containing the error.
	File    string
	Message string
	// Path refers to the location of the error in the configuration file.
	// It is expressed as a YAMLPath, as described in https://pkg.go.dev/github.com/goccy/go-yaml#PathString
	Path            string
	AnnotatedSource string
	Suggestions     []string
}

func (e ValidationError) Error() string {
	return e.Message
}

type UnmarshalError struct {
	File string
	Err  error
}

func (e UnmarshalError) Error() string {
	return e.Err.Error()
}

func ContextNotFound(name string) error {
	return fmt.Errorf("invalid context \"%s\": %w", name, ErrContextNotFound)
}
