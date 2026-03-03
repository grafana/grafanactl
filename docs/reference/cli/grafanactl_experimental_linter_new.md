## grafanactl experimental linter new

Creates a new resource linter

### Synopsis

Creates a new resource linter.

```
grafanactl experimental linter new resource-type name [flags]
```

### Examples

```

	# Creates a new dashboard linter in the current directory:

	grafanactl experimental linter new dashboard test-linter

	# Creates a new dashboard linter in another directory:

	grafanactl experimental linter new dashboard test-linter -o custom-rules

```

### Options

```
  -c, --category string   Rule category (default "idiomatic")
  -h, --help              help for new
  -o, --output string     Output directory
```

### Options inherited from parent commands

```
      --no-color        Disable color output
  -v, --verbose count   Verbose mode. Multiple -v options increase the verbosity (maximum: 3).
```

### SEE ALSO

* [grafanactl experimental linter](grafanactl_experimental_linter.md)	 - Lint Grafana resources

