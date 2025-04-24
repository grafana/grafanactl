## grafanactl resources delete

Delete resources from Grafana

### Synopsis

Delete resources from Grafana.

```
grafanactl resources delete [RESOURCE_SELECTOR]... [flags]
```

### Examples

```

	# Delete a single dashboard
	grafanactl resources delete dashboards/some-dashboard

	# Delete multiple dashboards
	grafanactl resources delete dashboards/some-dashboard,other-dashboard

	# Delete a dashboard and a folder
	grafanactl resources delete dashboards/some-dashboard folders/some-folder

	# Delete every dashboard
	grafanactl resources delete dashboards --all

	# Delete every resource from the given directory
	grafanactl resources delete -d ./resources/Dashboard

```

### Options

```
      --all                  Delete all resources of the specified resource types
  -d, --directory strings    Directories on disk containing the resources to delete
      --dry-run              If set, the delete operation will be simulated
  -h, --help                 help for delete
      --max-concurrent int   Maximum number of concurrent operations (default 10)
      --stop-on-error        Stop pulling resources when an error occurs
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

