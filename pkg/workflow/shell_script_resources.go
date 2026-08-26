package workflow

import "github.com/github/gh-aw/pkg/sliceutil"

// ShellScriptResource is a shell script defined in workflow frontmatter that is
// not rendered as a GitHub Actions run step.
type ShellScriptResource struct {
	Name   string
	Script string
	Shell  string
}

// ShellScriptResources returns frontmatter shell scripts that require linting
// in addition to run steps extracted from the generated lock file.
func (data *WorkflowData) ShellScriptResources() []ShellScriptResource {
	if data == nil {
		return nil
	}

	var resources []ShellScriptResource
	if data.MCPScripts != nil {
		for _, name := range sliceutil.SortedKeys(data.MCPScripts.Tools) {
			tool := data.MCPScripts.Tools[name]
			if tool != nil && tool.Run != "" {
				resources = append(resources, ShellScriptResource{
					Name:   "mcp-scripts." + name,
					Script: tool.Run,
					Shell:  "bash",
				})
			}
		}
	}

	if data.Graders != nil {
		for _, name := range sliceutil.SortedKeys(data.Graders.Graders) {
			grader := data.Graders.Graders[name]
			if grader != nil && (grader.Enabled == nil || *grader.Enabled) && grader.evaluatorContent != "" {
				resources = append(resources, ShellScriptResource{
					Name:   "graders." + name,
					Script: grader.evaluatorContent,
					Shell:  "bash",
				})
			}
		}
	}

	return resources
}
