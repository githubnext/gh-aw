package cli

import (
	"github.com/github/gh-aw/pkg/workflow/aw"
	"github.com/github/gh-aw/pkg/workflow/codemod"
)

// These scopes do not accept read access.
var writeOnlyPermissions = map[string]bool{
	"id-token":         true,
	"copilot-requests": true,
}

func getMigrateWritePermissionsToReadCodemod() Codemod {
	return Codemod{
		ID:           "write-permissions-to-read-migration",
		Name:         "Convert write permissions to read",
		Description:  "Converts all write permissions to read permissions to comply with the new security policy",
		IntroducedIn: "0.4.0",
		Apply: func(content string, _ map[string]any) (string, bool, error) {
			return codemod.Apply(content, func(f *codemod.Frontmatter) error {
				return f.Edit(func(document *aw.Document) error {
					switch permissions := document.Frontmatter["permissions"].(type) {
					case string:
						switch permissions {
						case "write":
							document.Frontmatter["permissions"] = "read"
						case "write-all":
							document.Frontmatter["permissions"] = "read-all"
						}
					case map[string]any:
						for key, value := range permissions {
							if value == "write" && !writeOnlyPermissions[key] {
								permissions[key] = "read"
							}
						}
					}
					return nil
				})
			})
		},
	}
}
