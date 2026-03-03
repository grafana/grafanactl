# METADATA
# description: Alerts should have a runbook.
# custom:
#  severity: warning
package grafanactl.rules.alertrule.idiomatic["alert-runbook-link"]

import data.grafanactl.result
import data.grafanactl.utils

report contains violation if {
	utils.resource_is_alert_v0(input)

	object.get(input.spec, ["annotations", "runbook_url"], "") == ""

	violation := result.fail(rego.metadata.chain(), "alert rule has no runbook")
}
