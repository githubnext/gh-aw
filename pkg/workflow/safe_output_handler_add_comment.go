package workflow

import (
	"encoding/json"

	"github.com/githubnext/gh-aw/pkg/logger"
)

var addCommentHandlerLog = logger.New("workflow:safe_output_handler_add_comment")

// AddCommentHandler handles add_comment safe output messages
type AddCommentHandler struct{}

// NewAddCommentHandler creates a new add_comment handler
func NewAddCommentHandler() *AddCommentHandler {
	return &AddCommentHandler{}
}

// GetType returns the type identifier for this handler
func (h *AddCommentHandler) GetType() string {
	return "add_comment"
}

// IsEnabled checks if add_comment is enabled in the workflow configuration
func (h *AddCommentHandler) IsEnabled(data *WorkflowData) bool {
	return data.SafeOutputs != nil && data.SafeOutputs.AddComments != nil
}

// BuildStepConfig builds the step configuration for add_comment
func (h *AddCommentHandler) BuildStepConfig(c *Compiler, data *WorkflowData, ctx *SafeOutputContext) *SafeOutputStepConfig {
	if !h.IsEnabled(data) {
		return nil
	}

	addCommentHandlerLog.Printf("Building add_comment step: target=%s, discussion=%v",
		data.SafeOutputs.AddComments.Target,
		data.SafeOutputs.AddComments.Discussion != nil && *data.SafeOutputs.AddComments.Discussion)

	// Build custom environment variables specific to add-comment
	var customEnvVars []string

	// Pass the comment target configuration
	if data.SafeOutputs.AddComments.Target != "" {
		customEnvVars = append(customEnvVars, "          GH_AW_COMMENT_TARGET: \""+data.SafeOutputs.AddComments.Target+"\"\n")
	}

	// Pass the discussion flag configuration
	if data.SafeOutputs.AddComments.Discussion != nil && *data.SafeOutputs.AddComments.Discussion {
		customEnvVars = append(customEnvVars, "          GITHUB_AW_COMMENT_DISCUSSION: \"true\"\n")
	}

	// Pass the hide-older-comments flag configuration
	if data.SafeOutputs.AddComments.HideOlderComments {
		customEnvVars = append(customEnvVars, "          GH_AW_HIDE_OLDER_COMMENTS: \"true\"\n")
	}

	// Pass the allowed-reasons list configuration
	if len(data.SafeOutputs.AddComments.AllowedReasons) > 0 {
		reasonsJSON, err := json.Marshal(data.SafeOutputs.AddComments.AllowedReasons)
		if err == nil {
			customEnvVars = append(customEnvVars, "          GH_AW_ALLOWED_REASONS: \""+string(reasonsJSON)+"\"\n")
		}
	}

	// Add environment variables for outputs from previously processed handlers
	if ctx.IsProcessed("create_issue") {
		customEnvVars = append(customEnvVars, "          GH_AW_CREATED_ISSUE_URL: ${{ steps.create_issue.outputs.issue_url }}\n")
		customEnvVars = append(customEnvVars, "          GH_AW_CREATED_ISSUE_NUMBER: ${{ steps.create_issue.outputs.issue_number }}\n")
	}

	if ctx.IsProcessed("create_discussion") {
		customEnvVars = append(customEnvVars, "          GH_AW_CREATED_DISCUSSION_URL: ${{ steps.create_discussion.outputs.discussion_url }}\n")
		customEnvVars = append(customEnvVars, "          GH_AW_CREATED_DISCUSSION_NUMBER: ${{ steps.create_discussion.outputs.discussion_number }}\n")
	}

	// Note: create_pull_request is not in the initial migration list, but keeping reference structure
	// for when it's migrated later

	// Add temporary ID map if available
	if ctx.TempIDMapAvailable {
		customEnvVars = append(customEnvVars, "          GH_AW_TEMPORARY_ID_MAP: ${{ steps."+ctx.TempIDMapSource+".outputs.temporary_id_map }}\n")
		addCommentHandlerLog.Printf("Added temporary ID map reference from step: %s", ctx.TempIDMapSource)
	}

	// Build job condition with event check if target is not specified
	condition := BuildSafeOutputType("add_comment")
	if data.SafeOutputs.AddComments != nil && data.SafeOutputs.AddComments.Target == "" {
		eventCondition := BuildOr(
			BuildOr(
				BuildPropertyAccess("github.event.issue.number"),
				BuildPropertyAccess("github.event.pull_request.number"),
			),
			BuildPropertyAccess("github.event.discussion.number"),
		)
		condition = BuildAnd(condition, eventCondition)
	}
	if ctx.ThreatDetectionEnabled {
		condition = BuildAnd(condition, buildDetectionSuccessCondition())
	}

	// Create outputs map
	outputs := map[string]string{
		"comment_id":  "${{ steps.add_comment.outputs.comment_id }}",
		"comment_url": "${{ steps.add_comment.outputs.comment_url }}",
	}

	return &SafeOutputStepConfig{
		StepName:        "Add Comment",
		StepID:          "add_comment",
		ScriptName:      "add_comment",
		CustomEnvVars:   customEnvVars,
		Condition:       condition,
		Token:           data.SafeOutputs.AddComments.GitHubToken,
		UseCopilotToken: false,
		UseAgentToken:   false,
		PreSteps:        nil,
		PostSteps:       nil,
		Outputs:         outputs,
	}
}

// GetOutputs returns the outputs that add_comment produces
func (h *AddCommentHandler) GetOutputs() map[string]string {
	return map[string]string{
		"add_comment_comment_id":  "${{ steps.add_comment.outputs.comment_id }}",
		"add_comment_comment_url": "${{ steps.add_comment.outputs.comment_url }}",
	}
}

// RequiresTempIDMap returns true if this handler needs access to the temporary ID map
func (h *AddCommentHandler) RequiresTempIDMap() bool {
	return true // add_comment consumes the temporary ID map from create_issue
}
