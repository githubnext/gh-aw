package workflow

import "github.com/githubnext/gh-aw/pkg/logger"

var closeDiscussionLog = logger.New("workflow:close_discussion")

// CloseDiscussionsConfig holds configuration for closing GitHub discussions from agent output
type CloseDiscussionsConfig = CloseEntityConfig

// parseCloseDiscussionsConfig handles close-discussion configuration
func (c *Compiler) parseCloseDiscussionsConfig(outputMap map[string]any) *CloseDiscussionsConfig {
	params := CloseEntityJobParams{
		EntityType: CloseEntityDiscussion,
		ConfigKey:  "close-discussion",
	}
	return c.parseCloseEntityConfig(outputMap, params, closeDiscussionLog)
}

// buildCreateOutputCloseDiscussionJob creates the close_discussion job
func (c *Compiler) buildCreateOutputCloseDiscussionJob(data *WorkflowData, mainJobName string) (*Job, error) {
	params := CloseEntityJobParams{
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
	}
	return c.buildCloseEntityJob(data, mainJobName, data.SafeOutputs.CloseDiscussions, params, closeDiscussionLog)
}
