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
	// CopilotIntegrationIDFeatureFlag gates injection of the
	// GITHUB_COPILOT_INTEGRATION_ID environment variable into the agent step.
	// Default off — the env var may cause Copilot CLI failures.
	// See https://github.com/github/gh-aw/issues/25516
	//
	// Workflow frontmatter usage:
	//
	//	features:
	//	  copilot-integration-id: true
	CopilotIntegrationIDFeatureFlag FeatureFlag = "copilot-integration-id"
)
