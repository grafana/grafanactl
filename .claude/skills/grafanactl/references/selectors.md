# Resource Selector Syntax

Complete guide to selecting and filtering Grafana resources with grafanactl.

## Selector Overview

grafanactl uses kubectl-style selectors to specify which resources to operate on:

```bash
grafanactl resources pull [selector1] [selector2] ...
grafanactl resources push [selector1] [selector2] ...
grafanactl resources delete selector1 [selector2] ...
```

## Selector Forms

### Short Form - Kind Only

Select all resources of a kind:

```bash
grafanactl resources pull dashboards
grafanactl resources pull folders
grafanactl resources pull dashboards folders
```

**When to use:** Pulling or pushing all resources of specific types

### Short Form - Kind with UIDs

Select specific resources by UID:

```bash
# Single resource
grafanactl resources pull dashboards/my-dashboard-uid

# Multiple resources (comma-separated, no spaces)
grafanactl resources pull dashboards/uid1,uid2,uid3

# Mix kinds and UIDs
grafanactl resources pull dashboards/dash1 folders/folder1
```

**When to use:** Operating on specific known resources

### Long Form - Kind.ResourceType

Fully qualified kind with resource type:

```bash
grafanactl resources pull dashboard.dashboards
grafanactl resources pull folder.folders
```

**When to use:** Disambiguating when multiple resource types share a kind name (rare)

### Fully Qualified Form

Complete GVK (Group, Version, Kind) with optional UIDs:

```bash
# Full GVK
grafanactl resources pull dashboards.v1alpha1.dashboard.grafana.app

# Full GVK with UID
grafanactl resources pull dashboards.v1alpha1.dashboard.grafana.app/my-dashboard-uid

# Multiple UIDs
grafanactl resources pull dashboards.v1alpha1.dashboard.grafana.app/uid1,uid2
```

**When to use:**
- Selecting specific API versions
- Scripting where explicit version is important
- Resolving ambiguous resource names

## Selector Examples

### Pull Operations

```bash
# Pull all dashboards
grafanactl resources pull dashboards

# Pull all dashboards and folders
grafanactl resources pull dashboards folders

# Pull specific dashboard
grafanactl resources pull dashboards/abc123def

# Pull multiple specific dashboards
grafanactl resources pull dashboards/abc123,def456,ghi789

# Pull dashboard and folder
grafanactl resources pull dashboards/mydash folders/myfolder

# Pull all resources (not recommended for large instances)
grafanactl resources pull

# Pull with specific version
grafanactl resources pull dashboards.v1alpha1.dashboard.grafana.app
```

### Push Operations

```bash
# Push all resources from directory
grafanactl resources push

# Push only dashboards
grafanactl resources push dashboards

# Push dashboards and folders
grafanactl resources push dashboards folders

# Push specific dashboard
grafanactl resources push dashboards/abc123

# Push with dry-run
grafanactl resources push dashboards --dry-run
```

### Delete Operations

```bash
# Delete specific dashboard
grafanactl resources delete dashboards/abc123

# Delete multiple dashboards
grafanactl resources delete dashboards/abc123,def456

# Delete dashboard and folder
grafanactl resources delete dashboards/mydash folders/myfolder
```

### Edit Operations

```bash
# Edit specific dashboard (opens in $EDITOR)
grafanactl resources edit dashboards/abc123
```

## UID Resolution

### Where UIDs Come From

UIDs are set in resource metadata:

```yaml
apiVersion: dashboard.grafana.app/v1alpha1
kind: Dashboard
metadata:
  name: my-dashboard
  uid: abc123def  # This is the UID used in selectors
```

### Finding UIDs

```bash
# Pull resources to see their UIDs
grafanactl resources pull dashboards -o yaml

# Or use jq with JSON output
grafanactl resources pull dashboards -o json | jq '.[] | {name: .metadata.name, uid: .metadata.uid}'
```

### UID vs Name

**IMPORTANT:** Selectors use UID, not name!

```bash
# Correct - using UID
grafanactl resources pull dashboards/abc123def

# Wrong - using name (will fail)
grafanactl resources pull dashboards/my-dashboard
```

## Version Selection

### Preferred Version (Default)

By default, grafanactl uses the preferred version (usually latest stable):

```bash
# Uses preferred version
grafanactl resources pull dashboards
```

### All Versions

Select all versions of a resource:

```bash
grafanactl resources pull dashboards --all-versions
```

**Use case:** When you need to see all available versions of a resource type

### Specific Version

Select a specific API version:

```bash
# v1alpha1
grafanactl resources pull dashboards.v1alpha1.dashboard.grafana.app

# v1beta1 (if available)
grafanactl resources pull dashboards.v1beta1.dashboard.grafana.app
```

**Use case:**
- Maintaining backward compatibility
- Testing against specific API versions
- Scripting where version matters

## Selector Patterns

### Pattern: Pull Everything

```bash
# Pull all resources (use with caution on large instances)
grafanactl resources pull -p ./backup
```

### Pattern: Pull Core Resources Only

```bash
# Most common resources
grafanactl resources pull dashboards folders datasources -p ./core-resources
```

### Pattern: Pull by Resource Type

```bash
# Dashboards only
grafanactl resources pull dashboards -p ./dashboards

# Folders only
grafanactl resources pull folders -p ./folders

# Alerting resources
grafanactl resources pull alert-rules contact-points -p ./alerts
```

### Pattern: Selective Pull

```bash
# Specific dashboards from production
grafanactl config use-context prod
grafanactl resources pull dashboards/dashboard-1,dashboard-2,dashboard-3 -p ./prod-critical
```

### Pattern: Incremental Push

```bash
# Push only changed resources by specifying paths
grafanactl resources push -p ./dashboards/modified-dashboard.yaml
```

### Pattern: Safe Production Push

```bash
# Always dry-run first, then push
grafanactl config use-context prod
grafanactl resources push dashboards --dry-run
# Review output
grafanactl resources push dashboards
```

## Filtering During Operations

### Output Path Filtering

Control where resources are saved:

```bash
# Save to custom path
grafanactl resources pull dashboards -p ./custom/path

# Different paths for different resource types
grafanactl resources pull dashboards -p ./dashboards
grafanactl resources pull folders -p ./folders
```

### Format Filtering

Control output format:

```bash
# JSON format
grafanactl resources pull dashboards -o json

# YAML format (default)
grafanactl resources pull dashboards -o yaml
```

### Managed Resources Filtering

Control which resources to include:

```bash
# Only grafanactl-managed resources (default)
grafanactl resources pull dashboards

# Include resources managed by other tools (UI, Terraform, etc.)
grafanactl resources pull dashboards --include-managed
```

## Selector Resolution Process

grafanactl resolves selectors through these steps:

1. **Parse selector**: Extract kind, version, group, UIDs
2. **Discover API**: Query Grafana for available resources
3. **Match descriptor**: Find matching resource type in discovery results
4. **Resolve version**: Use preferred version if not specified
5. **Create filter**: Convert to fully-qualified filter for API calls

Example resolution:

```
Input:     dashboards/abc123
Parse:     kind=dashboards, uid=abc123
Discover:  dashboard.grafana.app/v1alpha1 (Dashboard)
Match:     Found match
Resolve:   Use v1alpha1 (preferred)
Filter:    GET /apis/dashboard.grafana.app/v1alpha1/namespaces/default/dashboards/abc123
```

## Error Messages

### "Resource type not found"

**Cause:** Selector doesn't match any available resource type

**Example:**
```bash
grafanactl resources pull dashboard  # Wrong (singular)
```

**Solution:**
```bash
grafanactl resources pull dashboards  # Correct (plural)
```

### "Ambiguous selector"

**Cause:** Selector matches multiple resource types

**Solution:** Use fully qualified form:
```bash
grafanactl resources pull dashboards.v1alpha1.dashboard.grafana.app
```

### "UID not found"

**Cause:** Specified UID doesn't exist in Grafana

**Solution:** Verify UID exists:
```bash
grafanactl resources pull dashboards -o json | jq '.[] | .metadata.uid'
```

## Advanced Selector Patterns

### Scripting with Selectors

```bash
#!/bin/bash

# Get all dashboard UIDs
DASHBOARD_UIDS=$(grafanactl resources pull dashboards -o json | jq -r '.[] | .metadata.uid' | tr '\n' ',')

# Pull those dashboards
grafanactl resources pull dashboards/$DASHBOARD_UIDS
```

### Conditional Selectors

```bash
#!/bin/bash

if grafanactl resources pull dashboards/critical-dashboard 2>/dev/null; then
  echo "Critical dashboard exists"
else
  echo "Critical dashboard missing!"
  exit 1
fi
```

### Batch Operations

```bash
#!/bin/bash

# List of dashboards to operate on
DASHBOARDS=(
  "dashboard-1"
  "dashboard-2"
  "dashboard-3"
)

# Pull each
for uid in "${DASHBOARDS[@]}"; do
  echo "Pulling dashboards/$uid"
  grafanactl resources pull dashboards/$uid -p ./backups
done
```

### Dynamic Selector Construction

```bash
#!/bin/bash

# Build selector from external source (e.g., inventory file)
UIDS=$(cat dashboard-inventory.txt | tr '\n' ',')
grafanactl resources pull dashboards/$UIDS
```

## Selector Best Practices

1. **Be specific when possible**: `dashboards/uid1,uid2` is faster than pulling all dashboards

2. **Use dry-run for destructive operations**: Always test with `--dry-run` first

3. **Version explicit selectors in scripts**: Use fully qualified form for reproducibility

4. **Verify UIDs before operations**: Check UID exists before delete/edit

5. **Group related operations**: `dashboards folders` instead of separate commands

6. **Use variables for repeated UIDs**: Store commonly used UIDs in shell variables

7. **Document selector patterns**: Keep a README with commonly used selectors

8. **Test selectors in dev first**: Verify selector behavior in non-production context

## Selector Syntax Summary

| Form | Example | Use Case |
|------|---------|----------|
| Kind only | `dashboards` | Pull/push all of a kind |
| Kind with UID | `dashboards/abc123` | Operate on specific resource |
| Multiple UIDs | `dashboards/uid1,uid2` | Operate on multiple resources |
| Multiple kinds | `dashboards folders` | Operate on multiple kinds |
| Long form | `dashboard.dashboards` | Disambiguate resource types |
| Fully qualified | `dashboards.v1alpha1.dashboard.grafana.app` | Explicit version selection |
| With flags | `dashboards --include-managed` | Modify filtering behavior |

## Common Selector Workflows

### Workflow: Backup Specific Dashboards

```bash
# Identify critical dashboards
grafanactl resources pull dashboards -o json | jq '.[] | select(.metadata.labels.critical=="true") | .metadata.uid'

# Backup them
grafanactl resources pull dashboards/uid1,uid2,uid3 -p ./critical-backup
```

### Workflow: Sync Between Environments

```bash
# Pull from prod
grafanactl config use-context prod
grafanactl resources pull dashboards -p ./prod-dashboards

# Push to staging
grafanactl config use-context staging
grafanactl resources push -p ./prod-dashboards --dry-run
grafanactl resources push -p ./prod-dashboards
```

### Workflow: Clean Up Resources

```bash
# List resources to delete
grafanactl resources pull dashboards -o json | jq '.[] | select(.metadata.labels.deprecated=="true") | .metadata.uid'

# Delete them
grafanactl resources delete dashboards/uid1,uid2,uid3
```
