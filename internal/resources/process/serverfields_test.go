package process_test

import (
	"testing"

	"github.com/grafana/grafana/pkg/apimachinery/utils"
	"github.com/grafana/grafanactl/internal/resources"
	"github.com/grafana/grafanactl/internal/resources/process"
	"github.com/stretchr/testify/require"
)

func TestServerFieldsStripper_Process(t *testing.T) {
	tests := []struct {
		name    string
		input   *resources.Resources
		want    *resources.Resources
		wantErr bool
	}{
		{
			name:    "no resources",
			input:   resources.NewResources(),
			want:    resources.NewResources(),
			wantErr: false,
		},
		{
			name: "one resource",
			input: resources.NewResources(
				resources.MustFromObject(map[string]any{
					"apiVersion": "dashboard.grafana.app/v1",
					"kind":       "Dashboard",
					"metadata": map[string]any{
						"name":              "example",
						"namespace":         "default",
						"uid":               "test",
						"generation":        1,
						"resourceVersion":   "1",
						"creationTimestamp": "2021-01-01T00:00:00Z",
						"annotations": map[string]any{
							utils.AnnoKeyCreatedBy:        "test",
							utils.AnnoKeyUpdatedBy:        "test",
							utils.AnnoKeyUpdatedTimestamp: "2021-01-01T00:00:00Z",
							"test-annotation":             "test",
						},
						"labels": map[string]any{
							utils.LabelKeyDeprecatedInternalID: "test",
							"test-label":                       "test",
						},
					},
					"spec": map[string]any{
						"foo": "bar",
					},
				}),
			),
			want: resources.NewResources(
				resources.MustFromObject(map[string]any{
					"apiVersion": "dashboard.grafana.app/v1",
					"kind":       "Dashboard",
					"metadata": map[string]any{
						"name":      "example",
						"namespace": "default",
						"annotations": map[string]string{
							"test-annotation": "test",
						},
						"labels": map[string]string{
							"test-label": "test",
						},
					},
					"spec": map[string]any{
						"foo": "bar",
					},
				}),
			),
			wantErr: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			proc := &process.ServerFieldsStripper{}
			actual, err := proc.Process(test.input)
			if test.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			require.Equal(t, actual.ToUnstructuredList(), test.want.ToUnstructuredList())
		})
	}
}
