## grafanactl resources push

Push resources to Grafana

### Synopsis

Push resources from Grafana.

TODO: more information.


```
grafanactl resources push RESOURCES_PATH [flags]
```

### Examples

```

	main resources push
```

### Options

```
      --continue-on-error   Continue pushing resources even if an error occurs
  -h, --help                help for push
      --kind stringArray    Resource kinds to push. If omitted, all supported kinds will be pulled
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

