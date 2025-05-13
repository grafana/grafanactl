package remote

import "github.com/grafana/grafanactl/internal/resources"

// Processor can be used to modify a resource in-place,
// before it is written or after it is read from local sources.
//
// They can be used to e.g. strip server-side fields from a resource,
// or add extra metadata after a resource has been read from a file.
type Processor interface {
	Process(res *resources.Resource) error
}
