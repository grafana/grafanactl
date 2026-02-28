# METADATA
# description: Dashboards should not be editable.
# related_resources:
#  - ref: https://github.com/grafana/dashboard-linter/blob/main/docs/rules/template-uneditable-rule.md
#    description: documentation
# custom:
#  severity: warning
package grafanactl.rules.dashboard["uneditable-dashboard"]

import data.grafanactl.result
import data.grafanactl.utils

report contains violation if {
	utils.resource_is_dashboard_v1alpha1(input)

	input.spec.editable != false

	violation := result.fail(rego.metadata.chain(), "dashboard is editable")
}
