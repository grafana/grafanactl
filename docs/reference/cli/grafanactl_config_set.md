## grafanactl config set

Set an single value in a configuration file

### Synopsis

Set an single value in a configuration file

PROPERTY_NAME is a dot-delimited reference to the value to unset. It can either represent a field or a map entry.

PROPERTY_VALUE is the new value to set.

```
grafanactl config set PROPERTY_NAME PROPERTY_VALUE [flags]
```

### Examples

```

	# Set the "server" field on the "dev-instance" context to "https://grafana-dev.example"
	grafanactl config set contexts.dev-instance.grafana.server https://grafana-dev.example

	# Disable the validation of the server's SSL certificate in the "dev-instance" context
	grafanactl config set contexts.dev-instance.grafana.insecure-skip-tls-verify true
```

### Options

```
  -h, --help   help for set
```

### Options inherited from parent commands

```
      --config string    Path to the configuration file to use
      --context string   Name of the context to use
      --no-color         Disable color output
  -v, --verbose count    Verbose mode. Multiple -v options increase the verbosity (maximum: 3).
```

### SEE ALSO

* [grafanactl config](grafanactl_config.md)	 - View or manipulate configuration settings

