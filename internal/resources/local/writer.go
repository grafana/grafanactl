package local

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/grafana/grafana-app-sdk/logging"
	"github.com/grafana/grafanactl/internal/format"
	"github.com/grafana/grafanactl/internal/logs"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

type FileNamer func(resource unstructured.Unstructured) (string, error)

// GroupResourcesByKind organizes resources by kind, writing resources in a
// folder named after their kind.
// File names are generated as follows: `{Kind}/{Name}.{extension}`.
func GroupResourcesByKind(extension string) FileNamer {
	return func(resource unstructured.Unstructured) (string, error) {
		if resource.GetName() == "" {
			return "", errors.New("resource has no name")
		}

		return filepath.Join(resource.GetKind(), resource.GetName()+"."+extension), nil
	}
}

type FSWriter struct {
	// Directory on the filesystem where resources should be written.
	Directory string
	// Namer is a function mapping a resource to a path on the filesystem
	// (relative to Directory).
	// The naming strategy used here directly controls the file hierarchy created
	// by FSWriter.
	// Note: the path should contain an extension.
	Namer FileNamer
	// Encoder to use when encoding resources.
	Encoder format.Encoder
	// Whether to stop writing resources upon encountering an error.
	StopOnError bool
}

func (writer *FSWriter) Write(ctx context.Context, resources unstructured.UnstructuredList) error {
	if len(resources.Items) == 0 {
		return nil
	}

	logger := logging.FromContext(ctx).With(slog.String("directory", writer.Directory))
	logger.Debug("Writing resources", slog.Int("resources", len(resources.Items)))

	// Create the directory if it doesn't exist
	if err := ensureDirectoryExists(writer.Directory); err != nil {
		return err
	}

	for _, resource := range resources.Items {
		if err := writer.writeSingle(resource); err != nil {
			if writer.StopOnError {
				return err
			}

			logger.Warn("could not write resource: skipping", slog.String("kind", resource.GetKind()), logs.Err(err))
		}
	}

	return nil
}

func (writer *FSWriter) writeSingle(resource unstructured.Unstructured) error {
	filename, err := writer.Namer(resource)
	if err != nil {
		return fmt.Errorf("could not generate resource path: %w", err)
	}

	fullFileName := filepath.Join(writer.Directory, filename)
	if err := ensureDirectoryExists(filepath.Dir(fullFileName)); err != nil {
		return fmt.Errorf("could ensure resource directory exists: %w", err)
	}

	file, err := os.OpenFile(fullFileName, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return fmt.Errorf("could open resource file: %w", err)
	}
	defer file.Close()

	// MarshalJSON() methods for [unstructured.UnstructuredList] and
	// [unstructured.Unstructured] types are defined on pointer receivers,
	// so we need to make sure we dereference `resource` before formatting it.
	if err := writer.Encoder.Encode(file, &resource); err != nil {
		return fmt.Errorf("could write resource: %w", err)
	}

	return nil
}

func ensureDirectoryExists(directory string) error {
	if _, err := os.Stat(directory); err != nil {
		if os.IsNotExist(err) {
			err = os.MkdirAll(directory, 0755)
			if err != nil {
				return err
			}
		}

		return err
	}

	return nil
}
