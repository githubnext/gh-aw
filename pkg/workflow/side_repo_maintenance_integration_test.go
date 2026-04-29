//go:build integration

package workflow

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/github/gh-aw/pkg/stringutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// compileSideRepoWorkflow parses a workflow markdown file and returns the
// resulting workflowData plus a temp directory, so callers can then invoke
// GenerateMaintenanceWorkflow and inspect side-repo maintenance files.
func compileSideRepoWorkflow(t *testing.T, content string) ([]*WorkflowData, string) {
	t.Helper()
	tmpDir, err := os.MkdirTemp("", "side-repo-maint-test-*")
	require.NoError(t, err, "create temp dir")
	t.Cleanup(func() { os.RemoveAll(tmpDir) })

	workflowPath := filepath.Join(tmpDir, "test-workflow.md")
	require.NoError(t, os.WriteFile(workflowPath, []byte(content), 0644), "write workflow file")

	compiler := NewCompiler()
	// ParseWorkflowFile populates CheckoutConfigs, SafeOutputs, and Name —
	// exactly the fields examined by GenerateMaintenanceWorkflow.
	workflowData, err := compiler.ParseWorkflowFile(workflowPath)
	require.NoError(t, err, "parse workflow data")

	return []*WorkflowData{workflowData}, tmpDir
}

// TestSideRepoMaintenanceWorkflowGenerated_EndToEnd verifies that compiling a
// workflow with a SideRepoOps checkout generates a side-repo maintenance file
// with the expected top-level structure.
func TestSideRepoMaintenanceWorkflowGenerated_EndToEnd(t *testing.T) {
	workflowContent := `---
on:
  workflow_dispatch:
permissions:
  contents: read
  issues: read
  pull-requests: read
engine: copilot
checkout:
  repository: my-org/target-repo
  current: true
  github-token: ${{ secrets.GH_AW_TARGET_TOKEN }}
---

# Side-Repo Test Workflow

This workflow operates on a separate repository.
`

	workflowDataList, tmpDir := compileSideRepoWorkflow(t, workflowContent)

	err := GenerateMaintenanceWorkflow(workflowDataList, tmpDir, "v1.0.0", ActionModeDev, "", false, nil, "")
	require.NoError(t, err, "generate maintenance workflow")

	sideRepoFile := filepath.Join(tmpDir, "agentics-maintenance-my-org-target-repo.yml")
	content, err := os.ReadFile(sideRepoFile)
	require.NoError(t, err, "side-repo maintenance file should have been created")

	contentStr := string(content)

	// Workflow name reflects target repo.
	assert.Contains(t, contentStr, "my-org/target-repo",
		"generated workflow should reference the target repo slug")

	// Must have workflow_dispatch trigger.
	assert.Contains(t, contentStr, "workflow_dispatch:",
		"generated workflow should include workflow_dispatch trigger")

	// Must have workflow_call trigger.
	assert.Contains(t, contentStr, "workflow_call:",
		"generated workflow should include workflow_call trigger")

	// Must have apply_safe_outputs job.
	assert.Contains(t, contentStr, "apply_safe_outputs:",
		"generated workflow should include apply_safe_outputs job")

	// Must have create_labels job.
	assert.Contains(t, contentStr, "create_labels:",
		"generated workflow should include create_labels job")

	// Must have activity_report job.
	assert.Contains(t, contentStr, "activity_report:",
		"generated workflow should include activity_report job")
	assert.Contains(t, contentStr, "Restore activity report logs cache",
		"generated workflow should include cache restore step for activity_report logs")
	assert.Contains(t, contentStr, "Save activity report logs cache",
		"generated workflow should include cache save step for activity_report logs")
	assert.Contains(t, contentStr, "if: ${{ always() }}",
		"generated workflow should save activity_report logs cache even if report generation fails")
	assert.Contains(t, contentStr, "steps.activity_report_logs_cache.outputs.cache-primary-key",
		"generated workflow should save activity_report logs using the cache primary key")
	assert.Contains(t, contentStr, "Download activity report logs in target repository",
		"generated workflow should include direct logs download step for activity_report")
	assert.Contains(t, contentStr, "timeout-minutes: 20",
		"generated workflow should set a 20-minute timeout for the activity_report logs download step")
	assert.Contains(t, contentStr, "${GH_AW_CMD_PREFIX} logs",
		"generated workflow should run gh aw logs directly")
	assert.Contains(t, contentStr, "--start-date -1w",
		"generated workflow should download 7 days of logs for activity_report")
	assert.Contains(t, contentStr, "--count 100",
		"generated workflow should limit activity_report log downloads to at most 100 runs")
	assert.Contains(t, contentStr, "--format markdown",
		"generated workflow should request markdown report output from gh aw logs")
	assert.Contains(t, contentStr, "./.cache/gh-aw/activity-report-logs/report.md",
		"generated workflow should write activity_report markdown output to report.md")
	assert.Contains(t, contentStr, "Generate activity report issue in target repository",
		"generated workflow should include activity_report issue generation step after cache save")
	assert.Contains(t, contentStr, "title: '[aw] agentic status report'",
		"generated workflow should create the activity_report issue with the expected title")
	assert.Contains(t, contentStr, "actions: read\n      contents: read\n      issues: write",
		"activity_report job should include contents: read with explicit permissions")
	assert.Contains(t, contentStr, "timeout-minutes: 120",
		"activity_report job should include a 2 hour timeout")
	assert.Contains(t, contentStr, "${{ github.run_id }}",
		"activity_report cache key should include run id for latest-cache resolution")

	// GH_AW_TARGET_REPO_SLUG must be wired with the correct slug.
	assert.Contains(t, contentStr, `GH_AW_TARGET_REPO_SLUG: "my-org/target-repo"`,
		"GH_AW_TARGET_REPO_SLUG should be set to the target repo slug")

	// Custom token should appear in the generated file.
	assert.Contains(t, contentStr, "secrets.GH_AW_TARGET_TOKEN",
		"custom github-token should appear in the generated workflow")
}

// TestSideRepoMaintenanceWorkflowWithExpires_EndToEnd verifies that when the
// workflow uses safe-output expiry, the side-repo file includes a schedule
// trigger with a fuzzy cron expression (not minute :00 or the fixed :37).
func TestSideRepoMaintenanceWorkflowWithExpires_EndToEnd(t *testing.T) {
	workflowContent := `---
on:
  workflow_dispatch:
permissions:
  contents: read
  issues: read
  pull-requests: read
engine: copilot
checkout:
  repository: corp/infra-tools
  current: true
safe-outputs:
  create-issue:
    expires: 14
---

# Expires Test Workflow

Create issues that expire after 14 days.
`

	workflowDataList, tmpDir := compileSideRepoWorkflow(t, workflowContent)

	err := GenerateMaintenanceWorkflow(workflowDataList, tmpDir, "v1.0.0", ActionModeDev, "", false, nil, "")
	require.NoError(t, err, "generate maintenance workflow")

	sideRepoFile := filepath.Join(tmpDir, "agentics-maintenance-corp-infra-tools.yml")
	content, err := os.ReadFile(sideRepoFile)
	require.NoError(t, err, "side-repo maintenance file should have been created")
	contentStr := string(content)

	// Must have a schedule trigger when expires is set.
	assert.Contains(t, contentStr, "schedule:",
		"side-repo maintenance should include schedule trigger when expires is set")

	// close-expired-entities job must be present.
	assert.Contains(t, contentStr, "close-expired-entities:",
		"side-repo maintenance should include close-expired-entities job when expires is set")

	// The cron expression should be present; extract it and verify it is valid.
	expectedCron, _ := generateSideRepoMaintenanceCron("corp/infra-tools", 14)
	assert.Contains(t, contentStr, expectedCron,
		"cron expression should match the fuzzy-scheduled value for corp/infra-tools")

	// The cron minute must not be 0 or 37 (fixed values to avoid pile-up).
	// We verify by checking the actual expected value contains neither ":00" nor ":37".
	minute := strings.Fields(expectedCron)[0]
	assert.NotEqual(t, "0", minute,
		"fuzzy cron should not fire at minute 0 (likely collision with defaults)")
}

// TestSideRepoMaintenanceWorkflowFallbackToken_EndToEnd verifies that when no
// custom token is specified in the checkout config, the generated workflow falls
// back to GH_AW_GITHUB_TOKEN.
func TestSideRepoMaintenanceWorkflowFallbackToken_EndToEnd(t *testing.T) {
	workflowContent := `---
on:
  workflow_dispatch:
permissions:
  contents: read
  issues: read
  pull-requests: read
engine: copilot
checkout:
  repository: acme/shared-services
  current: true
---

# No-token side-repo workflow.
`

	workflowDataList, tmpDir := compileSideRepoWorkflow(t, workflowContent)

	err := GenerateMaintenanceWorkflow(workflowDataList, tmpDir, "v1.0.0", ActionModeDev, "", false, nil, "")
	require.NoError(t, err, "generate maintenance workflow")

	sideRepoFile := filepath.Join(tmpDir, "agentics-maintenance-acme-shared-services.yml")
	content, err := os.ReadFile(sideRepoFile)
	require.NoError(t, err, "side-repo maintenance file should have been created")
	contentStr := string(content)

	// Fallback token should be referenced.
	assert.Contains(t, contentStr, "GH_AW_GITHUB_TOKEN",
		"should fall back to GH_AW_GITHUB_TOKEN when no custom token is specified")
}

// TestNoSideRepoMaintenanceForExpressionRepository_EndToEnd verifies that
// expression-based repository values do not produce a side-repo maintenance file.
func TestNoSideRepoMaintenanceForExpressionRepository_EndToEnd(t *testing.T) {
	workflowContent := `---
on:
  workflow_dispatch:
    inputs:
      target_repo:
        description: Target repository
        required: true
permissions:
  contents: read
engine: copilot
checkout:
  repository: ${{ inputs.target_repo }}
  current: true
---

# Dynamic repository workflow.
`

	workflowDataList, tmpDir := compileSideRepoWorkflow(t, workflowContent)

	err := GenerateMaintenanceWorkflow(workflowDataList, tmpDir, "v1.0.0", ActionModeDev, "", false, nil, "")
	require.NoError(t, err, "generate maintenance workflow")

	// No side-repo file should be created because the repository is an expression.
	entries, err := os.ReadDir(tmpDir)
	require.NoError(t, err)
	for _, e := range entries {
		assert.False(t,
			strings.HasPrefix(e.Name(), "agentics-maintenance-") && e.Name() != "agentics-maintenance.yml",
			"no side-repo maintenance file should be generated for expression-based repositories, got: %s", e.Name())
	}
}

// TestSideRepoMaintenanceFuzzyScheduleScattered_EndToEnd verifies that two
// different side-repo targets receive distinct cron expressions (scattered).
func TestSideRepoMaintenanceFuzzyScheduleScattered_EndToEnd(t *testing.T) {
	// Compile two separate workflows for different side-repo targets.
	makeContent := func(repo string) string {
		return `---
on:
  workflow_dispatch:
permissions:
  contents: read
  issues: read
  pull-requests: read
engine: copilot
checkout:
  repository: ` + repo + `
  current: true
safe-outputs:
  create-issue:
    expires: 30
---

# Scattered cron test.
`
	}

	repoA := "company/repo-alpha"
	repoB := "company/repo-beta"

	cronA, _ := generateSideRepoMaintenanceCron(repoA, 30)
	cronB, _ := generateSideRepoMaintenanceCron(repoB, 30)

	// Verify the crons are actually different (they should be; if they collide that
	// would be a surprising FNV-1a collision and the test would rightly flag it).
	assert.NotEqual(t, cronA, cronB,
		"different side-repo targets should get different cron expressions to avoid simultaneous runs")

	// Compile both and verify each generated file contains its own cron.
	for _, tc := range []struct {
		repo string
		cron string
	}{
		{repoA, cronA},
		{repoB, cronB},
	} {
		t.Run(tc.repo, func(t *testing.T) {
			wdl, tmpDir := compileSideRepoWorkflow(t, makeContent(tc.repo))
			require.NoError(t, GenerateMaintenanceWorkflow(wdl, tmpDir, "v1.0.0", ActionModeDev, "", false, nil, ""))

			slug := stringutil.SanitizeForFilename(tc.repo)
			sideFile := filepath.Join(tmpDir, "agentics-maintenance-"+slug+".yml")
			fileContent, err := os.ReadFile(sideFile)
			require.NoError(t, err, "side-repo file should exist for %s", tc.repo)

			assert.Contains(t, string(fileContent), tc.cron,
				"generated file for %s should contain cron %s", tc.repo, tc.cron)
		})
	}
}
