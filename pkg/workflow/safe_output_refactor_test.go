package workflow

import (
	"strings"
	"testing"
)

// TestCreatePRReviewCommentUsesHelper verifies that create_pr_review_comment.go
// uses the buildSafeOutputJobEnvVars helper correctly
func TestCreatePRReviewCommentUsesHelper(t *testing.T) {
	c := NewCompiler(false, "", "test")

	workflowData := &WorkflowData{
		Name: "test-workflow",
		SafeOutputs: &SafeOutputsConfig{
			Staged: true,
			CreatePullRequestReviewComments: &CreatePullRequestReviewCommentsConfig{
				BaseSafeOutputConfig: BaseSafeOutputConfig{Max: 10},
				TargetRepoSlug:       "owner/target-repo",
			},
		},
	}

	job, err := c.buildCreateOutputPullRequestReviewCommentJob(workflowData, "main_job")
	if err != nil {
		t.Fatalf("Unexpected error building PR review comment job: %v", err)
	}

	// Convert steps to a single string for testing
	stepsContent := strings.Join(job.Steps, "")

	// Verify that GH_AW_SAFE_OUTPUTS_STAGED is present
	if !strings.Contains(stepsContent, "          GH_AW_SAFE_OUTPUTS_STAGED: \"true\"\n") {
		t.Error("Expected GH_AW_SAFE_OUTPUTS_STAGED to be set in create-pull-request-review-comment job")
	}

	// Verify that GH_AW_TARGET_REPO_SLUG is present with the correct value
	if !strings.Contains(stepsContent, "          GH_AW_TARGET_REPO_SLUG: \"owner/target-repo\"\n") {
		t.Error("Expected GH_AW_TARGET_REPO_SLUG to be set correctly in create-pull-request-review-comment job")
	}
}

// TestCreateDiscussionUsesHelper verifies that create_discussion.go
// uses the buildSafeOutputJobEnvVars helper correctly
func TestCreateDiscussionUsesHelper(t *testing.T) {
	c := NewCompiler(false, "", "test")

	workflowData := &WorkflowData{
		Name: "test-workflow",
		SafeOutputs: &SafeOutputsConfig{
			Staged: true,
			CreateDiscussions: &CreateDiscussionsConfig{
				BaseSafeOutputConfig: BaseSafeOutputConfig{Max: 1},
				Category:             "12345",
				TargetRepoSlug:       "owner/target-repo",
			},
		},
	}

	job, err := c.buildCreateOutputDiscussionJob(workflowData, "main_job")
	if err != nil {
		t.Fatalf("Unexpected error building discussion job: %v", err)
	}

	// Convert steps to a single string for testing
	stepsContent := strings.Join(job.Steps, "")

	// Verify that GH_AW_SAFE_OUTPUTS_STAGED is present
	if !strings.Contains(stepsContent, "          GH_AW_SAFE_OUTPUTS_STAGED: \"true\"\n") {
		t.Error("Expected GH_AW_SAFE_OUTPUTS_STAGED to be set in create-discussion job")
	}

	// Verify that GH_AW_TARGET_REPO_SLUG is present with the correct value
	if !strings.Contains(stepsContent, "          GH_AW_TARGET_REPO_SLUG: \"owner/target-repo\"\n") {
		t.Error("Expected GH_AW_TARGET_REPO_SLUG to be set correctly in create-discussion job")
	}
}

// TestTrialModeWithoutTargetRepo verifies that trial mode without explicit
// target-repo config uses the trial repo slug
func TestTrialModeWithoutTargetRepo(t *testing.T) {
	c := NewCompiler(false, "", "test")
	c.SetTrialMode(true)
	c.SetTrialLogicalRepoSlug("owner/trial-repo")

	workflowData := &WorkflowData{
		Name:             "test-workflow",
		TrialMode:        true,
		TrialLogicalRepo: "owner/trial-repo",
		SafeOutputs: &SafeOutputsConfig{
			CreateDiscussions: &CreateDiscussionsConfig{
				BaseSafeOutputConfig: BaseSafeOutputConfig{Max: 1},
				Category:             "12345",
			},
		},
	}

	job, err := c.buildCreateOutputDiscussionJob(workflowData, "main_job")
	if err != nil {
		t.Fatalf("Unexpected error building discussion job: %v", err)
	}

	// Convert steps to a single string for testing
	stepsContent := strings.Join(job.Steps, "")

	// Verify that GH_AW_SAFE_OUTPUTS_STAGED is present (trial mode sets this)
	if !strings.Contains(stepsContent, "          GH_AW_SAFE_OUTPUTS_STAGED: \"true\"\n") {
		t.Error("Expected GH_AW_SAFE_OUTPUTS_STAGED to be set in trial mode")
	}

	// Verify that GH_AW_TARGET_REPO_SLUG uses trial repo slug
	if !strings.Contains(stepsContent, "          GH_AW_TARGET_REPO_SLUG: \"owner/trial-repo\"\n") {
		t.Error("Expected GH_AW_TARGET_REPO_SLUG to use trial repo slug in trial mode")
	}
}

// TestNoStagedNorTrialMode verifies that neither staged flag nor target repo slug
// are added when not configured
func TestNoStagedNorTrialMode(t *testing.T) {
	c := NewCompiler(false, "", "test")

	workflowData := &WorkflowData{
		Name: "test-workflow",
		SafeOutputs: &SafeOutputsConfig{
			CreatePullRequestReviewComments: &CreatePullRequestReviewCommentsConfig{
				BaseSafeOutputConfig: BaseSafeOutputConfig{Max: 10},
			},
		},
	}

	job, err := c.buildCreateOutputPullRequestReviewCommentJob(workflowData, "main_job")
	if err != nil {
		t.Fatalf("Unexpected error building PR review comment job: %v", err)
	}

	// Convert steps to a single string for testing
	stepsContent := strings.Join(job.Steps, "")

	// Verify that GH_AW_SAFE_OUTPUTS_STAGED is NOT present
	if strings.Contains(stepsContent, "GH_AW_SAFE_OUTPUTS_STAGED:") {
		t.Error("Expected GH_AW_SAFE_OUTPUTS_STAGED to not be set when staged is false")
	}

	// Verify that GH_AW_TARGET_REPO_SLUG is NOT present
	if strings.Contains(stepsContent, "GH_AW_TARGET_REPO_SLUG:") {
		t.Error("Expected GH_AW_TARGET_REPO_SLUG to not be set when not configured")
	}
}

// TestTargetRepoOverridesTrialRepo verifies that explicit target-repo config
// takes precedence over trial mode repo slug
func TestTargetRepoOverridesTrialRepo(t *testing.T) {
	c := NewCompiler(false, "", "test")
	c.SetTrialMode(true)
	c.SetTrialLogicalRepoSlug("owner/trial-repo")

	workflowData := &WorkflowData{
		Name: "test-workflow",
		SafeOutputs: &SafeOutputsConfig{
			CreatePullRequestReviewComments: &CreatePullRequestReviewCommentsConfig{
				BaseSafeOutputConfig: BaseSafeOutputConfig{Max: 10},
				TargetRepoSlug:       "owner/explicit-target",
			},
		},
	}

	job, err := c.buildCreateOutputPullRequestReviewCommentJob(workflowData, "main_job")
	if err != nil {
		t.Fatalf("Unexpected error building PR review comment job: %v", err)
	}

	// Convert steps to a single string for testing
	stepsContent := strings.Join(job.Steps, "")

	// Verify that GH_AW_TARGET_REPO_SLUG uses explicit target, not trial repo
	if !strings.Contains(stepsContent, "          GH_AW_TARGET_REPO_SLUG: \"owner/explicit-target\"\n") {
		t.Error("Expected GH_AW_TARGET_REPO_SLUG to use explicit target-repo, not trial repo")
	}

	// Verify that trial repo slug is NOT used
	if strings.Contains(stepsContent, "          GH_AW_TARGET_REPO_SLUG: \"owner/trial-repo\"\n") {
		t.Error("Expected trial repo slug to be overridden by explicit target-repo")
	}
}
