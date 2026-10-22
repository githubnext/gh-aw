package cli

import "github.com/github/gh-aw/pkg/workflow/codemod"

func getExpandPermissionsShorthandCodemod() Codemod {
	return Codemod{
		ID:           "permissions-read-to-read-all",
		Name:         "Convert invalid permissions shorthand",
		Description:  "Converts 'permissions: read' to 'permissions: read-all' and 'permissions: write' to 'permissions: write-all' as per GitHub Actions spec",
		IntroducedIn: "0.5.0",
		Apply: func(content string, _ map[string]any) (string, bool, error) {
			return codemod.Apply(content, func(f *codemod.Frontmatter) error {
				value, _ := f.Lookup("permissions")
				permissions, ok := value.(string)
				if ok && (permissions == "read" || permissions == "write") {
					return f.Set([]string{"permissions"}, permissions+"-all")
				}
				return nil
			})
		},
	}
}
