package grafanactl.utils_test

import data.grafanactl.utils

test_resource_is_dashboard_v1_ignores_non_dashboards if {
	resource := {
	    "kind": "Folder",
	    "apiVersion": "folder.grafana.app/v1beta1",
	    "metadata": {"name": "sandbox"},
	    "spec": {"title": "Sandbox"}
	}

	not utils.resource_is_dashboard_v1(resource)
}
