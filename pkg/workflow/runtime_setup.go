package workflow

import (
	"fmt"
	"strings"

	"github.com/githubnext/gh-aw/pkg/constants"
	"github.com/goccy/go-yaml"
)

// RuntimeType represents a runtime environment that might need setup
type RuntimeType string

const (
	RuntimeNode   RuntimeType = "node"
	RuntimePython RuntimeType = "python"
	RuntimeGo     RuntimeType = "go"
	RuntimeRuby   RuntimeType = "ruby"
	RuntimeUV     RuntimeType = "uv" // Python package installer
)

// RuntimeRequirement represents a detected runtime requirement
type RuntimeRequirement struct {
	Type    RuntimeType
	Version string // Empty string means use default/latest
}

// commandPatterns maps command patterns to runtime types
var commandPatterns = map[string]RuntimeType{
	"node":    RuntimeNode,
	"npm":     RuntimeNode,
	"npx":     RuntimeNode,
	"yarn":    RuntimeNode,
	"pnpm":    RuntimeNode,
	"python":  RuntimePython,
	"python3": RuntimePython,
	"pip":     RuntimePython,
	"pip3":    RuntimePython,
	"uvx":     RuntimeUV,
	"uv":      RuntimeUV,
	"go":      RuntimeGo,
	"ruby":    RuntimeRuby,
	"gem":     RuntimeRuby,
	"bundle":  RuntimeRuby,
}

// DetectRuntimeRequirements analyzes workflow data to detect required runtimes
func DetectRuntimeRequirements(workflowData *WorkflowData) []RuntimeRequirement {
	requirements := make(map[RuntimeType]string) // map of runtime -> highest version

	// Detect from custom steps
	if workflowData.CustomSteps != "" {
		detectFromCustomSteps(workflowData.CustomSteps, requirements)
	}

	// Detect from MCP server configurations
	if workflowData.Tools != nil {
		detectFromMCPConfigs(workflowData.Tools, requirements)
	}

	// Detect from engine requirements
	if workflowData.EngineConfig != nil && len(workflowData.EngineConfig.Steps) > 0 {
		detectFromEngineSteps(workflowData.EngineConfig.Steps, requirements)
	}

	// Convert map to sorted slice
	var result []RuntimeRequirement
	runtimeOrder := []RuntimeType{RuntimeNode, RuntimePython, RuntimeGo, RuntimeRuby, RuntimeUV}
	for _, rt := range runtimeOrder {
		if version, exists := requirements[rt]; exists {
			result = append(result, RuntimeRequirement{
				Type:    rt,
				Version: version,
			})
		}
	}

	return result
}

// detectFromCustomSteps scans custom steps YAML for runtime commands
func detectFromCustomSteps(customSteps string, requirements map[RuntimeType]string) {
	// First check if setup actions already exist
	if hasExistingSetupAction(customSteps) {
		return // Don't auto-add if user already has setup actions
	}

	lines := strings.Split(customSteps, "\n")
	currentStepRun := ""

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Accumulate run command lines
		if strings.HasPrefix(trimmed, "run:") {
			currentStepRun = strings.TrimPrefix(trimmed, "run:")
			currentStepRun = strings.TrimSpace(currentStepRun)
		} else if currentStepRun != "" && (strings.HasPrefix(line, "    ") || strings.HasPrefix(line, "\t")) {
			// Multi-line run command
			currentStepRun += " " + trimmed
		} else if currentStepRun != "" {
			// End of run command, analyze it
			analyzeCommand(currentStepRun, requirements)
			currentStepRun = ""
		}

		// Also check if it's a single-line run
		if strings.HasPrefix(trimmed, "run:") && !strings.HasSuffix(trimmed, "|") && !strings.HasSuffix(trimmed, ">") {
			cmd := strings.TrimPrefix(trimmed, "run:")
			cmd = strings.TrimSpace(cmd)
			analyzeCommand(cmd, requirements)
		}
	}

	// Analyze final run command if any
	if currentStepRun != "" {
		analyzeCommand(currentStepRun, requirements)
	}
}

// detectFromMCPConfigs scans MCP server configurations for runtime commands
func detectFromMCPConfigs(tools map[string]any, requirements map[RuntimeType]string) {
	for _, toolValue := range tools {
		if toolConfig, ok := toolValue.(map[string]any); ok {
			// Check for command field
			if command, hasCommand := toolConfig["command"]; hasCommand {
				if commandStr, ok := command.(string); ok {
					// Detect runtime from command
					if runtime, found := commandPatterns[commandStr]; found {
						// Check if we need to update version
						if version, exists := toolConfig["version"]; exists {
							if versionStr, ok := version.(string); ok {
								updateRequiredVersion(runtime, versionStr, requirements)
							}
						} else if _, alreadyHas := requirements[runtime]; !alreadyHas {
							requirements[runtime] = "" // No specific version
						}
					}
				}
			}
		}
	}
}

// detectFromEngineSteps scans custom engine steps for runtime requirements
func detectFromEngineSteps(steps []map[string]any, requirements map[RuntimeType]string) {
	for _, step := range steps {
		// Check for 'run' field in step
		if runCmd, hasRun := step["run"]; hasRun {
			if runStr, ok := runCmd.(string); ok {
				analyzeCommand(runStr, requirements)
			}
		}

		// Check for 'uses' field to see if setup actions are present
		if uses, hasUses := step["uses"]; hasUses {
			if usesStr, ok := uses.(string); ok {
				if strings.Contains(usesStr, "setup-node") ||
					strings.Contains(usesStr, "setup-python") ||
					strings.Contains(usesStr, "setup-go") ||
					strings.Contains(usesStr, "setup-ruby") {
					// User already has setup actions, don't auto-add
					return
				}
			}
		}
	}
}

// analyzeCommand detects runtime requirements from a shell command
func analyzeCommand(command string, requirements map[RuntimeType]string) {
	// Split command into tokens
	tokens := strings.Fields(command)

	// Track if we've seen uv, since "uv pip" shouldn't also trigger pip detection
	uvSeen := false

	for i, token := range tokens {
		// Remove common shell operators and get base command
		baseCmd := strings.TrimLeft(token, "&|;")
		baseCmd = strings.Split(baseCmd, "=")[0] // Handle VAR=value cases

		// Check if this matches a runtime command
		if runtime, found := commandPatterns[baseCmd]; found {
			// Special case: if this is "pip" and we previously saw "uv", skip it
			if (baseCmd == "pip" || baseCmd == "pip3") && uvSeen {
				continue
			}

			// Track if we see uv
			if baseCmd == "uv" || baseCmd == "uvx" {
				uvSeen = true
			}

			if _, alreadyHas := requirements[runtime]; !alreadyHas {
				requirements[runtime] = "" // No specific version from command
			}
		}

		// Also check the previous token for context (e.g., "uv pip" pattern)
		if i > 0 {
			prevToken := tokens[i-1]
			prevBaseCmd := strings.TrimLeft(prevToken, "&|;")
			prevBaseCmd = strings.Split(prevBaseCmd, "=")[0]

			// If previous was "uv" and current is "pip", we already marked uv, so continue
			if (prevBaseCmd == "uv") && (baseCmd == "pip" || baseCmd == "pip3") {
				continue
			}
		}
	}
}

// hasExistingSetupAction checks if custom steps already contain setup actions
func hasExistingSetupAction(customSteps string) bool {
	return strings.Contains(customSteps, "actions/setup-node") ||
		strings.Contains(customSteps, "actions/setup-python") ||
		strings.Contains(customSteps, "actions/setup-go") ||
		strings.Contains(customSteps, "actions/setup-ruby") ||
		strings.Contains(customSteps, "astral-sh/setup-uv")
}

// updateRequiredVersion updates the version requirement, choosing the highest version
func updateRequiredVersion(runtime RuntimeType, newVersion string, requirements map[RuntimeType]string) {
	existing, exists := requirements[runtime]

	if !exists || existing == "" {
		requirements[runtime] = newVersion
		return
	}

	// If new version is empty, keep existing
	if newVersion == "" {
		return
	}

	// Compare versions and keep the higher one
	if compareVersions(newVersion, existing) > 0 {
		requirements[runtime] = newVersion
	}
}

// compareVersions compares two semantic versions, returns 1 if v1 > v2, -1 if v1 < v2, 0 if equal
// Note: Non-numeric version parts (e.g., 'beta', 'alpha') default to 0 for comparison purposes
func compareVersions(v1, v2 string) int {
	parts1 := strings.Split(v1, ".")
	parts2 := strings.Split(v2, ".")

	maxLen := len(parts1)
	if len(parts2) > maxLen {
		maxLen = len(parts2)
	}

	for i := 0; i < maxLen; i++ {
		var p1, p2 int

		if i < len(parts1) {
			_, _ = fmt.Sscanf(parts1[i], "%d", &p1) // Ignore error, defaults to 0 for non-numeric parts
		}
		if i < len(parts2) {
			_, _ = fmt.Sscanf(parts2[i], "%d", &p2) // Ignore error, defaults to 0 for non-numeric parts
		}

		if p1 > p2 {
			return 1
		} else if p1 < p2 {
			return -1
		}
	}

	return 0
}

// GenerateRuntimeSetupSteps creates GitHub Actions steps for runtime setup
func GenerateRuntimeSetupSteps(requirements []RuntimeRequirement) []GitHubActionStep {
	var steps []GitHubActionStep

	for _, req := range requirements {
		switch req.Type {
		case RuntimeNode:
			steps = append(steps, generateNodeSetup(req.Version))
		case RuntimePython:
			steps = append(steps, generatePythonSetup(req.Version))
		case RuntimeGo:
			steps = append(steps, generateGoSetup(req.Version))
		case RuntimeRuby:
			steps = append(steps, generateRubySetup(req.Version))
		case RuntimeUV:
			steps = append(steps, generateUVSetup(req.Version))
		}
	}

	return steps
}

// generateNodeSetup creates a setup-node step
func generateNodeSetup(version string) GitHubActionStep {
	if version == "" {
		version = constants.DefaultNodeVersion
	}
	return GitHubActionStep{
		"      - name: Setup Node.js",
		"        uses: actions/setup-node@v4",
		"        with:",
		fmt.Sprintf("          node-version: '%s'", version),
	}
}

// generatePythonSetup creates a setup-python step
func generatePythonSetup(version string) GitHubActionStep {
	if version == "" {
		version = constants.DefaultPythonVersion
	}
	return GitHubActionStep{
		"      - name: Setup Python",
		"        uses: actions/setup-python@v5",
		"        with:",
		fmt.Sprintf("          python-version: '%s'", version),
	}
}

// generateGoSetup creates a setup-go step
func generateGoSetup(version string) GitHubActionStep {
	step := GitHubActionStep{
		"      - name: Setup Go",
		"        uses: actions/setup-go@v5",
	}

	if version != "" {
		step = append(step, "        with:")
		step = append(step, fmt.Sprintf("          go-version: '%s'", version))
	} else {
		// Use go-version-file if no specific version
		step = append(step, "        with:")
		step = append(step, "          go-version-file: go.mod")
		step = append(step, "          cache: true")
	}

	return step
}

// generateRubySetup creates a setup-ruby step
func generateRubySetup(version string) GitHubActionStep {
	if version == "" {
		version = constants.DefaultRubyVersion
	}
	return GitHubActionStep{
		"      - name: Setup Ruby",
		"        uses: ruby/setup-ruby@v1",
		"        with:",
		fmt.Sprintf("          ruby-version: '%s'", version),
	}
}

// generateUVSetup creates a setup-uv step
func generateUVSetup(version string) GitHubActionStep {
	step := GitHubActionStep{
		"      - name: Setup uv",
		"        uses: astral-sh/setup-uv@v5",
	}

	if version != "" {
		step = append(step, "        with:")
		step = append(step, fmt.Sprintf("          version: '%s'", version))
	}

	return step
}

// ShouldSkipRuntimeSetup checks if we should skip automatic runtime setup
// This returns true if the workflow already has setup actions in custom steps
func ShouldSkipRuntimeSetup(workflowData *WorkflowData) bool {
	if workflowData.CustomSteps != "" && hasExistingSetupAction(workflowData.CustomSteps) {
		return true
	}

	// Also check engine steps
	if workflowData.EngineConfig != nil {
		for _, step := range workflowData.EngineConfig.Steps {
			if uses, hasUses := step["uses"]; hasUses {
				if usesStr, ok := uses.(string); ok {
					if strings.Contains(usesStr, "setup-node") ||
						strings.Contains(usesStr, "setup-python") ||
						strings.Contains(usesStr, "setup-go") ||
						strings.Contains(usesStr, "setup-ruby") ||
						strings.Contains(usesStr, "astral-sh/setup-uv") {
						return true
					}
				}
			}
		}
	}

	return false
}

// ExtractVersionFromSteps tries to extract version requirements from existing setup actions
func ExtractVersionFromSteps(customSteps string) map[RuntimeType]string {
	versions := make(map[RuntimeType]string)

	// Parse YAML to extract version information
	var stepsWrapper struct {
		Steps []map[string]any `yaml:"steps"`
	}

	if err := yaml.Unmarshal([]byte(customSteps), &stepsWrapper); err == nil {
		for _, step := range stepsWrapper.Steps {
			if uses, hasUses := step["uses"]; hasUses {
				if usesStr, ok := uses.(string); ok {
					// Check for setup actions and extract version
					if strings.Contains(usesStr, "setup-node") {
						if withMap, hasWith := step["with"].(map[string]any); hasWith {
							if version, hasVersion := withMap["node-version"]; hasVersion {
								if versionStr, ok := version.(string); ok {
									versions[RuntimeNode] = strings.Trim(versionStr, "'\"")
								}
							}
						}
					} else if strings.Contains(usesStr, "setup-python") {
						if withMap, hasWith := step["with"].(map[string]any); hasWith {
							if version, hasVersion := withMap["python-version"]; hasVersion {
								if versionStr, ok := version.(string); ok {
									versions[RuntimePython] = strings.Trim(versionStr, "'\"")
								}
							}
						}
					} else if strings.Contains(usesStr, "setup-go") {
						if withMap, hasWith := step["with"].(map[string]any); hasWith {
							if version, hasVersion := withMap["go-version"]; hasVersion {
								if versionStr, ok := version.(string); ok {
									versions[RuntimeGo] = strings.Trim(versionStr, "'\"")
								}
							}
						}
					} else if strings.Contains(usesStr, "setup-ruby") {
						if withMap, hasWith := step["with"].(map[string]any); hasWith {
							if version, hasVersion := withMap["ruby-version"]; hasVersion {
								if versionStr, ok := version.(string); ok {
									versions[RuntimeRuby] = strings.Trim(versionStr, "'\"")
								}
							}
						}
					}
				}
			}
		}
	}

	return versions
}
