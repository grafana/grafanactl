# Plan: Migrate noise-report and history commands from annotations to state history API

## Task Description
The `alerts history` and `alerts noise-report` commands currently use the annotations API (`/api/annotations?type=alert`) to query alert state changes. On Grafana 12+ with unified alerting, state history is stored in a Loki backend (or database), not annotations. The annotations endpoint returns empty results on these instances, making both commands non-functional. This plan migrates them to use the dedicated state history API endpoint (`GET /api/v1/rules/history`) which connects to whatever backend is configured (Loki, annotations, or both).

## Objective
When this plan is complete:
1. `alerts history` queries the state history API and displays state transitions correctly on Grafana 12+ instances
2. `alerts noise-report` computes noise analysis from state history data instead of annotations
3. All existing tests are updated and new tests cover the state history response parsing
4. CLI documentation reflects the new data source

## Problem Statement
The annotations API (`/api/annotations?type=alert`) was the legacy mechanism for alert state history. Grafana has moved to a dedicated state history backend (Loki-based or database-based) that records all state transitions as structured log entries. The current commands return empty results on Grafana 12+ instances because:
- State history is written to Loki, not the annotations table
- The annotations endpoint returns nothing when the backend is Loki-only
- Even when annotations are written alongside Loki, the data is limited (5000 entry cap, no label filtering)

The state history API (`GET /api/v1/rules/history`) was introduced in Grafana 9.4 (PR #62166) and connects to all configured backends.

## Solution Approach
Replace `fetchAlertAnnotations` with a new `fetchStateHistory` function that calls `GET /api/v1/rules/history`, parses the Grafana data frame response, and extracts structured state transition entries. Both `history.go` and `noise_report.go` will consume the new data type instead of `alertAnnotation`.

### State History API Details

**Endpoint**: `GET /api/v1/rules/history`

**Query Parameters**:
- `from` (int64): Unix timestamp in seconds — start of time range
- `to` (int64): Unix timestamp in seconds — end of time range
- `ruleUID` (string, optional): Filter by specific rule UID
- `labels` (map, optional): Label equality filters (limited support)

**Response**: Grafana data frame JSON with three fields:
- `time` ([]time.Time): Timestamp of each state transition
- `line` ([]string): JSON-serialized `LokiEntry` objects
- `labels` ([]string): JSON stream labels

**LokiEntry structure** (parsed from `line` field):
```json
{
  "schemaVersion": 1,
  "previous": "Normal",
  "current": "Alerting",
  "error": "",
  "values": {"A": 42.5},
  "condition": "A",
  "dashboardUID": "abc123",
  "panelID": 1,
  "fingerprint": "abc123def456",
  "ruleTitle": "High CPU Usage",
  "ruleID": 123,
  "ruleUID": "rule-uid-1",
  "labels": {"alertname": "HighCPU", "instance": "server-1"}
}
```

## Relevant Files
Use these files to complete the task:

- `cmd/grafanactl/alerts/alerting_client.go` — Add `fetchStateHistory` function, `stateHistoryFrame`/`stateHistoryEntry` types. Remove `fetchAlertAnnotations` and `alertAnnotation` type.
- `cmd/grafanactl/alerts/alerting_client_test.go` — Replace annotation parsing tests with state history frame parsing tests.
- `cmd/grafanactl/alerts/history.go` — Replace `fetchAlertAnnotations` call with `fetchStateHistory`. Update `annotationsToHistoryEntries` to convert from `stateHistoryEntry` instead.
- `cmd/grafanactl/alerts/history_test.go` — Update tests for new data type.
- `cmd/grafanactl/alerts/noise_report.go` — Replace `fetchAlertAnnotations` call with `fetchStateHistory`. Update `analyzeNoise` to accept `[]stateHistoryEntry`.
- `cmd/grafanactl/alerts/noise_report_test.go` — Update tests for new data type.
- `cmd/grafanactl/alerts/command.go` — No changes needed (subcommand registration unchanged).
- `cmd/grafanactl/alerts/list.go` — No changes needed (uses rules API, not annotations).

### Reference Files (do not modify, use for inspiration)
- `../../mcp-grafana/main/tools/alerting_client.go` — Shows HTTP client pattern for alerting APIs
- The Grafana source PR #62166 — Shows the API handler, query model, and response format

### New Files
None — all changes are modifications to existing files.

## Implementation Phases

### Phase 1: Foundation — State History Client
Add the state history API types and fetch function to `alerting_client.go`:
- Define `stateHistoryFrame` struct matching the Grafana data frame JSON response
- Define `stateHistoryEntry` struct matching the LokiEntry JSON within each `line` field
- Implement `fetchStateHistory(ctx, gCtx, from, to int64) ([]stateHistoryEntry, error)` that:
  - Calls `GET /api/v1/rules/history?from={from}&to={to}` (note: `from`/`to` are Unix seconds, not milliseconds)
  - Parses the data frame response
  - Extracts and deserializes each `line` JSON into `stateHistoryEntry`
  - Returns the flat list of entries
- Remove `alertAnnotation` struct and `fetchAlertAnnotations` function

### Phase 2: Core Implementation — Migrate Commands
Update `history.go`:
- Replace `fetchAlertAnnotations` with `fetchStateHistory`
- Convert `from`/`to` from milliseconds to seconds (the state history API uses Unix seconds)
- Replace `annotationsToHistoryEntries` with a function that converts `[]stateHistoryEntry` to `[]historyEntry`
- Map fields: `entry.current` → `NewState`, `entry.previous` → `PrevState`, `entry.ruleTitle` → `AlertName`, timestamp from the frame's `time` field
- Update command Long description to reference "state history" instead of "annotations"

Update `noise_report.go`:
- Replace `fetchAlertAnnotations` with `fetchStateHistory`
- Convert `from`/`to` from milliseconds to seconds
- Update `analyzeNoise` to accept `[]stateHistoryEntry` instead of `[]alertAnnotation`
- Map: group by `ruleTitle`, count transitions where `current` is "Alerting"/"Firing" as fires, `current` is "Normal"/"OK" as resolves
- Use the `ruleUID` field from entries for the UID column (instead of `dashboardUID`)

### Phase 3: Integration & Polish
Update all tests:
- `alerting_client_test.go`: Replace annotation JSON parsing tests with data frame response parsing tests. Include realistic sample data frame JSON.
- `history_test.go`: Update `annotationsToHistoryEntries` tests to use new conversion function and `stateHistoryEntry` input
- `noise_report_test.go`: Update `analyzeNoise` tests to use `stateHistoryEntry` input instead of `alertAnnotation`

Run `make build && make tests && make lint` to verify.

## Team Orchestration

- You operate as the Grafana investigator trying to solve for migrating the alert noise investigation tools from annotations to the state history API.
- You are assigned ONE type of Grafana investigation. Focus entirely on completing it.
- Run `bin/grafanactl --help` in order to ensure you have the tools needed to accomplish your task. You may run --help on subcommands if necessary.
- If you do not see the tools required in `grafanactl` in order to accomplish your task, you will need to spawn subagents to add new tools to `grafanactl` in order to accomplish your investigation.
- Tell your sub-agents to reference @../../mcp-grafana/main and @../../app-env-cli/main for inspiration and examples about how to call various APIs in Grafana related to this task.
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
  - Name: builder-state-history-client
  - Role: Implement `fetchStateHistory`, new types, and remove annotation-based code in `alerting_client.go` and `alerting_client_test.go`
  - Agent Type: builder
  - Resume: true

- Builder
  - Name: builder-history-cmd
  - Role: Migrate `history.go` and `history_test.go` from annotations to state history entries
  - Agent Type: builder
  - Resume: true

- Builder
  - Name: builder-noise-report-cmd
  - Role: Migrate `noise_report.go` and `noise_report_test.go` from annotations to state history entries
  - Agent Type: builder
  - Resume: true

- Validator
  - Name: validator-all
  - Role: Run `make build`, `make tests`, `make lint` and verify all commands work correctly against the live system
  - Agent Type: validator
  - Resume: false

## Step by Step Tasks

### 1. Implement state history client types and fetch function
- **Task ID**: state-history-client
- **Depends On**: none
- **Assigned To**: builder-state-history-client
- **Agent Type**: builder
- **Parallel**: false (foundation — other tasks depend on this)
- Add `stateHistoryFrame` struct to `alerting_client.go` representing the Grafana data frame JSON:
  ```go
  type stateHistoryFrame struct {
      Schema struct {
          Fields []struct {
              Name string `json:"name"`
              Type string `json:"type"`
          } `json:"fields"`
      } `json:"schema"`
      Data struct {
          Values []json.RawMessage `json:"values"`
      } `json:"data"`
  }
  ```
- Add `stateHistoryEntry` struct representing the parsed LokiEntry from each `line` value:
  ```go
  type stateHistoryEntry struct {
      SchemaVersion int               `json:"schemaVersion"`
      Previous      string            `json:"previous"`
      Current       string            `json:"current"`
      Error         string            `json:"error,omitempty"`
      Values        map[string]any    `json:"values,omitempty"`
      Condition     string            `json:"condition,omitempty"`
      DashboardUID  string            `json:"dashboardUID,omitempty"`
      PanelID       int64             `json:"panelID,omitempty"`
      Fingerprint   string            `json:"fingerprint,omitempty"`
      RuleTitle     string            `json:"ruleTitle"`
      RuleID        int64             `json:"ruleID,omitempty"`
      RuleUID       string            `json:"ruleUID"`
      Labels        map[string]string `json:"labels,omitempty"`
      Timestamp     time.Time         // populated from the frame's time field, not from JSON
  }
  ```
- Implement `fetchStateHistory(ctx context.Context, gCtx *config.Context, from, to int64) ([]stateHistoryEntry, error)`:
  - Build URL: `{server}/api/v1/rules/history?from={from}&to={to}` (from/to are Unix seconds)
  - Use `makeAlertingRequest` pattern (60s timeout, auth headers)
  - Parse the response as a data frame (the response may be a single frame object or wrapped — probe the live endpoint to confirm)
  - Find the `time` values array (index 0), `line` values array (index 1), `labels` values array (index 2) by matching field names from the schema
  - For each entry: parse the timestamp, JSON-decode the `line` string into `stateHistoryEntry`, set `Timestamp` from the time field
  - Return the complete list sorted by timestamp descending
- Remove `alertAnnotation` struct and `fetchAlertAnnotations` function (they are no longer used)
- Update `alerting_client_test.go`:
  - Remove annotation response parsing tests
  - Add tests for `stateHistoryFrame` parsing with realistic sample JSON
  - Test edge cases: empty frame, frame with no entries, malformed line JSON

### 2. Migrate history command to state history API
- **Task ID**: migrate-history-cmd
- **Depends On**: state-history-client
- **Assigned To**: builder-history-cmd
- **Agent Type**: builder
- **Parallel**: true (can run in parallel with task 3 after task 1 completes)
- In `history.go`:
  - Replace `fetchAlertAnnotations(cmd.Context(), currentCtx, from, to)` with `fetchStateHistory(cmd.Context(), currentCtx, from/1000, to/1000)` (convert ms to seconds)
  - Rename `annotationsToHistoryEntries` to `stateHistoryToHistoryEntries` and change signature to accept `[]stateHistoryEntry`
  - Map fields: `entry.RuleTitle` → `AlertName`, `entry.Previous` → `PrevState`, `entry.Current` → `NewState`, `entry.Timestamp` → `Time`
  - For duration: if the entry doesn't have TimeEnd, omit duration (state history entries represent point-in-time transitions, not ranges)
  - Update command `Long` description: change "annotations" to "state history"
- In `history_test.go`:
  - Update `TestAnnotationsToHistoryEntries` (or rename it) to use `[]stateHistoryEntry` input
  - Update test fixtures to use state history entry fields

### 3. Migrate noise-report command to state history API
- **Task ID**: migrate-noise-report
- **Depends On**: state-history-client
- **Assigned To**: builder-noise-report-cmd
- **Agent Type**: builder
- **Parallel**: true (can run in parallel with task 2 after task 1 completes)
- In `noise_report.go`:
  - Replace `fetchAlertAnnotations(cmd.Context(), currentCtx, from, to)` with `fetchStateHistory(cmd.Context(), currentCtx, from/1000, to/1000)` (convert ms to seconds)
  - Update `analyzeNoise` signature: `func analyzeNoise(entries []stateHistoryEntry, threshold int) []NoiseEntry`
  - Map fields: group by `entry.RuleTitle` (was `ann.AlertName`), use `entry.RuleUID` for UID (was `ann.DashboardUID`)
  - Count fires: `entry.Current` is "Alerting" or "Firing"
  - Count resolves: `entry.Current` is "Normal" or "OK"
  - Duration: state history entries are point-in-time, so average duration calculation needs adjustment — compute duration between consecutive "Alerting" → "Normal" transitions for the same rule, or omit if not feasible
- In `noise_report_test.go`:
  - Update all test fixtures from `alertAnnotation` to `stateHistoryEntry`
  - Update field references throughout

### 4. Validate all changes
- **Task ID**: validate-all
- **Depends On**: state-history-client, migrate-history-cmd, migrate-noise-report
- **Assigned To**: validator-all
- **Agent Type**: validator
- **Parallel**: false
- Run `make build` — must succeed
- Run `make tests` — all tests must pass
- Run `make lint` — no new lint errors
- Verify `bin/grafanactl alerts history --help` shows updated description
- Verify `bin/grafanactl alerts noise-report --help` shows updated description
- Run `bin/grafanactl alerts history --from 7d` on the live system — must return state transition data (not empty)
- Run `bin/grafanactl alerts noise-report --period 7d` on the live system — must produce noise analysis

## Acceptance Criteria
- `fetchAlertAnnotations` and `alertAnnotation` are removed from `alerting_client.go`
- `fetchStateHistory` calls `GET /api/v1/rules/history` and correctly parses the data frame response
- `alerts history` displays state transitions from the state history backend
- `alerts noise-report` computes noise analysis from state history data
- All existing tests pass (updated for new types)
- New tests cover data frame parsing, including edge cases
- `make build`, `make tests`, and `make lint` all pass
- Commands produce non-empty output when run against the live Grafana 12+ instance

## Validation Commands
Execute these commands to validate the task is complete:

- `make build` — Binary compiles successfully
- `make tests` — All unit tests pass with race detection
- `make lint` — No lint errors
- `bin/grafanactl alerts history --from 24h` — Returns state history entries (not empty)
- `bin/grafanactl alerts noise-report --period 7d` — Returns noise analysis (not empty)
- `bin/grafanactl alerts history --help` — Description references state history, not annotations
- `bin/grafanactl alerts noise-report --help` — No reference to annotations

## Notes
- The state history API uses Unix **seconds** for `from`/`to`, while the annotations API used **milliseconds**. The `parseTimeArg` function in `history.go` returns milliseconds — the callers must convert.
- The data frame response format may vary slightly between Grafana versions. The implementation should handle both the case where `data.values` contains arrays directly and the case where the response wraps the frame in an additional object.
- If the Grafana instance doesn't have the state history API enabled (very old versions), the endpoint will return 404. The implementation should detect this and return a clear error message suggesting the user upgrade Grafana or enable the `alertingCentralAlertHistory` feature toggle.
- State history entries are point-in-time transitions (previous → current), not ranges with a TimeEnd. The noise report's average duration calculation must be adapted: pair consecutive "Alerting"→"Normal" transitions per rule to estimate firing duration, or simply omit duration if pairing is unreliable.
- The `ruleUID` field is available on every state history entry, which is more reliable than the `dashboardUID` from annotations. Use `ruleUID` in the noise report UID column.
