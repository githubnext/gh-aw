package workflow

import (
	"fmt"
	"strings"

	"github.com/github/gh-aw/pkg/logger"
)

var compilerYamlStepGenerationLog = logger.New("workflow:compiler_yaml_step_generation")

// generateCheckoutActionsFolder generates the checkout step for the actions folder
// when running in dev mode and not using the action-tag feature. This is used to
// checkout the local actions before running the setup action.
//
// Returns a slice of strings that can be appended to a steps array, where each
// string represents a line of YAML for the checkout step. Returns nil if:
// - Not in dev or script mode
// - action-tag feature is specified (uses remote actions instead)
func (c *Compiler) generateCheckoutActionsFolder(data *WorkflowData) []string {
	compilerYamlStepGenerationLog.Printf("Generating checkout actions folder step: actionMode=%s, version=%s", c.actionMode, c.version)
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

// generateRestoreActionsSetupStep generates a single "Restore actions folder" step that
// re-checks out only the actions/setup subfolder from github/gh-aw. This is used in dev mode
// after a job step has checked out a different repository (or a different git branch) and
// replaced the workspace content, removing the actions/setup directory. Without restoring it,
// the GitHub Actions runner's post-step for "Setup Scripts" would fail with
// "Can't find 'action.yml', 'action.yaml' or 'Dockerfile' under .../actions/setup".
//
// The step is guarded by `if: always()` so it runs even if prior steps fail, ensuring
// the post-step cleanup can always complete.
//
// Returns the YAML for the step as a single string (for inclusion in a []string steps slice).
func (c *Compiler) generateRestoreActionsSetupStep() string {
	var step strings.Builder
	step.WriteString("      - name: Restore actions folder\n")
	step.WriteString("        if: always()\n")
	fmt.Fprintf(&step, "        uses: %s\n", GetActionPin("actions/checkout"))
	step.WriteString("        with:\n")
	step.WriteString("          repository: github/gh-aw\n")
	step.WriteString("          sparse-checkout: |\n")
	step.WriteString("            actions/setup\n")
	step.WriteString("          sparse-checkout-cone-mode: true\n")
	step.WriteString("          persist-credentials: false\n")
	return step.String()
}

// generateSetupStep generates the setup step based on the action mode.
// In script mode, it runs the setup.sh script directly from the checked-out source.
// In other modes (dev/release), it uses the setup action.
//
// Parameters:
//   - setupActionRef: The action reference for setup action (e.g., "./actions/setup" or "github/gh-aw/actions/setup@sha")
//   - destination: The destination path where files should be copied (e.g., SetupActionDestination)
//   - enableCustomTokens: Whether to enable custom-token support (installs @actions/github so handler_auth.cjs can create per-handler Octokit clients)
//   - traceID: Optional OTLP trace ID expression for cross-job span correlation (e.g., "${{ needs.activation.outputs.setup-trace-id }}"). Empty string means a new trace ID is generated.
//
// Returns a slice of strings representing the YAML lines for the setup step.
func (c *Compiler) generateSetupStep(setupActionRef string, destination string, enableCustomTokens bool, traceID string) []string {
	// Script mode: run the setup.sh script directly
	if c.actionMode.IsScript() {
		lines := []string{
			"      - name: Setup Scripts\n",
			"        id: setup\n",
			"        run: |\n",
			"          bash /tmp/gh-aw/actions-source/actions/setup/setup.sh\n",
			"        env:\n",
			fmt.Sprintf("          INPUT_DESTINATION: %s\n", destination),
			"          INPUT_JOB_NAME: ${{ github.job }}\n",
		}
		if traceID != "" {
			lines = append(lines, fmt.Sprintf("          INPUT_TRACE_ID: %s\n", traceID))
		}
		if enableCustomTokens {
			lines = append(lines, "          INPUT_SAFE_OUTPUT_CUSTOM_TOKENS: 'true'\n")
		}
		return lines
	}

	// Dev/Release mode: use the setup action
	lines := []string{
		"      - name: Setup Scripts\n",
		"        id: setup\n",
		fmt.Sprintf("        uses: %s\n", setupActionRef),
		"        with:\n",
		fmt.Sprintf("          destination: %s\n", destination),
		"          job-name: ${{ github.job }}\n",
	}
	if traceID != "" {
		lines = append(lines, fmt.Sprintf("          trace-id: %s\n", traceID))
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

// generateScriptModeCleanupStep generates a cleanup step for script mode that sends an OTLP
// conclusion span and removes /tmp/gh-aw/. This mirrors the post.js post step that runs
// automatically when using a `uses:` action in dev/release/action mode.
//
// The step is guarded by `if: always()` so it runs even if prior steps fail, ensuring
// trace spans are exported and temporary files are cleaned up in all cases.
//
// Only call this in script mode (c.actionMode.IsScript()).
func (c *Compiler) generateScriptModeCleanupStep() string {
	var step strings.Builder
	step.WriteString("      - name: Clean Scripts\n")
	step.WriteString("        if: always()\n")
	step.WriteString("        run: |\n")
	step.WriteString("          bash /tmp/gh-aw/actions-source/actions/setup/clean.sh\n")
	step.WriteString("        env:\n")
	fmt.Fprintf(&step, "          INPUT_DESTINATION: %s\n", SetupActionDestination)
	step.WriteString("          INPUT_JOB_NAME: ${{ github.job }}\n")
	return step.String()
}
