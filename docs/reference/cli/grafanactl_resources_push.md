## grafanactl resources push

Push resources to Grafana

### Synopsis

Push resources to Grafana using a specific format. See examples below for more details.

```
grafanactl resources push [RESOURCE_SELECTOR]... [flags]
```

### Examples

```

  Everything:

  main resources push

  All instances for a given kind(s):

  main resources push dashboards
  main resources push dashboards folders

  Single resource kind, one or more resource instances:

  main resources push dashboards/foo
  main resources push dashboards/foo,bar

  Single resource kind, long kind format:

  main resources push dashboard.dashboards/foo
  main resources push dashboard.dashboards/foo,bar

  Single resource kind, long kind format with version:

  main resources push dashboards.v1alpha1.dashboard.grafana.app/foo
  main resources push dashboards.v1alpha1.dashboard.grafana.app/foo,bar

  Multiple resource kinds, one or more resource instances:

  main resources push dashboards/foo folders/qux
  main resources push dashboards/foo,bar folders/qux,quux

  Multiple resource kinds, long kind format:

  main resources push dashboard.dashboards/foo folder.folders/qux
  main resources push dashboard.dashboards/foo,bar folder.folders/qux,quux

  Multiple resource kinds, long kind format with version:

  main resources push dashboards.v1alpha1.dashboard.grafana.app/foo folders.v1alpha1.folder.grafana.app/qux

```

### Options

```
  -d, --directory strings    Directories on disk from which to read the resources to push. (default [./resources])
      --dry-run              If set, the push operation will be simulated, without actually creating or updating any resources.
  -h, --help                 help for push
      --max-concurrent int   Maximum number of concurrent operations (default 10)
      --overwrite            Overwrite existing resources
      --stop-on-error        Stop pushing resources when an error occurs
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

