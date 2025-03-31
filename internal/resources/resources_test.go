package resources_test

import (
	"testing"

	"github.com/grafana/grafanactl/internal/resources"
	"github.com/stretchr/testify/assert"
)

func TestParsePullCommands(t *testing.T) {
	tests := []struct {
		name string
		cmds []string
		want []resources.PullCommand
	}{
		{
			name: "should parse all resources of a type",
			cmds: []string{"dashboards"},
			want: []resources.PullCommand{
				{
					Kind: resources.PullCommandTypeAll,
					GVK: resources.DynamicGroupVersionKind{
						Group:   "",
						Version: "",
						Kind:    "dashboards",
					},
				},
			},
		},
		{
			name: "should parse single resource",
			cmds: []string{"dashboards/foo"},
			want: []resources.PullCommand{
				{
					Kind: resources.PullCommandTypeSingle,
					GVK: resources.DynamicGroupVersionKind{
						Group:   "",
						Version: "",
						Kind:    "dashboards",
					},
					UIDs: []string{"foo"},
				},
			},
		},
		{
			name: "should parse multiple resources of the same type",
			cmds: []string{"dashboards/foo,bar"},
			want: []resources.PullCommand{
				{
					Kind: resources.PullCommandTypeMultiple,
					GVK: resources.DynamicGroupVersionKind{
						Group:   "",
						Version: "",
						Kind:    "dashboards",
					},
					UIDs: []string{"foo", "bar"},
				},
			},
		},
		{
			name: "should parse multiple resources with the same FQDN",
			cmds: []string{"dashboards.v1alpha1.dashboard.grafana.app/foo,bar"},
			want: []resources.PullCommand{
				{
					Kind: resources.PullCommandTypeMultiple,
					GVK: resources.DynamicGroupVersionKind{
						Group:   "dashboard.grafana.app",
						Version: "v1alpha1",
						Kind:    "dashboards",
					},
					UIDs: []string{"foo", "bar"},
				},
			},
		},
		{
			name: "should parse single resources of different types",
			cmds: []string{
				"dashboards/foo",
				"folders/bar",
			},
			want: []resources.PullCommand{
				{
					Kind: resources.PullCommandTypeSingle,
					GVK: resources.DynamicGroupVersionKind{
						Group:   "",
						Version: "",
						Kind:    "dashboards",
					},
					UIDs: []string{"foo"},
				},
				{
					Kind: resources.PullCommandTypeSingle,
					GVK: resources.DynamicGroupVersionKind{
						Group:   "",
						Version: "",
						Kind:    "folders",
					},
					UIDs: []string{"bar"},
				},
			},
		},
		{
			name: "should parse multiple resources of different types",
			cmds: []string{
				"dashboards/foo,bar",
				"folders/qux,quux",
			},
			want: []resources.PullCommand{
				{
					Kind: resources.PullCommandTypeMultiple,
					GVK: resources.DynamicGroupVersionKind{
						Group:   "",
						Version: "",
						Kind:    "dashboards",
					},
					UIDs: []string{"foo", "bar"},
				},
				{
					Kind: resources.PullCommandTypeMultiple,
					GVK: resources.DynamicGroupVersionKind{
						Group:   "",
						Version: "",
						Kind:    "folders",
					},
					UIDs: []string{"qux", "quux"},
				},
			},
		},
		{
			name: "should parse multiple resources of different types with mixed format",
			cmds: []string{
				"dashboards/foo,bar",
				"folders.folder/qux,quux",
			},
			want: []resources.PullCommand{
				{
					Kind: resources.PullCommandTypeMultiple,
					GVK: resources.DynamicGroupVersionKind{
						Group:   "",
						Version: "",
						Kind:    "dashboards",
					},
					UIDs: []string{"foo", "bar"},
				},
				{
					Kind: resources.PullCommandTypeMultiple,
					GVK: resources.DynamicGroupVersionKind{
						Group:   "folder",
						Version: "",
						Kind:    "folders",
					},
					UIDs: []string{"qux", "quux"},
				},
			},
		},
		{
			name: "should parse multiple FQDNs",
			cmds: []string{
				"dashboards.v1alpha1.dashboard.grafana.app/foo",
				"folders.v1alpha1.folder.grafana.app/bar",
			},
			want: []resources.PullCommand{
				{
					Kind: resources.PullCommandTypeSingle,
					GVK: resources.DynamicGroupVersionKind{
						Group:   "dashboard.grafana.app",
						Version: "v1alpha1",
						Kind:    "dashboards",
					},
					UIDs: []string{"foo"},
				},
				{
					Kind: resources.PullCommandTypeSingle,
					GVK: resources.DynamicGroupVersionKind{
						Group:   "folder.grafana.app",
						Version: "v1alpha1",
						Kind:    "folders",
					},
					UIDs: []string{"bar"},
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := resources.ParsePullCommands(test.cmds)

			assert.ElementsMatch(t, test.want, got)
			assert.NoError(t, err)
		})
	}
}
