package cli

import (
	"github.com/github/gh-aw/pkg/sliceutil"
	"github.com/github/gh-aw/pkg/workflow/codemod"
)

// getWorkflowDispatchRequiredFalseCodemod makes manual inputs optional for command triggers.
func getWorkflowDispatchRequiredFalseCodemod() Codemod {
	return Codemod{
		ID:           "workflow-dispatch-required-false-with-slash-command",
		Name:         "Set workflow_dispatch inputs required: false for command triggers",
		Description:  "When on.slash_command or on.label_command is present, rewrites workflow_dispatch.inputs.*.required: true to required: false because auto-dispatched triggers cannot enforce required manual inputs.",
		IntroducedIn: "1.5.0",
		Apply: func(content string, frontmatter map[string]any) (string, bool, error) {
			onMap, ok := frontmatter["on"].(map[string]any)
			if !ok {
				return content, false, nil
			}
			_, slash := onMap["slash_command"]
			_, label := onMap["label_command"]
			if !slash && !label {
				return content, false, nil
			}
			wd, ok := onMap["workflow_dispatch"].(map[string]any)
			if !ok {
				return content, false, nil
			}
			inputs, ok := wd["inputs"].(map[string]any)
			if !ok {
				return content, false, nil
			}
			return codemod.Apply(content, func(f *codemod.Frontmatter) error {
				for _, name := range sliceutil.SortedKeys(inputs) {
					if input, ok := inputs[name].(map[string]any); ok {
						if required, ok := input["required"].(bool); ok && required {
							if err := f.Set([]string{"on", "workflow_dispatch", "inputs", name, "required"}, false); err != nil {
								return err
							}
						}
					}
				}
				return nil
			})
		},
	}
}
