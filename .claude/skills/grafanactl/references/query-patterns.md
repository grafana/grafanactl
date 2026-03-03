# Query Patterns

Advanced patterns for querying Prometheus and Loki datasources with grafanactl.

## Datasource UID Resolution

**CRITICAL**: Always use datasource UID, never the name.

### Finding Datasource UIDs

```bash
# List all datasources
grafanactl datasources list

# Filter by type
grafanactl datasources list --type prometheus
grafanactl datasources list --type loki

# Get JSON for scripting
DS_UID=$(grafanactl datasources list --type prometheus -o json | jq -r '.[0].uid')
```

### Setting Default Datasource

Avoid repeating `-d` flag:

```bash
# Set default Prometheus datasource
grafanactl config set contexts.mystack.default-prometheus-datasource <uid>

# Set default Loki datasource
grafanactl config set contexts.mystack.default-loki-datasource <uid>

# Now queries work without -d flag
grafanactl query -e 'up'
grafanactl query -t loki -e '{job="varlogs"}'
```

## Prometheus Query Patterns

### Instant Queries

Query current values:

```bash
# Current uptime for all targets
grafanactl query -d <uid> -e 'up'

# CPU usage by job
grafanactl query -d <uid> -e 'avg by(job) (rate(cpu_usage_seconds[5m]))'

# Memory usage with threshold
grafanactl query -d <uid> -e 'node_memory_MemAvailable_bytes < 1000000000'
```

### Range Queries

Query over time periods:

```bash
# HTTP request rate over last hour
grafanactl query -d <uid> -e 'rate(http_requests_total[5m])' \
  --start now-1h --end now --step 1m

# CPU usage for specific time period
grafanactl query -d <uid> -e 'avg(cpu_usage)' \
  --start 2026-03-01T00:00:00Z --end 2026-03-01T12:00:00Z --step 5m

# Disk usage over last 24 hours
grafanactl query -d <uid> -e 'disk_used_percent' \
  --start now-24h --end now --step 15m
```

### Time Range Formats

grafanactl supports multiple time formats:

```bash
# Relative time (recommended for most cases)
--start now-1h --end now
--start now-24h --end now-1h
--start now-7d --end now

# RFC3339 timestamps
--start 2026-03-01T00:00:00Z --end 2026-03-01T12:00:00Z

# Unix timestamps
--start 1709280000 --end 1709366400
```

### Step Interval

Choose step based on time range:

```bash
# Short ranges: 1-5 second steps
grafanactl query -d <uid> -e 'rate(requests[1m])' \
  --start now-5m --end now --step 1s

# Medium ranges: 1-5 minute steps
grafanactl query -d <uid> -e 'rate(requests[5m])' \
  --start now-6h --end now --step 1m

# Long ranges: 15-60 minute steps
grafanactl query -d <uid> -e 'rate(requests[1h])' \
  --start now-7d --end now --step 1h
```

**Rule of thumb**: Step should be ~1/100th of total range for smooth charts.

### Aggregation Patterns

```bash
# Sum across all instances
grafanactl query -d <uid> -e 'sum(http_requests_total)'

# Average by label
grafanactl query -d <uid> -e 'avg by(job) (cpu_usage)'

# Top 5 by value
grafanactl query -d <uid> -e 'topk(5, http_requests_total)'

# 95th percentile
grafanactl query -d <uid> -e 'histogram_quantile(0.95, rate(http_request_duration_seconds_bucket[5m]))'
```

### Combining with Graph

```bash
# Line chart (default)
grafanactl query -d <uid> -e 'rate(http_requests_total[5m])' \
  --start now-1h --end now --step 1m -o json | \
  grafanactl graph --title "HTTP Request Rate"

# Bar chart for instant queries
grafanactl query -d <uid> -e 'up' -o json | \
  grafanactl graph --type bar --title "Service Uptime"

# Custom dimensions
grafanactl query -d <uid> -e 'cpu_usage' --start now-6h --end now --step 5m -o json | \
  grafanactl graph --width 120 --height 30 --title "CPU Usage (6h)"
```

## Loki Query Patterns

### Log Stream Selectors

Basic log filtering:

```bash
# All logs from a job
grafanactl query -d <loki-uid> -t loki -e '{job="varlogs"}'

# Multiple labels (AND)
grafanactl query -d <loki-uid> -t loki -e '{job="varlogs",level="error"}'

# Regex matching
grafanactl query -d <loki-uid> -t loki -e '{job=~"mysql.*",level!="debug"}'

# Exclude specific values
grafanactl query -d <loki-uid> -t loki -e '{namespace="production",pod!~"test.*"}'
```

### Log Stream Operators

```bash
# Contains text
grafanactl query -d <loki-uid> -t loki -e '{job="varlogs"} |= "error"'

# Doesn't contain text
grafanactl query -d <loki-uid> -t loki -e '{job="varlogs"} != "debug"'

# Regex match in log line
grafanactl query -d <loki-uid> -t loki -e '{job="varlogs"} |~ "error|exception"'

# JSON parsing
grafanactl query -d <loki-uid> -t loki -e '{job="varlogs"} | json | level="error"'
```

### Log Range Queries

Query logs over time:

```bash
# Last hour of logs
grafanactl query -d <loki-uid> -t loki -e '{job="varlogs"}' \
  --start now-1h --end now

# Specific time range
grafanactl query -d <loki-uid> -t loki -e '{namespace="prod"}' \
  --start 2026-03-01T00:00:00Z --end 2026-03-01T12:00:00Z
```

### Log Metrics (Rate Queries)

Calculate metrics from logs:

```bash
# Log rate per second
grafanactl query -d <loki-uid> -t loki \
  -e 'rate({job="varlogs"}[5m])' \
  --start now-1h --end now --step 1m

# Sum of log rates
grafanactl query -d <loki-uid> -t loki \
  -e 'sum(rate({namespace="production"}[5m]))' \
  --start now-6h --end now --step 5m

# Count by level
grafanactl query -d <loki-uid> -t loki \
  -e 'sum by(level) (rate({job="varlogs"} | json [5m]))' \
  --start now-1h --end now --step 1m
```

### Combining Loki with Graph

```bash
# Visualize log volume
grafanactl query -d <loki-uid> -t loki \
  -e 'sum(rate({job="varlogs"}[5m]))' \
  --start now-6h --end now --step 5m -o json | \
  grafanactl graph --title "Log Volume (logs/sec)"

# Error rate over time
grafanactl query -d <loki-uid> -t loki \
  -e 'sum(rate({job="app"} |= "error" [5m]))' \
  --start now-24h --end now --step 15m -o json | \
  grafanactl graph --title "Error Rate"
```

## Prometheus Datasource Operations

### Exploring Metrics

```bash
# List all available labels
grafanactl datasources prometheus labels -d <uid>

# Get values for specific label
grafanactl datasources prometheus labels -d <uid> --label job
grafanactl datasources prometheus labels -d <uid> --label instance

# Get metric metadata
grafanactl datasources prometheus metadata -d <uid>
grafanactl datasources prometheus metadata -d <uid> --metric http_requests_total

# List scrape targets
grafanactl datasources prometheus targets -d <uid>
```

### Discovery Workflow

1. Find interesting labels:
```bash
grafanactl datasources prometheus labels -d <uid>
```

2. Get values for label:
```bash
grafanactl datasources prometheus labels -d <uid> --label job
```

3. Query specific job:
```bash
grafanactl query -d <uid> -e 'up{job="prometheus"}'
```

4. Explore available metrics for that job:
```bash
grafanactl datasources prometheus metadata -d <uid> | grep -i <keyword>
```

## Loki Datasource Operations

### Exploring Log Streams

```bash
# List all available labels
grafanactl datasources loki labels -d <loki-uid>

# Get values for specific label
grafanactl datasources loki labels -d <loki-uid> --label job
grafanactl datasources loki labels -d <loki-uid> --label namespace

# Find series matching selectors
grafanactl datasources loki series -d <loki-uid> -M '{job="varlogs"}'
grafanactl datasources loki series -d <loki-uid> -M '{namespace="production"}' -M '{level="error"}'
```

### Discovery Workflow

1. Find available labels:
```bash
grafanactl datasources loki labels -d <loki-uid>
```

2. Get values for interesting labels:
```bash
grafanactl datasources loki labels -d <loki-uid> --label job
grafanactl datasources loki labels -d <loki-uid> --label namespace
```

3. Find series combinations:
```bash
grafanactl datasources loki series -d <loki-uid> -M '{job="varlogs"}'
```

4. Query specific stream:
```bash
grafanactl query -d <loki-uid> -t loki -e '{job="varlogs",namespace="prod"}'
```

## Output Formats

### Table Format (Default)

Human-readable table output:

```bash
grafanactl query -d <uid> -e 'up'
# Output:
# METRIC    VALUE  TIMESTAMP
# up{...}   1      2026-03-03T12:00:00Z
```

### JSON Format

Machine-readable for scripting:

```bash
grafanactl query -d <uid> -e 'up' -o json
```

JSON structure:
```json
{
  "status": "success",
  "data": {
    "resultType": "vector",
    "result": [
      {
        "metric": {"__name__": "up", "job": "prometheus"},
        "value": [1709467200, "1"]
      }
    ]
  }
}
```

### YAML Format

```bash
grafanactl query -d <uid> -e 'up' -o yaml
```

### Piping to jq

```bash
# Extract specific fields
grafanactl query -d <uid> -e 'up' -o json | jq '.data.result[].metric.job'

# Filter results
grafanactl query -d <uid> -e 'up' -o json | jq '.data.result[] | select(.value[1] == "1")'

# Count results
grafanactl query -d <uid> -e 'up' -o json | jq '.data.result | length'
```

## Scripting Patterns

### Automated Monitoring

```bash
#!/bin/bash
DS_UID=$(grafanactl datasources list --type prometheus -o json | jq -r '.[0].uid')

# Check if service is up
UP=$(grafanactl query -d $DS_UID -e 'up{job="critical-service"}' -o json | \
     jq -r '.data.result[0].value[1]')

if [ "$UP" != "1" ]; then
  echo "ALERT: critical-service is down!"
  exit 1
fi
```

### Batch Queries

```bash
#!/bin/bash
DS_UID="<your-datasource-uid>"

QUERIES=(
  "up"
  "rate(http_requests_total[5m])"
  "node_memory_MemAvailable_bytes"
)

for query in "${QUERIES[@]}"; do
  echo "Query: $query"
  grafanactl query -d $DS_UID -e "$query" --start now-5m --end now -o json | \
    grafanactl graph --title "$query"
  echo "---"
done
```

### Exporting Data

```bash
# Export query results to file
grafanactl query -d <uid> -e 'cpu_usage' --start now-24h --end now --step 1m -o json > cpu-data.json

# Convert to CSV (using jq)
grafanactl query -d <uid> -e 'up' -o json | \
  jq -r '.data.result[] | [.metric.job, .value[0], .value[1]] | @csv' > results.csv
```

## Performance Tips

### Query Optimization

1. **Use specific label filters**: More specific = faster queries
```bash
# Slow
grafanactl query -d <uid> -e 'http_requests_total'

# Fast
grafanactl query -d <uid> -e 'http_requests_total{job="api",status="200"}'
```

2. **Choose appropriate range selectors**:
```bash
# For rate queries, match range to step
grafanactl query -d <uid> -e 'rate(requests[5m])' --step 5m

# Don't use huge ranges for instant queries
grafanactl query -d <uid> -e 'rate(requests[5m])'  # Good
grafanactl query -d <uid> -e 'rate(requests[1h])'  # Usually unnecessary
```

3. **Limit time ranges**:
```bash
# Query only what you need
grafanactl query -d <uid> -e 'up' --start now-1h --end now  # Good
grafanactl query -d <uid> -e 'up' --start now-30d --end now  # Slow
```

### Loki Performance

1. **Use indexed labels for filtering**:
```bash
# Fast (uses indexed labels)
grafanactl query -d <loki-uid> -t loki -e '{job="varlogs",namespace="prod"}'

# Slow (line filter, not indexed)
grafanactl query -d <loki-uid> -t loki -e '{job="varlogs"} |= "namespace:prod"'
```

2. **Limit log queries**:
```bash
# The default limit is 1000 lines
# For production, consider increasing or narrowing time range
grafanactl query -d <loki-uid> -t loki -e '{job="varlogs"}' --start now-5m --end now
```

## Common Patterns

### Health Check

```bash
# Check if services are up
grafanactl query -d <uid> -e 'up{job="critical-service"}' | grep "1"
```

### Error Rate

```bash
# HTTP error rate
grafanactl query -d <uid> -e 'rate(http_requests_total{status=~"5.."}[5m])' \
  --start now-1h --end now --step 1m -o json | grafanactl graph
```

### Resource Usage

```bash
# Memory usage by pod
grafanactl query -d <uid> -e 'container_memory_usage_bytes{namespace="production"}' | \
  grafanactl graph --type bar
```

### Log Analysis

```bash
# Count errors in last hour
grafanactl query -d <loki-uid> -t loki \
  -e 'count_over_time({job="app"} |= "error" [1h])'
```

### Comparison Queries

```bash
# Compare current vs 24h ago
grafanactl query -d <uid> -e 'rate(requests[5m])' --start now-1h --end now -o json > now.json
grafanactl query -d <uid> -e 'rate(requests[5m])' --start now-25h --end now-24h -o json > yesterday.json
```
