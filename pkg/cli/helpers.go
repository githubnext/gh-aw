package cli

import (
	"os"
	"strings"

	"github.com/github/gh-aw/pkg/logger"
	"github.com/github/gh-aw/pkg/parser"
)

var helpersLog = logger.New("cli:helpers")

// getParentDir returns the directory part of a path
func getParentDir(path string) string {
	idx := strings.LastIndex(path, "/")
	if idx == -1 {
		return ""
	}
	return path[:idx]
}

// readSourceRepoFromFile reads the 'source' frontmatter field from a local workflow file
// and returns the "owner/repo" portion (e.g. "github/gh-aw"). Returns "" if the file
// cannot be read, has no source field, or the field is not in the expected format.
func readSourceRepoFromFile(path string) string {
	helpersLog.Printf("Reading source repo from file: %s", path)
	content, err := os.ReadFile(path)
	if err != nil {
		helpersLog.Printf("Failed to read file: %s", err)
		return ""
	}
	result, err := parser.ExtractFrontmatterFromContent(string(content))
	if err != nil || result.Frontmatter == nil {
		return ""
	}
	sourceRaw, ok := result.Frontmatter["source"]
	if !ok {
		return ""
	}
	source, ok := sourceRaw.(string)
	if !ok || source == "" {
		return ""
	}
	// source format: "owner/repo/path/to/file.md@ref" — extract just "owner/repo"
	slashParts := strings.SplitN(source, "/", 3)
	if len(slashParts) < 2 {
		return ""
	}
	repo := slashParts[0] + "/" + slashParts[1]
	helpersLog.Printf("Extracted source repo: %s", repo)
	return repo
}

// sourceRepoLabel returns the source repo string for display in error messages.
// When the repo string is empty (file has no source field or is not a markdown file),
// a human-readable placeholder is returned so the error message is not confusing.
func sourceRepoLabel(repo string) string {
	if repo == "" {
		return "(no source field)"
	}
	return repo
}
