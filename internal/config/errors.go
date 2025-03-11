package config

import (
	"errors"
	"fmt"
	"os"
	"path"

	"github.com/grafana/grafanactl/internal/fail"
)

var ErrContextNotFound = errors.New("context not found")

func ContextNotFound(name string) error {
	return fail.DetailedError{
		Summary: "Context \"" + name + "\" does not exist",
		Parent:  ErrContextNotFound,
		Suggestions: []string{
			"Check for typos in the context name",
			fmt.Sprintf("Review your configuration for existing contexts: %s config view", path.Base(os.Args[0])),
		},
	}
}
