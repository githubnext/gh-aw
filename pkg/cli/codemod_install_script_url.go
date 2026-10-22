package cli

import (
	"regexp"
	"strings"

	"github.com/github/gh-aw/pkg/workflow/codemod"
)

// Include dots in the full repository token: a bare trailing period is ambiguous
// with a repository name, unlike punctuation outside a quoted reference.
var installScriptRepositoryPattern = regexp.MustCompile(`[A-Za-z0-9_.-]+/gh-aw[A-Za-z0-9_.-]*`)

// getInstallScriptURLCodemod creates a codemod for migrating githubnext/gh-aw to github/gh-aw in install script URLs
func getInstallScriptURLCodemod() Codemod {
	return Codemod{
		ID:           "install-script-url-migration",
		Name:         "Migrate install script URL from githubnext/gh-aw to github/gh-aw",
		Description:  "Updates githubnext/gh-aw repository references to github/gh-aw without changing other repository names",
		IntroducedIn: "0.9.0",
		Apply: func(content string, frontmatter map[string]any) (string, bool, error) {
			return codemod.Apply(content, func(f *codemod.Frontmatter) error {
				return f.RewriteText(func(source string) string {
					return installScriptRepositoryPattern.ReplaceAllStringFunc(source, func(repository string) string {
						switch repository {
						case "githubnext/gh-aw", "githubnext/gh-aw.git":
							return strings.Replace(repository, "githubnext/gh-aw", "github/gh-aw", 1)
						default:
							return repository
						}
					})
				})
			})
		},
	}
}
