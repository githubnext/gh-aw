package workflow

import (
	"regexp"
	"strings"

	"github.com/github/gh-aw/pkg/constants"
	"github.com/github/gh-aw/pkg/logger"
)

var compilerYamlLookupsLog = logger.New("workflow:compiler_yaml_lookups")

// gitDescribeSHAPattern matches git-describe output ending with -N-gSHA (pre-compiled for performance)
var gitDescribeSHAPattern = regexp.MustCompile(`-\d+-g([0-9a-f]+)$`)

// getVersionForSetup returns the agent version to inject as GH_AW_INFO_VERSION in setup steps.
// It mirrors getInstallationVersion but derives the engine ID from data fields (not from a
// CodingAgentEngine object), allowing it to be called in contexts where the engine object
// is not available (e.g. compiler_yaml_step_generation.go).
func getVersionForSetup(data *WorkflowData) string {
	if data == nil {
		return ""
	}
	if data.EngineConfig != nil && data.EngineConfig.Version != "" {
		return data.EngineConfig.Version
	}
	engineID := ""
	if data.EngineConfig != nil && data.EngineConfig.ID != "" {
		engineID = data.EngineConfig.ID
	} else if data.AI != "" {
		engineID = data.AI
	}
	switch engineID {
	case "copilot":
		return string(constants.DefaultCopilotVersion)
	case "claude":
		return string(constants.DefaultClaudeCodeVersion)
	case "codex":
		return string(constants.DefaultCodexVersion)
	case "opencode":
		return string(constants.DefaultOpenCodeVersion)
	case "crush":
		return string(constants.DefaultCrushVersion)
	case "pi":
		return string(constants.DefaultPiVersion)
	default:
		return ""
	}
}

// getAWFVersionForSetup returns the AWF runtime version to inject as
// GH_AW_INFO_AWF_VERSION in setup steps so all job-local OTel spans can
// attribute telemetry to the firewall runtime that executed the job.
func getAWFVersionForSetup(data *WorkflowData) string {
	if data == nil {
		return ""
	}
	firewallConfig := getFirewallConfig(data)
	if firewallConfig == nil || !firewallConfig.Enabled {
		return ""
	}
	if firewallConfig.Version != "" {
		return firewallConfig.Version
	}
	return string(constants.DefaultFirewallVersion)
}

// getInstallationVersion returns the version that will be installed for the given engine.
// This matches the logic in BuildStandardNpmEngineInstallSteps.
func getInstallationVersion(data *WorkflowData, engine CodingAgentEngine) string {
	engineID := engine.GetID()
	compilerYamlLookupsLog.Printf("Getting installation version for engine: %s", engineID)

	// If version is specified in engine config, use it
	if data.EngineConfig != nil && data.EngineConfig.Version != "" {
		compilerYamlLookupsLog.Printf("Using engine config version: %s", data.EngineConfig.Version)
		return data.EngineConfig.Version
	}

	// Otherwise, use the default version for the engine
	switch engineID {
	case "copilot":
		return string(constants.DefaultCopilotVersion)
	case "claude":
		return string(constants.DefaultClaudeCodeVersion)
	case "codex":
		return string(constants.DefaultCodexVersion)
	case "opencode":
		return string(constants.DefaultOpenCodeVersion)
	case "crush":
		return string(constants.DefaultCrushVersion)
	case "pi":
		return string(constants.DefaultPiVersion)
	default:
		// Custom or unknown engines don't have a default version
		compilerYamlLookupsLog.Printf("No default version for custom engine: %s", engineID)
		return ""
	}
}

// getDefaultAgentModel returns the model display value to use when no explicit model is configured.
// For the copilot engine this matches the CopilotBYOKDefaultModel used in COPILOT_MODEL so that
// GH_AW_INFO_MODEL and COPILOT_MODEL agree on the same fallback.
// Returns "agent" for other known engines whose model is dynamically determined by the AI provider,
// or empty string for custom/unknown engines.
func getDefaultAgentModel(engineID string) string {
	switch engineID {
	case "copilot":
		return constants.CopilotBYOKDefaultModel
	case "claude", "gemini", "opencode", "crush", "pi":
		return "agent"
	case "codex":
		return constants.CodexDefaultModel
	default:
		return ""
	}
}

// versionToGitRef converts a compiler version string to a valid git ref for use
// in actions/checkout ref: fields.
//
// The version string is typically produced by `git describe --tags --always --dirty`
// and may contain suffixes that are not valid git refs. This function normalises it:
//   - "dev" or empty → "" (no ref, checkout will use the repository default branch)
//   - "v1.2.3-60-ge284d1e" → "e284d1e" (extract SHA from git-describe output)
//   - "v1.2.3-60-ge284d1e-dirty" → "e284d1e" (strip -dirty, then extract SHA)
//   - "v1.2.3-dirty" → "v1.2.3" (strip -dirty, valid tag)
//   - "v1.2.3" → "v1.2.3" (valid tag, used as-is)
//   - "e284d1e" → "e284d1e" (plain short SHA, used as-is)
func versionToGitRef(version string) string {
	compilerYamlLookupsLog.Printf("Converting version to git ref: %s", version)
	if version == "" || version == "dev" {
		return ""
	}
	// Strip optional -dirty suffix (appended by `git describe --dirty`)
	clean := strings.TrimSuffix(version, "-dirty")
	// If the version looks like `git describe` output with -N-gSHA, extract the SHA.
	// Pattern: anything ending with -<digits>-g<hexchars>
	if m := gitDescribeSHAPattern.FindStringSubmatch(clean); m != nil {
		compilerYamlLookupsLog.Printf("Extracted SHA from git-describe version: %s -> %s", version, m[1])
		return m[1]
	}
	compilerYamlLookupsLog.Printf("Using version as git ref: %s -> %s", version, clean)
	return clean
}
