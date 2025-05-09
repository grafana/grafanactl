## grafanactl resources pull

Pull resources from Grafana

### Synopsis

Pull resources from Grafana using a specific format. See examples below for more details.

```
grafanactl resources pull [RESOURCE_SELECTOR]... [flags]
```

### Examples

```

	# Everything:

	grafanactl resources pull

	# All instances for a given kind(s):

	grafanactl resources pull dashboards
	grafanactl resources pull dashboards folders

	# Single resource kind, one or more resource instances:

	grafanactl resources pull dashboards/foo
	grafanactl resources pull dashboards/foo,bar

	# Single resource kind, long kind format:

	grafanactl resources pull dashboard.dashboards/foo
	grafanactl resources pull dashboard.dashboards/foo,bar

	# Single resource kind, long kind format with version:

	grafanactl resources pull dashboards.v1alpha1.dashboard.grafana.app/foo
	grafanactl resources pull dashboards.v1alpha1.dashboard.grafana.app/foo,bar

	# Multiple resource kinds, one or more resource instances:

	grafanactl resources pull dashboards/foo folders/qux
	grafanactl resources pull dashboards/foo,bar folders/qux,quux

	# Multiple resource kinds, long kind format:

	grafanactl resources pull dashboard.dashboards/foo folder.folders/qux
	grafanactl resources pull dashboard.dashboards/foo,bar folder.folders/qux,quux

	# Multiple resource kinds, long kind format with version:

	grafanactl resources pull dashboards.v1alpha1.dashboard.grafana.app/foo folders.v1alpha1.folder.grafana.app/qux
```

### Options

```
  -h, --help              help for pull
      --include-managed   Include resources managed by tools other than grafanactl
  -o, --output string     Output format. One of: json, yaml (default "json")
  -p, --path string       Path on disk in which the resources will be written (default "./resources")
      --stop-on-error     Stop pulling resources when an error occurs
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

