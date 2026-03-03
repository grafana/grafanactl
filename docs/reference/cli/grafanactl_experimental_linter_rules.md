## grafanactl experimental linter rules

List available linter rules

### Synopsis

List available linter rules.

```
grafanactl experimental linter rules [flags]
```

### Examples

```

	# List built-in rules:

	grafanactl experimental linter rules

	# List built-in and custom rules:

	grafanactl experimental linter rules -r ./custom-rules

```

### Options

```
      --debug               Enable debug mode
  -h, --help                help for rules
  -o, --output string       Output format. One of: json, yaml (default "json")
  -r, --rules stringArray   Path to custom rules.
```

### Options inherited from parent commands

```
      --no-color        Disable color output
  -v, --verbose count   Verbose mode. Multiple -v options increase the verbosity (maximum: 3).
```

### SEE ALSO

* [grafanactl experimental linter](grafanactl_experimental_linter.md)	 - Lint Grafana resources

