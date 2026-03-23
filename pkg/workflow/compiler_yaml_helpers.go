package workflow

import (
	"fmt"
	"path/filepath"
	"regexp"
	"slices"
	"strings"

	"github.com/github/gh-aw/pkg/constants"
	"github.com/github/gh-aw/pkg/logger"
	"github.com/goccy/go-yaml"
)

var compilerYamlHelpersLog = logger.New("workflow:compiler_yaml_helpers")

// ContainsCheckout returns true if the given custom steps contain an actions/checkout step
func ContainsCheckout(customSteps string) bool {
	if customSteps == "" {
		return false
	}

	// Look for actions/checkout usage patterns
	checkoutPatterns := []string{
		"actions/checkout@",
		"uses: actions/checkout",
		"- uses: actions/checkout",
	}

	lowerSteps := strings.ToLower(customSteps)
	for _, pattern := range checkoutPatterns {
		if strings.Contains(lowerSteps, strings.ToLower(pattern)) {
			compilerYamlHelpersLog.Print("Detected actions/checkout in custom steps")
			return true
		}
	}

	return false
}

// GetWorkflowIDFromPath extracts the workflow ID from a markdown file path.
// The workflow ID is the filename without the .md extension.
// Example: "/path/to/ai-moderator.md" -> "ai-moderator"
func GetWorkflowIDFromPath(markdownPath string) string {
	return strings.TrimSuffix(filepath.Base(markdownPath), ".md")
}

// SanitizeWorkflowIDForCacheKey sanitizes a workflow ID for use in cache keys.
// It removes all hyphens and converts to lowercase to create a filesystem-safe identifier.
// Example: "Smoke-Copilot" -> "smokecopilot"
func SanitizeWorkflowIDForCacheKey(workflowID string) string {
	// Convert to lowercase
	sanitized := strings.ToLower(workflowID)
	// Remove all hyphens
	sanitized = strings.ReplaceAll(sanitized, "-", "")
	return sanitized
}

// ConvertStepToYAML converts a step map to YAML string with proper indentation.
// This is a shared utility function used by all engines and the compiler.
func ConvertStepToYAML(stepMap map[string]any) (string, error) {
	// Use OrderMapFields to get ordered MapSlice
	orderedStep := OrderMapFields(stepMap, constants.PriorityStepFields)

	// Wrap in array for step list format and marshal with proper options
	yamlBytes, err := yaml.MarshalWithOptions([]yaml.MapSlice{orderedStep}, DefaultMarshalOptions...)
	if err != nil {
		return "", fmt.Errorf("failed to marshal step to YAML: %w", err)
	}

	// Convert to string and adjust base indentation to match GitHub Actions format
	yamlStr := string(yamlBytes)

	// Post-process to move version comments outside of quoted uses values
	// This handles cases like: uses: "slug@sha # v1"  ->  uses: slug@sha # v1
	yamlStr = unquoteUsesWithComments(yamlStr)

	// Add 6 spaces to the beginning of each line to match GitHub Actions step indentation
	lines := strings.Split(strings.TrimSpace(yamlStr), "\n")
	var result strings.Builder

	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			result.WriteString("\n")
		} else {
			result.WriteString("      " + line + "\n")
		}
	}

	return result.String(), nil
}

// convertStepToYAML is the method form of ConvertStepToYAML for callers that hold a *Compiler.
func (c *Compiler) convertStepToYAML(stepMap map[string]any) (string, error) {
	return ConvertStepToYAML(stepMap)
}

// unquoteUsesWithComments removes quotes from uses values that contain version comments.
// Transforms: uses: "slug@sha # v1"  ->  uses: slug@sha # v1
// This is needed because the YAML marshaller quotes strings containing #, but GitHub Actions
// expects unquoted uses values with inline comments.
func unquoteUsesWithComments(yamlStr string) string {
	lines := strings.Split(yamlStr, "\n")
	for i, line := range lines {
		// Look for uses: followed by a quoted string containing a # comment
		// This handles various indentation levels and formats
		trimmed := strings.TrimSpace(line)

		// Check if line contains uses: with a quoted value
		if !strings.Contains(trimmed, "uses: \"") {
			continue
		}

		// Check if the quoted value contains a version comment
		if !strings.Contains(trimmed, " # ") {
			continue
		}

		// Find the position of uses: " in the original line
		usesIdx := strings.Index(line, "uses: \"")
		if usesIdx == -1 {
			continue
		}

		// Extract the part before uses: (indentation)
		prefix := line[:usesIdx]

		// Find the opening and closing quotes
		quoteStart := usesIdx + 7 // len("uses: \"")
		quoteEnd := strings.Index(line[quoteStart:], "\"")
		if quoteEnd == -1 {
			continue
		}
		quoteEnd += quoteStart

		// Extract the quoted content
		quotedContent := line[quoteStart:quoteEnd]

		// Extract any content after the closing quote
		suffix := line[quoteEnd+1:]

		// Reconstruct the line without quotes
		lines[i] = prefix + "uses: " + quotedContent + suffix
	}
	return strings.Join(lines, "\n")
}

// getInstallationVersion returns the version that will be installed for the given engine.
// This matches the logic in BuildStandardNpmEngineInstallSteps.
func getInstallationVersion(data *WorkflowData, engine CodingAgentEngine) string {
	engineID := engine.GetID()
	compilerYamlHelpersLog.Printf("Getting installation version for engine: %s", engineID)

	// If version is specified in engine config, use it
	if data.EngineConfig != nil && data.EngineConfig.Version != "" {
		compilerYamlHelpersLog.Printf("Using engine config version: %s", data.EngineConfig.Version)
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
		compilerYamlHelpersLog.Printf("No default version for custom engine: %s", engineID)
		return ""
	}
}

// getDefaultAgentModel returns the model display value to use when no explicit model is configured.
// Returns "auto" for known engines whose model is dynamically determined by the AI provider
// (i.e. the provider chooses the model automatically), or empty string for custom/unknown engines.
func getDefaultAgentModel(engineID string) string {
	switch engineID {
	case "copilot", "claude", "codex", "gemini":
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
	compilerYamlHelpersLog.Printf("Converting version to git ref: %s", version)
	if version == "" || version == "dev" {
		return ""
	}
	// Strip optional -dirty suffix (appended by `git describe --dirty`)
	clean := strings.TrimSuffix(version, "-dirty")
	// If the version looks like `git describe` output with -N-gSHA, extract the SHA.
	// Pattern: anything ending with -<digits>-g<hexchars>
	shaRe := regexp.MustCompile(`-\d+-g([0-9a-f]+)$`)
	if m := shaRe.FindStringSubmatch(clean); m != nil {
		compilerYamlHelpersLog.Printf("Extracted SHA from git-describe version: %s -> %s", version, m[1])
		return m[1]
	}
	compilerYamlHelpersLog.Printf("Using version as git ref: %s -> %s", version, clean)
	return clean
}

// generateCheckoutActionsFolder generates the checkout step for the actions folder
// when running in dev mode and not using the action-tag feature. This is used to
// checkout the local actions before running the setup action.
//
// Returns a slice of strings that can be appended to a steps array, where each
// string represents a line of YAML for the checkout step. Returns nil if:
// - Not in dev or script mode
// - action-tag feature is specified (uses remote actions instead)
func (c *Compiler) generateCheckoutActionsFolder(data *WorkflowData) []string {
	compilerYamlHelpersLog.Printf("Generating checkout actions folder step: actionMode=%s, version=%s", c.actionMode, c.version)
	// Check if action-tag is specified - if so, we're using remote actions
	if data != nil && data.Features != nil {
		if actionTagVal, exists := data.Features["action-tag"]; exists {
			if actionTagStr, ok := actionTagVal.(string); ok && actionTagStr != "" {
				// action-tag is set, use remote actions - no checkout needed
				return nil
			}
		}
	}

	// Derive a clean git ref from the compiler's version string.
	// Required so that cross-repo callers checkout github/gh-aw at the correct
	// commit rather than the default branch, which may be missing JS modules
	// that were added after the latest tag.
	ref := versionToGitRef(c.version)

	// Script mode: checkout .github folder from github/gh-aw to /tmp/gh-aw/actions-source/
	if c.actionMode.IsScript() {
		lines := []string{
			"      - name: Checkout actions folder\n",
			fmt.Sprintf("        uses: %s\n", GetActionPin("actions/checkout")),
			"        with:\n",
			"          repository: github/gh-aw\n",
		}
		if ref != "" {
			lines = append(lines, fmt.Sprintf("          ref: %s\n", ref))
		}
		lines = append(lines,
			"          sparse-checkout: |\n",
			"            actions\n",
			"          path: /tmp/gh-aw/actions-source\n",
			"          fetch-depth: 1\n",
			"          persist-credentials: false\n",
		)
		return lines
	}

	// Dev mode: checkout actions folder from github/gh-aw so that cross-repo
	// callers (e.g. event-driven relays) can find the actions/ directory.
	// Without repository: the runner defaults to the caller's repo, which has
	// no actions/ directory, causing Setup Scripts to fail immediately.
	if c.actionMode.IsDev() {
		lines := []string{
			"      - name: Checkout actions folder\n",
			fmt.Sprintf("        uses: %s\n", GetActionPin("actions/checkout")),
			"        with:\n",
			"          repository: github/gh-aw\n",
			"          sparse-checkout: |\n",
			"            actions\n",
			"          persist-credentials: false\n",
		}
		return lines
	}

	// Release mode or other modes: no checkout needed
	return nil
}

// generateCheckoutGitHubFolder generates the checkout step for the .github and .agents folders
// for the agent job. This ensures workflows have access to workflow configurations,
// runtime imports, and skills even when they don't do a full repository checkout.
//
// This checkout works in all modes (dev, script, release) and uses shallow clone
// for minimal overhead. It should only be called in the main agent job.
//
// Returns a slice of strings that can be appended to a steps array, where each
// string represents a line of YAML for the checkout step. Returns nil if:

// generateGitHubScriptWithRequire is implemented in compiler_github_actions_steps.go

// generateInlineGitHubScriptStep is implemented in compiler_github_actions_steps.go

// generateSetupStep generates the setup step based on the action mode.
// In script mode, it runs the setup.sh script directly from the checked-out source.
// In other modes (dev/release), it uses the setup action.
//
// Parameters:
//   - setupActionRef: The action reference for setup action (e.g., "./actions/setup" or "github/gh-aw/actions/setup@sha")
//   - destination: The destination path where files should be copied (e.g., SetupActionDestination)
//   - enableCustomTokens: Whether to enable custom-token support (installs @actions/github so handler_auth.cjs can create per-handler Octokit clients)
//
// Returns a slice of strings representing the YAML lines for the setup step.
func (c *Compiler) generateSetupStep(setupActionRef string, destination string, enableCustomTokens bool) []string {
	// Script mode: run the setup.sh script directly
	if c.actionMode.IsScript() {
		lines := []string{
			"      - name: Setup Scripts\n",
			"        run: |\n",
			"          bash /tmp/gh-aw/actions-source/actions/setup/setup.sh\n",
			"        env:\n",
			fmt.Sprintf("          INPUT_DESTINATION: %s\n", destination),
		}
		if enableCustomTokens {
			lines = append(lines, "          INPUT_SAFE_OUTPUT_CUSTOM_TOKENS: 'true'\n")
		}
		return lines
	}

	// Dev/Release mode: use the setup action
	lines := []string{
		"      - name: Setup Scripts\n",
		fmt.Sprintf("        uses: %s\n", setupActionRef),
		"        with:\n",
		fmt.Sprintf("          destination: %s\n", destination),
	}
	if enableCustomTokens {
		lines = append(lines, "          safe-output-custom-tokens: 'true'\n")
	}
	return lines
}

// generateSetRuntimePathsStep generates a step that sets RUNNER_TEMP-based env vars
// via $GITHUB_OUTPUT. These cannot be set in job-level env: because the runner context
// is not available there (only in step-level env: and run: blocks).
// The step ID "set-runtime-paths" is referenced by downstream steps that consume these outputs.
func (c *Compiler) generateSetRuntimePathsStep() []string {
	return []string{
		"      - name: Set runtime paths\n",
		"        id: set-runtime-paths\n",
		"        run: |\n",
		"          echo \"GH_AW_SAFE_OUTPUTS=${RUNNER_TEMP}/gh-aw/safeoutputs/outputs.jsonl\" >> \"$GITHUB_OUTPUT\"\n",
		"          echo \"GH_AW_SAFE_OUTPUTS_CONFIG_PATH=${RUNNER_TEMP}/gh-aw/safeoutputs/config.json\" >> \"$GITHUB_OUTPUT\"\n",
		"          echo \"GH_AW_SAFE_OUTPUTS_TOOLS_PATH=${RUNNER_TEMP}/gh-aw/safeoutputs/tools.json\" >> \"$GITHUB_OUTPUT\"\n",
	}
}

// renderStepFromMap renders a GitHub Actions step from a map to YAML
func (c *Compiler) renderStepFromMap(yaml *strings.Builder, step map[string]any, data *WorkflowData, indent string) {
	// Start the step with a dash
	yaml.WriteString(indent + "- ")

	// Track if we've written the first line
	firstField := true

	// Order of fields to write (matches GitHub Actions convention)
	fieldOrder := []string{"name", "id", "if", "uses", "with", "run", "env", "working-directory", "continue-on-error", "timeout-minutes", "shell"}

	for _, field := range fieldOrder {
		if value, exists := step[field]; exists {
			// Add proper indentation for non-first fields
			if !firstField {
				yaml.WriteString(indent + "  ")
			}
			firstField = false

			// Render the field based on its type
			switch v := value.(type) {
			case string:
				// Handle multi-line strings (especially for 'run' field)
				if field == "run" && strings.Contains(v, "\n") {
					fmt.Fprintf(yaml, "%s: |\n", field)
					lines := strings.SplitSeq(v, "\n")
					for line := range lines {
						fmt.Fprintf(yaml, "%s    %s\n", indent, line)
					}
				} else {
					fmt.Fprintf(yaml, "%s: %s\n", field, v)
				}
			case map[string]any:
				// For complex fields like "with" or "env"
				fmt.Fprintf(yaml, "%s:\n", field)
				for key, val := range v {
					fmt.Fprintf(yaml, "%s    %s: %v\n", indent, key, val)
				}
			default:
				fmt.Fprintf(yaml, "%s: %v\n", field, v)
			}
		}
	}

	// Add any remaining fields not in the predefined order
	for field, value := range step {
		// Skip fields we've already processed
		skip := slices.Contains(fieldOrder, field)
		if skip {
			continue
		}

		if !firstField {
			yaml.WriteString(indent + "  ")
		}
		firstField = false

		switch v := value.(type) {
		case string:
			// Handle multi-line strings
			if strings.Contains(v, "\n") {
				fmt.Fprintf(yaml, "%s: |\n", field)
				lines := strings.SplitSeq(v, "\n")
				for line := range lines {
					fmt.Fprintf(yaml, "%s    %s\n", indent, line)
				}
			} else {
				fmt.Fprintf(yaml, "%s: %s\n", field, v)
			}
		case map[string]any:
			fmt.Fprintf(yaml, "%s:\n", field)
			for key, val := range v {
				fmt.Fprintf(yaml, "%s    %s: %v\n", indent, key, val)
			}
		default:
			fmt.Fprintf(yaml, "%s: %v\n", field, v)
		}
	}
}
