package workflow

import (
	"strings"
	"testing"
)

func TestBuildCopilotParticipantSteps_EmptyParticipants(t *testing.T) {
	config := CopilotParticipantConfig{
		Participants:       []string{},
		ParticipantType:    "assignee",
		CustomToken:        "",
		SafeOutputsToken:   "",
		WorkflowToken:      "",
		ConditionStepID:    "create_issue",
		ConditionOutputKey: "issue_number",
	}

	steps := buildCopilotParticipantSteps(config)

	if steps != nil {
		t.Errorf("Expected nil steps for empty participants, got %d steps", len(steps))
	}
}

func TestBuildCopilotParticipantSteps_IssueAssignee(t *testing.T) {
	config := CopilotParticipantConfig{
		Participants:       []string{"user1", "user2"},
		ParticipantType:    "assignee",
		CustomToken:        "",
		SafeOutputsToken:   "",
		WorkflowToken:      "",
		ConditionStepID:    "create_issue",
		ConditionOutputKey: "issue_number",
	}

	steps := buildCopilotParticipantSteps(config)
	stepsContent := strings.Join(steps, "")

	// Check that checkout step is included
	if !strings.Contains(stepsContent, "Checkout repository for gh CLI") {
		t.Error("Expected checkout step for gh CLI")
	}

	// Check that assignee steps are included
	if !strings.Contains(stepsContent, "Assign issue to user1") {
		t.Error("Expected assignee step for user1")
	}
	if !strings.Contains(stepsContent, "Assign issue to user2") {
		t.Error("Expected assignee step for user2")
	}

	// Check that actions/github-script is used for assigning
	if !strings.Contains(stepsContent, "actions/github-script") {
		t.Error("Expected actions/github-script to be used for assignee steps")
	}

	// Check that the condition references the correct step
	if !strings.Contains(stepsContent, "if: steps.create_issue.outputs.issue_number != ''") {
		t.Error("Expected conditional on create_issue.outputs.issue_number")
	}

	// Check that default token is used
	if !strings.Contains(stepsContent, "GH_TOKEN: ${{ secrets.GH_AW_GITHUB_TOKEN || secrets.GITHUB_TOKEN }}") {
		t.Error("Expected default GitHub token")
	}
}

func TestBuildCopilotParticipantSteps_CopilotAssignee(t *testing.T) {
	config := CopilotParticipantConfig{
		Participants:       []string{"copilot"},
		ParticipantType:    "assignee",
		CustomToken:        "",
		SafeOutputsToken:   "",
		WorkflowToken:      "",
		ConditionStepID:    "create_issue",
		ConditionOutputKey: "issue_number",
	}

	steps := buildCopilotParticipantSteps(config)
	stepsContent := strings.Join(steps, "")

	// Check that "copilot" is mapped to "@copilot"
	if !strings.Contains(stepsContent, `ASSIGNEE: "@copilot"`) {
		t.Error("Expected ASSIGNEE environment variable to be set to '@copilot'")
	}

	// Check that Copilot token precedence is used
	if !strings.Contains(stepsContent, "GH_TOKEN: ${{ secrets.GH_AW_COPILOT_TOKEN || secrets.GH_AW_GITHUB_TOKEN }}") {
		t.Error("Expected Copilot token precedence")
	}

	// Verify GITHUB_TOKEN is NOT in the fallback chain for copilot assignees
	if strings.Contains(stepsContent, "|| secrets.GITHUB_TOKEN }}") {
		t.Error("Did not expect GITHUB_TOKEN in fallback chain for copilot assignees")
	}
}

func TestBuildCopilotParticipantSteps_PRReviewer(t *testing.T) {
	config := CopilotParticipantConfig{
		Participants:       []string{"user1"},
		ParticipantType:    "reviewer",
		CustomToken:        "",
		SafeOutputsToken:   "",
		WorkflowToken:      "",
		ConditionStepID:    "create_pull_request",
		ConditionOutputKey: "pull_request_url",
	}

	steps := buildCopilotParticipantSteps(config)
	stepsContent := strings.Join(steps, "")

	// Check that reviewer step is included
	if !strings.Contains(stepsContent, "Add user1 as reviewer") {
		t.Error("Expected reviewer step for user1")
	}

	// Check that gh pr edit command is present
	if !strings.Contains(stepsContent, "gh pr edit") {
		t.Error("Expected gh pr edit command in steps")
	}

	// Check that the condition references the correct step
	if !strings.Contains(stepsContent, "if: steps.create_pull_request.outputs.pull_request_url != ''") {
		t.Error("Expected conditional on create_pull_request.outputs.pull_request_url")
	}

	// Verify environment variable for reviewer
	if !strings.Contains(stepsContent, `REVIEWER: "user1"`) {
		t.Error("Expected REVIEWER environment variable to be set")
	}
}

func TestBuildCopilotParticipantSteps_CopilotReviewer(t *testing.T) {
	config := CopilotParticipantConfig{
		Participants:       []string{"copilot"},
		ParticipantType:    "reviewer",
		CustomToken:        "",
		SafeOutputsToken:   "",
		WorkflowToken:      "",
		ConditionStepID:    "create_pull_request",
		ConditionOutputKey: "pull_request_url",
	}

	steps := buildCopilotParticipantSteps(config)
	stepsContent := strings.Join(steps, "")

	// Check that it uses the GitHub API (not gh pr edit)
	if !strings.Contains(stepsContent, "gh api --method POST") {
		t.Error("Expected GitHub API call for copilot reviewer")
	}

	// Check that it uses the correct bot name
	if !strings.Contains(stepsContent, "copilot-pull-request-reviewer[bot]") {
		t.Error("Expected copilot-pull-request-reviewer[bot] as the reviewer")
	}

	// Verify that gh pr edit is NOT used for copilot
	if strings.Contains(stepsContent, "gh pr edit") {
		t.Error("Should not use gh pr edit for copilot reviewer")
	}

	// Check that Copilot token precedence is used
	if !strings.Contains(stepsContent, "GH_TOKEN: ${{ secrets.GH_AW_COPILOT_TOKEN || secrets.GH_AW_GITHUB_TOKEN }}") {
		t.Error("Expected Copilot token precedence")
	}
}

func TestBuildCopilotParticipantSteps_CustomToken(t *testing.T) {
	config := CopilotParticipantConfig{
		Participants:       []string{"user1"},
		ParticipantType:    "assignee",
		CustomToken:        "${{ secrets.CUSTOM_PAT }}",
		SafeOutputsToken:   "",
		WorkflowToken:      "",
		ConditionStepID:    "create_issue",
		ConditionOutputKey: "issue_number",
	}

	steps := buildCopilotParticipantSteps(config)
	stepsContent := strings.Join(steps, "")

	// Check that the custom token is used (highest precedence)
	if !strings.Contains(stepsContent, "GH_TOKEN: ${{ secrets.CUSTOM_PAT }}") {
		t.Error("Expected custom GitHub token to be used")
	}

	// Verify default token is NOT used
	if strings.Contains(stepsContent, "GH_TOKEN: ${{ secrets.GH_AW_GITHUB_TOKEN") {
		t.Error("Did not expect default token when custom token is configured")
	}
}

func TestBuildCopilotParticipantSteps_MixedParticipants(t *testing.T) {
	config := CopilotParticipantConfig{
		Participants:       []string{"user1", "copilot", "user2"},
		ParticipantType:    "assignee",
		CustomToken:        "",
		SafeOutputsToken:   "",
		WorkflowToken:      "",
		ConditionStepID:    "create_issue",
		ConditionOutputKey: "issue_number",
	}

	steps := buildCopilotParticipantSteps(config)
	stepsContent := strings.Join(steps, "")

	// Check that all participants are included
	if !strings.Contains(stepsContent, "Assign issue to user1") {
		t.Error("Expected assignee step for user1")
	}
	if !strings.Contains(stepsContent, "Assign issue to copilot") {
		t.Error("Expected assignee step for copilot")
	}
	if !strings.Contains(stepsContent, "Assign issue to user2") {
		t.Error("Expected assignee step for user2")
	}

	// When copilot is in the list, all steps should use Copilot token
	if !strings.Contains(stepsContent, "GH_TOKEN: ${{ secrets.GH_AW_COPILOT_TOKEN || secrets.GH_AW_GITHUB_TOKEN }}") {
		t.Error("Expected Copilot token precedence when copilot is in the list")
	}

	// Verify GITHUB_TOKEN is NOT in the fallback chain
	if strings.Contains(stepsContent, "|| secrets.GITHUB_TOKEN }}") {
		t.Error("Did not expect GITHUB_TOKEN in fallback chain when copilot is in the list")
	}
}
