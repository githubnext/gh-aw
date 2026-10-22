package cli

import (
	"github.com/github/gh-aw/pkg/logger"
	"github.com/github/gh-aw/pkg/workflow/codemod"
)

var minIntegrityNoneBashCodemodLog = logger.New("cli:codemod_min_integrity_none_bash")

// getMinIntegrityNoneRequiresBashCodemod creates a codemod that adds an explicit
// 'tools.bash: false' when 'tools.github.min-integrity' is set to 'none' and
// 'tools.bash' is not already specified.
//
// Strict mode requires bash access to be explicit whenever min-integrity is none, since
// any external user can trigger the workflow. No bash tool was configured before, so
// inserting 'bash: false' preserves the existing behavior while satisfying the new
// strict-mode requirement.
func getMinIntegrityNoneRequiresBashCodemod() Codemod {
	return Codemod{
		ID:           "min-integrity-none-requires-bash",
		Name:         "Add explicit 'tools.bash: false' when 'tools.github.min-integrity' is 'none'",
		Description:  "Inserts 'tools.bash: false' when 'tools.github.min-integrity' is set to 'none' and 'tools.bash' is not already specified, preserving current behavior while satisfying strict mode",
		IntroducedIn: "1.5.0",
		Apply: func(content string, frontmatter map[string]any) (string, bool, error) {
			toolsMap, ok := frontmatter["tools"].(map[string]any)
			if !ok {
				return content, false, nil
			}

			if _, hasBash := toolsMap["bash"]; hasBash {
				return content, false, nil
			}

			githubMap, ok := toolsMap["github"].(map[string]any)
			if !ok {
				return content, false, nil
			}

			minIntegrity, ok := githubMap["min-integrity"].(string)
			if !ok || minIntegrity != "none" {
				return content, false, nil
			}

			newContent, applied, err := codemod.Apply(content, func(f *codemod.Frontmatter) error {
				return f.Set([]string{"tools", "bash"}, false)
			})
			if applied {
				minIntegrityNoneBashCodemodLog.Print("Inserted 'tools.bash: false' because tools.github.min-integrity is 'none'")
			}
			return newContent, applied, err
		},
	}
}
