// Package workflow implements job-construction helpers for workflow compilation.
//
// The compiler job builders are split into focused modules for maintainability:
//
//   - compiler_jobs.go: Core job orchestration and cross-job dependency wiring
//   - compiler_custom_jobs.go: Custom job extraction, property mapping, and step customization
//
// This separation keeps the orchestration flow compact while preserving the
// existing custom job behavior.
package workflow

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/goccy/go-yaml"

	"github.com/github/gh-aw/pkg/constants"
	"github.com/github/gh-aw/pkg/setutil"
)

var exactSetupStepIDPattern = regexp.MustCompile(`(?m)^\s*id:\s*setup\s*$`)

// buildCustomJobs creates custom jobs defined in the frontmatter jobs section
func (c *Compiler) buildCustomJobs(data *WorkflowData, activationJobCreated bool) error {
	compilerJobsLog.Printf("Building %d custom jobs", len(data.Jobs))

	promptReferencedJobs, onNeedsJobs := c.getCustomJobDependencySets(data)

	for jobName, jobConfig := range data.Jobs {
		if c.shouldSkipCustomJob(jobName) {
			continue
		}
		configMap, ok := jobConfig.(map[string]any)
		if !ok {
			continue
		}

		job, err := c.buildCustomJob(
			jobName,
			configMap,
			data,
			activationJobCreated,
			promptReferencedJobs,
			onNeedsJobs,
		)
		if err != nil {
			return err
		}

		if err := c.jobManager.AddJob(job); err != nil {
			return fmt.Errorf("failed to add custom job '%s': %w", jobName, err)
		}
		compilerJobsLog.Printf("Successfully added custom job '%s' with %d needs dependencies", jobName, len(job.Needs))
	}

	compilerJobsLog.Print("Completed building all custom jobs")
	return nil
}

func (c *Compiler) getCustomJobDependencySets(data *WorkflowData) (map[string]struct{}, map[string]struct{}) {
	// Pre-compute jobs referenced in the markdown body with no explicit needs.
	// These run before activation (not after), so we must not auto-add activation to them.
	promptReferencedJobsSlice := c.getCustomJobsReferencedInPromptWithNoActivationDep(data)
	promptReferencedJobs := make(map[string]struct {
	}, len(promptReferencedJobsSlice))
	for _, j := range promptReferencedJobsSlice {
		promptReferencedJobs[j] = struct {
		}{}
	}

	onNeedsJobs := make(map[string]struct {
	}, len(data.OnNeeds))
	for _, j := range data.OnNeeds {
		onNeedsJobs[j] = struct {
		}{}
	}

	return promptReferencedJobs, onNeedsJobs
}

func (c *Compiler) shouldSkipCustomJob(jobName string) bool {
	// Skip jobs.pre-activation (or pre_activation) as it's handled specially in buildPreActivationJob
	if jobName == string(constants.PreActivationJobName) || jobName == "pre-activation" {
		compilerJobsLog.Printf("Skipping jobs.%s (handled in buildPreActivationJob)", jobName)
		return true
	}

	// Built-in jobs are already created before buildCustomJobs; treat jobs.<builtin>
	// entries as customization-only and do not create duplicate jobs.
	if _, exists := c.jobManager.GetJob(jobName); exists {
		compilerJobsLog.Printf("Skipping jobs.%s (built-in job already exists)", jobName)
		return true
	}

	return false
}

func (c *Compiler) buildCustomJob(
	jobName string,
	configMap map[string]any,
	data *WorkflowData,
	activationJobCreated bool,
	promptReferencedJobs map[string]struct {
	}, onNeedsJobs map[string]struct {
	}) (*Job, error) {
	job := &Job{Name: jobName}

	hasExplicitNeeds := extractCustomJobNeeds(job, configMap)
	c.applyAutomaticActivationDependency(job, jobName, hasExplicitNeeds, activationJobCreated, promptReferencedJobs, onNeedsJobs)

	if err := c.extractCustomJobProperties(job, jobName, configMap); err != nil {
		return nil, err
	}

	if err := c.configureCustomJobExecution(job, jobName, configMap, data); err != nil {
		return nil, err
	}

	return job, nil
}

func extractCustomJobNeeds(job *Job, configMap map[string]any) bool {
	needs, hasNeeds := configMap["needs"]
	if !hasNeeds {
		return false
	}

	if needsList, ok := needs.([]any); ok {
		for _, need := range needsList {
			if needStr, ok := need.(string); ok {
				job.Needs = append(job.Needs, needStr)
			}
		}
	} else if needStr, ok := needs.(string); ok {
		// Single dependency as string
		job.Needs = append(job.Needs, needStr)
	}

	return true
}

func (c *Compiler) applyAutomaticActivationDependency(
	job *Job,
	jobName string,
	hasExplicitNeeds bool,
	activationJobCreated bool,
	promptReferencedJobs map[string]struct {
	}, onNeedsJobs map[string]struct {
	}) {
	// If no explicit needs and activation job exists, automatically add activation as dependency
	// This ensures custom jobs wait for workflow validation before executing.
	// Exception: jobs whose outputs are referenced in the markdown body run before activation
	// (so the activation job can include their outputs in the prompt).
	isReferencedInMarkdown := setutil.Contains(promptReferencedJobs, jobName)
	isOnNeedsDependency := setutil.Contains(onNeedsJobs, jobName)

	if !hasExplicitNeeds && activationJobCreated && !isReferencedInMarkdown && !isOnNeedsDependency {
		job.Needs = append(job.Needs, string(constants.ActivationJobName))
		compilerJobsLog.Printf("Added automatic dependency: custom job '%s' now depends on '%s'", jobName, string(constants.ActivationJobName))
	} else if !hasExplicitNeeds && isReferencedInMarkdown {
		compilerJobsLog.Printf("Custom job '%s' referenced in markdown body runs before activation (no auto-added dependency)", jobName)
	} else if !hasExplicitNeeds && isOnNeedsDependency {
		compilerJobsLog.Printf("Custom job '%s' listed in on.needs runs before activation (no auto-added dependency)", jobName)
	}
}

func (c *Compiler) extractCustomJobProperties(job *Job, jobName string, configMap map[string]any) error {
	if err := c.extractCustomJobCoreProperties(job, jobName, configMap); err != nil {
		return err
	}
	extractCustomJobOutputs(job, jobName, configMap)
	return nil
}

func (c *Compiler) extractCustomJobCoreProperties(job *Job, jobName string, configMap map[string]any) error {
	if err := c.extractCustomJobRunsOn(job, jobName, configMap); err != nil {
		return err
	}

	if ifCond, hasIf := configMap["if"]; hasIf {
		if ifStr, ok := ifCond.(string); ok {
			job.If = c.extractExpressionFromIfString(ifStr)
		}
	}

	if permissions, hasPermissions := configMap["permissions"]; hasPermissions {
		if permsMap, ok := permissions.(map[string]any); ok {
			formattedPerms, err := formatIndentedYAMLField("permissions", permsMap, false)
			if err != nil {
				return fmt.Errorf("failed to convert permissions to YAML for job '%s': %w", jobName, err)
			}
			job.Permissions = formattedPerms
		}
	}

	if strategy, hasStrategy := configMap["strategy"]; hasStrategy {
		if strategyMap, ok := strategy.(map[string]any); ok {
			formattedStrategy, err := formatIndentedYAMLField("strategy", strategyMap, false)
			if err != nil {
				return fmt.Errorf("failed to convert strategy to YAML for job '%s': %w", jobName, err)
			}
			job.Strategy = formattedStrategy
		}
	}

	// Extract name (display name) for custom jobs
	if name, hasName := configMap["name"]; hasName {
		if nameStr, ok := name.(string); ok {
			job.DisplayName = nameStr
		}
	}

	if err := extractCustomJobTimeoutMinutes(job, jobName, configMap); err != nil {
		return err
	}

	if err := extractCustomJobConcurrency(job, jobName, configMap); err != nil {
		return err
	}

	extractCustomJobEnv(job, configMap)

	if err := extractCustomJobContainer(job, jobName, configMap); err != nil {
		return err
	}
	if err := extractCustomJobServices(job, jobName, configMap); err != nil {
		return err
	}
	extractCustomJobContinueOnError(job, configMap)

	if err := extractCustomJobEnvironment(job, jobName, configMap); err != nil {
		return err
	}

	return nil
}

func (c *Compiler) extractCustomJobRunsOn(job *Job, jobName string, configMap map[string]any) error {
	runsOn, hasRunsOn := configMap["runs-on"]
	if !hasRunsOn {
		return nil
	}
	if runsOnStr, ok := runsOn.(string); ok {
		job.RunsOn = "runs-on: " + runsOnStr
		return nil
	}

	// Array or object form: marshal the value and build indented YAML snippet
	formattedRunsOn, err := formatIndentedYAMLField("runs-on", runsOn, true)
	if err != nil {
		return fmt.Errorf("failed to convert runs-on to YAML for job '%s': %w", jobName, err)
	}
	job.RunsOn = formattedRunsOn
	return nil
}

func extractCustomJobTimeoutMinutes(job *Job, jobName string, configMap map[string]any) error {
	timeout, hasTimeout := configMap["timeout-minutes"]
	if !hasTimeout {
		return nil
	}

	switch v := timeout.(type) {
	case int:
		job.TimeoutMinutes = v
	case uint64:
		if v <= uint64(^uint(0)>>1) {
			job.TimeoutMinutes = int(v)
		}
	case float64:
		job.TimeoutMinutes = int(v)
	case string:
		// isExpression validates full GitHub Actions expression syntax (${{
		// ... }}) and is defined in expression_patterns.go.
		if isExpression(v) {
			job.TimeoutMinutesExpression = v
		} else {
			return fmt.Errorf(
				"job '%s' timeout-minutes must be an integer or a GitHub Actions expression, got %q. Example: timeout-minutes: 30 or ${{ inputs.timeout }}",
				jobName,
				v,
			)
		}
	}

	return nil
}

func extractCustomJobConcurrency(job *Job, jobName string, configMap map[string]any) error {
	concurrency, hasConcurrency := configMap["concurrency"]
	if !hasConcurrency {
		return nil
	}

	switch v := concurrency.(type) {
	case string:
		job.Concurrency = "concurrency: " + v
	case map[string]any:
		// Default cancel-in-progress to false for non-agent jobs if not explicitly set.
		// This prevents accidental cancellation of queued runs when multiple agents
		// are running the same workflow concurrently.
		if _, hasCancelInProgress := v["cancel-in-progress"]; !hasCancelInProgress {
			v["cancel-in-progress"] = false
		}

		formattedConcurrency, err := formatIndentedYAMLField("concurrency", v, false)
		if err != nil {
			return fmt.Errorf("failed to convert concurrency to YAML for job '%s': %w", jobName, err)
		}
		job.Concurrency = formattedConcurrency
	}

	return nil
}

func extractCustomJobEnv(job *Job, configMap map[string]any) {
	env, hasEnv := configMap["env"]
	if !hasEnv {
		return
	}
	envMap, ok := env.(map[string]any)
	if !ok {
		return
	}

	job.Env = make(map[string]string)
	for key, val := range envMap {
		if valStr, ok := val.(string); ok {
			job.Env[key] = valStr
		} else if val != nil {
			// Arrays and maps are serialized as JSON so that shell consumers
			// (e.g. jq --argjson) receive valid JSON.
			job.Env[key] = marshalEnvValue(val)
		}
	}
}

func extractCustomJobContainer(job *Job, jobName string, configMap map[string]any) error {
	container, hasContainer := configMap["container"]
	if !hasContainer {
		return nil
	}

	switch v := container.(type) {
	case string:
		job.Container = "container: " + v
	case map[string]any:
		formattedContainer, err := formatIndentedYAMLField("container", v, false)
		if err != nil {
			return fmt.Errorf("failed to convert container to YAML for job '%s': %w", jobName, err)
		}
		job.Container = formattedContainer
	}

	return nil
}

func extractCustomJobServices(job *Job, jobName string, configMap map[string]any) error {
	services, hasServices := configMap["services"]
	if !hasServices {
		return nil
	}
	servicesMap, ok := services.(map[string]any)
	if !ok {
		return nil
	}

	formattedServices, err := formatIndentedYAMLField("services", servicesMap, false)
	if err != nil {
		return fmt.Errorf("failed to convert services to YAML for job '%s': %w", jobName, err)
	}
	job.Services = formattedServices
	return nil
}

func extractCustomJobContinueOnError(job *Job, configMap map[string]any) {
	continueOnError, hasCOE := configMap["continue-on-error"]
	if !hasCOE {
		return
	}
	if coeVal, ok := continueOnError.(bool); ok {
		job.ContinueOnError = &coeVal
	}
}

func extractCustomJobEnvironment(job *Job, jobName string, configMap map[string]any) error {
	environment, hasEnvironment := configMap["environment"]
	if !hasEnvironment {
		return nil
	}

	switch v := environment.(type) {
	case string:
		job.Environment = "environment: " + v
	case map[string]any:
		formattedEnvironment, err := formatIndentedYAMLField("environment", v, true)
		if err != nil {
			return fmt.Errorf("failed to convert environment to YAML for job '%s': %w", jobName, err)
		}
		job.Environment = formattedEnvironment
	}

	return nil
}

func extractCustomJobOutputs(job *Job, jobName string, configMap map[string]any) {
	outputs, hasOutputs := configMap["outputs"]
	if !hasOutputs {
		return
	}
	outputsMap, ok := outputs.(map[string]any)
	if !ok {
		return
	}

	job.Outputs = make(map[string]string)
	for key, val := range outputsMap {
		if valStr, ok := val.(string); ok {
			job.Outputs[key] = valStr
		} else {
			compilerJobsLog.Printf("Warning: output '%s' in job '%s' has non-string value (type: %T), ignoring", key, jobName, val)
		}
	}
}

func (c *Compiler) configureCustomJobExecution(job *Job, jobName string, configMap map[string]any, data *WorkflowData) error {
	uses, hasUses := configMap["uses"]
	if hasUses {
		if usesStr, ok := uses.(string); ok {
			return configureCustomReusableWorkflow(job, jobName, usesStr, configMap)
		}
	}

	return c.configureCustomJobSteps(job, jobName, configMap, data)
}

func configureCustomReusableWorkflow(job *Job, jobName string, usesStr string, configMap map[string]any) error {
	compilerJobsLog.Printf("Custom job '%s' is a reusable workflow call: %s", jobName, usesStr)

	// restore-memory cannot inject steps into reusable-workflow call jobs (no steps block).
	if rm, ok := configMap["restore-memory"]; ok && rm != nil && rm != false {
		return fmt.Errorf("jobs.%s.restore-memory: not supported for reusable workflow call jobs (uses: %s)", jobName, usesStr)
	}

	job.Uses = usesStr

	// Extract with parameters for reusable workflow
	if with, hasWith := configMap["with"]; hasWith {
		if withMap, ok := with.(map[string]any); ok {
			job.With = withMap
		}
	}

	// Extract secrets for reusable workflow
	if secrets, hasSecrets := configMap["secrets"]; hasSecrets {
		switch sv := secrets.(type) {
		case string:
			if sv == "inherit" {
				job.SecretsInherit = true
			}
		case map[string]any:
			job.Secrets = make(map[string]string)
			for key, val := range sv {
				if valStr, ok := val.(string); ok {
					// Validate that the secret value is a proper GitHub Actions expression
					// Note: We don't pass the key to validateSecretsExpression to prevent
					// CodeQL from detecting sensitive data flow to error messages/logs
					if err := validateSecretsExpression(valStr); err != nil {
						return err
					}
					job.Secrets[key] = valStr
				}
			}
		}
	}

	return nil
}

func (c *Compiler) configureCustomJobSteps(job *Job, jobName string, configMap map[string]any, data *WorkflowData) error {
	_, hasSetupStepsField := configMap["setup-steps"]
	_, hasPreStepsField := configMap["pre-steps"]
	_, hasStepsField := configMap["steps"]
	setupSteps, preSteps, regularSteps, err := c.extractCustomJobStepGroups(jobName, configMap, data)
	if err != nil {
		return err
	}
	restoreMemCfg, err := extractRestoreMemoryConfig(configMap, jobName, data)
	if err != nil {
		return err
	}
	hasRestoreMemory := restoreMemCfg != nil
	ensureRestoreMemoryCacheEnv(job, restoreMemCfg, data)
	if hasSetupStepsField || hasPreStepsField || hasStepsField || hasRestoreMemory {
		job.Steps = append(job.Steps, setupSteps...)
		job.Steps = append(job.Steps, generateGHESHostConfigurationStep())
		if hasRestoreMemory {
			if err := c.appendCustomJobRestoreMemorySteps(job, restoreMemCfg, jobName, data); err != nil {
				return err
			}
		}
		job.Steps = append(job.Steps, preSteps...)
		job.Steps = append(job.Steps, regularSteps...)
	}
	return nil
}

func (c *Compiler) extractCustomJobStepGroups(jobName string, configMap map[string]any, data *WorkflowData) ([]string, []string, []string, error) {
	setupSteps, err := c.extractCustomJobStepGroup("setup-steps", jobName, configMap, data)
	if err != nil {
		return nil, nil, nil, err
	}
	preSteps, err := c.extractCustomJobStepGroup("pre-steps", jobName, configMap, data)
	if err != nil {
		return nil, nil, nil, err
	}
	regularSteps, err := c.extractCustomJobStepGroup("steps", jobName, configMap, data)
	if err != nil {
		return nil, nil, nil, err
	}
	return setupSteps, preSteps, regularSteps, nil
}

func (c *Compiler) extractCustomJobStepGroup(fieldName, jobName string, configMap map[string]any, data *WorkflowData) ([]string, error) {
	if _, hasField := configMap[fieldName]; !hasField {
		return nil, nil
	}
	steps, err := c.extractPinnedJobSteps(fieldName, jobName, configMap, data)
	if err != nil {
		return nil, fmt.Errorf("failed to process %s for job '%s': %w", fieldName, jobName, err)
	}
	return steps, nil
}

func ensureRestoreMemoryCacheEnv(job *Job, restoreMemCfg *restoreMemoryConfig, data *WorkflowData) {
	if restoreMemCfg == nil || !restoreMemCfg.CacheMemory || data.WorkflowID == "" {
		return
	}
	if job.Env == nil {
		job.Env = make(map[string]string)
	}
	if _, alreadySet := job.Env["GH_AW_WORKFLOW_ID_SANITIZED"]; !alreadySet {
		job.Env["GH_AW_WORKFLOW_ID_SANITIZED"] = SanitizeWorkflowIDForCacheKey(data.WorkflowID)
	}
}

func (c *Compiler) appendCustomJobRestoreMemorySteps(job *Job, restoreMemCfg *restoreMemoryConfig, jobName string, data *WorkflowData) error {
	memorySetupLines, memoryRestoreLines, err := c.buildRestoreMemorySteps(restoreMemCfg, jobName, data)
	if err != nil {
		return err
	}
	job.Steps = append(job.Steps, memorySetupLines...)
	job.Steps = append(job.Steps, memoryRestoreLines...)
	return nil
}

func formatIndentedYAMLField(fieldName string, value any, trimTrailingNewline bool) (string, error) {
	yamlBytes, err := yaml.Marshal(value)
	if err != nil {
		return "", err
	}

	lines := strings.Split(strings.TrimSpace(string(yamlBytes)), "\n")
	var b strings.Builder
	b.WriteString(fieldName + ":\n")
	for _, line := range lines {
		b.WriteString("      " + line + "\n")
	}

	formatted := b.String()
	if trimTrailingNewline {
		return strings.TrimSuffix(formatted, "\n"), nil
	}
	return formatted, nil
}

func (c *Compiler) applyBuiltinJobPreSteps(data *WorkflowData) error {
	if data == nil || data.Jobs == nil {
		return nil
	}

	for jobName, jobConfig := range data.Jobs {
		configMap, ok := jobConfig.(map[string]any)
		if !ok {
			return fmt.Errorf("jobs.%s must be an object, got %T. Example: jobs:\n  job-name:\n    setup-steps: []", jobName, jobConfig)
		}

		_, hasSetupSteps := configMap["setup-steps"]
		_, hasPreSteps := configMap["pre-steps"]
		if err := validateRestrictedBuiltinSetupSteps(jobName, hasSetupSteps); err != nil {
			return err
		}
		if !hasSetupSteps && !hasPreSteps {
			continue
		}

		targetJobName := jobName
		if jobName == "pre-activation" {
			targetJobName = string(constants.PreActivationJobName)
		}

		job, exists := c.jobManager.GetJob(targetJobName)
		if !exists {
			continue
		}

		var setupSteps []string
		var preSteps []string
		if hasSetupSteps {
			steps, err := c.extractPinnedJobSteps("setup-steps", jobName, configMap, data)
			if err != nil {
				return fmt.Errorf("failed to process setup-steps for built-in job '%s': %w", jobName, err)
			}
			setupSteps = append(setupSteps, steps...)
		}
		if hasPreSteps {
			steps, err := c.extractPinnedJobSteps("pre-steps", jobName, configMap, data)
			if err != nil {
				return fmt.Errorf("failed to process pre-steps for built-in job '%s': %w", jobName, err)
			}
			preSteps = append(preSteps, steps...)
		}
		if len(setupSteps) == 0 && len(preSteps) == 0 {
			continue
		}

		job.Steps = insertPreStepsAtEarliestBoundary(job.Steps, preSteps)
		job.Steps = insertSetupStepsAtStart(job.Steps, setupSteps)
		compilerJobsLog.Printf("Inserted %d setup-step(s) and %d pre-step(s) into built-in job '%s'", len(setupSteps), len(preSteps), targetJobName)
	}

	return nil
}

func normalizeBuiltinJobAlias(jobName string) string {
	switch jobName {
	case string(constants.PreActivationHyphenJobName):
		return string(constants.PreActivationJobName)
	case string(constants.SafeOutputsHyphenJobName):
		return string(constants.SafeOutputsJobName)
	default:
		return jobName
	}
}

func extractBuiltinJobNeedsAugmentation(jobName string, configMap map[string]any) ([]string, error) {
	needsValue, exists := configMap["needs"]
	if !exists || needsValue == nil {
		return nil, nil
	}

	switch typedNeeds := needsValue.(type) {
	case string:
		return []string{typedNeeds}, nil
	case []any:
		needs := make([]string, 0, len(typedNeeds))
		for i, rawNeed := range typedNeeds {
			need, ok := rawNeed.(string)
			if !ok {
				return nil, fmt.Errorf("jobs.%s.needs[%d] must be a string, got %T. Example: needs: ['build', 'test']", jobName, i, rawNeed)
			}
			needs = append(needs, need)
		}
		return needs, nil
	default:
		return nil, fmt.Errorf("jobs.%s.needs must be a string or array of strings, got %T", jobName, needsValue)
	}
}

// applyBuiltinJobNeedsAugmentations merges jobs.<built-in>.needs into compiler-generated job needs.
// This is additive-only and de-duplicated, and never removes compiler-computed dependencies.
func (c *Compiler) applyBuiltinJobNeedsAugmentations(data *WorkflowData) error {
	if data == nil || data.Jobs == nil {
		return nil
	}
	allJobs := c.jobManager.GetAllJobs()
	for configuredJobName, rawConfig := range data.Jobs {
		targetJobName := normalizeBuiltinJobAlias(configuredJobName)
		if !isBuiltinJobName(targetJobName) {
			continue
		}

		configMap, ok := rawConfig.(map[string]any)
		if !ok {
			return fmt.Errorf("jobs.%s must be an object, got %T", configuredJobName, rawConfig)
		}

		augmentedNeeds, err := extractBuiltinJobNeedsAugmentation(configuredJobName, configMap)
		if err != nil {
			return err
		}
		if len(augmentedNeeds) == 0 {
			continue
		}
		targetJob, exists := c.jobManager.GetJob(targetJobName)
		if !exists {
			return fmt.Errorf("jobs.%s.needs: cannot augment %q because this workflow does not generate that job", configuredJobName, targetJobName)
		}
		normalizedNeeds, err := normalizeBuiltinJobNeeds(configuredJobName, targetJobName, augmentedNeeds, allJobs)
		if err != nil {
			return err
		}
		targetJob.Needs = mergeBuiltinJobNeeds(targetJob.Needs, normalizedNeeds)
		compilerJobsLog.Printf("Applied jobs.%s.needs augmentation to %q: %v", configuredJobName, targetJobName, normalizedNeeds)
	}
	return nil
}

func normalizeBuiltinJobNeeds(configuredJobName, targetJobName string, augmentedNeeds []string, allJobs map[string]*Job) ([]string, error) {
	normalizedNeeds := make([]string, 0, len(augmentedNeeds))
	for _, rawNeed := range augmentedNeeds {
		need := normalizeBuiltinJobAlias(rawNeed)
		if need == targetJobName {
			return nil, fmt.Errorf("jobs.%s.needs: %q cannot depend on itself", configuredJobName, rawNeed)
		}
		if _, known := allJobs[need]; !known {
			return nil, fmt.Errorf("jobs.%s.needs: unknown job %q", configuredJobName, rawNeed)
		}
		normalizedNeeds = append(normalizedNeeds, need)
	}
	return normalizedNeeds, nil
}

func mergeBuiltinJobNeeds(existingNeeds, normalizedNeeds []string) []string {
	seen := make(map[string]struct{}, len(existingNeeds)+len(normalizedNeeds))
	mergedNeeds := make([]string, 0, len(existingNeeds)+len(normalizedNeeds))
	for _, need := range append(existingNeeds, normalizedNeeds...) {
		if _, alreadySeen := seen[need]; alreadySeen {
			continue
		}
		seen[need] = struct{}{}
		mergedNeeds = append(mergedNeeds, need)
	}
	return mergedNeeds
}

func validateRestrictedBuiltinSetupSteps(jobName string, hasSetupSteps bool) error {
	if !hasSetupSteps {
		return nil
	}

	if jobName == string(constants.ActivationJobName) ||
		jobName == string(constants.PreActivationJobName) ||
		jobName == "pre-activation" {
		return fmt.Errorf(
			"jobs.%s.setup-steps is not allowed: setup-steps are refused for activation/pre-activation jobs because they can short-circuit protections",
			jobName,
		)
	}

	return nil
}

// insertSetupStepsAtStart places setup-steps at the start of the job so they
// run before any compiler-generated setup, checkout, or token-mint steps.
func insertSetupStepsAtStart(steps []string, setupSteps []string) []string {
	if len(setupSteps) == 0 {
		return steps
	}

	result := make([]string, 0, safeAllocationCapacity(len(steps), len(setupSteps)))
	result = append(result, setupSteps...)
	result = append(result, steps...)
	return result
}

func insertPreStepsAtEarliestBoundary(steps []string, preSteps []string) []string {
	if len(preSteps) == 0 {
		return steps
	}
	insertIdx := findPreStepsInsertIndex(steps)
	result := make([]string, 0, safeAllocationCapacity(len(steps), len(preSteps)))
	result = append(result, steps[:insertIdx]...)
	result = append(result, preSteps...)
	result = append(result, steps[insertIdx:]...)
	return result
}

func findPreStepsInsertIndex(steps []string) int {
	firstCheckoutIdx := -1
	firstTokenMintIdx := -1
	lastSetupIdx := -1
	for i, step := range steps {
		if firstCheckoutIdx == -1 && strings.Contains(step, "uses: actions/checkout@") {
			firstCheckoutIdx = findStepBoundaryBefore(steps, i)
		}
		if firstTokenMintIdx == -1 && strings.Contains(step, "uses: actions/create-github-app-token@") {
			firstTokenMintIdx = findStepBoundaryBefore(steps, i)
		}
		if exactSetupStepIDPattern.MatchString(step) {
			lastSetupIdx = i
		}
	}
	return choosePreStepsInsertIndex(steps, firstCheckoutIdx, firstTokenMintIdx, lastSetupIdx)
}

func findStepBoundaryBefore(steps []string, index int) int {
	for j := index; j >= 0; j-- {
		trimmed := strings.TrimLeft(steps[j], " ")
		if strings.HasPrefix(trimmed, "- ") {
			return j
		}
	}
	return index
}

func choosePreStepsInsertIndex(steps []string, firstCheckoutIdx, firstTokenMintIdx, lastSetupIdx int) int {
	insertIdx := len(steps)
	if lastSetupIdx >= 0 {
		for i := lastSetupIdx + 1; i < len(steps); i++ {
			trimmed := strings.TrimLeft(steps[i], " ")
			if strings.HasPrefix(trimmed, "- ") {
				insertIdx = i
				break
			}
		}
		if insertIdx == len(steps) {
			compilerJobsLog.Print("No step boundary found after setup step; appending pre-steps at end")
		}
	} else if firstTokenMintIdx >= 0 {
		insertIdx = firstTokenMintIdx
		if firstCheckoutIdx >= 0 {
			if firstCheckoutIdx < insertIdx {
				insertIdx = firstCheckoutIdx
			}
		}
	} else if firstCheckoutIdx >= 0 {
		insertIdx = firstCheckoutIdx
	}
	if insertIdx > len(steps) {
		insertIdx = len(steps)
	}
	return insertIdx
}

func (c *Compiler) extractPinnedJobSteps(fieldName string, jobName string, configMap map[string]any, data *WorkflowData) ([]string, error) {
	raw, hasField := configMap[fieldName]
	if !hasField {
		return nil, nil
	}

	stepsList, ok := raw.([]any)
	if !ok {
		return nil, fmt.Errorf("%s for job '%s' must be an array of step objects", fieldName, jobName)
	}

	pinnedSteps := make([]string, 0, len(stepsList))
	for i, step := range stepsList {
		stepMap, ok := step.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("%s for job '%s' contains invalid step at index %d: expected object", fieldName, jobName, i)
		}

		typedStep, err := MapToStep(stepMap)
		if err != nil {
			return nil, fmt.Errorf("failed to convert %s to typed step for job '%s': %w", fieldName, jobName, err)
		}

		pinnedStep, err := applyActionPinToTypedStep(typedStep, data)
		if err != nil {
			return nil, fmt.Errorf("failed to pin action for %s in job '%s': %w", fieldName, jobName, err)
		}
		finalStepMap := pinnedStep.ToMap()
		ensureCheckoutPersistCredentials(finalStepMap)
		sanitizedMap, warnings, _ := sanitizeRunStepExpressions(finalStepMap)
		for _, w := range warnings {
			compilerJobsLog.Printf("sanitized run: expression in job '%s' step: %s", jobName, w)
		}
		stepYAML, err := ConvertStepToYAML(sanitizedMap)
		if err != nil {
			return nil, fmt.Errorf("failed to convert %s to YAML for job '%s': %w", fieldName, jobName, err)
		}
		pinnedSteps = append(pinnedSteps, stepYAML)
	}

	return pinnedSteps, nil
}

// ensureCheckoutPersistCredentials enforces with.persist-credentials: false for
// actions/checkout steps when not explicitly configured by the user.
func ensureCheckoutPersistCredentials(stepMap map[string]any) {
	uses, ok := stepMap["uses"].(string)
	if !ok || !isCheckoutAction(uses) {
		return
	}

	withRaw, hasWith := stepMap["with"]
	if !hasWith || withRaw == nil {
		stepMap["with"] = map[string]any{
			"persist-credentials": false,
		}
		return
	}

	withMap, ok := withRaw.(map[string]any)
	if !ok {
		return
	}
	if v, exists := withMap["persist-credentials"]; exists && v != nil {
		return
	}
	withMap["persist-credentials"] = false
}

// isCheckoutAction reports whether a uses value points to actions/checkout,
// including either unpinned or version-pinned forms.
func isCheckoutAction(uses string) bool {
	trimmed := strings.Trim(strings.TrimSpace(uses), "\"'")
	return strings.EqualFold(trimmed, "actions/checkout") || strings.HasPrefix(strings.ToLower(trimmed), "actions/checkout@")
}
