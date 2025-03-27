## grafanactl config unset

Unset an single value in a configuration file

### Synopsis

Unset an single value in a configuration file.

PROPERTY_NAME is a dot-delimited reference to the value to unset. It can either represent a field or a map entry.

```
grafanactl config unset PROPERTY_NAME [flags]
```

### Examples

```

	# Unset the "foo" context
	main config unset contexts.foo

	# Unset the "insecure-skip-tls-verify" flag in the "dev-instance" context
	main config unset contexts.dev-instance.grafana.insecure-skip-tls-verify
```

### Options

```
  -h, --help   help for unset
```

### Options inherited from parent commands

```
      --config string    Path to the configuration file to use
      --context string   Name of the context to use
      --no-color         Disable color output
```

### SEE ALSO

* [grafanactl config](grafanactl_config.md)	 - View or manipulate configuration settings

