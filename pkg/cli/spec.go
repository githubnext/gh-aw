package cli

import (
	"fmt"
	"path/filepath"
	"strings"
)

// WorkflowSpec represents a parsed workflow specification
type WorkflowSpec struct {
	Spec         string // e.g., "owner/repo/workflow@v1"
	Repo         string // e.g., "owner/repo"
	WorkflowPath string // e.g., "workflows/workflow-name.md"
	WorkflowName string // e.g., "workflow-name"
	Version      string // optional version/tag/SHA
}

// parseRepoSpec parses repository specification like "org/repo@version" or "org/repo@branch" or "org/repo@commit"
func parseRepoSpec(repoSpec string) (repo, version string, err error) {
	parts := strings.SplitN(repoSpec, "@", 2)
	repo = parts[0]

	// Validate repository format (org/repo)
	repoParts := strings.Split(repo, "/")
	if len(repoParts) != 2 || repoParts[0] == "" || repoParts[1] == "" {
		return "", "", fmt.Errorf("repository must be in format 'org/repo'")
	}

	if len(parts) == 2 {
		version = parts[1]
	}

	return repo, version, nil
}

// parseWorkflowSpec parses a workflow specification in the new format
// Format: owner/repo/workflows/workflow-name[@version] or owner/repo/workflow-name[@version]
func parseWorkflowSpec(spec string) (*WorkflowSpec, error) {
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
	workflowPath := strings.Join(slashParts[2:], "/")

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
		Spec:         spec,
		Repo:         fmt.Sprintf("%s/%s", owner, repo),
		WorkflowPath: workflowPath,
		WorkflowName: strings.TrimSuffix(filepath.Base(workflowPath), ".md"),
		Version:      version,
	}, nil
}

// parseSourceSpec parses a source specification like "owner/repo/path@ref"
// This is used for parsing the source field from workflow frontmatter
func parseSourceSpec(source string) (repo, path, ref string, err error) {
	// Split on @ to separate ref
	parts := strings.SplitN(source, "@", 2)
	pathPart := parts[0]
	if len(parts) == 2 {
		ref = parts[1]
	}

	// Parse path: owner/repo/path/to/workflow.md
	slashParts := strings.Split(pathPart, "/")
	if len(slashParts) < 3 {
		return "", "", "", fmt.Errorf("invalid source format: must be owner/repo/path[@ref]")
	}

	repo = fmt.Sprintf("%s/%s", slashParts[0], slashParts[1])
	path = strings.Join(slashParts[2:], "/")

	return repo, path, ref, nil
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
