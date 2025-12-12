package workflow

import (
	"fmt"

	"github.com/githubnext/gh-aw/pkg/logger"
)

var assignToUserLog = logger.New("workflow:assign_to_user")

// AssignToUserConfig holds configuration for assigning users to issues from agent output
type AssignToUserConfig struct {
	BaseSafeOutputConfig   `yaml:",inline"`
	SafeOutputTargetConfig `yaml:",inline"`
	Allowed                []string `yaml:"allowed,omitempty"` // Optional list of allowed usernames. If omitted, any users are allowed.
}

// buildAssignToUserJob creates the assign_to_user job
func (c *Compiler) buildAssignToUserJob(data *WorkflowData, mainJobName string) (*Job, error) {
	assignToUserLog.Printf("Building assign_to_user job for workflow: %s, main_job: %s", data.Name, mainJobName)

	if data.SafeOutputs == nil || data.SafeOutputs.AssignToUser == nil {
		return nil, fmt.Errorf("safe-outputs.assign-to-user configuration is required")
	}

	cfg := data.SafeOutputs.AssignToUser

	// Build list job config
	listJobConfig := ListJobConfig{
		SafeOutputTargetConfig: cfg.SafeOutputTargetConfig,
		Allowed:                cfg.Allowed,
	}

	// Build extra condition: only run if in issue context when target is not specified
	var extraCondition ConditionNode
	if cfg.Target == "" {
		extraCondition = BuildPropertyAccess("github.event.issue.number")
	}

	// Use shared builder for list-based safe-output jobs
	return c.BuildListSafeOutputJob(data, mainJobName, listJobConfig, cfg.BaseSafeOutputConfig, ListJobBuilderConfig{
		JobName:        "assign_to_user",
		StepName:       "Assign to User",
		StepID:         "assign_to_user",
		EnvPrefix:      "GH_AW_ASSIGNEES",
		OutputName:     "assigned_users",
		Script:         getAssignToUserScript(),
		Permissions:    NewPermissionsContentsReadIssuesWrite(),
		DefaultMax:     1,
		ExtraCondition: extraCondition,
	})
}

// parseAssignToUserConfig handles assign-to-user configuration
func (c *Compiler) parseAssignToUserConfig(outputMap map[string]any) *AssignToUserConfig {
	if configData, exists := outputMap["assign-to-user"]; exists {
		assignToUserLog.Print("Parsing assign-to-user configuration")
		assignToUserConfig := &AssignToUserConfig{}

		if configMap, ok := configData.(map[string]any); ok {
			// Parse list job config (target, target-repo, allowed)
			listJobConfig, _ := ParseListJobConfig(configMap, "allowed")
			assignToUserConfig.SafeOutputTargetConfig = listJobConfig.SafeOutputTargetConfig
			assignToUserConfig.Allowed = listJobConfig.Allowed

			// Parse common base fields (github-token, max) with default max of 1
			c.parseBaseSafeOutputConfig(configMap, &assignToUserConfig.BaseSafeOutputConfig, 1)
			assignToUserLog.Printf("Parsed configuration: allowed_count=%d, target=%s", len(assignToUserConfig.Allowed), assignToUserConfig.Target)
		} else {
			// If configData is nil or not a map (e.g., "assign-to-user:" with no value),
			// use defaults
			assignToUserLog.Print("Using default configuration")
			assignToUserConfig.Max = 1
		}

		return assignToUserConfig
	}

	return nil
}
