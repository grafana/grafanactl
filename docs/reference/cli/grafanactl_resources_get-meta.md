## grafanactl resources get-meta

Get partial object metadata for Grafana resources

### Synopsis

Get partial object metadata (name, namespace, labels, annotations) for Grafana resources.

```
grafanactl resources get-meta RESOURCE_SELECTOR [flags]
```

### Examples

```

	# All instances of a resource type:

	grafanactl resources get-meta dashboards

	# One or more specific instances:

	grafanactl resources get-meta dashboards/foo
	grafanactl resources get-meta dashboards/foo,bar

	# Long kind format with version:

	grafanactl resources get-meta dashboards.v1alpha1.dashboard.grafana.app
	grafanactl resources get-meta dashboards.v1alpha1.dashboard.grafana.app/foo

```

### Options

```
  -h, --help              help for get-meta
  -o, --output string     Output format. One of: json, text, wide, yaml (default "text")
  -l, --selector string   Filter resources by label selector (e.g. -l key=value,other=value)
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

