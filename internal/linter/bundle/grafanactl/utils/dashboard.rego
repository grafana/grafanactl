package grafanactl.utils

resource_is_dashboard_v1alpha1(resource) if {
	resource.kind == "Dashboard"
	resource.apiVersion == "dashboard.grafana.app/v1alpha1"
}

dashboard_v1alpha1_panels(dashboard) := [panel | panel := dashboard.spec.panels[i]; panel.type != "row"]

dashboard_v1alpha1_variables(dashboard) := dashboard.spec.templating.list
