# METADATA
# description: Panels should have a title and description.
# related_resources:
#  - ref: https://github.com/grafana/dashboard-linter/blob/main/docs/rules/panel-title-description-rule.md
#    description: documentation
# custom:
#  severity: warning
package grafanactl.rules.dashboard["panel-title-description"]

import data.grafanactl.result
import data.grafanactl.utils

report contains violation if {
	utils.resource_is_dashboard_v1alpha1(input)

	panels := utils.dashboard_v1alpha1_panels(input)

	invalid_panels := [panels[i] | object.get(panels[i], ["title"], "") == ""]

	some i
	invalid_panels[i]

	violation := result.fail(rego.metadata.chain(), sprintf("panel %d has no title", [invalid_panels[i].id]))
}

report contains violation if {
	utils.resource_is_dashboard_v1alpha1(input)

	panels := utils.dashboard_v1alpha1_panels(input)

	invalid_panels := [panels[i] | object.get(panels[i], ["description"], "") == ""]

	some i
	invalid_panels[i]

	violation := result.fail(rego.metadata.chain(), sprintf("panel %d has no description", [invalid_panels[i].id]))
}
