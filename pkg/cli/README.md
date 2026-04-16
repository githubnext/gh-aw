# cli Package

> CLI command implementations for the `gh aw` extension — the primary user interface for authoring, compiling, running, and monitoring agentic GitHub workflows.

## Overview

The `cli` package implements all commands exposed through the `gh aw` CLI extension. Each command is implemented as a Cobra command with a dedicated `New*Command()` constructor and a `Run*()` function that encapsulates the testable business logic.

The package is intentionally decomposed into many small files grouped by feature domain (e.g., `compile_*.go`, `audit_*.go`, `run_*.go`, `mcp_*.go`). This structure keeps individual files under 300 lines and promotes independent testing of each sub-domain.

All diagnostic output MUST go to `stderr` using `console` formatting helpers. Structured output (JSON, hashes, graphs) goes to `stdout`.

## Command Groups

| Command | Entry Point | Description |
|---------|-------------|-------------|
| `gh aw add` | `NewAddCommand` | Add remote or local workflows to the repository |
| `gh aw add-wizard` | `NewAddWizardCommand` | Interactive wizard for adding workflows |
| `gh aw compile` | (compile_command.go) | Compile `.md` workflow files into GitHub Actions `.lock.yml` |
| `gh aw run` | `NewRunCommand` (run_command.go) | Dispatch and monitor workflow runs |
| `gh aw audit` | `NewAuditCommand` | Audit a specific workflow run by run ID |
| `gh aw audit diff` | `NewAuditDiffSubcommand` | Diff audit data between multiple runs |
| `gh aw logs` | `NewLogsCommand` | Download and analyze workflow run logs |
| `gh aw mcp` | `NewMCPCommand` | Manage MCP server configurations |
| `gh aw mcp add` | `NewMCPAddSubcommand` | Add an MCP tool to a workflow |
| `gh aw mcp inspect` | `NewMCPInspectSubcommand` | Inspect MCP servers in a workflow |
| `gh aw mcp list` | `NewMCPListSubcommand` | List workflows using MCP servers |
| `gh aw mcp list-tools` | `NewMCPListToolsSubcommand` | List tools for a specific MCP server |
| `gh aw mcp server` | `NewMCPServerCommand` | Run as an MCP server (for IDE integration) |
| `gh aw update` | `NewUpdateCommand` | Update workflows from upstream sources |
| `gh aw upgrade` | `NewUpgradeCommand` | Upgrade workflows to latest format |
| `gh aw validate` | `NewValidateCommand` | Validate workflow files without compiling |
| `gh aw fix` | `NewFixCommand` | Apply automatic codemods to fix deprecated patterns |
| `gh aw status` | `NewStatusCommand` | Show status of workflows in the repository |
| `gh aw health` | `NewHealthCommand` | Compute health metrics across workflow runs |
| `gh aw checks` | `NewChecksCommand` | Show CI check results for a PR |
| `gh aw domains` | `NewDomainsCommand` | List domains used by workflows |
| `gh aw hash` | `NewHashCommand` | Print frontmatter hash of a workflow file |
| `gh aw init` | `NewInitCommand` | Initialize a repository for agentic workflows |
| `gh aw list` | `NewListCommand` | List installed workflows |
| `gh aw pr` | `NewPRCommand` | Pull-request helpers |
| `gh aw project` | `NewProjectCommand` | Project management helpers |
| `gh aw remove` | `NewRemoveCommand` | Remove workflow files from the repository |
| `gh aw secrets` | `NewSecretsCommand` | Manage workflow secrets |
| `gh aw trial` | `NewTrialCommand` | Run trial workflow executions |
| `gh aw deps` | (deps_*.go) | Dependency inspection and security advisories |
| `gh aw completion` | `NewCompletionCommand` | Generate shell completion scripts |

## Public API

### Key Types

| Type | File | Description |
|------|------|-------------|
| `CompileConfig` | `compile_config.go` | Configuration for `CompileWorkflows` — file list, flags, validation options |
| `ValidationResult` | `compile_config.go` | Result of a compilation validation pass |
| `AddOptions` | `add_command.go` | Options controlling workflow addition behavior |
| `AddWorkflowsResult` | `add_command.go` | Result of `AddWorkflows` / `AddResolvedWorkflows` |
| `ResolvedWorkflow` | `add_workflow_resolution.go` | A single resolved workflow with source metadata |
| `ResolvedWorkflows` | `add_workflow_resolution.go` | Collection of resolved workflows |
| `RunOptions` | `run_workflow_execution.go` | Options for `RunWorkflowOnGitHub` |
| `WorkflowRunResult` | `run_workflow_execution.go` | Result of a triggered workflow run |
| `AuditData` | `audit_report.go` | Full audit data structure for a workflow run |
| `AuditDiff` | `audit_diff.go` | Diff between two audit runs |
| `CrossRunAuditReport` | `audit_cross_run.go` | Cross-run trend analysis |
| `HealthConfig` | `health_command.go` | Configuration for health computation |
| `WorkflowHealth` | `health_metrics.go` | Per-workflow health metrics |
| `HealthSummary` | `health_metrics.go` | Aggregate health across all workflows |
| `DependencyReport` | `deps_report.go` | Full dependency report |
| `OutdatedDependency` | `deps_outdated.go` | An outdated dependency entry |
| `SecurityAdvisory` | `deps_security.go` | A security advisory entry |
| `WorkflowStatus` | `status_command.go` | Run status for a single workflow |
| `MCPRegistryClient` | `mcp_registry.go` | Client for the MCP registry API |
| `ToolGraph` | `tool_graph.go` | Dependency graph of MCP tools |
| `DependencyGraph` | `dependency_graph.go` | Dependency graph across workflows |
| `FileTracker` | `file_tracker.go` | Tracks files modified during an operation |
| `RepeatOptions` | `retry.go` | Options for `ExecuteWithRepeat` polling loop |
| `PollOptions` | `signal_aware_poll.go` | Options for `PollWithSignalHandling` |
| `FixConfig` | `fix_command.go` | Configuration for `RunFix` codemods |
| `TrialOptions` | `trial_types.go` | Options for `RunWorkflowTrials` |
| `WorkflowTrialResult` | `trial_types.go` | Result of a trial run |
| `UpgradeConfig` | `upgrade_command.go` | Configuration for `NewUpgradeCommand` |
| `ChecksConfig` | `checks_command.go` | Configuration for `RunChecks` |
| `ChecksResult` | `checks_command.go` | Result of `FetchChecksResult` |

### Key Functions

| Function | Signature | Description |
|----------|-----------|-------------|
| `CompileWorkflows` | `func(ctx, CompileConfig) ([]*workflow.WorkflowData, error)` | Orchestrates compilation of one or more workflow files |
| `CompileWorkflowWithValidation` | `func(*workflow.Compiler, filePath string, ...) error` | Compiles and validates a single workflow file |
| `AddWorkflows` | `func([]string, AddOptions) (*AddWorkflowsResult, error)` | Adds workflows from string specs |
| `ResolveWorkflows` | `func([]string, bool) (*ResolvedWorkflows, error)` | Resolves workflow specs to local paths and metadata |
| `RunWorkflowOnGitHub` | `func(ctx, string, RunOptions) error` | Dispatches a single workflow run on GitHub |
| `RunWorkflowsOnGitHub` | `func(ctx, []string, RunOptions) error` | Dispatches multiple workflows |
| `AuditWorkflowRun` | `func(ctx, runID int64, ...) error` | Downloads and renders an audit report for a run |
| `RunAuditDiff` | `func(ctx, baseRunID, compareRunIDs, ...) error` | Renders a diff between audit runs |
| `DownloadWorkflowLogs` | `func(ctx, workflowName string, ...) error` | Downloads and analyzes workflow logs |
| `RunListWorkflows` | `func(repo, path, pattern string, ...) error` | Lists installed workflows |
| `StatusWorkflows` | `func(pattern string, ...) error` | Prints workflow run status |
| `GetWorkflowStatuses` | `func(pattern, ref, ...) ([]WorkflowStatus, error)` | Fetches workflow statuses |
| `RunHealth` | `func(HealthConfig) error` | Computes and renders workflow health metrics |
| `CalculateWorkflowHealth` | `func(string, []WorkflowRun, float64) WorkflowHealth` | Pure health computation for a single workflow |
| `CalculateHealthSummary` | `func([]WorkflowHealth, string, float64) HealthSummary` | Aggregate health computation |
| `RunFix` | `func(FixConfig) error` | Applies automatic codemods |
| `GetAllCodemods` | `func() []Codemod` | Returns all available codemods |
| `InitRepository` | `func(InitOptions) error` | Initializes a repo with the `gh-aw` setup |
| `CreateWorkflowMarkdownFile` | `func(string, bool, bool, string) error` | Creates a new workflow markdown file |
| `IsRunnable` | `func(string) (bool, error)` | Checks whether a workflow file is runnable |
| `RunWorkflowInteractively` | `func(ctx, ...) error` | Interactive workflow selection and dispatch |
| `AddMCPTool` | `func(string, string, ...) error` | Adds an MCP server to a workflow file |
| `InspectWorkflowMCP` | `func(string, ...) error` | Inspects MCP server configurations |
| `ListWorkflowMCP` | `func(string, bool) error` | Lists MCP server info for a workflow |
| `UpdateActions` | `func(bool, bool, bool) error` | Bulk-updates GitHub Action versions in workflows |
| `ActionsBuildCommand` | `func() error` | Builds all custom actions in `actions/` |
| `ActionsValidateCommand` | `func() error` | Validates all `action.yml` files under `actions/` |
| `ActionsCleanCommand` | `func() error` | Removes generated action build artifacts |
| `GenerateActionMetadataCommand` | `func() error` | Generates `action.yml` and README metadata for selected action modules |
| `UpdateWorkflows` | `func([]string, ...) error` | Updates workflows from upstream sources |
| `RemoveWorkflows` | `func(string, bool, string) error` | Removes workflow files |
| `ValidateWorkflowName` | `func(string) error` | Validates a workflow name identifier |
| `GetBinaryPath` | `func() (string, error)` | Returns the path to the `gh-aw` binary |
| `GetCurrentRepoSlug` | `func() (string, error)` | Returns `owner/repo` for the current directory |
| `GetVersion` | `func() string` | Returns the current CLI version |
| `SetVersionInfo` | `func(string)` | Sets the version at startup |
| `EnableWorkflowsByNames` | `func([]string, string) error` | Enables GitHub Actions workflows |
| `DisableWorkflowsByNames` | `func([]string, string) error` | Disables GitHub Actions workflows |
| `CheckOutdatedDependencies` | `func(bool) ([]OutdatedDependency, error)` | Checks for outdated dependencies |
| `CheckSecurityAdvisories` | `func(bool) ([]SecurityAdvisory, error)` | Checks for known CVEs |
| `GenerateDependencyReport` | `func(bool) (*DependencyReport, error)` | Full dependency analysis report |
| `InstallShellCompletion` | `func(bool, CommandProvider) error` | Installs shell completions |
| `PollWithSignalHandling` | `func(PollOptions) error` | Polls a predicate with SIGINT handling |
| `ExecuteWithRepeat` | `func(RepeatOptions) error` | Repeats an operation with delay |
| `IsRunningInCI` | `func() bool` | Detects CI environment |
| `DetectShell` | `func() ShellType` | Detects the user's current shell |

## Usage Examples

### Compiling a workflow

```go
data, err := cli.CompileWorkflows(ctx, cli.CompileConfig{
    MarkdownFiles: []string{".github/workflows/my-workflow.md"},
    Verbose:       true,
    Validate:      true,
    Strict:        false,
})
```

### Running a workflow

```go
err := cli.RunWorkflowOnGitHub(ctx, "my-workflow", cli.RunOptions{
    Repo:    "owner/repo",
    Verbose: true,
})
```

### Auditing a run

```go
err := cli.AuditWorkflowRun(ctx, runID, "owner", "repo", "github.com",
    "/tmp/output", true, true, false, 0, 0, nil)
```

### Checking workflow health

```go
err := cli.RunHealth(cli.HealthConfig{
    Pattern:   "*.md",
    Threshold: 0.8,
    Period:    "30d",
})
```

## Design Decisions

- **File-per-feature decomposition**: Large feature domains (compile, audit, logs, run) are split into multiple files (`_command.go`, `_config.go`, `_helpers.go`, `_orchestrator.go`, etc.) to keep each file focused and under 300 lines.
- **Testable Run functions**: Every command has a `New*Command()` for Cobra wiring and a `Run*()` function with explicit parameters for unit testing without CLI arg parsing overhead.
- **Stderr for diagnostics**: All user-visible messages use `console.Format*Message` helpers and write to `stderr`, preserving `stdout` for structured machine-readable output.
- **Context propagation**: Long-running operations accept `context.Context` to support cancellation (SIGINT, timeouts).
- **Config structs**: Command options are collected into dedicated `*Config` or `*Options` structs rather than passed as long argument lists, improving readability and testability.

## Dependencies

**Internal**:
- `pkg/workflow` — workflow compilation and data types
- `pkg/parser` — markdown frontmatter parsing
- `pkg/console` — terminal output formatting
- `pkg/logger` — structured debug logging
- `pkg/constants` — engine names, job names, feature flags
- `pkg/stringutil`, `pkg/fileutil`, `pkg/gitutil`, `pkg/repoutil` — utilities

**External**:
- `github.com/spf13/cobra` — CLI framework
- `github.com/cli/go-gh/v2` — GitHub CLI integration

## Thread Safety

Individual command `Run*` functions are not concurrently safe unless explicitly documented. The `CompileWorkflows` orchestrator serializes compilation by default; parallel compilation is gated by `CompileConfig` flags.

---

*This specification is automatically maintained by the [spec-extractor](../../.github/workflows/spec-extractor.md) workflow.*
