---
name: grafanactl
description: Get information about and manage Grafana resources, datasources, and queries from the command line. Use this skill when the user wants to interact with grafana instances.
---

# grafanactl

Manage Grafana resources, datasources, and queries from the command line using kubectl-style patterns.

# Instructions

### Step 1: Configure Context

Before interacting with Grafana, set up a context with your instance credentials:

```bash
# For Grafana Cloud (auto-discovers stack ID)
grafanactl config set contexts.mystack.server https://mystack.grafana.net
grafanactl config set contexts.mystack.auth.type token
grafanactl config set contexts.mystack.auth.token <api-token>
grafanactl config use-context mystack

# For on-premise Grafana
grafanactl config set contexts.local.server http://localhost:3000
grafanactl config set contexts.local.auth.type basic
grafanactl config set contexts.local.auth.username admin
grafanactl config set contexts.local.auth.password admin
grafanactl config set contexts.local.namespace 1  # org-id
grafanactl config use-context local
```

**Expected output:** "Switched to context 'mystack'"

Verify your configuration:
```bash
grafanactl config check
```

### Step 2: Discover Available Resources

List what resource types are available in your Grafana instance:

```bash
grafanactl resources list
```

**Expected output:** Table of available resources (dashboards, folders, datasources, alerts, etc.)

### Step 3: Pull Resources

Fetch resources from Grafana to local files:

```bash
# Pull all dashboards and folders
grafanactl resources pull dashboards folders -p ./grafana-resources

# Pull specific resources by UID
grafanactl resources pull dashboards/my-dashboard-uid
```

**Expected output:** Resources saved to `./grafana-resources/` directory organized by kind

### Step 4: Query Datasources

**CRITICAL**: Always use datasource **UID**, not name. Get UIDs first:

```bash
# List datasources to find UIDs
grafanactl datasources list

# Query using UID from the list
grafanactl query -d <datasource-uid> -e 'up{job="grafana"}' --start now-1h --end now -o json | grafanactl graph
```

**Expected output:** ASCII chart showing metric values over time

For more detailed patterns, see `references/query-patterns.md`.

### Step 5: Push or Modify Resources

Make changes and push back to Grafana:

```bash
# Preview changes first
grafanactl resources push --dry-run

# Push to Grafana
grafanactl resources push

# Edit resource interactively
grafanactl resources edit dashboards/my-dashboard-uid
```

**Expected output:** Summary of created/updated resources

# Examples

## Example 1: GitOps Workflow

**User says:** "Pull production dashboards, commit to git, and deploy to staging"

**Actions:**
1. Switch to production context: `grafanactl config use-context production`
2. Pull dashboards: `grafanactl resources pull dashboards -p ./prod-dashboards`
3. Commit to git: `git add prod-dashboards && git commit -m "Snapshot prod dashboards"`
4. Switch to staging: `grafanactl config use-context staging`
5. Deploy: `grafanactl resources push -p ./prod-dashboards --dry-run` then `grafanactl resources push -p ./prod-dashboards`

**Result:** Dashboards from production are now in staging, with full git history

## Example 2: Query and Visualize Metrics

**User says:** "Show me HTTP request rate for the last 6 hours"

**Actions:**
1. Find Prometheus datasource UID: `grafanactl datasources list --type prometheus`
2. Query and graph: `grafanactl query -d <uid> -e 'rate(http_requests_total[5m])' --start now-6h --end now --step 5m -o json | grafanactl graph --title "HTTP Request Rate"`

**Result:** ASCII line chart showing request rate over 6 hours

## Example 3: Dashboard Development with Live Reload

**User says:** "I want to develop dashboards locally with live preview"

**Actions:**
1. Create dashboard generator script (Go, Python, etc.)
2. Start serve command: `grafanactl resources serve --script 'go run dashboard-gen/*.go' --watch ./dashboard-gen --script-format yaml`
3. Open http://localhost:8080 in browser
4. Edit generator code - changes appear immediately

**Result:** Local Grafana instance with live reload, no need to push to remote instance

## Example 4: Multi-Environment Promotion

**User says:** "Promote dashboards from dev to staging to production"

**Actions:**
1. Configure contexts for all environments (dev, staging, prod)
2. Pull from dev: `grafanactl config use-context dev && grafanactl resources pull -p ./dashboards`
3. Push to staging: `grafanactl config use-context staging && grafanactl resources push -p ./dashboards --dry-run && grafanactl resources push -p ./dashboards`
4. After validation, push to prod: `grafanactl config use-context prod && grafanactl resources push -p ./dashboards --dry-run && grafanactl resources push -p ./dashboards`

**Result:** Controlled promotion through environments with dry-run validation

## Example 5: Explore Loki Logs

**User says:** "Show log volume for a specific service"

**Actions:**
1. Find Loki datasource: `grafanactl datasources list --type loki`
2. List available labels: `grafanactl datasources loki labels -d <loki-uid>`
3. Query log rate: `grafanactl query -d <loki-uid> -t loki -e 'sum(rate({job="my-service"}[5m]))' --start now-1h --end now --step 1m -o json | grafanactl graph`

**Result:** ASCII chart showing log volume over time

# Troubleshooting

## Error: "datasource UID is required"

**Cause:** You used datasource name instead of UID, or no default is configured

**Solution:**
1. Get datasource UID: `grafanactl datasources list`
2. Either use `-d <uid>` flag, or set default: `grafanactl config set contexts.mycontext.default-prometheus-datasource <uid>`

## Error: "resource not managed by grafanactl"

**Cause:** Trying to modify a resource created by Grafana UI or another tool

**Solution:**
- To view only: No action needed
- To modify: Add `--include-managed` flag (use with caution - may conflict with other managers)
```bash
grafanactl resources push --include-managed
```

## Error: "failed to parse JSON" when piping to graph

**Cause:** Query output is not in JSON format

**Solution:** Always use `-o json` flag when piping to graph:
```bash
grafanactl query -d <uid> -e 'up' --start now-1h --end now -o json | grafanactl graph
```

## Error: "Connection refused" or "Connection timeout"

**Cause:** Cannot reach Grafana server

**Solution:**
1. Verify server URL: `grafanactl config view`
2. Check network connectivity: `curl -I <grafana-server-url>`
3. Verify authentication: `grafanactl config check`
4. For Grafana Cloud, ensure stack ID is correct (auto-discovered from URL)

## Error: "401 Unauthorized" or "403 Forbidden"

**Cause:** Invalid credentials or insufficient permissions

**Solution:**
1. Verify token is valid: Check in Grafana UI under Service Accounts
2. For basic auth, verify username/password
3. Ensure token has required permissions (Editor or Admin for push operations)
4. Re-configure auth: `grafanactl config set contexts.mycontext.auth.token <new-token>`

## Error: "No data to display" when using graph command

**Cause:** Query returned no results, or wrong time range

**Solution:**
1. Verify query without graph: `grafanactl query -d <uid> -e 'up' --start now-1h --end now -o json`
2. Check if time range contains data
3. For range queries, ensure `--step` is appropriate for the time range
4. Verify metric exists: `grafanactl datasources prometheus metadata -d <uid> --metric <metric-name>`

## Error: "Folder not found" when pushing dashboards

**Cause:** Dashboard references a folder that doesn't exist yet

**Solution:** grafanactl automatically handles folder dependencies by pushing folders first. If this fails:
1. Manually create folder: Push folder resource first
2. Check folder UID in dashboard spec matches existing folder
3. Use `--dry-run` to preview what will be created

## Performance: Slow pulls with many resources

**Cause:** Pulling thousands of resources can be slow

**Solution:**
1. Filter to specific resource types: `grafanactl resources pull dashboards` (not all resources)
2. Use selectors to pull specific resources: `grafanactl resources pull dashboards/uid1,uid2`
3. Adjust concurrency if needed (default is 10 concurrent operations)

## Serve command: "Feature toggle kubernetesDashboards not enabled"

**Cause:** Grafana instance doesn't have required feature flag enabled

**Solution:**
Enable feature flag in Grafana configuration:
```ini
[feature_toggles]
enable = kubernetesDashboards
```
Or use environment variable: `GF_FEATURE_TOGGLES_ENABLE=kubernetesDashboards`

---

**For detailed reference documentation, see:**
- `references/resource-model.md` - Grafana resource relationships and dependencies
- `references/query-patterns.md` - Advanced query patterns and examples
- `references/configuration.md` - Complete configuration options
- `references/selectors.md` - Resource selector syntax guide
