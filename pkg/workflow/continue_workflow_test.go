//go:build !integration

package workflow

import (
	"bytes"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestContinueWorkflowExperimentalWarning(t *testing.T) {
	const warning = "Using experimental feature: cross-run continuation"
	tests := []struct {
		name    string
		data    *WorkflowData
		enabled bool
	}{
		{"ordinary workflow", &WorkflowData{}, false},
		{"queue observer", &WorkflowData{Tools: map[string]any{"work-queue": map[string]any{}}}, false},
		{"queue dispatcher", &WorkflowData{Tools: map[string]any{"work-queue": map[string]any{"worker": false}}}, false},
		{"queue worker", &WorkflowData{Tools: map[string]any{"work-queue": map[string]any{"worker": true}}}, true},
		{"standalone continuation", continuationTestData(), true},
		{"both paths", &WorkflowData{
			Tools:       map[string]any{"work-queue": map[string]any{"worker": true}},
			SafeOutputs: &SafeOutputsConfig{ContinueWorkflow: &ContinueWorkflowConfig{}},
		}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for _, batch := range []bool{false, true} {
				compiler := NewCompiler()
				compiler.SetBatchMode(batch)
				var output bytes.Buffer
				compiler.emitExperimentalFeatureWarningsTo(tt.data, &output)
				expected := 0
				if tt.enabled {
					expected = 1
				}
				assert.Equal(t, expected, compiler.GetWarningCount())
				assert.Equal(t, expected, compiler.GetExperimentalWarningCount())
				if batch {
					assert.Empty(t, output.String())
					assert.Equal(t, expected, compiler.GetExperimentalFeatureUsage()[warning])
				} else {
					assert.Equal(t, tt.enabled, strings.Contains(output.String(), warning))
				}
			}
		})
	}
}

func TestContinueWorkflowConfig(t *testing.T) {
	tests := []struct {
		name    string
		outputs map[string]any
		enabled bool
		hops    int
	}{
		{"absent", map[string]any{}, false, 0},
		{"disabled", map[string]any{"continue-workflow": false}, false, 0},
		{"boolean", map[string]any{"continue-workflow": true}, true, 3},
		{"null", map[string]any{"continue-workflow": nil}, true, 3},
		{"empty object", map[string]any{"continue-workflow": map[string]any{}}, true, 3},
		{"custom budget", map[string]any{"continue-workflow": map[string]any{"max-hops": 10}}, true, 10},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := NewCompiler().parseContinueWorkflowConfig(tt.outputs)
			if !tt.enabled {
				assert.Nil(t, config)
				return
			}
			require.NotNil(t, config)
			assert.Equal(t, tt.hops, config.EffectiveMaxHops())
			assert.Equal(t, strPtr("1"), config.Max)
		})
	}
}

func continuationTestData() *WorkflowData {
	return &WorkflowData{
		On:          "on:\n  workflow_dispatch:\n",
		Permissions: "contents: read",
		SafeOutputs: &SafeOutputsConfig{
			ContinueWorkflow:   &ContinueWorkflowConfig{},
			CreatePullRequests: &CreatePullRequestsConfig{},
		},
	}
}

func TestContinueWorkflowValidation(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*WorkflowData)
		error  string
	}{
		{"valid", func(*WorkflowData) {}, ""},
		{"disabled", func(d *WorkflowData) { d.SafeOutputs.ContinueWorkflow = nil }, ""},
		{"missing PR", func(d *WorkflowData) { d.SafeOutputs.CreatePullRequests = nil }, "requires safe-outputs.create-pull-request"},
		{"target repository", func(d *WorkflowData) { d.SafeOutputs.CreatePullRequests.TargetRepoSlug = "o/r" }, "target-repo or head-repo"},
		{"head repository", func(d *WorkflowData) { d.SafeOutputs.CreatePullRequests.HeadRepoSlug = "o/r" }, "target-repo or head-repo"},
		{"preserved name", func(d *WorkflowData) { d.SafeOutputs.CreatePullRequests.PreserveBranchName = true }, "preserve-branch-name"},
		{"patch transport", func(d *WorkflowData) { d.SafeOutputs.CreatePullRequests.PatchFormat = "am" }, "patch-format: bundle"},
		{"no checkout", func(d *WorkflowData) { d.CheckoutDisabled = true }, "repository checkout and contents: read"},
		{"no read permission", func(d *WorkflowData) { d.Permissions = "contents: none" }, "repository checkout and contents: read"},
		{"cached no read permission", func(d *WorkflowData) {
			d.CachedPermissions = NewPermissionsFromMap(map[PermissionScope]PermissionLevel{PermissionContents: PermissionNone})
		}, "contents: read"},
		{"no dispatch", func(d *WorkflowData) { d.On = "on: push" }, "requires an on.workflow_dispatch trigger"},
		{"required input", func(d *WorkflowData) {
			d.On = "on:\n  workflow_dispatch:\n    inputs:\n      task:\n        required: true\n"
		}, "task are required without a default"},
		{"required input with default", func(d *WorkflowData) {
			d.On = "on:\n  workflow_dispatch:\n    inputs:\n      task:\n        required: true\n        default: work\n"
		}, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data := continuationTestData()
			tt.mutate(data)
			err := validateContinueWorkflow(data)
			if tt.error == "" {
				require.NoError(t, err)
			} else {
				require.ErrorContains(t, err, tt.error)
			}
		})
	}
}

func TestContinueWorkflowHopBudget(t *testing.T) {
	for _, hops := range []int{-1, 0, 1, 3, 10, 11} {
		data := continuationTestData()
		data.SafeOutputs.ContinueWorkflow.MaxHops = &hops
		err := validateContinueWorkflow(data)
		if hops >= 1 && hops <= 10 {
			require.NoError(t, err)
		} else {
			require.ErrorContains(t, err, "must be between 1 and 10")
		}
	}
}

func TestContinueWorkflowHandlerConfigSharesPullRequestPolicy(t *testing.T) {
	data := continuationTestData()
	pr := data.SafeOutputs.CreatePullRequests
	pr.BranchPrefix = "automation/"
	pr.BaseBranch = "release"
	pr.MaxPatchSize = 512
	pr.MaxPatchFiles = 20
	pr.AllowedFiles = []string{"src/**", "plan.md"}
	pr.SignedCommits = boolPtr(false)
	pr.GitHubToken = "${{ secrets.PR_TOKEN }}"
	config := buildContinueWorkflowHandlerConfig(data.SafeOutputs)
	assert.Equal(t, 1, config["max"])
	assert.Equal(t, 3, config["max_hops"])
	assert.Equal(t, "automation/", config["branch_prefix"])
	assert.Equal(t, "release", config["base_branch"])
	assert.Equal(t, 512, config["max_patch_size"])
	assert.Equal(t, 20, config["max_patch_files"])
	assert.Equal(t, []string{"src/**", "plan.md"}, config["allowed_files"])
	assert.Equal(t, false, config["signed_commits"])
	assert.Equal(t, "${{ secrets.PR_TOKEN }}", config["github-token"])
	data.SafeOutputs.ContinueWorkflow.GitHubToken = "${{ secrets.CONTINUE_TOKEN }}"
	assert.Equal(t, "${{ secrets.CONTINUE_TOKEN }}", buildContinueWorkflowHandlerConfig(data.SafeOutputs)["github-token"])
}

func TestContinueWorkflowHandlerConfigUsesContinuationStaging(t *testing.T) {
	data := continuationTestData()
	data.SafeOutputs.CreatePullRequests.Staged = templatableBoolPtr("true")
	config := buildContinueWorkflowHandlerConfig(data.SafeOutputs)
	assert.NotContains(t, config, "staged")

	data.SafeOutputs.ContinueWorkflow.Staged = templatableBoolPtr("true")
	config = buildContinueWorkflowHandlerConfig(data.SafeOutputs)
	assert.Equal(t, true, config["staged"])
}

func TestContinueWorkflowCheckoutUsesRootCheckoutToken(t *testing.T) {
	data := continuationTestData()
	data.CheckoutConfigs = []*CheckoutConfig{{GitHubToken: "${{ secrets.CHECKOUT_TOKEN }}"}}
	var yaml strings.Builder
	NewCompiler().generateContinueWorkflowCheckout(&yaml, data)
	assert.Contains(t, yaml.String(), "github-token: ${{ secrets.CHECKOUT_TOKEN }}")
	assert.NotContains(t, yaml.String(), "github-token: ${{ secrets.GITHUB_TOKEN }}")
}

func TestContinueWorkflowStagingIsIndependentOfPullRequest(t *testing.T) {
	tests := []struct {
		name       string
		global     string
		continueAs string
		expected   any
		checkout   bool
	}{
		{"unstaged continuation with staged PR", "", "", nil, true},
		{"global staged", "true", "", nil, false},
		{"global staged cannot be overridden", "true", "false", false, false},
		{"explicit staged", "", "true", true, false},
		{"expression", "", "${{ inputs.preview }}", "${{ inputs.preview }}", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := continuationTestData().SafeOutputs
			config.CreatePullRequests.Staged = templatableBoolPtr("true")
			if tt.global != "" {
				config.Staged = templatableBoolPtr(tt.global)
			}
			if tt.continueAs != "" {
				config.ContinueWorkflow.Staged = templatableBoolPtr(tt.continueAs)
			}
			handler := buildContinueWorkflowHandlerConfig(config)
			if tt.expected == nil {
				assert.NotContains(t, handler, "staged")
			} else {
				assert.Equal(t, tt.expected, handler["staged"])
			}
			assert.Equal(t, tt.checkout, usesPatchesAndCheckouts(config))
		})
	}
}
