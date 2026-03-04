## grafanactl linter rules

List available linter rules

### Synopsis

List available linter rules.

```
grafanactl linter rules [flags]
```

### Examples

```

	# List built-in rules:

	grafanactl linter rules

	# List built-in and custom rules:

	grafanactl linter rules -r ./custom-rules

```

### Options

```
  -h, --help                help for rules
  -o, --output string       Output format. One of: json, yaml (default "yaml")
  -r, --rules stringArray   Path to custom rules.
```

### Options inherited from parent commands

```
      --no-color        Disable color output
  -v, --verbose count   Verbose mode. Multiple -v options increase the verbosity (maximum: 3).
```

### SEE ALSO

* [grafanactl linter](grafanactl_linter.md)	 - Lint Grafana resources

