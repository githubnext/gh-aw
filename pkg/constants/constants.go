package constants

import "path/filepath"

// CLIExtensionPrefix is the prefix used in user-facing output to refer to the CLI extension
const CLIExtensionPrefix = "gh aw"

// MaxExpressionLineLength is the maximum length for a single line expression before breaking into multiline
const MaxExpressionLineLength = 120

// ExpressionBreakThreshold is the threshold for breaking long lines at logical points
const ExpressionBreakThreshold = 100

// DefaultMCPRegistryURL is the default MCP registry URL
const DefaultMCPRegistryURL = "https://api.mcp.github.com/v0"

// DefaultClaudeCodeVersion is the default version of the Claude Code CLI
const DefaultClaudeCodeVersion = "2.0.11"

// DefaultCopilotVersion is the default version of the GitHub Copilot CLI
const DefaultCopilotVersion = "0.0.337"

// DefaultCodexVersion is the default version of the OpenAI Codex CLI
const DefaultCodexVersion = "0.46.0"

// DefaultNodeVersion is the default version of Node.js for runtime setup
const DefaultNodeVersion = "24"

// DefaultPythonVersion is the default version of Python for runtime setup
const DefaultPythonVersion = "3.12"

// DefaultRubyVersion is the default version of Ruby for runtime setup
const DefaultRubyVersion = "3.3"

// DefaultDotNetVersion is the default version of .NET for runtime setup
const DefaultDotNetVersion = "8.0"

// DefaultJavaVersion is the default version of Java for runtime setup
const DefaultJavaVersion = "21"

// DefaultElixirVersion is the default version of Elixir for runtime setup
const DefaultElixirVersion = "1.17"

// DefaultHaskellVersion is the default version of GHC for runtime setup
const DefaultHaskellVersion = "9.10"

// DefaultAgenticWorkflowTimeoutMinutes is the default timeout for agentic workflow execution in minutes
const DefaultAgenticWorkflowTimeoutMinutes = 20

// DefaultAllowedDomains defines the default localhost domains with port variations
// that are always allowed for Playwright browser automation
var DefaultAllowedDomains = []string{"localhost", "localhost:*", "127.0.0.1", "127.0.0.1:*"}

// SafeWorkflowEvents defines events that are considered safe and don't require permission checks
var SafeWorkflowEvents = []string{"workflow_dispatch", "workflow_run", "schedule"}

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
	"github.event.label.id",
	"github.event.milestone.id",
	"github.event.organization.id",
	"github.event.page.id",
	"github.event.project.id",
	"github.event.project_card.id",
	"github.event.project_column.id",
	"github.event.pull_request.number",
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
const DetectionJobName = "detection"
const SafeOutputArtifactName = "safe_output.jsonl"
const AgentOutputArtifactName = "agent_output.json"

var AgenticEngines = []string{"claude", "codex", "copilot"}

// DefaultGitHubTools defines the default read-only GitHub MCP tools
var DefaultGitHubTools = []string{
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
	"get_issue",
	"get_issue_comments",
	"list_issues",
	"search_issues",
	// notifications
	"get_notification_details",
	"list_notifications",
	// organizations
	"search_orgs",
	// prs
	"get_pull_request",
	"get_pull_request_comments",
	"get_pull_request_diff",
	"get_pull_request_files",
	"get_pull_request_reviews",
	"get_pull_request_status",
	"list_pull_requests",
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
	"list_sub_issues",
}

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
}

// PriorityStepFields defines the conventional field order for GitHub Actions workflow steps
// Fields appear in this order first, followed by remaining fields alphabetically
var PriorityStepFields = []string{"name", "id", "if", "run", "uses", "script", "env", "with"}

// PriorityJobFields defines the conventional field order for GitHub Actions workflow jobs
// Fields appear in this order first, followed by remaining fields alphabetically
var PriorityJobFields = []string{"name", "runs-on", "needs", "if", "permissions", "environment", "concurrency", "outputs", "env", "defaults", "steps"}

// PriorityWorkflowFields defines the conventional field order for top-level GitHub Actions workflow frontmatter
// Fields appear in this order first, followed by remaining fields alphabetically
var PriorityWorkflowFields = []string{"on", "permissions", "if", "network", "imports", "safe-outputs", "steps"}

func GetWorkflowDir() string {
	return filepath.Join(".github", "workflows")
}
