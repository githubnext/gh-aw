//go:build !integration

package workflow

import (
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/github/gh-aw/pkg/parser"
	"github.com/stretchr/testify/require"
)

func TestPromptDiagnosticNames(t *testing.T) {
	t.Parallel()
	root, err := findRepoRoot()
	require.NoError(t, err)
	legacy := regexp.MustCompile(`\bmissing_(tool|data)\b`)
	for _, dir := range []string{"actions/setup/md", "pkg/workflow/prompts", ".github/workflows"} {
		t.Run(dir, func(t *testing.T) {
			err := filepath.WalkDir(filepath.Join(root, dir), func(path string, entry fs.DirEntry, walkErr error) error {
				if walkErr != nil {
					return walkErr
				}
				if entry.IsDir() || filepath.Ext(path) != ".md" {
					return nil
				}
				content, readErr := os.ReadFile(path)
				if readErr != nil {
					return readErr
				}
				prompt := string(content)
				if dir == ".github/workflows" {
					// Failure categories and upstream-owned prompts retain their own contracts.
					if entry.Name() == "example-failure-category-filter.md" {
						return nil
					}
					parsed, parseErr := parser.ExtractFrontmatterFromContent(prompt)
					if parseErr != nil {
						return parseErr
					}
					if _, upstream := parsed.Frontmatter["source"]; upstream {
						return nil
					}
					prompt = parsed.Markdown
				}
				for index, line := range strings.Split(prompt, "\n") {
					if legacy.MatchString(line) {
						t.Errorf("%s:%d: prompt references a legacy diagnostic tool: %s", path, index+1, line)
					}
				}
				return nil
			})
			require.NoError(t, err)
		})
	}
}
