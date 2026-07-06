// Package workflow - BinEval evaluation step builders.
package workflow

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/github/gh-aw/pkg/constants"
)

// defaultEvalsModel is the default LLM model hint used for evaluations when no model is specified.
// "small" lets the runtime pick an appropriate small/cheap model.
const defaultEvalsModel = "small"

// buildEvalsJobSteps assembles all evaluation harness steps for the evals job.
func (c *Compiler) buildEvalsJobSteps(data *WorkflowData) []string {
	var steps []string

	steps = append(steps, "      # --- BinEval Evaluations ---\n")

	// Step 1: Prepare evaluation context files.
	steps = append(steps, c.buildEvalsPrepareFilesStep()...)

	// Step 2: Run the evaluation harness (one LLM call per question).
	steps = append(steps, c.buildEvalsHarnessStep(data)...)

	// Step 3: Render step summary from JSONL results.
	steps = append(steps, c.buildEvalsStepSummaryStep(data)...)

	return steps
}

// buildEvalsPrepareFilesStep creates a step that copies agent output to the
// evaluation working directory.
func (c *Compiler) buildEvalsPrepareFilesStep() []string {
	return []string{
		"      - name: Prepare evaluation context\n",
		"        run: |\n",
		"          mkdir -p /tmp/gh-aw/evals\n",
		"          cp /tmp/gh-aw/agent_output.json /tmp/gh-aw/evals/agent_output.json 2>/dev/null || echo '{}' > /tmp/gh-aw/evals/agent_output.json\n",
		"          cp /tmp/gh-aw/aw-prompts/prompt.txt /tmp/gh-aw/evals/prompt.txt 2>/dev/null || true\n",
		"          echo \"Evaluation context prepared:\"\n",
		"          ls -la /tmp/gh-aw/evals/ 2>/dev/null || true\n",
	}
}

// buildEvalsHarnessStep creates the GitHub Actions script step that runs all declared
// BinEval questions by requiring run_evals_harness.cjs and writing results to
// /tmp/gh-aw/evals/evals.jsonl.
//
// The model is configurable via the evals.model field (or per-question model override);
// the default alias is "small".
func (c *Compiler) buildEvalsHarnessStep(data *WorkflowData) []string {
	// Serialize questions as JSON for the env var.
	// evalQ mirrors EvalDefinition for JSON output; Model is omitempty so questions
	// without a per-question override do not include the field in the serialized JSON.
	type evalQ struct {
		ID       string `json:"id"`
		Question string `json:"question"`
		// Model is an optional per-question model override (omitted when empty).
		Model string `json:"model,omitempty"`
	}
	questions := make([]evalQ, len(data.Evals.Questions))
	for i, q := range data.Evals.Questions {
		questions[i] = evalQ(q)
	}
	questionsJSON, _ := json.Marshal(questions)

	// Resolve default model: use evals.model if provided, else the compiler default.
	model := defaultEvalsModel
	if data.Evals.Model != "" {
		model = data.Evals.Model
	}

	script := `const { setupGlobals } = require('` + SetupActionDestination + `/setup_globals.cjs');
setupGlobals(core, github, context, exec, io, getOctokit);
const { main } = require('` + SetupActionDestination + `/run_evals_harness.cjs');
await main();`

	var step strings.Builder
	step.WriteString("      - name: Run BinEval evaluation harness\n")
	step.WriteString("        id: evals_harness\n")
	step.WriteString("        if: always()\n")
	step.WriteString("        continue-on-error: true\n")
	fmt.Fprintf(&step, "        uses: %s\n", getCachedActionPin("actions/github-script", data))
	step.WriteString("        env:\n")
	step.WriteString("          GITHUB_TOKEN: ${{ github.token }}\n")
	step.WriteString("          GITHUB_RUN_ID: ${{ github.run_id }}\n")
	fmt.Fprintf(&step, "          GH_AW_EVALS_QUESTIONS: '%s'\n", escapeYAMLSingleQuote(string(questionsJSON)))
	fmt.Fprintf(&step, "          GH_AW_EVALS_MODEL: '%s'\n", model)
	step.WriteString("        with:\n")
	step.WriteString("          script: |\n")
	for line := range strings.SplitSeq(script, "\n") {
		fmt.Fprintf(&step, "            %s\n", line)
	}

	return []string{step.String()}
}

// buildEvalsStepSummaryStep creates a step that renders the evaluation results as a
// GitHub Actions step summary in Markdown by requiring render_evals_summary.cjs.
func (c *Compiler) buildEvalsStepSummaryStep(data *WorkflowData) []string {
	script := `const { setupGlobals } = require('` + SetupActionDestination + `/setup_globals.cjs');
setupGlobals(core, github, context, exec, io, getOctokit);
const { main } = require('` + SetupActionDestination + `/render_evals_summary.cjs');
await main();`

	var step strings.Builder
	step.WriteString("      - name: Render evaluation summary\n")
	step.WriteString("        id: evals_summary\n")
	step.WriteString("        if: always()\n")
	step.WriteString("        continue-on-error: true\n")
	fmt.Fprintf(&step, "        uses: %s\n", getCachedActionPin("actions/github-script", data))
	step.WriteString("        with:\n")
	step.WriteString("          script: |\n")
	for line := range strings.SplitSeq(script, "\n") {
		fmt.Fprintf(&step, "            %s\n", line)
	}

	return []string{step.String()}
}

// buildEvalsArtifactUploadStep creates a step that uploads the evals JSONL file as the
// `eval` (or `<prefix>eval`) artifact.
func (c *Compiler) buildEvalsArtifactUploadStep(data *WorkflowData, artifactPrefix string) []string {
	artifactName := artifactPrefix + constants.EvalsArtifactName

	var step strings.Builder
	step.WriteString("      - name: Upload eval artifact\n")
	step.WriteString("        id: upload_eval_artifact\n")
	step.WriteString("        if: always()\n")
	step.WriteString("        continue-on-error: true\n")
	fmt.Fprintf(&step, "        uses: %s\n", c.getActionPin("actions/upload-artifact"))
	step.WriteString("        with:\n")
	fmt.Fprintf(&step, "          name: %s\n", artifactName)
	fmt.Fprintf(&step, "          path: %s\n", "/tmp/gh-aw/evals/"+constants.EvalsResultFilename)
	step.WriteString("          if-no-files-found: ignore\n")

	return []string{step.String()}
}

// buildEvalsBranchPersistStep creates a step that commits the evaluation results to the
// evals/<workflow-id> git branch so results accumulate across runs for historical comparison.
func (c *Compiler) buildEvalsBranchPersistStep(data *WorkflowData) []string {
	branchName := evalsBranchName(data.WorkflowID)
	resultsFilename := constants.EvalsResultFilename

	var step strings.Builder
	step.WriteString("      - name: Persist evals to git branch\n")
	step.WriteString("        id: persist_evals\n")
	step.WriteString("        if: always()\n")
	step.WriteString("        continue-on-error: true\n")
	step.WriteString("        env:\n")
	step.WriteString("          GH_TOKEN: ${{ github.token }}\n")
	fmt.Fprintf(&step, "          GH_AW_EVALS_BRANCH: %s\n", branchName)
	fmt.Fprintf(&step, "          GH_AW_EVALS_FILE: %s\n", resultsFilename)
	step.WriteString("          GITHUB_RUN_ID: ${{ github.run_id }}\n")
	step.WriteString("          GITHUB_REPOSITORY: ${{ github.repository }}\n")
	step.WriteString("          GITHUB_SERVER_URL: ${{ github.server_url }}\n")
	step.WriteString("        run: |\n")
	step.WriteString("          set -euo pipefail\n")
	step.WriteString("          RESULTS_FILE=\"/tmp/gh-aw/evals/${GH_AW_EVALS_FILE}\"\n")
	step.WriteString("          if [ ! -f \"${RESULTS_FILE}\" ]; then\n")
	step.WriteString("            echo \"No eval results file found, skipping branch persistence.\"\n")
	step.WriteString("            exit 0\n")
	step.WriteString("          fi\n")
	step.WriteString("          git config --global user.email \"github-actions[bot]@users.noreply.github.com\"\n")
	step.WriteString("          git config --global user.name \"github-actions[bot]\"\n")
	step.WriteString("          WORK_DIR=$(mktemp -d)\n")
	step.WriteString("          REPO_URL=\"${GITHUB_SERVER_URL}/${GITHUB_REPOSITORY}.git\"\n")
	step.WriteString("          cd \"${WORK_DIR}\"\n")
	step.WriteString("          if git ls-remote --exit-code \"${REPO_URL}\" \"refs/heads/${GH_AW_EVALS_BRANCH}\" >/dev/null 2>&1; then\n")
	step.WriteString("            git clone --depth=1 --branch \"${GH_AW_EVALS_BRANCH}\" \"${REPO_URL}\" .\n")
	step.WriteString("          else\n")
	step.WriteString("            git init\n")
	step.WriteString("            git remote add origin \"${REPO_URL}\"\n")
	step.WriteString("            git checkout --orphan \"${GH_AW_EVALS_BRANCH}\"\n")
	step.WriteString("          fi\n")
	step.WriteString("          cp \"${RESULTS_FILE}\" \"${GH_AW_EVALS_FILE}\"\n")
	step.WriteString("          git add \"${GH_AW_EVALS_FILE}\"\n")
	step.WriteString("          git commit -m \"evals: run ${GITHUB_RUN_ID}\" || echo \"Nothing to commit\"\n")
	step.WriteString("          git push origin \"HEAD:refs/heads/${GH_AW_EVALS_BRANCH}\" || echo \"Push failed (branch may be protected)\"\n")
	step.WriteString("          echo \"Evals results persisted to branch ${GH_AW_EVALS_BRANCH}\"\n")

	return []string{step.String()}
}

// escapeYAMLSingleQuote escapes a string for use inside YAML single-quoted scalars
// by doubling any single-quote characters.
func escapeYAMLSingleQuote(s string) string {
	return strings.ReplaceAll(s, "'", "''")
}
