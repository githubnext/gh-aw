package cli

import (
	"regexp"
	"strings"

	"github.com/github/gh-aw/pkg/logger"
)

var installScriptURLCodemodLog = logger.New("cli:codemod_install_script_url")

var installScriptRepositoryPattern = regexp.MustCompile(`[A-Za-z0-9_.-]+/gh-aw[A-Za-z0-9_.-]*`)

// getInstallScriptURLCodemod creates a codemod for migrating githubnext/gh-aw to github/gh-aw in install script URLs
func getInstallScriptURLCodemod() Codemod {
	return Codemod{
		ID:           "install-script-url-migration",
		Name:         "Migrate install script URL from githubnext/gh-aw to github/gh-aw",
		Description:  "Updates githubnext/gh-aw repository references to github/gh-aw without changing other repository names",
		IntroducedIn: "0.9.0",
		Apply: func(content string, frontmatter map[string]any) (string, bool, error) {
			return applyFrontmatterLineTransform(content, func(lines []string) ([]string, bool) {
				modified := false
				result := make([]string, 0, len(lines))
				for i, line := range lines {
					modifiedLine := installScriptRepositoryPattern.ReplaceAllStringFunc(line, func(repository string) string {
						switch repository {
						case "githubnext/gh-aw", "githubnext/gh-aw.git":
							return strings.Replace(repository, "githubnext/gh-aw", "github/gh-aw", 1)
						default:
							return repository
						}
					})
					if modifiedLine != line {
						modified = true
						installScriptURLCodemodLog.Printf("Migrated githubnext/gh-aw repository reference on line %d", i+1)
					}
					result = append(result, modifiedLine)
				}
				if modified {
					installScriptURLCodemodLog.Print("Applied install script URL migration")
				}
				return result, modified
			})
		},
	}
}
