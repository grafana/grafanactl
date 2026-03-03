# METADATA
# description: Timezone should be utc. For reasons.
package grafanactl.rules.dashboard.idiomatic["timezone-utc"]

import data.grafanactl.result
import data.grafanactl.utils

# Dashboard v1
report contains violation if {
	utils.resource_is_dashboard_v1(input)

	input.spec.timezone != "utc"

	violation := result.fail(rego.metadata.chain(), sprintf("timezone is '%s', expected 'utc'", [input.spec.timezone]))
}

# Dashboard v2
report contains violation if {
	utils.resource_is_dashboard_v2(input)

	input.spec.timeSettings.timezone != "utc"

	violation := result.fail(rego.metadata.chain(), sprintf("timezone is '%s', expected 'utc'", [input.spec.timeSettings.timezone]))
}
