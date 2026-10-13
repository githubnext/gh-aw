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
