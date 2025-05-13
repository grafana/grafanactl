package process

import (
	"github.com/grafana/grafana/pkg/apimachinery/utils"
	"github.com/grafana/grafanactl/internal/resources"
)

// ManagerFieldsAppender is a processor that appends manager and source fields to a resource.
// It will return an error if the resource is already managed by another manager.
type ManagerFieldsAppender struct {
}

func (m *ManagerFieldsAppender) Process(r *resources.Resource) error {
	if r.IsEmpty() {
		return nil
	}

	if !r.IsManaged() {
		// If the resource is not managed by grafanactl,
		// we don't want to set the manager fields.
		return nil
	}

	r.Raw.SetManagerProperties(utils.ManagerProperties{
		Kind:     resources.ResourceManagerKind,
		Identity: "grafanactl", // TODO: use version information to set the identity.
	})

	// TODO: should we set timestamp & checksum as well?
	r.Raw.SetSourceProperties(utils.SourceProperties{
		Path: r.Source.String(),
	})

	return nil
}
