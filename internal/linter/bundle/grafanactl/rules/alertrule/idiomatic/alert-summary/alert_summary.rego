# METADATA
# description: Alerts must have a summary.
# custom:
#  severity: error
package grafanactl.rules.alertrule.idiomatic["alert-summary"]

import data.grafanactl.result
import data.grafanactl.utils

report contains violation if {
	utils.resource_is_alert_v0(input)

	object.get(input.spec, ["annotations", "summary"], "") == ""

	violation := result.fail(rego.metadata.chain(), "alert rule has no summary")
}
