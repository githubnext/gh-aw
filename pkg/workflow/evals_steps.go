// Package workflow - step builders for the BinEval evaluation job.
package workflow

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/github/gh-aw/pkg/constants"
	"github.com/github/gh-aw/pkg/logger"
	"github.com/github/gh-aw/pkg/workflow/compilerenv"
)

var evalsStepsLog = logger.New("workflow:evals_steps")

const (
	// evalsDir is the BinEval working directory on the runner.
	evalsDir = "/tmp/gh-aw/evals"

	// evalsLogPath is the engine output log written during the evals engine execution step.
	evalsLogPath = "/tmp/gh-aw/evals/evals.log"

	// evalsResultsPath is the parsed JSONL results file produced by the parse step.
	evalsResultsPath = "/tmp/gh-aw/" + constants.EvalsResultFilename
)

// buildEvalsJobSteps builds all steps that run inside the evals job.
// The steps analyse the agent artifact using a BinEval prompt and write
// per-question YES/NO results to evals.jsonl.
func (c *Compiler) buildEvalsJobSteps(data *WorkflowData) []string {
	if !data.Evals.HasEvals() {
		return nil
	}

	var steps []string

	steps = append(steps, "      # --- BinEval Evaluations ---\n")

	// Step 1: Clean stale firewall files from the agent artifact download so the
	// AWF squid container does not fail when the evals job pre-pulls images.
	steps = append(steps, c.buildCleanFirewallDirsStep()...)

	// Step 2: Pre-pull AWF container images for faster engine execution.
	steps = append(steps, c.buildPullAWFContainersStep(data)...)

	// Step 3: Copy agent output files into the evals working directory.
	steps = append(steps, buildPrepareEvalsFilesStep()...)

	// Step 4: Setup evals – writes the multi-question BinEval prompt via JS.
	steps = append(steps, c.buildSetupEvalsStep(data)...)

	// Step 5: Ensure the evals directory and log file exist before engine execution.
	steps = append(steps, buildEnsureEvalsDirStep()...)

	// Steps 6 & 7: Install engine and execute via AWF (network-restricted sandbox).
	steps = append(steps, c.buildEvalsEngineSteps(data)...)

	// Step 8: Parse engine output and write evals.jsonl.
	steps = append(steps, c.buildParseEvalsResultsStep(data)...)

	// Step 9: Redact secrets from evals results before upload.
	steps = append(steps, c.buildRedactEvalsSecretsStep(data)...)

	// Step 10: Upload evals.jsonl as the evals artifact.
	steps = append(steps, c.buildUploadEvalsArtifactStep(data)...)

	return steps
}

// buildPrepareEvalsFilesStep creates a step that copies agent output files into the
// evals working directory so the JS harness can read them for context.
func buildPrepareEvalsFilesStep() []string {
	return []string{
		"      - name: Prepare evals files\n",
		"        run: |\n",
		fmt.Sprintf("          mkdir -p %s\n", evalsDir),
		"          cp /tmp/gh-aw/agent_output.json " + evalsDir + "/agent_output.json 2>/dev/null || true\n",
		"          cp /tmp/gh-aw/aw-prompts/prompt.txt " + evalsDir + "/prompt.txt 2>/dev/null || true\n",
		fmt.Sprintf("          ls -la %s/ 2>/dev/null || true\n", evalsDir),
	}
}

// buildEnsureEvalsDirStep creates a step that ensures the evals directory and log
// file are present before the engine execution step writes to them.
func buildEnsureEvalsDirStep() []string {
	return []string{
		"      - name: Ensure evals directory and log\n",
		"        run: |\n",
		fmt.Sprintf("          mkdir -p %s\n", evalsDir),
		fmt.Sprintf("          touch %s\n", evalsLogPath),
	}
}

// buildSetupEvalsStep creates the github-script step that writes the multi-question
// BinEval evaluation prompt to /tmp/gh-aw/aw-prompts/prompt.txt.
func (c *Compiler) buildSetupEvalsStep(data *WorkflowData) []string {
	if data.Evals == nil {
		return nil
	}

	questionsJSON := marshalEvalsQuestions(data.Evals.Questions)
	model := data.Evals.Model
	if model == "" {
		model = "small"
	}

	script := `const { setupGlobals } = require('` + SetupActionDestination + `/setup_globals.cjs');
setupGlobals(core, github, context, exec, io, getOctokit);
const { main } = require('` + SetupActionDestination + `/run_evals.cjs');
await main();`

	steps := []string{
		"      - name: Setup BinEval evaluations\n",
		"        if: always()\n",
		fmt.Sprintf("        uses: %s\n", getCachedActionPin("actions/github-script", data)),
		"        env:\n",
		fmt.Sprintf("          GH_AW_EVALS_QUESTIONS: '%s'\n", escapeYAMLSingleQuoted(questionsJSON)),
		fmt.Sprintf("          GH_AW_EVALS_MODEL: %q\n", model),
		"          GH_AW_EVALS_PHASE: setup\n",
		"        with:\n",
		"          script: |\n",
	}
	steps = append(steps, FormatJavaScriptForYAML(script)...)
	return steps
}

// buildEvalsEngineSteps generates the engine installation and engine execution steps
// for the evals job. These mirror the inline detection engine execution path:
//  1. Install the agentic engine (same binary as the agent job)
//  2. Execute the engine through AWF (network-restricted sandbox) to answer eval questions
func (c *Compiler) buildEvalsEngineSteps(data *WorkflowData) []string {
	// Determine engine ID (same resolution order as detection).
	engineID := c.getEvalsEngineID(data)

	// Build the evals engine config by shallow-copying the main engine config.
	// This preserves all fields including Auth (OIDC/Azure), LLMProvider, permissions,
	// token weights, and other engine settings that should apply to eval runs.
	var evalsEngineConfig *EngineConfig
	if data.EngineConfig == nil {
		evalsEngineConfig = &EngineConfig{ID: engineID}
	} else {
		// Shallow copy all fields from the main engine config
		copy := *data.EngineConfig
		evalsEngineConfig = &copy
		if evalsEngineConfig.ID == "" {
			evalsEngineConfig.ID = engineID
		}
	}

	// Override model from evals frontmatter if specified.
	if data.Evals != nil && data.Evals.Model != "" {
		evalsEngineConfig.Model = data.Evals.Model
	}

	// Apply engine and enterprise default detection model (cost-effective for Q&A tasks).
	engine, err := c.getAgenticEngine(engineID)
	if err != nil {
		return []string{
			"      # Evals engine not available, skipping engine installation and execution\n",
		}
	}

	if evalsEngineConfig.Model == "" {
		if defaultModel := compilerenv.ResolveDefaultDetectionModel(""); defaultModel != "" {
			evalsEngineConfig.Model = defaultModel
		} else if defaultModel := engine.GetDefaultDetectionModel(); defaultModel != "" {
			evalsEngineConfig.Model = defaultModel
		}
	}

	// Inherit APITarget from the main engine config for GHE/custom endpoints.
	if evalsEngineConfig.APITarget == "" && data.EngineConfig != nil && data.EngineConfig.APITarget != "" {
		evalsEngineConfig.APITarget = data.EngineConfig.APITarget
	}

	// Normalize Pi engine model to bare model ID for Copilot CLI.
	originalEngineID := data.AI
	if data.EngineConfig != nil && data.EngineConfig.ID != "" {
		originalEngineID = data.EngineConfig.ID
	}
	if engineID == "copilot" && originalEngineID == "pi" {
		evalsEngineConfig.Model = extractPiModelID(evalsEngineConfig.Model)
	}

	// Build a minimal WorkflowData for evals engine execution.
	// IsDetectionRun reuses detection-style network restrictions and MaxAI credits,
	// which are appropriate for binary (YES/NO) evaluation tasks.
	// RunnerConfig is propagated from the main workflow data so that arc-dind topology
	// handling (daemon-visible Copilot staging step + daemon-visible spawn path) applies
	// to the evals job the same way it applies to the agent job.
	evalsData := &WorkflowData{
		Tools: map[string]any{
			"bash": []any{"*"},
		},
		SafeOutputs:       nil,
		EngineConfig:      evalsEngineConfig,
		AI:                engineID,
		Features:          data.Features,
		Permissions:       data.Permissions,
		CachedPermissions: data.CachedPermissions,
		IsDetectionRun:    true,
		RunnerConfig:      data.RunnerConfig, // propagate runner.topology (e.g. arc-dind) to the evals job
		NetworkPermissions: &NetworkPermissions{
			Allowed: getThreatDetectionAdditionalAllowedDomains(data),
		},
		SandboxConfig: &SandboxConfig{
			Agent: &AgentSandboxConfig{
				Type: SandboxTypeAWF,
			},
		},
	}

	var steps []string

	// Install the engine binary (fresh runner has no engine installed).
	installSteps := engine.GetInstallationSteps(evalsData)

	// Ensure Node.js is on PATH when the engine harness requires it.
	// Guard against engines whose install steps already bundle Setup Node.js.
	if engineRequiresNodeHarness(engine) && !installStepsContainNodeSetup(installSteps) {
		for _, line := range GenerateNodeJsSetupStep() {
			steps = append(steps, line+"\n")
		}
	}
	for _, step := range installSteps {
		for _, line := range step {
			steps = append(steps, line+"\n")
		}
	}

	// Codex requires MCP gateway config (OpenAI proxy provider in config.toml).
	if engine.GetID() == "codex" {
		var mcpSetup strings.Builder
		if err := c.generateMCPSetup(&mcpSetup, evalsData.Tools, engine, evalsData); err == nil {
			for line := range strings.SplitSeq(mcpSetup.String(), "\n") {
				if line != "" {
					steps = append(steps, line+"\n")
				}
			}
		} else {
			evalsStepsLog.Printf("Failed to generate MCP setup for Codex evals; OpenAI proxy configuration may be incomplete: %v", err)
		}
	}

	// Execute the engine through AWF; output is written to evalsLogPath.
	executionSteps := engine.GetExecutionSteps(evalsData, evalsLogPath)
	for _, step := range executionSteps {
		// Track whether we've injected the if/continue-on-error fields yet
		injected := false
		for _, line := range step {
			// Prefix the agentic_execution step ID to avoid collisions with the agent job step
			// IDs — job managers validate for duplicate step IDs across the compiled YAML.
			// This mirrors the same pattern used in buildDetectionEngineExecutionStep (see
			// threat_detection_inline_engine.go), where the ID is also a well-known literal
			// produced by every engine's GetExecutionSteps implementation.
			prefixed := strings.Replace(line, "id: agentic_execution", "id: evals_agentic_execution", 1)
			steps = append(steps, prefixed+"\n")
			// Inject always() condition and continue-on-error after the "- name:" line
			// so that infrastructure failures do not block the parse step that follows.
			// Search for the name field instead of assuming it's always at index 0 to handle
			// engines that might emit comments or other fields before the name.
			if !injected && strings.Contains(strings.TrimSpace(line), "- name:") {
				steps = append(steps, "        if: always()\n")
				steps = append(steps, "        continue-on-error: true\n")
				injected = true
			}
		}
	}

	return steps
}

// buildParseEvalsResultsStep creates the github-script step that reads the engine
// output log and writes structured per-question YES/NO records to evals.jsonl.
func (c *Compiler) buildParseEvalsResultsStep(data *WorkflowData) []string {
	if data.Evals == nil {
		return nil
	}

	questionsJSON := marshalEvalsQuestions(data.Evals.Questions)
	model := data.Evals.Model
	if model == "" {
		model = "small"
	}

	script := `const { setupGlobals } = require('` + SetupActionDestination + `/setup_globals.cjs');
setupGlobals(core, github, context, exec, io, getOctokit);
const { main } = require('` + SetupActionDestination + `/run_evals.cjs');
await main();`

	steps := []string{
		"      - name: Parse BinEval results\n",
		"        if: always()\n",
		"        continue-on-error: true\n",
		fmt.Sprintf("        uses: %s\n", getCachedActionPin("actions/github-script", data)),
		"        env:\n",
		fmt.Sprintf("          GH_AW_EVALS_QUESTIONS: '%s'\n", escapeYAMLSingleQuoted(questionsJSON)),
		fmt.Sprintf("          GH_AW_EVALS_MODEL: %q\n", model),
		"          GH_AW_EVALS_PHASE: parse\n",
		"        with:\n",
		"          script: |\n",
	}
	steps = append(steps, FormatJavaScriptForYAML(script)...)
	return steps
}

// buildRedactEvalsSecretsStep creates a step that runs redact_secrets.cjs to
// remove any credential patterns from evals.jsonl before the artifact is uploaded.
func (c *Compiler) buildRedactEvalsSecretsStep(data *WorkflowData) []string {
	script := `const { setupGlobals } = require('` + SetupActionDestination + `/setup_globals.cjs');
setupGlobals(core, github, context, exec, io, getOctokit);
const { main } = require('` + SetupActionDestination + `/redact_secrets.cjs');
await main();`

	steps := []string{
		"      - name: Redact secrets in evals results\n",
		"        if: always()\n",
		"        continue-on-error: true\n",
		fmt.Sprintf("        uses: %s\n", getCachedActionPin("actions/github-script", data)),
		"        with:\n",
		"          script: |\n",
	}
	steps = append(steps, FormatJavaScriptForYAML(script)...)
	return steps
}

// buildUploadEvalsArtifactStep creates the step that uploads evals.jsonl as the
// evals artifact for downstream consumption.
func (c *Compiler) buildUploadEvalsArtifactStep(data *WorkflowData) []string {
	evalsArtifactName := artifactPrefixExprForDownstreamJob(data) + constants.EvalsArtifactName
	return []string{
		"      - name: Upload evals results\n",
		"        if: always()\n",
		fmt.Sprintf("        uses: %s\n", c.getActionPin("actions/upload-artifact")),
		"        with:\n",
		"          name: " + evalsArtifactName + "\n",
		"          path: " + evalsResultsPath + "\n",
		"          if-no-files-found: ignore\n",
	}
}

// ---------------------------------------------------------------------------
// Engine configuration helpers
// ---------------------------------------------------------------------------

// getEvalsEngineID returns the engine ID to use for evals execution.
// Evals reuse the main workflow engine.
func (c *Compiler) getEvalsEngineID(data *WorkflowData) string {
	if data.EngineConfig != nil && data.EngineConfig.ID != "" {
		return data.EngineConfig.ID
	}
	if data.AI != "" {
		return data.AI
	}
	return "copilot"
}

// ---------------------------------------------------------------------------
// Utility helpers
// ---------------------------------------------------------------------------

// marshalEvalsQuestions serialises eval question definitions to a compact JSON
// array string suitable for embedding in a GitHub Actions env var.
func marshalEvalsQuestions(questions []EvalDefinition) string {
	if len(questions) == 0 {
		return "[]"
	}
	var sb strings.Builder
	sb.WriteString("[")
	for i, q := range questions {
		if i > 0 {
			sb.WriteString(",")
		}
		// Use json.Marshal for robust string quoting (handles all JSON escape sequences)
		idJSON, _ := json.Marshal(q.ID)             //nolint:jsonmarshalignoredeerror // marshaling a string cannot fail
		questionJSON, _ := json.Marshal(q.Question) //nolint:jsonmarshalignoredeerror // marshaling a string cannot fail
		fmt.Fprintf(&sb, `{"id":%s,"question":%s}`, idJSON, questionJSON)
	}
	sb.WriteString("]")
	return sb.String()
}
