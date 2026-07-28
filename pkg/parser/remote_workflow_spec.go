//go:build !js && !wasm

package parser

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/github/gh-aw/pkg/gitutil"
)

var (
	downloadFileFromGitHubFunc          = downloadFileFromGitHub
	downloadFileFromGitHubWithDepthFunc = downloadFileFromGitHubWithDepth
	resolveRefToSHAFunc                 = resolveRefToSHA
)

// IsWorkflowSpec checks if a path looks like a workflowspec (owner/repo/path[@ref]).
func IsWorkflowSpec(path string) bool {
	// Remove section reference if present
	cleanPath := path
	if before, _, ok := strings.Cut(path, "#"); ok {
		cleanPath = before
	}

	// Remove ref if present
	if idx := strings.Index(cleanPath, "@"); idx != -1 {
		cleanPath = cleanPath[:idx]
	}

	// Check if it has at least 3 parts (owner/repo/path)
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

	// Reject paths that start with "." (local paths like .github/workflows/...)
	if strings.HasPrefix(cleanPath, ".") {
		return false
	}

	// Reject paths that start with "shared/" (local shared files)
	if strings.HasPrefix(cleanPath, "shared/") {
		return false
	}

	// Reject absolute paths
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

// downloadIncludeFromWorkflowSpec downloads an include file from GitHub using workflowspec.
// It first checks the cache, and only downloads if not cached.
func downloadIncludeFromWorkflowSpec(ctx context.Context, spec string, cache *ImportCache) (string, error) {
	remoteLog.Printf("Downloading from workflowspec: %s", spec)
	host, owner, repo, filePath, ref, err := parseWorkflowSpecParts(spec)
	if err != nil {
		return "", err
	}
	remoteLog.Printf("Parsed workflowspec: host=%s, owner=%s, repo=%s, file=%s, ref=%s", host, owner, repo, filePath, ref)

	sha := resolveWorkflowSpecSHAForCache(ctx, owner, repo, ref, host, cache)
	if cache != nil && sha != "" {
		if cachedPath, found := cache.Get(owner, repo, filePath, sha); found {
			remoteLog.Printf("Using cached import: %s/%s/%s@%s (SHA: %s)", owner, repo, filePath, ref, sha)
			return cachedPath, nil
		}
	}

	remoteLog.Printf("Fetching file from GitHub: %s/%s/%s@%s", owner, repo, filePath, ref)
	var content []byte
	if host == "" {
		content, err = downloadFileFromGitHubFunc(ctx, owner, repo, filePath, ref)
	} else {
		content, err = downloadFileFromGitHubWithDepthFunc(ctx, owner, repo, filePath, ref, 0, host)
	}
	if err != nil {
		return "", fmt.Errorf("failed to download include from %s: %w", spec, err)
	}
	remoteLog.Printf("Successfully downloaded file: size=%d bytes", len(content))

	if cache != nil && sha != "" {
		cachedPath, err := cache.Set(owner, repo, filePath, sha, content)
		if err != nil {
			remoteLog.Printf("Failed to cache import: %v", err)
		} else {
			remoteLog.Printf("Successfully cached download at: %s", cachedPath)
			return cachedPath, nil
		}
	}
	return writeDownloadedIncludeToTempFile(content)
}

func parseWorkflowSpecParts(spec string) (string, string, string, string, string, error) {
	cleanSpec := spec
	if before, _, ok := strings.Cut(spec, "#"); ok {
		cleanSpec = before
	}
	parts := strings.SplitN(cleanSpec, "@", 2)
	pathPart := parts[0]
	ref := "main"
	if len(parts) == 2 {
		ref = parts[1]
	} else {
		remoteLog.Print("No ref specified, defaulting to 'main'")
	}

	if err := gitutil.ValidateGitRef(ref); err != nil {
		return "", "", "", "", "", fmt.Errorf("invalid workflowspec ref: %w", err)
	}

	slashParts := strings.Split(pathPart, "/")
	if len(slashParts) < 3 {
		remoteLog.Printf("Invalid workflowspec format: %s", spec)
		return "", "", "", "", "", errors.New("invalid workflowspec: must be owner/repo/path[@ref]")
	}

	// Optional host-prefixed format: host/owner/repo/path[@ref]
	if len(slashParts) >= 4 && strings.Contains(slashParts[0], ".") {
		filePath := strings.Join(slashParts[3:], "/")
		if err := gitutil.ValidateGitPath(filePath); err != nil {
			return "", "", "", "", "", fmt.Errorf("invalid workflowspec path: %w", err)
		}
		return slashParts[0], slashParts[1], slashParts[2], filePath, ref, nil
	}

	filePath := strings.Join(slashParts[2:], "/")
	if err := gitutil.ValidateGitPath(filePath); err != nil {
		return "", "", "", "", "", fmt.Errorf("invalid workflowspec path: %w", err)
	}
	return "", slashParts[0], slashParts[1], filePath, ref, nil
}

func resolveWorkflowSpecSHAForCache(ctx context.Context, owner, repo, ref, host string, cache *ImportCache) string {
	if cache == nil {
		return ""
	}
	resolvedSHA, err := resolveRefToSHAFunc(ctx, owner, repo, ref, host)
	if err != nil {
		remoteLog.Printf("Failed to resolve ref to SHA, will skip cache: %v", err)
		return ""
	}
	return resolvedSHA
}

func writeDownloadedIncludeToTempFile(content []byte) (string, error) {
	tempFile, err := os.CreateTemp("", "gh-aw-include-*.md")
	if err != nil {
		return "", fmt.Errorf("failed to create temp file: %w", err)
	}
	cleanupOnError := true
	fileClosed := false
	defer func() {
		if cleanupOnError {
			if !fileClosed {
				if closeErr := tempFile.Close(); closeErr != nil {
					remoteLog.Printf("Warning: failed to close temp file during deferred cleanup: %v", closeErr)
				}
			}
			if rmErr := os.Remove(tempFile.Name()); rmErr != nil && !os.IsNotExist(rmErr) {
				remoteLog.Printf("Warning: failed to remove temp file %s: %v", tempFile.Name(), rmErr)
			}
		}
	}()
	if _, err := tempFile.Write(content); err != nil {
		if closeErr := tempFile.Close(); closeErr != nil {
			remoteLog.Printf("Warning: failed to close temp file during cleanup: %v", closeErr)
		}
		fileClosed = true
		return "", fmt.Errorf("failed to write temp file: %w", err)
	}
	if err := tempFile.Close(); err != nil {
		fileClosed = true
		return "", fmt.Errorf("failed to close temp file: %w", err)
	}
	cleanupOnError = false
	fileClosed = true
	return tempFile.Name(), nil
}
