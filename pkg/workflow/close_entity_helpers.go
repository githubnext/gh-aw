package workflow

import (
	"fmt"

	"github.com/githubnext/gh-aw/pkg/logger"
)

// CloseEntityType represents the type of entity being closed
type CloseEntityType string

const (
	CloseEntityIssue       CloseEntityType = "issue"
	CloseEntityPullRequest CloseEntityType = "pull_request"
	CloseEntityDiscussion  CloseEntityType = "discussion"
)

// CloseEntityConfig holds the configuration for a close entity operation
type CloseEntityConfig struct {
	BaseSafeOutputConfig             `yaml:",inline"`
	SafeOutputTargetConfig           `yaml:",inline"`
	SafeOutputFilterConfig           `yaml:",inline"`
	SafeOutputDiscussionFilterConfig `yaml:",inline"` // Only used for discussions
}

// CloseEntityJobParams holds the parameters needed to build a close entity job
type CloseEntityJobParams struct {
	EntityType       CloseEntityType
	ConfigKey        string // e.g., "close-issue", "close-pull-request"
	EnvVarPrefix     string // e.g., "GH_AW_CLOSE_ISSUE", "GH_AW_CLOSE_PR"
	JobName          string // e.g., "close_issue", "close_pull_request"
	StepName         string // e.g., "Close Issue", "Close Pull Request"
	OutputNumberKey  string // e.g., "issue_number", "pull_request_number"
	OutputURLKey     string // e.g., "issue_url", "pull_request_url"
	EventNumberPath1 string // e.g., "github.event.issue.number"
	EventNumberPath2 string // e.g., "github.event.comment.issue.number"
	ScriptGetter     func() string
	PermissionsFunc  func() *Permissions
}

// parseCloseEntityConfig is a generic function to parse close entity configurations
func (c *Compiler) parseCloseEntityConfig(outputMap map[string]any, params CloseEntityJobParams, logger *logger.Logger) *CloseEntityConfig {
	if configData, exists := outputMap[params.ConfigKey]; exists {
		logger.Printf("Parsing %s configuration", params.ConfigKey)
		config := &CloseEntityConfig{}

		if configMap, ok := configData.(map[string]any); ok {
			// For discussions, parse target and filter configs separately
			if params.EntityType == CloseEntityDiscussion {
				logger.Printf("Parsing discussion-specific configuration for %s", params.ConfigKey)
				// Parse target config
				targetConfig, isInvalid := ParseTargetConfig(configMap)
				if isInvalid {
					logger.Print("Invalid target-repo configuration")
					return nil
				}
				config.SafeOutputTargetConfig = targetConfig

				// Parse discussion filter config (includes required-category)
				config.SafeOutputDiscussionFilterConfig = ParseDiscussionFilterConfig(configMap)
				config.SafeOutputFilterConfig = config.SafeOutputDiscussionFilterConfig.SafeOutputFilterConfig
			} else {
				logger.Printf("Parsing standard close job configuration for %s", params.EntityType)
				// For issues and PRs, use the standard close job config parser
				closeJobConfig, isInvalid := ParseCloseJobConfig(configMap)
				if isInvalid {
					logger.Print("Invalid target-repo configuration")
					return nil
				}
				config.SafeOutputTargetConfig = closeJobConfig.SafeOutputTargetConfig
				config.SafeOutputFilterConfig = closeJobConfig.SafeOutputFilterConfig
			}

			// Parse common base fields with default max of 1
			c.parseBaseSafeOutputConfig(configMap, &config.BaseSafeOutputConfig, 1)
			logger.Printf("Parsed %s configuration: max=%d, target=%s", params.ConfigKey, config.Max, config.Target)
		} else {
			// If configData is nil or not a map, still set the default max
			logger.Print("Config is not a map, using default max=1")
			config.Max = 1
		}

		return config
	}

	logger.Printf("No configuration found for %s", params.ConfigKey)
	return nil
}

// buildCloseEntityJob is a generic function to build close entity jobs
func (c *Compiler) buildCloseEntityJob(data *WorkflowData, mainJobName string, config *CloseEntityConfig, params CloseEntityJobParams, logger *logger.Logger) (*Job, error) {
	logger.Printf("Building %s job for workflow: %s", params.JobName, data.Name)

	if config == nil {
		return nil, fmt.Errorf("safe-outputs.%s configuration is required", params.ConfigKey)
	}

	// Build custom environment variables specific to this close entity
	closeJobConfig := CloseJobConfig{
		SafeOutputTargetConfig: config.SafeOutputTargetConfig,
		SafeOutputFilterConfig: config.SafeOutputFilterConfig,
	}
	customEnvVars := BuildCloseJobEnvVars(params.EnvVarPrefix, closeJobConfig)

	// Add required-category env var for discussions
	if params.EntityType == CloseEntityDiscussion {
		customEnvVars = append(customEnvVars, BuildRequiredCategoryEnvVar(params.EnvVarPrefix+"_REQUIRED_CATEGORY", config.RequiredCategory)...)
	}

	logger.Printf("Configured %d custom environment variables for %s close", len(customEnvVars), params.EntityType)

	// Add standard environment variables (metadata + staged/target repo)
	customEnvVars = append(customEnvVars, c.buildStandardSafeOutputEnvVars(data, config.TargetRepoSlug)...)

	// Create outputs for the job
	outputs := map[string]string{
		params.OutputNumberKey: fmt.Sprintf("${{ steps.%s.outputs.%s }}", params.JobName, params.OutputNumberKey),
		params.OutputURLKey:    fmt.Sprintf("${{ steps.%s.outputs.%s }}", params.JobName, params.OutputURLKey),
		"comment_url":          fmt.Sprintf("${{ steps.%s.outputs.comment_url }}", params.JobName),
	}

	// Build job condition with event check only for "triggering" target
	// If target is "*" (any entity) or explicitly set, allow agent to provide the entity number
	jobCondition := BuildSafeOutputType(params.JobName)
	if config.Target == "" || config.Target == "triggering" {
		// Only require event context for "triggering" target
		eventCondition := BuildOr(
			BuildPropertyAccess(params.EventNumberPath1),
			BuildPropertyAccess(params.EventNumberPath2),
		)
		jobCondition = BuildAnd(jobCondition, eventCondition)
	}

	// Use the shared builder function to create the job
	return c.buildSafeOutputJob(data, SafeOutputJobConfig{
		JobName:        params.JobName,
		StepName:       params.StepName,
		StepID:         params.JobName,
		MainJobName:    mainJobName,
		CustomEnvVars:  customEnvVars,
		Script:         params.ScriptGetter(),
		Permissions:    params.PermissionsFunc(),
		Outputs:        outputs,
		Condition:      jobCondition,
		Token:          config.GitHubToken,
		TargetRepoSlug: config.TargetRepoSlug,
	})
}

// closeEntityDefinition holds all parameters for a close entity type
type closeEntityDefinition struct {
	EntityType       CloseEntityType
	ConfigKey        string
	EnvVarPrefix     string
	JobName          string
	StepName         string
	OutputNumberKey  string
	OutputURLKey     string
	EventNumberPath1 string
	EventNumberPath2 string
	ScriptGetter     func() string
	PermissionsFunc  func() *Permissions
	Logger           *logger.Logger
}

// closeEntityRegistry holds all close entity definitions
var closeEntityRegistry = []closeEntityDefinition{
	{
		EntityType:       CloseEntityIssue,
		ConfigKey:        "close-issue",
		EnvVarPrefix:     "GH_AW_CLOSE_ISSUE",
		JobName:          "close_issue",
		StepName:         "Close Issue",
		OutputNumberKey:  "issue_number",
		OutputURLKey:     "issue_url",
		EventNumberPath1: "github.event.issue.number",
		EventNumberPath2: "github.event.comment.issue.number",
		ScriptGetter:     getCloseIssueScript,
		PermissionsFunc:  NewPermissionsContentsReadIssuesWrite,
		Logger:           logger.New("workflow:close_issue"),
	},
	{
		EntityType:       CloseEntityPullRequest,
		ConfigKey:        "close-pull-request",
		EnvVarPrefix:     "GH_AW_CLOSE_PR",
		JobName:          "close_pull_request",
		StepName:         "Close Pull Request",
		OutputNumberKey:  "pull_request_number",
		OutputURLKey:     "pull_request_url",
		EventNumberPath1: "github.event.pull_request.number",
		EventNumberPath2: "github.event.comment.pull_request.number",
		ScriptGetter:     getClosePullRequestScript,
		PermissionsFunc:  NewPermissionsContentsReadPRWrite,
		Logger:           logger.New("workflow:close_pull_request"),
	},
	{
		EntityType:       CloseEntityDiscussion,
		ConfigKey:        "close-discussion",
		EnvVarPrefix:     "GH_AW_CLOSE_DISCUSSION",
		JobName:          "close_discussion",
		StepName:         "Close Discussion",
		OutputNumberKey:  "discussion_number",
		OutputURLKey:     "discussion_url",
		EventNumberPath1: "github.event.discussion.number",
		EventNumberPath2: "github.event.comment.discussion.number",
		ScriptGetter:     getCloseDiscussionScript,
		PermissionsFunc:  NewPermissionsContentsReadDiscussionsWrite,
		Logger:           logger.New("workflow:close_discussion"),
	},
}

// Type aliases for backward compatibility
type CloseIssuesConfig = CloseEntityConfig
type ClosePullRequestsConfig = CloseEntityConfig
type CloseDiscussionsConfig = CloseEntityConfig

// parseCloseIssuesConfig handles close-issue configuration
func (c *Compiler) parseCloseIssuesConfig(outputMap map[string]any) *CloseIssuesConfig {
	def := closeEntityRegistry[0] // issue
	params := CloseEntityJobParams{
		EntityType: def.EntityType,
		ConfigKey:  def.ConfigKey,
	}
	return c.parseCloseEntityConfig(outputMap, params, def.Logger)
}

// parseClosePullRequestsConfig handles close-pull-request configuration
func (c *Compiler) parseClosePullRequestsConfig(outputMap map[string]any) *ClosePullRequestsConfig {
	def := closeEntityRegistry[1] // pull request
	params := CloseEntityJobParams{
		EntityType: def.EntityType,
		ConfigKey:  def.ConfigKey,
	}
	return c.parseCloseEntityConfig(outputMap, params, def.Logger)
}

// parseCloseDiscussionsConfig handles close-discussion configuration
func (c *Compiler) parseCloseDiscussionsConfig(outputMap map[string]any) *CloseDiscussionsConfig {
	def := closeEntityRegistry[2] // discussion
	params := CloseEntityJobParams{
		EntityType: def.EntityType,
		ConfigKey:  def.ConfigKey,
	}
	return c.parseCloseEntityConfig(outputMap, params, def.Logger)
}

// buildCreateOutputCloseIssueJob creates the close_issue job
func (c *Compiler) buildCreateOutputCloseIssueJob(data *WorkflowData, mainJobName string) (*Job, error) {
	def := closeEntityRegistry[0] // issue
	params := CloseEntityJobParams{
		EntityType:       def.EntityType,
		ConfigKey:        def.ConfigKey,
		EnvVarPrefix:     def.EnvVarPrefix,
		JobName:          def.JobName,
		StepName:         def.StepName,
		OutputNumberKey:  def.OutputNumberKey,
		OutputURLKey:     def.OutputURLKey,
		EventNumberPath1: def.EventNumberPath1,
		EventNumberPath2: def.EventNumberPath2,
		ScriptGetter:     def.ScriptGetter,
		PermissionsFunc:  def.PermissionsFunc,
	}
	return c.buildCloseEntityJob(data, mainJobName, data.SafeOutputs.CloseIssues, params, def.Logger)
}

// buildCreateOutputClosePullRequestJob creates the close_pull_request job
func (c *Compiler) buildCreateOutputClosePullRequestJob(data *WorkflowData, mainJobName string) (*Job, error) {
	def := closeEntityRegistry[1] // pull request
	params := CloseEntityJobParams{
		EntityType:       def.EntityType,
		ConfigKey:        def.ConfigKey,
		EnvVarPrefix:     def.EnvVarPrefix,
		JobName:          def.JobName,
		StepName:         def.StepName,
		OutputNumberKey:  def.OutputNumberKey,
		OutputURLKey:     def.OutputURLKey,
		EventNumberPath1: def.EventNumberPath1,
		EventNumberPath2: def.EventNumberPath2,
		ScriptGetter:     def.ScriptGetter,
		PermissionsFunc:  def.PermissionsFunc,
	}
	return c.buildCloseEntityJob(data, mainJobName, data.SafeOutputs.ClosePullRequests, params, def.Logger)
}

// buildCreateOutputCloseDiscussionJob creates the close_discussion job
func (c *Compiler) buildCreateOutputCloseDiscussionJob(data *WorkflowData, mainJobName string) (*Job, error) {
	def := closeEntityRegistry[2] // discussion
	params := CloseEntityJobParams{
		EntityType:       def.EntityType,
		ConfigKey:        def.ConfigKey,
		EnvVarPrefix:     def.EnvVarPrefix,
		JobName:          def.JobName,
		StepName:         def.StepName,
		OutputNumberKey:  def.OutputNumberKey,
		OutputURLKey:     def.OutputURLKey,
		EventNumberPath1: def.EventNumberPath1,
		EventNumberPath2: def.EventNumberPath2,
		ScriptGetter:     def.ScriptGetter,
		PermissionsFunc:  def.PermissionsFunc,
	}
	return c.buildCloseEntityJob(data, mainJobName, data.SafeOutputs.CloseDiscussions, params, def.Logger)
}
