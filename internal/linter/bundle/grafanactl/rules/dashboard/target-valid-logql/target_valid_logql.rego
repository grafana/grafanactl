# METADATA
# description: Checks that Loki targets defined in dashboard panels use valid LogQL queries.
# related_resources:
#  - ref: https://github.com/grafana/dashboard-linter/blob/main/docs/rules/target-logql-rule.md
#    description: documentation
# custom:
#  severity: error
package grafanactl.rules.dashboard["target-valid-logql"]

import data.grafanactl.result
import data.grafanactl.utils

report contains violation if {
	utils.resource_is_dashboard_v1alpha1(input)

	variables := utils.dashboard_v1alpha1_variables(input)
	panels := utils.dashboard_v1alpha1_panels(input)
	loki_targets := _loki_targets(panels)

	queries := [query | query := {
		"expr": loki_targets[i].expr,
		"result": validate_logql(loki_targets[i].expr, variables),
	}]
	invalid_queries := [queries[i] | queries[i].result != ""]

	some i
	invalid_queries[i]

	violation := result.fail(rego.metadata.chain(), sprintf("%s\n%s", [invalid_queries[i].expr, invalid_queries[i].result]))
}

_loki_targets(panels) := loki_targets if {
	targets := [target | panels[i].targets[j]; target := panels[i].targets[j]]

	# TODO: handle cases where no datasource is defined at the target level
	loki_targets := [target | targets[i].datasource.type == "loki"; target := targets[i]]
}
