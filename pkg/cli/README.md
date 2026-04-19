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
| `gh aw run` | `RunWorkflowOnGitHub` (main.go) | Dispatch and monitor workflow runs |
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
| `gh aw pr transfer` | `NewPRTransferSubcommand` | Transfer a pull request to another repository |
| `gh aw project` | `NewProjectCommand` | Project management helpers |
| `gh aw project new` | `NewProjectNewCommand` | Create a new GitHub Project V2 board |
| `gh aw remove` | `RemoveWorkflows` (main.go) | Remove workflow files from the repository |
| `gh aw secrets` | `NewSecretsCommand` | Manage workflow secrets |
| `gh aw secrets set` | (secret_set_command.go) | Create or update a repository secret |
| `gh aw secrets bootstrap` | (secret_set_command.go) | Validate and configure all required secrets for workflows |
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
| `AddResolvedWorkflows` | `func([]string, *ResolvedWorkflows, AddOptions) (*AddWorkflowsResult, error)` | Adds pre-resolved workflows |
| `FetchWorkflowFromSource` | `func(*WorkflowSpec, bool) (*FetchedWorkflow, error)` | Fetches a workflow from a remote or local source |
| `FetchIncludeFromSource` | `func(string, *WorkflowSpec, bool) ([]byte, string, error)` | Fetches an `@include` target from source |
| `MergeWorkflowContent` | `func(base, current, new, oldSpec, newSpec, localPath string, bool) (string, bool, error)` | Three-way merge of workflow content |
| `CompileWorkflowDataWithValidation` | `func(*workflow.Compiler, *workflow.WorkflowData, string, ...) error` | Compiles a pre-loaded WorkflowData and runs security validators |
| `ResolveWorkflowPath` | `func(string) (string, error)` | Resolves a workflow name to its absolute file path |
| `ExtractWorkflowDescription` | `func(string) string` | Extracts the `description` field from workflow markdown content |
| `ExtractWorkflowDescriptionFromFile` | `func(string) string` | Extracts the `description` field from a workflow file |
| `ExtractWorkflowEngine` | `func(string) string` | Extracts the `engine` field from workflow markdown content |
| `ExtractWorkflowPrivate` | `func(string) bool` | Returns true if the workflow is marked private |
| `UpdateFieldInFrontmatter` | `func(content, fieldName, fieldValue string) (string, error)` | Sets a field in frontmatter YAML |
| `SetFieldInOnTrigger` | `func(content, fieldName, fieldValue string) (string, error)` | Sets a field inside the `on:` trigger block |
| `RemoveFieldFromOnTrigger` | `func(content, fieldName string) (string, error)` | Removes a field from the `on:` trigger block |
| `UpdateScheduleInOnBlock` | `func(content, scheduleExpr string) (string, error)` | Updates the cron schedule in the `on:` block |
| `ScanWorkflowsForMCP` | `func(workflowsDir, serverFilter string, verbose bool) ([]WorkflowMCPMetadata, error)` | Scans all workflows for MCP server configurations |
| `ListToolsForMCP` | `func(workflowFile, mcpServerName string, verbose bool) error` | Lists tools for a specific MCP server in a workflow |
| `CollectLockFileManifests` | `func(workflowsDir string) map[string]*workflow.GHAWManifest` | Reads all `*.lock.yml` manifests from a directory |
| `WritePriorManifestFile` | `func(map[string]*workflow.GHAWManifest) (string, error)` | Writes manifest cache to a temporary file |
| `GroupRunsByWorkflow` | `func([]WorkflowRun) map[string][]WorkflowRun` | Groups a flat slice of runs by workflow name |
| `WaitForWorkflowCompletion` | `func(ctx, repoSlug, runID string, timeoutMinutes int, verbose bool) error` | Polls until a workflow run finishes or times out |
| `ValidArtifactSetNames` | `func() []string` | Returns the valid artifact set name strings |
| `ResolveArtifactFilter` | `func([]string) []string` | Expands artifact set aliases to concrete artifact names |
| `ValidateArtifactSets` | `func([]string) error` | Validates that all provided artifact set names are known |
| `ParseCopilotCodingAgentLogMetrics` | `func(logContent string, verbose bool) workflow.LogMetrics` | Parses Copilot coding-agent logs into metrics |
| `ExtractLogMetricsFromRun` | `func(ProcessedRun) workflow.LogMetrics` | Extracts log metrics from a processed run |
| `TrainDrain3Weights` | `func([]ProcessedRun, outputDir string, verbose bool) error` | Trains Drain3 anomaly-detection weights from run history |
| `DisplayOutdatedDependencies` | `func([]OutdatedDependency, int)` | Renders an outdated-dependencies table to stdout |
| `DisplayDependencyReport` | `func(*DependencyReport)` | Renders a full dependency report to stdout |
| `DisplayDependencyReportJSON` | `func(*DependencyReport) error` | Renders a dependency report as JSON to stdout |
| `DisplaySecurityAdvisories` | `func([]SecurityAdvisory)` | Renders a security-advisory table to stdout |
| `IsDockerAvailable` | `func() bool` | Returns true if the Docker daemon is reachable |
| `IsDockerImageAvailable` | `func(string) bool` | Returns true if a Docker image is present locally |
| `IsDockerImageDownloading` | `func(string) bool` | Returns true if an image pull is in progress |
| `StartDockerImageDownload` | `func(ctx, image string) bool` | Begins a background image pull; returns false if already pulling |
| `CheckAndPrepareDockerImages` | `func(ctx, useZizmor, usePoutine, useActionlint, useRunnerGuard bool) error` | Pre-pulls security-scanner Docker images |
| `UpdateContainerPins` | `func(ctx, workflowDir string, verbose bool) error` | Updates container image SHA pins in workflow files |
| `CreatePRWithChanges` | `func(branchPrefix, commitMessage, prTitle, prBody string, verbose bool) (string, error)` | Creates a GitHub PR from uncommitted changes |
| `AutoMergePullRequestsCreatedAfter` | `func(repoSlug string, createdAfter time.Time, verbose bool) error` | Auto-merges eligible PRs created after a given time |
| `PreflightCheckForCreatePR` | `func(bool) error` | Validates prerequisites before creating a PR |
| `DisableAllWorkflowsExcept` | `func(repoSlug string, exceptWorkflows []string, verbose bool) error` | Disables all workflows in a repo except the named ones |
| `GetEngineSecretNameAndValue` | `func(engine string, existingSecrets map[string]bool) (string, string, bool, error)` | Prompts for and validates an engine API secret |
| `CheckForUpdatesAsync` | `func(ctx, noCheckUpdate, verbose bool)` | Checks for a newer `gh-aw` version in the background |
| `FetchChecksResult` | `func(repoOverride, prNumber string) (*ChecksResult, error)` | Fetches CI check results for a pull request |
| `ValidEngineNames` | `func() []string` | Returns the supported engine names for shell completion |
| `CompleteWorkflowNames` | `func(*cobra.Command, []string, string) ([]string, cobra.ShellCompDirective)` | Shell-completion provider for workflow names |
| `CompleteEngineNames` | `func(*cobra.Command, []string, string) ([]string, cobra.ShellCompDirective)` | Shell-completion provider for engine names |
| `CompleteDirectories` | `func(*cobra.Command, []string, string) ([]string, cobra.ShellCompDirective)` | Shell-completion provider for directory paths |
| `RegisterEngineFlagCompletion` | `func(*cobra.Command)` | Registers shell completions for the `--engine` flag |
| `RegisterDirFlagCompletion` | `func(*cobra.Command, string)` | Registers shell completions for a directory flag |
| `UninstallShellCompletion` | `func(verbose bool) error` | Uninstalls shell completion scripts |
| `IsCommitSHA` | `func(string) bool` | Returns true if the string is a full Git commit SHA |
| `ValidateWorkflowIntent` | `func(string) error` | Validates the workflow intent string |

### Additional Exported Types

The `cli` package exports many types used across its command implementations. The following supplements the main "Key Types" table above:

| Type | Kind | Description |
|------|------|-------------|
| `AccessLogEntry` | struct | A single entry from an AWF network access log |
| `AccessLogSummary` | struct | Aggregated summary of access log entries |
| `ActionInput` | struct | An input parameter definition from `action.yml` |
| `ActionMetadata` | struct | Parsed `action.yml` metadata for a composite action |
| `ActionOutput` | struct | An output definition from `action.yml` |
| `ActionlintStats` | struct | Static-analysis statistics from an actionlint run |
| `AddInteractiveConfig` | struct | Configuration for the interactive `add-wizard` command |
| `AgenticAssessment` | struct | Agentic behavior assessment derived from audit logs |
| `ArtifactSet` | string alias | Named set of artifacts (e.g. `"agent"`, `"detection"`) |
| `AuditComparisonData` | struct | Full comparison between two audit runs |
| `AuditComparisonBaseline` | struct | Baseline metrics for an audit comparison |
| `AuditComparisonDelta` | struct | Numeric delta between baseline and compare run |
| `AuditEngineConfig` | struct | Engine configuration captured in an audit run |
| `AuditLogEntry` | struct | A structured entry from the agent audit log |
| `AwContext` | struct | Agentic workflow context parsed from the run |
| `AwInfo` | struct | Top-level `gh-aw` metadata block from an audit artifact |
| `BehaviorFingerprint` | struct | Pattern fingerprint of agent behavior across turns |
| `CheckState` | string alias | CI check state (`"success"`, `"failure"`, `"pending"`, ...) |
| `CodemodResult` | struct | Result of a single codemod transformation |
| `CompilationStats` | struct | Statistics from a compilation run (files, errors, warnings) |
| `CompileValidationError` | struct | A validation error emitted during compilation |
| `CombinedTrialResult` | struct | Combined results from multiple trial runs |
| `ContinuationData` | struct | State for multi-turn agent continuations |
| `CopilotCodingAgentDetector` | struct | Detector for Copilot coding-agent log patterns |
| `CrossRunSummary` | struct | Summary of cross-run metrics across multiple workflow runs |
| `DependencyInfo` | struct | Metadata for a single dependency in `go.mod` or `package.json` |
| `DevcontainerConfig` | struct | Parsed `.devcontainer/devcontainer.json` configuration |
| `DomainAnalysis` | struct | Aggregated per-domain network request analysis |
| `DomainBuckets` | struct | Domain requests bucketed by category (allow, deny, unknown) |
| `DomainDiffEntry` | struct | Per-domain diff between two runs |
| `DownloadResult` | struct | Result of a log artifact download |
| `EpisodeData` | struct | A single agent episode (one tool-call turn) |
| `ErrorInfo` | struct | Structured error captured from an agent run |
| `ErrorSummary` | struct | Aggregated error summary for a workflow run |
| `FetchedWorkflow` | struct | A workflow fetched from a remote or local source with metadata |
| `FileInfo` | struct | File metadata captured during a workflow run |
| `Finding` | struct | A finding from a security scanner (Zizmor/Poutine/Actionlint) |
| `FirewallAnalysis` | struct | Analysis of AWF network firewall logs |
| `FirewallLogEntry` | struct | A single entry from the AWF firewall log |
| `GatewayLogEntry` | struct | A log entry from the MCP gateway proxy |
| `GatewayMetrics` | struct | Aggregate metrics from MCP gateway logs |
| `GitHubRateLimitEntry` | struct | A GitHub API rate-limit snapshot from the agent run |
| `GitHubWorkflow` | struct | Minimal GitHub Actions workflow metadata |
| `GuardPolicySummary` | struct | Summary of guard-policy evaluations during a run |
| `InitOptions` | struct | Options for `InitRepository` |
| `JobData` | struct | Data for a single GitHub Actions job |
| `JobInfo` | struct | Metadata for a GitHub Actions job |
| `ListWorkflowRunsOptions` | struct | Options for listing workflow runs |
| `LockFileStatus` | struct | Status of a compiled `.lock.yml` file |
| `LogsData` | struct | Full log data downloaded for a workflow run |
| `LogsSummary` | struct | Summary view of downloaded log data |
| `MCPConfig` | struct | MCP server configuration as parsed from a workflow |
| `MCPFailureReport` | struct | Report of MCP server failures during a run |
| `MCPLogsGuardrailResponse` | struct | Guardrail evaluation response from MCP log analysis |
| `MCPPackage` | struct | An npm/pip package entry used by an MCP server |
| `MCPRegistryServerForProcessing` | struct | Server entry retrieved from the MCP registry |
| `MCPServerHealth` | struct | Health metrics for a single MCP server |
| `MCPSlowestToolCall` | struct | The slowest tool call recorded for an MCP server |
| `MCPToolCall` | struct | A single MCP tool invocation from an agent turn |
| `MCPToolSummary` | struct | Aggregated MCP tool usage summary |
| `MCPToolUsageData` | struct | Per-tool usage counts and latencies |
| `MetricsData` | struct | Core performance metrics for a workflow run |
| `MetricsTrendData` | struct | Trend data for a metric across multiple runs |
| `ModelTokenUsage` | struct | Token usage for a single AI model |
| `NoopReport` | struct | Report for a noop safe-output event |
| `ObservabilityInsight` | struct | An insight derived from observability data |
| `OverviewData` | struct | High-level overview data for a workflow run |
| `PRCheckRun` | struct | A single CI check run attached to a pull request |
| `PRCommitStatus` | struct | A commit status context for a pull request |
| `PRInfo` | struct | Pull-request metadata used by `gh aw pr` commands |
| `PerformanceMetrics` | struct | Performance counters for a workflow run |
| `PolicyAnalysis` | struct | Analysis of guard-policy evaluation results |
| `PolicyManifest` | struct | A manifest of guard policies applied during a run |
| `ProcessedRun` | struct | A fully-processed workflow run with parsed artifacts |
| `ProjectConfig` | struct | Configuration for `gh aw project new` |
| `PromptAnalysis` | struct | Analysis of the prompt sent to the agent |
| `RPCMessageEntry` | struct | A single RPC message from MCP gateway logs |
| `Recommendation` | struct | An actionable recommendation derived from audit data |
| `RedactedDomainsAnalysis` | struct | Analysis of redacted domain entries in firewall logs |
| `Release` | struct | A GitHub release entry |
| `Remote` | struct | A Git remote |
| `RepoSpec` | struct | A parsed repository specifier (`owner/repo[@ref]`) |
| `Repository` | struct | A GitHub repository |
| `RuleHitStats` | struct | Statistics for a single AWF firewall rule |
| `RunData` | struct | All data collected for a single workflow run |
| `RunSummary` | struct | Summary of a workflow run |
| `SafeOutputSummary` | struct | Summary of safe-output events in a run |
| `SecretInfo` | struct | Metadata for a configured repository secret |
| `SecretRequirement` | struct | A required secret for a workflow |
| `SessionAnalysis` | struct | Analysis of agent session metadata |
| `ShellType` | string alias | Shell type detected by `DetectShell` (e.g. `"bash"`, `"zsh"`) |
| `SourceSpec` | struct | A parsed workflow source specifier (local, remote, or registry) |
| `TokenUsageEntry` | struct | Per-request token usage from the agent |
| `TokenUsageSummary` | struct | Aggregated token usage for a workflow run |
| `ToolTransition` | struct | A transition between tool calls in an agent episode |
| `ToolUsageInfo` | struct | Usage information for a single tool |
| `ToolUsageSummary` | struct | Aggregated tool usage statistics |
| `Transport` | struct | MCP server transport configuration |
| `TrendDirection` | int alias | Direction of a metric trend (`Up`, `Down`, `Stable`) |
| `TrialArtifacts` | struct | Artifacts generated during a trial run |
| `TrialRepoContext` | struct | Repository context used during a trial run |
| `VSCodeMCPServer` | struct | An MCP server entry in `.vscode/mcp.json` |
| `VSCodeSettings` | struct | Parsed `.vscode/settings.json` |
| `Workflow` | struct | Minimal workflow metadata used in list operations |
| `WorkflowDomainsDetail` | struct | Detailed per-workflow domain information |
| `WorkflowDomainsSummary` | struct | Summary of domains used across workflows |
| `WorkflowFailure` | struct | A workflow failure record |
| `WorkflowFileStatus` | struct | Status of a workflow file (exists, outdated, etc.) |
| `WorkflowJob` | struct | A GitHub Actions job within a workflow run |
| `WorkflowListItem` | struct | A single item in the `gh aw list` output |
| `WorkflowMCPMetadata` | struct | MCP server metadata scanned from a workflow file |
| `WorkflowNode` | struct | A node in the workflow dependency graph |
| `WorkflowOption` | struct | A selectable workflow option for interactive prompts |
| `WorkflowRunInfo` | struct | Summary of a workflow run from the GitHub API |
| `WorkflowSpec` | struct | A fully resolved workflow specification with source metadata |
| `WorkflowStats` | struct | Aggregate statistics for a workflow |
| `LogMetrics` | type alias | Alias for `workflow.LogMetrics` — log parsing metrics |
| `PostTransformFunc` | func type | A post-compilation transformation function |
| `LogParser[T]` | generic func type | Generic log-parser function type parameterized on analysis result |

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
