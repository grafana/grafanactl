# Plan: Investigate Gamma Service Excessive Info-Level Logging

## Task Description
The Gamma service (a Kubernetes-based extensible metadata system for Grafana, located at `../../gamma/main`) is generating excessive info-level log statements that flood Loki instances. The investigator must analyze the actual log volume from Gamma in the Gamma namespace using Loki, map high-volume log patterns back to the Gamma source code, and produce a detailed recommendation for each of the 44 info-level log statements: keep as-is, downgrade to debug level, or remove entirely.

Since `grafanactl` currently has no Loki/log querying capabilities, new `logs` commands must be built first to enable the investigation.

## Objective
When this plan is complete:
1. New `grafanactl logs` commands provide Loki query capabilities (query, stats, patterns, labels)
2. A new `investigate-excessive-logs` skill documents the investigation workflow
3. A complete analysis of Gamma's log output in Loki, mapping patterns to source code
4. A prioritized recommendation report for each info-level log statement with specific code change suggestions for the Gamma codebase

## Problem Statement
Gamma is a Kubernetes operator that manages relationships between Grafana AppPlatform resources. It runs reconciliation loops and Kubernetes watchers that generate info-level log lines on every watch event, reconciliation cycle, and validation. The 44 info-level log statements across 12 files likely produce enormous volume in a busy cluster because:

- **Watcher events** (8 log lines in `pkg/watchers/`) fire on EVERY Kubernetes watch event for RelationDefinition and AppRelationDefinition resources
- **SideWatcher events** (5 log lines in `pkg/engine/side_operator.go`) fire on every resource Add/Update/Delete with verbose field logging
- **Reconciliation lifecycle** (3 log lines in `pkg/engine/reconciler.go`) fire on every reconciliation cycle (debounce + reconcile + finish)
- **Relation processing** (6 log lines in `pkg/engine/relations.go`) fire during each reconciliation for every page of subjects/targets fetched
- **Orchestrator lifecycle** (4 log lines in `pkg/engine/orchestrator.go`) fire on operator create/run/stop
- **Validation results** (6 log lines in `pkg/validation/`) fire on every validation attempt

`grafanactl` currently has NO Loki querying capabilities, so new commands must be built to enable this investigation.

## Solution Approach
1. **Build `grafanactl logs` command group** with Loki API access via Grafana's datasource proxy (`/api/datasources/proxy/uid/{uid}/loki/api/v1/...`), following the same patterns established by the alerts commands
2. **Build `investigate-excessive-logs` skill** documenting the systematic workflow
3. **Run the investigation** using the new tools against live Loki data from the Gamma namespace
4. **Produce recommendations** mapping each high-volume log pattern to its source code location with a keep/downgrade/remove classification

## Relevant Files
Use these files to complete the task:

### Service Under Investigation: Gamma (`../../gamma/main`)

Gamma is a Go 1.24 Kubernetes operator using `grafana-app-sdk/logging` (wrapping `log/slog`). The 44 info-level log statements are distributed across:

| File | Count | Category | Volume Risk |
|------|-------|----------|-------------|
| `pkg/engine/side_operator.go` | 5 | K8s watch events (Add/Update/Delete/Skip) | **VERY HIGH** - fires per resource per event |
| `pkg/engine/relations.go` | 6 | Reconciliation internals (pagination, relation building) | **HIGH** - fires per reconcile cycle per relDef |
| `pkg/watchers/watcher_relationdefinition.go` | 4 | K8s watch events for RelationDefinitions | **HIGH** - fires per watch event |
| `pkg/watchers/watcher_apprelationdefinition.go` | 4 | K8s watch events for AppRelationDefinitions | **HIGH** - fires per watch event |
| `pkg/engine/orchestrator.go` | 4 | Operator lifecycle (create/run/stop) | **LOW** - fires on operator start/stop only |
| `pkg/engine/reconciler.go` | 3 | Reconciliation lifecycle (debounce/start/finish) | **MEDIUM** - fires per reconcile cycle |
| `cmd/gamma-tool/serve.go` | 6 | HTTP request processing, server startup | **LOW** - fires per web request |
| `cmd/operator/main.go` | 3 | Process startup/shutdown | **VERY LOW** - fires once |
| `pkg/asserts/operator.go` | 2 | Relation watcher events | **HIGH** - fires per relation change |
| `pkg/validation/relation_definitions.go` | 2 | Validation results | **MEDIUM** - fires per validation |
| `pkg/validation/app_relation_definition.go` | 2 | Validation results | **MEDIUM** - fires per validation |
| `pkg/validation/relation_types.go` | 2 | Validation events | **MEDIUM** - fires per validation |

### grafanactl Files to Create/Modify

- `cmd/grafanactl/root/command.go` - Register new `logs` command group (add import + `rootCmd.AddCommand`)
- `cmd/grafanactl/alerts/alerting_client.go` - Reference pattern for HTTP client with auth and Grafana API access
- `cmd/grafanactl/alerts/command.go` - Reference pattern for command group registration
- `cmd/grafanactl/alerts/list.go` - Reference pattern for table output with custom codec
- `cmd/grafanactl/alerts/export.go` - Reference pattern for direct HTTP API calls using `internal/httputils`
- `cmd/grafanactl/io/format.go` - IO options and codec registration pattern
- `internal/httputils/client.go` - HTTP client construction with TLS support
- `internal/config/` - Configuration and context loading

### New Files

- `cmd/grafanactl/logs/command.go` - Logs command group with subcommand registration
- `cmd/grafanactl/logs/loki_client.go` - Loki HTTP client via Grafana datasource proxy
- `cmd/grafanactl/logs/loki_client_test.go` - Tests for Loki client response parsing
- `cmd/grafanactl/logs/query.go` - `logs query` command: execute LogQL queries
- `cmd/grafanactl/logs/query_test.go` - Tests for query command
- `cmd/grafanactl/logs/stats.go` - `logs stats` command: get log volume statistics
- `cmd/grafanactl/logs/stats_test.go` - Tests for stats command
- `cmd/grafanactl/logs/patterns.go` - `logs patterns` command: detect high-volume log patterns
- `cmd/grafanactl/logs/patterns_test.go` - Tests for patterns command
- `cmd/grafanactl/logs/labels.go` - `logs labels` command: list label names and values
- `cmd/grafanactl/logs/labels_test.go` - Tests for labels command
- `skills/investigate-excessive-logs/SKILL.md` - Investigation skill document

Take implementation details from the following repositories. Some of the access patterns that you'll need or are trying to solve for have already been implemented here. Use those as "donor DNA".
- @../../mcp-grafana/main - **PRIMARY REFERENCE**: `tools/loki.go` has the complete Loki API client with all endpoints, response structs, query patterns, and pattern detection. `tools/search_logs.go` has LogQL generation patterns.
- @../../grafana-assistant-app/main - Investigation format patterns and agent orchestration
- @../../app-env-cli/main - CLI patterns and HTTP client patterns

## Implementation Phases
### Phase 1: Foundation
Build the Loki HTTP client (`loki_client.go`) and the `logs` command group scaffold. The Loki client proxies through Grafana's datasource proxy API at `/api/datasources/proxy/uid/{datasource-uid}/loki/api/v1/...`, reusing the same HTTP client and auth pattern from `alerting_client.go`.

Key API endpoints (all via Grafana datasource proxy):
- `GET /api/datasources/proxy/uid/{uid}/loki/api/v1/labels` - List label names
- `GET /api/datasources/proxy/uid/{uid}/loki/api/v1/label/{name}/values` - Label values
- `GET /api/datasources/proxy/uid/{uid}/loki/api/v1/query_range` - Execute LogQL range queries
- `GET /api/datasources/proxy/uid/{uid}/loki/api/v1/query` - Execute LogQL instant queries
- `GET /api/datasources/proxy/uid/{uid}/loki/api/v1/index/stats` - Get stream statistics
- `GET /api/datasources/proxy/uid/{uid}/loki/api/v1/patterns` - Detect log patterns

Response structures (reference `../../mcp-grafana/main/tools/loki.go`):
```go
// lokiQueryResponse wraps Loki query API responses
type lokiQueryResponse struct {
    Status string `json:"status"`
    Data   struct {
        ResultType string          `json:"resultType"` // "streams", "vector", "matrix"
        Result     json.RawMessage `json:"result"`
    } `json:"data"`
}

// lokiLogStream represents a stream of log entries
type lokiLogStream struct {
    Stream map[string]string   `json:"stream"`
    Values [][]json.RawMessage `json:"values"` // [[ts_nanos_string, log_line], ...]
}

// lokiStats represents volume statistics
type lokiStats struct {
    Streams int `json:"streams"`
    Chunks  int `json:"chunks"`
    Entries int `json:"entries"`
    Bytes   int `json:"bytes"`
}

// lokiLabelResponse wraps label API responses
type lokiLabelResponse struct {
    Status string   `json:"status"`
    Data   []string `json:"data,omitempty"`
}

// lokiPatternsResponse wraps patterns API responses
type lokiPatternsResponse struct {
    Status string `json:"status"`
    Data   []struct {
        Pattern string     `json:"pattern"`
        Samples [][2]int64 `json:"samples"` // [[timestamp, count], ...]
    } `json:"data"`
}

// lokiPattern represents a detected pattern with total count
type lokiPattern struct {
    Pattern    string `json:"pattern"`
    TotalCount int64  `json:"totalCount"`
}
```

### Phase 2: Core Implementation
Build four commands in parallel:

**2a. `logs query`** - Execute LogQL queries against Loki via Grafana datasource proxy.
- Flags: `--datasource-uid` (required), `--logql` (required), `--from` (default 1h), `--to` (default now), `--limit` (default 100, max 5000), `--direction` (forward/backward, default backward)
- Table output: TIMESTAMP, LABELS, LINE
- Supports `-o json/yaml` output

**2b. `logs stats`** - Get log volume statistics for a label selector.
- Flags: `--datasource-uid` (required), `--selector` (required LogQL label matcher), `--from` (default 1h), `--to` (default now)
- Table output: STREAMS, CHUNKS, ENTRIES, BYTES
- Supports `-o json/yaml` output

**2c. `logs patterns`** - Detect high-volume log patterns (CRITICAL for this investigation).
- Flags: `--datasource-uid` (required), `--selector` (required LogQL label matcher), `--from` (default 1h), `--to` (default now), `--step` (default 5m)
- Table output sorted by count descending: PATTERN, COUNT
- Supports `-o json/yaml` output

**2d. `logs labels`** - List available label names and values.
- Flags: `--datasource-uid` (required), `--label` (optional, shows values for a specific label), `--from` (default 1h), `--to` (default now)
- Table output: LABEL (or VALUE if --label specified)
- Supports `-o json/yaml` output

### Phase 3: Integration & Polish
- Register `logs` command in `root/command.go`
- Run `make build` to rebuild the binary
- Create the `investigate-excessive-logs` skill
- Run the investigation using the new tools
- Produce the final recommendations report

## Team Orchestration

- You operate as the Grafana investigator trying to solve for excessive info-level logging in the Gamma service
- You are assigned ONE type of grafana investigation. Focus entirely on completing it.
- Draft a claude-code skill in the @skills directory that details the steps needed to accomplish the task. If the `grafanactl` commands do not exist, imagine what you might want those commands to be. These can be written.
- Run `bin/grafanactl help-tree` in order to ensure you have the tools needed to accomplish your task. You may run `grafanactl help-tree [command]` on subcommands if necessary.
- If you do not see the tools required in `grafanactl` in order to accomplish your task, you will need to spawn subagents to add new tools to `grafanactl` in order to accomplish your investigation.
- Tell your sub-agents to Reference @../../mcp-grafana/main and @../../grafana-assistant-app/main and @../../app-env-cli/main for inspiration and examples about how to call various APIs in Grafana related to this task.
- You will create tasks and delegate them to sub-agents (using .claude/agents/team/builder.md and .claude/agents/team/validator.md) in order to implement the tools that you need with the documentation and tests.
- Spawn one builder and one validator per tool that requires creation.
- Subagents will build and validate their work. You will need to run `make build` in order to ensure `grafanactl` is up to date when those tasks complete.
- Use `TaskGet` to read your assigned task details if a task ID is provided.
- Use `TaskCreate` to create new tasks for your sub-agents in order to implement new features in grafanactl that are required for you to accomplish your task.
- When finished, use `TaskUpdate` to mark your task as `completed`.
- If you encounter blockers, update the task with details but do NOT stop - attempt to resolve or work around.
- Spawn other agents to implement tools that are not available to you within grafanactl in order for you to accomplish your work.
- Stay focused on the single task. Do not expand scope.
- `grafanactl` is already configured to access a live system. See `grafanactl resources list` to check that.
- You're responsible for deploying the right team members with the right context to execute the plan.
- IMPORTANT: You NEVER operate directly on the codebase. You use `Task` and `Task*` tools to deploy team members to the investigation, and perhaps the delegating of building, validating, testing, deploying, and other tasks.
  - This is critical. Your job is to act as a high level director of the team, not a builder.
  - Your role is to validate all work is going well and make sure the team is on track to complete the plan.
  - You'll orchestrate this by using the Task* Tools to manage coordination between the team members.
  - Communication is paramount. You'll use the Task* Tools to communicate with the team members and ensure they're on track to complete the plan.
- Take note of the session id of each team member. This is how you'll reference them.
- However, use the tools that are built for you by your sub-agents in order to validate the task at hand.

### Team Members

- Builder
  - Name: builder-loki-client
  - Role: Build the Loki HTTP client (`cmd/grafanactl/logs/loki_client.go`) that wraps Grafana's datasource proxy Loki API endpoints, including all response struct definitions, JSON parsing, authenticated requests, and label/query/stats/patterns methods. Reference `../../mcp-grafana/main/tools/loki.go` for exact API patterns and `cmd/grafanactl/alerts/alerting_client.go` for the grafanactl HTTP client pattern.
  - Agent Type: builder
  - Resume: true

- Builder
  - Name: builder-logs-command-group
  - Role: Build the `logs` command group scaffold (`cmd/grafanactl/logs/command.go`) and register it in `cmd/grafanactl/root/command.go`. Follow the exact pattern of `cmd/grafanactl/alerts/command.go` for command group structure and `cmd/grafanactl/root/command.go` for root registration.
  - Agent Type: builder
  - Resume: true

- Builder
  - Name: builder-logs-query-cmd
  - Role: Build the `logs query` command that executes LogQL queries against Loki via the Loki client. Supports `--datasource-uid`, `--logql`, `--from`, `--to`, `--limit`, `--direction` flags with table and JSON/YAML output. Follow `cmd/grafanactl/alerts/list.go` for the opts pattern and custom table codec.
  - Agent Type: builder
  - Resume: true

- Builder
  - Name: builder-logs-stats-cmd
  - Role: Build the `logs stats` command that gets log volume statistics from Loki's index/stats API. Supports `--datasource-uid`, `--selector`, `--from`, `--to` flags with table and JSON/YAML output.
  - Agent Type: builder
  - Resume: true

- Builder
  - Name: builder-logs-patterns-cmd
  - Role: Build the `logs patterns` command that detects high-volume log patterns using Loki's patterns API. Supports `--datasource-uid`, `--selector`, `--from`, `--to`, `--step` flags. Output sorted by count descending. This is the MOST CRITICAL command for the investigation.
  - Agent Type: builder
  - Resume: true

- Builder
  - Name: builder-logs-labels-cmd
  - Role: Build the `logs labels` command that lists Loki label names and values. Supports `--datasource-uid`, `--label` (optional for values), `--from`, `--to` flags.
  - Agent Type: builder
  - Resume: true

- Validator
  - Name: validator-logs-commands
  - Role: Validate all new logs commands compile, pass tests, and work correctly with `bin/grafanactl`. Run `make build`, `go test`, `make lint`, and verify help output for all new subcommands.
  - Agent Type: validator
  - Resume: false

- Builder
  - Name: builder-skill
  - Role: Create the `investigate-excessive-logs` skill document at `skills/investigate-excessive-logs/SKILL.md` following the pattern of `skills/investigate-noisy-alerts/SKILL.md`. The skill should document the systematic workflow for investigating excessive logging using the new `grafanactl logs` commands.
  - Agent Type: builder
  - Resume: true

- Builder
  - Name: investigator-agent
  - Role: The actual Grafana investigator who uses the built tools (`bin/grafanactl logs ...`) to examine live Loki data from the Gamma namespace, cross-references log patterns with Gamma source code at `../../gamma/main`, and produces the final investigation report with keep/downgrade/remove recommendations for each info-level log statement.
  - Agent Type: general-purpose
  - Resume: true

## Step by Step Tasks

- IMPORTANT: Execute every step in order, top to bottom. Each task maps directly to a `TaskCreate` call.
- Before you start, run `TaskCreate` to create the initial task list that all team members can see and execute.

### 1. Build Loki HTTP Client
- **Task ID**: build-loki-client
- **Depends On**: none
- **Assigned To**: builder-loki-client
- **Agent Type**: builder
- **Parallel**: true (no dependencies)
- Create `cmd/grafanactl/logs/loki_client.go` with:
  - Response struct types: `lokiQueryResponse`, `lokiLogStream`, `lokiMetricSample`, `lokiStats`, `lokiLabelResponse`, `lokiPatternsResponse`, `lokiPattern` (reference `../../mcp-grafana/main/tools/loki.go` lines 28-60, 304-326, 694-765 for exact field names and JSON tags)
  - `lokiClient` struct wrapping `*http.Client`, base URL, and datasource UID
  - `newLokiClient(ctx context.Context, gCtx *config.Context, datasourceUID string) (*lokiClient, error)` - constructs client using Grafana datasource proxy URL pattern: `{server}/api/datasources/proxy/uid/{datasource-uid}`. Uses `httputils.NewHTTPClient(gCtx)` for the underlying HTTP client. Sets auth headers (Bearer token or basic auth) and Org-ID header following the pattern from `cmd/grafanactl/alerts/export.go` lines 105-113.
  - `(c *lokiClient) queryLogs(ctx context.Context, logql, start, end string, limit int, direction string) ([]logEntry, error)` - calls `/loki/api/v1/query_range`
  - `(c *lokiClient) queryStats(ctx context.Context, selector, start, end string) (*lokiStats, error)` - calls `/loki/api/v1/index/stats`
  - `(c *lokiClient) queryPatterns(ctx context.Context, selector, start, end, step string) ([]lokiPattern, error)` - calls `/loki/api/v1/patterns`
  - `(c *lokiClient) listLabels(ctx context.Context, start, end string) ([]string, error)` - calls `/loki/api/v1/labels`
  - `(c *lokiClient) listLabelValues(ctx context.Context, labelName, start, end string) ([]string, error)` - calls `/loki/api/v1/label/{name}/values`
  - All methods parse duration strings ("1h", "7d", "30d") into RFC3339/nanosecond timestamps for the Loki API
  - Follow the HTTP client pattern from `cmd/grafanactl/alerts/alerting_client.go` for constructing requests with proper auth
- Create `cmd/grafanactl/logs/loki_client_test.go` with unit tests for:
  - Response struct JSON parsing (use `encoding/json` to decode test fixtures)
  - URL construction verification
  - Time duration parsing
- Run `go test ./cmd/grafanactl/logs/...` to verify

### 2. Build Logs Command Group Scaffold
- **Task ID**: build-logs-command-group
- **Depends On**: none
- **Assigned To**: builder-logs-command-group
- **Agent Type**: builder
- **Parallel**: true (no dependencies, can run alongside task 1)
- Create `cmd/grafanactl/logs/command.go`:
  - Follow EXACT pattern of `cmd/grafanactl/alerts/command.go`
  - `Command()` function returning `*cobra.Command` with `Use: "logs"`, `Short: "Query and analyze logs from Loki datasources"`
  - Register config options on persistent flags
  - Initially register placeholder subcommands (will be replaced as builders complete)
- Modify `cmd/grafanactl/root/command.go`:
  - Add import: `"github.com/grafana/grafanactl/cmd/grafanactl/logs"`
  - Add: `rootCmd.AddCommand(logs.Command())`
- Run `make build` to verify compilation

### 3. Build Logs Query Command
- **Task ID**: build-logs-query-cmd
- **Depends On**: build-loki-client, build-logs-command-group
- **Assigned To**: builder-logs-query-cmd
- **Agent Type**: builder
- **Parallel**: true (can run alongside tasks 4, 5, 6 once dependencies met)
- Create `cmd/grafanactl/logs/query.go`:
  - New `query` subcommand using the Loki client's `queryLogs()` method
  - Flags: `--datasource-uid` (required), `--logql` (required), `--from` (default "1h"), `--to` (default "now"), `--limit` (default 100, max 5000), `--direction` (forward/backward, default backward)
  - From/to accept duration strings like "1h", "7d" (relative to now) OR RFC3339 timestamps
  - Custom table codec with columns: TIMESTAMP, LABELS, LINE
  - Support `-o json` and `-o yaml` output formats
  - Follow the same opts pattern as `cmd/grafanactl/alerts/list.go` (struct with setup/Validate methods)
- Create `cmd/grafanactl/logs/query_test.go` with unit tests
- Register command in `command.go`: `cmd.AddCommand(queryCmd(configOpts))`
- Run `go test ./cmd/grafanactl/logs/...` to verify

### 4. Build Logs Stats Command
- **Task ID**: build-logs-stats-cmd
- **Depends On**: build-loki-client, build-logs-command-group
- **Assigned To**: builder-logs-stats-cmd
- **Agent Type**: builder
- **Parallel**: true (can run alongside tasks 3, 5, 6)
- Create `cmd/grafanactl/logs/stats.go`:
  - New `stats` subcommand using the Loki client's `queryStats()` method
  - Flags: `--datasource-uid` (required), `--selector` (required LogQL label matcher, e.g., `{namespace="gamma"}`), `--from` (default "1h"), `--to` (default "now")
  - Custom table codec with columns: STREAMS, CHUNKS, ENTRIES, BYTES
  - Support `-o json` and `-o yaml` output
- Create `cmd/grafanactl/logs/stats_test.go` with unit tests
- Register command in `command.go`: `cmd.AddCommand(statsCmd(configOpts))`
- Run `go test ./cmd/grafanactl/logs/...` to verify

### 5. Build Logs Patterns Command
- **Task ID**: build-logs-patterns-cmd
- **Depends On**: build-loki-client, build-logs-command-group
- **Assigned To**: builder-logs-patterns-cmd
- **Agent Type**: builder
- **Parallel**: true (can run alongside tasks 3, 4, 6)
- Create `cmd/grafanactl/logs/patterns.go`:
  - New `patterns` subcommand using the Loki client's `queryPatterns()` method
  - Flags: `--datasource-uid` (required), `--selector` (required LogQL label matcher), `--from` (default "1h"), `--to` (default "now"), `--step` (default "5m")
  - Custom table codec with columns: COUNT, PATTERN - sorted by count descending (highest volume first)
  - Support `-o json` and `-o yaml` output for machine consumption
  - This is the MOST CRITICAL command for the investigation - it identifies which log patterns consume the most volume
  - Reference `../../mcp-grafana/main/tools/loki.go` lines 718-765 for the patterns API implementation
- Create `cmd/grafanactl/logs/patterns_test.go` with unit tests for pattern sorting and counting
- Register command in `command.go`: `cmd.AddCommand(patternsCmd(configOpts))`
- Run `go test ./cmd/grafanactl/logs/...` to verify

### 6. Build Logs Labels Command
- **Task ID**: build-logs-labels-cmd
- **Depends On**: build-loki-client, build-logs-command-group
- **Assigned To**: builder-logs-labels-cmd
- **Agent Type**: builder
- **Parallel**: true (can run alongside tasks 3, 4, 5)
- Create `cmd/grafanactl/logs/labels.go`:
  - New `labels` subcommand using the Loki client's `listLabels()` and `listLabelValues()` methods
  - Flags: `--datasource-uid` (required), `--label` (optional - when provided, shows values for that label instead of label names), `--from` (default "1h"), `--to` (default "now")
  - Custom table codec: single column LABEL (or VALUE when --label is specified)
  - Support `-o json` and `-o yaml` output
- Create `cmd/grafanactl/logs/labels_test.go` with unit tests
- Register command in `command.go`: `cmd.AddCommand(labelsCmd(configOpts))`
- Run `go test ./cmd/grafanactl/logs/...` to verify

### 7. Validate All Logs Commands
- **Task ID**: validate-logs-commands
- **Depends On**: build-loki-client, build-logs-command-group, build-logs-query-cmd, build-logs-stats-cmd, build-logs-patterns-cmd, build-logs-labels-cmd
- **Assigned To**: validator-logs-commands
- **Agent Type**: validator
- **Parallel**: false (must wait for all builders)
- Run `make build` to ensure the binary compiles with all new commands
- Run `go test ./cmd/grafanactl/logs/...` to ensure all tests pass
- Run `bin/grafanactl logs --help` and verify all subcommands appear: query, stats, patterns, labels
- Run `bin/grafanactl logs query --help` and verify `--datasource-uid`, `--logql`, `--from`, `--to`, `--limit`, `--direction` flags
- Run `bin/grafanactl logs stats --help` and verify `--datasource-uid`, `--selector` flags
- Run `bin/grafanactl logs patterns --help` and verify `--datasource-uid`, `--selector`, `--step` flags
- Run `bin/grafanactl logs labels --help` and verify `--datasource-uid`, `--label` flags
- Run `bin/grafanactl help-tree logs` to verify the command tree output
- Run `make lint` to check for linting issues
- Report pass/fail status for each check

### 8. Create Investigation Skill
- **Task ID**: create-skill
- **Depends On**: validate-logs-commands
- **Assigned To**: builder-skill
- **Agent Type**: builder
- **Parallel**: false (needs validated commands to document accurately)
- Create `skills/investigate-excessive-logs/SKILL.md` following the pattern of `skills/investigate-noisy-alerts/SKILL.md`
- The skill should document:
  - When to use (excessive logging, log volume investigation, Loki cost reduction)
  - Prerequisites (`grafanactl` configured with context, Loki datasource UID known)
  - Available tools (the `grafanactl logs` command surface from `help-tree logs`)
  - Systematic workflow: Discovery (labels) → Volume Assessment (stats) → Pattern Analysis (patterns) → Drill-Down (query) → Source Mapping → Recommendations
  - Quick reference table of commands
  - Common mistakes

### 9. Run Excessive Logging Investigation
- **Task ID**: run-investigation
- **Depends On**: validate-logs-commands, create-skill
- **Assigned To**: investigator-agent
- **Agent Type**: general-purpose
- **Parallel**: false (must wait for validated tools and skill)
- **Step 1: Discover** - Run `bin/grafanactl logs labels --datasource-uid <UID>` to find available labels, then `bin/grafanactl logs labels --datasource-uid <UID> --label namespace` to confirm "gamma" namespace exists
- **Step 2: Volume Assessment** - Run `bin/grafanactl logs stats --datasource-uid <UID> --selector '{namespace="gamma"}' --from 24h` to get overall volume, then `bin/grafanactl logs stats --datasource-uid <UID> --selector '{namespace="gamma"} | json | level="info"' --from 24h` for info-level volume specifically
- **Step 3: Pattern Analysis** - Run `bin/grafanactl logs patterns --datasource-uid <UID> --selector '{namespace="gamma"}' --from 24h` to identify the highest-volume log patterns, then `bin/grafanactl logs patterns --datasource-uid <UID> --selector '{namespace="gamma"} | json | level="info"' --from 24h` for info-level patterns
- **Step 4: Drill-Down** - For each high-volume pattern, run `bin/grafanactl logs query --datasource-uid <UID> --logql '{namespace="gamma"} |= "<pattern fragment>"' --from 1h --limit 10` to see actual log lines
- **Step 5: Source Code Mapping** - Read Gamma source files to map each detected pattern to its exact source location:
  - `../../gamma/main/pkg/engine/side_operator.go` (lines 196, 212, 236, 258, 263)
  - `../../gamma/main/pkg/engine/relations.go` (lines 80, 103, 183, 193, 245, 357)
  - `../../gamma/main/pkg/engine/reconciler.go` (lines 131, 139, 185)
  - `../../gamma/main/pkg/engine/orchestrator.go` (lines 51, 63, 75, 85)
  - `../../gamma/main/pkg/watchers/watcher_relationdefinition.go` (lines 43, 60, 77, 93)
  - `../../gamma/main/pkg/watchers/watcher_apprelationdefinition.go` (lines 42, 59, 76, 92)
  - `../../gamma/main/pkg/asserts/operator.go` (lines 98, 118)
  - `../../gamma/main/pkg/validation/relation_definitions.go` (lines 54, 67)
  - `../../gamma/main/pkg/validation/app_relation_definition.go` (lines 50, 63)
  - `../../gamma/main/pkg/validation/relation_types.go` (lines 28, 40)
  - `../../gamma/main/cmd/gamma-tool/serve.go` (lines 269, 288, 294, 308, 322, 360)
  - `../../gamma/main/cmd/operator/main.go` (lines 235, 244, 254)
- **Step 6: Classify** - For each info-level log statement, determine:
  - **KEEP** (as Info): Log provides essential operational visibility that would be missed at debug level
  - **DOWNGRADE** (to Debug): Log is useful for debugging but generates too much volume at info level; moving to debug eliminates it from default production logging
  - **REMOVE**: Log provides no useful information even for debugging
- **Step 7: Report** - Produce a structured report with:
  - Executive summary (total log volume, info-level percentage, top patterns)
  - Per-file analysis with each log line, its volume, classification, and rationale
  - Specific code change recommendations (exact line numbers and replacement code)
  - Estimated volume reduction from implementing recommendations

### 10. Final Validation
- **Task ID**: validate-all
- **Depends On**: build-loki-client, build-logs-command-group, build-logs-query-cmd, build-logs-stats-cmd, build-logs-patterns-cmd, build-logs-labels-cmd, validate-logs-commands, create-skill, run-investigation
- **Assigned To**: validator-logs-commands
- **Agent Type**: validator
- **Parallel**: false
- Run all validation commands
- Verify acceptance criteria met
- Confirm investigation report was produced with specific recommendations for Gamma code changes
- Verify the skill document is complete and accurate

## Acceptance Criteria
- `bin/grafanactl logs query --datasource-uid <UID> --logql '{namespace="gamma"}'` returns log entries from the Gamma namespace
- `bin/grafanactl logs stats --datasource-uid <UID> --selector '{namespace="gamma"}'` returns volume statistics
- `bin/grafanactl logs patterns --datasource-uid <UID> --selector '{namespace="gamma"}'` returns detected patterns sorted by count
- `bin/grafanactl logs labels --datasource-uid <UID>` returns available label names
- All new commands support `-o json` and `-o yaml` output formats
- All tests pass: `go test ./cmd/grafanactl/logs/...`
- `make build` succeeds
- `make lint` passes (or only has pre-existing warnings)
- `bin/grafanactl help-tree logs` shows complete command tree
- `skills/investigate-excessive-logs/SKILL.md` exists and documents the workflow
- Investigation report produced with keep/downgrade/remove classification for each of the 44 info-level log statements in Gamma
- Report includes specific code change recommendations with file paths and line numbers
- Report includes estimated volume reduction

## Validation Commands
Execute these commands to validate the task is complete:

- `make build` - Verify the binary compiles with all new commands
- `go test ./cmd/grafanactl/logs/...` - Run all logs command tests
- `bin/grafanactl logs --help` - Verify all subcommands are registered
- `bin/grafanactl logs query --help` - Verify query command with --datasource-uid, --logql flags
- `bin/grafanactl logs stats --help` - Verify stats command with --datasource-uid, --selector flags
- `bin/grafanactl logs patterns --help` - Verify patterns command with --datasource-uid, --selector, --step flags
- `bin/grafanactl logs labels --help` - Verify labels command with --datasource-uid, --label flags
- `bin/grafanactl help-tree logs` - Verify command tree
- `make lint` - Check for linting issues
- `test -f skills/investigate-excessive-logs/SKILL.md` - Verify skill document exists

## Notes
- The Loki API is accessed via Grafana's datasource proxy, NOT directly. All requests go through `{grafana-server}/api/datasources/proxy/uid/{datasource-uid}/loki/api/v1/...`. This means the auth is Grafana auth (API token or basic auth), not Loki auth.
- The `--datasource-uid` flag is required for all logs commands. Users can discover Loki datasource UIDs by running `grafanactl resources list datasources` and looking for Loki-type datasources, or via the Grafana UI.
- The patterns API (`/loki/api/v1/patterns`) is the key tool for this investigation. It uses Loki's built-in pattern detection to group similar log lines and count them. This directly maps to identifying which log statements produce the most volume.
- The Gamma operator's default log level is "error" (set in `cmd/gamma-tool/main.go:25`), but in the Kubernetes deployment the environment or command-line flags likely set it to "info" or "debug", causing all info-level logs to be emitted.
- Gamma uses two logging mechanisms: `logging.FromContext(ctx).Info(...)` (grafana-app-sdk wrapper) and `slog.Info(...)` (direct stdlib). Both produce structured JSON logs. The patterns API should detect both.
- The mcp-grafana codebase at `../../mcp-grafana/main/tools/loki.go` is the BEST reference for Loki API implementation. Builders MUST read this file.
- The existing alerting commands at `cmd/grafanactl/alerts/` are the BEST reference for grafanactl command patterns (opts struct, custom codecs, command registration, HTTP client usage). Builders MUST reference these.
- HTTP client timeout is 10 seconds by default (`internal/httputils/client.go:31`). For large Loki queries this may need to be increased. Consider using a 60-second timeout for the Loki client specifically.
- The investigation agent will need to know the Loki datasource UID. If discovery fails, the investigator should try `grafanactl resources list datasources -o json` to find Loki datasources.
