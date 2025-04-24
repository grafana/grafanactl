## grafanactl resources edit

Edit resources from Grafana

### Synopsis

Edit resources from Grafana using the default editor.

This command allows the edition of any resource that can be accessed by this CLI tool.

It will open the default editor as configured by the EDITOR environment variable, or fall back to 'vi' for Linux or 'notepad' for Windows.
The editor will be started in the shell set by the SHELL environment variable. If undefined, '/bin/bash' is used for Linux or 'cmd' for Windows.

The edition will be cancelled if no changes are written to the file or if the file after edition is empty.


```
grafanactl resources edit RESOURCE_SELECTOR [flags]
```

### Examples

```

	# Editing a dashboard
	grafanactl resources dashboard/foo

	# Editing a dashboard in JSON
	grafanactl resources -o json dashboard/foo

	# Using an alternative editor
	EDITOR=nvim grafanactl resources dashboard/foo

```

### Options

```
  -h, --help            help for edit
  -o, --output string   Output format. One of: json, yaml (default "json")
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

