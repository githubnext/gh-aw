package workflow

import (
	"fmt"
	"maps"
	"slices"
	"strings"

	"github.com/github/gh-aw/pkg/constants"
	"github.com/github/gh-aw/pkg/logger"
	"github.com/github/gh-aw/pkg/stringutil"
)

var safeJobsLog = logger.New("workflow:safe_jobs")

// SafeJobConfig defines a safe job configuration with GitHub Actions job properties
type SafeJobConfig struct {
	// Standard GitHub Actions job properties
	Name        string            `yaml:"name,omitempty"`
	Description string            `yaml:"description,omitempty"`
	RunsOn      any               `yaml:"runs-on,omitempty"`
	If          string            `yaml:"if,omitempty"`
	Needs       []string          `yaml:"needs,omitempty"`
	Steps       []any             `yaml:"steps,omitempty"`
	Env         map[string]string `yaml:"env,omitempty"`
	Permissions map[string]string `yaml:"permissions,omitempty"`

	// Additional safe-job specific properties
	Inputs      map[string]*InputDefinition `yaml:"inputs,omitempty"`
	GitHubToken string                      `yaml:"github-token,omitempty"`
	Output      string                      `yaml:"output,omitempty"`
	Max         int                         `yaml:"max,omitempty"` // Maximum number of times this output type may be emitted per run (default: 1)
}

// parseSafeJobsConfig parses safe-jobs configuration from a jobs map.
// This function expects a map of job configurations directly (from safe-outputs.jobs).
// The top-level "safe-jobs" key is NOT supported - only "safe-outputs.jobs" is valid.
func (c *Compiler) parseSafeJobsConfig(jobsMap map[string]any) map[string]*SafeJobConfig {
	if jobsMap == nil {
		return nil
	}

	safeJobsLog.Printf("Parsing %d safe-jobs from jobs map", len(jobsMap))
	result := make(map[string]*SafeJobConfig)

	for jobName, jobValue := range jobsMap {
		jobConfig, ok := jobValue.(map[string]any)
		if !ok {
			continue
		}
		safeJob := c.parseSafeJobConfig(jobName, jobConfig)

		safeJobsLog.Printf("Parsed safe-job configuration: name=%s, has_steps=%v, has_inputs=%v, max=%d", jobName, len(safeJob.Steps) > 0, len(safeJob.Inputs) > 0, safeJob.Max)
		result[jobName] = safeJob
	}

	return result
}

func (c *Compiler) parseSafeJobConfig(jobName string, jobConfig map[string]any) *SafeJobConfig {
	safeJob := &SafeJobConfig{}
	parseSafeJobStringFields(jobConfig, safeJob)
	if runsOn, exists := jobConfig["runs-on"]; exists {
		safeJob.RunsOn = runsOn
	} else if runner, exists := jobConfig["runner"]; exists {
		safeJob.RunsOn = runner
	}
	if ifCond, exists := jobConfig["if"]; exists {
		if ifStr, ok := ifCond.(string); ok {
			safeJob.If = c.extractExpressionFromIfString(ifStr)
		}
	}
	safeJob.Needs = parseSafeJobNeeds(jobConfig["needs"])
	if steps, exists := jobConfig["steps"]; exists {
		if stepsList, ok := steps.([]any); ok {
			safeJob.Steps = stepsList
		}
	}
	safeJob.Env = parseStringMapField(jobConfig["env"])
	safeJob.Permissions = parseStringMapField(jobConfig["permissions"])
	if inputsMap, ok := jobConfig["inputs"].(map[string]any); ok {
		safeJob.Inputs = ParseInputDefinitions(inputsMap)
	}
	parseSafeJobMax(jobName, jobConfig, safeJob)
	return safeJob
}

func parseSafeJobStringFields(jobConfig map[string]any, safeJob *SafeJobConfig) {
	if name, exists := jobConfig["name"]; exists {
		if nameStr, ok := name.(string); ok {
			safeJob.Name = nameStr
		}
	}
	if description, exists := jobConfig["description"]; exists {
		if descStr, ok := description.(string); ok {
			safeJob.Description = descStr
		}
	}
	if token, exists := jobConfig["github-token"]; exists {
		if tokenStr, ok := token.(string); ok {
			safeJob.GitHubToken = tokenStr
		}
	}
	if output, exists := jobConfig["output"]; exists {
		if outputStr, ok := output.(string); ok {
			safeJob.Output = outputStr
		}
	} else if agentOutput, exists := jobConfig["agent-output"]; exists {
		if agentOutputStr, ok := agentOutput.(string); ok {
			safeJob.Output = agentOutputStr
		}
	}
}

func parseSafeJobNeeds(needs any) []string {
	var result []string
	if needsList, ok := needs.([]any); ok {
		for _, need := range needsList {
			if needStr, ok := need.(string); ok {
				result = append(result, needStr)
			}
		}
	} else if needStr, ok := needs.(string); ok {
		result = append(result, needStr)
	}
	return result
}

func parseStringMapField(value any) map[string]string {
	valueMap, ok := value.(map[string]any)
	if !ok {
		return nil
	}
	result := make(map[string]string)
	for key, value := range valueMap {
		if valueStr, ok := value.(string); ok {
			result[key] = valueStr
		}
	}
	return result
}

func parseSafeJobMax(jobName string, jobConfig map[string]any, safeJob *SafeJobConfig) {
	maxVal, exists := jobConfig["max"]
	if !exists {
		return
	}
	maxInt := safeJobMaxInt(jobName, maxVal)
	if maxInt > 0 {
		safeJob.Max = maxInt
	}
}

func safeJobMaxInt(jobName string, maxVal any) int {
	switch v := maxVal.(type) {
	case int:
		return v
	case int64:
		return int(v)
	case uint64:
		return int(v)
	case float64:
		if v != float64(int(v)) {
			safeJobsLog.Printf("Warning: ignoring non-integer max for safe-job %q: %v", jobName, v)
			return 0
		}
		return int(v)
	default:
		safeJobsLog.Printf("Warning: ignoring non-numeric max for safe-job %q: %T", jobName, maxVal)
		return 0
	}
}

// buildSafeJobs creates custom safe-output jobs defined in SafeOutputs.Jobs
func (c *Compiler) buildSafeJobs(data *WorkflowData, threatDetectionEnabled bool) ([]string, error) {
	if data.SafeOutputs == nil || len(data.SafeOutputs.Jobs) == 0 {
		return nil, nil
	}

	safeJobsLog.Printf("Building %d safe-jobs, threatDetectionEnabled=%v", len(data.SafeOutputs.Jobs), threatDetectionEnabled)
	var safeJobNames []string

	entries := sortedSafeJobEntries(data.SafeOutputs.Jobs)

	for _, entry := range entries {
		job, err := c.buildSafeJob(entry, data, threatDetectionEnabled)
		if err != nil {
			return nil, err
		}
		if err := c.jobManager.AddJob(job); err != nil {
			safeJobsLog.Printf("Failed to add safe-job %s: %v", entry.normalizedName, err)
			return nil, fmt.Errorf("failed to add safe job %s: %w", entry.normalizedName, err)
		}
		safeJobsLog.Printf("Created safe-job: %s with %d dependencies and %d steps", entry.normalizedName, len(job.Needs), len(job.Steps))
		safeJobNames = append(safeJobNames, entry.normalizedName)
	}

	safeJobsLog.Printf("Successfully built %d safe-jobs", len(safeJobNames))
	return safeJobNames, nil
}

type safeJobEntry struct {
	normalizedName string
	config         *SafeJobConfig
}

func sortedSafeJobEntries(jobs map[string]*SafeJobConfig) []safeJobEntry {
	entries := make([]safeJobEntry, 0, len(jobs))
	for rawName, cfg := range jobs {
		entries = append(entries, safeJobEntry{stringutil.NormalizeSafeOutputIdentifier(rawName), cfg})
	}
	slices.SortFunc(entries, func(a, b safeJobEntry) int { return strings.Compare(a.normalizedName, b.normalizedName) })
	return entries
}

func (c *Compiler) buildSafeJob(entry safeJobEntry, data *WorkflowData, threatDetectionEnabled bool) (*Job, error) {
	jobConfig := entry.config
	job := &Job{Name: entry.normalizedName, Environment: c.indentYAMLLines(resolveSafeOutputsEnvironment(data), "    ")}
	if jobConfig.Name != "" {
		job.DisplayName = jobConfig.Name
	}
	job.Needs = safeJobNeeds(jobConfig, threatDetectionEnabled)
	job.RunsOn = renderSafeJobRunsOn(jobConfig.RunsOn)
	job.If = c.renderSafeJobCondition(entry.normalizedName, jobConfig, data)
	steps, err := c.buildSafeJobSteps(entry.normalizedName, jobConfig, data)
	if err != nil {
		return nil, err
	}
	job.Steps = steps
	job.Permissions = renderSafeJobPermissions(jobConfig.Permissions)
	return job, nil
}

func safeJobNeeds(jobConfig *SafeJobConfig, threatDetectionEnabled bool) []string {
	needs := []string{string(constants.AgentJobName)}
	if threatDetectionEnabled {
		needs = append(needs, string(constants.DetectionJobName))
	}
	return append(needs, jobConfig.Needs...)
}

func renderSafeJobRunsOn(runsOn any) string {
	if runsOn == nil {
		return "runs-on: ubuntu-latest"
	}
	if runsOnStr, ok := runsOn.(string); ok {
		return "runs-on: " + runsOnStr
	}
	runsOnList, ok := runsOn.([]any)
	if !ok {
		return ""
	}
	var runsOnItems []string
	for _, item := range runsOnList {
		if itemStr, ok := item.(string); ok {
			runsOnItems = append(runsOnItems, "      - "+itemStr)
		}
	}
	if len(runsOnItems) == 0 {
		return ""
	}
	return "runs-on:\n" + strings.Join(runsOnItems, "\n")
}

func (c *Compiler) renderSafeJobCondition(normalizedJobName string, jobConfig *SafeJobConfig, data *WorkflowData) string {
	safeOutputCondition := BuildSafeOutputType(normalizedJobName)
	baseCondition := safeOutputCondition
	if IsConditionalDetection(data.SafeOutputs) {
		baseCondition = BuildAnd(BuildAnd(BuildFunctionCall("always"), safeOutputCondition), buildDetectionPassedCondition())
	}
	if jobConfig.If == "" {
		return RenderCondition(baseCondition)
	}
	userCondition := &ExpressionNode{Expression: c.extractExpressionFromIfString(jobConfig.If)}
	return RenderCondition(BuildAnd(baseCondition, userCondition))
}

func (c *Compiler) buildSafeJobSteps(normalizedJobName string, jobConfig *SafeJobConfig, data *WorkflowData) ([]string, error) {
	steps := buildArtifactDownloadSteps(ArtifactDownloadConfig{
		ArtifactName: artifactPrefixExprForAgentDownstreamJob(data) + constants.AgentArtifactName,
		DownloadPath: SafeJobsDownloadDirExpr,
		SetupEnvStep: false,
		StepName:     "Download agent output artifact",
	}, c.getActionPin)
	if len(jobConfig.Steps) == 0 {
		return steps, nil
	}
	customSteps, err := buildSafeJobCustomSteps(normalizedJobName, jobConfig, data)
	if err != nil {
		return nil, err
	}
	return append(steps, customSteps...), nil
}

func buildSafeJobCustomSteps(normalizedJobName string, jobConfig *SafeJobConfig, data *WorkflowData) ([]string, error) {
	setupEnvVars := map[string]string{"GH_AW_AGENT_OUTPUT": SafeJobsDownloadDirExpr + constants.AgentOutputFilename}
	maps.Copy(setupEnvVars, jobConfig.Env)
	var steps []string
	for _, step := range jobConfig.Steps {
		stepMap, ok := step.(map[string]any)
		if !ok {
			continue
		}
		stepYAML, err := buildSafeJobCustomStepYAML(normalizedJobName, stepMap, setupEnvVars, data)
		if err != nil {
			return nil, err
		}
		steps = append(steps, stepYAML)
	}
	return steps, nil
}

func buildSafeJobCustomStepYAML(normalizedJobName string, stepMap map[string]any, setupEnvVars map[string]string, data *WorkflowData) (string, error) {
	typedStep, err := MapToStep(stepMap)
	if err != nil {
		return "", fmt.Errorf("failed to convert step to typed step for safe job %s: %w", normalizedJobName, err)
	}
	if typedStep.Env == nil {
		typedStep.Env = make(map[string]string)
	}
	for k, v := range setupEnvVars {
		if _, exists := typedStep.Env[k]; !exists {
			typedStep.Env[k] = v
		}
	}
	pinnedStep, err := applyActionPinToTypedStep(typedStep, data)
	if err != nil {
		return "", fmt.Errorf("failed to pin action for step in safe job %s: %w", normalizedJobName, err)
	}
	stepYAML, err := ConvertStepToYAML(pinnedStep.ToMap())
	if err != nil {
		return "", fmt.Errorf("failed to convert step to YAML for safe job %s: %w", normalizedJobName, err)
	}
	return stepYAML, nil
}

func renderSafeJobPermissions(config map[string]string) string {
	if len(config) == 0 {
		return ""
	}
	perms := NewPermissions()
	for perm, level := range config {
		perms.Set(PermissionScope(perm), PermissionLevel(level))
	}
	return perms.RenderToYAML()
}

// extractSafeJobsFromFrontmatter extracts safe-jobs configuration from frontmatter.
// Only checks the safe-outputs.jobs location. The top-level "safe-jobs" syntax is NOT supported.
func extractSafeJobsFromFrontmatter(frontmatter map[string]any) map[string]*SafeJobConfig {
	// Check location: safe-outputs.jobs
	if safeOutputs, exists := frontmatter["safe-outputs"]; exists {
		if safeOutputsMap, ok := safeOutputs.(map[string]any); ok {
			if jobs, exists := safeOutputsMap["jobs"]; exists {
				if jobsMap, ok := jobs.(map[string]any); ok {
					c := NewCompiler() // Create a temporary compiler instance for parsing
					return c.parseSafeJobsConfig(jobsMap)
				}
			}
		}
	}

	return make(map[string]*SafeJobConfig)
}

// mergeSafeJobs merges safe-jobs from multiple sources and detects name conflicts
func mergeSafeJobs(base map[string]*SafeJobConfig, additional map[string]*SafeJobConfig) (map[string]*SafeJobConfig, error) {
	if additional == nil {
		return base, nil
	}

	if base == nil {
		base = make(map[string]*SafeJobConfig)
	}

	result := make(map[string]*SafeJobConfig)

	// Copy base safe-jobs
	maps.Copy(result, base)

	// Add additional safe-jobs, checking for conflicts
	for name, config := range additional {
		if _, exists := result[name]; exists {
			return nil, fmt.Errorf("safe-job name conflict: '%s' is defined in both main workflow and included files", name)
		}
		result[name] = config
	}

	return result, nil
}
