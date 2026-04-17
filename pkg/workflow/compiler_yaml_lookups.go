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
	default:
		// Custom or unknown engines don't have a default version
		compilerYamlLookupsLog.Printf("No default version for custom engine: %s", engineID)
		return ""
	}
}

// getDefaultAgentModel returns the model display value to use when no explicit model is configured.
// Returns "auto" for known engines whose model is dynamically determined by the AI provider
// (i.e. the provider chooses the model automatically), or empty string for custom/unknown engines.
func getDefaultAgentModel(engineID string) string {
	switch engineID {
	case "copilot", "claude", "codex", "gemini", "opencode":
		return "auto"
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
