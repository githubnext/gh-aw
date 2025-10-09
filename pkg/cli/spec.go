package cli

import (
	"fmt"
	"net/url"
	"path/filepath"
	"strings"
)

// RepoSpec represents a parsed repository specification
type RepoSpec struct {
	Repo    string // e.g., "owner/repo"
	Version string // optional version/tag/SHA/branch
}

// SourceSpec represents a parsed source specification from workflow frontmatter
type SourceSpec struct {
	Repo string // e.g., "owner/repo"
	Path string // e.g., "workflows/workflow-name.md"
	Ref  string // optional ref (version/tag/SHA/branch)
}

// WorkflowSpec represents a parsed workflow specification
type WorkflowSpec struct {
	RepoSpec            // embedded RepoSpec for Repo and Version fields
	WorkflowPath string // e.g., "workflows/workflow-name.md"
	WorkflowName string // e.g., "workflow-name"
}

// String returns the canonical string representation of the workflow spec
// in the format "owner/repo/path[@version]"
func (w *WorkflowSpec) String() string {
	spec := w.Repo + "/" + w.WorkflowPath
	if w.Version != "" {
		spec += "@" + w.Version
	}
	return spec
}

// parseRepoSpec parses repository specification like "org/repo@version" or "org/repo@branch" or "org/repo@commit"
func parseRepoSpec(repoSpec string) (*RepoSpec, error) {
	parts := strings.SplitN(repoSpec, "@", 2)
	repo := parts[0]

	// Validate repository format (org/repo)
	repoParts := strings.Split(repo, "/")
	if len(repoParts) != 2 || repoParts[0] == "" || repoParts[1] == "" {
		return nil, fmt.Errorf("repository must be in format 'org/repo'")
	}

	spec := &RepoSpec{
		Repo: repo,
	}

	if len(parts) == 2 {
		spec.Version = parts[1]
	}

	return spec, nil
}

// parseGitHubURL attempts to parse a GitHub URL and extract workflow specification components
// Supports URLs like:
//   - https://github.com/owner/repo/blob/branch/path/to/workflow.md
//   - https://github.com/owner/repo/blob/main/workflows/workflow.md
//   - https://github.com/owner/repo/tree/branch/path/to/workflow.md
//   - https://github.com/owner/repo/raw/branch/path/to/workflow.md
//   - https://raw.githubusercontent.com/owner/repo/refs/heads/branch/path/to/workflow.md
//   - https://raw.githubusercontent.com/owner/repo/COMMIT_SHA/path/to/workflow.md
//   - https://raw.githubusercontent.com/owner/repo/refs/tags/tag/path/to/workflow.md
func parseGitHubURL(spec string) (*WorkflowSpec, error) {
	// Parse the URL
	parsedURL, err := url.Parse(spec)
	if err != nil {
		return nil, fmt.Errorf("invalid URL: %w", err)
	}

	// Check for raw.githubusercontent.com URLs
	if parsedURL.Host == "raw.githubusercontent.com" {
		return parseRawGitHubURL(parsedURL)
	}

	// Must be a GitHub URL
	if parsedURL.Host != "github.com" {
		return nil, fmt.Errorf("URL must be from github.com or raw.githubusercontent.com")
	}

	// Parse the path: /owner/repo/{blob|tree|raw}/ref/path/to/file
	pathParts := strings.Split(strings.Trim(parsedURL.Path, "/"), "/")

	// Need at least: owner, repo, type (blob/tree/raw), ref, and filename
	if len(pathParts) < 5 {
		return nil, fmt.Errorf("invalid GitHub URL format: path too short")
	}

	owner := pathParts[0]
	repo := pathParts[1]
	urlType := pathParts[2] // blob, tree, or raw
	ref := pathParts[3]     // branch name, tag, or commit SHA
	filePath := strings.Join(pathParts[4:], "/")

	// Validate URL type
	if urlType != "blob" && urlType != "tree" && urlType != "raw" {
		return nil, fmt.Errorf("invalid GitHub URL format: expected /blob/, /tree/, or /raw/, got /%s/", urlType)
	}

	// Ensure the file path ends with .md
	if !strings.HasSuffix(filePath, ".md") {
		return nil, fmt.Errorf("GitHub URL must point to a .md file")
	}

	// Validate owner and repo
	if owner == "" || repo == "" {
		return nil, fmt.Errorf("invalid GitHub URL: owner and repo cannot be empty")
	}

	if !isValidGitHubIdentifier(owner) || !isValidGitHubIdentifier(repo) {
		return nil, fmt.Errorf("invalid GitHub URL: '%s/%s' does not look like a valid GitHub repository", owner, repo)
	}

	return &WorkflowSpec{
		RepoSpec: RepoSpec{
			Repo:    fmt.Sprintf("%s/%s", owner, repo),
			Version: ref,
		},
		WorkflowPath: filePath,
		WorkflowName: strings.TrimSuffix(filepath.Base(filePath), ".md"),
	}, nil
}

// parseRawGitHubURL parses raw.githubusercontent.com URLs
// Supports URLs like:
//   - https://raw.githubusercontent.com/owner/repo/refs/heads/branch/path/to/workflow.md
//   - https://raw.githubusercontent.com/owner/repo/COMMIT_SHA/path/to/workflow.md
//   - https://raw.githubusercontent.com/owner/repo/refs/tags/tag/path/to/workflow.md
func parseRawGitHubURL(parsedURL *url.URL) (*WorkflowSpec, error) {
	// Parse the path: /owner/repo/ref-or-sha/path/to/file
	// or /owner/repo/refs/heads/branch/path/to/file
	// or /owner/repo/refs/tags/tag/path/to/file
	pathParts := strings.Split(strings.Trim(parsedURL.Path, "/"), "/")

	// Need at least: owner, repo, ref-or-sha, and filename
	if len(pathParts) < 4 {
		return nil, fmt.Errorf("invalid raw.githubusercontent.com URL format: path too short")
	}

	owner := pathParts[0]
	repo := pathParts[1]

	// Determine the reference and file path based on the third part
	var ref string
	var filePath string

	if pathParts[2] == "refs" {
		// Format: /owner/repo/refs/heads/branch/path/to/file
		// or /owner/repo/refs/tags/tag/path/to/file
		if len(pathParts) < 5 {
			return nil, fmt.Errorf("invalid raw.githubusercontent.com URL format: refs path too short")
		}
		// pathParts[3] is "heads" or "tags"
		ref = pathParts[4] // branch or tag name
		filePath = strings.Join(pathParts[5:], "/")
	} else {
		// Format: /owner/repo/COMMIT_SHA/path/to/file or /owner/repo/branch/path/to/file
		ref = pathParts[2]
		filePath = strings.Join(pathParts[3:], "/")
	}

	// Ensure the file path ends with .md
	if !strings.HasSuffix(filePath, ".md") {
		return nil, fmt.Errorf("raw.githubusercontent.com URL must point to a .md file")
	}

	// Validate owner and repo
	if owner == "" || repo == "" {
		return nil, fmt.Errorf("invalid raw.githubusercontent.com URL: owner and repo cannot be empty")
	}

	if !isValidGitHubIdentifier(owner) || !isValidGitHubIdentifier(repo) {
		return nil, fmt.Errorf("invalid raw.githubusercontent.com URL: '%s/%s' does not look like a valid GitHub repository", owner, repo)
	}

	return &WorkflowSpec{
		RepoSpec: RepoSpec{
			Repo:    fmt.Sprintf("%s/%s", owner, repo),
			Version: ref,
		},
		WorkflowPath: filePath,
		WorkflowName: strings.TrimSuffix(filepath.Base(filePath), ".md"),
	}, nil
}

// parseWorkflowSpec parses a workflow specification in the new format
// Format: owner/repo/workflows/workflow-name[@version] or owner/repo/workflow-name[@version]
// Also supports full GitHub URLs like https://github.com/owner/repo/blob/branch/path/to/workflow.md
func parseWorkflowSpec(spec string) (*WorkflowSpec, error) {
	// Check if this is a GitHub URL
	if strings.HasPrefix(spec, "http://") || strings.HasPrefix(spec, "https://") {
		return parseGitHubURL(spec)
	}

	// Handle version first (anything after @)
	parts := strings.SplitN(spec, "@", 2)
	specWithoutVersion := parts[0]
	var version string
	if len(parts) == 2 {
		version = parts[1]
	}

	// Split by slashes
	slashParts := strings.Split(specWithoutVersion, "/")

	// Must have at least 3 parts: owner/repo/workflow-path
	if len(slashParts) < 3 {
		return nil, fmt.Errorf("workflow specification must be in format 'owner/repo/workflow-name[@version]'")
	}

	owner := slashParts[0]
	repo := slashParts[1]

	// Check if this is a /files/REF/ format (e.g., owner/repo/files/main/path.md)
	// This is the format used when copying file paths from GitHub UI
	var workflowPath string
	if len(slashParts) >= 4 && slashParts[2] == "files" {
		// Extract the ref (branch/tag/commit) from slashParts[3]
		ref := slashParts[3]
		// The file path is everything after /files/REF/
		workflowPath = strings.Join(slashParts[4:], "/")

		// If version was not explicitly provided via @, use the ref from /files/REF/
		if version == "" {
			version = ref
		}
	} else {
		// Standard format: owner/repo/path or owner/repo/workflow-name
		workflowPath = strings.Join(slashParts[2:], "/")
	}

	// Validate owner and repo parts are not empty
	if owner == "" || repo == "" {
		return nil, fmt.Errorf("invalid workflow specification: owner and repo cannot be empty")
	}

	// Basic validation that owner and repo look like GitHub identifiers
	if !isValidGitHubIdentifier(owner) || !isValidGitHubIdentifier(repo) {
		return nil, fmt.Errorf("invalid workflow specification: '%s/%s' does not look like a valid GitHub repository", owner, repo)
	}

	// Handle different cases based on the number of path parts
	if len(slashParts) == 3 && !strings.HasSuffix(workflowPath, ".md") {
		// Three-part spec: owner/repo/workflow-name
		// Add "workflows/" prefix
		workflowPath = "workflows/" + workflowPath + ".md"
	} else {
		// Four or more parts: owner/repo/workflows/workflow-name or owner/repo/path/to/workflow-name
		// Require .md extension to be explicit
		if !strings.HasSuffix(workflowPath, ".md") {
			return nil, fmt.Errorf("workflow specification with path must end with '.md' extension: %s", workflowPath)
		}
	}

	return &WorkflowSpec{
		RepoSpec: RepoSpec{
			Repo:    fmt.Sprintf("%s/%s", owner, repo),
			Version: version,
		},
		WorkflowPath: workflowPath,
		WorkflowName: strings.TrimSuffix(filepath.Base(workflowPath), ".md"),
	}, nil
}

// parseSourceSpec parses a source specification like "owner/repo/path@ref"
// This is used for parsing the source field from workflow frontmatter
func parseSourceSpec(source string) (*SourceSpec, error) {
	// Split on @ to separate ref
	parts := strings.SplitN(source, "@", 2)
	pathPart := parts[0]

	// Parse path: owner/repo/path/to/workflow.md
	slashParts := strings.Split(pathPart, "/")
	if len(slashParts) < 3 {
		return nil, fmt.Errorf("invalid source format: must be owner/repo/path[@ref]")
	}

	spec := &SourceSpec{
		Repo: fmt.Sprintf("%s/%s", slashParts[0], slashParts[1]),
		Path: strings.Join(slashParts[2:], "/"),
	}

	if len(parts) == 2 {
		spec.Ref = parts[1]
	}

	return spec, nil
}

// buildSourceString builds the source string in the format owner/repo/path@ref
func buildSourceString(workflow *WorkflowSpec) string {
	if workflow.Repo == "" || workflow.WorkflowPath == "" {
		return ""
	}

	// Format: owner/repo/path@ref (consistent with add command syntax)
	source := workflow.Repo + "/" + workflow.WorkflowPath
	if workflow.Version != "" {
		source += "@" + workflow.Version
	}

	return source
}

// buildSourceStringWithCommitSHA builds the source string with the actual commit SHA
// This is used when adding workflows to include the precise commit that was installed
func buildSourceStringWithCommitSHA(workflow *WorkflowSpec, commitSHA string) string {
	if workflow.Repo == "" || workflow.WorkflowPath == "" {
		return ""
	}

	// Format: owner/repo/path@commitSHA
	source := workflow.Repo + "/" + workflow.WorkflowPath
	if commitSHA != "" {
		source += "@" + commitSHA
	} else if workflow.Version != "" {
		// Fallback to the version if no commit SHA is available
		source += "@" + workflow.Version
	}

	return source
}

// isValidGitHubIdentifier checks if a string looks like a valid GitHub username or repository name
// GitHub allows alphanumeric characters, hyphens, and underscores, but cannot start or end with hyphen
func isValidGitHubIdentifier(identifier string) bool {
	if len(identifier) == 0 {
		return false
	}

	// Cannot start or end with hyphen
	if identifier[0] == '-' || identifier[len(identifier)-1] == '-' {
		return false
	}

	// Must contain only alphanumeric chars, hyphens, and underscores
	for _, char := range identifier {
		if !((char >= 'a' && char <= 'z') ||
			(char >= 'A' && char <= 'Z') ||
			(char >= '0' && char <= '9') ||
			char == '-' || char == '_') {
			return false
		}
	}

	return true
}

// isCommitSHA checks if a version string looks like a commit SHA (40-character hex string)
func isCommitSHA(version string) bool {
	if len(version) != 40 {
		return false
	}
	// Check if all characters are hexadecimal
	for _, char := range version {
		if !((char >= '0' && char <= '9') || (char >= 'a' && char <= 'f') || (char >= 'A' && char <= 'F')) {
			return false
		}
	}
	return true
}
