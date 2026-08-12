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
			return fmt.Errorf("custom job '%s' could not be added: %w. Check the job configuration for conflicting names or unsupported fields", jobName, err)
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
	promptReferencedJobs := make(map[string]struct{}, len(promptReferencedJobsSlice))
	for _, j := range promptReferencedJobsSlice {
		promptReferencedJobs[j] = struct{}{}
	}

	// Also include jobs with no explicit needs that are referenced in engine.env.
	// These run before activation (activation depends on them for secret validation etc.),
	// so we must not auto-add activation to them either — doing so would create a cycle
	// (activation → job → activation).
	for _, j := range c.getEngineEnvReferencedCustomJobsWithNoExplicitNeeds(data) {
		promptReferencedJobs[j] = struct{}{}
	}

	onNeedsJobs := make(map[string]struct{}, len(data.OnNeeds))
	for _, j := range data.OnNeeds {
		onNeedsJobs[j] = struct{}{}
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
	if _, hasInputs := configMap["inputs"]; hasInputs {
		return fmt.Errorf("jobs.%s.inputs: inputs are not supported on jobs; use 'env' to pass values to job steps", jobName)
	}

	if err := c.extractCustomJobRunsOn(job, jobName, configMap); err != nil {
		return err
	}

	if ifCond, hasIf := configMap["if"]; hasIf {
		if ifStr, ok := ifCond.(string); ok {
			job.If = c.extractExpressionFromIfString(ifStr)
		}
	}

	if permissions, hasPermissions := configMap["permissions"]; hasPermissions {
		formattedPerms := NewPermissionsParserFromValue(permissions).ToPermissions().RenderToYAML()
		if formattedPerms != "" {
			job.Permissions = formattedPerms
		}
	}

	if strategy, hasStrategy := configMap["strategy"]; hasStrategy {
		if strategyMap, ok := strategy.(map[string]any); ok {
			formattedStrategy, err := formatIndentedYAMLField("strategy", strategyMap, false)
			if err != nil {
				return fmt.Errorf("strategy field for job '%s' could not be converted to YAML: %w. Check that strategy is a valid object, for example: strategy:\n  matrix:\n    os: [ubuntu-latest]", jobName, err)
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
		return fmt.Errorf("runs-on field for job '%s' could not be converted to YAML: %w. Check that runs-on is a valid string, array, or object, for example: runs-on: ubuntu-latest", jobName, err)
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
			return fmt.Errorf("concurrency field for job '%s' could not be converted to YAML: %w. Check that concurrency is a valid object, for example: concurrency:\n  group: my-group", jobName, err)
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
			return fmt.Errorf("container field for job '%s' could not be converted to YAML: %w. Check that container is a valid object, for example: container:\n  image: node:20", jobName, err)
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
		return fmt.Errorf("services field for job '%s' could not be converted to YAML: %w. Check that services is a valid object, for example: services:\n  redis:\n    image: redis", jobName, err)
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
			return fmt.Errorf("environment field for job '%s' could not be converted to YAML: %w. Check that environment is a valid object, for example: environment:\n  name: production", jobName, err)
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
	if job.RunsOn == "" {
		job.RunsOn = c.indentYAMLLines(data.RunsOn, "    ")
		if job.RunsOn == "" {
			job.RunsOn = "runs-on: ubuntu-latest"
		}
	}

	// Add basic steps if specified (only for non-reusable workflow jobs).
	// `setup-steps` and `pre-steps` stay distinct so setup-steps can remain the
	// first injected steps in the job, followed by compiler scaffolding,
	// `pre-steps`, and the regular `steps` list.
	var setupSteps []string
	var preSteps []string
	var regularSteps []string
	_, hasSetupStepsField := configMap["setup-steps"]
	_, hasPreStepsField := configMap["pre-steps"]
	_, hasStepsField := configMap["steps"]

	if hasSetupStepsField {
		var err error
		setupSteps, err = c.extractPinnedJobSteps("setup-steps", jobName, configMap, data)
		if err != nil {
			return fmt.Errorf("setup-steps for job '%s' could not be processed: %w. Check that setup-steps is an array of valid step objects", jobName, err)
		}
	}
	if hasPreStepsField {
		var err error
		preSteps, err = c.extractPinnedJobSteps("pre-steps", jobName, configMap, data)
		if err != nil {
			return fmt.Errorf("pre-steps for job '%s' could not be processed: %w. Check that pre-steps is an array of valid step objects", jobName, err)
		}
	}
	if hasStepsField {
		var err error
		regularSteps, err = c.extractPinnedJobSteps("steps", jobName, configMap, data)
		if err != nil {
			return fmt.Errorf("steps for job '%s' could not be processed: %w. Check that steps is an array of valid step objects", jobName, err)
		}
	}

	// Parse restore-memory configuration.
	// restore-memory injects read-only memory restore steps into the custom job.
	// No write-back or commit steps are ever emitted for memory in custom jobs.
	restoreMemCfg, err := extractRestoreMemoryConfig(configMap, jobName, data)
	if err != nil {
		return err
	}

	hasRestoreMemory := restoreMemCfg != nil

	// When cache-memory restore is requested, inject GH_AW_WORKFLOW_ID_SANITIZED so that
	// restore keys match those used by the agent job.  Only set it when the user has not
	// already provided the variable in their job's env: block.
	if hasRestoreMemory && restoreMemCfg.CacheMemory && data.WorkflowID != "" {
		sanitized := SanitizeWorkflowIDForCacheKey(data.WorkflowID)
		if job.Env == nil {
			job.Env = make(map[string]string)
		}
		if _, alreadySet := job.Env["GH_AW_WORKFLOW_ID_SANITIZED"]; !alreadySet {
			job.Env["GH_AW_WORKFLOW_ID_SANITIZED"] = sanitized
		}
	}

	if hasSetupStepsField || hasPreStepsField || hasStepsField || hasRestoreMemory {
		job.Steps = append(job.Steps, setupSteps...)
		// Prepend GH_HOST configuration step for GHES/GHEC compatibility.
		// Custom frontmatter jobs run as independent GitHub Actions jobs that
		// don't inherit GITHUB_ENV from the agent job, so the gh CLI won't
		// know which host to target without this step.
		job.Steps = append(job.Steps, generateGHESHostConfigurationStep())

		// Inject gh-aw setup + memory restore steps when restore-memory is requested.
		// Setup lines come first (they install scripts needed by repo/comment memory).
		// Memory lines follow immediately after (restore/clone/prepare steps).
		if hasRestoreMemory {
			memorySetupLines, memoryRestoreLines, memErr := c.buildRestoreMemorySteps(restoreMemCfg, jobName, data)
			if memErr != nil {
				return memErr
			}
			job.Steps = append(job.Steps, memorySetupLines...)
			job.Steps = append(job.Steps, memoryRestoreLines...)
		}

		job.Steps = append(job.Steps, preSteps...)
		job.Steps = append(job.Steps, regularSteps...)
	}

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
		_, hasSteps := configMap["steps"]
		if err := validateRestrictedBuiltinSetupSteps(jobName, hasSetupSteps); err != nil {
			return err
		}
		if !hasSetupSteps && !hasPreSteps && !hasSteps {
			continue
		}

		targetJobName := jobName
		if jobName == "pre-activation" {
			targetJobName = string(constants.PreActivationJobName)
		}

		if err := validateRestrictedBuiltinSteps(jobName, targetJobName, hasSteps); err != nil {
			return err
		}

		job, exists := c.jobManager.GetJob(targetJobName)
		if !exists {
			continue
		}

		var setupSteps []string
		var preSteps []string
		var regularSteps []string
		if hasSetupSteps {
			steps, err := c.extractPinnedJobSteps("setup-steps", jobName, configMap, data)
			if err != nil {
				return fmt.Errorf("setup-steps for built-in job '%s' could not be processed: %w. Check that setup-steps is an array of valid step objects", jobName, err)
			}
			setupSteps = append(setupSteps, steps...)
		}
		if hasPreSteps {
			steps, err := c.extractPinnedJobSteps("pre-steps", jobName, configMap, data)
			if err != nil {
				return fmt.Errorf("pre-steps for built-in job '%s' could not be processed: %w. Check that pre-steps is an array of valid step objects", jobName, err)
			}
			preSteps = append(preSteps, steps...)
		}
		if hasSteps && targetJobName == string(constants.ActivationJobName) {
			steps, err := c.extractPinnedJobSteps("steps", jobName, configMap, data)
			if err != nil {
				return fmt.Errorf("steps for built-in job '%s' could not be processed: %w. Check that steps is an array of valid step objects", jobName, err)
			}
			regularSteps = append(regularSteps, steps...)
		}
		if len(setupSteps) == 0 && len(preSteps) == 0 && len(regularSteps) == 0 {
			continue
		}

		job.Steps = insertActivationStepsBeforeArtifactStaging(targetJobName, job.Steps, regularSteps)
		job.Steps = insertPreStepsAtEarliestBoundary(job.Steps, preSteps)
		job.Steps = insertSetupStepsAtStart(job.Steps, setupSteps)
		compilerJobsLog.Printf("Inserted %d setup-step(s), %d pre-step(s), and %d step(s) into built-in job '%s'", len(setupSteps), len(preSteps), len(regularSteps), targetJobName)
	}

	return nil
}

func insertActivationStepsBeforeArtifactStaging(jobName string, steps []string, activationSteps []string) []string {
	if len(activationSteps) == 0 {
		return steps
	}
	if jobName != string(constants.ActivationJobName) {
		return steps
	}

	insertIdx := len(steps)
	for i, step := range steps {
		if strings.Contains(step, "name: "+constants.ActivationStageAmbientFoldersStepName) ||
			strings.Contains(step, "name: "+constants.ActivationUploadArtifactStepName) {
			insertIdx = i
			break
		}
	}

	result := make([]string, 0, safeAllocationCapacity(len(steps), len(activationSteps)))
	result = append(result, steps[:insertIdx]...)
	result = append(result, activationSteps...)
	result = append(result, steps[insertIdx:]...)
	return result
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
		return nil, fmt.Errorf("jobs.%s.needs expects a string or array of strings, got %T. Example: needs: [build, test]", jobName, needsValue)
	}
}

func extractBuiltinJobIfAugmentation(jobName string, configMap map[string]any) (string, error) {
	ifValue, exists := configMap["if"]
	if !exists || ifValue == nil {
		return "", nil
	}

	ifCondition, ok := ifValue.(string)
	if !ok {
		return "", fmt.Errorf("jobs.%s.if expects a string, got %T. Example: if: github.event_name == 'push'", jobName, ifValue)
	}

	// Strip "if: " prefix to match the Job.If contract (bare expression, no prefix).
	// This mirrors how custom jobs normalize their if fields via extractExpressionFromIfString.
	if strings.HasPrefix(ifCondition, "if: ") {
		ifCondition = strings.TrimSpace(ifCondition[4:])
	}

	return ifCondition, nil
}

// applyBuiltinJobAugmentations merges jobs.<built-in>.needs and jobs.<built-in>.if into
// compiler-generated jobs. needs entries are added additively; if conditions are combined
// with compiler-generated conditions via logical AND. Both augmentations are additive-only
// and never remove compiler-computed behavior.
func (c *Compiler) applyBuiltinJobAugmentations(data *WorkflowData) error {
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
			return fmt.Errorf("jobs.%s expects an object, got %T. Example: jobs:\n  %s:\n    runs-on: ubuntu-latest", configuredJobName, rawConfig, configuredJobName)
		}

		augmentedNeeds, err := extractBuiltinJobNeedsAugmentation(configuredJobName, configMap)
		if err != nil {
			return err
		}
		augmentedIf, err := extractBuiltinJobIfAugmentation(configuredJobName, configMap)
		if err != nil {
			return err
		}
		_, hasPermissions := configMap["permissions"]
		if len(augmentedNeeds) == 0 && augmentedIf == "" && !hasPermissions {
			continue
		}

		targetJob, exists := c.jobManager.GetJob(targetJobName)
		if !exists {
			// Report the actual field(s) the author configured so they can identify the problem.
			augmentedField := configuredJobName + ".needs"
			if len(augmentedNeeds) == 0 {
				if augmentedIf != "" {
					augmentedField = configuredJobName + ".if"
				} else {
					augmentedField = configuredJobName + ".permissions"
				}
			} else if augmentedIf != "" || hasPermissions {
				augmentedField = configuredJobName
			}
			return fmt.Errorf("jobs.%s requires an existing built-in job %q, but this workflow does not generate it. Add the corresponding trigger/feature, or rename the job", augmentedField, targetJobName)
		}

		if hasPermissions {
			if err := applyBuiltinJobPermissionsAugmentation(configuredJobName, targetJobName, configMap, targetJob); err != nil {
				return err
			}
		}

		normalizedNeeds := make([]string, 0, len(augmentedNeeds))
		for _, rawNeed := range augmentedNeeds {
			need := normalizeBuiltinJobAlias(rawNeed)
			if need == targetJobName {
				return fmt.Errorf("jobs.%s.needs lists %q, but a job should not depend on itself. Remove the self-reference from needs", configuredJobName, rawNeed)
			}
			if _, known := allJobs[need]; !known {
				return fmt.Errorf("jobs.%s.needs: unknown job %q. Expected a job defined in this workflow or a generated built-in job. Example:\njobs:\n  %s:\n    needs: [activation]", configuredJobName, rawNeed, configuredJobName)
			}
			normalizedNeeds = append(normalizedNeeds, need)
		}

		// Capture compiler-owned needs before adding user-supplied ones.
		// These are the prerequisites the compiler established (e.g. activation) and
		// must be guarded explicitly when the user condition contains a status function.
		compilerOwnedNeeds := make([]string, len(targetJob.Needs))
		copy(compilerOwnedNeeds, targetJob.Needs)

		seen := make(map[string]struct{}, len(targetJob.Needs)+len(normalizedNeeds))
		mergedNeeds := make([]string, 0, len(targetJob.Needs)+len(normalizedNeeds))
		for _, need := range targetJob.Needs {
			if _, alreadySeen := seen[need]; alreadySeen {
				continue
			}
			seen[need] = struct{}{}
			mergedNeeds = append(mergedNeeds, need)
		}
		for _, need := range normalizedNeeds {
			if _, alreadySeen := seen[need]; alreadySeen {
				continue
			}
			seen[need] = struct{}{}
			mergedNeeds = append(mergedNeeds, need)
		}
		targetJob.Needs = mergedNeeds
		if augmentedIf != "" {
			// Guard against status-function bypasses: when the user condition contains a
			// status function (always, failure, cancelled, success), GitHub Actions removes
			// the implicit success() check for all needs. Compiler-owned prerequisites such
			// as activation perform security and permission checks and must always succeed,
			// so we add explicit result == 'success' guards for those jobs only.
			guardedIf := c.guardIfAgainstStatusFuncBypass(augmentedIf, compilerOwnedNeeds)
			targetJob.If = c.combineJobIfConditions(targetJob.If, guardedIf)
			compilerJobsLog.Printf("Applied jobs.%s.if augmentation to %q", configuredJobName, targetJobName)
		}
		if len(normalizedNeeds) > 0 {
			compilerJobsLog.Printf("Applied jobs.%s.needs augmentation to %q: %v", configuredJobName, targetJobName, normalizedNeeds)
		}
	}
	return nil
}

// applyBuiltinJobPermissionsAugmentation merges user-declared jobs.<built-in>.permissions
// into a compiler-generated built-in job (e.g. safe_outputs, conclusion). The merge is
// additive: the compiler-computed permissions are preserved and the user's declared scopes
// are added on top, with write overriding read. This ensures scopes such as id-token: write
// that authors declare under jobs.*.permissions are retained in the compiled lock file rather
// than being dropped by the minimal least-privilege permission computation.
func applyBuiltinJobPermissionsAugmentation(configuredJobName, targetJobName string, configMap map[string]any, targetJob *Job) error {
	permissionsValue, exists := configMap["permissions"]
	if !exists || permissionsValue == nil {
		return nil
	}

	userPermissions := NewPermissionsParserFromValue(permissionsValue).ToPermissions()
	if userPermissions == nil {
		return nil
	}

	// Start from the compiler-computed permissions already rendered on the job, then merge
	// the user-declared permissions additively so no compiler-required scope is lost.
	merged := NewPermissionsParser(targetJob.Permissions).ToPermissions()
	merged.Merge(userPermissions)
	targetJob.Permissions = merged.RenderToYAML()
	compilerJobsLog.Printf("Applied jobs.%s.permissions augmentation to %q", configuredJobName, targetJobName)
	return nil
}

// guardIfAgainstStatusFuncBypass returns userCondition augmented with explicit
// needs.<need>.result == 'success' guards for each compiler-owned prerequisite, but only
// when userCondition contains a GitHub Actions status function (always, failure, cancelled,
// success).
//
// GitHub Actions removes the implicit success() check for ALL needs entries the moment any
// status function appears in a job's if expression. Compiler-owned prerequisites such as
// activation perform security and permission checks; they must always succeed before the
// target job runs. This function makes those guards explicit so user-supplied status functions
// cannot inadvertently (or intentionally) bypass them. User-supplied needs are intentionally
// excluded: authors choose their own result semantics for setup jobs they own.
func (c *Compiler) guardIfAgainstStatusFuncBypass(userCondition string, compilerNeeds []string) string {
	if len(compilerNeeds) == 0 {
		return userCondition
	}

	// Use string-based detection: the expression parser tokenises status function calls such as
	// failure() as single ExpressionNode literals, so AST-based containsStatusFunc cannot be used
	// here. A substring check is sufficient since GitHub Actions has a fixed, well-known set of
	// status functions and user-defined functions are not supported in the expression language.
	bare := stripExpressionWrapper(userCondition)
	if !ifExpressionContainsStatusFunc(bare) {
		return userCondition
	}

	// Build explicit success guards for each compiler-owned prerequisite.
	compilerJobsLog.Printf("Status function detected in user if condition; adding explicit success guards for compiler needs: %v", compilerNeeds)
	combined := ConditionNode(&ExpressionNode{Expression: bare})
	for _, need := range compilerNeeds {
		guard := &ExpressionNode{Expression: fmt.Sprintf("needs.%s.result == 'success'", need)}
		combined = BuildAnd(combined, guard)
	}
	return RenderCondition(combined)
}

// ifExpressionContainsStatusFunc reports whether the GitHub Actions expression string
// contains a call to any of the four status check functions (always, success, failure,
// cancelled). When present, GitHub Actions removes the implicit success() gate that would
// otherwise be applied to all needs entries.
func ifExpressionContainsStatusFunc(expr string) bool {
	return strings.Contains(expr, "always(") ||
		strings.Contains(expr, "success(") ||
		strings.Contains(expr, "failure(") ||
		strings.Contains(expr, "cancelled(")
}

// validateRestrictedBuiltinSetupSteps rejects jobs.<name>.setup-steps for the
// activation and pre-activation jobs. setup-steps run before any
// compiler-generated token-mint or short-circuit protection steps, so
// allowing arbitrary user-authored steps there could bypass those
// protections. By contrast, jobs.activation.steps (see
// validateRestrictedBuiltinSteps) is inserted later in the job, before
// artifact staging but after the activation gate/output has already run, so
// it is not equivalent to setup-steps and is intentionally allowed for the
// activation job. Injected steps content is still scanned for GitHub CLI
// write-command usage (see cacheActivationPreStepPermissions) regardless of
// which field it came from.
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

// validateRestrictedBuiltinSteps rejects jobs.<name>.steps on built-in jobs
// other than activation. Unlike setup-steps and pre-steps, steps are only
// applied to the activation job (inserted before artifact staging); silently
// accepting the field on other built-in jobs (e.g. pre-activation,
// safe_outputs) would discard the user's configuration without feedback.
// Custom (non-built-in) jobs are unaffected since their steps field is a
// regular job definition field, not an injection field.
func validateRestrictedBuiltinSteps(jobName string, targetJobName string, hasSteps bool) error {
	if !hasSteps || !isBuiltinJobName(targetJobName) || targetJobName == string(constants.ActivationJobName) {
		return nil
	}
	// jobs.pre-activation.steps is a distinct, already-validated custom field
	// (see extractPreActivationCustomFields) handled outside of this
	// built-in setup/pre-steps injection path; it is not subject to this
	// restriction.
	if targetJobName == string(constants.PreActivationJobName) {
		return nil
	}

	return fmt.Errorf(
		"jobs.%s.steps is not allowed: steps are only supported for the activation job",
		jobName,
	)
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

	firstCheckoutIdx := -1
	firstTokenMintIdx := -1
	lastSetupIdx := -1
	for i, step := range steps {
		if firstCheckoutIdx == -1 && strings.Contains(step, "uses: actions/checkout@") {
			firstCheckoutIdx = i
			// Walk backward to the checkout step's list-item boundary ("- ").
			// If no boundary is found, keep the current index so insertion still
			// occurs before the checkout uses-line.
			for j := i; j >= 0; j-- {
				trimmed := strings.TrimLeft(steps[j], " ")
				if strings.HasPrefix(trimmed, "- ") {
					firstCheckoutIdx = j
					break
				}
			}
		}
		if firstTokenMintIdx == -1 && strings.Contains(step, "uses: actions/create-github-app-token@") {
			firstTokenMintIdx = i
			// Walk backward to the token-mint step's list-item boundary ("- ").
			// If no boundary is found, keep the current index so insertion still
			// occurs before the token-mint uses-line.
			for j := i; j >= 0; j-- {
				trimmed := strings.TrimLeft(steps[j], " ")
				if strings.HasPrefix(trimmed, "- ") {
					firstTokenMintIdx = j
					break
				}
			}
		}
		if exactSetupStepIDPattern.MatchString(step) {
			lastSetupIdx = i
		}
	}

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

	result := make([]string, 0, safeAllocationCapacity(len(steps), len(preSteps)))
	result = append(result, steps[:insertIdx]...)
	result = append(result, preSteps...)
	result = append(result, steps[insertIdx:]...)
	return result
}

func (c *Compiler) extractPinnedJobSteps(fieldName string, jobName string, configMap map[string]any, data *WorkflowData) ([]string, error) {
	raw, hasField := configMap[fieldName]
	if !hasField {
		return nil, nil
	}

	stepsList, ok := raw.([]any)
	if !ok {
		return nil, fmt.Errorf("%s for job '%s' expects an array of step objects. Example: %s:\n  - run: echo hello", fieldName, jobName, fieldName)
	}

	pinnedSteps := make([]string, 0, len(stepsList))
	for i, step := range stepsList {
		stepMap, ok := step.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("%s for job '%s' has a step at index %d that is not an object. Expected each entry to be a step mapping. Example: %s:\n  - run: echo hello", fieldName, jobName, i, fieldName)
		}

		typedStep, err := MapToStep(stepMap)
		if err != nil {
			return nil, fmt.Errorf("%s entry for job '%s' could not be converted to a step: %w. Check that each step has valid fields such as run, uses, or with", fieldName, jobName, err)
		}

		pinnedStep, err := applyActionPinToTypedStep(typedStep, data)
		if err != nil {
			return nil, fmt.Errorf("action in %s for job '%s' could not be pinned: %w. Check that the 'uses' field references a valid action and version", fieldName, jobName, err)
		}
		finalStepMap := pinnedStep.ToMap()
		ensureCheckoutPersistCredentials(finalStepMap)
		sanitizedMap, warnings, _ := sanitizeRunStepExpressions(finalStepMap)
		for _, w := range warnings {
			compilerJobsLog.Printf("sanitized run: expression in job '%s' step: %s", jobName, w)
		}
		stepYAML, err := ConvertStepToYAML(sanitizedMap)
		if err != nil {
			return nil, fmt.Errorf("%s for job '%s' could not be converted to YAML: %w. Check that each step is a valid object", fieldName, jobName, err)
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
