## grafanactl resources push

Push resources to Grafana

### Synopsis

Push resources to Grafana using a specific format. See examples below for more details.

```
grafanactl resources push [RESOURCE_SELECTOR]... [flags]
```

### Examples

```

	# Everything:

	grafanactl resources push

	# All instances for a given kind(s):

	grafanactl resources push dashboards
	grafanactl resources push dashboards folders

	# Single resource kind, one or more resource instances:

	grafanactl resources push dashboards/foo
	grafanactl resources push dashboards/foo,bar

	# Single resource kind, long kind format:

	grafanactl resources push dashboard.dashboards/foo
	grafanactl resources push dashboard.dashboards/foo,bar

	# Single resource kind, long kind format with version:

	grafanactl resources push dashboards.v1alpha1.dashboard.grafana.app/foo
	grafanactl resources push dashboards.v1alpha1.dashboard.grafana.app/foo,bar

	# Multiple resource kinds, one or more resource instances:

	grafanactl resources push dashboards/foo folders/qux
	grafanactl resources push dashboards/foo,bar folders/qux,quux

	# Multiple resource kinds, long kind format:

	grafanactl resources push dashboard.dashboards/foo folder.folders/qux
	grafanactl resources push dashboard.dashboards/foo,bar folder.folders/qux,quux

	# Multiple resource kinds, long kind format with version:

	grafanactl resources push dashboards.v1alpha1.dashboard.grafana.app/foo folders.v1alpha1.folder.grafana.app/qux
```

### Options

```
      --dry-run               If set, the push operation will be simulated, without actually creating or updating any resources
  -h, --help                  help for push
      --include-managed       If set, resources managed by other tools will be included in the push operation
      --max-concurrent int    Maximum number of concurrent operations (default 10)
      --omit-manager-fields   If set, the manager fields will not be appended to the resources
  -p, --path strings          Paths on disk from which to read the resources to push (default [./resources])
      --stop-on-error         Stop pushing resources when an error occurs
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

