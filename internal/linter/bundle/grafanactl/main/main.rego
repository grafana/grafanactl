package grafanactl.main

lint.violations = report

report contains violation if {
	not is_object(input)

	violation := {
		"category": "error",
		"rule": "invalid-input",
		"severity": "error",
		"description": "provided input must be a JSON document",
	}
}

# Built-in rules
report contains violation if {
    some category, rule

    not _rule_disabled(data.internal, category, rule)

    some violation in data.grafanactl.rules[category][rule].report
}

# Custom rules
report contains violation if {
    some category, rule

    not _rule_disabled(data.internal, category, rule)

    some violation in data.custom.grafanactl.rules[category][rule].report
}

# Enabled/disabled rules
_rule_disabled(params, _, rule) if {
    rule in params.disabled_rules
}

_rule_disabled(params, category, _) if {
	category in params.disabled_categories
}

_rule_disabled(params, category, rule) if {
	params.disable_all == true
	not category in params.enabled_categories
	not rule in params.enabled_rules
}
