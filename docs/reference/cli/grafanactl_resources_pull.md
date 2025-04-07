## grafanactl resources pull

Pull resources from Grafana

### Synopsis

Pull resources from Grafana using a specific format. See examples below for more details.

```
grafanactl resources pull RESOURCES_PATHS [flags]
```

### Examples

```

  Everything:

  main resources pull

  All instances for a given kind(s):

  main resources pull dashboards
  main resources pull dashboards folders

  Single resource kind, one or more resource instances:

  main resources pull dashboards/foo
  main resources pull dashboards/foo,bar

  Single resource kind, long kind format:

  main resources pull dashboard.dashboards/foo
  main resources pull dashboard.dashboards/foo,bar

  Single resource kind, long kind format with version:

  main resources pull dashboards.v1alpha1.dashboard.grafana.app/foo
  main resources pull dashboards.v1alpha1.dashboard.grafana.app/foo,bar

  Multiple resource kinds, one or more resource instances:

  main resources pull dashboards/foo folders/qux
  main resources pull dashboards/foo,bar folders/qux,quux

  Multiple resource kinds, long kind format:

  main resources pull dashboard.dashboards/foo folder.folders/qux
  main resources pull dashboard.dashboards/foo,bar folder.folders/qux,quux

  Multiple resource kinds, long kind format with version:

  main resources pull dashboards.v1alpha1.dashboard.grafana.app/foo folders.v1alpha1.folder.grafana.app/qux

```

### Options

```
      --continue-on-error   Continue pulling resources even if an error occurs
  -d, --directory string    Directory on disk in which the resources will be written. If left empty, nothing will be written on disk and resource details will be printed on stdout
  -h, --help                help for pull
  -o, --output string       Output format. One of: json, text, yaml (default "text")
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

