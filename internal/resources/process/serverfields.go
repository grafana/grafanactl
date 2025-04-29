package process

import (
	"github.com/grafana/grafana/pkg/apimachinery/utils"
	"github.com/grafana/grafanactl/internal/resources"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// ServerFieldsStripper is a processor that strips server-side fields from resources.
type ServerFieldsStripper struct{}

// Process strips server-side fields from resources.
func (m *ServerFieldsStripper) Process(src *resources.Resources) (*resources.Resources, error) {
	list := unstructured.UnstructuredList{
		Items: make([]unstructured.Unstructured, 0, src.Len()),
	}

	if err := src.ForEach(func(r *resources.Resource) error {
		spec, err := r.Raw.GetSpec()
		if err != nil {
			return err
		}

		// Remove annotations set by the server.
		annotations := r.Raw.GetAnnotations()
		delete(annotations, utils.AnnoKeyCreatedBy)
		delete(annotations, utils.AnnoKeyUpdatedBy)
		delete(annotations, utils.AnnoKeyUpdatedTimestamp)

		// Remove labels set by the server.
		labels := r.Raw.GetLabels()
		delete(labels, utils.LabelKeyDeprecatedInternalID)

		list.Items = append(list.Items, unstructured.Unstructured{
			Object: map[string]any{
				"apiVersion": r.APIVersion(),
				"kind":       r.Kind(),
				"metadata": map[string]any{
					"name":        r.Name(),
					"namespace":   r.Namespace(),
					"annotations": annotations,
					"labels":      labels,
				},
				"spec": spec,
			},
		})

		return nil
	}); err != nil {
		return nil, err
	}

	return resources.NewResourcesFromUnstructured(list)
}

// Name returns the name of the processor.
func (m *ServerFieldsStripper) Name() string {
	return "strip-server-fields"
}
