## grafanactl linter lint

Lint Grafana resources

### Synopsis

Lint Grafana resources.

```
grafanactl linter lint PATH... [flags]
```

### Examples

```

	# Lint Grafana resources using builtin rules:

	grafanactl linter lint ./resources

	# Lint specific files:

	grafanactl linter lint ./resources/file.json ./resources/other.yaml

	# Display compact results:

	grafanactl linter lint ./resources -o compact

	# Use custom rules:

	grafanactl linter lint --rules ./custom-rules ./resources

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

### Options

```
      --debug                          Enable debug mode
      --disable stringArray            Disable a rule
      --disable-all                    Disable all rules
      --disable-category stringArray   Disable all rules in a category
      --disable-resource stringArray   Disable all rules for a resource type
      --enable stringArray             Enable a rule
      --enable-all                     Enable all rules
      --enable-category stringArray    Enable all rules in a category
      --enable-resource stringArray    Enable all rules for a resource type
  -h, --help                           help for lint
      --max-concurrent int             Maximum number of concurrent operations (default 10)
  -o, --output string                  Output format. One of: compact, json, pretty, yaml (default "pretty")
  -r, --rules stringArray              Path to custom rules
```

### Options inherited from parent commands

```
      --no-color        Disable color output
  -v, --verbose count   Verbose mode. Multiple -v options increase the verbosity (maximum: 3).
```

### SEE ALSO

* [grafanactl linter](grafanactl_linter.md)	 - Lint Grafana resources

