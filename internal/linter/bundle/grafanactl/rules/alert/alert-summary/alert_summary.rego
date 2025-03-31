# METADATA
# description: Alerts must have a summary.
# custom:
#  severity: error
package grafanactl.rules.alert["alert-summary"]

import data.grafanactl.result
import data.grafanactl.utils

report contains violation if {
	utils.resource_is_alert(input)

	some i
	rule := input.rules[i]

	object.get(rule, ["annotations", "summary"], "") == ""

	violation := result.fail(rego.metadata.chain(), sprintf("rule %d has no summary", [i]))
}
