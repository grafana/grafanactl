package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"

	"github.com/go-chi/chi"
	"github.com/grafana/grafana-app-sdk/logging"
	"github.com/grafana/grafana/pkg/apimachinery/utils"
	"github.com/grafana/grafanactl/internal/config"
	"github.com/grafana/grafanactl/internal/format"
	"github.com/grafana/grafanactl/internal/httputils"
	"github.com/grafana/grafanactl/internal/resources"
	"github.com/grafana/grafanactl/internal/server/grafana"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

var _ ResourceHandler = &DashboardProxy{}

// DashboardProxy describes how to proxy Dashboard resources.
type DashboardProxy struct {
	context   *config.Context
	resources *resources.Resources
}

func NewDashboardProxy(context *config.Context, resources *resources.Resources) *DashboardProxy {
	return &DashboardProxy{
		context:   context,
		resources: resources,
	}
}

func (c *DashboardProxy) ResourceType() resources.GroupVersionKind {
	return resources.GroupVersionKind{
		Group: "dashboard.grafana.app",
		Kind:  "Dashboard",
	}
}

func (c *DashboardProxy) ProxyURL(uid string) string {
	return fmt.Sprintf("/d/%s/slug", uid)
}

func (c *DashboardProxy) Endpoints() []HTTPEndpoint {
	return []HTTPEndpoint{
		{
			Method:  http.MethodGet,
			URL:     "/d/{uid}/{slug}",
			Handler: grafana.AuthenticateAndProxyHandler(c.context.Grafana),
		},
		{
			Method:  http.MethodGet,
			URL:     "/apis/dashboard.grafana.app/{version}/namespaces/{namespace}/dashboards/{name}/dto",
			Handler: c.dashboardJSONGetHandler(),
		},
		{
			Method:  http.MethodPut,
			URL:     "/apis/dashboard.grafana.app/{version}/namespaces/{namespace}/dashboards/{name}",
			Handler: c.dashboardJSONPostHandler(),
		},
	}
}

func (c *DashboardProxy) StaticEndpoints() StaticProxyConfig {
	return StaticProxyConfig{
		ProxyGet: []string{
			"/api/datasources/proxy/*",
			"/api/datasources/*",
			"/api/plugins/*",
		},
		ProxyPost: []string{
			"/api/datasources/proxy/*",
			"/api/ds/query",
		},
		MockGet: map[string]string{
			"/api/annotations":                 "[]",
			"/api/access-control/user/actions": `{"dashboards:write": true}`,
		},
	}
}

func (c *DashboardProxy) dashboardJSONGetHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		resource := c.dashboardFromRequest(w, r)
		if resource == nil {
			return
		}

		accessConfig := map[string]any{
			"slug":      "slug",
			"url":       "/d/" + resource.Name() + "/slug",
			"canSave":   true,
			"canEdit":   true,
			"canAdmin":  false,
			"canStar":   false,
			"canDelete": false,
			"annotationsPermissions": map[string]any{
				"dashboard": map[string]any{
					"canAdd":    false,
					"canEdit":   true,
					"canDelete": false,
				},
				"organization": map[string]any{
					"canAdd":    false,
					"canEdit":   true,
					"canDelete": false,
				},
			},
		}

		version := chi.URLParam(r, "version")

		if version != resource.Version() {
			logging.FromContext(r.Context()).Debug("resource version mismatch", slog.String("requested", version), slog.String("available", resource.Version()))
			httputils.WriteJSON(r, w, map[string]any{
				"kind":       "DashboardWithAccessInfo",
				"apiVersion": resource.APIVersion(),
				"metadata": map[string]any{
					"name":      resource.Name(),
					"namespace": resource.Namespace(),
				},
				"spec": nil,
				"status": map[string]any{
					"conversion": map[string]any{
						"failed":        true,
						"storedVersion": resource.Version(),
						"error":         "conversion in proxy implemented",
					},
				},
				"access": accessConfig,
			})
			return
		}

		spec, err := resource.Raw.GetSpec()
		if err != nil {
			err := errors.New("could not get resource spec")
			httputils.Error(r, w, err.Error(), err, http.StatusInternalServerError)
			return
		}
		object := map[string]any{
			"kind":       "DashboardWithAccessInfo",
			"apiVersion": resource.APIVersion(),
			"access":     accessConfig,
			"metadata": map[string]any{
				"name":        resource.Name(),
				"namespace":   resource.Namespace(),
				"labels":      resource.Raw.GetLabels(),
				"annotations": resource.Raw.GetAnnotations(),
			},
			"spec": spec,
		}

		httputils.WriteJSON(r, w, object)
	}
}

func (c *DashboardProxy) dashboardJSONPostHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		resource := c.dashboardFromRequest(w, r)
		if resource == nil {
			return
		}

		if resource.SourcePath() == "" {
			err := errors.New("resources generated from a script can not be persisted through grafanactl serve")
			httputils.Error(r, w, err.Error(), err, http.StatusBadRequest)
			return
		}

		input := map[string]any{}
		decoder := json.NewDecoder(r.Body)
		if err := decoder.Decode(&input); err != nil {
			httputils.Error(r, w, err.Error(), err, http.StatusBadRequest)
			return
		}

		object := &unstructured.Unstructured{}
		object.SetUnstructuredContent(input)
		object.SetGroupVersionKind(schema.GroupVersionKind{
			Group:   "dashboard.grafana.app",
			Version: chi.URLParam(r, "version"),
			Kind:    "Dashboard",
		})

		file, err := os.OpenFile(resource.SourcePath(), os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
		if err != nil {
			httputils.Error(r, w, err.Error(), err, http.StatusInternalServerError)
			return
		}
		defer file.Close()

		var codec format.Encoder = format.NewJSONCodec()
		if resource.SourceFormat() == "yaml" {
			codec = format.NewYAMLCodec()
		}

		if err := codec.Encode(file, object); err != nil {
			httputils.Error(r, w, err.Error(), err, http.StatusInternalServerError)
			return
		}

		metaAccessor, _ := utils.MetaAccessor(object)
		resource.Raw = metaAccessor

		httputils.WriteJSON(r, w, object)
	}
}

func (c *DashboardProxy) dashboardFromRequest(w http.ResponseWriter, r *http.Request) *resources.Resource {
	name := chi.URLParam(r, "name")
	if name == "" {
		httputils.Error(r, w, "No name specified", errors.New("no name specified within the URL"), http.StatusBadRequest)
		return nil
	}

	// TODO: kind + name isn't enough to unambiguously identify a resource
	resource, found := c.resources.Find("Dashboard", name)
	if !found {
		httputils.Error(r, w, fmt.Sprintf("Dashboard with name %s not found", name), fmt.Errorf("dashboard with UID %s not found", name), http.StatusNotFound)
		return nil
	}

	return resource
}
