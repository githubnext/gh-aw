package stringutil

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/github/gh-aw/pkg/constants"
	"github.com/github/gh-aw/pkg/logger"
)

var identifiersLog = logger.New("stringutil:identifiers")

// NormalizeWorkflowName removes .md, .lock.yml, and .lock.yaml extensions from workflow names.
// This is used to standardize workflow identifiers regardless of the file format.
//
// The function checks for extensions in order of specificity:
// 1. Removes .lock.yaml extension (the compiled workflow format, .yaml variant)
// 2. Removes .lock.yml extension (the compiled workflow format)
// 3. Removes .md extension (the markdown source format)
// 4. Returns the name unchanged if no recognized extension is found
//
// This function performs normalization only - it assumes the input is already
// a valid identifier and does NOT perform character validation or sanitization.
//
// Examples:
//
//	NormalizeWorkflowName("weekly-research")            // returns "weekly-research"
//	NormalizeWorkflowName("weekly-research.md")         // returns "weekly-research"
//	NormalizeWorkflowName("weekly-research.lock.yml")   // returns "weekly-research"
//	NormalizeWorkflowName("weekly-research.lock.yaml")  // returns "weekly-research"
//	NormalizeWorkflowName("my.workflow.md")             // returns "my.workflow"
func NormalizeWorkflowName(name string) string {
	// Remove .lock.yaml extension first (longer extension, .yaml variant)
	if before, ok := strings.CutSuffix(name, constants.LockExtensionYAML); ok {
		return before
	}

	// Remove .lock.yml extension (longer extension)
	if before, ok := strings.CutSuffix(name, constants.LockExtensionYML); ok {
		return before
	}

	// Remove .md extension
	if before, ok := strings.CutSuffix(name, ".md"); ok {
		return before
	}

	return name
}

// NormalizeSafeOutputIdentifier converts dashes to underscores for safe output identifiers.
// This standardizes identifier format from the user-facing dash-separated format
// to the internal underscore-separated format used in safe outputs configuration.
//
// Both dash-separated and underscore-separated formats are valid inputs.
// This function simply standardizes to the internal representation.
//
// This function performs normalization only - it assumes the input is already
// a valid identifier and does NOT perform character validation or sanitization.
//
// Examples:
//
//	NormalizeSafeOutputIdentifier("create-issue")      // returns "create_issue"
//	NormalizeSafeOutputIdentifier("create_issue")      // returns "create_issue" (unchanged)
//	NormalizeSafeOutputIdentifier("add-comment")       // returns "add_comment"
//	NormalizeSafeOutputIdentifier("update-pr")         // returns "update_pr"
func NormalizeSafeOutputIdentifier(identifier string) string {
	return strings.ReplaceAll(identifier, "-", "_")
}

// MarkdownToLockFile converts a workflow markdown file path to its compiled lock file path.
// This is the standard transformation for agentic workflow files.
//
// The function removes the .md extension and adds .lock.yml extension.
// If the input already has a .lock.yml or .lock.yaml extension, it returns the path unchanged.
//
// Examples:
//
//	MarkdownToLockFile("weekly-research.md")                    // returns "weekly-research.lock.yml"
//	MarkdownToLockFile(".github/workflows/test.md")             // returns ".github/workflows/test.lock.yml"
//	MarkdownToLockFile("workflow.lock.yml")                     // returns "workflow.lock.yml" (unchanged)
//	MarkdownToLockFile("workflow.lock.yaml")                    // returns "workflow.lock.yaml" (unchanged)
//	MarkdownToLockFile("my.workflow.md")                        // returns "my.workflow.lock.yml"
func MarkdownToLockFile(mdPath string) string {
	// If already a lock file, return unchanged
	if strings.HasSuffix(mdPath, constants.LockExtensionYML) || strings.HasSuffix(mdPath, constants.LockExtensionYAML) {
		return mdPath
	}

	cleaned := filepath.Clean(mdPath)
	lockPath := strings.TrimSuffix(cleaned, ".md") + constants.LockExtensionYML
	identifiersLog.Printf("MarkdownToLockFile: %s -> %s", mdPath, lockPath)
	return lockPath
}

// MarkdownToLockFileOnDisk converts a workflow markdown file path to its compiled lock file path,
// preferring an existing .lock.yaml file over the default .lock.yml.
//
// If a .lock.yaml file already exists on disk, that path is returned so that recompilation
// updates the existing file rather than creating a new .lock.yml alongside it.
// Otherwise behaves identically to MarkdownToLockFile.
//
// Examples (assuming workflow.lock.yaml exists on disk):
//
//	MarkdownToLockFileOnDisk("workflow.md")  // returns "workflow.lock.yaml"
//
// Examples (assuming no lock file exists):
//
//	MarkdownToLockFileOnDisk("workflow.md")  // returns "workflow.lock.yml"
func MarkdownToLockFileOnDisk(mdPath string) string {
	// If already a lock file, return unchanged
	if strings.HasSuffix(mdPath, constants.LockExtensionYML) || strings.HasSuffix(mdPath, constants.LockExtensionYAML) {
		return mdPath
	}

	cleaned := filepath.Clean(mdPath)
	base := strings.TrimSuffix(cleaned, ".md")

	// Prefer .lock.yaml if it already exists on disk
	lockYamlPath := base + constants.LockExtensionYAML
	if _, err := os.Stat(lockYamlPath); err == nil {
		identifiersLog.Printf("MarkdownToLockFileOnDisk: found existing .lock.yaml: %s -> %s", mdPath, lockYamlPath)
		return lockYamlPath
	}

	lockPath := base + constants.LockExtensionYML
	identifiersLog.Printf("MarkdownToLockFileOnDisk: %s -> %s", mdPath, lockPath)
	return lockPath
}

// LockFileToMarkdown converts a compiled lock file path back to its markdown source path.
// This is used when navigating from compiled workflows back to source files.
//
// The function removes the .lock.yml or .lock.yaml extension and adds .md extension.
// If the input already has a .md extension, it returns the path unchanged.
//
// Examples:
//
//	LockFileToMarkdown("weekly-research.lock.yml")              // returns "weekly-research.md"
//	LockFileToMarkdown("weekly-research.lock.yaml")             // returns "weekly-research.md"
//	LockFileToMarkdown(".github/workflows/test.lock.yml")       // returns ".github/workflows/test.md"
//	LockFileToMarkdown("workflow.md")                           // returns "workflow.md" (unchanged)
//	LockFileToMarkdown("my.workflow.lock.yml")                  // returns "my.workflow.md"
func LockFileToMarkdown(lockPath string) string {
	// If already a markdown file, return unchanged
	if strings.HasSuffix(lockPath, ".md") {
		return lockPath
	}

	cleaned := filepath.Clean(lockPath)
	var mdPath string
	if before, ok := strings.CutSuffix(cleaned, constants.LockExtensionYAML); ok {
		mdPath = before + ".md"
	} else {
		mdPath = strings.TrimSuffix(cleaned, constants.LockExtensionYML) + ".md"
	}
	identifiersLog.Printf("LockFileToMarkdown: %s -> %s", lockPath, mdPath)
	return mdPath
}

// StripLockExtension removes the .lock.yml or .lock.yaml extension from a lock file path,
// returning just the base path without the lock extension. This is useful when constructing
// related file paths (e.g. ".invalid.yml" files for debugging).
//
// Examples:
//
//	StripLockExtension("workflow.lock.yml")   // returns "workflow"
//	StripLockExtension("workflow.lock.yaml")  // returns "workflow"
//	StripLockExtension("workflow.md")         // returns "workflow.md" (unchanged)
func StripLockExtension(lockPath string) string {
	if before, ok := strings.CutSuffix(lockPath, constants.LockExtensionYAML); ok {
		return before
	}
	return strings.TrimSuffix(lockPath, constants.LockExtensionYML)
}
