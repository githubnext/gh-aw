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
		getAssignToAgentDefaultAgentCodemod(),         // Rename deprecated default-agent to name in assign-to-agent
		getPlaywrightDomainsToNetworkAllowedCodemod(), // Migrate tools.playwright.allowed_domains to network.allowed
		getExpiresIntegerToStringCodemod(),            // Convert expires integer (days) to string with 'd' suffix
		getGitHubAppCodemod(),                         // Rename deprecated 'app' to 'github-app'
		getSafeInputsToMCPScriptsCodemod(),            // Rename safe-inputs to mcp-scripts
		getPluginsToDependenciesCodemod(),             // Migrate plugins to dependencies (plugins removed in favour of APM)
		getGitHubReposToAllowedReposCodemod(),         // Rename deprecated tools.github.repos to tools.github.allowed-repos
		getDIFCProxyToIntegrityProxyCodemod(),         // Migrate deprecated features.difc-proxy to tools.github.integrity-proxy
	}
	fixCodemodsLog.Printf("Loaded codemod registry: %d codemods available", len(codemods))
	return codemods
}
