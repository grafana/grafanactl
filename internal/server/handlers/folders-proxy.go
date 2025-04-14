package handlers

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/grafana/grafanactl/internal/httputils"
	"github.com/grafana/grafanactl/internal/resources"
)

var _ ResourceHandler = &FoldersProxy{}

// FoldersProxy describes how to proxy Folder resources.
type FoldersProxy struct {
	resources *resources.Resources
}

func NewFoldersProxy(resources *resources.Resources) *FoldersProxy {
	return &FoldersProxy{
		resources: resources,
	}
}

// FIXME: resources stuff.
func (c *FoldersProxy) ResourceType() resources.GroupVersionKind {
	return resources.GroupVersionKind{
		Group: "folder.grafana.app",
		Kind:  "Folder",
	}
}

func (c *FoldersProxy) ProxyURL(_ string) string {
	return ""
}

func (c *FoldersProxy) Endpoints() []HTTPEndpoint {
	return []HTTPEndpoint{
		{
			Method:  http.MethodGet,
			URL:     "/api/folders/{name}",
			Handler: c.folderJSONGetHandler(),
		},
	}
}

func (c *FoldersProxy) StaticEndpoints() StaticProxyConfig {
	return StaticProxyConfig{}
}

func (c *FoldersProxy) folderJSONGetHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		name := chi.URLParam(r, "name")
		if name == "" {
			httputils.Error(r, w, "No name specified", errors.New("no name specified within the URL"), http.StatusBadRequest)
			return
		}

		// TODO: use at least group + kind to identify a resource
		resource, found := c.resources.Find("Folder", name)
		if !found {
			httputils.Error(r, w, fmt.Sprintf("Folder with name %s not found", name), fmt.Errorf("folder with UID %s not found", name), http.StatusNotFound)
			return
		}

		// TODO: this is far from complete, but it's enough to serve dashboards defined in a folder
		folder := map[string]any{
			"uid":   name,
			"title": resource.Raw.FindTitle(name),
		}

		httputils.WriteJSON(r, w, folder)
	}
}
