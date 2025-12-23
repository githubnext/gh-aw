package constants

import (
	"path/filepath"
	"time"
)

// CLIExtensionPrefix is the prefix used in user-facing output to refer to the CLI extension
const CLIExtensionPrefix = "gh aw"

// Semantic types for measurements and identifiers
//
// These type aliases provide meaningful names for primitive types, improving code clarity
// and type safety. They follow the semantic type alias pattern where the type name
// indicates both what the value represents and how it should be used.
//
// Benefits of semantic type aliases:
//   - Self-documenting: The type name explains the purpose
//   - Type safety: Prevents mixing different concepts with the same underlying type
//   - Clear intent: Signals to readers what the value represents
//   - Easy refactoring: Can change implementation without affecting API
//
// See specs/go-type-patterns.md for detailed guidance on type patterns.

// LineLength represents a line length in characters for expression formatting.
// This semantic type distinguishes line lengths from arbitrary integers,
// making formatting code more readable and preventing accidental misuse.
//
// Example usage:
//
//	if len(expression) > int(constants.MaxExpressionLineLength) {
//	    // Break into multiple lines
//	}
type LineLength int

// Version represents a software version string.
// This semantic type distinguishes version strings from arbitrary strings,
// enabling future validation logic (e.g., semver parsing) and making
// version requirements explicit in function signatures.
//
// Example usage:
//
//	const DefaultCopilotVersion Version = "0.0.369"
//	func InstallTool(name string, version Version) error { ... }
type Version string

// FeatureFlag represents a feature flag identifier
type FeatureFlag string

// MaxExpressionLineLength is the maximum length for a single line expression before breaking into multiline
const MaxExpressionLineLength LineLength = 120

// ExpressionBreakThreshold is the threshold for breaking long lines at logical points
const ExpressionBreakThreshold LineLength = 100

// DefaultMCPRegistryURL is the default MCP registry URL
const DefaultMCPRegistryURL = "https://api.mcp.github.com/v0"

// DefaultClaudeCodeVersion is the default version of the Claude Code CLI
const DefaultClaudeCodeVersion Version = "2.0.75"

// DefaultCopilotVersion is the default version of the GitHub Copilot CLI
// WARNING: UPGRADING COPILOT CLI REQUIRES A FULL INTEGRATION TEST RUN TO ENSURE COMPATIBILITY
const DefaultCopilotVersion Version = "0.0.372"

// DefaultCopilotDetectionModel is the default model for the Copilot engine when used in the detection job
const DefaultCopilotDetectionModel = "gpt-5-mini"

// Environment variable names for model configuration
const (
	// EnvVarModelAgentCopilot configures the default Copilot model for agent execution
	EnvVarModelAgentCopilot = "GH_AW_MODEL_AGENT_COPILOT"
	// EnvVarModelAgentClaude configures the default Claude model for agent execution
	EnvVarModelAgentClaude = "GH_AW_MODEL_AGENT_CLAUDE"
	// EnvVarModelAgentCodex configures the default Codex model for agent execution
	EnvVarModelAgentCodex = "GH_AW_MODEL_AGENT_CODEX"
	// EnvVarModelDetectionCopilot configures the default Copilot model for detection
	EnvVarModelDetectionCopilot = "GH_AW_MODEL_DETECTION_COPILOT"
	// EnvVarModelDetectionClaude configures the default Claude model for detection
	EnvVarModelDetectionClaude = "GH_AW_MODEL_DETECTION_CLAUDE"
	// EnvVarModelDetectionCodex configures the default Codex model for detection
	EnvVarModelDetectionCodex = "GH_AW_MODEL_DETECTION_CODEX"
)

// DefaultCodexVersion is the default version of the OpenAI Codex CLI
const DefaultCodexVersion Version = "0.77.0"

// DefaultGitHubMCPServerVersion is the default version of the GitHub MCP server Docker image
const DefaultGitHubMCPServerVersion Version = "v0.26.3"

// DefaultFirewallVersion is the default version of the gh-aw-firewall (AWF) binary
const DefaultFirewallVersion Version = "v0.7.0"

// DefaultPlaywrightMCPVersion is the default version of the @playwright/mcp package
const DefaultPlaywrightMCPVersion Version = "0.0.53"

// DefaultPlaywrightBrowserVersion is the default version of the Playwright browser Docker image
const DefaultPlaywrightBrowserVersion Version = "v1.57.0"

// DefaultMCPSDKVersion is the default version of the @modelcontextprotocol/sdk package
const DefaultMCPSDKVersion Version = "1.24.0"

// DefaultGitHubScriptVersion is the default version of the actions/github-script action
const DefaultGitHubScriptVersion Version = "v8"

// DefaultBunVersion is the default version of Bun for runtime setup
const DefaultBunVersion Version = "1.1"

// DefaultNodeVersion is the default version of Node.js for runtime setup
const DefaultNodeVersion Version = "24"

// DefaultPythonVersion is the default version of Python for runtime setup
const DefaultPythonVersion Version = "3.12"

// DefaultRubyVersion is the default version of Ruby for runtime setup
const DefaultRubyVersion Version = "3.3"

// DefaultDotNetVersion is the default version of .NET for runtime setup
const DefaultDotNetVersion Version = "8.0"

// DefaultJavaVersion is the default version of Java for runtime setup
const DefaultJavaVersion Version = "21"

// DefaultElixirVersion is the default version of Elixir for runtime setup
const DefaultElixirVersion Version = "1.17"

// DefaultGoVersion is the default version of Go for runtime setup
const DefaultGoVersion Version = "1.25"

// DefaultHaskellVersion is the default version of GHC for runtime setup
const DefaultHaskellVersion Version = "9.10"

// DefaultDenoVersion is the default version of Deno for runtime setup
const DefaultDenoVersion Version = "2.x"

// Timeout constants using time.Duration for type safety and clear units

// DefaultAgenticWorkflowTimeout is the default timeout for agentic workflow execution
const DefaultAgenticWorkflowTimeout = 20 * time.Minute

// DefaultToolTimeout is the default timeout for tool/MCP server operations
const DefaultToolTimeout = 60 * time.Second

// DefaultMCPStartupTimeout is the default timeout for MCP server startup
const DefaultMCPStartupTimeout = 120 * time.Second

// Legacy timeout constants for backward compatibility (deprecated)
// These are kept for existing code that expects integer values
// TODO: Remove these after all call sites are migrated to use time.Duration

// DefaultAgenticWorkflowTimeoutMinutes is the default timeout for agentic workflow execution in minutes
// Deprecated: Use DefaultAgenticWorkflowTimeout instead
const DefaultAgenticWorkflowTimeoutMinutes = int(DefaultAgenticWorkflowTimeout / time.Minute)

// DefaultToolTimeoutSeconds is the default timeout for tool/MCP server operations in seconds
// Deprecated: Use DefaultToolTimeout instead
const DefaultToolTimeoutSeconds = int(DefaultToolTimeout / time.Second)

// DefaultMCPStartupTimeoutSeconds is the default timeout for MCP server startup in seconds
// Deprecated: Use DefaultMCPStartupTimeout instead
const DefaultMCPStartupTimeoutSeconds = int(DefaultMCPStartupTimeout / time.Second)

// DefaultActivationJobRunnerImage is the default runner image for activation and pre-activation jobs
// See https://github.blog/changelog/2025-10-28-1-vcpu-linux-runner-now-available-in-github-actions-in-public-preview/
const DefaultActivationJobRunnerImage = "ubuntu-slim"

// DefaultAllowedDomains defines the default localhost domains with port variations
// that are always allowed for Playwright browser automation
var DefaultAllowedDomains = []string{"localhost", "localhost:*", "127.0.0.1", "127.0.0.1:*"}

// SafeWorkflowEvents defines events that are considered safe and don't require permission checks
// workflow_run is intentionally excluded because it has HIGH security risks:
// - Privilege escalation (inherits permissions from triggering workflow)
// - Branch protection bypass (can execute on protected branches via unprotected branches)
// - Secret exposure (secrets available even when triggered by untrusted code)
var SafeWorkflowEvents = []string{"workflow_dispatch", "schedule"}

// AllowedExpressions contains the GitHub Actions expressions that can be used in workflow markdown content
// see https://docs.github.com/en/actions/reference/workflows-and-actions/contexts#github-context
var AllowedExpressions = []string{
	"github.event.after",
	"github.event.before",
	"github.event.check_run.id",
	"github.event.check_suite.id",
	"github.event.comment.id",
	"github.event.deployment.id",
	"github.event.deployment_status.id",
	"github.event.head_commit.id",
	"github.event.installation.id",
	"github.event.issue.number",
	"github.event.discussion.number",
	"github.event.pull_request.number",
	"github.event.milestone.number",
	"github.event.check_run.number",
	"github.event.check_suite.number",
	"github.event.workflow_job.run_id",
	"github.event.workflow_run.number",
	"github.event.label.id",
	"github.event.milestone.id",
	"github.event.organization.id",
	"github.event.page.id",
	"github.event.project.id",
	"github.event.project_card.id",
	"github.event.project_column.id",
	"github.event.release.assets[0].id",
	"github.event.release.id",
	"github.event.release.tag_name",
	"github.event.repository.id",
	"github.event.repository.default_branch",
	"github.event.review.id",
	"github.event.review_comment.id",
	"github.event.sender.id",
	"github.event.workflow_run.id",
	"github.event.workflow_run.conclusion",
	"github.event.workflow_run.html_url",
	"github.event.workflow_run.head_sha",
	"github.event.workflow_run.run_number",
	"github.event.workflow_run.event",
	"github.event.workflow_run.status",
	"github.event.issue.state",
	"github.event.issue.title",
	"github.event.pull_request.state",
	"github.event.pull_request.title",
	"github.event.discussion.title",
	"github.event.discussion.category.name",
	"github.event.release.name",
	"github.event.workflow_job.id",
	"github.event.deployment.environment",
	"github.event.pull_request.head.sha",
	"github.event.pull_request.base.sha",
	"github.actor",
	"github.job",
	"github.owner",
	"github.repository",
	"github.run_id",
	"github.run_number",
	"github.server_url",
	"github.workflow",
	"github.workspace",
} // needs., steps. already allowed

const AgentJobName = "agent"
const ActivationJobName = "activation"
const PreActivationJobName = "pre_activation"
const DetectionJobName = "detection"
const SafeOutputArtifactName = "safe_output.jsonl"
const AgentOutputArtifactName = "agent_output.json"

// SafeOutputsMCPServerID is the identifier for the safe-outputs MCP server
const SafeOutputsMCPServerID = "safeoutputs"

// SafeInputsMCPServerID is the identifier for the safe-inputs MCP server
const SafeInputsMCPServerID = "safeinputs"

// SafeInputsMCPVersion is the version of the safe-inputs MCP server
const SafeInputsMCPVersion = "1.0.0"

// Feature flag identifiers
const (
	// SafeInputsFeatureFlag is the name of the feature flag for safe-inputs
	SafeInputsFeatureFlag FeatureFlag = "safe-inputs"
	// SandboxRuntimeFeatureFlag is the feature flag name for sandbox runtime
	SandboxRuntimeFeatureFlag FeatureFlag = "sandbox-runtime"
)

// Step IDs for pre-activation job
const CheckMembershipStepID = "check_membership"
const CheckStopTimeStepID = "check_stop_time"
const CheckSkipIfMatchStepID = "check_skip_if_match"
const CheckCommandPositionStepID = "check_command_position"

// Output names for pre-activation job steps
const IsTeamMemberOutput = "is_team_member"
const StopTimeOkOutput = "stop_time_ok"
const SkipCheckOkOutput = "skip_check_ok"
const CommandPositionOkOutput = "command_position_ok"
const ActivatedOutput = "activated"

var AgenticEngines = []string{"claude", "codex", "copilot"}

// DefaultReadOnlyGitHubTools defines the default read-only GitHub MCP tools.
// This list is shared by both local (Docker) and remote (hosted) modes.
// Currently, both modes use identical tool lists, but this may diverge in the future
// if different modes require different default tool sets.
var DefaultReadOnlyGitHubTools = []string{
	// actions
	"download_workflow_run_artifact",
	"get_job_logs",
	"get_workflow_run",
	"get_workflow_run_logs",
	"get_workflow_run_usage",
	"list_workflow_jobs",
	"list_workflow_run_artifacts",
	"list_workflow_runs",
	"list_workflows",
	// code security
	"get_code_scanning_alert",
	"list_code_scanning_alerts",
	// context
	"get_me",
	// dependabot
	"get_dependabot_alert",
	"list_dependabot_alerts",
	// discussions
	"get_discussion",
	"get_discussion_comments",
	"list_discussion_categories",
	"list_discussions",
	// issues
	"issue_read",
	"list_issues",
	"search_issues",
	// notifications
	"get_notification_details",
	"list_notifications",
	// organizations
	"search_orgs",
	// labels
	"get_label",
	"list_label",
	// prs
	"get_pull_request",
	"get_pull_request_comments",
	"get_pull_request_diff",
	"get_pull_request_files",
	"get_pull_request_reviews",
	"get_pull_request_status",
	"list_pull_requests",
	"pull_request_read",
	"search_pull_requests",
	// repos
	"get_commit",
	"get_file_contents",
	"get_tag",
	"list_branches",
	"list_commits",
	"list_tags",
	"search_code",
	"search_repositories",
	// secret protection
	"get_secret_scanning_alert",
	"list_secret_scanning_alerts",
	// users
	"search_users",
	// additional unique tools (previously duplicated block extras)
	"get_latest_release",
	"get_pull_request_review_comments",
	"get_release_by_tag",
	"list_issue_types",
	"list_releases",
	"list_starred_repositories",
}

// DefaultGitHubToolsLocal defines the default read-only GitHub MCP tools for local (Docker) mode.
// Currently identical to DefaultReadOnlyGitHubTools. Kept separate for backward compatibility
// and to allow future divergence if local mode requires different defaults.
var DefaultGitHubToolsLocal = DefaultReadOnlyGitHubTools

// DefaultGitHubToolsRemote defines the default read-only GitHub MCP tools for remote (hosted) mode.
// Currently identical to DefaultReadOnlyGitHubTools. Kept separate for backward compatibility
// and to allow future divergence if remote mode requires different defaults.
var DefaultGitHubToolsRemote = DefaultReadOnlyGitHubTools

// DefaultGitHubTools is deprecated. Use DefaultGitHubToolsLocal or DefaultGitHubToolsRemote instead.
// Kept for backward compatibility and defaults to local mode tools.
var DefaultGitHubTools = DefaultGitHubToolsLocal

// DefaultBashTools defines basic bash commands that should be available by default when bash is enabled
var DefaultBashTools = []string{
	"echo",
	"ls",
	"pwd",
	"cat",
	"head",
	"tail",
	"grep",
	"wc",
	"sort",
	"uniq",
	"date",
	"yq",
}

// PriorityStepFields defines the conventional field order for GitHub Actions workflow steps
// Fields appear in this order first, followed by remaining fields alphabetically
var PriorityStepFields = []string{"name", "id", "if", "run", "uses", "script", "env", "with"}

// PriorityJobFields defines the conventional field order for GitHub Actions workflow jobs
// Fields appear in this order first, followed by remaining fields alphabetically
var PriorityJobFields = []string{"name", "runs-on", "needs", "if", "permissions", "environment", "concurrency", "outputs", "env", "steps"}

// PriorityWorkflowFields defines the conventional field order for top-level GitHub Actions workflow frontmatter
// Fields appear in this order first, followed by remaining fields alphabetically
var PriorityWorkflowFields = []string{"on", "permissions", "if", "network", "imports", "safe-outputs", "steps"}

// IgnoredFrontmatterFields are fields that should be silently ignored during frontmatter validation
// NOTE: This is now empty as description and applyTo are properly validated by the schema
var IgnoredFrontmatterFields = []string{}

func GetWorkflowDir() string {
	return filepath.Join(".github", "workflows")
}
