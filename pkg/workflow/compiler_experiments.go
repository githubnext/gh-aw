package workflow

import (
	"encoding/json"
	"fmt"
	"math"
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

// ExperimentsStorageCache uses GitHub Actions cache to persist experiment state.
const ExperimentsStorageCache = "cache"

// ExperimentsStorageRepo uses a git branch (repo-memory) to persist experiment state.
// This is the default because experiment data is valuable and repo storage is more durable.
const ExperimentsStorageRepo = "repo"

// experimentsBranchPrefix is the git branch prefix used when storage: repo is selected.
// Branches are named "experiments/{sanitizedWorkflowID}".
const experimentsBranchPrefix = "experiments"

// experimentsStorageReservedKey is the reserved key in the experiments map that controls storage mode.
const experimentsStorageReservedKey = "storage"

// experimentNamePattern validates experiment names as identifier-style keys.
// Experiment names must match [a-zA-Z_][a-zA-Z0-9_]* so they can be used
// as GitHub Actions step output names and in ${{ experiments.<name> }} expressions without
// bracket notation.  Names that do not match are skipped with a warning.
var experimentNamePattern = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*$`)

// experimentVariantsFromConfigs derives the simple name→variants map from a configs map.
// Returns nil when configs is empty so callers can use len-checks without special-casing.
func experimentVariantsFromConfigs(configs map[string]*ExperimentConfig) map[string][]string {
	if len(configs) == 0 {
		return nil
	}
	result := make(map[string][]string, len(configs))
	for name, cfg := range configs {
		result[name] = cfg.Variants
	}
	return result
}

// extractExperimentConfigsFromFrontmatter reads the "experiments" map and returns
// fully-typed ExperimentConfig objects.  Both the bare-array form and the new object
// form are accepted.
func extractExperimentConfigsFromFrontmatter(frontmatter map[string]any) map[string]*ExperimentConfig {
	raw, ok := frontmatter["experiments"]
	if !ok || raw == nil {
		return nil
	}
	rawMap, ok := raw.(map[string]any)
	if !ok {
		return nil
	}
	result := make(map[string]*ExperimentConfig, len(rawMap))
	for name, val := range rawMap {
		// "storage" is a reserved key that controls persistence mode, not an experiment name.
		if name == experimentsStorageReservedKey {
			continue
		}
		if !experimentNamePattern.MatchString(name) {
			experimentsLog.Printf("Skipping experiment %q: name must match [a-zA-Z_][a-zA-Z0-9_]*", name)
			continue
		}
		cfg := extractOneExperimentConfig(name, val)
		if cfg != nil {
			result[name] = cfg
		}
	}
	if len(result) == 0 {
		return nil
	}
	return result
}

// extractExperimentsStorageFromFrontmatter reads the "storage" key from the experiments
// map and returns the resolved storage mode.  Returns ExperimentsStorageRepo when the
// key is absent or has an unrecognised value.
func extractExperimentsStorageFromFrontmatter(frontmatter map[string]any) string {
	raw, ok := frontmatter["experiments"]
	if !ok || raw == nil {
		return ExperimentsStorageRepo
	}
	rawMap, ok := raw.(map[string]any)
	if !ok {
		return ExperimentsStorageRepo
	}
	if storageRaw, ok := rawMap[experimentsStorageReservedKey]; ok {
		if s, ok := storageRaw.(string); ok {
			switch s {
			case ExperimentsStorageCache, ExperimentsStorageRepo:
				return s
			default:
				experimentsLog.Printf("Unknown experiments storage %q; falling back to %q", s, ExperimentsStorageRepo)
			}
		}
	}
	return ExperimentsStorageRepo
}

// experimentsBranchName returns the git branch name used for repo-based experiment storage.
// Format: "experiments/{sanitizedWorkflowID}"
func experimentsBranchName(workflowID string) string {
	sanitized := SanitizeWorkflowIDForCacheKey(workflowID)
	if sanitized == "" {
		sanitized = "default"
	}
	return experimentsBranchPrefix + "/" + sanitized
}

// extractOneExperimentConfig converts a single raw experiment value into an ExperimentConfig.
// Returns nil when the value is invalid (e.g. fewer than two variants).
func extractOneExperimentConfig(name string, val any) *ExperimentConfig {
	switch v := val.(type) {
	case []string:
		if len(v) >= 2 {
			return &ExperimentConfig{Variants: v}
		}
	case []any:
		var variants []string
		for _, item := range v {
			if s, ok := item.(string); ok {
				variants = append(variants, s)
			}
		}
		if len(variants) >= 2 {
			return &ExperimentConfig{Variants: variants}
		}
	case map[string]any:
		// New object form: extract variants and optional metadata fields.
		cfg := &ExperimentConfig{}
		varRaw, ok := v["variants"]
		if !ok {
			experimentsLog.Printf("Skipping experiment %q: object form requires 'variants' field", name)
			return nil
		}
		switch vv := varRaw.(type) {
		case []string:
			cfg.Variants = vv
		case []any:
			for _, item := range vv {
				if s, ok := item.(string); ok {
					cfg.Variants = append(cfg.Variants, s)
				}
			}
		}
		if len(cfg.Variants) < 2 {
			experimentsLog.Printf("Skipping experiment %q: must have at least 2 variants", name)
			return nil
		}
		if d, ok := v["description"].(string); ok {
			cfg.Description = d
		}
		if m, ok := v["metric"].(string); ok {
			cfg.Metric = m
		}
		if sd, ok := v["start_date"].(string); ok {
			cfg.StartDate = sd
		}
		if ed, ok := v["end_date"].(string); ok {
			cfg.EndDate = ed
		}
		if n, ok := extractIntField(v["issue"]); ok {
			cfg.Issue = n
		}
		if weightRaw, ok := v["weight"]; ok {
			cfg.Weight = extractIntSlice(weightRaw)
		}
		if h, ok := v["hypothesis"].(string); ok {
			cfg.Hypothesis = h
		}
		if smRaw, ok := v["secondary_metrics"]; ok {
			cfg.SecondaryMetrics = extractStringSlice(smRaw)
		}
		if gmRaw, ok := v["guardrail_metrics"]; ok {
			cfg.GuardrailMetrics = extractGuardrailMetrics(gmRaw)
		}
		if n, ok := extractIntField(v["min_samples"]); ok {
			cfg.MinSamples = n
		}
		if at, ok := v["analysis_type"].(string); ok {
			cfg.AnalysisType = at
		}
		if tagsRaw, ok := v["tags"]; ok {
			cfg.Tags = extractStringSlice(tagsRaw)
		}
		if notifyRaw, ok := v["notify"]; ok {
			if notifyMap, ok := notifyRaw.(map[string]any); ok {
				notify := &ExperimentNotify{}
				hasNotify := false
				if n, ok := extractIntField(notifyMap["discussion"]); ok {
					notify.Discussion = n
					hasNotify = true
				}
				if n, ok := extractIntField(notifyMap["issue"]); ok {
					notify.Issue = n
					hasNotify = true
				}
				if hasNotify {
					cfg.Notify = notify
				}
			}
		}
		return cfg
	}
	return nil
}

// extractIntField converts a numeric any value to int.
// Returns (int(value), true) on success; (0, false) when val is nil, not a supported
// numeric type, negative, or out of int range.
// float64 values that are not integral (e.g. 12.9) are rejected.
func extractIntField(val any) (int, bool) {
	switch n := val.(type) {
	case int:
		if n < 0 {
			return 0, false
		}
		return n, true
	case int64:
		if n < 0 || n > math.MaxInt {
			return 0, false
		}
		return int(n), true
	case uint64:
		if n > uint64(math.MaxInt) {
			return 0, false
		}
		return int(n), true
	case float64:
		// Reject non-integral or out-of-range float64 values.
		if n < 0 || n > float64(math.MaxInt) || n != math.Trunc(n) {
			return 0, false
		}
		return int(n), true
	}
	return 0, false
}

// extractStringSlice converts a raw value to a []string, accepting []any of string values.
func extractStringSlice(raw any) []string {
	switch v := raw.(type) {
	case []string:
		return v
	case []any:
		var result []string
		for _, item := range v {
			if s, ok := item.(string); ok {
				result = append(result, s)
			}
		}
		return result
	}
	return nil
}

// extractGuardrailMetrics converts a raw guardrail_metrics value into a []GuardrailMetric.
// Each entry must be a map with "name" and "threshold" string fields.
func extractGuardrailMetrics(raw any) []GuardrailMetric {
	items, ok := raw.([]any)
	if !ok {
		return nil
	}
	var result []GuardrailMetric
	for _, item := range items {
		m, ok := item.(map[string]any)
		if !ok {
			continue
		}
		name, _ := m["name"].(string)
		threshold, _ := m["threshold"].(string)
		if name == "" || threshold == "" {
			continue
		}
		result = append(result, GuardrailMetric{Name: name, Threshold: threshold})
	}
	return result
}

// extractIntSlice converts a raw value to a []int, accepting []any of numeric values.
func extractIntSlice(raw any) []int {
	switch v := raw.(type) {
	case []int:
		return v
	case []any:
		var result []int
		for _, item := range v {
			switch n := item.(type) {
			case int:
				result = append(result, n)
			case int64:
				result = append(result, int(n))
			case uint64:
				result = append(result, int(n))
			case float64:
				result = append(result, int(n))
			}
		}
		return result
	}
	return nil
}

// generateExperimentSteps creates the steps that pick and upload A/B experiment variants.
//
// When storage is "cache" (legacy) the steps are:
//  1. Restore experiment cache   – actions/cache/restore keyed by workflow ID
//  2. Pick variants              – pick_experiment.cjs (reads/writes state.json, sets step outputs,
//     writes a Markdown step summary); outputs: one per experiment (e.g. "caveman=yes") + "experiments" JSON blob
//  3. Save experiment cache      – actions/cache/save keyed by workflow ID
//  4. Upload experiment artifact – actions/upload-artifact named "{workflowID}-experiment"
//
// When storage is "repo" (default) the steps are:
//  1. Restore experiment state from git – load_experiment_state_from_repo.cjs fetches state.json
//     from the "experiments/{sanitizedID}" branch via the GitHub API (read-only; falls back to
//     empty state when the branch/file does not yet exist)
//  2. Pick variants              – same as cache mode
//  3. Upload experiment artifact – same as cache mode (NO cache save; a separate push job commits state)
func (c *Compiler) generateExperimentSteps(data *WorkflowData) []string {
	if len(data.Experiments) == 0 {
		return nil
	}

	experimentNames := sortedExperimentNames(data.Experiments)
	experimentsLog.Printf("Generating experiment steps for %d experiment(s): %v (storage=%s)", len(experimentNames), experimentNames, data.ExperimentsStorage)

	if data.ExperimentsStorage == ExperimentsStorageCache {
		return c.generateExperimentCacheSteps(data, experimentNames)
	}
	// Default: repo storage.
	return c.generateExperimentRepoSteps(data, experimentNames)
}

// generateExperimentCacheSteps generates the experiment steps using GitHub Actions cache for persistence.
func (c *Compiler) generateExperimentCacheSteps(data *WorkflowData, experimentNames []string) []string {
	// Use the literal sanitized workflow ID in the cache key so it is correct in the
	// activation job, which does not have GH_AW_WORKFLOW_ID_SANITIZED in its environment.
	sanitizedID := SanitizeWorkflowIDForCacheKey(data.WorkflowID)
	cacheKey := fmt.Sprintf("experiments-%s-${{ github.run_id }}", sanitizedID)
	restoreKey := fmt.Sprintf("experiments-%s-", sanitizedID)

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

	steps = append(steps, c.generatePickExperimentStep(data, experimentNames)...)

	// ── Step 3: Save experiment cache ─────────────────────────────────────────
	steps = append(steps,
		"      - name: Save experiment state\n",
		"        if: always()\n",
		fmt.Sprintf("        uses: %s\n", getActionPin("actions/cache/save")),
		"        with:\n",
		fmt.Sprintf("          key: %s\n", cacheKey),
		fmt.Sprintf("          path: %s\n", experimentsCacheDir),
	)

	steps = append(steps, c.generateExperimentArtifactUploadStep(data, sanitizedID)...)
	return steps
}

// generateExperimentRepoSteps generates the experiment steps using a git branch for durable persistence.
// The activation job restores state via the GitHub API (read-only); a separate push_experiments_state
// job commits and pushes the updated state after the activation job succeeds.
func (c *Compiler) generateExperimentRepoSteps(data *WorkflowData, experimentNames []string) []string {
	sanitizedID := SanitizeWorkflowIDForCacheKey(data.WorkflowID)
	branchName := experimentsBranchName(data.WorkflowID)

	var steps []string

	// ── Step 1: Restore experiment state from git branch ─────────────────────
	steps = append(steps,
		"      - name: Restore experiment state from git\n",
		"        id: restore-experiment-state\n",
		fmt.Sprintf("        uses: %s\n", getCachedActionPin("actions/github-script", data)),
		"        env:\n",
		fmt.Sprintf("          GH_AW_EXPERIMENT_STATE_FILE: %s\n", experimentStateFile),
		fmt.Sprintf("          GH_AW_EXPERIMENT_STATE_DIR: %s\n", experimentsCacheDir),
		fmt.Sprintf("          GH_AW_EXPERIMENT_BRANCH: %s\n", branchName),
		"        with:\n",
		"          script: |\n",
		"            const { setupGlobals } = require('"+SetupActionDestination+"/setup_globals.cjs');\n",
		"            setupGlobals(core, github, context, exec, io, getOctokit);\n",
		"            const { main } = require('"+SetupActionDestination+"/load_experiment_state_from_repo.cjs');\n",
		"            await main();\n",
	)

	steps = append(steps, c.generatePickExperimentStep(data, experimentNames)...)
	steps = append(steps, c.generateExperimentArtifactUploadStep(data, sanitizedID)...)
	return steps
}

// generatePickExperimentStep generates the "Pick experiment variants" step shared by both storage modes.
func (c *Compiler) generatePickExperimentStep(data *WorkflowData, experimentNames []string) []string {
	specJSON := buildExperimentSpecJSON(data.Experiments, data.ExperimentConfigs, experimentNames)
	return []string{
		"      - name: Pick experiment variants\n",
		"        id: pick-experiment\n",
		fmt.Sprintf("        uses: %s\n", getCachedActionPin("actions/github-script", data)),
		"        env:\n",
		fmt.Sprintf("          GH_AW_EXPERIMENT_SPEC: '%s'\n", strings.ReplaceAll(specJSON, "'", "''")),
		fmt.Sprintf("          GH_AW_EXPERIMENT_STATE_FILE: %s\n", experimentStateFile),
		fmt.Sprintf("          GH_AW_EXPERIMENT_STATE_DIR: %s\n", experimentsCacheDir),
		"        with:\n",
		"          script: |\n",
		"            const { setupGlobals } = require('" + SetupActionDestination + "/setup_globals.cjs');\n",
		"            setupGlobals(core, github, context, exec, io, getOctokit);\n",
		"            const { main } = require('" + SetupActionDestination + "/pick_experiment.cjs');\n",
		"            await main();\n",
	}
}

// generateExperimentArtifactUploadStep generates the artifact upload step shared by both storage modes.
func (c *Compiler) generateExperimentArtifactUploadStep(data *WorkflowData, sanitizedID string) []string {
	// For workflow_call the artifact prefix expression is prepended at runtime.
	// For regular workflows the sanitized workflow ID is used as a prefix so the
	// artifact name uniquely identifies which workflow produced it.
	experimentArtifactName := experimentArtifactUploadName(data, sanitizedID)
	return []string{
		"      - name: Upload experiment artifact\n",
		"        if: always()\n",
		fmt.Sprintf("        uses: %s\n", getActionPin("actions/upload-artifact")),
		"        with:\n",
		fmt.Sprintf("          name: %s\n", experimentArtifactName),
		fmt.Sprintf("          path: %s\n", experimentsCacheDir),
		"          if-no-files-found: ignore\n",
		"          retention-days: 30\n",
	}
}

// buildExperimentSpecJSON builds a compact JSON object from the experiments map.
// When configs is non-nil and contains an entry for a name, the full ExperimentConfig
// (variants + metadata) is embedded so that pick_experiment.cjs can use weighted
// selection, date-range gating, and other metadata.
// When no config is available a bare variants array is emitted for backward compatibility.
// Uses encoding/json for proper escaping of all special characters.
// Caller is responsible for escaping single quotes when embedding the result in a YAML
// single-quoted scalar (each ' must be doubled to ” per YAML spec §7.3.3).
func buildExperimentSpecJSON(experiments map[string][]string, configs map[string]*ExperimentConfig, names []string) string {
	var sb strings.Builder
	sb.WriteString("{")
	for i, name := range names {
		if i > 0 {
			sb.WriteString(",")
		}
		keyBytes, _ := json.Marshal(name)
		sb.Write(keyBytes)
		sb.WriteString(":")

		// Use the full config when available so the JS can consume metadata.
		if cfg, ok := configs[name]; ok && cfg != nil {
			cfgBytes, _ := json.Marshal(cfg)
			sb.Write(cfgBytes)
		} else {
			// Fallback: bare variants array (legacy behaviour).
			varBytes, _ := json.Marshal(experiments[name])
			sb.Write(varBytes)
		}
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

// experimentArtifactUploadName returns the artifact name used when uploading the experiment
// artifact from the activation job.
// For workflow_call workflows the runtime prefix expression is prepended.
// For regular workflows the sanitized workflow ID is used as a prefix so the artifact name
// uniquely identifies the producing workflow (e.g. "smokecopilot-experiment").
// An empty sanitizedID falls back to the base name for defensive compatibility; in practice
// the compiler always sets a non-empty WorkflowID before this function is called.
func experimentArtifactUploadName(data *WorkflowData, sanitizedID string) string {
	if hasWorkflowCallTrigger(data.On) {
		return artifactPrefixExprForActivationJob(data) + constants.ExperimentArtifactName
	}
	if sanitizedID == "" {
		return constants.ExperimentArtifactName
	}
	return sanitizedID + "-" + constants.ExperimentArtifactName
}

// experimentArtifactDownloadName returns the artifact name used when downloading the experiment
// artifact from a downstream job.
// For workflow_call workflows the runtime prefix expression is prepended.
// For regular workflows the sanitized workflow ID is used as a prefix, matching the name
// produced by experimentArtifactUploadName.
// An empty sanitizedID falls back to the base name for defensive compatibility; in practice
// the compiler always sets a non-empty WorkflowID before this function is called.
func experimentArtifactDownloadName(data *WorkflowData) string {
	if hasWorkflowCallTrigger(data.On) {
		return artifactPrefixExprForDownstreamJob(data) + constants.ExperimentArtifactName
	}
	sanitizedID := SanitizeWorkflowIDForCacheKey(data.WorkflowID)
	if sanitizedID == "" {
		return constants.ExperimentArtifactName
	}
	return sanitizedID + "-" + constants.ExperimentArtifactName
}

// buildExperimentArtifactDownloadSteps creates a download step for the experiment artifact.
// The artifact is downloaded to experimentsCacheDir so the detection agent can read the
// current variant assignments from state.json.
// The step is a no-op when no experiments are declared.
func buildExperimentArtifactDownloadSteps(data *WorkflowData) []string {
	if len(data.Experiments) == 0 {
		return nil
	}
	artifactName := experimentArtifactDownloadName(data)
	return buildArtifactDownloadSteps(ArtifactDownloadConfig{
		ArtifactName: artifactName,
		DownloadPath: experimentsCacheDir + "/",
		StepName:     "Download experiment artifact",
	})
}

// buildPushExperimentsStateJob creates a job that downloads the experiment-state artifact and
// commits it to a git branch ("experiments/{sanitizedID}") for durable storage across runs.
// Returns nil when there are no experiments or the storage mode is not "repo".
func (c *Compiler) buildPushExperimentsStateJob(data *WorkflowData) (*Job, error) {
	if len(data.Experiments) == 0 || data.ExperimentsStorage != ExperimentsStorageRepo {
		return nil, nil
	}

	experimentsLog.Printf("Building push_experiments_state job (branch=%s)", experimentsBranchName(data.WorkflowID))

	var steps []string

	// Setup step so the push_experiment_state.cjs script is available.
	setupActionRef := c.resolveActionReference("./actions/setup", data)
	if setupActionRef != "" || c.actionMode.IsScript() {
		steps = append(steps, c.generateCheckoutActionsFolder(data)...)
		traceID := fmt.Sprintf("${{ needs.%s.outputs.setup-trace-id }}", constants.ActivationJobName)
		steps = append(steps, c.generateSetupStep(data, setupActionRef, SetupActionDestination, false, traceID)...)
	}

	// Checkout step – configure git credentials without downloading workspace files.
	var checkoutStep strings.Builder
	checkoutStep.WriteString("      - name: Checkout repository\n")
	fmt.Fprintf(&checkoutStep, "        uses: %s\n", getActionPin("actions/checkout"))
	checkoutStep.WriteString("        with:\n")
	checkoutStep.WriteString("          persist-credentials: false\n")
	checkoutStep.WriteString("          sparse-checkout: .\n")
	steps = append(steps, checkoutStep.String())

	// Git configuration (author, email).
	steps = append(steps, c.generateGitConfigurationSteps()...)

	// Download the experiment artifact uploaded by the activation job.
	artifactName := experimentArtifactDownloadName(data)
	var downloadStep strings.Builder
	downloadStep.WriteString("      - name: Download experiment artifact\n")
	fmt.Fprintf(&downloadStep, "        uses: %s\n", getActionPin("actions/download-artifact"))
	downloadStep.WriteString("        continue-on-error: true\n")
	downloadStep.WriteString("        with:\n")
	fmt.Fprintf(&downloadStep, "          name: %s\n", artifactName)
	fmt.Fprintf(&downloadStep, "          path: %s\n", experimentsCacheDir)
	steps = append(steps, downloadStep.String())

	// Push experiment state to the git branch via push_experiment_state.cjs.
	// This helper uses pushSignedCommits to create verified (signed) commits.
	branchName := experimentsBranchName(data.WorkflowID)

	var pushStep strings.Builder
	pushStep.WriteString("      - name: Push experiment state to git\n")
	pushStep.WriteString("        id: push_experiments_state\n")
	pushStep.WriteString("        if: always()\n")
	fmt.Fprintf(&pushStep, "        uses: %s\n", getCachedActionPin("actions/github-script", data))
	pushStep.WriteString("        env:\n")
	pushStep.WriteString("          GH_TOKEN: ${{ github.token }}\n")
	pushStep.WriteString("          GITHUB_RUN_ID: ${{ github.run_id }}\n")
	pushStep.WriteString("          GITHUB_SERVER_URL: ${{ github.server_url }}\n")
	fmt.Fprintf(&pushStep, "          GH_AW_EXPERIMENT_STATE_DIR: %s\n", experimentsCacheDir)
	fmt.Fprintf(&pushStep, "          GH_AW_EXPERIMENT_BRANCH: %s\n", branchName)
	pushStep.WriteString("        with:\n")
	pushStep.WriteString("          script: |\n")
	pushStep.WriteString("            const { setupGlobals } = require('" + SetupActionDestination + "/setup_globals.cjs');\n")
	pushStep.WriteString("            setupGlobals(core, github, context, exec, io, getOctokit);\n")
	pushStep.WriteString("            const { main } = require('" + SetupActionDestination + "/push_experiment_state.cjs');\n")
	pushStep.WriteString("            await main();\n")
	steps = append(steps, pushStep.String())

	// Restore the checkout in dev mode (same reason as push_repo_memory).
	if c.actionMode.IsDev() {
		steps = append(steps, c.generateRestoreActionsSetupStep())
	}

	// The push_experiments_state job runs after the activation job succeeds.
	// It does not depend on the agent job because experiment state was fully resolved in activation.
	activationSucceeded := BuildEquals(
		BuildPropertyAccess(fmt.Sprintf("needs.%s.result", constants.ActivationJobName)),
		BuildStringLiteral("success"),
	)
	notCancelled := &NotNode{Child: BuildFunctionCall("cancelled")}
	jobCondition := RenderCondition(BuildAnd(BuildAnd(BuildFunctionCall("always"), notCancelled), activationSucceeded))

	job := &Job{
		Name:        "push_experiments_state",
		RunsOn:      c.formatFrameworkJobRunsOn(data),
		If:          jobCondition,
		Permissions: "permissions:\n      contents: write",
		Needs:       []string{string(constants.ActivationJobName)},
		Steps:       steps,
	}

	return job, nil
}
