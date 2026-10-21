package cli

import (
	"strings"

	"github.com/github/gh-aw/pkg/workflow/codemod"
)

// getRunInstallScriptsToRuntimesNodeCodemod moves the install policy to the Node runtime.
func getRunInstallScriptsToRuntimesNodeCodemod() Codemod {
	return Codemod{
		ID:           "run-install-scripts-to-runtimes-node",
		Name:         "Move run-install-scripts under runtimes.node",
		Description:  "Moves the deprecated top-level 'run-install-scripts' field to 'runtimes.node.run-install-scripts', which is the only runtime that generates npm install commands.",
		IntroducedIn: "1.5.0",
		Apply: func(content string, frontmatter map[string]any) (string, bool, error) {
			value, exists := frontmatter["run-install-scripts"]
			if !exists {
				return content, false, nil
			}
			enabled := true
			switch value := value.(type) {
			case bool:
				enabled = value
			case string:
				enabled = !strings.EqualFold(value, "false")
			}
			return codemod.Apply(content, func(f *codemod.Frontmatter) error {
				if _, exists := f.Lookup("runtimes", "node", "run-install-scripts"); !exists {
					// A scalar Node version is shorthand for node.version.
					if node, exists := f.Lookup("runtimes", "node"); exists {
						if _, ok := node.(map[string]any); !ok {
							nested := map[string]any{}
							if node != nil {
								nested["version"] = node
							}
							if err := f.Set([]string{"runtimes", "node"}, nested); err != nil {
								return err
							}
						}
					}
					destination := []string{"runtimes", "node", "run-install-scripts"}
					if err := f.Move([]string{"run-install-scripts"}, destination); err != nil {
						return err
					}
					if err := f.Set(destination, enabled); err != nil {
						return err
					}
				}
				return f.Remove("run-install-scripts")
			})
		},
	}
}
