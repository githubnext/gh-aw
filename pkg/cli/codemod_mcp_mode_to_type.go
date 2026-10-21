package cli

import (
	"maps"
	"slices"

	"github.com/github/gh-aw/pkg/workflow/codemod"
)

// getMCPModeToTypeCodemod migrates mode to type in custom MCP server configurations.
func getMCPModeToTypeCodemod() Codemod {
	return Codemod{
		ID:           "mcp-mode-to-type-migration",
		Name:         "Migrate MCP 'mode' to 'type'",
		Description:  "Renames 'mode' field to 'type' in custom MCP server configurations (mcp-servers section). Does not affect GitHub or Serena tool configurations.",
		IntroducedIn: "0.7.0",
		Apply: func(content string, _ map[string]any) (string, bool, error) {
			return codemod.Apply(content, func(f *codemod.Frontmatter) error {
				value, _ := f.Lookup("mcp-servers")
				servers, ok := value.(map[string]any)
				if !ok {
					return nil
				}
				for _, name := range slices.Sorted(maps.Keys(servers)) {
					if err := f.Rename([]string{"mcp-servers", name, "mode"}, "type"); err != nil {
						return err
					}
				}
				return nil
			})
		},
	}
}
