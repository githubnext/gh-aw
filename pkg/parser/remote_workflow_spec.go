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

// downloadIncludeFromWorkflowSpec downloads an include file from GitHub using workflowspec.
// It first checks the cache, and only downloads if not cached.
//
// NOTE: This function is called from ResolveIncludePath which has no context.Context
// parameter. Threading ctx through ResolveIncludePath and its 6+ callers across multiple
// packages is tracked as a follow-up task; context.Background() is used in the interim.
func downloadIncludeFromWorkflowSpec(spec string, cache *ImportCache) (string, error) {
	remoteLog.Printf("Downloading from workflowspec: %s", spec)
	host, owner, repo, filePath, ref, err := parseWorkflowSpecParts(spec)
	if err != nil {
		return "", err
	}
	remoteLog.Printf("Parsed workflowspec: host=%s, owner=%s, repo=%s, file=%s, ref=%s", host, owner, repo, filePath, ref)

	sha := resolveWorkflowSpecSHAForCache(owner, repo, ref, host, cache)
	if cache != nil && sha != "" {
		if cachedPath, found := cache.Get(owner, repo, filePath, sha); found {
			remoteLog.Printf("Using cached import: %s/%s/%s@%s (SHA: %s)", owner, repo, filePath, ref, sha)
			return cachedPath, nil
		}
	}

	remoteLog.Printf("Fetching file from GitHub: %s/%s/%s@%s", owner, repo, filePath, ref)
	var content []byte
	if host == "" {
		content, err = downloadFileFromGitHub(context.Background(), owner, repo, filePath, ref)
	} else {
		content, err = downloadFileFromGitHubWithDepth(context.Background(), owner, repo, filePath, ref, 0, host)
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
		host := slashParts[0]
		owner := slashParts[1]
		repo := slashParts[2]
		if !IsGitHubHost(host) {
			return "", "", "", "", "", fmt.Errorf("invalid workflowspec host %q — expected a GitHub host: 'github.com', 'raw.githubusercontent.com', '*.ghe.com' or '*.github.com' (for example: 'github.com/owner/repo/workflows/ci.md@main')", host)
		}
		if !IsValidGitHubIdentifier(owner) || !IsValidGitHubRepositoryName(repo) {
			return "", "", "", "", "", fmt.Errorf("invalid workflowspec repository '%s/%s' — expected 'host/owner/repo/path[@ref]' format (for example: 'github.com/github/gh-aw/workflows/ci.md@main')", owner, repo)
		}
		// Raw content is served from raw.githubusercontent.com but the API and
		// git remotes live on github.com, so normalize it here.
		if host == "raw.githubusercontent.com" {
			host = "github.com"
		}
		filePath := strings.Join(slashParts[3:], "/")
		if err := gitutil.ValidateGitPath(filePath); err != nil {
			return "", "", "", "", "", fmt.Errorf("invalid workflowspec path: %w", err)
		}
		return host, owner, repo, filePath, ref, nil
	}

	filePath := strings.Join(slashParts[2:], "/")
	if err := gitutil.ValidateGitPath(filePath); err != nil {
		return "", "", "", "", "", fmt.Errorf("invalid workflowspec path: %w", err)
	}
	return "", slashParts[0], slashParts[1], filePath, ref, nil
}

func resolveWorkflowSpecSHAForCache(owner, repo, ref, host string, cache *ImportCache) string {
	if cache == nil {
		return ""
	}
	resolvedSHA, err := resolveRefToSHA(context.Background(), owner, repo, ref, host)
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
