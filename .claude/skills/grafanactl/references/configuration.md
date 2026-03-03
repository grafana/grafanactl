# Configuration Reference

Complete guide to configuring grafanactl for different environments and use cases.

## Configuration File Location

grafanactl stores configuration in:
```
$XDG_CONFIG_HOME/grafanactl/config.yaml
```

Typically: `~/.config/grafanactl/config.yaml`

## Configuration Structure

```yaml
current-context: mystack
contexts:
  mystack:
    server: https://mystack.grafana.net
    namespace: ""  # Auto-discovered for Grafana Cloud
    auth:
      type: token
      token: <api-token>
    tls:
      insecure: false
    default-prometheus-datasource: <uid>
    default-loki-datasource: <uid>

  local:
    server: http://localhost:3000
    namespace: "1"  # org-id for on-prem
    auth:
      type: basic
      username: admin
      password: admin
```

## Configuration Commands

### Set Values

```bash
# Set context server
grafanactl config set contexts.mystack.server https://mystack.grafana.net

# Set authentication
grafanactl config set contexts.mystack.auth.type token
grafanactl config set contexts.mystack.auth.token <api-token>

# Set namespace (org-id for on-prem)
grafanactl config set contexts.local.namespace 1

# Set default datasources
grafanactl config set contexts.mystack.default-prometheus-datasource <uid>
grafanactl config set contexts.mystack.default-loki-datasource <uid>
```

### Unset Values

```bash
# Remove a configuration value
grafanactl config unset contexts.mystack.default-prometheus-datasource
```

### Switch Contexts

```bash
# Use a different context
grafanactl config use-context local

# List all contexts
grafanactl config list-contexts

# Show current context
grafanactl config current-context
```

### View Configuration

```bash
# View entire configuration
grafanactl config view

# Check configuration for issues
grafanactl config check
```

## Authentication Types

### API Token (Recommended)

Best for automation and Grafana Cloud:

```bash
grafanactl config set contexts.mystack.auth.type token
grafanactl config set contexts.mystack.auth.token glsa_xxx
```

**How to create:**
1. In Grafana UI: Administration > Service Accounts
2. Create new service account
3. Add token with appropriate permissions (Editor or Admin)
4. Copy token and use in configuration

**Permissions required:**
- **Viewer**: Read-only access (pull, list, query)
- **Editor**: Read/write access (push, edit, delete)
- **Admin**: Full access (manage datasources, folders)

### Basic Authentication

For on-premise Grafana with basic auth:

```bash
grafanactl config set contexts.local.auth.type basic
grafanactl config set contexts.local.auth.username admin
grafanactl config set contexts.local.auth.password admin
```

**Security note:** Passwords stored in plaintext in config file. Use restrictive file permissions (0600).

## Namespace Configuration

### Grafana Cloud

For Grafana Cloud, namespace is the **stack ID**:

```bash
# Auto-discovered from server URL
grafanactl config set contexts.mystack.server https://mystack.grafana.net
# namespace automatically set to stack ID
```

To override:
```bash
grafanactl config set contexts.mystack.namespace <stack-id>
```

### On-Premise Grafana

For on-premise, namespace is the **organization ID**:

```bash
# Set org ID
grafanactl config set contexts.local.namespace 1

# Default org is usually 1
# Find org ID in Grafana UI: /org
```

## TLS Configuration

### Skip TLS Verification (Development Only)

**WARNING:** Only use for local development. Never in production.

```bash
grafanactl config set contexts.local.tls.insecure true
```

### Custom CA Certificate

For self-signed certificates:

```bash
# Set CA certificate path
grafanactl config set contexts.mystack.tls.ca-cert /path/to/ca.crt

# Or embed in config (base64-encoded)
grafanactl config set contexts.mystack.tls.ca-data <base64-encoded-cert>
```

### Client Certificate Authentication

For mutual TLS:

```bash
# Set client certificate and key
grafanactl config set contexts.mystack.tls.cert-file /path/to/client.crt
grafanactl config set contexts.mystack.tls.key-file /path/to/client.key

# Or embed in config (base64-encoded)
grafanactl config set contexts.mystack.tls.cert-data <base64-encoded-cert>
grafanactl config set contexts.mystack.tls.key-data <base64-encoded-key>
```

## Environment Variables

Override configuration at runtime:

### Core Settings
- `GRAFANA_SERVER`: Grafana server URL
- `GRAFANA_TOKEN`: API token
- `GRAFANA_USER`: Username for basic auth
- `GRAFANA_PASSWORD`: Password for basic auth
- `GRAFANA_ORG_ID`: Organization ID (on-prem)
- `GRAFANA_STACK_ID`: Stack ID (Grafana Cloud)

### TLS Settings
- `GRAFANA_TLS_INSECURE`: Skip TLS verification (true/false)
- `GRAFANA_TLS_CA_CERT`: Path to CA certificate
- `GRAFANA_TLS_CLIENT_CERT`: Path to client certificate
- `GRAFANA_TLS_CLIENT_KEY`: Path to client key

### Example Usage

```bash
# Override server and token for one command
GRAFANA_SERVER=https://dev.grafana.net \
GRAFANA_TOKEN=glsa_dev_xxx \
grafanactl resources list

# Use in CI/CD without config file
export GRAFANA_SERVER=https://prod.grafana.net
export GRAFANA_TOKEN=${GRAFANA_API_TOKEN}  # From CI secrets
grafanactl resources push -p ./dashboards
```

## Multi-Environment Setup

### Development, Staging, Production

```bash
# Development
grafanactl config set contexts.dev.server https://dev.grafana.net
grafanactl config set contexts.dev.auth.token <dev-token>

# Staging
grafanactl config set contexts.staging.server https://staging.grafana.net
grafanactl config set contexts.staging.auth.token <staging-token>

# Production
grafanactl config set contexts.prod.server https://prod.grafana.net
grafanactl config set contexts.prod.auth.token <prod-token>

# Switch between them
grafanactl config use-context dev
grafanactl resources pull

grafanactl config use-context staging
grafanactl resources push --dry-run
```

### Per-Project Contexts

```bash
# Project A contexts
grafanactl config set contexts.projecta-dev.server https://projecta-dev.grafana.net
grafanactl config set contexts.projecta-dev.auth.token <token>

grafanactl config set contexts.projecta-prod.server https://projecta-prod.grafana.net
grafanactl config set contexts.projecta-prod.auth.token <token>

# Project B contexts
grafanactl config set contexts.projectb-dev.server https://projectb-dev.grafana.net
grafanactl config set contexts.projectb-dev.auth.token <token>

grafanactl config set contexts.projectb-prod.server https://projectb-prod.grafana.net
grafanactl config set contexts.projectb-prod.auth.token <token>
```

## Default Datasource Configuration

Set default datasources to avoid repeating `-d` flag:

```bash
# Find datasource UIDs
grafanactl datasources list

# Set defaults
grafanactl config set contexts.mystack.default-prometheus-datasource abc123def
grafanactl config set contexts.mystack.default-loki-datasource xyz789ghi

# Now queries work without -d flag
grafanactl query -e 'up'
grafanactl query -t loki -e '{job="varlogs"}'
```

## Configuration Best Practices

### Security

1. **Protect config file**: Set restrictive permissions
```bash
chmod 600 ~/.config/grafanactl/config.yaml
```

2. **Use API tokens, not passwords**: Tokens are scoped and revocable

3. **Use environment variables in CI/CD**: Don't commit tokens to git
```bash
# In CI/CD
export GRAFANA_SERVER=${PROD_GRAFANA_SERVER}
export GRAFANA_TOKEN=${PROD_GRAFANA_TOKEN}
grafanactl resources push
```

4. **Rotate tokens regularly**: Create new tokens, update config, revoke old ones

5. **Use least privilege**: Viewer role for read-only, Editor for push operations

### Organization

1. **Descriptive context names**: Use `<environment>-<purpose>` pattern
```bash
grafanactl config set contexts.prod-monitoring.server ...
grafanactl config set contexts.dev-testing.server ...
```

2. **Document your contexts**: Keep a README with context purposes

3. **One context per environment**: Don't reuse contexts for multiple purposes

### Workflow

1. **Check before pushing**: Always verify current context
```bash
grafanactl config current-context
grafanactl config check
```

2. **Use dry-run in production**: Preview changes before applying
```bash
grafanactl config use-context prod
grafanactl resources push --dry-run
# Review output
grafanactl resources push
```

3. **Set default datasources**: Improves ergonomics for frequent queries

## Configuration Troubleshooting

### Issue: "Failed to connect to server"

**Check:**
1. Server URL is correct: `grafanactl config view`
2. Network connectivity: `curl -I <server-url>`
3. Firewall/proxy settings

**Solution:**
```bash
# Verify URL
grafanactl config set contexts.mystack.server https://correct-url.grafana.net

# Test connection
grafanactl config check
```

### Issue: "401 Unauthorized"

**Check:**
1. Token is valid and not expired
2. Token has correct permissions
3. Correct authentication type configured

**Solution:**
```bash
# Regenerate token in Grafana UI
# Update configuration
grafanactl config set contexts.mystack.auth.token <new-token>
```

### Issue: "Organization not found"

**Check:**
1. Namespace (org-id) is correct
2. User has access to that organization

**Solution:**
```bash
# For on-prem, set correct org-id
grafanactl config set contexts.local.namespace 1

# For cloud, namespace auto-discovered (leave empty)
grafanactl config unset contexts.mystack.namespace
```

### Issue: "TLS certificate verification failed"

**Check:**
1. Certificate is valid
2. Hostname matches certificate
3. CA certificate trusted

**Solution:**
```bash
# For dev/test only: skip verification
grafanactl config set contexts.local.tls.insecure true

# For production: add CA certificate
grafanactl config set contexts.mystack.tls.ca-cert /path/to/ca.crt
```

## Example Configurations

### Grafana Cloud (Production)

```yaml
current-context: prod-cloud
contexts:
  prod-cloud:
    server: https://myorg.grafana.net
    auth:
      type: token
      token: glsa_xxx_production
    default-prometheus-datasource: cloudprom123
    default-loki-datasource: cloudloki456
```

### On-Premise (Development)

```yaml
current-context: local-dev
contexts:
  local-dev:
    server: http://localhost:3000
    namespace: "1"
    auth:
      type: basic
      username: admin
      password: admin
    tls:
      insecure: true
    default-prometheus-datasource: localprom
```

### Multi-Tenant Setup

```yaml
current-context: tenant-a-prod
contexts:
  tenant-a-prod:
    server: https://tenant-a.grafana.net
    auth:
      type: token
      token: glsa_tenant_a_prod

  tenant-a-dev:
    server: https://tenant-a-dev.grafana.net
    auth:
      type: token
      token: glsa_tenant_a_dev

  tenant-b-prod:
    server: https://tenant-b.grafana.net
    auth:
      type: token
      token: glsa_tenant_b_prod

  tenant-b-dev:
    server: https://tenant-b-dev.grafana.net
    auth:
      type: token
      token: glsa_tenant_b_dev
```

### Corporate with Custom CA

```yaml
current-context: corp-prod
contexts:
  corp-prod:
    server: https://grafana.corp.example.com
    namespace: "1"
    auth:
      type: token
      token: corp_api_token_xxx
    tls:
      ca-cert: /etc/ssl/certs/corp-ca.crt
```

## Migration from Other Tools

### From Terraform

If using Terraform Grafana provider:

```bash
# Extract datasource UIDs from Terraform state
terraform show -json | jq '.values.root_module.resources[] | select(.type=="grafana_data_source") | {name, uid: .values.uid}'

# Set as defaults in grafanactl
grafanactl config set contexts.mystack.default-prometheus-datasource <uid>
```

### From Grizzly

If migrating from Grizzly:

```bash
# Grizzly config typically in grr.yaml
# Extract server and token

# Set up equivalent grafanactl context
grafanactl config set contexts.grizzly-migration.server <server-from-grr>
grafanactl config set contexts.grizzly-migration.auth.token <token-from-grr>
```

## Advanced Configuration

### Custom Rate Limits (Future)

Currently hardcoded (QPS=50, Burst=100). Future versions will support:

```yaml
contexts:
  mystack:
    server: https://mystack.grafana.net
    rate-limit:
      qps: 100
      burst: 200
```

### Custom User Agent (Future)

Future versions will support custom user agent:

```yaml
contexts:
  mystack:
    server: https://mystack.grafana.net
    user-agent: "grafanactl/1.0 (my-automation)"
```
