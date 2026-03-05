---
title: Lint resources
---

Grafana CLI offers a linter that can be used to verify that resources – dashboards, alerts, … —
comply with good practices and environment-specific policies.

## Using the linter

Resources can be linted using:

```shell
grafanactl linter lint ./resources
```

Directories are recursively explored and all [built-in rules](../reference/linter-rules/index.md)
are enabled by default.

For a finer control, the rules used to lint resources can be configured:

```shell
# Disable all rules for a resource type:
grafanactl linter lint --disable-resource dashboard ./resources

# Disable all rules in a category:
grafanactl linter lint --disable-category idiomatic ./resources

# Disable specific rules:
grafanactl linter lint --disable uneditable-dashboard --disable panel-title-description ./resources

# Enable rules for specific resource types:
grafanactl linter lint --disable-all --enable-resource dashboard ./resources

# Enable only some categories:
grafanactl linter lint --disable-all --enable-category idiomatic ./resources

# Enable only specific rules:
grafanactl linter lint --disable-all --enable uneditable-dashboard ./resources
```

## Define custom linting rules

Custom and built-in rules are defined in [Rego](https://www.openpolicyagent.org/docs/policy-language),
the policy language used by [Open Policy Agent (OPA)](https://www.openpolicyagent.org/).

They can be extremely useful to make sure that resources comply with policies
specific to your environment.

New custom rules can be scaffolded with `grafanactl`:

```shell
# Creates a new "dashboard" linter rule in the current directory:
grafanactl linter new dashboard custom-rule
```

As a result, a file with the bootstrapped rule is generated:

```rego
# METADATA
# description: Briefly describe the rule here.
# custom:
#  severity: warning
package custom.grafanactl.rules.dashboard.idiomatic["custom-rule"]

import data.grafanactl.result
import data.grafanactl.utils

# Dashboard v1
report contains violation if {
	utils.resource_is_dashboard_v1(input)

	input.spec.timezone != "utc"

	violation := result.fail(rego.metadata.chain(), sprintf("timezone is '%s', expected 'utc'", input.spec.timezone))
}

# Dashboard v2
report contains violation if {
	utils.resource_is_dashboard_v2(input)

	input.spec.timeSettings.timezone != "utc"

	violation := result.fail(rego.metadata.chain(), sprintf("timezone is '%s', expected 'utc'", input.spec.timeSettings.timezone))
}
```

[Built-in rules](https://github.com/grafana/grafanactl/tree/main/internal/linter/bundle/grafanactl/rules)
can be a good source of inspiration when writing custom ones.

The severity level of a rule can be changed by updating the `custom.severity` annotation.
Valid values are `warning` and `error`.

Rules can also be created for other resource types than dashboards, or in other
categories than the `idiomatic` one:

```shell
# Creates a new "alertrule" linter rule, categorized under "bug":
grafanactl linter new alertrule custom-rule -c bug
```