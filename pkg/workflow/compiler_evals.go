// Package workflow - BinEval evaluation job assembler and frontmatter extraction.
package workflow

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/github/gh-aw/pkg/constants"
	"github.com/github/gh-aw/pkg/logger"
	"github.com/github/gh-aw/pkg/workflow/compilerenv"
)

var evalsLog = logger.New("workflow:compiler_evals")

// evalsWorkDir is the runtime directory where eval inputs and outputs are stored.
const evalsWorkDir = constants.EvalDir

const evalJobConditionTemplate = "always() && !cancelled() && needs.%s.result != 'skipped'"

// evalStepCondition is the condition used on each engine execution step in the eval job.
// It mirrors detectionStepCondition: always run so that the parse step can report failures.
const evalStepCondition = "always()"

// extractEvalsFromFrontmatter reads the "evals" array from the frontmatter map
// and returns a slice of typed EvalDefinition values.
// Returns nil when the key is absent or empty.
func extractEvalsFromFrontmatter(frontmatter map[string]any) []EvalDefinition {
	raw, ok := frontmatter["evals"]
	if !ok || raw == nil {
		return nil
	}
	rawSlice, ok := raw.([]any)
	if !ok {
		evalsLog.Printf("evals: unexpected type %T, expected []any", raw)
		return nil
	}
	defs := make([]EvalDefinition, 0, len(rawSlice))
	for i, item := range rawSlice {
		m, ok := item.(map[string]any)
		if !ok {
			evalsLog.Printf("evals[%d]: unexpected type %T, expected map", i, item)
			continue
		}
		id, _ := m["id"].(string)
		question, _ := m["question"].(string)
		if id == "" || question == "" {
			evalsLog.Printf("evals[%d]: skipping entry with missing id or question", i)
			continue
		}
		defs = append(defs, EvalDefinition{ID: id, Question: question})
	}
	if len(defs) == 0 {
		return nil
	}
	return defs
}

// validateEvals checks that the eval definitions satisfy all constraints:
//   - each ID is non-empty (already enforced by extractEvalsFromFrontmatter)
//   - each question is non-empty (already enforced above)
//   - IDs are unique within the list
//
// Returns the first validation error encountered.
func validateEvals(evals []EvalDefinition) error {
	if len(evals) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(evals))
	for _, e := range evals {
		if e.ID == "" {
			return errors.New("evals: each evaluation must have a non-empty id")
		}
		if e.Question == "" {
			return fmt.Errorf("evals: evaluation %q must have a non-empty question", e.ID)
		}
		if _, exists := seen[e.ID]; exists {
			return fmt.Errorf("evals: duplicate evaluation id %q — all ids must be unique", e.ID)
		}
		seen[e.ID] = struct{}{}
	}
	return nil
}

// buildEvalSpecJSON serializes the eval definitions to a compact JSON array suitable
// for embedding in a YAML step env var.
// Uses encoding/json for correct escaping of special characters.
func buildEvalSpecJSON(evals []EvalDefinition) string {
	b, err := json.Marshal(evals)
	if err != nil {
		evalsLog.Printf("buildEvalSpecJSON: marshal error: %v", err)
		return "[]"
	}
	return string(b)
}

// getEvalEngineID returns the effective engine ID for the eval job.
// Defaults to "claude" (same as detection) and normalises Pi → Copilot.
func (c *Compiler) getEvalEngineID(data *WorkflowData) string {
	engineID := data.AI
	if engineID == "" && data.EngineConfig != nil && data.EngineConfig.ID != "" {
		engineID = data.EngineConfig.ID
	}
	if engineID == "" {
		engineID = "claude"
	}
	// Pi is not supported in eval; normalise to Copilot.
	if engineID == "pi" {
		return "copilot"
	}
	return engineID
}

// buildEvalJob creates the eval job that runs after the agent (and detection, if present)
// to execute all declared BinEval questions via the configured agentic engine inside AWF,
// and uploads the results artifact.
// Returns nil when no evals are declared.
func (c *Compiler) buildEvalJob(data *WorkflowData) (*Job, error) {
	if len(data.Evals) == 0 {
		evalsLog.Print("No evals declared, skipping eval job")
		return nil, nil
	}

	evalsLog.Printf("Building eval job for %d evaluation(s)", len(data.Evals))

	var steps []string

	// Setup action (same as detection job — sets up runtime tools).
	setupActionRef := c.resolveActionReference("./actions/setup", data)
	if setupActionRef != "" || c.actionMode.IsScript() {
		steps = append(steps, c.generateCheckoutActionsFolder(data)...)
		evalTraceID := fmt.Sprintf("${{ needs.%s.outputs.setup-trace-id }}", constants.ActivationJobName)
		evalParentSpanID := setupParentSpanNeedsExpr(constants.ActivationJobName)
		steps = append(steps, c.generateSetupStep(data, setupActionRef, SetupActionDestination, false, evalTraceID, evalParentSpanID)...)
	}

	// Download agent output artifact for context (prompt, agent_output.json).
	agentArtifactPrefix := artifactPrefixExprForAgentDownstreamJob(data)
	steps = append(steps, buildAgentOutputDownloadStepsForEval(agentArtifactPrefix, c.getActionPin)...)

	// Clean stale firewall files from the agent artifact download (same as detection job).
	steps = append(steps, c.buildCleanFirewallDirsStep()...)

	// Pull AWF container images so the eval engine runs inside AWF.
	steps = append(steps, c.buildPullAWFContainersStep(data)...)

	// Clear MCP config so the eval engine runs without MCP servers.
	steps = append(steps, buildClearMCPConfigForEvalStep()...)

	// Setup eval: create the prompt file for the engine.
	steps = append(steps, c.buildSetupEvalStep(data)...)

	// Engine installation and execution inside AWF.
	engineSteps, err := c.buildEvalEngineExecutionSteps(data)
	if err != nil {
		evalsLog.Printf("Warning: failed to build eval engine steps: %v", err)
	} else {
		steps = append(steps, engineSteps...)
	}

	// Parse eval results from the engine log and write eval_results.json.
	steps = append(steps, c.buildParseEvalResultsStep(data)...)

	// Upload eval results artifact.
	steps = append(steps, c.buildEvalArtifactUploadStep(data)...)

	// The eval job always depends on the agent and activation jobs.
	needs := []string{string(constants.AgentJobName), string(constants.ActivationJobName)}

	// When threat detection is enabled, eval also depends on detection.
	if data.SafeOutputs != nil && IsDetectionJobEnabled(data.SafeOutputs) {
		needs = append(needs, string(constants.DetectionJobName))
	}

	// Eval job condition: always run whenever the agent job ran (regardless of outcome).
	condition := fmt.Sprintf(evalJobConditionTemplate, constants.AgentJobName)

	return &Job{
		Name:    string(constants.EvalJobName),
		Needs:   needs,
		If:      condition,
		RunsOn:  "runs-on: ubuntu-latest",
		Outputs: nil,
		Steps:   steps,
	}, nil
}

// buildAgentOutputDownloadStepsForEval creates steps to download the agent output
// artifact into the evals working directory.
func buildAgentOutputDownloadStepsForEval(artifactPrefix string, pinAction func(string) string) []string {
	downloadAction := pinAction("actions/download-artifact")
	return []string{
		"      - name: Download agent output artifact\n",
		fmt.Sprintf("        uses: %s\n", downloadAction),
		"        with:\n",
		fmt.Sprintf("          name: %s%s\n", artifactPrefix, constants.AgentArtifactName),
		fmt.Sprintf("          path: %s\n", evalsWorkDir),
		"          merge-multiple: true\n",
		"        continue-on-error: true\n",
	}
}

// buildClearMCPConfigForEvalStep creates a step that removes MCP configuration files
// so the eval engine runs without any MCP servers.
func buildClearMCPConfigForEvalStep() []string {
	return []string{
		"      - name: Clear MCP config for eval\n",
		"        run: |\n",
		"          rm -f \"${RUNNER_TEMP}/gh-aw/mcp-config/mcp-servers.json\"\n",
		"          rm -f \"$HOME/.copilot/mcp-config.json\"\n",
	}
}

// buildSetupEvalStep generates the github-script step that calls setup_eval.cjs,
// which writes the eval prompt to /tmp/gh-aw/aw-prompts/prompt.txt.
func (c *Compiler) buildSetupEvalStep(data *WorkflowData) []string {
	specJSON := buildEvalSpecJSON(data.Evals)
	escapedSpec := strings.ReplaceAll(specJSON, "'", "''")

	return []string{
		"      - name: Setup BinEval prompt\n",
		"        id: setup-eval\n",
		fmt.Sprintf("        uses: %s\n", getCachedActionPin("actions/github-script", data)),
		"        env:\n",
		fmt.Sprintf("          GH_AW_EVAL_SPEC: '%s'\n", escapedSpec),
		fmt.Sprintf("          GH_AW_EVAL_WORK_DIR: %s\n", evalsWorkDir),
		"        with:\n",
		"          script: |\n",
		"            const { setupGlobals } = require('" + SetupActionDestination + "/setup_globals.cjs');\n",
		"            setupGlobals(core, github, context, exec, io, getOctokit);\n",
		"            const { main } = require('" + SetupActionDestination + "/setup_eval.cjs');\n",
		"            await main();\n",
	}
}

// buildEvalEngineExecutionSteps generates the engine installation and execution steps
// for the eval job. The engine runs inside AWF with no MCP servers and limited network
// access (only the inference API), similar to the inline threat detection path.
func (c *Compiler) buildEvalEngineExecutionSteps(data *WorkflowData) ([]string, error) {
	engineSetting := c.getEvalEngineID(data)

	engine, err := c.getAgenticEngine(engineSetting)
	if err != nil {
		return nil, fmt.Errorf("eval engine %q not found: %w", engineSetting, err)
	}

	// Build eval engine config: inherit from the main engine config but apply
	// eval-specific overrides (credits cap, no MaxTurns/Concurrency).
	evalEngineConfig := data.EngineConfig
	if evalEngineConfig == nil {
		evalEngineConfig = &EngineConfig{ID: engineSetting}
	} else {
		evalEngineConfig = &EngineConfig{
			ID:            evalEngineConfig.ID,
			Model:         evalEngineConfig.Model,
			Version:       evalEngineConfig.Version,
			Env:           evalEngineConfig.Env,
			Config:        evalEngineConfig.Config,
			Args:          evalEngineConfig.Args,
			APITarget:     evalEngineConfig.APITarget,
			HarnessScript: evalEngineConfig.HarnessScript,
			Driver:        evalEngineConfig.Driver,
		}
	}
	if evalEngineConfig.ID == "" {
		evalEngineConfig.ID = engineSetting
	}

	// Apply eval AI credits budget (smaller than detection: binary questions only).
	evalEngineConfig.MaxAICredits = constants.DefaultEvalMaxAICredits

	// Apply detection default model when no model is explicitly configured; eval
	// questions are lightweight yes/no tasks so the detection model is appropriate.
	if evalEngineConfig.Model == "" {
		if envModel := compilerenv.ResolveDefaultDetectionModel(""); envModel != "" {
			evalEngineConfig.Model = envModel
		} else if engineModel := engine.GetDefaultDetectionModel(); engineModel != "" {
			evalEngineConfig.Model = engineModel
		}
	}

	// Normalise Pi model to bare model ID for Copilot CLI compatibility.
	if engineSetting == "copilot" && data.AI == "pi" {
		evalEngineConfig.Model = extractPiModelID(evalEngineConfig.Model)
	}

	// Build minimal WorkflowData for the eval engine run. Mirrors the pattern used
	// by buildDetectionEngineExecutionStep: AWF sandbox, no MCP, no safe outputs,
	// minimal network (only the inference API).
	evalData := &WorkflowData{
		Tools: map[string]any{
			"bash": []any{"*"},
		},
		SafeOutputs:  nil,
		EngineConfig: evalEngineConfig,
		AI:           engineSetting,
		Features:     data.Features,
		Permissions:  data.Permissions,
		// IsDetectionRun: reuse detection semantics for domain allow-listing and credits.
		IsDetectionRun: true,
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

	// Install the engine (eval job runs on a fresh runner).
	installSteps := engine.GetInstallationSteps(evalData)

	// Ensure node is on PATH when the engine needs a JS harness.
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

	// Codex needs MCP gateway bootstrap for OpenAI proxy provider configuration.
	if engine.GetID() == "codex" {
		var mcpSetup strings.Builder
		if err := c.generateMCPSetup(&mcpSetup, evalData.Tools, engine, evalData); err == nil {
			for line := range strings.SplitSeq(mcpSetup.String(), "\n") {
				if line != "" {
					steps = append(steps, line+"\n")
				}
			}
		} else {
			evalsLog.Printf("Failed to generate MCP setup for Codex eval; OpenAI proxy config may be incomplete: %v", err)
		}
	}

	executionSteps := engine.GetExecutionSteps(evalData, constants.EvalLogPath)
	for _, step := range executionSteps {
		for i, line := range step {
			// Prefix step IDs with "eval_" to avoid conflicts.
			prefixed := strings.Replace(line, "id: agentic_execution", "id: eval_agentic_execution", 1)
			steps = append(steps, prefixed+"\n")
			// Inject the if condition and continue-on-error after the first line (- name:).
			// continue-on-error: true ensures infrastructure failures don't fail the eval job;
			// the parse step uses if: always() and handles missing logs gracefully.
			if i == 0 {
				steps = append(steps, fmt.Sprintf("        if: %s\n", evalStepCondition))
				steps = append(steps, "        continue-on-error: true\n")
			}
		}
	}

	return steps, nil
}

// buildParseEvalResultsStep generates the github-script step that calls
// parse_eval_results.cjs to extract EVAL_RESULT:{...} from the engine log,
// write eval_results.json, and set job outputs.
func (c *Compiler) buildParseEvalResultsStep(data *WorkflowData) []string {
	return []string{
		"      - name: Parse BinEval results\n",
		"        id: parse-eval-results\n",
		"        if: always()\n",
		fmt.Sprintf("        uses: %s\n", getCachedActionPin("actions/github-script", data)),
		"        env:\n",
		fmt.Sprintf("          GH_AW_EVAL_WORK_DIR: %s\n", evalsWorkDir),
		"        with:\n",
		"          script: |\n",
		"            const { setupGlobals } = require('" + SetupActionDestination + "/setup_globals.cjs');\n",
		"            setupGlobals(core, github, context, exec, io, getOctokit);\n",
		"            const { main } = require('" + SetupActionDestination + "/parse_eval_results.cjs');\n",
		"            await main();\n",
	}
}

// buildEvalArtifactUploadStep uploads the eval results as a workflow artifact.
func (c *Compiler) buildEvalArtifactUploadStep(data *WorkflowData) []string {
	uploadAction := c.getActionPin("actions/upload-artifact")
	artifactName := evalArtifactUploadName(data)
	return []string{
		"      - name: Upload eval results artifact\n",
		"        if: always()\n",
		fmt.Sprintf("        uses: %s\n", uploadAction),
		"        with:\n",
		fmt.Sprintf("          name: %s\n", artifactName),
		fmt.Sprintf("          path: %s\n", evalsWorkDir),
		"          if-no-files-found: ignore\n",
		"          retention-days: 30\n",
	}
}

// evalArtifactUploadName returns the artifact name for the eval results.
// For workflow_call, the prefix expression is prepended at runtime.
func evalArtifactUploadName(data *WorkflowData) string {
	if data == nil {
		return constants.EvalArtifactName
	}
	sanitizedID := SanitizeWorkflowIDForCacheKey(data.WorkflowID)
	if strings.Contains(data.On, "workflow_call") {
		return fmt.Sprintf("${{ needs.%s.outputs.%s }}%s-%s",
			constants.ActivationJobName,
			constants.ArtifactPrefixOutputName,
			sanitizedID,
			constants.EvalArtifactName,
		)
	}
	if sanitizedID != "" {
		return fmt.Sprintf("%s-%s", sanitizedID, constants.EvalArtifactName)
	}
	return constants.EvalArtifactName
}

// buildAndAddEvalJob validates evals, builds the eval job, and registers it with the
// job manager.  It is a no-op when no evals are declared.
func (c *Compiler) buildAndAddEvalJob(data *WorkflowData) error {
	if len(data.Evals) == 0 {
		return nil
	}
	if err := validateEvals(data.Evals); err != nil {
		return err
	}
	evalJob, err := c.buildEvalJob(data)
	if err != nil {
		return err
	}
	if evalJob == nil {
		return nil
	}
	if err := c.jobManager.AddJob(evalJob); err != nil {
		return fmt.Errorf("failed to add eval job: %w", err)
	}
	evalsLog.Print("Added eval job")
	return nil
}
