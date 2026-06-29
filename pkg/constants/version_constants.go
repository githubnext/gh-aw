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
	return v != ""
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
const DefaultClaudeCodeVersion Version = "2.1.195"

// DefaultCopilotVersion is the default version of the GitHub Copilot CLI.
//
// When unpinning or upgrading this version, verify:
//   - MCPs are not blocked from loading (tools.mcp configuration still works end-to-end)
//   - /models does not silently fail on PATs (check that model listing works with PAT auth)
const DefaultCopilotVersion Version = "1.0.65"

// DefaultCopilotSDKVersion is the default version of the @github/copilot-sdk package.
const DefaultCopilotSDKVersion Version = "1.0.4"

// DefaultCodexVersion is the default version of the OpenAI Codex CLI
const DefaultCodexVersion Version = "0.142.3"

// DefaultGeminiVersion is the default version of the Google Gemini CLI
const DefaultGeminiVersion Version = "0.39.1"

// DefaultAntigravityVersion is the default version of the Antigravity CLI
const DefaultAntigravityVersion Version = "1.0.2-6113393518706688"

// DefaultCrushVersion is the default version of the Crush CLI
const DefaultCrushVersion Version = "0.59.0"

// DefaultPiVersion is the default version of the Pi CLI
const DefaultPiVersion Version = "0.80.2"

// DefaultOpenCodeVersion is the default version of the OpenCode CLI
const DefaultOpenCodeVersion Version = "1.2.14"

// DefaultGitHubMCPServerVersion is the default version of the GitHub MCP server Docker image
const DefaultGitHubMCPServerVersion Version = "v1.4.0"

// DefaultFirewallVersion is the default version of the gh-aw-firewall (AWF) binary
//
// ⚠️  IMPORTANT: When updating this version, you must run a full rebuild and recompile twice:
//
//	make build && make recompile && make recompile
//
// The first recompile regenerates all lock files using the new version; the second recompile
// refreshes the container SHA pins that were resolved during the first pass.
const DefaultFirewallVersion Version = "v0.27.13"

// AWFExcludeEnvMinVersion is the minimum AWF version that supports the --exclude-env flag.
// Workflows pinning an older AWF version must not emit --exclude-env flags or the run will fail.
const AWFExcludeEnvMinVersion Version = "v0.25.3"

// AWFCliProxyMinVersion is the minimum supported AWF version for emitting the CLI proxy flags
// (--difc-proxy-host, --difc-proxy-ca-cert). Workflows pinning an older AWF version than
// v0.25.17 must not emit CLI proxy flags or the run will fail.
const AWFCliProxyMinVersion Version = "v0.25.17"

// AWFAllowHostPortsMinVersion is the minimum AWF version that supports the
// --allow-host-ports flag. Workflows pinning an older AWF version must not emit
// --allow-host-ports or the run will fail at startup with an unknown flag error.
const AWFAllowHostPortsMinVersion Version = "v0.25.24"

// AWFDockerHostPathPrefixMinVersion is the minimum AWF version that supports the
// --docker-host-path-prefix flag used for ARC/DinD split runner/daemon filesystems.
// Workflows pinning an older AWF version must not emit this flag.
const AWFDockerHostPathPrefixMinVersion Version = "v0.25.43"

// AWFTokenSteeringMinVersion is the minimum AWF version that supports
// apiProxy.enableTokenSteering (mapped from frontmatter firewall.effective-token-steering).
const AWFTokenSteeringMinVersion Version = "v0.25.44"

// AWFChrootConfigMinVersion is the minimum AWF version that supports
// chroot.binariesSourcePath and chroot.identity.* in the config file.
// These fields let AWF handle binary staging and identity resolution natively
// for ARC/DinD split runner/daemon filesystem topologies, removing the need
// for bootstrap actions that manually copy binaries and pre-seed /etc/passwd.
const AWFChrootConfigMinVersion Version = "v0.27.1"

// CopilotNoAskUserMinVersion is the minimum Copilot CLI version that supports the --no-ask-user
// flag, which enables fully autonomous agentic runs by suppressing interactive prompts.
// Workflows using an older Copilot CLI version must not emit --no-ask-user or the run will fail.
const CopilotNoAskUserMinVersion Version = "1.0.19"

// DefaultMCPGatewayVersion is the default version of the MCP Gateway (gh-aw-mcpg) Docker image
//
// ⚠️  IMPORTANT: When updating this version, you must run a full rebuild and recompile twice:
//
//	make build && make recompile && make recompile
//
// The first recompile regenerates all lock files using the new version; the second recompile
// refreshes the container SHA pins that were resolved during the first pass.
const DefaultMCPGatewayVersion Version = "v0.3.32"

// MCPGIntegrityReactionsMinVersion is the minimum MCPG version that supports
// endorsement-reactions and disapproval-reactions in the allow-only policy.
const MCPGIntegrityReactionsMinVersion Version = "v0.2.18"

// DefaultPlaywrightMCPVersion is the default version of the @playwright/mcp package
const DefaultPlaywrightMCPVersion Version = "0.0.76"

// DefaultPlaywrightCLIVersion is the default version of the @playwright/cli package
// Used when tools.playwright.mode is "cli" to install the CLI tool instead of the MCP server.
const DefaultPlaywrightCLIVersion Version = "0.1.14"

// DefaultPlaywrightBrowserVersion is the default version of the Playwright browser Docker image
const DefaultPlaywrightBrowserVersion Version = "v1.61.1"

// DefaultMCPSDKVersion is the default version of the @modelcontextprotocol/sdk package
const DefaultMCPSDKVersion Version = "1.24.0"

// DefaultGitHubScriptVersion is the default version of the actions/github-script action
const DefaultGitHubScriptVersion Version = "v9"

// DefaultThreatDetectVersion is the default version of the gh-aw-threat-detection binary.
// This version is downloaded from GitHub Releases when `features: gh-aw-detection: true`
// is set in the workflow frontmatter, enabling the external threat-detect binary path instead
// of the inline engine execution path.
const DefaultThreatDetectVersion Version = "v0.2.2"

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
const DefaultGoVersion Version = "1.26"

// DefaultHaskellVersion is the default version of GHC for runtime setup
const DefaultHaskellVersion Version = "9.10"

// DefaultDenoVersion is the default version of Deno for runtime setup
const DefaultDenoVersion Version = "2.x"
