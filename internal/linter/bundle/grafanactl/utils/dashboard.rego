package grafanactl.utils

resource_is_dashboard_v1(resource) if {
	resource.kind == "Dashboard"
	resource.apiVersion in {"dashboard.grafana.app/v0alpha1", "dashboard.grafana.app/v1beta1"}
}

dashboard_v1_panels(dashboard) := [panel | panel := dashboard.spec.panels[i]; panel.type != "row"]

dashboard_v1_variables(dashboard) := dashboard.spec.templating.list
