package cli

import (
	"github.com/github/gh-aw/pkg/logger"
	"github.com/github/gh-aw/pkg/workflow/codemod"
)

var bashAnonymousCodemodLog = logger.New("cli:codemod_bash_anonymous")

// getBashAnonymousRemovalCodemod creates a codemod for removing anonymous bash tool syntax
func getBashAnonymousRemovalCodemod() Codemod {
	return Codemod{
		ID:           "bash-anonymous-removal",
		Name:         "Replace anonymous bash tool syntax with explicit true",
		Description:  "Replaces 'bash:' (anonymous/nil syntax) with 'bash: true' to make configuration explicit",
		IntroducedIn: "0.9.0",
		Apply: func(content string, frontmatter map[string]any) (string, bool, error) {
			// Check if tools.bash exists
			toolsValue, hasTools := frontmatter["tools"]
			if !hasTools {
				return content, false, nil
			}

			toolsMap, ok := toolsValue.(map[string]any)
			if !ok {
				return content, false, nil
			}

			// Check if bash field exists and is nil
			bashValue, hasBash := toolsMap["bash"]
			if !hasBash {
				return content, false, nil
			}

			// Only modify if bash is nil (anonymous syntax)
			if bashValue != nil {
				return content, false, nil
			}

			newContent, applied, err := codemod.Apply(content, func(f *codemod.Frontmatter) error {
				return f.Set([]string{"tools", "bash"}, true)
			})
			if applied {
				bashAnonymousCodemodLog.Print("Applied bash anonymous removal, replaced with 'bash: true'")
			}
			return newContent, applied, err
		},
	}
}
