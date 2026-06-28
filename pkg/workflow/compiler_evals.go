// Package workflow - BinEval evaluation job assembler and frontmatter extraction.
package workflow

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/github/gh-aw/pkg/constants"
	"github.com/github/gh-aw/pkg/logger"
)

var evalsLog = logger.New("workflow:compiler_evals")

// evalsWorkDir is the runtime directory where eval inputs and outputs are stored.
const evalsWorkDir = "/tmp/gh-aw/evals"

// evalDefaultModel is the default GitHub Models choice for evals when no model is configured.
// Evals deliberately use a small, low-latency, cost-efficient model because each
// question is a lightweight yes/no classification task rather than open-ended generation.
const evalDefaultModel = "gpt-4o-mini"

const evalJobConditionTemplate = "always() && !cancelled() && needs.%s.result != 'skipped'"

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

// buildEvalJob creates the eval job that runs after the agent (and detection, if present)
// to execute all declared BinEval questions and upload the results artifact.
// Returns nil when no evals are declared.
func (c *Compiler) buildEvalJob(data *WorkflowData) (*Job, error) {
	if len(data.Evals) == 0 {
		evalsLog.Print("No evals declared, skipping eval job")
		return nil, nil
	}

	evalsLog.Printf("Building eval job for %d evaluation(s)", len(data.Evals))

	var steps []string

	// Setup action (same as detection job — sets up runtime tools)
	setupActionRef := c.resolveActionReference("./actions/setup", data)
	if setupActionRef != "" || c.actionMode.IsScript() {
		steps = append(steps, c.generateCheckoutActionsFolder(data)...)
		// Eval job shares the agent trace ID for cohesive OTLP traces.
		evalTraceID := fmt.Sprintf("${{ needs.%s.outputs.setup-trace-id }}", constants.ActivationJobName)
		evalParentSpanID := setupParentSpanNeedsExpr(constants.ActivationJobName)
		steps = append(steps, c.generateSetupStep(data, setupActionRef, SetupActionDestination, false, evalTraceID, evalParentSpanID)...)
	}

	// Download agent output artifact for context (prompt, agent_output.json, patches).
	agentArtifactPrefix := artifactPrefixExprForAgentDownstreamJob(data)
	steps = append(steps, buildAgentOutputDownloadStepsForEval(agentArtifactPrefix, c.getActionPin)...)

	// Run the eval harness via github-script.
	steps = append(steps, c.buildEvalHarnessStep(data)...)

	// Upload eval results artifact.
	steps = append(steps, c.buildEvalArtifactUploadStep(data)...)

	// The eval job always depends on the agent and activation jobs.
	needs := []string{string(constants.AgentJobName), string(constants.ActivationJobName)}

	// When threat detection is enabled, eval also depends on detection so that
	// detection conclusions are visible in the artifact download path.
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
// artifact into the evals working directory.  It mirrors the detection job download
// but writes to evalsWorkDir instead of the threat-detection directory.
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

// buildEvalHarnessStep generates the github-script step that invokes eval_harness.cjs.
func (c *Compiler) buildEvalHarnessStep(data *WorkflowData) []string {
	specJSON := buildEvalSpecJSON(data.Evals)
	// Escape single quotes for YAML single-quoted scalar embedding (YAML §7.3.3).
	escapedSpec := strings.ReplaceAll(specJSON, "'", "''")

	return []string{
		"      - name: Run BinEval evaluations\n",
		"        id: run-evals\n",
		fmt.Sprintf("        uses: %s\n", getCachedActionPin("actions/github-script", data)),
		"        env:\n",
		fmt.Sprintf("          GH_AW_EVAL_SPEC: '%s'\n", escapedSpec),
		fmt.Sprintf("          GH_AW_EVAL_WORK_DIR: %s\n", evalsWorkDir),
		fmt.Sprintf("          GH_AW_EVAL_MODEL: %s\n", evalDefaultModel),
		"        with:\n",
		"          script: |\n",
		"            const { setupGlobals } = require('" + SetupActionDestination + "/setup_globals.cjs');\n",
		"            setupGlobals(core, github, context, exec, io, getOctokit);\n",
		"            const { main } = require('" + SetupActionDestination + "/eval_harness.cjs');\n",
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
