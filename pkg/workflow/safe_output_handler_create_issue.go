package workflow

import (
	"fmt"

	"github.com/githubnext/gh-aw/pkg/logger"
)

var createIssueHandlerLog = logger.New("workflow:safe_output_handler_create_issue")

// CreateIssueHandler handles create_issue safe output messages
type CreateIssueHandler struct{}

// NewCreateIssueHandler creates a new create_issue handler
func NewCreateIssueHandler() *CreateIssueHandler {
	return &CreateIssueHandler{}
}

// GetType returns the type identifier for this handler
func (h *CreateIssueHandler) GetType() string {
	return "create_issue"
}

// IsEnabled checks if create_issue is enabled in the workflow configuration
func (h *CreateIssueHandler) IsEnabled(data *WorkflowData) bool {
	return data.SafeOutputs != nil && data.SafeOutputs.CreateIssues != nil
}

// BuildStepConfig builds the step configuration for create_issue
func (h *CreateIssueHandler) BuildStepConfig(c *Compiler, data *WorkflowData, ctx *SafeOutputContext) *SafeOutputStepConfig {
	if !h.IsEnabled(data) {
		return nil
	}

	createIssueHandlerLog.Printf("Building create_issue step config: workflow=%s, assignees=%d, labels=%d",
		data.Name, len(data.SafeOutputs.CreateIssues.Assignees), len(data.SafeOutputs.CreateIssues.Labels))

	// Build custom environment variables specific to create-issue
	var customEnvVars []string
	customEnvVars = append(customEnvVars, buildTitlePrefixEnvVar("GH_AW_ISSUE_TITLE_PREFIX", data.SafeOutputs.CreateIssues.TitlePrefix)...)
	customEnvVars = append(customEnvVars, buildLabelsEnvVar("GH_AW_ISSUE_LABELS", data.SafeOutputs.CreateIssues.Labels)...)
	customEnvVars = append(customEnvVars, buildLabelsEnvVar("GH_AW_ISSUE_ALLOWED_LABELS", data.SafeOutputs.CreateIssues.AllowedLabels)...)
	customEnvVars = append(customEnvVars, buildAllowedReposEnvVar("GH_AW_ALLOWED_REPOS", data.SafeOutputs.CreateIssues.AllowedRepos)...)

	// Add expires value if set
	if data.SafeOutputs.CreateIssues.Expires > 0 {
		customEnvVars = append(customEnvVars, fmt.Sprintf("          GH_AW_ISSUE_EXPIRES: \"%d\"\n", data.SafeOutputs.CreateIssues.Expires))
	}

	// Check if copilot is in assignees - if so, we'll output issues for assign_to_agent job
	assignCopilot := hasCopilotAssignee(data.SafeOutputs.CreateIssues.Assignees)
	if assignCopilot {
		customEnvVars = append(customEnvVars, "          GH_AW_ASSIGN_COPILOT: \"true\"\n")
		createIssueHandlerLog.Print("Copilot assignment requested - will output issues_to_assign_copilot")
	}

	// Build post-steps for non-copilot assignees
	var postSteps []string
	nonCopilotAssignees := filterNonCopilotAssignees(data.SafeOutputs.CreateIssues.Assignees)
	if len(nonCopilotAssignees) > 0 {
		var safeOutputsToken string
		if data.SafeOutputs != nil {
			safeOutputsToken = data.SafeOutputs.GitHubToken
		}

		postSteps = buildCopilotParticipantSteps(CopilotParticipantConfig{
			Participants:       nonCopilotAssignees,
			ParticipantType:    "assignee",
			CustomToken:        data.SafeOutputs.CreateIssues.GitHubToken,
			SafeOutputsToken:   safeOutputsToken,
			WorkflowToken:      data.GitHubToken,
			ConditionStepID:    "create_issue",
			ConditionOutputKey: "issue_number",
		})
	}

	// Add post-step for copilot assignment using agent token
	if assignCopilot {
		postSteps = append(postSteps, buildCopilotAssignmentStep(data.SafeOutputs.CreateIssues.GitHubToken)...)
	}

	// Build step condition
	condition := BuildSafeOutputType("create_issue")
	if ctx.ThreatDetectionEnabled {
		condition = BuildAnd(condition, buildDetectionSuccessCondition())
	}

	// Create outputs map
	outputs := map[string]string{
		"issue_number":     "${{ steps.create_issue.outputs.issue_number }}",
		"issue_url":        "${{ steps.create_issue.outputs.issue_url }}",
		"temporary_id_map": "${{ steps.create_issue.outputs.temporary_id_map }}",
	}

	// Add issues_to_assign_copilot output if copilot assignment is requested
	if assignCopilot {
		outputs["issues_to_assign_copilot"] = "${{ steps.create_issue.outputs.issues_to_assign_copilot }}"
	}

	return &SafeOutputStepConfig{
		StepName:        "Create Issue",
		StepID:          "create_issue",
		ScriptName:      "create_issue",
		CustomEnvVars:   customEnvVars,
		Condition:       condition,
		Token:           data.SafeOutputs.CreateIssues.GitHubToken,
		UseCopilotToken: false,
		UseAgentToken:   false,
		PreSteps:        nil,
		PostSteps:       postSteps,
		Outputs:         outputs,
	}
}

// GetOutputs returns the outputs that create_issue produces
func (h *CreateIssueHandler) GetOutputs() map[string]string {
	return map[string]string{
		"create_issue_issue_number":     "${{ steps.create_issue.outputs.issue_number }}",
		"create_issue_issue_url":        "${{ steps.create_issue.outputs.issue_url }}",
		"create_issue_temporary_id_map": "${{ steps.create_issue.outputs.temporary_id_map }}",
	}
}

// RequiresTempIDMap returns true if this handler needs access to the temporary ID map
func (h *CreateIssueHandler) RequiresTempIDMap() bool {
	return false // create_issue generates the map, doesn't consume it
}
