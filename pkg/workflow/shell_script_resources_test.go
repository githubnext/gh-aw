package workflow

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestShellScriptResources(t *testing.T) {
	data := &WorkflowData{
		WorkflowID: "my-workflow",
		MCPScripts: &MCPScriptsConfig{
			Tools: map[string]*MCPScriptToolConfig{
				"zulu":  {Run: "echo zulu"},
				"alpha": {Run: "echo alpha"},
				"node":  {Script: "return {}"},
			},
		},
		Graders: &GradersConfig{
			Graders: map[string]*GraderDefinition{
				"loops": {ID: "loops"},
			},
		},
	}

	assert.Equal(t, []ShellScriptResource{
		{Name: "mcp-scripts.alpha", Script: "echo alpha", Shell: "bash", Source: "my-workflow"},
		{Name: "mcp-scripts.zulu", Script: "echo zulu", Shell: "bash", Source: "my-workflow"},
	}, data.ShellScriptResources())
}
