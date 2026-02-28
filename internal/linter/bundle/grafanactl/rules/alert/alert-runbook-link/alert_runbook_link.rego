# METADATA
# description: Alerts should have a runbook.
# custom:
#  severity: warning
package grafanactl.rules.alert["alert-runbook-link"]

import data.grafanactl.result
import data.grafanactl.utils

report contains violation if {
	utils.resource_is_alert(input)

	some i
	rule := input.rules[i]

	object.get(rule, ["annotations", "runbook_url"], "") == ""

	violation := result.fail(rego.metadata.chain(), sprintf("rule %d has no runbook", [i]))
}
