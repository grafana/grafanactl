## grafanactl config view

Display the current configuration

```
grafanactl config view [flags]
```

### Examples

```

	grafanactl config view
```

### Options

```
  -h, --help            help for view
      --minify          Remove all information not used by current-context from the output
  -o, --output string   Output format. One of: json, yaml (default "yaml")
      --raw             Display sensitive information
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

