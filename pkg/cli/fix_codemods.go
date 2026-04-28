package cli

import "github.com/github/gh-aw/pkg/logger"

var fixCodemodsLog = logger.New("cli:fix_codemods")

// Codemod represents a single code transformation that can be applied to workflow files
type Codemod struct {
	ID           string // Unique identifier for the codemod
	Name         string // Human-readable name
	Description  string // Description of what the codemod does
	IntroducedIn string // Version where this codemod was introduced
	Apply        func(content string, frontmatter map[string]any) (string, bool, error)
}

// CodemodResult represents the result of applying a codemod
type CodemodResult struct {
	Applied bool   // Whether the codemod was applied
	Message string // Description of what changed
}

// GetAllCodemods returns all available codemods in the registry
func GetAllCodemods() []Codemod {
	codemods := []Codemod{
		getTimeoutMinutesCodemod(),
		getNetworkFirewallCodemod(),
		getCommandToSlashCommandCodemod(),
		getMCPScriptsModeCodemod(),
		getUploadAssetsCodemod(),
		getMigrateWritePermissionsToReadCodemod(),
		getExpandPermissionsShorthandCodemod(), // Fix permissions: read -> permissions: read-all
		getAgentTaskToAgentSessionCodemod(),
		getSandboxFalseToAgentFalseCodemod(), // Convert sandbox: false to sandbox.agent: false
		getScheduleAtToAroundCodemod(),
		getDeleteSchemaFileCodemod(),
		getGrepToolRemovalCodemod(),
		getMCPNetworkMigrationCodemod(),
		getDiscussionFlagRemovalCodemod(),
		getMCPModeToTypeCodemod(),
		getInstallScriptURLCodemod(),
		getBashAnonymousRemovalCodemod(),              // Replace bash: with bash: false
		getActivationOutputsCodemod(),                 // Transform needs.activation.outputs.* to steps.sanitized.outputs.*
		getRolesToOnRolesCodemod(),                    // Move top-level roles to on.roles
		getBotsToOnBotsCodemod(),                      // Move top-level bots to on.bots
		getEngineStepsToTopLevelCodemod(),             // Move engine.steps to top-level steps
		getStepsRunSecretsToEnvCodemod(),              // Move inline secrets in step run fields to step env bindings
		getEngineEnvSecretsCodemod(),                  // Remove unsafe secret-bearing engine.env entries
		getAssignToAgentDefaultAgentCodemod(),         // Rename deprecated default-agent to name in assign-to-agent
		getPlaywrightDomainsToNetworkAllowedCodemod(), // Migrate tools.playwright.allowed_domains to network.allowed
		getExpiresIntegerToDayStringCodemod(),         // Convert expires integer (days) to string with 'd' suffix
		getGitHubAppCodemod(),                         // Rename deprecated 'app' to 'github-app'
		getGitHubAppClientIDCodemod(),                 // Rename deprecated github-app.app-id to github-app.client-id
		getSafeInputsToMCPScriptsCodemod(),            // Rename safe-inputs to mcp-scripts
		getPluginsToDependenciesCodemod(),             // Migrate plugins to dependencies (plugins removed in favour of APM)
		getSerenaToSharedImportCodemod(),              // Migrate removed tools.serena to shared/mcp/serena.md import
		getWorkflowRunBranchesCodemod(),               // Add default branches to bare on.workflow_run trigger
		getDependabotPermissionsCodemod(),             // Add vulnerability-alerts: read when dependabot toolset is used
		getGitHubReposToAllowedReposCodemod(),         // Rename deprecated tools.github.repos to tools.github.allowed-repos
		getByokCopilotFeatureRemovalCodemod(),         // Remove deprecated features.byok-copilot (Copilot BYOK is default)
		getCliProxyFeatureToGitHubModeCodemod(),       // Migrate features.cli-proxy: true to tools.github.mode: gh-proxy
		getDIFCProxyToIntegrityProxyCodemod(),         // Migrate deprecated features.difc-proxy to tools.github.integrity-proxy
		getMountAsCLIsToCLIProxyCodemod(),             // Rename tools.mount-as-clis to tools.cli-proxy and remove features.mcp-cli
	}
	fixCodemodsLog.Printf("Loaded codemod registry: %d codemods available", len(codemods))
	return codemods
}
