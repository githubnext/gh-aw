package workflow

import "github.com/github/gh-aw/pkg/sliceutil"

// ShellScriptResource is a shell script defined in workflow frontmatter that is
// not rendered as a GitHub Actions run step.
type ShellScriptResource struct {
	Name   string
	Script string
	Shell  string
	// Source identifies where the script originated (e.g. the workflow that
	// declared it and, for graders, the evaluator file path) so findings can
	// be traced back to the responsible frontmatter, even when multiple
	// workflows define resources with the same Name.
	Source string
}

// ShellScriptResources returns frontmatter shell scripts that require linting
// in addition to run steps extracted from the generated lock file.
func (data *WorkflowData) ShellScriptResources() []ShellScriptResource {
	if data == nil {
		return nil
	}

	workflowID := data.WorkflowID
	if workflowID == "" {
		workflowID = data.Name
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
					Source: workflowID,
				})
			}
		}
	}

	if data.Graders != nil {
		for _, name := range sliceutil.SortedKeys(data.Graders.Graders) {
			grader := data.Graders.Graders[name]
			if grader != nil && (grader.Enabled == nil || *grader.Enabled) && grader.evaluatorContent != "" {
				source := workflowID
				if grader.Run != "" {
					source = grader.Run
				}
				resources = append(resources, ShellScriptResource{
					Name:   "graders." + name,
					Script: grader.evaluatorContent,
					Shell:  "bash",
					Source: source,
				})
			}
		}
	}

	return resources
}
