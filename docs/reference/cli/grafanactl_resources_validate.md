## grafanactl resources validate

Validate resources

### Synopsis

Validate resources.

This command validates its inputs against a remote Grafana instance.


```
grafanactl resources validate [RESOURCE_SELECTOR]... [flags]
```

### Examples

```

	# Validate all resources in the default directory
	grafanactl resources validate

	# Validate a single resource kind
	grafanactl resources validate dashboards

	# Validate a multiple resource kinds
	grafanactl resources validate dashboards folders

	# Displaying validation results as YAML
	grafanactl resources validate -o yaml

	# Displaying validation results as JSON
	grafanactl resources validate -o json

```

### Options

```
  -h, --help                 help for validate
      --max-concurrent int   Maximum number of concurrent operations (default 10)
  -o, --output string        Output format. One of: json, text, yaml (default "text")
  -p, --path strings         Paths on disk from which to read the resources. (default [./resources])
      --stop-on-error        Stop validating resources when an error occurs
```

### Options inherited from parent commands

```
      --config string    Path to the configuration file to use
      --context string   Name of the context to use
      --no-color         Disable color output
  -v, --verbose count    Verbose mode. Multiple -v options increase the verbosity (maximum: 3).
```

### SEE ALSO

* [grafanactl resources](grafanactl_resources.md)	 - Manipulate Grafana resources

