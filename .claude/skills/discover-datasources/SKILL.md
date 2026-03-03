---
name: discover-datasources
description: Discover all the datasources in a Grafana instance, and what information is available in each one. Supports Prometheus (metrics, labels, metadata, targets) and Loki (labels, log streams/series). This is useful when the user needs to understand where to find the data they need. Use when the user asks "where is the data for this dashboard?" or "what datasources are available?" or "what metrics are available?" or "what log streams exist?" or "where can i find metrics/logs for this system?"
allowed-tools: grafanactl
---

# Datasource Discovery

## Instructions

### Step 1: List Available Datasources

Start by identifying all datasources in the Grafana instance.

```bash
# List all datasources
grafanactl datasources list

# Filter by type if you know what you need
grafanactl datasources list --type prometheus
grafanactl datasources list --type loki
```

**Expected output:** Table showing UID, NAME, TYPE, URL, and DEFAULT columns.

**Important:** Always use the UID (not the name) in subsequent commands.

### Step 2: Explore Datasource Contents

Choose the appropriate exploration path based on datasource type.

#### For Prometheus Datasources

```bash
# List all available labels
grafanactl datasources prometheus labels -d <datasource-uid>

# Get values for a specific label to understand what's being monitored
grafanactl datasources prometheus labels -d <datasource-uid> --label job

# List all available metrics with descriptions
grafanactl datasources prometheus metadata -d <datasource-uid>

# Check what systems are being scraped
grafanactl datasources prometheus targets -d <datasource-uid>
```

**Expected output:** Tables showing labels, metrics, or targets depending on command.

#### For Loki Datasources

```bash
# List all available labels
grafanactl datasources loki labels -d <datasource-uid>

# Get values for a specific label
grafanactl datasources loki labels -d <datasource-uid> --label job

# List log streams matching a selector (required)
grafanactl datasources loki series -d <datasource-uid> -M '{job="varlogs"}'
```

**Expected output:** Tables showing labels or log stream series.

**Note:** The `series` command requires at least one `-M` (match) selector using LogQL syntax.

### Step 3: Test Queries (Optional)

Once you've identified available data, verify with a test query.

```bash
# For Prometheus - instant query
grafanactl query -d <datasource-uid> -e 'up'

# For Prometheus - range query
grafanactl query -d <datasource-uid> -e 'rate(http_requests_total[5m])' --start now-1h --end now
```

**Expected output:** Table showing metric values with labels and timestamps.

### Step 4: Set Default Datasource (Optional)

To avoid passing `-d <uid>` repeatedly, configure defaults:

```bash
# Set default Prometheus datasource
grafanactl config set contexts.<context-name>.default-prometheus-datasource <uid>

# Set default Loki datasource
grafanactl config set contexts.<context-name>.default-loki-datasource <uid>
```

After setting defaults, you can omit the `-d` flag in datasource commands.

## Examples

### Example 1: Finding HTTP metrics

**User says:** "What HTTP metrics are available?"

**Actions:**
1. List Prometheus datasources: `grafanactl datasources list --type prometheus`
2. Get datasource UID from output
3. Search for HTTP metrics: `grafanactl datasources prometheus metadata -d <uid> -o json | jq '.data | to_entries[] | select(.key | contains("http"))'`
4. Get details on specific metric: `grafanactl datasources prometheus metadata -d <uid> --metric http_requests_total`

**Result:** Metric name, type (counter/gauge), and help text showing what the metric measures.

### Example 2: Discovering which services are logging to Loki

**User says:** "What applications are sending logs to Loki?"

**Actions:**
1. List Loki datasources: `grafanactl datasources list --type loki`
2. Get datasource UID from output
3. List label names: `grafanactl datasources loki labels -d <uid>`
4. Get job values: `grafanactl datasources loki labels -d <uid> --label job`
5. List streams for a specific job: `grafanactl datasources loki series -d <uid> -M '{job="varlogs"}'`

**Result:** List of job names and their associated log streams.

### Example 3: Troubleshooting missing dashboard data

**User says:** "My dashboard shows no data for service X"

**Actions:**
1. Verify datasource exists: `grafanactl datasources get <uid>`
2. Check if service is being monitored:
   - Prometheus: `grafanactl datasources prometheus targets -d <uid>`
   - Look for service in scrape targets
3. Verify labels exist: `grafanactl datasources prometheus labels -d <uid> --label job`
4. Test simple query: `grafanactl query -d <uid> -e 'up{job="service-x"}'`

**Result:** Identifies whether datasource is misconfigured, service isn't being scraped, or label selectors are wrong.

### Example 4: Finding logs for a specific namespace

**User says:** "Show me all log streams from the production namespace"

**Actions:**
1. Get Loki datasource UID: `grafanactl datasources list --type loki`
2. Verify namespace label exists: `grafanactl datasources loki labels -d <uid> --label namespace`
3. List all streams in namespace: `grafanactl datasources loki series -d <uid> -M '{namespace="production"}'`

**Result:** Table showing all label combinations for log streams in the production namespace.

## Troubleshooting

### Error: "datasource UID is required"

**Cause:** The `-d` flag was omitted and no default datasource is configured.

**Solution:**
```bash
# Option 1: Pass the UID explicitly
grafanactl datasources prometheus labels -d <datasource-uid>

# Option 2: Set a default datasource
grafanactl config set contexts.<context-name>.default-prometheus-datasource <uid>
```

### Error: "at least one --match selector is required"

**Cause:** The `loki series` command was called without a `-M` flag.

**Solution:** Loki series requires at least one LogQL selector:
```bash
# Correct
grafanactl datasources loki series -d <uid> -M '{job="varlogs"}'

# Wrong - will fail
grafanactl datasources loki series -d <uid>
```

### Error: "parse error on line 1, column X: bare \" in non-quoted-field"

**Cause:** Shell is interpreting quotes in the LogQL selector incorrectly.

**Solution:** Use single quotes around the entire selector:
```bash
# Correct - single quotes outside
grafanactl datasources loki series -d <uid> -M '{name="value", cluster="prod"}'

# Wrong - shell interprets quotes incorrectly
grafanactl datasources loki series -d <uid> -M {name="value"}
```

### Error: "datasource.prometheus.datasource.grafana.app \"<uid>\" not found"

**Cause:** The datasource UID doesn't exist or you don't have access to it.

**Solution:**
1. List datasources to verify UID: `grafanactl datasources list`
2. Check you're using the correct context: `grafanactl config current-context`
3. Verify datasource exists: `grafanactl datasources get <uid>`

### No output from labels/series commands

**Cause:** Datasource has no data or hasn't scraped/ingested anything yet.

**Solution:**
1. For Prometheus: Check targets are active: `grafanactl datasources prometheus targets -d <uid>`
2. For Loki: Verify labels exist: `grafanactl datasources loki labels -d <uid>`
3. Check datasource URL is reachable: `grafanactl datasources get <uid>`

## Advanced Usage

For detailed patterns, LogQL syntax guide, and advanced discovery workflows, see:
- [`references/discovery-patterns.md`](references/discovery-patterns.md) - Common discovery patterns and workflows
- [`references/logql-syntax.md`](references/logql-syntax.md) - LogQL selector syntax guide

## Output Formats

All commands support `-o json` or `-o yaml` for programmatic use:

```bash
# Get JSON output for piping to jq
grafanactl datasources prometheus labels -d <uid> -o json

# Example: Count total metrics
grafanactl datasources prometheus metadata -d <uid> -o json | jq '.data | length'
```

Default output is `table` format for human readability.
