package resources_test

import (
	"errors"
	"path/filepath"
	"testing"

	"github.com/grafana/grafanactl/internal/format"
	"github.com/grafana/grafanactl/internal/resources"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func TestFSWriter_Write(t *testing.T) {
	req := require.New(t)
	outputDir := filepath.Join(t.TempDir(), "output")

	writer := resources.FSWriter{
		Directory: outputDir,
		Formatter: format.YAML,
		Namer: func(resource unstructured.Unstructured) (string, error) {
			return resource.GetName() + ".yaml", nil
		},
	}

	err := writer.Write(t.Context(), *testResources())
	req.NoError(err)

	req.FileExists(filepath.Join(outputDir, "folder-uid.yaml"))
	req.FileExists(filepath.Join(outputDir, "sa-uid.yaml"))
}

func TestFSWriter_Write_continueOnError(t *testing.T) {
	req := require.New(t)
	outputDir := filepath.Join(t.TempDir(), "output")

	writer := resources.FSWriter{
		Directory:       outputDir,
		Formatter:       format.YAML,
		ContinueOnError: true,
		Namer: func(resource unstructured.Unstructured) (string, error) {
			if resource.GetKind() == "Folder" {
				return "", errors.New("woops, folders are causing some trouble :(")
			}
			return resource.GetName() + ".yaml", nil
		},
	}

	err := writer.Write(t.Context(), *testResources())
	req.NoError(err)

	req.NoFileExists(filepath.Join(outputDir, "folder-uid.yaml"), "not created because of an error somewhere")
	req.FileExists(filepath.Join(outputDir, "sa-uid.yaml"), "continued on error and got created")
}

func TestFSWriter_Write_groupedByKind(t *testing.T) {
	req := require.New(t)
	outputDir := filepath.Join(t.TempDir(), "output")

	writer := resources.FSWriter{
		Directory: outputDir,
		Formatter: format.JSON,
		Namer:     resources.GroupResourcesByKind("json"),
	}

	err := writer.Write(t.Context(), *testResources())
	req.NoError(err)

	req.FileExists(filepath.Join(outputDir, "Folder", "folder-uid.json"))
	req.FileExists(filepath.Join(outputDir, "ServiceAccount", "sa-uid.json"))
}

func TestFSWriter_Write_doesNothingWithNoResources(t *testing.T) {
	req := require.New(t)
	outputDir := filepath.Join(t.TempDir(), "output")
	input := &unstructured.UnstructuredList{
		Items: []unstructured.Unstructured{},
	}

	writer := resources.FSWriter{
		Directory: outputDir,
		Formatter: format.YAML,
		Namer: func(resource unstructured.Unstructured) (string, error) {
			return resource.GetName() + ".yaml", nil
		},
	}

	err := writer.Write(t.Context(), *input)
	req.NoError(err)

	req.NoDirExists(outputDir)
}

func testResources() *unstructured.UnstructuredList {
	return &unstructured.UnstructuredList{
		Items: []unstructured.Unstructured{
			{
				Object: map[string]any{
					"apiVersion": "folder.grafana.app/v0alpha1",
					"kind":       "Folder",
					"metadata": map[string]any{
						"name":      "folder-uid",
						"namespace": "default",
					},
					"spec": map[string]any{
						"title": "Test folder",
					},
				},
			},
			{
				Object: map[string]any{
					"apiVersion": "iam.grafana.app/v0alpha1",
					"kind":       "ServiceAccount",
					"metadata": map[string]any{
						"name":      "sa-uid",
						"namespace": "default",
					},
					"spec": map[string]any{
						"title": "editor",
					},
				},
			},
		},
	}
}
