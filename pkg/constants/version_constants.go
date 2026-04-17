package constants

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

// String returns the string representation of the version
func (v Version) String() string {
	return string(v)
}

// IsValid returns true if the version is non-empty
func (v Version) IsValid() bool {
	return len(v) > 0
}

// ModelName represents an AI model name identifier.
// This semantic type distinguishes model names from arbitrary strings,
// making model selection explicit in function signatures.
//
// Example usage:
//
//	const DefaultCopilotDetectionModel ModelName = "gpt-5-mini"
//	func ExecuteWithModel(model ModelName) error { ... }
type ModelName string

// DefaultClaudeCodeVersion is the default version of the Claude Code CLI.
const DefaultClaudeCodeVersion Version = "2.1.98"

// DefaultCopilotVersion is the default version of the GitHub Copilot CLI.
//
// When unpinning or upgrading this version, verify:
//   - MCPs are not blocked from loading (tools.mcp configuration still works end-to-end)
//   - /models does not silently fail on PATs (check that model listing works with PAT auth)
const DefaultCopilotVersion Version = "1.0.21"

// DefaultCodexVersion is the default version of the OpenAI Codex CLI
const DefaultCodexVersion Version = "0.118.0"

// DefaultGeminiVersion is the default version of the Google Gemini CLI
const DefaultGeminiVersion Version = "0.37.2"

// DefaultOpenCodeVersion is the default version of the OpenCode CLI
const DefaultOpenCodeVersion Version = "1.2.14"

// DefaultGitHubMCPServerVersion is the default version of the GitHub MCP server Docker image
const DefaultGitHubMCPServerVersion Version = "v0.32.0"

// DefaultFirewallVersion is the default version of the gh-aw-firewall (AWF) binary
const DefaultFirewallVersion Version = "v0.25.22"

// AWFExcludeEnvMinVersion is the minimum AWF version that supports the --exclude-env flag.
// Workflows pinning an older AWF version must not emit --exclude-env flags or the run will fail.
const AWFExcludeEnvMinVersion Version = "v0.25.3"

// AWFCliProxyMinVersion is the minimum supported AWF version for emitting the CLI proxy flags
// (--difc-proxy-host, --difc-proxy-ca-cert). Workflows pinning an older AWF version than
// v0.25.17 must not emit CLI proxy flags or the run will fail.
const AWFCliProxyMinVersion Version = "v0.25.17"

// CopilotNoAskUserMinVersion is the minimum Copilot CLI version that supports the --no-ask-user
// flag, which enables fully autonomous agentic runs by suppressing interactive prompts.
// Workflows using an older Copilot CLI version must not emit --no-ask-user or the run will fail.
const CopilotNoAskUserMinVersion Version = "1.0.19"

// DefaultMCPGatewayVersion is the default version of the MCP Gateway (gh-aw-mcpg) Docker image
const DefaultMCPGatewayVersion Version = "v0.2.22"

// MCPGIntegrityReactionsMinVersion is the minimum MCPG version that supports
// endorsement-reactions and disapproval-reactions in the allow-only policy.
const MCPGIntegrityReactionsMinVersion Version = "v0.2.18"

// DefaultPlaywrightMCPVersion is the default version of the @playwright/mcp package
const DefaultPlaywrightMCPVersion Version = "0.0.70"

// DefaultPlaywrightBrowserVersion is the default version of the Playwright browser Docker image
const DefaultPlaywrightBrowserVersion Version = "v1.59.1"

// DefaultMCPSDKVersion is the default version of the @modelcontextprotocol/sdk package
const DefaultMCPSDKVersion Version = "1.24.0"

// DefaultGitHubScriptVersion is the default version of the actions/github-script action
const DefaultGitHubScriptVersion Version = "v9"

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
