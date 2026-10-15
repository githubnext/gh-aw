//go:build !integration

package workflow

import (
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/goccy/go-yaml"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDedicatedUploaderLocalWorkflowMetadata(t *testing.T) {
	for _, tc := range []struct {
		name           string
		on             string
		inlinedImports bool
		workflowCall   bool
	}{
		{name: "workflow_call", on: "on:\n  workflow_call:\n", workflowCall: true},
		{name: "mixed triggers", on: "on:\n  workflow_call:\n  workflow_dispatch:\n", workflowCall: true},
		{name: "inlined workflow_call", on: "on:\n  workflow_call:\n", inlinedImports: true, workflowCall: true},
		{name: "ordinary workflow", on: "on:\n  workflow_dispatch:\n"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			app := &GitHubAppConfig{
				AppID:      "${{ vars.APP_ID }}",
				PrivateKey: "${{ secrets.APP_PRIVATE_KEY }}",
			}
			data := &WorkflowData{
				On:             tc.on,
				InlinedImports: tc.inlinedImports,
				CheckoutConfigs: []*CheckoutConfig{
					{GitHubApp: app},
				},
				SafeOutputs: &SafeOutputsConfig{
					CreateCodeScanningAlerts: &CreateCodeScanningAlertsConfig{},
					UploadCodeCoverage: &UploadCodeCoverageConfig{
						BaseSafeOutputConfig: BaseSafeOutputConfig{GitHubApp: app},
					},
				},
			}
			c := NewCompiler()
			sarif, err := c.buildCodeScanningUploadJob(data)
			require.NoError(t, err)
			coverage, err := c.buildUploadCodeCoverageJob(data, "agent")
			require.NoError(t, err)

			for _, uploader := range []struct {
				name string
				job  *Job
				id   string
			}{
				{name: "SARIF", job: sarif, id: "checkout-restore-app-token"},
				{name: "coverage", job: coverage, id: "upload-code-coverage-app-token"},
			} {
				t.Run(uploader.name, func(t *testing.T) {
					steps := strings.Join(uploader.job.Steps, "")
					assert.NotContains(t, steps, "needs.activation.outputs.target_")
					if !tc.workflowCall {
						assert.NotContains(t, steps, "id: resolve-host-repo")
						assert.Contains(t, steps, "repositories: ${{ github.event.repository.name }}")
						assert.Contains(t, steps, "owner: ${{ github.repository_owner }}")
						return
					}

					assert.Equal(t, 1, strings.Count(steps, "id: resolve-host-repo\n"))
					assert.Contains(t, steps, "repositories: "+activationTargetRepoNameExpr)
					assert.Contains(t, steps, "GH_AW_TARGET_REPOSITORY: "+activationTargetRepoExpr)
					assert.Contains(t, steps, "owner: ${{ steps."+uploader.id+"-owner.outputs.owner }}")
					resolvePos := strings.Index(steps, "id: resolve-host-repo\n")
					mintPos := strings.Index(steps, "id: "+uploader.id+"\n")
					require.Greater(t, mintPos, resolvePos, "resolve metadata before minting the token")
					if tc.inlinedImports {
						assert.NotContains(t, steps, "Refusing to upload to the caller repository.")
						assert.Contains(t, steps, "JOB_WORKFLOW_REPOSITORY: ${{ github.repository }}")
						assert.Contains(t, steps, "JOB_WORKFLOW_SHA: ${{ github.sha }}")
						assert.NotContains(t, steps, "JOB_WORKFLOW_REPOSITORY: ${{ job.workflow_repository }}")
					} else {
						assert.Contains(t, steps, "Refusing to upload to the caller repository.")
						assert.Contains(t, steps, `if [[ "$TARGET_REPOSITORY" != "$CURRENT_REPOSITORY" ]]; then`)
						assert.Contains(t, steps, "JOB_WORKFLOW_REPOSITORY: ${{ job.workflow_repository }}")
						assert.Contains(t, steps, "JOB_WORKFLOW_SHA: ${{ job.workflow_sha }}")
						assert.Contains(t, steps, "JOB_WORKFLOW_REF: ${{ job.workflow_ref }}")
					}
				})
			}

			sarifSteps := strings.Join(sarif.Steps, "")
			assert.Contains(t, sarifSteps, "persist-credentials: false")
			assert.Contains(t, sarifSteps, "fetch-depth: 1")
			assert.Contains(t, sarifSteps, "token: ${{ steps.checkout-restore-app-token.outputs.token }}")
			if tc.workflowCall && !tc.inlinedImports {
				assert.Contains(t, sarifSteps, "repository: "+activationTargetRepoExpr)
				assert.Contains(t, sarifSteps, "ref: ${{ steps.resolve-host-repo.outputs.target_ref }}")
				assert.Contains(t, sarifSteps, "ref: ${{ steps.resolve-host-repo.outputs.target_checkout_ref }}")
				assert.Contains(t, sarifSteps, "sha: ${{ steps.resolve-host-repo.outputs.target_checkout_ref }}")
				assert.NotContains(t, sarifSteps, "ref: ${{ github.ref }}")
				assert.NotContains(t, sarifSteps, "ref: ${{ github.sha }}")
				assert.NotContains(t, sarifSteps, "sha: ${{ github.sha }}")
				assert.Contains(t, sarifSteps, `if [[ -z "$TARGET_REF" || -z "$TARGET_SHA" ]]; then`)
			} else {
				assert.Contains(t, sarifSteps, "ref: ${{ github.sha }}")
				assert.Contains(t, sarifSteps, "ref: ${{ github.ref }}")
				assert.Contains(t, sarifSteps, "sha: ${{ github.sha }}")
				assert.NotContains(t, sarifSteps, "repository: "+activationTargetRepoExpr)
			}
		})
	}
}

func TestDedicatedUploaderExplicitCredentials(t *testing.T) {
	for _, repositories := range [][]string{{"chosen-repo"}, {"*"}} {
		t.Run(strings.Join(repositories, ","), func(t *testing.T) {
			app := &GitHubAppConfig{
				AppID:        "${{ vars.APP_ID }}",
				PrivateKey:   "${{ secrets.APP_PRIVATE_KEY }}",
				Owner:        "chosen-owner",
				Repositories: repositories,
			}

			data := &WorkflowData{
				On:              "on:\n  workflow_call:\n",
				CheckoutConfigs: []*CheckoutConfig{{GitHubApp: app}},
				SafeOutputs: &SafeOutputsConfig{
					GitHubApp:                app,
					CreateCodeScanningAlerts: &CreateCodeScanningAlertsConfig{},
					UploadCodeCoverage:       &UploadCodeCoverageConfig{},
				},
			}
			c := NewCompiler()
			sarif, err := c.buildCodeScanningUploadJob(data)
			require.NoError(t, err)
			coverage, err := c.buildUploadCodeCoverageJob(data, "agent")
			require.NoError(t, err)
			for _, job := range []*Job{sarif, coverage} {
				steps := strings.Join(job.Steps, "")
				assert.Contains(t, steps, "owner: chosen-owner")
				assert.NotContains(t, steps, "repositories: "+activationTargetRepoNameExpr)
				if repositories[0] == "*" {
					assert.NotContains(t, steps, "          repositories:")
				} else {
					assert.Contains(t, steps, "repositories: chosen-repo")
				}
			}

			data.SafeOutputs.GitHubToken = "${{ secrets.UPLOAD_TOKEN }}"
			sarif, err = c.buildCodeScanningUploadJob(data)
			require.NoError(t, err)
			coverage, err = c.buildUploadCodeCoverageJob(data, "agent")
			require.NoError(t, err)
			assert.Contains(t, strings.Join(sarif.Steps, ""), "token: ${{ secrets.UPLOAD_TOKEN }}")
			coverageSteps := strings.Join(coverage.Steps, "")
			assert.Contains(t, coverageSteps, "token: ${{ secrets.UPLOAD_TOKEN }}")
			assert.NotContains(t, coverageSteps, "id: upload-code-coverage-app-token\n")
		})
	}
}

func TestDedicatedUploaderRepositoryGuards(t *testing.T) {
	data := &WorkflowData{
		On: "on:\n  workflow_call:\n",
		SafeOutputs: &SafeOutputsConfig{
			CreateCodeScanningAlerts: &CreateCodeScanningAlertsConfig{},
			UploadCodeCoverage:       &UploadCodeCoverageConfig{},
		},
	}
	c := NewCompiler()
	sarif, err := c.buildCodeScanningUploadJob(data)
	require.NoError(t, err)
	coverage, err := c.buildUploadCodeCoverageJob(data, "agent")
	require.NoError(t, err)
	for _, uploader := range []struct {
		name string
		job  *Job
	}{
		{name: "SARIF", job: sarif},
		{name: "coverage", job: coverage},
	} {
		t.Run(uploader.name, func(t *testing.T) {
			var document struct {
				Steps []struct {
					Name string `yaml:"name"`
					Run  string `yaml:"run"`
				} `yaml:"steps"`
			}
			require.NoError(t, yaml.Unmarshal([]byte("steps:\n"+strings.Join(uploader.job.Steps, "")), &document))
			var script string
			for _, step := range document.Steps {
				if strings.HasPrefix(step.Name, "Validate ") {
					script = step.Run
					break
				}
			}
			require.NotEmpty(t, script)
			for _, tc := range []struct {
				name       string
				repository string
				ref        string
				sha        string
				wantError  bool
			}{
				{name: "same repository", repository: "platform/repo", ref: "refs/heads/main", sha: "abc123"},
				{name: "cross repository", repository: "caller/repo", ref: "refs/heads/main", sha: "abc123", wantError: true},
				{name: "missing ref", repository: "platform/repo", sha: "abc123", wantError: true},
				{name: "missing sha", repository: "platform/repo", ref: "refs/heads/main", wantError: true},
			} {
				t.Run(tc.name, func(t *testing.T) {
					cmd := exec.Command("bash", "-c", script)
					cmd.Env = append(os.Environ(),
						"TARGET_REPOSITORY=platform/repo",
						"CURRENT_REPOSITORY="+tc.repository,
						"TARGET_REF="+tc.ref,
						"TARGET_SHA="+tc.sha,
						"UPLOAD_EVENT_NAME=push",
						"UPLOAD_REF=refs/heads/main",
						"UPLOAD_SHA=abc123",
						"PR_HEAD_REF=",
						"PR_HEAD_SHA=",
					)
					output, err := cmd.CombinedOutput()
					if tc.wantError {
						require.Error(t, err)
						assert.Contains(t, string(output), "::error::")
						assert.Contains(t, string(output), "Refusing to")
					} else {
						require.NoError(t, err, "%s", output)
					}
				})
			}
			if uploader.name == "coverage" {
				for _, tc := range []struct {
					name      string
					event     string
					ref       string
					sha       string
					prHeadRef string
					prHeadSHA string
					wantError bool
				}{
					{name: "different caller sha", event: "push", ref: "refs/heads/main", sha: "def456", wantError: true},
					{name: "different caller ref", event: "push", ref: "refs/heads/release", sha: "abc123", wantError: true},
					{name: "PR head differs from workflow", event: "pull_request", ref: "refs/pull/1/merge", sha: "abc123", prHeadRef: "main", prHeadSHA: "def456", wantError: true},
					{name: "PR head matches workflow", event: "pull_request", ref: "refs/pull/1/merge", sha: "def456", prHeadRef: "main", prHeadSHA: "abc123"},
					{name: "PR target head matches workflow", event: "pull_request_target", ref: "refs/heads/release", sha: "def456", prHeadRef: "main", prHeadSHA: "abc123"},
					{name: "PR head branch differs", event: "pull_request", ref: "refs/pull/1/merge", sha: "abc123", prHeadRef: "feature", prHeadSHA: "abc123", wantError: true},
					{name: "missing PR head metadata", event: "pull_request", ref: "refs/heads/main", sha: "abc123", wantError: true},
				} {
					t.Run(tc.name, func(t *testing.T) {
						cmd := exec.Command("bash", "-c", script)
						cmd.Env = append(os.Environ(),
							"TARGET_REPOSITORY=platform/repo",
							"CURRENT_REPOSITORY=platform/repo",
							"TARGET_REF=refs/heads/main",
							"TARGET_SHA=abc123",
							"UPLOAD_EVENT_NAME="+tc.event,
							"UPLOAD_REF="+tc.ref,
							"UPLOAD_SHA="+tc.sha,
							"PR_HEAD_REF="+tc.prHeadRef,
							"PR_HEAD_SHA="+tc.prHeadSHA,
						)
						output, err := cmd.CombinedOutput()
						if tc.wantError {
							require.Error(t, err)
							assert.Contains(t, string(output), "Refusing to upload coverage for a different workflow revision.")
						} else {
							require.NoError(t, err, "%s", output)
						}
					})
				}
			}
		})
	}
}
