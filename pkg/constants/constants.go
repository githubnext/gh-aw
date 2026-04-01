package constants

import (
	"path/filepath"
	"time"
)

// CLIExtensionPrefix is the prefix used in user-facing output to refer to the CLI extension.
const CLIExtensionPrefix CommandPrefix = "gh aw"

// Semantic types for measurements and identifiers
//
// These type aliases provide meaningful names for primitive types, improving code clarity
// and type safety. They follow the semantic type alias pattern where the type name
// indicates both what the value represents and how it should be used.
//
// # Intentional Method Duplication
//
// Several string-based types below define identical String() and IsValid() method bodies.
// This duplication is intentional: Go does not allow shared method sets for distinct named
// types, so each type must define its own methods. The bodies are deliberately simple and
// unlikely to diverge.

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

// CommandPrefix represents a CLI command prefix.
// This semantic type distinguishes command prefixes from arbitrary strings,
// making command-related operations explicit.
//
// Example usage:
//
//	const CLIExtensionPrefix CommandPrefix = "gh aw"
//	func FormatCommand(prefix CommandPrefix, cmd string) string { ... }
type CommandPrefix string

// String returns the string representation of the command prefix
func (c CommandPrefix) String() string {
	return string(c)
}

// IsValid returns true if the command prefix is non-empty
func (c CommandPrefix) IsValid() bool {
	return len(c) > 0
}

// WorkflowID represents a workflow identifier (basename without .md extension).
// This semantic type distinguishes workflow identifiers from arbitrary strings,
// preventing mixing of workflow IDs with other string types like file paths.
//
// Example usage:
//
//	func GetWorkflow(id WorkflowID) (*Workflow, error) { ... }
//	func CompileWorkflow(id WorkflowID) error { ... }
type WorkflowID string

// MaxExpressionLineLength is the maximum length for a single line expression before breaking into multiline.
const MaxExpressionLineLength LineLength = 120

// ExpressionBreakThreshold is the threshold for breaking long lines at logical points.
const ExpressionBreakThreshold LineLength = 100

// Network port constants
//
// These constants define standard network port values used throughout the codebase
// for MCP servers, gateway services, and validation ranges.

const (
	// DefaultMCPGatewayPort is the default port for the MCP gateway HTTP service
	DefaultMCPGatewayPort = 80

	// DefaultMCPServerPort is the default port for MCP servers (mcp-scripts server)
	DefaultMCPServerPort = 3000

	// DefaultMCPInspectorPort is the default port for the MCP inspector (safe-outputs server)
	DefaultMCPInspectorPort = 3001

	// DefaultQmdMCPPort is the TCP port for the qmd HTTP MCP server started in the agent job.
	// qmd runs as a Docker container (node:24) with `qmd mcp --http --port PORT`; using HTTP
	// transport avoids node-llama-cpp's direct process.stdout writes (dot-progress during model
	// loading) from corrupting the stdio JSON-RPC stream.
	DefaultQmdMCPPort = 8181

	// MinNetworkPort is the minimum valid network port number
	MinNetworkPort = 1

	// MaxNetworkPort is the maximum valid network port number
	MaxNetworkPort = 65535

	// ClaudeLLMGatewayPort is the port for the Claude LLM gateway
	ClaudeLLMGatewayPort = 10000

	// CodexLLMGatewayPort is the port for the Codex LLM gateway
	CodexLLMGatewayPort = 10001

	// CopilotLLMGatewayPort is the port for the Copilot LLM gateway
	CopilotLLMGatewayPort = 10002

	// GeminiLLMGatewayPort is the port for the Gemini LLM gateway
	GeminiLLMGatewayPort = 10003
)

// DefaultGitHubLockdown is the default value for the GitHub MCP server lockdown setting.
// Lockdown mode restricts the GitHub MCP server to the triggering repository only.
// Defaults to false (lockdown disabled).
const DefaultGitHubLockdown = false

// AWF (Agentic Workflow Firewall) constants

// AWFDefaultCommand is the default AWF command prefix
const AWFDefaultCommand = "sudo -E awf"

// AWFProxyLogsDir is the default directory for AWF proxy logs
const AWFProxyLogsDir = "/tmp/gh-aw/sandbox/firewall/logs"

// AWFAuditDir is the directory for AWF audit files (policy-manifest.json, squid.conf, docker-compose.redacted.yml).
// These files are written by AWF when --audit-dir is specified and provide structured policy/configuration data
// needed by the `awf logs audit` command for enriching log entries with policy rule matching.
const AWFAuditDir = "/tmp/gh-aw/sandbox/firewall/audit"

// FirewallAuditArtifactName is the artifact name used for dedicated firewall audit log uploads.
// All compiled agentic workflows upload the AWF structured audit/observability logs under this name.
const FirewallAuditArtifactName = "firewall-audit-logs"

// AWFDefaultLogLevel is the default log level for AWF
const AWFDefaultLogLevel = "info"

// DefaultMCPGatewayContainer is the default container image for the MCP Gateway
const DefaultMCPGatewayContainer = "ghcr.io/github/gh-aw-mcpg"

// DefaultMCPGatewayPayloadDir is the default directory for MCP gateway payload files
// This directory is shared between the agent container and MCP gateway for large payload exchange
const DefaultMCPGatewayPayloadDir = "/tmp/gh-aw/mcp-payloads"

// DefaultMCPGatewayPayloadSizeThreshold is the default size threshold (in bytes) for storing payloads to disk.
// Payloads larger than this threshold are stored to disk, smaller ones are returned inline.
// Default: 524288 bytes (512KB) - chosen to accommodate typical MCP tool responses including
// GitHub API queries (list_commits, list_issues, etc.) without triggering disk storage.
// This prevents agent looping issues when payloadPath is not accessible in agent containers.
const DefaultMCPGatewayPayloadSizeThreshold = 524288

// DefaultFirewallRegistry is the container image registry for AWF (gh-aw-firewall) Docker images
const DefaultFirewallRegistry = "ghcr.io/github/gh-aw-firewall"

// DefaultQmdIndexingRunnerImage is the default runner image for the qmd indexing job.
// Users can override this with the runs-on: field in the qmd config.
const DefaultQmdIndexingRunnerImage = "ubuntu-latest"

// DefaultNodeAlpineLTSImage is the default Node.js Alpine LTS container image for MCP servers
// Using node:lts-alpine provides the latest LTS version with minimal footprint
const DefaultNodeAlpineLTSImage = "node:lts-alpine"

// DefaultPythonAlpineLTSImage is the default Python Alpine LTS container image for MCP servers
// Using python:alpine provides the latest stable version with minimal footprint
const DefaultPythonAlpineLTSImage = "python:alpine"

// DefaultAlpineImage is the default minimal Alpine container image for running Go binaries
// Used for MCP servers that run statically-linked Go binaries like gh-aw mcp-server
const DefaultAlpineImage = "alpine:latest"

// DevModeGhAwImage is the Docker image tag for locally built gh-aw container in dev mode
// This image is built during workflow execution and includes the gh-aw binary and dependencies
const DevModeGhAwImage = "localhost/gh-aw:dev"

// GhAwRootDir is the base directory for gh-aw files on the runner.
// Uses ${{ runner.temp }} for compatibility with self-hosted runners that may not
// have write access to /opt/gh-aw/. The expression is resolved by GitHub Actions
// at workflow runtime before any step execution.
// Use this in YAML `with:` fields, `env:` value declarations, and Docker mounts
// where GitHub Actions template expressions are needed.
const GhAwRootDir = "${{ runner.temp }}/gh-aw"

// GhAwRootDirShell is the same path as GhAwRootDir but using the shell environment
// variable $RUNNER_TEMP instead of the GitHub Actions expression ${{ runner.temp }}.
// Use this inside shell `run:` blocks where the env var is already available.
// This is shorter than the Actions expression and avoids expression-length issues.
const GhAwRootDirShell = "${RUNNER_TEMP}/gh-aw"

// DefaultGhAwMount is the mount path for the gh-aw directory in containerized MCP servers
// The gh-aw binary and supporting files are mounted read-only from the runner temp directory.
// Uses the shell env var form since mounts are resolved in a shell context.
const DefaultGhAwMount = GhAwRootDirShell + ":" + GhAwRootDirShell + ":ro"

// DefaultGhBinaryMount is the mount path for the gh CLI binary in containerized MCP servers
// The gh CLI is required for agentic-workflows MCP server to run gh commands
const DefaultGhBinaryMount = "/usr/bin/gh:/usr/bin/gh:ro"

// DefaultTmpGhAwMount is the mount path for temporary gh-aw files in containerized MCP servers
// Used for logs, cache, and other runtime data that needs read-write access
const DefaultTmpGhAwMount = "/tmp/gh-aw:/tmp/gh-aw:rw"

// DefaultWorkspaceMount is the mount path for the GitHub workspace directory in containerized MCP servers
// Security: Uses GITHUB_WORKSPACE environment variable instead of template expansion to prevent template injection
// The GITHUB_WORKSPACE environment variable is automatically set by GitHub Actions and passed to the MCP gateway
const DefaultWorkspaceMount = "\\${GITHUB_WORKSPACE}:\\${GITHUB_WORKSPACE}:rw"

// Timeout constants using time.Duration for type safety and clear units

// DefaultAgenticWorkflowTimeout is the default timeout for agentic workflow execution
const DefaultAgenticWorkflowTimeout = 20 * time.Minute

// DefaultToolTimeout is the default timeout for tool/MCP server operations
const DefaultToolTimeout = 60 * time.Second

// DefaultMCPStartupTimeout is the default timeout for MCP server startup
const DefaultMCPStartupTimeout = 120 * time.Second

// DefaultActivationJobRunnerImage is the default runner image for activation and pre-activation jobs
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
// NOTE: user-invokable is a GitHub Copilot custom agent field that is not part of the gh-aw schema
var IgnoredFrontmatterFields = []string{"user-invokable"}

// SharedWorkflowForbiddenFields lists fields that cannot be used in shared/included workflows.
// These fields are only allowed in main workflows (workflows with an 'on' trigger field).
//
// This list is maintained in constants.go to enable easy mining by agents and automated tools.
// The compiler enforces these restrictions at compile time with clear error messages.
//
// Forbidden fields fall into these categories:
//   - Workflow triggers: on (defines it as a main workflow)
//   - Workflow execution: command, run-name, runs-on, concurrency, if, timeout-minutes, timeout_minutes
//   - Workflow metadata: name, tracker-id, strict
//   - Workflow features: container, env, environment, sandbox, features
//   - Access control: roles, github-token
//
// All other fields defined in main_workflow_schema.json can be used in shared workflows
// and will be properly imported and merged when the shared workflow is imported.
var SharedWorkflowForbiddenFields = []string{
	"on",              // Trigger field - only for main workflows
	"command",         // Command for workflow execution
	"concurrency",     // Concurrency control
	"container",       // Container configuration
	"env",             // Environment variables
	"environment",     // Deployment environment
	"features",        // Feature flags
	"github-token",    // GitHub token configuration
	"if",              // Conditional execution
	"name",            // Workflow name
	"roles",           // Role requirements
	"run-name",        // Run display name
	"runs-on",         // Runner specification
	"sandbox",         // Sandbox configuration
	"strict",          // Strict mode
	"timeout-minutes", // Timeout in minutes
	"timeout_minutes", // Timeout in minutes (underscore variant)
	"tracker-id",      // Tracker ID
}

func GetWorkflowDir() string {
	return filepath.Join(".github", "workflows")
}

// MaxSymlinkDepth limits recursive symlink resolution when fetching remote files.
// The GitHub Contents API doesn't follow symlinks in path components, so gh-aw
// resolves them manually. This constant caps recursion to prevent infinite loops
// when symlinks chain to each other.
const MaxSymlinkDepth = 5

// DefaultAllowedMemoryExtensions is the default list of allowed file extensions for cache-memory and repo-memory storage.
// An empty slice means all file extensions are allowed. When this is empty, the validation step is not emitted.
var DefaultAllowedMemoryExtensions = []string{}
