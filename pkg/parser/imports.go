package parser

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
)

// ImportSpec represents a single import specification from frontmatter
// Format: "org/repo version path" (go.mod style)
type ImportSpec struct {
	Org     string // GitHub organization
	Repo    string // Repository name
	Version string // Version/tag/branch/commit
	Path    string // Path to the file within the repository
}

// ParseImportSpec parses a single import string into an ImportSpec
// Expected format: "org/repo version path"
// Example: "microsoft/genaiscript v1.5 agentics/engine.md"
func ParseImportSpec(importStr string) (*ImportSpec, error) {
	// Trim whitespace
	importStr = strings.TrimSpace(importStr)
	if importStr == "" {
		return nil, fmt.Errorf("empty import specification")
	}

	// Split by whitespace
	parts := strings.Fields(importStr)
	if len(parts) < 3 {
		return nil, fmt.Errorf("invalid import format '%s': expected 'org/repo version path'", importStr)
	}

	// Parse org/repo
	repoSpec := parts[0]
	if !strings.Contains(repoSpec, "/") {
		return nil, fmt.Errorf("invalid repository format '%s': expected 'org/repo'", repoSpec)
	}
	repoParts := strings.SplitN(repoSpec, "/", 2)
	if len(repoParts) != 2 {
		return nil, fmt.Errorf("invalid repository format '%s': expected 'org/repo'", repoSpec)
	}

	version := parts[1]
	path := strings.Join(parts[2:], " ") // Allow spaces in path

	// Validate version format (basic check)
	if version == "" {
		return nil, fmt.Errorf("empty version in import specification")
	}

	// Validate path
	if path == "" {
		return nil, fmt.Errorf("empty path in import specification")
	}

	return &ImportSpec{
		Org:     repoParts[0],
		Repo:    repoParts[1],
		Version: version,
		Path:    path,
	}, nil
}

// ParseImports parses an array of import strings from frontmatter
func ParseImports(importsValue interface{}) ([]*ImportSpec, error) {
	if importsValue == nil {
		return nil, nil
	}

	// Handle array of strings
	importsArray, ok := importsValue.([]interface{})
	if !ok {
		return nil, fmt.Errorf("imports must be an array of strings")
	}

	var imports []*ImportSpec
	for i, item := range importsArray {
		importStr, ok := item.(string)
		if !ok {
			return nil, fmt.Errorf("import at index %d is not a string", i)
		}

		spec, err := ParseImportSpec(importStr)
		if err != nil {
			return nil, fmt.Errorf("import at index %d: %w", i, err)
		}

		imports = append(imports, spec)
	}

	return imports, nil
}

// String returns the import specification as a string
func (i *ImportSpec) String() string {
	return fmt.Sprintf("%s/%s %s %s", i.Org, i.Repo, i.Version, i.Path)
}

// RepoSlug returns the repository slug (org/repo)
func (i *ImportSpec) RepoSlug() string {
	return fmt.Sprintf("%s/%s", i.Org, i.Repo)
}

// ImportedFilePath returns the expected file path within the imported repository
func (i *ImportSpec) ImportedFilePath() string {
	return i.Path
}

// ImportLockEntry represents an entry in the import lock file
type ImportLockEntry struct {
	ImportSpec *ImportSpec
	CommitSHA  string   // Resolved commit SHA
	ResolvedAt string   // Timestamp when resolved
	Files      []string // List of transitive files included
}

// ImportLockFile represents the complete lock file structure
type ImportLockFile struct {
	Version string             // Lock file format version
	Entries []*ImportLockEntry // Locked import entries
}

// ValidatePath checks if a path looks valid
func (i *ImportSpec) ValidatePath() error {
	// Basic validation
	if strings.HasPrefix(i.Path, "/") {
		return fmt.Errorf("path must be relative, not absolute: %s", i.Path)
	}
	if strings.Contains(i.Path, "..") {
		return fmt.Errorf("path must not contain '..': %s", i.Path)
	}
	return nil
}

// IsVersionTag checks if the version looks like a semver tag
func (i *ImportSpec) IsVersionTag() bool {
	// Check if version starts with 'v' followed by digits
	matched, _ := regexp.MatchString(`^v\d+`, i.Version)
	return matched
}

// IsCommitSHA checks if the version looks like a commit SHA
func (i *ImportSpec) IsCommitSHA() bool {
	// Check if version is a 40-character hex string (full SHA)
	matched, _ := regexp.MatchString(`^[0-9a-f]{40}$`, i.Version)
	return matched
}

// NormalizeVersion normalizes the version string for consistent comparison
func (i *ImportSpec) NormalizeVersion() string {
	// Remove 'refs/tags/' or 'refs/heads/' prefix if present
	version := i.Version
	version = strings.TrimPrefix(version, "refs/tags/")
	version = strings.TrimPrefix(version, "refs/heads/")
	return version
}

// GetLocalCachePath returns the local cache path for this import
func (i *ImportSpec) GetLocalCachePath(importsDir string) string {
	// Store in .aw/imports/org/repo/version/
	return filepath.Join(importsDir, i.Org, i.Repo, i.Version)
}
