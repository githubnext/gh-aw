// Package workflow - BinEval evaluation step builders.
package workflow

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/github/gh-aw/pkg/constants"
)

// defaultEvalsModel is the default LLM used for evaluations when no engine-config is provided.
const defaultEvalsModel = "openai/gpt-4o-mini"

// defaultEvalsModelEndpoint is the GitHub Models inference endpoint.
const defaultEvalsModelEndpoint = "https://models.github.ai/inference/chat/completions"

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
// BinEval questions and writes results to /tmp/gh-aw/evals/evals.jsonl.
//
// The harness calls the GitHub Models inference API for each question. The model
// is configurable via engine-config; the default is openai/gpt-4o-mini.
//
// Response format: any reasoning text followed by a final YES or NO answer on its own
// line. The harness captures reasoning (everything before the final YES/NO) and the
// binary answer separately.
func (c *Compiler) buildEvalsHarnessStep(data *WorkflowData) []string {
	// Serialize questions as JSON for the env var.
	type evalQ struct {
		ID       string `json:"id"`
		Question string `json:"question"`
	}
	questions := make([]evalQ, len(data.Evals.Questions))
	for i, q := range data.Evals.Questions {
		questions[i] = evalQ(q)
	}
	questionsJSON, _ := json.Marshal(questions)

	// Resolve model: use engine-config model if provided, else default.
	model := defaultEvalsModel
	endpoint := defaultEvalsModelEndpoint
	if data.Evals.EngineConfig != nil && data.Evals.EngineConfig.Model != "" {
		model = data.Evals.EngineConfig.Model
	}
	if data.Evals.EngineConfig != nil && data.Evals.EngineConfig.APITarget != "" {
		endpoint = data.Evals.EngineConfig.APITarget
	}

	harnessScript := buildEvalsHarnessScript(endpoint)

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
	fmt.Fprintf(&step, "          GH_AW_EVALS_ENDPOINT: '%s'\n", endpoint)
	step.WriteString("        with:\n")
	step.WriteString("          script: |\n")
	// Indent the script body.
	for line := range strings.SplitSeq(harnessScript, "\n") {
		fmt.Fprintf(&step, "            %s\n", line)
	}

	return []string{step.String()}
}

// buildEvalsHarnessScript returns the JavaScript body that drives the evaluation harness.
// The script is injected into the github-script action's `script:` field.
func buildEvalsHarnessScript(endpoint string) string {
	return `const fs = require('fs');
const path = require('path');

const questionsEnv = process.env.GH_AW_EVALS_QUESTIONS || '[]';
const model = process.env.GH_AW_EVALS_MODEL || '` + defaultEvalsModel + `';
const evalEndpoint = process.env.GH_AW_EVALS_ENDPOINT || '` + endpoint + `';
const token = process.env.GITHUB_TOKEN;

let questions;
try {
  questions = JSON.parse(questionsEnv);
} catch (e) {
  core.setFailed('Failed to parse GH_AW_EVALS_QUESTIONS: ' + e.message);
  return;
}

// Read agent context files.
const agentOutputPath = '/tmp/gh-aw/evals/agent_output.json';
const promptPath = '/tmp/gh-aw/evals/prompt.txt';
let agentContext = '';
try {
  const agentOutput = JSON.parse(fs.readFileSync(agentOutputPath, 'utf8') || '{}');
  agentContext += 'Agent output:\n' + JSON.stringify(agentOutput, null, 2) + '\n\n';
} catch (_) {}
try {
  agentContext += 'Agent prompt:\n' + fs.readFileSync(promptPath, 'utf8') + '\n\n';
} catch (_) {}

// Evaluate each question.
const results = [];
let anyError = false;

for (const q of questions) {
  core.info('Evaluating: ' + q.id + ' — ' + q.question);
  const userMessage = agentContext +
    'Evaluate the following question about the agent\'s execution:\n' +
    q.question + '\n\n' +
    'Instructions:\n' +
    '- You may include brief reasoning before your final answer.\n' +
    '- Your final line must be exactly YES or NO (case-insensitive).\n' +
    '- YES means the statement is true / the condition is met.\n' +
    '- NO means the statement is false / the condition is not met.';

  let passed = false;
  let rationale = '';

  try {
    const response = await fetch(evalEndpoint, {
      method: 'POST',
      headers: {
        'Authorization': 'Bearer ' + token,
        'Content-Type': 'application/json',
      },
      body: JSON.stringify({
        model: model,
        messages: [{ role: 'user', content: userMessage }],
        temperature: 0,
        max_tokens: 512,
      }),
    });

    if (!response.ok) {
      const errText = await response.text();
      throw new Error('API error ' + response.status + ': ' + errText);
    }

    const data = await response.json();
    const content = (data.choices?.[0]?.message?.content || '').trim();

    // Parse: reasoning (optional) followed by YES or NO on the last non-empty line.
    const lines = content.split('\n').filter(l => l.trim() !== '');
    const lastLine = lines[lines.length - 1]?.trim().toUpperCase() || '';
    if (lastLine === 'YES' || lastLine.endsWith('YES')) {
      passed = true;
    } else if (lastLine === 'NO' || lastLine.endsWith('NO')) {
      passed = false;
    } else {
      // Cannot determine answer; default to false and record full content.
      core.warning('Could not determine YES/NO answer for eval "' + q.id + '". Raw response: ' + content);
      passed = false;
    }
    // Rationale is everything before the final answer line.
    rationale = lines.slice(0, -1).join(' ').trim();
    if (!rationale) rationale = content;

    core.info('  Result: ' + (passed ? 'YES' : 'NO') + (rationale ? ' | ' + rationale.slice(0, 120) : ''));
  } catch (err) {
    core.error('Failed to evaluate "' + q.id + '": ' + err.message);
    rationale = 'Error: ' + err.message;
    anyError = true;
  }

  results.push({ id: q.id, passed, rationale, model });
}

// Write JSONL results.
const outDir = '/tmp/gh-aw/evals';
fs.mkdirSync(outDir, { recursive: true });
const jsonlPath = path.join(outDir, 'evals.jsonl');
const jsonlContent = results.map(r => JSON.stringify(r)).join('\n') + '\n';
fs.writeFileSync(jsonlPath, jsonlContent, 'utf8');
core.info('Wrote ' + results.length + ' eval results to ' + jsonlPath);

// Set outputs.
const passed = results.filter(r => r.passed).length;
const failed = results.filter(r => !r.passed).length;
core.setOutput('total', results.length);
core.setOutput('passed', passed);
core.setOutput('failed', failed);
core.setOutput('pass_rate', results.length > 0 ? (passed / results.length).toFixed(2) : '0.00');

if (anyError) {
  core.warning('One or more evaluations encountered errors. Check the log for details.');
}`
}

// buildEvalsStepSummaryStep creates a step that renders the evaluation results as a
// GitHub Actions step summary in Markdown.
func (c *Compiler) buildEvalsStepSummaryStep(data *WorkflowData) []string {
	var step strings.Builder
	step.WriteString("      - name: Render evaluation summary\n")
	step.WriteString("        id: evals_summary\n")
	step.WriteString("        if: always()\n")
	step.WriteString("        continue-on-error: true\n")
	fmt.Fprintf(&step, "        uses: %s\n", getCachedActionPin("actions/github-script", data))
	step.WriteString("        with:\n")
	step.WriteString("          script: |\n")
	step.WriteString("            const fs = require('fs');\n")
	step.WriteString("            const jsonlPath = '/tmp/gh-aw/evals/evals.jsonl';\n")
	step.WriteString("            if (!fs.existsSync(jsonlPath)) {\n")
	step.WriteString("              core.info('No evals results file found, skipping summary.');\n")
	step.WriteString("              return;\n")
	step.WriteString("            }\n")
	step.WriteString("            const lines = fs.readFileSync(jsonlPath, 'utf8').split('\\n').filter(l => l.trim());\n")
	step.WriteString("            const results = lines.map(l => JSON.parse(l));\n")
	step.WriteString("            const passed = results.filter(r => r.passed).length;\n")
	step.WriteString("            const total = results.length;\n")
	step.WriteString("            const passRate = total > 0 ? Math.round((passed / total) * 100) : 0;\n")
	step.WriteString("            let md = '## 🔬 BinEval Evaluation Results\\n\\n';\n")
	step.WriteString("            md += `**${passed}/${total} passed** (${passRate}% pass rate)\\n\\n`;\n")
	step.WriteString("            md += '| Question ID | Result | Rationale |\\n';\n")
	step.WriteString("            md += '|---|---|---|\\n';\n")
	step.WriteString("            for (const r of results) {\n")
	step.WriteString("              const icon = r.passed ? '✅ YES' : '❌ NO';\n")
	step.WriteString("              const rationale = (r.rationale || '').replace(/[|\\n]/g, ' ').slice(0, 200);\n")
	step.WriteString("              md += `| ${r.id} | ${icon} | ${rationale} |\\n`;\n")
	step.WriteString("            }\n")
	step.WriteString("            await core.summary.addRaw(md).write();\n")
	step.WriteString("            core.info('Evaluation summary written.');\n")

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
