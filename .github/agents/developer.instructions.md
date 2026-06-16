---
description: Developer Instructions for GitHub Agentic Workflows
applyTo: "**/*"
---

# Developer Instructions

Guidelines, patterns, and standards for GitHub Agentic Workflows.

---

## Code Organization Patterns

#### 1. Create Functions Pattern (`create_*.go`)

One file per GitHub entity. Examples: `create_issue.go`, `create_pull_request.go`, `create_discussion.go`, `create_code_scanning_alert.go`, `create_agent_task.go`.

#### 2. Engine Separation Pattern

Each AI engine in its own file; shared helpers in `engine_helpers.go`. Examples: `copilot_engine.go`, `claude_engine.go`, `codex_engine.go`, `custom_engine.go`.

#### 3. Test Organization Pattern

Tests alongside implementation:
- Feature: `feature.go` + `feature_test.go`
- Integration: `feature_integration_test.go`
- Scenario: `feature_scenario_test.go`

### File Creation Decision Tree

```mermaid
graph TD
    A[Need New Functionality?] --> B{Size > 200 lines?}
    B -->|Yes| C[Create New File]
    B -->|No| D{Related to Existing File?}
    D -->|Yes| E[Add to Existing File]
    D -->|No| C
    C --> F{Multiple Related Operations?}
    F -->|Yes| G[Use Create Pattern: create_*.go]
    F -->|No| H[Use Domain Pattern]
    E --> I{File > 1000 lines?}
    I -->|Yes| J[Consider Splitting]
    I -->|No| K[Keep in Same File]
```

### File Size Guidelines

- **Small (50-200 lines)**: Utilities, helpers, simple features
- **Medium (200-500 lines)**: Domain-specific logic, focused features
- **Large (500-1000 lines)**: Complex features, comprehensive implementations
- **Very Large (1000+ lines)**: Consider splitting if not cohesive
---

## Validation Architecture

Validates workflow configs before compilation. Centralized in `validation.go`, or domain-specific in dedicated files.

### Validation Flow

```mermaid
graph TD
    A[Workflow YAML] --> B[Parser]
    B --> C[Validation System]
    C --> D[Centralized Validation]
    C --> E[Domain-Specific Validation]
    D --> F[validation.go]
    E --> G[strict_mode_validation.go]
    E --> H[pip.go]
    E --> I[npm.go]
    F --> J{Valid?}
    G --> J
    H --> J
    I --> J
    J -->|Yes| K[Compiler]
    J -->|No| L[Error Report]
```

### Centralized Validation: `pkg/workflow/validation.go`

- `validateExpressionSizes()` — GitHub Actions expression size limits
- `validateContainerImages()` — Docker images accessible
- `validateRuntimePackages()` — runtime package deps
- `validateGitHubActionsSchema()` — GitHub Actions YAML schema
- `validateNoDuplicateCacheIDs()` — unique cache IDs
- `validateSecretReferences()` — secret reference syntax
- `validateRepositoryFeatures()` — repo capabilities (issues, discussions)

### Domain-Specific Validation

#### Strict Mode: `strict_mode_validation.go`

- `validateStrictMode()` — main orchestrator
- `validateStrictPermissions()` — refuses write permissions
- `validateStrictNetwork()` — requires explicit network config
- `validateStrictMCPNetwork()` — requires network config on custom MCP servers
- `validateStrictBashTools()` — refuses bash wildcards

#### Package Validation

- **Python/pip**: `pip.go` — PyPI availability
- **Node.js/npm**: `npm.go` — npm packages used with npx

### Where to Add Validation

```mermaid
graph TD
    A[Need Validation?] --> B{Domain-Specific?}
    B -->|Yes| C{Security-Related?}
    B -->|No| D[validation.go]
    C -->|Yes| E[strict_mode_validation.go]
    C -->|No| F{Package Manager?}
    F -->|Python| G[pip.go]
    F -->|Node.js| H[npm.go]
    F -->|Other| I[Create New Domain File]
```
---

## Development Standards

### Capitalization Guidelines

- **Product**: "GitHub Agentic Workflows" (always capitalized)
- **Features**: sentence case (e.g., "safe output messages")
- **Filenames**: lowercase with hyphens (e.g., `code-organization.md`)
- **Code**: language conventions (`camelCase` JS, `snake_case` Python)

### Breaking Change Rules

**Breaking**:
- Renaming/removing CLI commands, flags, options
- Changing default behavior users depend on
- Removing config format support
- Changing exit codes or parsable error messages

**Non-Breaking**:
- New optional flags/commands
- New output formats
- Internal refactoring (same external behavior)
- New features not affecting existing
---

## String Processing

### Sanitize vs Normalize

```mermaid
graph TD
    A[Need String Processing?] --> B{Security Concern?}
    B -->|Yes| C[Sanitize]
    B -->|No| D{Consistency Needed?}
    C --> E[sanitizeGitHubLabel]
    C --> F[sanitizeGitHubBranch]
    C --> G[sanitizeGitHubIssueTitle]
    D -->|Yes| H[Normalize]
    D -->|No| I[Use As-Is]
    H --> J[normalizeWhitespace]
    H --> K[normalizeLineEndings]
```

**Sanitize** — fix chars that break security or GitHub API:
- `sanitizeGitHubLabel()` — label requirements (no emoji, length limits)
- `sanitizeGitHubBranch()` — Git ref rules
- `sanitizeGitHubIssueTitle()` — avoid problematic chars

**Normalize** — standardize format, no security implication:
- `normalizeWhitespace()` — spaces, tabs, newlines
- `normalizeLineEndings()` — CRLF → LF
- `normalizeMarkdown()` — markdown formatting
---

## YAML Handling

### YAML 1.1 vs 1.2 Gotchas

**Critical**: GitHub Actions uses YAML 1.1; many Go libraries default to 1.2.

**Differences**:
- `on`: YAML 1.1 = boolean `true`, YAML 1.2 = string
- `yes`/`no`: YAML 1.1 = booleans, YAML 1.2 = strings
- Octal numbers: different parsing rules

**Solution**: Use `goccy/go-yaml` (supports YAML 1.1)

```go
import "github.com/goccy/go-yaml"

// Correct YAML 1.1 parsing
var workflow map[string]interface{}
err := yaml.Unmarshal(data, &workflow)
```

**Affected Keywords**:
- Workflow triggers: `on`, `push`, `pull_request`
- Boolean values: `yes`, `no`, `true`, `false`, `on`, `off`
- Null values: `null`, `~`
---

## Safe Output Messages

Structured communication between agents and GitHub API.

### Message Categories

| Category | Purpose | Footer | Example |
|----------|---------|--------|---------|
| **Issues** | Create/update issues | With issue number | `> AI generated by [Workflow](url) for #123` |
| **Pull Requests** | Create/update PRs | With PR number | `> AI generated by [Workflow](url) for #456` |
| **Discussions** | Create discussions | With discussion number | `> AI generated by [Workflow](url)` |
| **Comments** | Add comments | Context-aware | `> AI generated by [Workflow](url) for #123` |

### Staged Mode Indicator

🎭 marks preview mode across all safe output types.

### Message Structure

```yaml
safe_outputs:
  create_issue:
    title: "Issue title"
    body: |
      ## Description

      Content here

      ---
      > AI generated by [WorkflowName](run_url)
```
---

## Custom GitHub Actions

### Architecture

```mermaid
graph LR
    MD[Workflow .md] --> Compiler
    Compiler --> YAML[.lock.yml]
    YAML --> GHA[GitHub Actions Runner]
    GHA --> Actions[Custom Actions]
    Actions --> API[GitHub API]
```

### Build System

Go: `pkg/cli/actions_build_command.go`. No JS build scripts.

**Commands**:
- `make actions-build` — build all
- `make actions-validate` — validate config
- `make actions-clean` — clean artifacts

**Directory**:
```
actions/
└── setup/
    ├── action.yml
    ├── setup.sh
    ├── js/
    └── sh/
```
---

## Security Best Practices

### Template Injection Prevention

**Rule**: Never interpolate user input into GitHub Actions expressions or shell commands.

**Vulnerable**:
```yaml
# ❌ UNSAFE
- run: echo "Title: ${{ github.event.issue.title }}"
```

**Safe**:
```yaml
# ✅ SAFE — env var
- env:
    TITLE: ${{ github.event.issue.title }}
  run: echo "Title: ${TITLE}"
```

### GitHub Actions Security

- Pin actions to commit SHAs, not tags
- Minimal `permissions:` block
- Validate external inputs
- Never log secrets/tokens
- Use OIDC for cloud auth

**Example**:
```yaml
permissions:
  contents: read
  issues: write
  pull-requests: write

steps:
  - uses: actions/checkout@a1b2c3d4... # Pinned SHA
```
---

## Testing Framework

### Test Types

| Test Type | Purpose | Location | Run Frequency |
|-----------|---------|----------|---------------|
| **Unit Tests** | Test individual functions | `*_test.go` | Every commit |
| **Integration Tests** | Test component interactions | `*_integration_test.go` | Pre-merge |
| **Security Regression Tests** | Prevent security issues | `security_regression_test.go` | Every commit |
| **Fuzz Tests** | Find edge cases | `*_fuzz_test.go` | Continuous |
| **Benchmark Tests** | Performance tracking | `*_benchmark_test.go` | Pre-release |

### Visual Regression Testing

Golden files capture expected console output (tables, boxes, trees, error formatting).

**Commands**:
```bash
go test -v ./pkg/console -run='^TestGolden_'
make update-golden  # only when intentionally changing output
```

**When to Update**:
- ✅ Intentionally improving formatting
- ✅ Fixing visual bugs
- ✅ Adding new columns/fields
- ❌ Tests fail unexpectedly
- ❌ Unrelated code changes
---

## Repo-Memory System

Persistent, git-backed storage across workflow runs. State lives in dedicated git branches with auto-sync.

### Architecture Overview

```mermaid
graph TD
    A[Agent Job Start] --> B[Clone memory/{id} branch]
    B --> C[Agent reads/writes files]
    C --> D[Upload artifact: repo-memory-{id}]
    D --> E[Push Repo Memory Job]
    E --> F[Download artifact]
    F --> G[Validate files]
    G --> H[Commit to memory/{id}]
    H --> I[Push to repository]
```

### Path Conventions

| Pattern | Format | Example | Purpose |
|---------|--------|---------|---------|
| **Memory Directory** | `/tmp/gh-aw/repo-memory/{id}` | `/tmp/gh-aw/repo-memory/default` | Runtime directory for agent |
| **Artifact Name** | `repo-memory-{id}` | `repo-memory-default` | GitHub Actions artifact |
| **Branch Name** | `memory/{id}` | `memory/default` | Git branch for storage |

### Data Flow

1. Clone `memory/{id}` branch
2. Agent reads/writes files
3. Upload directory as artifact
4. Download artifact, validate constraints
5. Commit and push to `memory/{id}`

### Key Configuration

```yaml
repo-memory:
  - id: default
    create-orphan: true
    allow-artifacts: true

  - id: orchestration
    create-orphan: true
    max-file-size: 1MB
    max-files: 100
```

**Validation Constraints**: max file size, max file count, allowed/blocked patterns, size/count tracking in commit messages.
---

## Hierarchical Agent Management

Meta-orchestrators manage multiple agents and workflows at scale.

### Meta-Orchestrator Roles

| Role | File | Purpose | Schedule |
|------|------|---------|----------|
| **Workflow Health Manager** | `workflow-health-manager.md` | Monitor workflow health | Daily |
| **Agent Performance Analyzer** | `agent-performance-analyzer.md` | Analyze agent quality | Daily |
---

## Release Management

### Changesets

```bash
npx changeset            # create
npx changeset version    # release
npx changeset publish
```

**Format**:
```markdown
---
"gh-aw": patch
---

Brief description of the change
```

**Version Types**: `major` (breaking), `minor` (new features, backward compatible), `patch` (bug fixes).

### End-to-End Feature Testing

1. Use `.github/workflows/dev.md` as test workflow
2. Add scenarios as PR comments
3. Dev Hawk verifies behavior
4. Do not merge dev.md changes — it stays a reusable test harness
---

## Scope Hints for Complex Workflows

More upfront constraints → faster, more accurate generated workflow.

### Workflow-Type Guidance

| Workflow Type | Key Constraints to Specify |
|---------------|---------------------------|
| **File-parsing** | File format (lcov, cobertura, jacoco, etc.) and path |
| **Cross-branch diff** | Branch strategy (base/head names, e.g. `main`/`feature`) |
| **Reporting** | Output format (markdown, JSON, HTML) |
| **Coverage analysis** | Coverage threshold (e.g. 80%), report location |
| **Dependency audit** | Package manager (npm, pip, cargo), severity filter |
| **Performance benchmarks** | Benchmark tool, metric to track, regression threshold |

### Examples

#### ✅ Constrained (faster, more accurate)

```
Create a workflow that reads an lcov coverage report from `coverage/lcov.info`
and comments on PRs if coverage drops below 80%.
```

```
Create a workflow that diffs `main` and the PR head branch, lists changed
Go files, and posts a markdown summary as a PR comment.
```

```
Create a workflow that runs `npm audit --json`, filters results for
high-severity vulnerabilities, and fails the check if any are found.
```

#### ⚠️ Unconstrained (may cause timeout or vague output)

```
Create a workflow that monitors test coverage.
```

```
Create a workflow that checks for dependency vulnerabilities.
```

```
Create a workflow that compares branches and reports differences.
```

### Timeout Prevention Checklist

Before submitting a complex workflow request:

- [ ] **Input path/format** — e.g. `coverage/lcov.info` in lcov format
- [ ] **Triggering event** — e.g. `pull_request`, `push to main`, `schedule`
- [ ] **Success/failure criterion** — e.g. coverage ≥ 80%, zero high-severity CVEs
- [ ] **Output destination** — e.g. PR comment, issue, Slack, artifact
- [ ] **Scope boundaries** — e.g. only changed files, only `src/`
---

## PR Deduplication Protocol

Run before every PR — repeated closed attempts waste CI and context.

### Pre-flight Duplicate PR Check

Search closed PRs via GitHub MCP `search_pull_requests`:

1. Extract 2–4 keywords from the feature/fix title.
2. Search e.g. `is:pr is:closed head:copilot/ <keywords>` or `is:pr is:closed <keywords>`.
3. None found → proceed.
4. Any found → do [Prior Failure Analysis](#prior-failure-analysis) first.

### Prior Failure Analysis

Before any code exploration:

1. Read the closed PR description, review comments, and timeline.
2. Identify **root cause of closure**:
   - Reviewer requested changes → list them
   - CI/test failures → identify failing checks
   - Scope mismatch → clarify what was actually requested
   - Duplicate of another fix → link to that fix
3. Verify the new implementation addresses the root cause.
4. Add a "## Prior Attempts" section to the new PR:
   - Link(s) to prior closed PR(s)
   - Why each closed
   - What is different this time

**Example:**

```markdown
## Prior Attempts

- #1234 (closed): CI failed on `TestFoo` due to missing nil check — fixed in this PR
- #1189 (closed): Reviewer requested scope reduction — this PR limits change to X only
```

### Retry Limit Circuit Breaker

If **two or more** closed PRs exist on the same topic:

1. **Do not open a third PR** without explicit human review.
2. Comment on the originating issue:
   - List prior closed PRs and close reasons
   - Explain what changed in the new approach
   - Request maintainer approval
3. Label the issue `copilot-retry-blocked`.
4. Wait for the maintainer to remove the label or approve.

**Rationale:** Two failed attempts indicate a systemic problem (unclear requirements, missing context, design issue) code alone cannot fix.

---

### File Locations

| Feature | Implementation File | Test File |
|---------|-------------------|-----------|
| Validation | `pkg/workflow/validation.go` | `pkg/workflow/validation_test.go` |
| Safe Outputs | `pkg/workflow/safe_outputs.go` | `pkg/workflow/safe_outputs_test.go` |
| String Processing | `pkg/workflow/strings.go` | `pkg/workflow/strings_test.go` |
| Actions Build | `pkg/cli/actions_build_command.go` | `pkg/cli/actions_build_command_test.go` |
| Schema Validation | `pkg/parser/schemas/` | Various test files |

### Common Patterns

**New GitHub entity handler**:
1. Create `create_<entity>.go` in `pkg/workflow/`
2. Implement `Create<Entity>()`
3. Add validation (centralized or domain-specific)
4. Add test file
5. Update safe output messages

**New validation**:
1. Centralized or domain-specific?
2. Add function in appropriate file
3. Call from main orchestrator
4. Test valid + invalid cases
5. Document rules

**New engine**:
1. Create `<engine>_engine.go` in `pkg/workflow/`
2. Implement engine interface
3. Use `engine_helpers.go` for shared logic
4. Add engine-specific tests
5. Register in engine factory

---

**Last Updated**: 2026-04-28
