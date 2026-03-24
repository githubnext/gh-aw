//go:build !integration

package workflow

import (
	"encoding/json"
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// allHandlersSafeOutputsConfig returns a SafeOutputsConfig with every handler
// set to a minimal non-nil value, used to validate dual-path config parity.
func allHandlersSafeOutputsConfig() *SafeOutputsConfig {
	maxOnePtr := strPtr("1")
	base := BaseSafeOutputConfig{Max: maxOnePtr}
	updateEntity := UpdateEntityConfig{BaseSafeOutputConfig: base}
	return &SafeOutputsConfig{
		CreateIssues:                    &CreateIssuesConfig{BaseSafeOutputConfig: base},
		AddComments:                     &AddCommentsConfig{BaseSafeOutputConfig: base, Target: "issue"},
		CreateDiscussions:               &CreateDiscussionsConfig{BaseSafeOutputConfig: base},
		UpdateDiscussions:               &UpdateDiscussionsConfig{UpdateEntityConfig: updateEntity},
		CloseDiscussions:                &CloseDiscussionsConfig{BaseSafeOutputConfig: base},
		CloseIssues:                     &CloseIssuesConfig{BaseSafeOutputConfig: base},
		ClosePullRequests:               &ClosePullRequestsConfig{BaseSafeOutputConfig: base},
		MarkPullRequestAsReadyForReview: &MarkPullRequestAsReadyForReviewConfig{BaseSafeOutputConfig: base},
		CreatePullRequests:              &CreatePullRequestsConfig{BaseSafeOutputConfig: base},
		CreatePullRequestReviewComments: &CreatePullRequestReviewCommentsConfig{BaseSafeOutputConfig: base},
		SubmitPullRequestReview:         &SubmitPullRequestReviewConfig{BaseSafeOutputConfig: base},
		ReplyToPullRequestReviewComment: &ReplyToPullRequestReviewCommentConfig{BaseSafeOutputConfig: base},
		ResolvePullRequestReviewThread:  &ResolvePullRequestReviewThreadConfig{BaseSafeOutputConfig: base},
		CreateCodeScanningAlerts:        &CreateCodeScanningAlertsConfig{BaseSafeOutputConfig: base},
		AutofixCodeScanningAlert:        &AutofixCodeScanningAlertConfig{BaseSafeOutputConfig: base},
		AddLabels:                       &AddLabelsConfig{BaseSafeOutputConfig: base},
		RemoveLabels:                    &RemoveLabelsConfig{BaseSafeOutputConfig: base},
		AddReviewer:                     &AddReviewerConfig{BaseSafeOutputConfig: base},
		AssignMilestone:                 &AssignMilestoneConfig{BaseSafeOutputConfig: base},
		AssignToAgent:                   &AssignToAgentConfig{BaseSafeOutputConfig: base},
		AssignToUser:                    &AssignToUserConfig{BaseSafeOutputConfig: base},
		UnassignFromUser:                &UnassignFromUserConfig{BaseSafeOutputConfig: base},
		UpdateIssues:                    &UpdateIssuesConfig{UpdateEntityConfig: updateEntity},
		UpdatePullRequests:              &UpdatePullRequestsConfig{UpdateEntityConfig: updateEntity},
		PushToPullRequestBranch:         &PushToPullRequestBranchConfig{BaseSafeOutputConfig: base},
		UploadAssets:                    &UploadAssetsConfig{BaseSafeOutputConfig: base},
		UpdateRelease:                   &UpdateReleaseConfig{UpdateEntityConfig: updateEntity},
		CreateAgentSessions:             &CreateAgentSessionConfig{BaseSafeOutputConfig: base},
		UpdateProjects:                  &UpdateProjectConfig{BaseSafeOutputConfig: base},
		CreateProjects:                  &CreateProjectsConfig{BaseSafeOutputConfig: base},
		CreateProjectStatusUpdates:      &CreateProjectStatusUpdateConfig{BaseSafeOutputConfig: base},
		LinkSubIssue:                    &LinkSubIssueConfig{BaseSafeOutputConfig: base},
		HideComment:                     &HideCommentConfig{BaseSafeOutputConfig: base},
		SetIssueType:                    &SetIssueTypeConfig{BaseSafeOutputConfig: base},
		DispatchWorkflow:                &DispatchWorkflowConfig{BaseSafeOutputConfig: base},
		DispatchRepository: &DispatchRepositoryConfig{
			// dispatch_repository requires at least one tool to produce a non-nil config
			Tools: map[string]*DispatchRepositoryToolConfig{
				"test_tool": {EventType: "ci"},
			},
		},
		CallWorkflow: &CallWorkflowConfig{BaseSafeOutputConfig: base},
		MissingTool:  &MissingToolConfig{BaseSafeOutputConfig: base},
		MissingData:  &MissingDataConfig{BaseSafeOutputConfig: base},
		NoOp:         &NoOpConfig{BaseSafeOutputConfig: base},
	}
}

// TestSafeOutputsConfigPathParity validates that every handler key produced by the
// handlerRegistry (new path: GH_AW_SAFE_OUTPUTS_HANDLER_CONFIG) is also produced by
// generateSafeOutputsConfig (old path: GH_AW_SAFE_OUTPUTS_CONFIG_PATH / config.json)
// when the corresponding handler is enabled in SafeOutputsConfig.
//
// This test enforces the sync guarantee described in the dual-config cross-reference
// comments in safe_outputs_config_generation.go and compiler_safe_outputs_config.go:
// adding a new handler to one path must also be added to the other.
func TestSafeOutputsConfigPathParity(t *testing.T) {
	cfg := allHandlersSafeOutputsConfig()

	// --- New path: collect handler keys from handlerRegistry ---
	registryKeys := make([]string, 0, len(handlerRegistry))
	for handlerName, builder := range handlerRegistry {
		handlerCfg := builder(cfg)
		if handlerCfg != nil {
			registryKeys = append(registryKeys, handlerName)
		}
	}
	sort.Strings(registryKeys)

	// --- Old path: collect top-level keys from generateSafeOutputsConfig ---
	data := &WorkflowData{SafeOutputs: cfg}
	jsonStr := generateSafeOutputsConfig(data)
	require.NotEmpty(t, jsonStr, "generateSafeOutputsConfig should return non-empty JSON")

	var oldPathMap map[string]any
	require.NoError(t, json.Unmarshal([]byte(jsonStr), &oldPathMap),
		"generateSafeOutputsConfig output must be valid JSON")

	oldPathKeys := make([]string, 0, len(oldPathMap))
	for k := range oldPathMap {
		oldPathKeys = append(oldPathKeys, k)
	}
	sort.Strings(oldPathKeys)

	// The old path may include extra auxiliary keys for missing_tool/missing_data issue
	// creation — these are sub-handlers that exist only in the old path (config.json).
	// Exclude them from the comparison since they have no equivalent registry entry.
	oldPathExtraKeys := map[string]bool{
		"create_missing_tool_issue": true,
		"create_missing_data_issue": true,
	}
	filteredOldPathKeys := make([]string, 0, len(oldPathKeys))
	for _, k := range oldPathKeys {
		if !oldPathExtraKeys[k] {
			filteredOldPathKeys = append(filteredOldPathKeys, k)
		}
	}

	assert.Equal(t, registryKeys, filteredOldPathKeys,
		"Both config generation paths must produce the same set of handler keys.\n"+
			"If a new handler was added to handlerRegistry (compiler_safe_outputs_config.go),\n"+
			"it must also be added to generateSafeOutputsConfig (safe_outputs_config_generation.go),\n"+
			"and vice versa.")
}
