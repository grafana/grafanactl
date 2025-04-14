## grafanactl resources serve

Serve Grafana resources locally

### Synopsis

Serve Grafana resources locally.

The server started by this command makes it easy to explore and review resources
locally.

While resources are loaded from disk, the server will use the Grafana instance
described in the current context to access some data (example: to run queries
when previewing dashboards).

Note on NFS/SMB support for --watch: fsnotify requires support from underlying
OS to work. The current NFS and SMB protocols does not provide network level
support for file notifications.


```
grafanactl resources serve [RESOURCE_DIR]... [flags]
```

### Examples

```

	# Serve resources from a directory:
	grafanactl resources serve ./resources

	# Serve resources from a directory and watch for changes:
	grafanactl resources serve ./resources --watch ./resources

	# Serve resources from a script that outputs a JSON resource and watch for changes:
	# Note: the Grafana Foundation SDK can be used to generate dashboards (https://grafana.github.io/grafana-foundation-sdk/)
	grafanactl resources serve --script 'go run dashboard-generator/*.go' --watch ./dashboard-generator --script-format json

```

### Options

```
      --address string         Address to bind (default "0.0.0.0")
  -h, --help                   help for serve
      --max-concurrent int     Maximum number of concurrent operations (default 10)
      --port int               Port on which the server will listen (default 8080)
  -S, --script string          Script to execute to generate a resource
  -f, --script-format string   Format of the data returned by the script (default "yaml")
  -w, --watch stringArray      Paths to watch for changes
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

