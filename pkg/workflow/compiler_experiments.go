package workflow

import (
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strings"

	"github.com/github/gh-aw/pkg/constants"
	"github.com/github/gh-aw/pkg/logger"
)

var experimentsLog = logger.New("workflow:compiler_experiments")

// experimentsCacheDir is the runtime directory where the experiment state JSON is stored.
const experimentsCacheDir = "/tmp/gh-aw/experiments"

// experimentStateFile is the path to the experiment state JSON written by pick_experiment.cjs.
const experimentStateFile = experimentsCacheDir + "/state.json"

// extractExperimentsFromFrontmatter reads the "experiments" map from a raw frontmatter map.
// Each key is an experiment name; each value must be a []string (or []any of strings) of
// variant values.  Invalid entries are silently skipped.
// Experiment names must match [a-zA-Z_][a-zA-Z0-9_]* (identifier style) so they can be used
// as GitHub Actions step output names and in ${{ experiments.<name> }} expressions without
// bracket notation.  Names that do not match are skipped with a warning.
var experimentNamePattern = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*$`)

func extractExperimentsFromFrontmatter(frontmatter map[string]any) map[string][]string {
	raw, ok := frontmatter["experiments"]
	if !ok || raw == nil {
		return nil
	}
	rawMap, ok := raw.(map[string]any)
	if !ok {
		return nil
	}
	result := make(map[string][]string, len(rawMap))
	for name, val := range rawMap {
		if !experimentNamePattern.MatchString(name) {
			experimentsLog.Printf("Skipping experiment %q: name must match [a-zA-Z_][a-zA-Z0-9_]*", name)
			continue
		}
		switch v := val.(type) {
		case []string:
			if len(v) >= 2 {
				result[name] = v
			}
		case []any:
			var variants []string
			for _, item := range v {
				if s, ok := item.(string); ok {
					variants = append(variants, s)
				}
			}
			if len(variants) >= 2 {
				result[name] = variants
			}
		}
	}
	if len(result) == 0 {
		return nil
	}
	return result
}

// generateExperimentSteps creates the steps that pick and upload A/B experiment variants.
//
// Steps generated (only when experiments are declared):
//  1. Restore experiment cache   – actions/cache/restore keyed by workflow ID
//  2. Pick variants              – pick_experiment.cjs (reads/writes state.json, sets step outputs,
//     writes a Markdown step summary); outputs: one per experiment (e.g. "caveman=yes") + "experiments" JSON blob
//  3. Save experiment cache      – actions/cache/save keyed by workflow ID
//  4. Upload experiment artifact – actions/upload-artifact named "experiment"
func (c *Compiler) generateExperimentSteps(data *WorkflowData) []string {
	if len(data.Experiments) == 0 {
		return nil
	}

	experimentNames := sortedExperimentNames(data.Experiments)
	experimentsLog.Printf("Generating experiment steps for %d experiment(s): %v", len(experimentNames), experimentNames)

	cacheKey := "experiments-${{ env.GH_AW_WORKFLOW_ID_SANITIZED }}-${{ github.run_id }}"
	restoreKey := "experiments-${{ env.GH_AW_WORKFLOW_ID_SANITIZED }}-"

	var steps []string

	// ── Step 1: Restore experiment cache ──────────────────────────────────────
	steps = append(steps,
		"      - name: Restore experiment state\n",
		"        id: restore-experiment-cache\n",
		fmt.Sprintf("        uses: %s\n", getActionPin("actions/cache/restore")),
		"        with:\n",
		fmt.Sprintf("          key: %s\n", cacheKey),
		fmt.Sprintf("          restore-keys: %s\n", restoreKey),
		fmt.Sprintf("          path: %s\n", experimentsCacheDir),
	)

	// ── Step 2: Pick experiment variants ──────────────────────────────────────
	// Build the JSON spec: {"feature1":["A","B"],...}
	specJSON := buildExperimentSpecJSON(data.Experiments, experimentNames)

	steps = append(steps,
		"      - name: Pick experiment variants\n",
		"        id: pick-experiment\n",
		fmt.Sprintf("        uses: %s\n", getCachedActionPin("actions/github-script", data)),
		"        env:\n",
		fmt.Sprintf("          GH_AW_EXPERIMENT_SPEC: '%s'\n", strings.ReplaceAll(specJSON, "'", "''")),
		fmt.Sprintf("          GH_AW_EXPERIMENT_STATE_FILE: %s\n", experimentStateFile),
		fmt.Sprintf("          GH_AW_EXPERIMENT_STATE_DIR: %s\n", experimentsCacheDir),
		"        with:\n",
		"          script: |\n",
		"            const { setupGlobals } = require('"+SetupActionDestination+"/setup_globals.cjs');\n",
		"            setupGlobals(core, github, context, exec, io, getOctokit);\n",
		"            const { main } = require('"+SetupActionDestination+"/pick_experiment.cjs');\n",
		"            await main();\n",
	)

	// ── Step 3: Save experiment cache ─────────────────────────────────────────
	steps = append(steps,
		"      - name: Save experiment state\n",
		"        if: always()\n",
		fmt.Sprintf("        uses: %s\n", getActionPin("actions/cache/save")),
		"        with:\n",
		fmt.Sprintf("          key: %s\n", cacheKey),
		fmt.Sprintf("          path: %s\n", experimentsCacheDir),
	)

	// ── Step 4: Upload experiment artifact ────────────────────────────────────
	experimentArtifactName := artifactPrefixExprForActivationJob(data) + constants.ExperimentArtifactName
	steps = append(steps,
		"      - name: Upload experiment artifact\n",
		"        if: always()\n",
		fmt.Sprintf("        uses: %s\n", getActionPin("actions/upload-artifact")),
		"        with:\n",
		fmt.Sprintf("          name: %s\n", experimentArtifactName),
		fmt.Sprintf("          path: %s\n", experimentsCacheDir),
		"          if-no-files-found: ignore\n",
		"          retention-days: 30\n",
	)

	return steps
}

// buildExperimentSpecJSON builds a compact JSON object from the experiments map.
// Uses encoding/json for proper escaping of all special characters.
// Caller is responsible for escaping single quotes (” in YAML) when embedding the
// result in a YAML single-quoted scalar, since JSON string values may contain literal
// single quotes (e.g. "Bob's").
func buildExperimentSpecJSON(experiments map[string][]string, names []string) string {
	// Build JSON manually with encoding/json for individual values to ensure
	// correct escaping of all special characters.  We iterate names (a sorted slice)
	// rather than the map directly to produce deterministic output.
	var sb strings.Builder
	sb.WriteString("{")
	for i, name := range names {
		if i > 0 {
			sb.WriteString(",")
		}
		keyBytes, _ := json.Marshal(name)
		varBytes, _ := json.Marshal(experiments[name])
		sb.Write(keyBytes)
		sb.WriteString(":")
		sb.Write(varBytes)
	}
	sb.WriteString("}")
	return sb.String()
}

// ExperimentExpressionMappings generates ExpressionMapping entries for all declared experiments.
//
// Each mapping maps the env-var name derived from "experiments.NAME"
// (e.g. GH_AW_EXPERIMENTS_CAVEMAN) to the step output expression
// "steps.pick-experiment.outputs.NAME".
//
// Adding these mappings to both expressionMappings and allExpressionMappings ensures:
//   - The "Interpolate variables and render templates" step has
//     GH_AW_EXPERIMENTS_NAME set from the step output, so that interpolate_prompt.cjs
//     can substitute __GH_AW_EXPERIMENTS_NAME__ placeholders BEFORE template rendering.
//   - The "Substitute placeholders" step can replace any remaining __GH_AW_EXPERIMENTS_NAME__
//     occurrences that were produced by the runtime-import mechanism.
func ExperimentExpressionMappings(experiments map[string][]string) []*ExpressionMapping {
	names := sortedExperimentNames(experiments)
	mappings := make([]*ExpressionMapping, 0, len(names))
	for _, name := range names {
		envVar := ExperimentEnvVarName(name) // e.g. GH_AW_EXPERIMENTS_CAVEMAN
		// The step output expression resolves to the variant selected at runtime.
		// The step ID "pick-experiment" is defined by generateExperimentSteps (the step with
		// `id: pick-experiment` in the activation job).
		content := "steps.pick-experiment.outputs." + name // e.g. steps.pick-experiment.outputs.caveman
		original := "${{ experiments." + name + " }}"      // original expression in the markdown

		mappings = append(mappings, &ExpressionMapping{
			Original: original,
			EnvVar:   envVar,
			Content:  content,
		})
	}
	return mappings
}

// sortedExperimentNames returns the experiment names in sorted order for deterministic output.
func sortedExperimentNames(experiments map[string][]string) []string {
	names := make([]string, 0, len(experiments))
	for name := range experiments {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// buildExperimentArtifactDownloadSteps creates a download step for the experiment artifact.
// The artifact is downloaded to experimentsCacheDir so the detection agent can read the
// current variant assignments from state.json.
// prefix must be the artifact prefix expression for a job that directly depends on the
// activation job (i.e. artifactPrefixExprForDownstreamJob).
// The step is a no-op when no experiments are declared.
func buildExperimentArtifactDownloadSteps(prefix string, experiments map[string][]string) []string {
	if len(experiments) == 0 {
		return nil
	}
	artifactName := prefix + constants.ExperimentArtifactName
	return buildArtifactDownloadSteps(ArtifactDownloadConfig{
		ArtifactName: artifactName,
		DownloadPath: experimentsCacheDir + "/",
		StepName:     "Download experiment artifact",
	})
}
