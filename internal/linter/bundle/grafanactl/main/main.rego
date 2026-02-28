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
    some violation in data.grafanactl.rules[category][rule].report
}

# Custom rules
report contains violation if {
    some category, rule
    some violation in data.custom.grafanactl.rules[category][rule].report
}
