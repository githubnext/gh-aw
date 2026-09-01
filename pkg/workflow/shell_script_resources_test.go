package workflow

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestShellScriptResources(t *testing.T) {
	disabled := false
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
				"operational-value": {Run: ".github/graders/example-operational-value.sh", evaluatorContent: "#!/usr/bin/env bash\necho grade"},
				"disabled":          {Enabled: &disabled, evaluatorContent: "#!/usr/bin/env bash\necho skipped"},
			},
		},
	}

	assert.Equal(t, []ShellScriptResource{
		{Name: "mcp-scripts.alpha", Script: "echo alpha", Shell: "bash", Source: "my-workflow"},
		{Name: "mcp-scripts.zulu", Script: "echo zulu", Shell: "bash", Source: "my-workflow"},
		{Name: "graders.operational-value", Script: "#!/usr/bin/env bash\necho grade", Shell: "bash", Source: ".github/graders/example-operational-value.sh"},
	}, data.ShellScriptResources())
}
