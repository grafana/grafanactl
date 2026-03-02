## grafanactl experimental linter lint

Lint Grafana resources

### Synopsis

Lint Grafana resources.

```
grafanactl experimental linter lint PATH... [flags]
```

### Examples

```

	# Lint Grafana resources using builtin rules:

	grafanactl experimental linter lint ./resources

	# Lint specific files:

	grafanactl experimental linter lint ./resources/file.json ./resources/other.yaml

	# Display compact results:

	grafanactl experimental linter lint ./resources -o compact

	# Use custom rules:

	grafanactl experimental linter lint --rules ./custom-rules ./resources

```

### Options

```
      --debug                Enable debug mode
  -h, --help                 help for lint
      --max-concurrent int   Maximum number of concurrent operations (default 10)
  -o, --output string        Output format. One of: compact, json, pretty, yaml (default "pretty")
  -r, --rules stringArray    Path to custom rules.
```

### Options inherited from parent commands

```
      --no-color        Disable color output
  -v, --verbose count   Verbose mode. Multiple -v options increase the verbosity (maximum: 3).
```

### SEE ALSO

* [grafanactl experimental linter](grafanactl_experimental_linter.md)	 - Lint Grafana resources

