// Package parser provides functions for parsing and processing workflow markdown files.
// import_remote.go handles remote import origin tracking and queue item types for
// resolving imports fetched from remote GitHub repositories via the workflowspec format.
package parser

import (
	"path"
	"strings"
)

// remoteImportOrigin tracks the remote repository context for an imported file.
// When a file is fetched from a remote GitHub repository via workflowspec,
// its nested relative imports must be resolved against the same remote repo.
type remoteImportOrigin struct {
	Owner    string // Repository owner (e.g., "elastic")
	Repo     string // Repository name (e.g., "ai-github-actions")
	Ref      string // Git ref - branch, tag, or SHA (e.g., "main", "v1.0.0", "abc123...")
	BasePath string // Base directory path within the repo (e.g., "gh-agent-workflows" for gh-agent-workflows/gh-aw-workflows/file.md)
}

// importQueueItem represents a file to be imported with its context
type importQueueItem struct {
	importPath   string              // Original import path (e.g., "file.md" or "file.md#Section")
	fullPath     string              // Resolved absolute file path
	sectionName  string              // Optional section name (from file.md#Section syntax)
	baseDir      string              // Base directory for resolving nested imports
	inputs       map[string]any      // Optional input values from parent import
	remoteOrigin *remoteImportOrigin // Remote origin context (non-nil when imported from a remote repo)
}

// parseRemoteOrigin extracts the remote origin (owner, repo, ref, basePath) from a workflowspec path.
// Returns nil if the path is not a valid workflowspec.
// Format: owner/repo/path[@ref] where ref defaults to "main" if not specified.
// BasePath is derived from the parent workflowspec path and used for resolving nested relative imports.
// For example, "elastic/ai-github-actions/gh-agent-workflows/gh-aw-workflows/file.md@main"
// produces BasePath="gh-agent-workflows" so nested imports resolve relative to that directory.
func parseRemoteOrigin(spec string) *remoteImportOrigin {
	// Remove section reference if present
	cleanSpec := spec
	if before, _, ok := strings.Cut(spec, "#"); ok {
		cleanSpec = before
	}

	// Split on @ to get path and ref
	parts := strings.SplitN(cleanSpec, "@", 2)
	pathPart := parts[0]
	ref := "main"
	if len(parts) == 2 {
		ref = parts[1]
	}

	// Parse path: owner/repo/path/to/file.md
	slashParts := strings.Split(pathPart, "/")
	if len(slashParts) < 3 {
		return nil
	}

	// Derive BasePath: everything between owner/repo and the last component (filename)
	// Since imports are always 2-level (dir/file.md), the base is everything before the filename
	// Examples:
	// - "owner/repo/.github/workflows/file.md" -> BasePath = ".github/workflows"
	// - "owner/repo/gh-agent-workflows/gh-aw-workflows/file.md" -> BasePath = "gh-agent-workflows/gh-aw-workflows"
	// - "owner/repo/a/b/c/d/file.md" -> BasePath = "a/b/c/d"
	var basePath string
	repoRelativeParts := slashParts[2:] // Everything after owner/repo
	if len(repoRelativeParts) >= 2 {
		// Take everything except the last component (the file itself)
		// For nested imports, we want the directory containing the file
		baseDirParts := repoRelativeParts[:len(repoRelativeParts)-1]
		if len(baseDirParts) > 0 {
			// Clean the path to normalize it (remove ./ and resolve ..)
			basePath = path.Clean(strings.Join(baseDirParts, "/"))
			importLog.Printf("Derived BasePath=%q from spec=%q (owner=%s, repo=%s, ref=%s)",
				basePath, spec, slashParts[0], slashParts[1], ref)
		}
	} else {
		importLog.Printf("No BasePath derived from spec=%q (file at repo root)", spec)
	}

	return &remoteImportOrigin{
		Owner:    slashParts[0],
		Repo:     slashParts[1],
		Ref:      ref,
		BasePath: basePath,
	}
}
