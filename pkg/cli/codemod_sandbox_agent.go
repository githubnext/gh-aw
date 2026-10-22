package cli

import "github.com/github/gh-aw/pkg/workflow/codemod"

func getSandboxFalseToAgentFalseCodemod() Codemod {
	return Codemod{
		ID:           "sandbox-false-to-agent-false",
		Name:         "Convert sandbox: false to sandbox.agent: false",
		Description:  "Converts top-level 'sandbox: false' to 'sandbox: { agent: false }' as top-level boolean is no longer supported",
		IntroducedIn: "0.10.0",
		Apply: func(content string, _ map[string]any) (string, bool, error) {
			return codemod.Apply(content, func(f *codemod.Frontmatter) error {
				value, _ := f.Lookup("sandbox")
				if disabled, ok := value.(bool); !ok || disabled {
					return nil
				}
				return f.Set([]string{"sandbox"}, map[string]any{"agent": false})
			})
		},
	}
}
