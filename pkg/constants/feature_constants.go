package constants

// FeatureFlag represents a feature flag identifier.
// This semantic type distinguishes feature flag names from arbitrary strings,
// making feature flag operations explicit and type-safe.
//
// Example usage:
//
//	const MCPGatewayFeatureFlag FeatureFlag = "mcp-gateway"
//	func IsFeatureEnabled(flag FeatureFlag) bool { ... }
type FeatureFlag string

// Feature flag identifiers
const (
	// NoFeatureFlag indicates that no feature-flag-based override should be applied.
	NoFeatureFlag FeatureFlag = ""
	// MCPScriptsFeatureFlag is the name of the feature flag for mcp-scripts
	MCPScriptsFeatureFlag FeatureFlag = "mcp-scripts"
	// MCPGatewayFeatureFlag is the feature flag name for enabling MCP gateway
	MCPGatewayFeatureFlag FeatureFlag = "mcp-gateway"
	// DisableXPIAPromptFeatureFlag is the feature flag name for disabling XPIA prompt
	DisableXPIAPromptFeatureFlag FeatureFlag = "disable-xpia-prompt"
	// CopilotRequestsFeatureFlag is the feature flag name for enabling copilot-requests mode.
	// When enabled: no secret validation step is generated, copilot-requests: write permission is added,
	// and the GitHub Actions token is used as the agentic engine secret.
	CopilotRequestsFeatureFlag FeatureFlag = "copilot-requests"
	// DIFCProxyFeatureFlag is the deprecated feature flag name for the DIFC proxy.
	// Deprecated: Use tools.github.integrity-proxy instead. The proxy is now enabled
	// by default when guard policies are configured. Set tools.github.integrity-proxy: false
	// to disable it. The codemod "features-difc-proxy-to-tools-github" migrates this flag.
	DIFCProxyFeatureFlag FeatureFlag = "difc-proxy"
	// CliProxyFeatureFlag enables the AWF CLI proxy sidecar.
	// When enabled, the compiler starts a difc-proxy on the host before AWF and
	// injects --difc-proxy-host and --difc-proxy-ca-cert into the AWF command,
	// giving the agent secure gh CLI access without exposing GITHUB_TOKEN.
	// The token is held in an mcpg DIFC proxy on the host, enforcing
	// guard policies and audit logging.
	//
	// Workflow frontmatter usage:
	//
	//	features:
	//	  cli-proxy: true
	CliProxyFeatureFlag FeatureFlag = "cli-proxy"
	// AwfDiagnosticLogsFeatureFlag enables AWF operational Docker diagnostics
	// collection on failure. When enabled, AWF collects capped container logs,
	// container exit codes, mount metadata, and sanitized compose config into
	// the diagnostics subdirectory of the firewall audit artifact.
	//
	// Workflow frontmatter usage:
	//
	//	features:
	//	  awf-diagnostic-logs: true
	AwfDiagnosticLogsFeatureFlag FeatureFlag = "awf-diagnostic-logs"
	// ByokCopilotFeatureFlag enables Copilot CLI offline BYOK mode.
	// When enabled with engine: copilot, the compiler:
	//   - injects a dummy COPILOT_API_KEY into the agent env to trigger AWF BYOK runtime behavior
	//   - implicitly enables the cli-proxy feature
	//   - installs the latest Copilot CLI version (un-pinned)
	//
	// Workflow frontmatter usage:
	//
	//	features:
	//	  byok-copilot: true
	ByokCopilotFeatureFlag FeatureFlag = "byok-copilot"
	// CopilotVersionFeatureFlag overrides the default Copilot CLI version.
	// When set to a non-empty string, Copilot CLI installation uses this
	// version instead of DefaultCopilotVersion. Supports explicit versions
	// (e.g. "1.0.21") and the "latest" tag.
	//
	// Workflow frontmatter usage:
	//
	//	features:
	//	  copilot-version: "latest"
	CopilotVersionFeatureFlag FeatureFlag = "copilot-version"
	// MCPGVersionFeatureFlag overrides the default MCP gateway container version.
	// When set to a non-empty string, MCP gateway image references use this
	// version instead of DefaultMCPGatewayVersion.
	//
	// Workflow frontmatter usage:
	//
	//	features:
	//	  mcpg-version: "v0.2.27"
	MCPGVersionFeatureFlag FeatureFlag = "mcpg-version"
	// FirewallVersionFeatureFlag overrides the default firewall container version.
	// When set to a non-empty string, AWF installation and image references use
	// this version instead of DefaultFirewallVersion.
	//
	// Workflow frontmatter usage:
	//
	//	features:
	//	  firewall-version: "v0.25.27"
	FirewallVersionFeatureFlag FeatureFlag = "firewall-version"
	// CodexVersionFeatureFlag overrides the default Codex CLI version.
	// Supports explicit versions (e.g. "0.121.0") and the "latest" tag.
	//
	// Workflow frontmatter usage:
	//
	//	features:
	//	  codex-version: "latest"
	CodexVersionFeatureFlag FeatureFlag = "codex-version"
	// ClaudeVersionFeatureFlag overrides the default Claude CLI version.
	// Supports explicit versions (e.g. "2.1.47") and the "latest" tag.
	//
	// Workflow frontmatter usage:
	//
	//	features:
	//	  claude-version: "latest"
	ClaudeVersionFeatureFlag FeatureFlag = "claude-version"
	// OpenCodeVersionFeatureFlag overrides the default OpenCode CLI version.
	// Supports explicit versions and the "latest" tag.
	//
	// Workflow frontmatter usage:
	//
	//	features:
	//	  opencode-version: "latest"
	OpenCodeVersionFeatureFlag FeatureFlag = "opencode-version"
	// GeminiVersionFeatureFlag overrides the default Gemini CLI version.
	// Supports explicit versions and the "latest" tag.
	//
	// Workflow frontmatter usage:
	//
	//	features:
	//	  gemini-version: "latest"
	GeminiVersionFeatureFlag FeatureFlag = "gemini-version"
	// IntegrityReactionsFeatureFlag enables reaction-based integrity promotion/demotion
	// in the MCPG allow-only policy. When enabled, the compiler injects
	// endorsement-reactions and disapproval-reactions fields into the allow-only policy.
	// Requires MCPG >= v0.2.18.
	//
	// Workflow frontmatter usage:
	//
	//	features:
	//	  integrity-reactions: true
	IntegrityReactionsFeatureFlag FeatureFlag = "integrity-reactions"
	// MCPCLIFeatureFlag gates the MCP CLI mounting feature. When enabled together
	// with tools.mount-as-clis: true, MCP servers are exposed as standalone CLI
	// tools on PATH. Without this feature flag, the mount-as-clis setting is
	// ignored and code generation remains unchanged.
	//
	// safeoutputs and mcpscripts CLI mounting is also gated behind this flag —
	// they are only CLI-mounted when both the feature flag is enabled and the
	// respective tool is configured.
	//
	// Workflow frontmatter usage:
	//
	//	features:
	//	  mcp-cli: true
	MCPCLIFeatureFlag FeatureFlag = "mcp-cli"
)
