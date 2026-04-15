//go:build !integration

package workflow

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// workflowCallRepo is the expression injected into the repository: field of the
// activation-job checkout step when a workflow_call trigger is detected.
// The resolve-host-repo step uses job.workflow_repository to identify
// the platform repo, correctly handling all relay patterns including cross-repo
// and cross-org scenarios.
const workflowCallRepo = "${{ steps.resolve-host-repo.outputs.target_repo }}"

// workflowCallRef is the expression injected into the ref: field of the activation-job
// checkout step when a workflow_call trigger is detected without inlined imports.
// Uses job.workflow_sha for immutable pinning to the exact executing revision.
const workflowCallRef = "${{ steps.resolve-host-repo.outputs.target_ref }}"

// sameRepoCondition is the if: condition injected into the .github checkout step when
// no custom activation token is configured. It restricts the checkout to same-repo
// workflow_call invocations to prevent failures when GITHUB_TOKEN cannot read a private
// callee repository in cross-repo scenarios.
const sameRepoCondition = "steps.resolve-host-repo.outputs.target_repo == github.repository"

func TestGenerateCheckoutGitHubFolderForActivation_WorkflowCall(t *testing.T) {
	tests := []struct {
		name                  string
		onSection             string
		features              map[string]any
		inlinedImports        bool   // whether InlinedImports is enabled in WorkflowData
		wantRepository        string // expected repository: value ("" means field absent)
		wantRef               string // expected ref: value ("" means field absent)
		wantNil               bool   // whether nil is expected (action-tag skip)
		wantGitHubSparse      bool   // whether .github / .agents should be in sparse-checkout
		wantPersistFalse      bool   // whether persist-credentials: false should be present
		wantFetchDepth1       bool   // whether fetch-depth: 1 should be present
		wantSameRepoCondition bool   // whether if: same-repo condition should be present
	}{
		{
			name: "workflow_call trigger - cross-repo checkout with conditional repository and ref",
			onSection: `"on":
  workflow_call:`,
			wantRepository:        workflowCallRepo,
			wantRef:               workflowCallRef,
			wantGitHubSparse:      true,
			wantPersistFalse:      true,
			wantFetchDepth1:       true,
			wantSameRepoCondition: true, // no custom token → restrict to same-repo only
		},
		{
			name: "workflow_call with inputs and mixed triggers",
			onSection: `"on":
  issue_comment:
    types: [created]
  workflow_call:
    inputs:
      issue_number:
        required: true
        type: number`,
			wantRepository:        workflowCallRepo,
			wantRef:               workflowCallRef,
			wantGitHubSparse:      true,
			wantPersistFalse:      true,
			wantFetchDepth1:       true,
			wantSameRepoCondition: true, // no custom token → restrict to same-repo only
		},
		{
			name: "workflow_call with inlined-imports - standard checkout without cross-repo expression",
			onSection: `"on":
  workflow_call:`,
			inlinedImports:        true,
			wantRepository:        "",
			wantRef:               "",
			wantGitHubSparse:      true,
			wantPersistFalse:      true,
			wantFetchDepth1:       true,
			wantSameRepoCondition: false, // inlined-imports → no cross-repo checkout path
		},
		{
			name: "no workflow_call - standard checkout without repository field",
			onSection: `"on":
  issues:
    types: [opened]`,
			wantRepository:        "",
			wantRef:               "",
			wantGitHubSparse:      true,
			wantPersistFalse:      true,
			wantFetchDepth1:       true,
			wantSameRepoCondition: false, // non-workflow_call trigger → no same-repo condition
		},
		{
			name: "issue_comment only - no repository field",
			onSection: `"on":
  issue_comment:
    types: [created]`,
			wantRepository:        "",
			wantRef:               "",
			wantGitHubSparse:      true,
			wantPersistFalse:      true,
			wantFetchDepth1:       true,
			wantSameRepoCondition: false, // non-workflow_call trigger → no same-repo condition
		},
		{
			name: "action-tag specified with workflow_call - no checkout emitted",
			onSection: `"on":
  workflow_call:`,
			features: map[string]any{"action-tag": "v1.0.0"},
			wantNil:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := NewCompilerWithVersion("dev")
			c.SetActionMode(ActionModeDev)

			data := &WorkflowData{
				On:             tt.onSection,
				Features:       tt.features,
				InlinedImports: tt.inlinedImports,
			}

			result := c.generateCheckoutGitHubFolderForActivation(data)

			if tt.wantNil {
				assert.Nil(t, result, "expected nil checkout steps for action-tag case")
				return
			}

			require.NotNil(t, result, "expected non-nil checkout steps")
			require.NotEmpty(t, result, "expected at least one checkout step line")

			combined := strings.Join(result, "")

			// Verify step structure
			assert.Contains(t, combined, "Checkout .github and .agents folders",
				"checkout step should have correct name")
			assert.Contains(t, combined, "actions/checkout",
				"checkout step should use actions/checkout")

			// Verify sparse-checkout includes required folders
			if tt.wantGitHubSparse {
				assert.Contains(t, combined, ".github", "sparse-checkout should include .github")
				assert.Contains(t, combined, ".agents", "sparse-checkout should include .agents")
				assert.Contains(t, combined, "actions/setup", "sparse-checkout should include actions/setup to preserve post step")
			}

			// Verify security defaults
			if tt.wantPersistFalse {
				assert.Contains(t, combined, "persist-credentials: false",
					"checkout should disable credential persistence")
			}
			if tt.wantFetchDepth1 {
				assert.Contains(t, combined, "fetch-depth: 1",
					"checkout should use shallow clone")
			}

			// Verify repository field
			if tt.wantRepository != "" {
				assert.Contains(t, combined, "repository: "+tt.wantRepository,
					"cross-repo checkout should include conditional repository expression")
			} else {
				assert.NotContains(t, combined, "repository:",
					"standard checkout should not include repository field")
			}

			// Verify ref field
			if tt.wantRef != "" {
				assert.Contains(t, combined, "ref: "+tt.wantRef,
					"cross-repo checkout should include ref expression to preserve callee branch")
			} else {
				assert.NotContains(t, combined, "ref:",
					"standard checkout should not include ref field")
			}

			// Verify same-repo if: condition
			if tt.wantSameRepoCondition {
				assert.Contains(t, combined, "if: "+sameRepoCondition,
					"workflow_call checkout without custom token should include same-repo guard to prevent cross-repo GITHUB_TOKEN failures")
			} else {
				assert.NotContains(t, combined, "if: "+sameRepoCondition,
					"checkout should not have same-repo guard when using custom token or non-workflow_call trigger")
			}
		})
	}
}

func TestGenerateGitHubFolderCheckoutStep(t *testing.T) {
	tests := []struct {
		name           string
		repository     string
		wantRepository bool
		wantRepoValue  string
	}{
		{
			name:           "empty repository - no repository field",
			repository:     "",
			wantRepository: false,
		},
		{
			name:           "literal repository value",
			repository:     "org/platform-repo",
			wantRepository: true,
			wantRepoValue:  "org/platform-repo",
		},
		{
			name:           "step output expression for cross-repo",
			repository:     workflowCallRepo,
			wantRepository: true,
			wantRepoValue:  workflowCallRepo,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := NewCheckoutManager(nil).GenerateGitHubFolderCheckoutStep(tt.repository, "", "", GetActionPin)

			require.NotEmpty(t, result, "should return at least one YAML line")

			combined := strings.Join(result, "")

			assert.Contains(t, combined, "Checkout .github and .agents folders",
				"should have correct step name")
			assert.Contains(t, combined, ".github", "should include .github in sparse-checkout")
			assert.Contains(t, combined, ".agents", "should include .agents in sparse-checkout")
			assert.NotContains(t, combined, "actions/setup", "base method should not include actions/setup without extraPaths")
			assert.Contains(t, combined, "sparse-checkout-cone-mode: true",
				"should enable cone mode for sparse checkout")
			assert.Contains(t, combined, "fetch-depth: 1", "should use shallow clone")
			assert.Contains(t, combined, "persist-credentials: false",
				"should disable credential persistence")

			if tt.wantRepository {
				assert.Contains(t, combined, "repository: "+tt.wantRepoValue,
					"should include repository field with correct value")
			} else {
				assert.NotContains(t, combined, "repository:",
					"should not include repository field when empty")
			}
		})
	}
}

// TestGenerateResolveHostRepoStep verifies that the resolve-host-repo step uses
// job.workflow_* context fields to resolve the platform repository.
func TestGenerateResolveHostRepoStep(t *testing.T) {
	c := NewCompilerWithVersion("dev")
	c.SetActionMode(ActionModeDev)

	result := c.generateResolveHostRepoStep()

	assert.Contains(t, result, "resolve-host-repo",
		"step should have the correct id")
	assert.Contains(t, result, "Resolve host repo for activation checkout",
		"step should have the correct name")
	assert.Contains(t, result, "actions/github-script",
		"step should use actions/github-script")
	assert.Contains(t, result, "resolve_host_repo.cjs",
		"step should require resolve_host_repo.cjs")

	// Values must be passed via env vars, not interpolated into script source
	assert.Contains(t, result, "JOB_WORKFLOW_REPOSITORY: ${{ job.workflow_repository }}",
		"step should pass job.workflow_repository via env var")
	assert.Contains(t, result, "JOB_WORKFLOW_SHA: ${{ job.workflow_sha }}",
		"step should pass job.workflow_sha via env var")
}

// TestCheckoutDoesNotUseEventNameExpression verifies that the checkout step for
// workflow_call triggers uses the resolve-host-repo step output instead of the
// broken event_name == 'workflow_call' expression.
func TestCheckoutDoesNotUseEventNameExpression(t *testing.T) {
	c := NewCompilerWithVersion("dev")
	c.SetActionMode(ActionModeDev)

	data := &WorkflowData{
		On: `"on":
  workflow_call:`,
	}

	result := c.generateCheckoutGitHubFolderForActivation(data)
	combined := strings.Join(result, "")

	// Must use the step output, not the broken expression
	assert.Contains(t, combined, "steps.resolve-host-repo.outputs.target_repo",
		"checkout must reference the resolve-host-repo step output")

	// Must NOT use the old broken event_name expression
	assert.NotContains(t, combined, "github.event_name == 'workflow_call'",
		"checkout must not use the broken event_name-based expression")
	assert.NotContains(t, combined, "github.action_repository",
		"checkout must not use github.action_repository")
}

// TestActivationJobTargetRepoOutput verifies that the activation job exposes target_repo as an
// output when a workflow_call trigger is present (without inlined imports), so that agent and
// safe_outputs jobs can reference needs.activation.outputs.target_repo.
func TestActivationJobTargetRepoOutput(t *testing.T) {
	tests := []struct {
		name             string
		onSection        string
		inlinedImports   bool
		expectTargetRepo bool
	}{
		{
			name: "workflow_call trigger - target_repo output added",
			onSection: `"on":
  workflow_call:`,
			expectTargetRepo: true,
		},
		{
			name: "mixed triggers with workflow_call - target_repo output added",
			onSection: `"on":
  issue_comment:
    types: [created]
  workflow_call:`,
			expectTargetRepo: true,
		},
		{
			name: "workflow_call with inlined-imports - no target_repo output",
			onSection: `"on":
  workflow_call:`,
			inlinedImports:   true,
			expectTargetRepo: false,
		},
		{
			name: "no workflow_call - no target_repo output",
			onSection: `"on":
  issues:
    types: [opened]`,
			expectTargetRepo: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			compiler := NewCompilerWithVersion("dev")
			compiler.SetActionMode(ActionModeDev)

			data := &WorkflowData{
				Name:           "test-workflow",
				On:             tt.onSection,
				InlinedImports: tt.inlinedImports,
				AI:             "copilot",
			}

			job, err := compiler.buildActivationJob(data, false, "", "test.lock.yml")
			require.NoError(t, err, "buildActivationJob should succeed")
			require.NotNil(t, job, "activation job should not be nil")

			if tt.expectTargetRepo {
				assert.Contains(t, job.Outputs, "target_repo",
					"activation job should expose target_repo output for downstream jobs")
				assert.Equal(t,
					"${{ steps.resolve-host-repo.outputs.target_repo }}",
					job.Outputs["target_repo"],
					"target_repo output should reference resolve-host-repo step")
			} else {
				assert.NotContains(t, job.Outputs, "target_repo",
					"activation job should not expose target_repo when workflow_call is absent or inlined-imports enabled")
			}
		})
	}
}

// TestActivationJobTargetRefOutput verifies that the activation job exposes target_ref as an
// output when a workflow_call trigger is present (without inlined imports), alongside target_repo.
// This enables callee-branch-pinned relays to check out the correct branch.
func TestActivationJobTargetRefOutput(t *testing.T) {
	tests := []struct {
		name            string
		onSection       string
		inlinedImports  bool
		expectTargetRef bool
	}{
		{
			name: "workflow_call trigger - target_ref output added",
			onSection: `"on":
  workflow_call:`,
			expectTargetRef: true,
		},
		{
			name: "mixed triggers with workflow_call - target_ref output added",
			onSection: `"on":
  issue_comment:
    types: [created]
  workflow_call:`,
			expectTargetRef: true,
		},
		{
			name: "workflow_call with inlined-imports - no target_ref output",
			onSection: `"on":
  workflow_call:`,
			inlinedImports:  true,
			expectTargetRef: false,
		},
		{
			name: "no workflow_call - no target_ref output",
			onSection: `"on":
  issues:
    types: [opened]`,
			expectTargetRef: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			compiler := NewCompilerWithVersion("dev")
			compiler.SetActionMode(ActionModeDev)

			data := &WorkflowData{
				Name:           "test-workflow",
				On:             tt.onSection,
				InlinedImports: tt.inlinedImports,
				AI:             "copilot",
			}

			job, err := compiler.buildActivationJob(data, false, "", "test.lock.yml")
			require.NoError(t, err, "buildActivationJob should succeed")
			require.NotNil(t, job, "activation job should not be nil")

			if tt.expectTargetRef {
				assert.Contains(t, job.Outputs, "target_ref",
					"activation job should expose target_ref output for downstream jobs")
				assert.Equal(t,
					"${{ steps.resolve-host-repo.outputs.target_ref }}",
					job.Outputs["target_ref"],
					"target_ref output should reference resolve-host-repo step")
			} else {
				assert.NotContains(t, job.Outputs, "target_ref",
					"activation job should not expose target_ref when workflow_call is absent or inlined-imports enabled")
			}
		})
	}
}

// TestActivationJobTargetRepoNameOutput verifies that the activation job exposes target_repo_name
// as an output when a workflow_call trigger is present (without inlined imports). This repo-name-only
// output is required for actions/create-github-app-token which expects repo names without the
// owner prefix when `owner` is also set.
func TestActivationJobTargetRepoNameOutput(t *testing.T) {
	tests := []struct {
		name                 string
		onSection            string
		inlinedImports       bool
		expectTargetRepoName bool
	}{
		{
			name: "workflow_call trigger - target_repo_name output added",
			onSection: `"on":
  workflow_call:`,
			expectTargetRepoName: true,
		},
		{
			name: "mixed triggers with workflow_call - target_repo_name output added",
			onSection: `"on":
  issue_comment:
    types: [created]
  workflow_call:`,
			expectTargetRepoName: true,
		},
		{
			name: "workflow_call with inlined-imports - no target_repo_name output",
			onSection: `"on":
  workflow_call:`,
			inlinedImports:       true,
			expectTargetRepoName: false,
		},
		{
			name: "no workflow_call - no target_repo_name output",
			onSection: `"on":
  issues:
    types: [opened]`,
			expectTargetRepoName: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			compiler := NewCompilerWithVersion("dev")
			compiler.SetActionMode(ActionModeDev)

			data := &WorkflowData{
				Name:           "test-workflow",
				On:             tt.onSection,
				InlinedImports: tt.inlinedImports,
				AI:             "copilot",
			}

			job, err := compiler.buildActivationJob(data, false, "", "test.lock.yml")
			require.NoError(t, err, "buildActivationJob should succeed")
			require.NotNil(t, job, "activation job should not be nil")

			if tt.expectTargetRepoName {
				assert.Contains(t, job.Outputs, "target_repo_name",
					"activation job should expose target_repo_name output for GitHub App token minting")
				assert.Equal(t,
					"${{ steps.resolve-host-repo.outputs.target_repo_name }}",
					job.Outputs["target_repo_name"],
					"target_repo_name output should reference resolve-host-repo step")
			} else {
				assert.NotContains(t, job.Outputs, "target_repo_name",
					"activation job should not expose target_repo_name when workflow_call is absent or inlined-imports enabled")
			}
		})
	}
}

// TestCheckoutGitHubFolderIncludesRef verifies that the activation checkout emits a ref: field
// when a workflow_call trigger is present. This ensures caller-hosted relays pinned to a
// feature branch check out the correct platform branch during activation.
func TestCheckoutGitHubFolderIncludesRef(t *testing.T) {
	tests := []struct {
		name           string
		onSection      string
		inlinedImports bool
		wantRef        bool
	}{
		{
			name: "workflow_call trigger - ref field emitted",
			onSection: `"on":
  workflow_call:`,
			wantRef: true,
		},
		{
			name: "mixed triggers with workflow_call - ref field emitted",
			onSection: `"on":
  issue_comment:
    types: [created]
  workflow_call:`,
			wantRef: true,
		},
		{
			name: "workflow_call with inlined-imports - no ref field",
			onSection: `"on":
  workflow_call:`,
			inlinedImports: true,
			wantRef:        false,
		},
		{
			name: "no workflow_call - no ref field",
			onSection: `"on":
  issues:
    types: [opened]`,
			wantRef: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := NewCompilerWithVersion("dev")
			c.SetActionMode(ActionModeDev)

			data := &WorkflowData{
				On:             tt.onSection,
				InlinedImports: tt.inlinedImports,
			}

			result := c.generateCheckoutGitHubFolderForActivation(data)
			combined := strings.Join(result, "")

			if tt.wantRef {
				assert.Contains(t, combined, "ref: "+workflowCallRef,
					"cross-repo checkout should include ref: expression")
			} else {
				assert.NotContains(t, combined, "ref:",
					"non-cross-repo checkout should not include ref: field")
			}
		})
	}
}

// TestGenerateCheckoutGitHubFolderForActivation_ActionsModeSetupPath verifies that
// actions/setup is included in the sparse-checkout only when in dev mode, because
// dev mode references the action via a local workspace path (./actions/setup) while
// release/script/action modes reference it remotely (runner cache, not workspace).
func TestGenerateCheckoutGitHubFolderForActivation_ActionsModeSetupPath(t *testing.T) {
	tests := []struct {
		name              string
		mode              ActionMode
		wantSetupInSparse bool
	}{
		{
			name:              "dev mode - actions/setup must be in sparse-checkout",
			mode:              ActionModeDev,
			wantSetupInSparse: true,
		},
		{
			name:              "release mode - actions/setup must NOT be in sparse-checkout",
			mode:              ActionModeRelease,
			wantSetupInSparse: false,
		},
		{
			name:              "script mode - actions/setup must NOT be in sparse-checkout",
			mode:              ActionModeScript,
			wantSetupInSparse: false,
		},
		{
			name:              "action mode - actions/setup must NOT be in sparse-checkout",
			mode:              ActionModeAction,
			wantSetupInSparse: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := NewCompilerWithVersion("dev")
			c.SetActionMode(tt.mode)

			data := &WorkflowData{
				On: `"on":
  issues:
    types: [opened]`,
			}

			result := c.generateCheckoutGitHubFolderForActivation(data)
			require.NotNil(t, result, "should return checkout steps")
			combined := strings.Join(result, "")

			if tt.wantSetupInSparse {
				assert.Contains(t, combined, "actions/setup",
					"dev mode should include actions/setup to preserve local action during post step")
			} else {
				assert.NotContains(t, combined, "actions/setup",
					"non-dev mode should not include actions/setup (action is in runner cache, not workspace)")
			}
		})
	}
}

// TestGenerateGitHubFolderCheckoutStep_ExtraPaths verifies that extraPaths are
// correctly appended to the sparse-checkout list.
func TestGenerateGitHubFolderCheckoutStep_ExtraPaths(t *testing.T) {
	result := NewCheckoutManager(nil).GenerateGitHubFolderCheckoutStep("", "", "", GetActionPin, "actions/setup", "custom/path")
	combined := strings.Join(result, "")

	assert.Contains(t, combined, ".github", "should include .github")
	assert.Contains(t, combined, ".agents", "should include .agents")
	assert.Contains(t, combined, "actions/setup", "should include extra path actions/setup")
	assert.Contains(t, combined, "custom/path", "should include extra path custom/path")
}

// TestGenerateGitHubFolderCheckoutStep_Token verifies that the token: field is emitted
// only for non-default tokens, supporting cross-org workflow_call scenarios.
func TestGenerateGitHubFolderCheckoutStep_Token(t *testing.T) {
	tests := []struct {
		name      string
		token     string
		wantToken bool
		wantValue string
	}{
		{
			name:      "empty token - no token field",
			token:     "",
			wantToken: false,
		},
		{
			name:      "default GITHUB_TOKEN - no token field emitted",
			token:     "${{ secrets.GITHUB_TOKEN }}",
			wantToken: false,
		},
		{
			name:      "custom PAT secret - token field emitted",
			token:     "${{ secrets.CROSS_ORG_TOKEN }}",
			wantToken: true,
			wantValue: "${{ secrets.CROSS_ORG_TOKEN }}",
		},
		{
			name:      "GitHub App minted token - token field emitted",
			token:     "${{ steps.activation-app-token.outputs.token }}",
			wantToken: true,
			wantValue: "${{ steps.activation-app-token.outputs.token }}",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := NewCheckoutManager(nil).GenerateGitHubFolderCheckoutStep("org/repo", "", tt.token, GetActionPin)
			combined := strings.Join(result, "")

			if tt.wantToken {
				assert.Contains(t, combined, "token: "+tt.wantValue,
					"should include token field with correct value")
			} else {
				assert.NotContains(t, combined, "token:",
					"should not include token field for default or empty token")
			}
		})
	}
}

// TestCheckoutTokenPropagatedToActivation verifies that the on.github-token frontmatter field
// is propagated to the activation job's .github checkout step for cross-org workflow_call support.
func TestCheckoutTokenPropagatedToActivation(t *testing.T) {
	tests := []struct {
		name            string
		activationToken string
		onSection       string
		wantTokenInStep bool
		wantTokenValue  string
	}{
		{
			name:            "custom token with workflow_call - token emitted in checkout",
			activationToken: "${{ secrets.CROSS_ORG_TOKEN }}",
			onSection: `"on":
  workflow_call:`,
			wantTokenInStep: true,
			wantTokenValue:  "${{ secrets.CROSS_ORG_TOKEN }}",
		},
		{
			name:            "default GITHUB_TOKEN - no token field in checkout",
			activationToken: "",
			onSection: `"on":
  workflow_call:`,
			wantTokenInStep: false,
		},
		{
			name:            "custom token without workflow_call - token emitted in checkout",
			activationToken: "${{ secrets.CROSS_ORG_TOKEN }}",
			onSection: `"on":
  issues:
    types: [opened]`,
			wantTokenInStep: true,
			wantTokenValue:  "${{ secrets.CROSS_ORG_TOKEN }}",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := NewCompilerWithVersion("dev")
			c.SetActionMode(ActionModeDev)

			data := &WorkflowData{
				On:                    tt.onSection,
				ActivationGitHubToken: tt.activationToken,
			}

			result := c.generateCheckoutGitHubFolderForActivation(data)
			combined := strings.Join(result, "")

			if tt.wantTokenInStep {
				assert.Contains(t, combined, "token: "+tt.wantTokenValue,
					"checkout step should include token field for cross-org support")
			} else {
				assert.NotContains(t, combined, "token:",
					"checkout step should not include token field when using default GITHUB_TOKEN")
			}
		})
	}
}

// TestCheckoutSameRepoGuardWithCustomToken verifies that the same-repo if: condition
// is NOT added to the checkout step when a custom activation token is configured.
// Cross-repo callers with a custom token can access private callee repositories, so
// the guard is not needed.
func TestCheckoutSameRepoGuardWithCustomToken(t *testing.T) {
	tests := []struct {
		name                  string
		activationToken       string
		onSection             string
		wantSameRepoCondition bool
	}{
		{
			name:            "custom PAT with workflow_call - no same-repo guard",
			activationToken: "${{ secrets.CROSS_ORG_TOKEN }}",
			onSection: `"on":
  workflow_call:`,
			wantSameRepoCondition: false,
		},
		{
			name:            "default GITHUB_TOKEN with workflow_call - same-repo guard present",
			activationToken: "",
			onSection: `"on":
  workflow_call:`,
			wantSameRepoCondition: true,
		},
		{
			name:            "default GITHUB_TOKEN without workflow_call - no same-repo guard",
			activationToken: "",
			onSection: `"on":
  issues:
    types: [opened]`,
			wantSameRepoCondition: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := NewCompilerWithVersion("dev")
			c.SetActionMode(ActionModeDev)

			data := &WorkflowData{
				On:                    tt.onSection,
				ActivationGitHubToken: tt.activationToken,
			}

			result := c.generateCheckoutGitHubFolderForActivation(data)
			combined := strings.Join(result, "")

			if tt.wantSameRepoCondition {
				assert.Contains(t, combined, "if: "+sameRepoCondition,
					"workflow_call checkout with default GITHUB_TOKEN should include same-repo guard to prevent cross-repo failures")
			} else {
				assert.NotContains(t, combined, "if: "+sameRepoCondition,
					"checkout should not have same-repo guard when custom token is configured or non-workflow_call trigger")
			}
		})
	}
}

// TestHashCheckTokenPropagation verifies that the on.github-token frontmatter field
// is propagated to the "Check workflow lock file" step for cross-org workflow_call support.
func TestHashCheckTokenPropagation(t *testing.T) {
	tests := []struct {
		name            string
		activationToken string
		wantTokenInStep bool
		wantTokenValue  string
	}{
		{
			name:            "custom token - github-token emitted in hash check step",
			activationToken: "${{ secrets.CROSS_ORG_TOKEN }}",
			wantTokenInStep: true,
			wantTokenValue:  "${{ secrets.CROSS_ORG_TOKEN }}",
		},
		{
			name:            "default GITHUB_TOKEN - no github-token field in hash check step",
			activationToken: "",
			wantTokenInStep: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			compiler := NewCompilerWithVersion("dev")
			compiler.SetActionMode(ActionModeDev)

			data := &WorkflowData{
				Name: "test-workflow",
				On: `"on":
  workflow_call:`,
				ActivationGitHubToken: tt.activationToken,
				AI:                    "copilot",
			}

			job, err := compiler.buildActivationJob(data, false, "", "test.lock.yml")
			require.NoError(t, err, "buildActivationJob should succeed")
			require.NotNil(t, job, "activation job should not be nil")

			// Find the check-lock-file step in the job steps
			combined := strings.Join(job.Steps, "")
			// Extract the check-lock-file step region
			lockFileIdx := strings.Index(combined, "id: check-lock-file")
			require.NotEqual(t, -1, lockFileIdx, "check-lock-file step should be present")

			// Get a window around the lock file step to check for github-token
			lockFileSection := combined[lockFileIdx:]
			nextStepIdx := strings.Index(lockFileSection[10:], "      - name:")
			if nextStepIdx != -1 {
				lockFileSection = lockFileSection[:nextStepIdx+10]
			}

			if tt.wantTokenInStep {
				assert.Contains(t, lockFileSection, "github-token: "+tt.wantTokenValue,
					"hash check step should include github-token field for cross-org support")
			} else {
				assert.NotContains(t, lockFileSection, "github-token:",
					"hash check step should not include github-token field when using default GITHUB_TOKEN")
			}
		})
	}
}

// TestInjectIfConditionAfterName verifies the robust line-oriented implementation of
// injectIfConditionAfterName: it finds "- name:", derives indentation from context,
// is idempotent, and logs a warning when no name line is found.
func TestInjectIfConditionAfterName(t *testing.T) {
	const cond = "some.condition == true"

	t.Run("injects after - name: line with inferred indent", func(t *testing.T) {
		step := "      - name: My step\n        uses: actions/checkout@abc\n"
		got := injectIfConditionAfterName(step, cond)
		assert.Contains(t, got, "        if: "+cond+"\n",
			"if: field should appear with the same 8-space indent as other fields")
		assert.Less(t, strings.Index(got, "if: "+cond), strings.Index(got, "uses:"),
			"if: field should appear before uses:")
	})

	t.Run("idempotent — does not double-inject", func(t *testing.T) {
		step := "      - name: My step\n        if: " + cond + "\n        uses: actions/checkout@abc\n"
		got := injectIfConditionAfterName(step, cond)
		assert.Equal(t, step, got, "step should be unchanged when if: is already present")
		assert.Equal(t, 1, strings.Count(got, "if: "+cond),
			"if: condition should appear exactly once")
	})

	t.Run("returns step unchanged when no - name: line found", func(t *testing.T) {
		step := "        uses: actions/checkout@abc\n        with:\n          fetch-depth: 1\n"
		got := injectIfConditionAfterName(step, cond)
		assert.Equal(t, step, got, "step without - name: should be returned unchanged")
	})
}
