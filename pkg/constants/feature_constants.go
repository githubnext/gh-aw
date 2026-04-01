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
	// DIFCProxyFeatureFlag is the feature flag name for enabling the DIFC proxy.
	// When enabled, the compiler injects DIFC proxy steps (start/stop) around pre-agent
	// gh CLI steps and qmd indexing steps when guard policies are configured.
	// By default (flag absent), DIFC proxy steps are not emitted.
	DIFCProxyFeatureFlag FeatureFlag = "difc-proxy"
)
