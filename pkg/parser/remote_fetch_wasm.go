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
	for !strings.HasSuffix(githubFolder, ".github") && githubFolder != "." && githubFolder != "/" {
		githubFolder = filepath.Dir(githubFolder)
		if githubFolder == "." || githubFolder == "/" {
			githubFolder = baseDir
			break
		}
	}

	resolveBase := baseDir
	securityBase := githubFolder
	if strings.HasSuffix(githubFolder, ".github") {
		repoRoot := filepath.Dir(githubFolder)
		if strings.HasPrefix(filePath, ".github/") {
			resolveBase = repoRoot
		} else if stripped, ok := strings.CutPrefix(filePath, "/"); ok {
			filePath = stripped
			resolveBase = repoRoot
			securityBase = repoRoot
		}
	}

	fullPath := filepath.Join(resolveBase, filePath)

	normalizedSecurityBase := filepath.Clean(securityBase)
	normalizedFullPath := filepath.Clean(fullPath)

	relativePath, err := filepath.Rel(normalizedSecurityBase, normalizedFullPath)
	if err != nil || relativePath == ".." || strings.HasPrefix(relativePath, ".."+string(filepath.Separator)) || filepath.IsAbs(relativePath) {
		return "", fmt.Errorf("security: path %s must be within .github folder (resolves to: %s)", filePath, relativePath)
	}

	// In wasm builds, check the virtual filesystem first
	if VirtualFileExists(fullPath) {
		return fullPath, nil
	}

	return "", fmt.Errorf("file not found: %s", fullPath)
}

func isWorkflowSpec(path string) bool {
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
	if strings.HasPrefix(cleanPath, ".") {
		return false
	}
	if strings.HasPrefix(cleanPath, "shared/") {
		return false
	}
	if strings.HasPrefix(cleanPath, "/") {
		return false
	}
	return true
}
