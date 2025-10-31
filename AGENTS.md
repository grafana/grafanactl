# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Overview

**grafanactl** is a command-line tool for managing Grafana resources through the REST API. It supports Grafana 12 and above, enabling users to authenticate, manage multiple environments, and perform administrative tasks from the terminal. The tool is particularly useful for dashboards-as-code workflows and CI/CD automation.

## Development Environment

This project uses **devbox** for consistent development environments:

```bash
# Enter devbox shell (includes all required tools)
devbox shell

# Run one-off commands within devbox
devbox run go version

# Add packages
devbox add go@1.24
```

## Common Commands

### Build and Test

```bash
# Build the binary to bin/grafanactl
make build

# Run all tests
make tests

# Run tests for a specific package
go test -v ./internal/config

# Install to $GOPATH/bin
make install

# Run linter
make lint

# Run all checks (lint, tests, build, docs)
make all
```

### Documentation

```bash
# Generate and build all documentation
make docs

# Generate CLI reference documentation
make cli-reference

# Generate environment variable reference
make env-var-reference

# Generate configuration file reference
make config-reference

# Serve documentation locally with live reload
make serve-docs

# Check for drift in generated documentation
make reference-drift
```

### Dependency Management

```bash
# Vendor dependencies
make deps

# Clean build artifacts and dependencies
make clean
```

## Architecture

### Command Structure

grafanactl follows the Cobra command pattern with two main command groups:

1. **config**: Manage configuration contexts for connecting to Grafana instances
   - `config set`: Set configuration values
   - `config unset`: Unset configuration values
   - `config use-context`: Switch between configured contexts
   - `config list-contexts`: List all configured contexts
   - `config current-context`: Show the current context
   - `config view`: View the current configuration
   - `config check`: Validate the configuration

2. **resources**: Manipulate Grafana resources (dashboards, folders, etc.)
   - `resources get`: Get resources from Grafana
   - `resources list`: List resources
   - `resources pull`: Pull resources from Grafana to local files
   - `resources push`: Push local resources to Grafana
   - `resources delete`: Delete resources from Grafana
   - `resources edit`: Edit resources interactively
   - `resources validate`: Validate resource manifests
   - `resources serve`: Serve resources locally with live reload

### Core Packages

**cmd/grafanactl/** - CLI command implementations
- `root/`: Root command setup with logging and flags
- `config/`: Configuration management commands
- `resources/`: Resource manipulation commands
- `fail/`: Error handling and detailed error messages
- `io/`: Output formatting and user messages

**internal/config/** - Configuration management
- Context-based configuration (similar to kubectl contexts)
- Support for multiple Grafana instances (contexts)
- Authentication: basic auth, API tokens
- Environment variable overrides (GRAFANA_SERVER, GRAFANA_TOKEN, etc.)
- Automatic Stack ID discovery for Grafana Cloud
- TLS configuration support

**internal/resources/** - Resource abstraction layer
- `Resource`: Wraps Kubernetes-style unstructured objects for Grafana resources
- `Resources`: Collection with filtering, grouping, and concurrent operations
- Uses `k8s.io/apimachinery` for resource representation
- Supports multiple source formats (JSON, YAML)

**internal/resources/local/** - Local file operations
- `reader.go`: Load resources from disk (supports directories and single files)
- `writer.go`: Save resources to disk with proper formatting

**internal/resources/remote/** - Remote Grafana operations
- `puller.go`: Fetch resources from Grafana API
- `pusher.go`: Upload resources to Grafana API (create/update)
- `deleter.go`: Delete resources from Grafana

**internal/resources/dynamic/** - Dynamic Kubernetes client wrapper
- `namespaced_client.go`: Per-namespace resource operations
- `versioned_client.go`: Version-aware resource client
- Wraps k8s.io/client-go dynamic client

**internal/resources/discovery/** - API discovery
- `registry.go`: Discover available resource types from Grafana API
- `registry_index.go`: Index and cache discovered resources

**internal/resources/process/** - Resource processing
- `managerfields.go`: Handle manager metadata (similar to kubectl's server-side apply)
- `serverfields.go`: Process server-managed fields

**internal/server/** - Local development server (for `resources serve`)
- Chi-based HTTP server with reverse proxy to Grafana
- Live reload via WebSocket
- File watching with fsnotify
- Dashboard and folder preview handlers
- Script execution for generated resources

**internal/grafana/** - Grafana API client
- Wraps grafana-openapi-client-go
- Client construction from context configuration

**internal/format/** - Format detection and conversion
- JSON, YAML format support
- Auto-detection from file extensions

**internal/httputils/** - HTTP utilities
- REST client helpers
- Request/response handling

**internal/secrets/** - Secret management
- Secure handling of sensitive configuration data

**internal/logs/** - Structured logging
- slog-based logging with verbosity levels
- Integration with k8s klog

### Key Design Patterns

1. **Context-Based Configuration**: Like kubectl, grafanactl uses named contexts to manage multiple Grafana instances. Each context contains server URL, authentication, and namespace (org-id or stack-id).

2. **Kubernetes-Style Resources**: Resources are represented as unstructured objects following Kubernetes conventions (apiVersion, kind, metadata, spec). This enables compatibility with k8s tooling and patterns.

3. **Resource Manager Metadata**: Tracks which tool manages each resource (grafanactl, UI, etc.) to prevent accidental overwrites. Only resources managed by grafanactl can be modified via grafanactl.

4. **Three-Way Merge**: When pushing resources, grafanactl performs server-side apply semantics similar to kubectl, allowing multiple managers to coexist.

5. **Dashboards as Code**: The `serve` command supports script-based dashboard generation with live preview, enabling code-first workflows with SDKs like grafana-foundation-sdk.

## Configuration

Configuration is stored in `$XDG_CONFIG_HOME/grafanactl/config.yaml` (typically `~/.config/grafanactl/config.yaml`).

### Environment Variables

Environment variables can override configuration values:
- `GRAFANA_SERVER`: Grafana server URL
- `GRAFANA_TOKEN`: API token for authentication
- `GRAFANA_USER`: Username for basic auth
- `GRAFANA_PASSWORD`: Password for basic auth
- `GRAFANA_ORG_ID`: Organization ID (on-prem)
- `GRAFANA_STACK_ID`: Stack ID (Grafana Cloud)

## Testing Patterns

- Unit tests use Go standard testing with testify/assert
- Configuration tests in `internal/config/*_test.go`
- Resource filtering/selector tests in `internal/resources/*_test.go`
- Test data in `testdata/` directories
- Use `make tests` to run all tests with race detection

## Code Generation

- CLI reference docs auto-generated from Cobra commands via `scripts/cmd-reference/`
- Environment variable docs generated via `scripts/env-vars-reference/`
- Config reference docs generated via `scripts/config-reference/`
- All generated docs must be committed (checked by CI via `make reference-drift`)

## Build Process

Build uses Make with devbox. Version information injected at build time:
- `version`: Git tag (exact match) or "SNAPSHOT"
- `commit`: Git commit SHA
- `date`: Build timestamp

Flags set in Makefile via `-ldflags` on `main.version`, `main.commit`, `main.date`.

## Important Notes

- Only supports Grafana 12+
- Project is in "public preview" maturity
- Uses vendored dependencies (committed to repo)
- Documentation built with mkdocs (Python-based)
- Feature toggle `kubernetesDashboards` must be enabled in Grafana for `resources serve`

## Codebase Architecture Insights

### System Overview

**grafanactl** is a ~12,500 line Go codebase that provides a kubectl-like interface for managing Grafana resources. It bridges Grafana's REST API with Kubernetes-style resource management patterns, enabling Infrastructure-as-Code workflows for Grafana dashboards, folders, and other resources.

The tool's architecture is heavily influenced by Kubernetes client patterns, using k8s.io/client-go and k8s.io/apimachinery libraries to provide a consistent, familiar interface for developers already comfortable with kubectl.

### Architectural Layers

The codebase follows a clean layered architecture:

```
CLI Layer (cmd/grafanactl/)
    ↓
Business Logic Layer (internal/resources/, internal/config/)
    ↓
Client Abstraction Layer (internal/resources/dynamic/)
    ↓
Transport Layer (internal/httputils/, internal/grafana/)
    ↓
Grafana REST API
```

### Core Abstractions

#### 1. Resource Abstraction (`internal/resources/resources.go`)

The `Resource` type is the fundamental building block, wrapping `k8s.io/apimachinery/unstructured.Unstructured`:

- **GrafanaMetaAccessor**: Provides typed access to Grafana-specific metadata (manager, source info)
- **Source Tracking**: Each resource tracks its origin (file path, format) for round-tripping
- **Manager Metadata**: Resources carry information about which tool manages them (grafanactl vs UI)
- **Collection Operations**: The `Resources` type provides filtering, grouping, and concurrent operations

#### 2. Discovery System (`internal/resources/discovery/`)

API discovery is critical for dynamic resource type resolution:

- **RegistryIndex**: Caches discovered resource types from Grafana's API server
- **Preferred Versions**: Like Kubernetes, tracks which version is preferred for each resource type
- **Selector Resolution**: Converts partial user input (e.g., "dashboards") to fully-qualified GVK
- **Filtering**: Excludes read-only or internal resource groups (featuretoggle, iam, etc.)

#### 3. Selector → Filter Pipeline

User input flows through a two-stage resolution process:

1. **Selectors** (`internal/resources/selector.go`): Partial specifications from user
   - Supports short forms: "dashboards", "dashboards/foo"
   - Supports long forms: "dashboards.v1alpha1.dashboard.grafana.app/foo"
   - Parsed into `PartialGVK` + resource UIDs

2. **Filters** (`internal/resources/filter.go`): Fully-resolved specifications
   - Contains complete `Descriptor` with GVK and plural/singular forms
   - Three filter types: All (list all), Multiple (get specific), Single (get one)
   - Used by readers/pullers to fetch the right resources

#### 4. Dynamic Client (`internal/resources/dynamic/`)

Wraps k8s.io/client-go's dynamic client for Grafana:

- **Namespaced Operations**: All operations scoped to a namespace (org-id or stack-id)
- **Pagination**: Automatic handling via k8s.io/client-go pager
- **Concurrent Gets**: When fetching multiple named resources, uses errgroup for parallelism
- **Error Translation**: Converts k8s StatusError to friendly error messages

### Execution Flows

#### Push Flow (Local → Grafana)

```
1. Read files from disk (FSReader)
   - Concurrent file reads with errgroup
   - Format detection (JSON/YAML)
   - Decode into unstructured.Unstructured

2. Apply filters (from selectors)
   - Skip resources not matching user criteria
   - Deduplicate by GVK+name

3. Process resources
   - ManagerFieldsAppender: Add grafanactl manager metadata
   - ServerFieldsStripper: Remove read-only fields (for dry-run)

4. Push to Grafana (Pusher)
   - Two-phase: folders first (dependency ordering)
   - Per-resource: Check if exists → Create or Update
   - Concurrent pushes (configurable concurrency)
   - Dry-run support via Kubernetes dry-run semantics
```

#### Pull Flow (Grafana → Local)

```
1. Discover API resources (Registry)
   - Call Grafana's ServerGroupsAndResources endpoint
   - Build index of supported resource types

2. Resolve selectors to filters
   - Convert user input to fully-qualified descriptors
   - Support version wildcards (all versions vs preferred)

3. Fetch from Grafana (Puller)
   - List or Get operations via dynamic client
   - Concurrent fetches with errgroup
   - Process: Strip server fields, check manager

4. Write to disk (FSWriter)
   - Group by kind (one file per kind or per resource)
   - Format as JSON/YAML
   - Create directory structure
```

#### Serve Flow (Local Development Server)

```
1. Start HTTP server (Chi router)
   - Reverse proxy to Grafana
   - Custom handlers for dashboards/folders
   - WebSocket for live reload

2. Watch files for changes (fsnotify)
   - Detect file modifications
   - Reload resources into memory
   - Trigger WebSocket reload event

3. Serve dashboard previews
   - Intercept Grafana dashboard API calls
   - Inject local resource data
   - Mock out unrelated API endpoints

4. Live reload in browser
   - WebSocket connection to browser
   - Automatic page refresh on file change
```

### Key Technology Choices

#### Kubernetes Client Libraries

Using k8s.io/client-go provides:
- Battle-tested dynamic client implementation
- Standard patterns for discovery, pagination, error handling
- Unstructured object representation (map[string]interface{})
- Compatibility with Grafana's Kubernetes-inspired API

Trade-off: Adds dependency weight (~many vendor packages) but saves enormous implementation effort.

#### Cobra for CLI

Standard Go CLI framework providing:
- Subcommand structure
- Flag binding and validation
- Help generation
- Consistent with other Cloud Native tools

#### Context-Based Configuration

Similar to kubectl config, enables:
- Multiple environment management (dev, staging, prod)
- Easy context switching
- Environment variable overrides
- Secure credential storage

#### Concurrent Operations

Heavy use of `golang.org/x/sync/errgroup`:
- File reads/writes parallelized
- API calls (Get, Create, Update) concurrent
- Configurable concurrency limits
- Proper error propagation and cancellation

### Code Organization Patterns

#### 1. Options Pattern

Commands use options structs with setup/validate methods:
```go
type pushOpts struct {
    Paths []string
    MaxConcurrent int
    // ...
}

func (opts *pushOpts) setup(flags *pflag.FlagSet) { ... }
func (opts *pushOpts) Validate() error { ... }
```

#### 2. Processor Pattern

Resource transformations via processor interface:
```go
type Processor interface {
    Process(*Resource) error
}
```

Used for:
- Adding manager fields (`ManagerFieldsAppender`)
- Stripping server fields (`ServerFieldsStripper`)
- Composable pipeline in push/pull operations

#### 3. Registry Pattern

Discovery registry caches API metadata:
- Lazy initialization on first use
- Index by GVK for fast lookups
- Supports partial GVK matching
- Preferred version tracking

#### 4. Source-Tracking Pattern

Resources carry their source information:
- File path where they were read
- Format (JSON/YAML)
- Enables round-tripping preserving format
- Used in error messages for context

### Testing Strategy

- **Unit Tests**: Focus on parsing, filtering, selector logic
- **Table-Driven Tests**: Common in selector/filter tests
- **Mock Filesystems**: Use `testdata/` directories for test fixtures
- **No Integration Tests**: All tests are unit tests (no live Grafana required)

Current test coverage focuses on:
- Configuration loading and validation
- Selector parsing and resolution
- Filter matching logic
- Resource grouping and sorting
- Manager field processing

### Performance Characteristics

**Concurrency Model**:
- Default 10 concurrent operations (configurable)
- errgroup with SetLimit for bounded concurrency
- File I/O parallelized (reading multiple files)
- Network I/O parallelized (multiple API calls)

**Memory Profile**:
- Resources loaded into memory during operations
- No streaming (entire resource collections in RAM)
- Reasonable for typical use cases (hundreds of dashboards)
- Could be problematic for very large deployments (thousands of resources)

**Network Efficiency**:
- Pagination handled automatically by k8s client
- List operations prefer List over Get-per-resource
- Get operations parallelized when fetching multiple named resources

### Extension Points

The codebase is designed for extension:

1. **New Resource Types**: Automatically discovered via Grafana API
2. **New Formats**: Add codec to `internal/format/codec.go`
3. **New Processors**: Implement Processor interface
4. **Custom Handlers**: Add to `internal/server/handlers/`
5. **Authentication Methods**: Extend `internal/config/rest.go`

### Dependencies of Note

**Core Libraries**:
- `k8s.io/client-go` v0.34.1: Dynamic client, discovery
- `k8s.io/apimachinery` v0.34.1: Unstructured, GVK, schemas
- `github.com/spf13/cobra` v1.10.1: CLI framework
- `github.com/go-chi/chi/v5` v5.2.3: HTTP router for serve
- `github.com/gorilla/websocket` v1.5.4: Live reload

**Grafana Libraries**:
- `github.com/grafana/grafana-openapi-client-go`: Generated API client
- `github.com/grafana/grafana/pkg/apimachinery`: Grafana's K8s extensions
- `github.com/grafana/grafana-app-sdk/logging`: Structured logging

**Utilities**:
- `golang.org/x/sync` v0.17.0: errgroup for concurrency
- `github.com/goccy/go-yaml` v1.18.0: YAML codec
- `github.com/fsnotify/fsnotify` v1.9.0: File watching

### Security Considerations

**Credential Management**:
- API tokens stored in config file with 0600 permissions
- Password fields tagged with `datapolicy:"secret"` for redaction
- TLS certificate data base64-encoded in config
- Environment variables supported for CI/CD (avoid config files)

**Network Security**:
- TLS verification enabled by default
- Option to disable (insecure-skip-verify) for dev/test
- Custom CA certificates supported
- Client certificate authentication supported

**Resource Management**:
- Manager metadata prevents accidental overwrites
- Dry-run mode for safe testing
- Include-managed flag required to modify non-grafanactl resources

### Common Gotchas

1. **Namespace Confusion**: org-id (on-prem) vs stack-id (cloud) are both called "namespace"
2. **Version Handling**: Preferred version vs all versions affects discovery
3. **Folder Ordering**: Folders must be pushed before dashboards (dependency)
4. **Manager Metadata**: Resources created by UI can't be pushed unless --include-managed
5. **Format Preservation**: Pull/push roundtrip preserves original format (JSON/YAML)

### Future Architecture Directions

Based on TODO comments in code:

1. **Proper Manager Kind**: Currently uses kubectl as placeholder
2. **Resource Versioning**: Proper resourceVersion handling in updates
3. **Subresource Support**: Currently excluded, but may be needed
4. **Three-Way Merge**: Currently basic upsert, could use kubectl-style apply
5. **Timestamp/Checksum**: Manager metadata could include more provenance data

## Development Priorities & Technical Debt

### Current State Assessment (as of 2025-10-31)

**Quality Scores**:
- Architecture Quality: 9/10 (Excellent)
- Code Quality: 8/10 (Very Good)
- Testing Coverage: 6/10 (Moderate - needs improvement)
- Documentation: 9/10 (Excellent)
- Maintainability: 8/10 (Very Good)
- Security: 7/10 (Good - improvements needed)

**Key Metrics**:
- **Test Coverage**: Estimated 40-50% (target: 70%+)
- **Integration Tests**: None (docker-compose setup exists but unused)
- **Performance Limit**: ~10,000 resources before memory pressure (~1MB per 100 dashboards)
- **Scalability**: Hardcoded QPS=50, Burst=100 in rest.go:26-27
- **Overall Assessment**: Production-ready with identified improvement areas

### Testing Gaps

**Current Coverage Focus**:
- Selector parsing and resolution
- Filter matching logic
- Configuration loading and validation
- Resource grouping and sorting
- Manager field processing

**Missing Test Areas**:
- ❌ Integration tests with real Grafana instances (despite docker-compose setup)
- ❌ Push/Pull error scenarios and edge cases
- ❌ Concurrency edge cases (errgroup error propagation, cancellation)
- ❌ Discovery failure handling
- ❌ TLS configuration scenarios
- ❌ Live reload server functionality
- ❌ Folder dependency sorting edge cases

**Recommendations**:
1. Add integration test suite using docker-compose
2. Create mock Grafana API with test fixtures
3. Add concurrency tests for errgroup behavior
4. Use property-based testing for selector parsing
5. Extend table-driven tests for error cases
6. Target 70%+ coverage incrementally

### Technical Debt Inventory

#### High Priority (Address in Next 2-3 Sprints)

1. **Resource Versioning** (pusher.go:284)
   - *Current*: No resourceVersion handling in updates
   - *Impact*: Medium - Can cause race conditions in concurrent updates
   - *Effort*: Medium - Standard K8s conflict detection pattern
   - *Fix*: Implement conflict detection, retry logic, proper resourceVersion tracking

2. **Integration Tests**
   - *Current*: None exist despite docker-compose test environment
   - *Impact*: High - Limits confidence in changes, no E2E validation
   - *Effort*: High - Requires test infrastructure setup
   - *Fix*: Use existing docker-compose for smoke tests, test push/pull/delete workflows

3. **Three-Way Merge** (pusher.go:285)
   - *Current*: Simple upsert logic without proper server-side apply
   - *Impact*: Medium - Can cause conflicts in multi-manager scenarios
   - *Effort*: High - Requires careful implementation of kubectl-style apply
   - *Fix*: Implement field manager metadata, conflict resolution, force-conflicts flag

#### Medium Priority (Address in 6 Months)

4. **Manager Kind Placeholder** (resources.go:19)
   - *Current*: Uses "kubectl" as placeholder manager kind
   - *Impact*: Low - Functional but inaccurate in metadata
   - *Effort*: Low - String replacement and testing
   - *Fix*: Replace with "grafanactl" as proper manager identifier

5. **Test Coverage Improvement**
   - *Current*: Estimated 40-50% coverage
   - *Impact*: Medium - Some code paths untested
   - *Effort*: Medium - Incremental improvement over time
   - *Fix*: Add tests incrementally, focus on critical paths first

6. **Subresource Support Decision**
   - *Current*: Subresources containing "/" excluded in discovery (registry.go:212)
   - *Impact*: Low - May limit future extensibility
   - *Effort*: Medium - Requires design decision and implementation
   - *Fix*: Evaluate use cases, implement if needed, document decision

7. **Config File Permission Validation**
   - *Current*: Docs mention 0600 permissions but no code enforcement
   - *Impact*: Medium - Security risk for credential exposure
   - *Effort*: Low - Add validation on config load
   - *Fix*: Check file permissions, warn or error if too permissive

#### Low Priority (Address Opportunistically)

8. **Hardcoded Configuration Values** (rest.go:26-27)
   - *Current*: QPS=50, Burst=100 hardcoded
   - *Impact*: Low - Works for most use cases
   - *Effort*: Low - Add config fields and CLI flags
   - *Fix*: Make configurable via flags or config file

9. **Global Variables** (discovery/registry.go:19)
   - *Current*: `ignoredResourceGroups` as global var with lint exception
   - *Impact*: Low - Reasonable use but affects testability
   - *Effort*: Low - Move to configuration struct
   - *Fix*: Refactor into registry configuration

10. **Error Wrapping Consistency**
    - *Current*: Mixed use of wrapped and unwrapped errors
    - *Impact*: Low - Inconsistent error handling patterns
    - *Effort*: Low - Standardize on `fmt.Errorf` with `%w`
    - *Fix*: Establish error wrapping guidelines, apply consistently

11. **User Agent Configuration** (TODO in code)
    - *Current*: User agent not configurable
    - *Impact*: Low - Useful for tracking/debugging
    - *Effort*: Low - Add config option
    - *Fix*: Add user agent configuration support

12. **Alerts Resource Group** (discovery/registry.go:53)
    - *Current*: TODO comment to verify with alerting team
    - *Impact*: Low - May incorrectly filter alerts resources
    - *Effort*: Low - Verification and potential fix
    - *Fix*: Confirm with team, adjust filter if needed

### Performance Bottlenecks & Recommendations

**Current Limitations**:
- **Memory**: All resources loaded into memory (no streaming)
- **Estimated Usage**: ~1MB per 100 dashboards
- **Practical Limit**: ~10,000 resources before memory pressure
- **Rate Limits**: QPS=50, Burst=100 (hardcoded, not configurable)
- **Connection Pooling**: No visible configuration

**Recommendations**:
1. **Add Streaming Support**: For large resource sets (>1000 resources), implement chunked processing
2. **Configurable Rate Limits**: Expose QPS/Burst as CLI flags or config options
3. **Progress Indicators**: Add for long-running operations (push/pull 1000+ resources)
4. **Caching**: Consider caching discovery results to disk for faster startup
5. **Memory Profiling**: Add `--profile` flag for memory/CPU profiling during operations
6. **Adaptive Rate Limiting**: Implement backoff and retry with adaptive rate adjustment

### Security Recommendations

**Current Security Posture**:
- ✅ API tokens via bearer authentication
- ✅ Password fields tagged with `datapolicy:"secret"` for redaction
- ✅ Environment variable support (avoid storing credentials in files)
- ✅ TLS verification enabled by default
- ✅ Custom CA certificates and client certificate authentication supported

**Security Gaps & Improvements**:

1. **Config File Permission Validation** (High Priority)
   - *Issue*: No explicit file permission check in code
   - *Risk*: Credentials exposed if file permissions too permissive
   - *Fix*: Add validation on config load, warn if not 0600

2. **Secret Encryption at Rest** (Medium Priority)
   - *Issue*: Credentials stored in plaintext in config file
   - *Risk*: Credentials compromised if file accessed
   - *Fix*: Consider OS keychain integration (e.g., keyring libraries)

3. **Insecure TLS Warning** (Medium Priority)
   - *Issue*: TLS verification can be disabled with `Insecure: true`
   - *Risk*: Man-in-the-middle attacks if used in production
   - *Fix*: Add prominent warnings when `insecure-skip-verify` is used

4. **Input Validation** (Low Priority)
   - *Current*: Relies on Grafana API validation (no client-side schema validation)
   - *Risk*: Poor error messages, unnecessary network round-trips
   - *Fix*: Implement client-side schema validation using OpenAPI specs

### Code Quality Standards

**Current Strengths**:
- Clean separation of concerns (no circular dependencies)
- Consistent options pattern for commands
- Thoughtful error handling with custom types
- Minimal, focused interfaces for testability
- Good adherence to Go conventions

**Areas for Improvement**:

1. **Error Wrapping Consistency**
   - Standardize on `fmt.Errorf` with `%w` for all error wrapping
   - Ensure error chains are preserved for debugging

2. **Magic Numbers**
   - Extract hardcoded values to named constants
   - Examples: QPS/Burst limits, concurrency defaults, timeouts

3. **Documentation Comments**
   - Ensure all exported functions have GoDoc comments
   - Document non-obvious behavior and edge cases

4. **Test Coverage**
   - Require 70%+ coverage for new code
   - Add integration tests for new workflows
   - Use table-driven tests for all parsing/validation logic

### Prioritized Roadmap

**Short-term (0-6 months)**:
1. ✅ Add integration test suite with docker-compose
2. ✅ Implement resource versioning with conflict detection
3. ✅ Add config file permission validation
4. ✅ Improve test coverage to 70%+
5. ✅ Add progress indicators for long-running operations
6. ✅ Implement client-side schema validation using OpenAPI specs

**Medium-term (6-12 months)**:
1. 🔄 Implement proper three-way merge with server-side apply semantics
2. 🔄 Add streaming support for large resource sets
3. 🔄 Implement watch operations for real-time updates
4. 🔄 Add diff command (show changes before push, like kubectl diff)
5. 🔄 Replace manager kind placeholder with "grafanactl"
6. 🔄 Make rate limits configurable

**Long-term (12+ months)**:
1. 🔮 Plugin system for custom resource handlers
2. 🔮 Multi-cluster support (manage multiple Grafana instances simultaneously)
3. 🔮 GitOps integration (native support for flux/argocd patterns)
4. 🔮 Resource templates (template rendering similar to Helm)
5. 🔮 Advanced filtering (label selectors, field selectors)
6. 🔮 Batch operations with rollback support

### Comparative Analysis

**vs kubectl**:
- ✅ grafanactl successfully adopts kubectl UX patterns
- ✅ Context-based configuration matches kubectl
- ✅ Resource abstraction similar to K8s objects
- ❌ kubectl has more mature three-way merge
- ❌ kubectl has extensive plugin ecosystem

**vs terraform**:
- ✅ grafanactl is lighter-weight and dashboard-focused
- ✅ Better live development experience (serve command)
- ❌ terraform has state management and planning
- ❌ terraform has broader resource support

**vs grizzly**:
- ✅ grafanactl uses standard K8s patterns vs custom
- ✅ grafanactl has discovery system
- ❌ grizzly has longer history and maturity
- ❌ grizzly has template support

**Unique Value Proposition**:
1. K8s-native patterns for Grafana resources
2. Dynamic discovery of available resources
3. Live reload server for development workflows
4. Official Grafana Labs tool (better integration and support)

**Ideal Use Cases**:
- Dashboards-as-code workflows
- GitOps for Grafana resources
- Multi-environment Grafana management (dev/staging/prod)
- CI/CD automation
- Development with live preview

**Not Ideal For**:
- One-off dashboard edits (UI is faster)
- Non-technical users
- Grafana <12 (not supported)
- Extremely large deployments (>10,000 resources without streaming)

## Go Code Organization Standards

When writing Go code, always organize code symbols in a given file in this order:

1. exported constants
2. unexported constants
3. exported variables
4. unexported variables
5. exported functions
6. exported interface definitions
7. exported type definitions
8. exported methods
9. unexported methods
10. unexported interface definitions
11. unexported functions