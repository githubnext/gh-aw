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
	// For codex, buildEvalsEngineSteps calls generateMCPSetup which already emits
	// the Docker download step via generateDownloadDockerImagesStep, so skip here
	// to avoid a duplicate "Download container images" step name.
	if c.getEvalsEngineID(data) != string(constants.CodexEngine) {
		steps = append(steps, c.buildPullAWFContainersStep(data)...)
	}

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

	// Step 10: Render evals results as a progressive disclosure step summary section.
	// Runs after redaction so the published summary is always free of secrets.
	steps = append(steps, c.buildRenderEvalsSummaryStep(data)...)

	// Step 11: Upload evals.jsonl as the evals artifact.
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
	model := c.resolveEvalsExecutionModel(data)

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

	// Apply engine and enterprise default detection model (cost-effective for Q&A tasks).
	engine, err := c.getAgenticEngine(engineID)
	if err != nil {
		return []string{
			"      # Evals engine not available, skipping engine installation and execution\n",
		}
	}

	evalsEngineConfig := c.buildEvalsEngineConfig(data, engineID)
	evalsEngineConfig.Model = c.resolveEvalsExecutionModel(data)
	evalsData := buildEvalsWorkflowData(data, evalsEngineConfig, engineID)

	var steps []string
	steps = append(steps, buildEvalsEngineInstallSteps(engine, evalsData)...)
	steps = append(steps, c.buildEvalsMCPSetupSteps(engine, evalsData)...)
	steps = append(steps, buildEvalsEngineExecutionSteps(engine, evalsData)...)

	return steps
}

func (c *Compiler) buildEvalsEngineConfig(data *WorkflowData, engineID string) *EngineConfig {
	if data.EngineConfig == nil {
		return &EngineConfig{ID: engineID}
	}
	copy := *data.EngineConfig
	if copy.ID == "" {
		copy.ID = engineID
	}
	if copy.APITarget == "" && data.EngineConfig.APITarget != "" {
		copy.APITarget = data.EngineConfig.APITarget
	}
	return &copy
}

func buildEvalsWorkflowData(data *WorkflowData, evalsEngineConfig *EngineConfig, engineID string) *WorkflowData {
	return &WorkflowData{
		Tools: map[string]any{
			"bash": []any{"*"},
		},
		EngineConfig:      evalsEngineConfig,
		AI:                engineID,
		Features:          data.Features,
		Permissions:       data.Permissions,
		CachedPermissions: data.CachedPermissions,
		IsDetectionRun:    true,
		RunnerConfig:      data.RunnerConfig,
		NetworkPermissions: &NetworkPermissions{
			Allowed: getThreatDetectionAdditionalAllowedDomains(data),
		},
		SandboxConfig: &SandboxConfig{
			Agent: &AgentSandboxConfig{
				Type: SandboxTypeAWF,
			},
		},
	}
}

func buildEvalsEngineInstallSteps(engine CodingAgentEngine, evalsData *WorkflowData) []string {
	var steps []string
	installSteps := engine.GetInstallationSteps(evalsData)
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
	return steps
}

func (c *Compiler) buildEvalsMCPSetupSteps(engine CodingAgentEngine, evalsData *WorkflowData) []string {
	var steps []string
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
	return steps
}

func buildEvalsEngineExecutionSteps(engine CodingAgentEngine, evalsData *WorkflowData) []string {
	var steps []string
	executionSteps := engine.GetExecutionSteps(evalsData, evalsLogPath)
	for _, step := range executionSteps {
		steps = append(steps, buildSingleEvalsExecutionStep(step)...)
	}
	return steps
}

func buildSingleEvalsExecutionStep(step []string) []string {
	var steps []string
	injected := false
	for _, line := range step {
		prefixed := strings.Replace(line, "id: agentic_execution", "id: evals_agentic_execution", 1)
		steps = append(steps, prefixed+"\n")
		if !injected && strings.Contains(strings.TrimSpace(line), "- name:") {
			steps = append(steps, "        if: always()\n")
			steps = append(steps, "        continue-on-error: true\n")
			injected = true
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
	model := c.resolveEvalsExecutionModel(data)

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
		"          GITHUB_RUN_ID: ${{ github.run_id }}\n",
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
const { main } = require('` + SetupActionDestination + `/redact_evals_results.cjs');
await main();`

	secretReferences := c.collectEvalsSecretReferences(data)

	steps := []string{
		"      - name: Redact secrets in evals results\n",
		"        id: redact_evals_results\n",
		"        if: always()\n",
		fmt.Sprintf("        uses: %s\n", getCachedActionPin("actions/github-script", data)),
	}
	if len(secretReferences) > 0 {
		steps = append(steps, "        env:\n")
		escapedRefs := make([]string, len(secretReferences))
		for i, ref := range secretReferences {
			escapedRefs[i] = escapeSingleQuoteBackslash(ref)
		}
		steps = append(steps, fmt.Sprintf("          GH_AW_SECRET_NAMES: '%s'\n", strings.Join(escapedRefs, ",")))
		for _, secretName := range secretReferences {
			escapedSecretName := escapeSingleQuoteBackslash(secretName)
			steps = append(steps, fmt.Sprintf("          SECRET_%s: ${{ secrets.%s }}\n", escapedSecretName, secretName))
		}
	}
	steps = append(steps,
		"        with:\n",
		"          script: |\n",
	)
	steps = append(steps, FormatJavaScriptForYAML(script)...)
	return steps
}

// buildRenderEvalsSummaryStep creates a step that reads the redacted evals.jsonl
// and renders the results as a collapsible progressive disclosure section in the
// GitHub Actions step summary. Running after secret redaction ensures no credentials
// appear in the published summary.
func (c *Compiler) buildRenderEvalsSummaryStep(data *WorkflowData) []string {
	script := `const { setupGlobals } = require('` + SetupActionDestination + `/setup_globals.cjs');
setupGlobals(core, github, context, exec, io, getOctokit);
const { main } = require('` + SetupActionDestination + `/render_evals_summary.cjs');
await main();`

	steps := []string{
		"      - name: Render evals results to step summary\n",
		"        if: steps.redact_evals_results.outcome == 'success'\n",
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
		"        if: steps.redact_evals_results.outcome == 'success'\n",
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

func (c *Compiler) resolveEvalsExecutionModel(data *WorkflowData) string {
	model := ""
	if data.EngineConfig != nil && data.EngineConfig.Model != "" {
		model = data.EngineConfig.Model
	}
	if data.Evals != nil && data.Evals.Model != "" {
		model = data.Evals.Model
	}

	engineID := c.getEvalsEngineID(data)
	if model == "" {
		if defaultModel := compilerenv.ResolveDefaultDetectionModel(""); defaultModel != "" {
			model = defaultModel
		} else if engine, err := c.getAgenticEngine(engineID); err == nil {
			model = engine.GetDefaultDetectionModel()
		}
	}
	if model == "" {
		model = "small"
	}

	originalEngineID := data.AI
	if data.EngineConfig != nil && data.EngineConfig.ID != "" {
		originalEngineID = data.EngineConfig.ID
	}
	if engineID == "copilot" && originalEngineID == "pi" {
		model = extractPiModelID(model)
	}

	return model
}

func (c *Compiler) collectEvalsSecretReferences(data *WorkflowData) []string {
	if data.Evals == nil {
		return nil
	}
	return CollectSecretReferences(marshalEvalsQuestions(data.Evals.Questions) + "\n" + c.resolveEvalsExecutionModel(data))
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
