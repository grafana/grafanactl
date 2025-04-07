package resources

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/grafana/grafana-app-sdk/logging"
	"github.com/grafana/grafanactl/internal/format"
	"golang.org/x/sync/errgroup"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// FSReader is a reader that reads resources from the filesystem.
//
// The reader will read all resources from the filesystem and return them as
// an unstructured list.
type FSReader struct {
	// The list of directories in which to search for resources.
	Directories []string
	// Decoder to use when decoding resources.
	Decoder format.Decoder
	// Whether to stop reading resources upon encountering an error.
	StopOnError bool
	// MaxConcurrentReads is the maximum number of concurrent file reads.
	// If not set, the default is 1.
	MaxConcurrentReads int
}

// Read reads all resources from the filesystem and returns them as an unstructured list.
func (reader *FSReader) Read(ctx context.Context, dst *unstructured.UnstructuredList) error {
	logger := logging.FromContext(ctx)

	if len(reader.Directories) == 0 {
		logger.Debug("no directories or resources to read")
		return nil
	}

	if reader.MaxConcurrentReads == 0 {
		reader.MaxConcurrentReads = 1
	}

	// Error group & channel for coordinating the reading and processing of resources.
	gr, ctx := errgroup.WithContext(ctx)

	// Read directories.
	pathCh := make(chan string, reader.MaxConcurrentReads)
	gr.Go(func() error {
		defer close(pathCh)

		for _, dir := range reader.Directories {
			if err := filepath.WalkDir(dir, func(path string, info os.DirEntry, err error) error {
				logger := logging.FromContext(ctx)

				// Early return if context is cancelled
				if ctx.Err() != nil {
					return filepath.SkipAll
				}

				if err != nil {
					if reader.StopOnError {
						return err
					}
					logger.Warn("failed to traverse directory", "path", path, "error", err)
					return nil
				}

				// For directories, return nil to continue traversing
				if info.IsDir() {
					logger.Debug("entering directory", "path", path) // Add debug logging
					return nil
				}

				select {
				case <-ctx.Done():
					return filepath.SkipAll
				case pathCh <- path:
				}

				return nil
			}); err != nil {
				return err
			}
		}

		return nil
	})

	resCh := make(chan readResult, reader.MaxConcurrentReads)
	gr.Go(func() error {
		defer close(resCh)

		readg, ctx := errgroup.WithContext(ctx)
		readg.SetLimit(reader.MaxConcurrentReads)

		for path := range pathCh {
			readg.Go(func() error {
				logger.Debug("processing file", "path", path)

				// Read and decode the file
				file, err := os.OpenFile(path, os.O_RDONLY, 0)
				if err != nil {
					if reader.StopOnError {
						return fmt.Errorf("failed to read file %s: %w", path, err)
					}

					logger.Warn("failed to read file", "path", path, "error", err)
					return nil
				}
				defer file.Close()

				var obj unstructured.Unstructured
				if err := reader.Decoder.Decode(file, &obj.Object); err != nil {
					if reader.StopOnError {
						return fmt.Errorf("failed to decode file %s: %w", path, err)
					}

					logger.Warn("failed to decode file", "path", path, "error", err)
					return nil
				}

				res := readResult{
					Object: obj,
					Path:   path,
				}

				select {
				case <-ctx.Done():
				case resCh <- res:
				}

				return nil
			})
		}

		return readg.Wait()
	})

	// Read all results in parallel.
	gr.Go(func() error {
		idx := make(map[objIdx]unstructured.Unstructured)
		dst.Items = make([]unstructured.Unstructured, 0, reader.MaxConcurrentReads)

		for res := range resCh {
			obj := res.Object

			if _, ok := idx[objIdx{
				gvk:  obj.GetObjectKind().GroupVersionKind(),
				name: obj.GetName(),
			}]; ok {
				logger.Info("skipping duplicate object",
					"gvk", obj.GetObjectKind().GroupVersionKind(),
					"name", obj.GetName(),
					"path", res.Path,
				)

				continue
			}

			logger.Debug("adding object",
				"gvk", obj.GetObjectKind().GroupVersionKind(),
				"name", obj.GetName(),
				"path", res.Path,
			)

			dst.Items = append(dst.Items, obj)
		}

		return nil
	})

	return gr.Wait()
}

type objIdx struct {
	gvk  schema.GroupVersionKind
	name string
}

type readResult struct {
	Object unstructured.Unstructured
	Path   string
}
