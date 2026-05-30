//go:build js || wasm

package parser

import (
	"fmt"
	"path/filepath"
	"strings"
)

func isUnderWorkflowsDirectory(filePath string) bool {
	normalizedPath := filepath.ToSlash(filePath)
	if !strings.Contains(normalizedPath, ".github/workflows/") {
		return false
	}
	parts := strings.Split(normalizedPath, ".github/workflows/")
	if len(parts) < 2 {
		return false
	}
	return !strings.Contains(parts[1], "/")
}

func isCustomAgentFile(filePath string) bool {
	normalizedPath := filepath.ToSlash(filePath)
	return strings.Contains(normalizedPath, ".github/agents/") && strings.HasSuffix(strings.ToLower(normalizedPath), ".md")
}

func isRepositoryImport(importPath string) bool {
	cleanPath := importPath
	if idx := strings.Index(importPath, "#"); idx != -1 {
		cleanPath = importPath[:idx]
	}
	pathWithoutRef := cleanPath
	if idx := strings.Index(cleanPath, "@"); idx != -1 {
		pathWithoutRef = cleanPath[:idx]
	}
	parts := strings.Split(pathWithoutRef, "/")
	if len(parts) != 2 {
		return false
	}
	if strings.HasPrefix(pathWithoutRef, ".") || strings.HasPrefix(pathWithoutRef, "/") {
		return false
	}
	if strings.HasPrefix(pathWithoutRef, "shared/") {
		return false
	}
	owner := parts[0]
	repo := parts[1]
	if owner == "" || repo == "" {
		return false
	}
	if strings.Contains(repo, ".") {
		return false
	}
	return true
}

func ResolveIncludePath(filePath, baseDir string, cache *ImportCache) (string, error) {
	// Handle builtin paths - these are embedded files that bypass filesystem resolution.
	if strings.HasPrefix(filePath, BuiltinPathPrefix) {
		if !BuiltinVirtualFileExists(filePath) {
			return "", fmt.Errorf("builtin file not found: %s", filePath)
		}
		return filePath, nil
	}

	if isWorkflowSpec(filePath) {
		return "", fmt.Errorf("remote imports not available in Wasm: %s", filePath)
	}

	githubFolder := baseDir
	for !strings.HasSuffix(githubFolder, ".github") {
		parent := filepath.Dir(githubFolder)
		if parent == githubFolder || parent == "." || parent == "/" {
			githubFolder = baseDir
			break
		}
		githubFolder = parent
	}

	resolveBase := baseDir
	securityBase := githubFolder
	if strings.HasSuffix(githubFolder, ".github") {
		repoRoot := filepath.Dir(githubFolder)
		filePathSlash := filepath.ToSlash(filePath)
		if strings.HasPrefix(filePathSlash, ".github/") {
			resolveBase = repoRoot
		} else if stripped, ok := strings.CutPrefix(filePathSlash, "/"); ok {
			// Repo-root-absolute path: only .github/ and .agents/ subdirectories are accessible.
			if !strings.HasPrefix(stripped, ".github/") && !strings.HasPrefix(stripped, ".agents/") {
				return "", fmt.Errorf("security: path %s must be within .github or .agents folder", filePath)
			}
			filePath = filepath.FromSlash(stripped)
			resolveBase = repoRoot
			if strings.HasPrefix(stripped, ".agents/") {
				securityBase = filepath.Join(repoRoot, ".agents")
			} else {
				// .github/-prefixed: security scope is the .github folder.
				securityBase = githubFolder
			}
		}
	}

	fullPath := filepath.Join(resolveBase, filePath)

	normalizedSecurityBase := filepath.Clean(securityBase)
	normalizedFullPath := filepath.Clean(fullPath)

	relativePath, err := filepath.Rel(normalizedSecurityBase, normalizedFullPath)
	if err != nil || relativePath == ".." || strings.HasPrefix(relativePath, ".."+string(filepath.Separator)) || filepath.IsAbs(relativePath) {
		allowedFolder := filepath.Base(normalizedSecurityBase)
		return "", fmt.Errorf("security: path %s must be within %s folder (resolves to: %s)", filePath, allowedFolder, relativePath)
	}

	// In wasm builds, check the virtual filesystem first
	if VirtualFileExists(fullPath) {
		return fullPath, nil
	}

	return "", fmt.Errorf("file not found: %s", fullPath)
}

// IsWorkflowSpec checks if a path looks like a workflowspec (owner/repo/path[@ref]).
func IsWorkflowSpec(path string) bool {
	cleanPath := path
	if idx := strings.Index(path, "#"); idx != -1 {
		cleanPath = path[:idx]
	}
	if idx := strings.Index(cleanPath, "@"); idx != -1 {
		cleanPath = cleanPath[:idx]
	}
	parts := strings.Split(cleanPath, "/")
	if len(parts) < 3 {
		return false
	}
	// Preserve legacy behavior expected by parser tests: URL-like paths are
	// currently treated as workflowspecs because downstream parsing supports
	// repository/path extraction from slash-delimited remote references.
	if strings.Contains(cleanPath, "://") {
		return true
	}
	if strings.HasPrefix(cleanPath, ".") {
		return false
	}
	if strings.HasPrefix(cleanPath, "shared/") {
		return false
	}
	if strings.HasPrefix(cleanPath, "/") {
		return false
	}
	// Safe indexing: len(parts) >= 3 is guaranteed above.
	owner := parts[0]
	repo := parts[1]
	if owner == "" || repo == "" {
		return false
	}
	return true
}

func isWorkflowSpec(path string) bool {
	return IsWorkflowSpec(path)
}
