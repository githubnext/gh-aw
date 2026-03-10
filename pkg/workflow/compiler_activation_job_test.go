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
const workflowCallRepo = "${{ github.event_name == 'workflow_call' && github.action_repository || github.repository }}"

func TestGenerateCheckoutGitHubFolderForActivation_WorkflowCall(t *testing.T) {
	tests := []struct {
		name             string
		onSection        string
		features         map[string]any
		inlinedImports   bool   // whether InlinedImports is enabled in WorkflowData
		wantRepository   string // expected repository: value ("" means field absent)
		wantNil          bool   // whether nil is expected (action-tag skip)
		wantGitHubSparse bool   // whether .github / .agents should be in sparse-checkout
		wantPersistFalse bool   // whether persist-credentials: false should be present
		wantFetchDepth1  bool   // whether fetch-depth: 1 should be present
	}{
		{
			name: "workflow_call trigger - cross-repo checkout with conditional repository",
			onSection: `"on":
  workflow_call:`,
			wantRepository:   workflowCallRepo,
			wantGitHubSparse: true,
			wantPersistFalse: true,
			wantFetchDepth1:  true,
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
			wantRepository:   workflowCallRepo,
			wantGitHubSparse: true,
			wantPersistFalse: true,
			wantFetchDepth1:  true,
		},
		{
			name: "workflow_call with inlined-imports - standard checkout without cross-repo expression",
			onSection: `"on":
  workflow_call:`,
			inlinedImports:   true,
			wantRepository:   "",
			wantGitHubSparse: true,
			wantPersistFalse: true,
			wantFetchDepth1:  true,
		},
		{
			name: "no workflow_call - standard checkout without repository field",
			onSection: `"on":
  issues:
    types: [opened]`,
			wantRepository:   "",
			wantGitHubSparse: true,
			wantPersistFalse: true,
			wantFetchDepth1:  true,
		},
		{
			name: "issue_comment only - no repository field",
			onSection: `"on":
  issue_comment:
    types: [created]`,
			wantRepository:   "",
			wantGitHubSparse: true,
			wantPersistFalse: true,
			wantFetchDepth1:  true,
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
			name:           "GitHub Actions expression for cross-repo",
			repository:     workflowCallRepo,
			wantRepository: true,
			wantRepoValue:  workflowCallRepo,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := NewCheckoutManager(nil).GenerateGitHubFolderCheckoutStep(tt.repository, GetActionPin)

			require.NotEmpty(t, result, "should return at least one YAML line")

			combined := strings.Join(result, "")

			assert.Contains(t, combined, "Checkout .github and .agents folders",
				"should have correct step name")
			assert.Contains(t, combined, ".github", "should include .github in sparse-checkout")
			assert.Contains(t, combined, ".agents", "should include .agents in sparse-checkout")
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
