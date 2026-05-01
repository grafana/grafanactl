# Plan: Investigate Firing Alerts for Noise vs Signal

## Task Description
A Grafana investigator needs to examine the current set of firing alerts to determine which are noisy (firing unnecessarily, frequently flapping) and which are meaningful (indicating real problems). The investigator will use `bin/grafanactl` as the primary tool to interact with a live Grafana instance. Since `grafanactl` currently only has basic alert rule management (list, get, search, export from the provisioning API), new commands must be built to surface runtime alert state, firing instances, and alert history data needed for noise analysis.

## Objective
When this plan is complete, the investigator will have:
1. New `grafanactl` commands that provide runtime alert state, firing instances, and alert state history
2. A complete noise analysis of all currently firing alerts, categorizing each as "noisy" or "meaningful"
3. A written report with recommendations for which alerts to silence, tune, or keep

## Problem Statement
The current `grafanactl alerts` commands only interact with the Provisioning API, which provides rule configuration (title, UID, folder, group, paused status) but NOT runtime state. To investigate alert noise, the investigator needs:
- **Runtime state**: Which alerts are currently firing, pending, or inactive (from the Prometheus-compatible rules API)
- **Firing instances**: Individual alert instances with their label sets (an alert rule can produce multiple firing instances)
- **State history**: How frequently alerts transition between states (firing/resolved) over time, which is the key signal for identifying noisy alerts
- **Annotations**: Alert state change events tracked as annotations in Grafana

None of these capabilities exist in `grafanactl` today.

## Solution Approach
Build three new `grafanactl alerts` subcommands that call Grafana REST APIs directly (following the pattern established by the existing `export` command which uses `internal/httputils`), plus enhance the existing `list` command:

1. **Enhance `alerts list`** - Add `--state` flag to show runtime state (firing/pending/inactive) by merging data from both the Provisioning API and the Prometheus-compatible rules API (`/api/prometheus/grafana/api/v1/rules`)
2. **New `alerts instances`** - List currently firing alert instances from `/api/prometheus/grafana/api/v1/alerts`
3. **New `alerts history`** - Query alert state change annotations from `/api/annotations?type=alert` with time range filtering
4. **New `alerts noise-report`** - Analyze alert history to compute firing frequency, mean time between fires, and classify alerts as noisy vs meaningful

The investigator agent will then use these tools to run the actual investigation on the live system.

## Relevant Files
Use these files to complete the task:

- `cmd/grafanactl/alerts/command.go` - Alert command group, needs new subcommands registered
- `cmd/grafanactl/alerts/list.go` - Existing list command, needs runtime state enhancement
- `cmd/grafanactl/alerts/list_test.go` - Existing list tests, needs updates for new state column
- `cmd/grafanactl/alerts/export.go` - Reference pattern for direct HTTP API calls using `internal/httputils`
- `cmd/grafanactl/alerts/get.go` - Reference pattern for single-item retrieval
- `cmd/grafanactl/alerts/search.go` - Reference pattern for client-side filtering
- `cmd/grafanactl/io/format.go` - IO options and codec registration pattern
- `internal/httputils/client.go` - HTTP client construction with TLS support
- `internal/grafana/client.go` - Grafana openapi client construction from config context
- `internal/config/` - Configuration and context loading
- `cmd/grafanactl/root/command.go` - Root command registration (alerts already registered)

### Reference Files (do not modify, use for inspiration)
- `../../mcp-grafana/main/tools/alerting.go` - Shows how to merge provisioning + runtime rule data, alert rule summaries with state
- `../../mcp-grafana/main/tools/alerting_client.go` - Shows API endpoints: `/api/prometheus/grafana/api/v1/rules` for runtime state, alerting client pattern
- `../../mcp-grafana/main/tools/annotations.go` - Shows annotations API patterns for alert history

### New Files
- `cmd/grafanactl/alerts/instances.go` - New command: list firing alert instances
- `cmd/grafanactl/alerts/instances_test.go` - Tests for instances command
- `cmd/grafanactl/alerts/history.go` - New command: query alert state history via annotations
- `cmd/grafanactl/alerts/history_test.go` - Tests for history command
- `cmd/grafanactl/alerts/noise_report.go` - New command: analyze alert noise patterns
- `cmd/grafanactl/alerts/noise_report_test.go` - Tests for noise report command
- `cmd/grafanactl/alerts/alerting_client.go` - Shared HTTP client for Prometheus-compatible Grafana alerting APIs

## Implementation Phases
### Phase 1: Foundation
Build the shared alerting HTTP client (`alerting_client.go`) that calls Grafana's Prometheus-compatible alerting APIs. This follows the same pattern as `export.go` (direct HTTP calls via `internal/httputils`) but encapsulates the JSON response parsing for the Prometheus API format. Reference `../../mcp-grafana/main/tools/alerting_client.go` for the exact API endpoints and response structures.

Key API endpoints:
- `GET /api/prometheus/grafana/api/v1/rules` - Returns all rule groups with runtime state (state, health, lastEvaluation, alerts)
- `GET /api/prometheus/grafana/api/v1/alerts` - Returns currently firing alert instances
- `GET /api/annotations?type=alert&from=X&to=Y` - Returns alert state change annotations

Response structures (from mcp-grafana reference):
```go
type rulesResponse struct {
    Data struct {
        RuleGroups []ruleGroup      `json:"groups"`
        Totals     map[string]int64 `json:"totals,omitempty"`
    } `json:"data"`
}

type ruleGroup struct {
    Name           string         `json:"name"`
    FolderUID      string         `json:"folderUid"`
    Rules          []alertingRule `json:"rules"`
    Interval       float64        `json:"interval"`
    LastEvaluation time.Time      `json:"lastEvaluation"`
}

type alertingRule struct {
    State          string           `json:"state"`
    Name           string           `json:"name"`
    Health         string           `json:"health"`
    LastEvaluation time.Time        `json:"lastEvaluation"`
    Alerts         []alertInstance  `json:"alerts"`
    UID            string           `json:"uid"`
    FolderUID      string           `json:"folderUid"`
    Labels         map[string]string `json:"labels"`
    Totals         map[string]int64 `json:"totals,omitempty"`
}

type alertInstance struct {
    Labels      map[string]string `json:"labels"`
    Annotations map[string]string `json:"annotations"`
    State       string            `json:"state"`
    ActiveAt    *time.Time        `json:"activeAt"`
    Value       string            `json:"value"`
}
```

### Phase 2: Core Implementation
Build four commands in parallel:

**2a. Enhance `alerts list` with state** - Add runtime state column by fetching from both provisioning API and Prometheus rules API, merging by rule title/UID. Add `--state` filter flag (e.g., `--state firing` to show only firing rules).

**2b. `alerts instances`** - List currently firing alert instances from `/api/prometheus/grafana/api/v1/alerts`. Table output shows: RULE, STATE, ACTIVE_SINCE, LABELS, VALUE. Supports `--state` filter and `-o json/yaml` output.

**2c. `alerts history`** - Query annotations API with `type=alert` filter. Supports `--from` and `--to` time range flags (default: last 24h). Table output shows: ALERT_NAME, NEW_STATE, PREVIOUS_STATE, TIME. Supports `-o json/yaml`.

**2d. `alerts noise-report`** - Fetches alert history over a configurable period (default 7 days), computes per-alert metrics: fire count, resolve count, mean time between fires, total firing duration. Outputs a table sorted by fire count (noisiest first). Flags: `--period` (default 7d), `--threshold` (fire count above which an alert is "noisy", default 5).

### Phase 3: Integration & Polish
- Register all new commands in `command.go`
- Run `make build` to rebuild the binary
- Investigator uses the new tools against the live system
- Produce the noise analysis report

## Team Orchestration

- You operate as the grafana investigator trying to solve for the alert noise investigation
- You are assigned ONE type of grafana investigation. Focus entirely on completing it.
- Run `bin/grafanactl --help` in order to ensure you have the tools needed to accomplish your task. You may run --help on subcommands if necessary.
- If you do not see the tools required in `grafanactl` in order to accomplish your task, you We'll need to spawn subagents to add new tools to `grafanactl` in order to accomplish your investigation.
- Tell youir sub-agents to Reference @../../mcp-grafana/main and @../../app-env-cli/main For inspiration and examples about how to call various APIs in Grafana related to this task.
- You will create tasks and delegate them to sub-agents (using .claude/agent/teams/builder.md and ./claude/agent/teams/validator.md) in order to implement the tools that you need with the documentation and tests.
- Spawn one builder and one validator per tool that requires creation.
- Subagents will build and validate their work. You will need to run `make build` in order to ensure `grafanactl` is up to date when those tasks complete.
- Use `TaskGet` to read your assigned task details if a task ID is provided.
- Use `TaskCreate`  To create new tasks for your sub-agents in order to implement new features in Grafana CTL that are required for you to accomplish your task.
- When finished, use `TaskUpdate` to mark your task as `completed`.
- If you encounter blockers, update the task with details but do NOT stop - attempt to resolve or work around.
- Spawn other agents to Implement tools that are not available to you within Grafana CTL in order for you to accomplish your work.
- Stay focused on the single task. Do not expand scope.
- `grafanactl` is already configured to access a live system. See `grafanactl resources list` to check that.
- You're responsible for deploying the right team members with the right context to execute the plan.
- IMPORTANT: You NEVER operate directly on the codebase. You use `Task` and `Task*` tools to deploy team members to to the investigation, and perhaps the delegating of building, validating, testing, deploying, and other tasks.
  - This is critical. You're job is to act as a high level director of the team, not a builder.
  - You're role is to validate all work is going well and make sure the team is on track to complete the plan.
  - You'll orchestrate this by using the Task* Tools to manage coordination between the team members.
  - Communication is paramount. You'll use the Task* Tools to communicate with the team members and ensure they're on track to complete the plan.
- Take note of the session id of each team member. This is how you'll reference them.
- However, use the tools that are built for you by your sub-agents in order to validate the task at hand.

### Team Members

- Builder
  - Name: builder-alerting-client
  - Role: Build the shared alerting HTTP client (`alerting_client.go`) that wraps Grafana's Prometheus-compatible alerting API endpoints, including response struct definitions and JSON parsing
  - Agent Type: builder
  - Resume: true

- Builder
  - Name: builder-list-enhancement
  - Role: Enhance the existing `alerts list` command to merge runtime state from the Prometheus rules API with provisioning data, adding STATE column and `--state` filter flag
  - Agent Type: builder
  - Resume: true

- Builder
  - Name: builder-instances-cmd
  - Role: Build the new `alerts instances` command that lists currently firing alert instances from `/api/prometheus/grafana/api/v1/alerts` with table and JSON/YAML output
  - Agent Type: builder
  - Resume: true

- Builder
  - Name: builder-history-cmd
  - Role: Build the new `alerts history` command that queries alert state change annotations from the Grafana annotations API with time range filtering
  - Agent Type: builder
  - Resume: true

- Builder
  - Name: builder-noise-report-cmd
  - Role: Build the new `alerts noise-report` command that analyzes alert history to compute firing frequency metrics and classify alerts as noisy vs meaningful
  - Agent Type: builder
  - Resume: true

- Validator
  - Name: validator-commands
  - Role: Validate all new and modified commands compile, pass tests, and work correctly with `bin/grafanactl`
  - Agent Type: validator
  - Resume: false

- Builder
  - Name: investigator-agent
  - Role: The actual Grafana investigator who uses the built tools (`bin/grafanactl alerts ...`) to examine the live system, analyze firing alerts, and produce the noise analysis report
  - Agent Type: general-purpose
  - Resume: true

## Step by Step Tasks

- IMPORTANT: Execute every step in order, top to bottom. Each task maps directly to a `TaskCreate` call.
- Before you start, run `TaskCreate` to create the initial task list that all team members can see and execute.

### 1. Build Shared Alerting Client
- **Task ID**: build-alerting-client
- **Depends On**: none
- **Assigned To**: builder-alerting-client
- **Agent Type**: builder
- **Parallel**: true (no dependencies)
- Create `cmd/grafanactl/alerts/alerting_client.go` with:
  - Response struct types: `rulesResponse`, `ruleGroup`, `alertingRule`, `alertInstance` (reference `../../mcp-grafana/main/tools/alerting_client.go` for exact field names and JSON tags)
  - `fetchRulesFromPrometheusAPI(ctx context.Context, gCtx *config.Context) (*rulesResponse, error)` - calls `GET /api/prometheus/grafana/api/v1/rules`
  - `fetchFiringAlerts(ctx context.Context, gCtx *config.Context) ([]alertInstance, error)` - calls `GET /api/prometheus/grafana/api/v1/alerts`
  - `fetchAlertAnnotations(ctx context.Context, gCtx *config.Context, from, to int64) ([]annotation, error)` - calls `GET /api/annotations?type=alert&from=X&to=Y`
  - Follow the HTTP client pattern from `export.go` (uses `internal/httputils.NewHTTPClient`, sets auth headers from context)
  - All functions should accept `*config.Context` and build HTTP requests with proper auth (bearer token or basic auth) and org-id headers
- Create `cmd/grafanactl/alerts/alerting_client_test.go` with unit tests for response struct parsing (use `encoding/json` to decode test JSON fixtures)
- Run `go test ./cmd/grafanactl/alerts/...` to verify

### 2. Enhance Alerts List with Runtime State
- **Task ID**: enhance-list-state
- **Depends On**: build-alerting-client
- **Assigned To**: builder-list-enhancement
- **Agent Type**: builder
- **Parallel**: false (depends on alerting client)
- Modify `cmd/grafanactl/alerts/list.go`:
  - After fetching provisioning rules, also call `fetchRulesFromPrometheusAPI()` to get runtime state
  - Merge provisioning rules with runtime rules by matching on rule title (see `mergeAlertRuleData()` in `../../mcp-grafana/main/tools/alerting.go` for the exact merge logic)
  - Update `alertTableCodec.Encode()` to add STATE column showing firing/pending/inactive/error
  - Add `--state` flag to filter by runtime state (e.g., `--state firing` shows only firing rules)
  - Ensure backward compatibility: if runtime API fails, fall back to showing "unknown" state
- Update `cmd/grafanactl/alerts/list_test.go` with tests for the new state column and filtering
- Run `go test ./cmd/grafanactl/alerts/...` to verify

### 3. Build Alerts Instances Command
- **Task ID**: build-instances-cmd
- **Depends On**: build-alerting-client
- **Assigned To**: builder-instances-cmd
- **Agent Type**: builder
- **Parallel**: true (can run alongside task 2, both depend only on task 1)
- Create `cmd/grafanactl/alerts/instances.go`:
  - New `instances` subcommand that calls `fetchFiringAlerts()` from the alerting client
  - Table output (custom codec) with columns: RULE, STATE, ACTIVE_SINCE, LABELS, VALUE
  - Support `-o json` and `-o yaml` output formats
  - Support `--state` flag to filter instances by state (firing, pending)
  - Follow the same opts pattern as `list.go` (struct with setup/Validate methods)
- Create `cmd/grafanactl/alerts/instances_test.go` with unit tests
- Register command in `command.go`: `cmd.AddCommand(instancesCmd(configOpts))`
- Run `go test ./cmd/grafanactl/alerts/...` to verify

### 4. Build Alerts History Command
- **Task ID**: build-history-cmd
- **Depends On**: build-alerting-client
- **Assigned To**: builder-history-cmd
- **Agent Type**: builder
- **Parallel**: true (can run alongside tasks 2 and 3)
- Create `cmd/grafanactl/alerts/history.go`:
  - New `history` subcommand that calls `fetchAlertAnnotations()` from the alerting client
  - Flags: `--from` (default: 24h ago), `--to` (default: now), `--limit` (default: 1000)
  - From/to accept duration strings like "24h", "7d", "30d" (relative to now) OR epoch milliseconds
  - Table output columns: TIME, ALERT_NAME, PREVIOUS_STATE, NEW_STATE, DURATION
  - Support `-o json` and `-o yaml` output
  - The Grafana annotations API returns `alertName`, `newState`, `prevState`, `time`, `timeEnd` fields for alert-type annotations
- Create `cmd/grafanactl/alerts/history_test.go` with unit tests
- Register command in `command.go`: `cmd.AddCommand(historyCmd(configOpts))`
- Run `go test ./cmd/grafanactl/alerts/...` to verify

### 5. Build Alerts Noise Report Command
- **Task ID**: build-noise-report-cmd
- **Depends On**: build-alerting-client
- **Assigned To**: builder-noise-report-cmd
- **Agent Type**: builder
- **Parallel**: true (can run alongside tasks 2, 3, 4)
- Create `cmd/grafanactl/alerts/noise_report.go`:
  - New `noise-report` subcommand
  - Flags: `--period` (default: "7d"), `--threshold` (fire count threshold for "noisy" classification, default: 5)
  - Fetches alert annotations over the period using `fetchAlertAnnotations()`
  - Groups annotations by alert name, computes per-alert:
    - `fire_count`: number of times it transitioned to "alerting"/"firing" state
    - `resolve_count`: number of times it transitioned to "ok"/"normal" state
    - `avg_firing_duration`: average time spent in firing state
    - `classification`: "noisy" if fire_count > threshold, "meaningful" otherwise
  - Table output sorted by fire_count descending (noisiest first), columns: ALERT_NAME, UID, FIRES, RESOLVES, AVG_DURATION, CLASSIFICATION
  - Support `-o json` and `-o yaml` output for machine consumption
- Create `cmd/grafanactl/alerts/noise_report_test.go` with unit tests for the analysis logic
- Register command in `command.go`: `cmd.AddCommand(noiseReportCmd(configOpts))`
- Run `go test ./cmd/grafanactl/alerts/...` to verify

### 6. Validate All Commands
- **Task ID**: validate-commands
- **Depends On**: build-alerting-client, enhance-list-state, build-instances-cmd, build-history-cmd, build-noise-report-cmd
- **Assigned To**: validator-commands
- **Agent Type**: validator
- **Parallel**: false (must wait for all builders)
- Run `make build` to ensure the binary compiles
- Run `go test ./cmd/grafanactl/alerts/...` to ensure all tests pass
- Run `bin/grafanactl alerts --help` and verify all subcommands appear: list, get, search, export, instances, history, noise-report
- Run `bin/grafanactl alerts list --help` and verify `--state` flag is documented
- Run `bin/grafanactl alerts instances --help` and verify command structure
- Run `bin/grafanactl alerts history --help` and verify `--from`/`--to` flags
- Run `bin/grafanactl alerts noise-report --help` and verify `--period`/`--threshold` flags
- Run `make lint` to check for linting issues
- Report pass/fail status for each check

### 7. Run Alert Noise Investigation
- **Task ID**: run-investigation
- **Depends On**: validate-commands
- **Assigned To**: investigator-agent
- **Agent Type**: general-purpose
- **Parallel**: false (must wait for validated tools)
- Run `bin/grafanactl alerts list --state firing` to see all currently firing alerts
- Run `bin/grafanactl alerts instances` to see all firing alert instances with labels
- Run `bin/grafanactl alerts noise-report --period 7d` to get the noise analysis
- Run `bin/grafanactl alerts history --from 7d` to see recent state transitions
- For alerts classified as "noisy", run `bin/grafanactl alerts get <UID>` to examine rule configuration
- Analyze the data and produce a report categorizing each firing alert as:
  - **Noisy**: High fire/resolve frequency, short firing durations, likely needs tuning or silencing
  - **Meaningful**: Stable firing state, long durations, likely indicates a real issue
  - **Borderline**: Moderate frequency, needs human judgment
- Include recommendations for each noisy alert (adjust threshold, add for duration, mute, or delete)
- Write the investigation report to stdout

### 8. Final Validation
- **Task ID**: validate-all
- **Depends On**: build-alerting-client, enhance-list-state, build-instances-cmd, build-history-cmd, build-noise-report-cmd, validate-commands, run-investigation
- **Assigned To**: validator-commands
- **Agent Type**: validator
- **Parallel**: false
- Run all validation commands
- Verify acceptance criteria met
- Confirm investigation report was produced

## Acceptance Criteria
- `bin/grafanactl alerts list` shows a STATE column with runtime alert state (firing/pending/inactive)
- `bin/grafanactl alerts list --state firing` filters to only firing alerts
- `bin/grafanactl alerts instances` lists currently firing alert instances with labels and values
- `bin/grafanactl alerts history --from 7d` shows alert state transitions over the last 7 days
- `bin/grafanactl alerts noise-report --period 7d` produces a noise analysis with per-alert metrics
- All new commands support `-o json` and `-o yaml` output formats
- All tests pass: `go test ./cmd/grafanactl/alerts/...`
- `make build` succeeds
- `make lint` passes (or only has pre-existing warnings)
- Investigation report is produced identifying noisy vs meaningful alerts

## Validation Commands
Execute these commands to validate the task is complete:

- `make build` - Verify the binary compiles with all new commands
- `go test ./cmd/grafanactl/alerts/...` - Run all alert command tests
- `bin/grafanactl alerts --help` - Verify all subcommands are registered
- `bin/grafanactl alerts list --help` - Verify --state flag exists
- `bin/grafanactl alerts instances --help` - Verify instances command exists
- `bin/grafanactl alerts history --help` - Verify history command exists with --from/--to flags
- `bin/grafanactl alerts noise-report --help` - Verify noise-report command exists with --period/--threshold flags
- `make lint` - Check for linting issues

## Notes
- The existing alerts commands are NEW (untracked in git, `?? cmd/grafanactl/alerts/`). They use the Grafana OpenAPI client for provisioning operations but the vendored dependencies may be incomplete. If builds fail due to missing vendored packages, run `go mod vendor` to update.
- The Prometheus-compatible rules API (`/api/prometheus/grafana/api/v1/rules`) is the key endpoint for runtime state. This is NOT the provisioning API - it's a separate Grafana API that returns Prometheus-format rule data with current state.
- The annotations API (`/api/annotations?type=alert`) returns Grafana-format annotations, not Prometheus-format. The response includes `alertName`, `newState`, `prevState`, `time`, `timeEnd` fields.
- Auth pattern: Follow `export.go` - read API token or basic auth from the config context, set headers manually on HTTP requests.
- The mcp-grafana codebase at `../../mcp-grafana/main/tools/alerting.go` and `../../mcp-grafana/main/tools/alerting_client.go` are the best reference for API endpoints, response structures, and merge logic. Builders MUST read these files.
- For the `app-env-cli` reference at `../../app-env-cli/main`, check for HTTP client patterns and CLI command patterns if needed.
