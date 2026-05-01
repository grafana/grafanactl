# Design: `grafanactl help-tree` command

## Problem

grafanactl has ~26 leaf commands across 3 groups. An LLM must call `--help` on each one to understand the full CLI surface. This wastes tokens and round-trips. We need a single command that dumps the entire (or partial) command tree in a compact, token-efficient encoding.

## Decision

Add a new visible subcommand `grafanactl help-tree [command]` that prints a compact indented DSL showing all commands, positional args, flags with types/defaults/enums, and short descriptions.

## Invocation

```
grafanactl help-tree              # full tree
grafanactl help-tree resources    # subtree rooted at "resources"
grafanactl help-tree alerts       # subtree rooted at "alerts"
```

## Output Format

### Full tree example

```
grafanactl [--no-color] [-v|--verbose count] [--config path] [--context name]
  alerts                            # Manage Grafana alert rules
    export <UID> [-o json|yaml|text]
    get <UID> [-o json|yaml|text]
    history [--from dur=24h] [--to dur=now] [--limit int=1000] [-o json|yaml|text]
    instances [-o json|yaml|text]
    list [--state firing|pending|inactive|unknown] [-o json|yaml|text]
    noise-report [--period dur=7d] [--threshold int=5] [-o json|yaml|text]
    search --name <pattern> [-o json|yaml|text]
  config                            # View or manipulate configuration
    check
    current-context
    list-contexts [-o json|yaml|text]
    set <key> <value>
    unset <key>
    use-context <name>
    view [-o json|yaml|text]
  resources                         # Manipulate Grafana resources
    delete [SELECTOR]... [-p path=./resources] [--on-error ignore|fail|abort]
    edit [SELECTOR]... [-e editor]
    get [SELECTOR]... [-o json|yaml|text] [--all-versions]
    list [-o json|yaml|text]
    pull [SELECTOR]... [-p path=./resources] [-o json|yaml] [--all-versions]
    push [SELECTOR]... [-p path=./resources] [--dry-run] [--include-managed] [--max-concurrent int=10] [--on-error ignore|fail|abort]
    serve [SELECTOR]... [-p path=./resources] [--port int=3001] [--grafana-url url]
    validate [SELECTOR]... [-p path=./resources]
```

### Subtree example (`grafanactl help-tree resources`)

```
grafanactl resources [--config path] [--context name]  # Manipulate Grafana resources
  delete [SELECTOR]... [-p path=./resources] [--on-error ignore|fail|abort]
  edit [SELECTOR]... [-e editor]
  get [SELECTOR]... [-o json|yaml|text] [--all-versions]
  list [-o json|yaml|text]
  pull [SELECTOR]... [-p path=./resources] [-o json|yaml] [--all-versions]
  push [SELECTOR]... [-p path=./resources] [--dry-run] [--include-managed] [--max-concurrent int=10] [--on-error ignore|fail|abort]
  serve [SELECTOR]... [-p path=./resources] [--port int=3001] [--grafana-url url]
  validate [SELECTOR]... [-p path=./resources]
```

The root line shows the path to the subtree with inherited flags for full invocation context.

## Encoding Rules

### Structure
- Root command on line 1 with global flags
- Each subcommand indented 2 spaces per depth level
- Group commands (those with children) get a `# short description` comment
- Leaf commands show only their args and flags (description omitted for compactness)
- Commands sorted alphabetically within each group

### Positional args
- Required: `<name>`
- Optional: `[name]`
- Variadic: `[SELECTOR]...`
- Extracted from Cobra's `Use` field by stripping the command name prefix

### Flag formatting

| Case | Format | Example |
|------|--------|---------|
| Bool | `[--flag]` | `[--dry-run]` |
| String with default | `[--flag type=default]` | `[--period dur=7d]` |
| String required (no default) | `--flag <type>` | `--name <pattern>` |
| Int with default | `[--flag int=N]` | `[--max-concurrent int=10]` |
| Enum | `[--flag val1\|val2\|val3]` | `[--on-error ignore\|fail\|abort]` |
| Short alias | `-s\|--long` | `-p\|--path` |
| Slice type | `[-flag type=default]` | `[-p path=./resources]` |

### Flag classification
- Global/inherited flags shown once on the root line, not repeated per command
- Local flags shown inline on each command
- Hidden flags excluded
- The `-o|--output` pattern abbreviated to `-o json|yaml|text` for compactness

### Enum detection
Flags whose usage text contains a pattern like `val1|val2` or a list of values are detected as enums. The enum values are shown inline instead of a generic type name.

## Implementation

### New file: `cmd/grafanactl/root/help_tree.go`

Single file containing:
- `helpTreeCmd()` — creates and returns the `*cobra.Command`
- `formatTree(cmd, depth, writer)` — recursive tree formatter
- `formatFlags(flagSet)` — converts a flag set to compact inline notation
- `extractArgs(use)` — extracts positional arg notation from Cobra `Use` string
- `detectEnum(usage)` — parses enum values from flag usage text

### Registration

In `cmd/grafanactl/root/command.go`, add `rootCmd.AddCommand(helpTreeCmd())`.

### Approach
- Walk `cmd.Commands()` recursively, skipping hidden commands
- For each command, separate `LocalFlags()` from `InheritedFlags()`
- Format using the encoding rules above
- Output to stdout with no color (plain text always)
