//go:build !integration

package workflow

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/goccy/go-yaml"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestContinueWorkflowCompilation(t *testing.T) {
	for _, setting := range []string{"true", "false", "{max-hops: 5}"} {
		t.Run(setting, func(t *testing.T) {
			directory := t.TempDir()
			file := filepath.Join(directory, "continue.md")
			source := `---
on: workflow_dispatch
permissions:
  contents: read
engine: copilot
steps:
  - name: Install project dependencies
    run: npm ci
safe-outputs:
  continue-workflow: ` + setting + `
  create-pull-request:
    fallback-as-issue: false
---
Complete the task, keeping plan.md up to date between runs.
`
			require.NoError(t, os.WriteFile(file, []byte(source), 0600))
			compiler := NewCompiler(WithVersion("1.0.0"))
			require.NoError(t, compiler.CompileWorkflow(file))
			compiled, err := os.ReadFile(filepath.Join(directory, "continue.lock.yml"))
			require.NoError(t, err)
			var document struct {
				Jobs map[string]struct {
					Permissions map[string]string `yaml:"permissions"`
					Steps       []struct {
						Name string `yaml:"name"`
						Uses string `yaml:"uses"`
					} `yaml:"steps"`
				} `yaml:"jobs"`
			}
			require.NoError(t, yaml.Unmarshal(compiled, &document))
			content := string(compiled)
			if setting == "false" {
				assert.NotContains(t, content, "Checkout continuation branch")
				assert.NotContains(t, content, "continue_workflow\"")
				assert.NotContains(t, document.Jobs["safe_outputs"].Permissions, "actions")
				return
			}
			assert.Contains(t, content, "checkout_continue_workflow_branch.cjs")
			assert.Contains(t, content, "safe_outputs_continue_workflow.md")
			assert.Contains(t, content, "continue_workflow")
			assert.Regexp(t, `max_hops\\*":\s*`+map[string]string{"true": "3", "{max-hops: 5}": "5"}[setting]+`[,}]`, content)
			assert.Equal(t, "read", document.Jobs["agent"].Permissions["contents"])
			assert.Equal(t, "write", document.Jobs["safe_outputs"].Permissions["contents"])
			assert.Equal(t, "write", document.Jobs["safe_outputs"].Permissions["actions"])
			assert.Less(t, strings.Index(content, "Checkout continuation branch"), strings.Index(content, "id: agentic_execution"))
			resume, dependencies, runtime := -1, -1, -1
			for index, step := range document.Jobs["agent"].Steps {
				switch {
				case step.Name == "Checkout continuation branch":
					resume = index
				case step.Name == "Install project dependencies":
					dependencies = index
				case strings.HasPrefix(step.Uses, "actions/setup-node@"):
					runtime = index
				}
			}
			require.NotEqual(t, -1, resume)
			require.NotEqual(t, -1, dependencies)
			require.NotEqual(t, -1, runtime)
			assert.Less(t, resume, dependencies)
			assert.Less(t, resume, runtime)
		})
	}
}

func TestContinueWorkflowCheckoutUsesCheckoutCredentials(t *testing.T) {
	for _, token := range []string{"", "${{ secrets.CHECKOUT_TOKEN }}"} {
		data := continuationTestData()
		data.SafeOutputs.GitHubToken = "${{ secrets.LEDGER_TOKEN }}"
		data.CheckoutConfigs = []*CheckoutConfig{{GitHubToken: token}}
		var rendered strings.Builder
		NewCompiler().generateContinueWorkflowCheckout(&rendered, data)
		assert.Contains(t, rendered.String(), "github-token: "+resolveGitHubToken(token))
		assert.NotContains(t, rendered.String(), "github-token: ${{ secrets.LEDGER_TOKEN }}")
	}
	data := continuationTestData()
	data.CheckoutConfigs = []*CheckoutConfig{{GitHubApp: &GitHubAppConfig{AppID: "${{ vars.APP_ID }}", PrivateKey: "${{ secrets.APP_KEY }}"}}}
	var rendered strings.Builder
	NewCompiler().generateContinueWorkflowCheckout(&rendered, data)
	assert.Contains(t, rendered.String(), "github-token: ${{ steps.checkout-app-token-0.outputs.token }}")
}

func TestContinueWorkflowCustomCheckoutBeforeSetup(t *testing.T) {
	root := `  - name: Checkout task
    uses: actions/checkout@v5
    with:
      persist-credentials: false
      token: ${{ secrets.CUSTOM_CHECKOUT_TOKEN }}
`
	sidecar := `  - name: Checkout sidecar
    uses: actions/checkout@v5
    with:
      persist-credentials: false
      path: vendor
`
	for _, tt := range []struct {
		name        string
		checkouts   string
		resumeCount int
	}{
		{"root", root, 1},
		{"sidecar before root", sidecar + root, 1},
		{"repeated root", root + root, 2},
	} {
		t.Run(tt.name, func(t *testing.T) {
			directory := t.TempDir()
			file := filepath.Join(directory, "continue.md")
			source := `---
on: workflow_dispatch
permissions:
  contents: read
engine: copilot
safe-outputs:
  continue-workflow: true
  create-pull-request:
    fallback-as-issue: false
steps:
` + tt.checkouts + `  - name: Install project dependencies
    run: npm ci
---
Continue the task.
`
			require.NoError(t, os.WriteFile(file, []byte(source), 0600))
			require.NoError(t, NewCompiler(WithVersion("1.0.0")).CompileWorkflow(file))
			compiled, err := os.ReadFile(filepath.Join(directory, "continue.lock.yml"))
			require.NoError(t, err)
			var document struct {
				Jobs map[string]struct {
					Steps []struct {
						Name string            `yaml:"name"`
						Uses string            `yaml:"uses"`
						ID   string            `yaml:"id"`
						With map[string]string `yaml:"with"`
					} `yaml:"steps"`
				} `yaml:"jobs"`
			}
			require.NoError(t, yaml.Unmarshal(compiled, &document))
			firstRoot, firstResume, lastResume, runtime, dependencies := -1, -1, -1, -1, -1
			resumeIDs := make(map[string]bool)
			for index, step := range document.Jobs["agent"].Steps {
				switch {
				case step.Name == "Checkout task" && firstRoot == -1:
					firstRoot = index
				case step.Name == "Checkout continuation branch":
					if firstResume == -1 {
						firstResume = index
					}
					lastResume = index
					assert.False(t, resumeIDs[step.ID], "continuation step IDs must be unique")
					resumeIDs[step.ID] = true
					assert.Equal(t, "${{ secrets.CUSTOM_CHECKOUT_TOKEN }}", step.With["github-token"])
				case strings.HasPrefix(step.Uses, "actions/setup-node@"):
					runtime = index
				case step.Name == "Install project dependencies":
					dependencies = index
				}
			}
			require.Len(t, resumeIDs, tt.resumeCount)
			require.NotEqual(t, -1, firstRoot)
			require.NotEqual(t, -1, runtime)
			require.NotEqual(t, -1, dependencies)
			assert.Less(t, firstRoot, firstResume)
			assert.Less(t, firstResume, runtime)
			assert.Less(t, lastResume, dependencies)
		})
	}
}

func TestContinueWorkflowMaintenanceGeneration(t *testing.T) {
	for _, enabled := range []bool{true, false} {
		t.Run(map[bool]string{true: "enabled", false: "disabled"}[enabled], func(t *testing.T) {
			directory := t.TempDir()
			data := continuationTestData()
			data.SafeOutputs.NoOp = &NoOpConfig{ReportAsIssue: strPtr("false")}
			if !enabled {
				data.SafeOutputs.ContinueWorkflow = nil
			}
			require.NoError(t, GenerateMaintenanceWorkflow(context.Background(), GenerateMaintenanceWorkflowOptions{
				WorkflowDataList: []*WorkflowData{data},
				WorkflowDir:      directory,
				Version:          "v1.0.0",
				ActionMode:       ActionModeDev,
			}))
			compiled, err := os.ReadFile(filepath.Join(directory, "agentics-maintenance.yml"))
			if !enabled {
				require.ErrorIs(t, err, os.ErrNotExist)
				return
			}
			require.NoError(t, err)
			var document struct {
				On struct {
					Schedule []struct {
						Cron string `yaml:"cron"`
					} `yaml:"schedule"`
				} `yaml:"on"`
				Jobs map[string]struct {
					Permissions map[string]string `yaml:"permissions"`
				} `yaml:"jobs"`
			}
			require.NoError(t, yaml.Unmarshal(compiled, &document))
			require.Len(t, document.On.Schedule, 1)
			assert.Equal(t, "37 */2 * * *", document.On.Schedule[0].Cron)
			assert.Equal(t, map[string]string{"contents": "write", "actions": "write"}, document.Jobs["continue_workflow_dispatch"].Permissions)
			assert.Contains(t, string(compiled), "dispatch_continue_workflow.cjs")
			assert.Contains(t, string(compiled), "'dispatch_continuations'")
		})
	}
}
