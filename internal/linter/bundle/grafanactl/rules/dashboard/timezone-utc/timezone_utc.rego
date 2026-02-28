# METADATA
# description: Timezone should be utc. For reasons.
package grafanactl.rules.dashboard["timezone-utc"]

import data.grafanactl.result
import data.grafanactl.utils

report contains violation if {
	utils.resource_is_dashboard_v1alpha1(input)

	input.spec.timezone != "utc"

	violation := result.fail(rego.metadata.chain(), sprintf("timezone is '%s', expected utc", [input.timezone]))
}
