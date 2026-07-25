package cli

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/github/gh-aw/pkg/gitutil"
	"github.com/github/gh-aw/pkg/parser"
	"github.com/github/gh-aw/pkg/workflow"
)

var sha256DigestPattern = regexp.MustCompile(`^sha256:[a-f0-9]{64}$`)

var resolveActionRefSHAForValidation = func(ctx context.Context, repo, ref string) (string, error) {
	baseRepo := gitutil.ExtractBaseRepo(repo)
	owner, name, ok := strings.Cut(baseRepo, "/")
	if !ok || owner == "" || name == "" {
		return "", fmt.Errorf("invalid action repository %q", repo)
	}
	return parser.ResolveRefToSHAForHost(ctx, owner, name, ref, "")
}

var resolveContainerDigestForValidation = func(ctx context.Context, image string) (string, error) {
	return fetchContainerDigest(ctx, image, false)
}

func validateUpdateSHAEntries(ctx context.Context, repoRoot string) error {
	if ctx == nil {
		ctx = context.Background()
	}

	actionsLockPath := filepath.Join(repoRoot, ".github", "aw", "actions-lock.json")
	if _, err := os.Stat(actionsLockPath); err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("failed to read actions-lock.json metadata: %w", err)
	}

	cache := workflow.NewActionCache(repoRoot)
	if err := cache.Load(); err != nil {
		return fmt.Errorf("failed to load actions-lock.json: %w", err)
	}

	var issues []string

	entryKeys := make([]string, 0, len(cache.Entries))
	for key := range cache.Entries {
		entryKeys = append(entryKeys, key)
	}
	sort.Strings(entryKeys)
	for _, key := range entryKeys {
		entry := cache.Entries[key]
		if entry.Repo == "" {
			issues = append(issues, fmt.Sprintf("action entry %q has empty repo", key))
		}
		if entry.Version == "" {
			issues = append(issues, fmt.Sprintf("action entry %q has empty version", key))
		}
		if entry.SHA == "" {
			issues = append(issues, fmt.Sprintf("action entry %q has empty SHA", key))
		} else if !IsCommitSHA(entry.SHA) {
			issues = append(issues, fmt.Sprintf("action entry %q has invalid SHA %q (expected 40-character commit SHA)", key, entry.SHA))
		} else if entry.Repo != "" {
			resolvedSHA, err := resolveActionRefSHAForValidation(ctx, entry.Repo, entry.SHA)
			if err != nil {
				issues = append(issues, fmt.Sprintf("action entry %q has unresolved commit SHA %q in %q: %v", key, entry.SHA, entry.Repo, err))
			} else if !strings.EqualFold(resolvedSHA, entry.SHA) {
				issues = append(issues, fmt.Sprintf("action entry %q resolved commit SHA mismatch: got %q from %q", key, resolvedSHA, entry.SHA))
			}
		}
		if entry.Repo != "" && entry.Version != "" {
			expectedKey := entry.Repo + "@" + entry.Version
			if key != expectedKey {
				issues = append(issues, fmt.Sprintf("action entry key/version mismatch: key %q should be %q", key, expectedKey))
			}
			resolvedVersionSHA, err := resolveActionRefSHAForValidation(ctx, entry.Repo, entry.Version)
			if err != nil {
				issues = append(issues, fmt.Sprintf("action entry %q has unresolved version %q in %q: %v", key, entry.Version, entry.Repo, err))
			} else if entry.SHA != "" && IsCommitSHA(entry.SHA) && !strings.EqualFold(resolvedVersionSHA, entry.SHA) {
				issues = append(issues, fmt.Sprintf("action entry %q SHA/version mismatch: version %q resolves to %q but entry has %q", key, entry.Version, resolvedVersionSHA, entry.SHA))
			}
		}
	}

	containerKeys := make([]string, 0, len(cache.ContainerPins))
	for image := range cache.ContainerPins {
		containerKeys = append(containerKeys, image)
	}
	sort.Strings(containerKeys)
	for _, image := range containerKeys {
		pin := cache.ContainerPins[image]
		if pin.Image != image {
			issues = append(issues, fmt.Sprintf("container pin key/image mismatch: key %q has image %q", image, pin.Image))
		}
		if !sha256DigestPattern.MatchString(pin.Digest) {
			issues = append(issues, fmt.Sprintf("container pin %q has invalid digest %q (expected sha256:<64 lowercase hex chars>)", image, pin.Digest))
		} else {
			resolvedDigest, err := resolveContainerDigestForValidation(ctx, image)
			if err != nil {
				issues = append(issues, fmt.Sprintf("container pin %q digest could not be resolved for %q: %v", image, image, err))
			} else if resolvedDigest != pin.Digest {
				issues = append(issues, fmt.Sprintf("container pin %q digest/version mismatch: expected %q but resolved %q", image, pin.Digest, resolvedDigest))
			}
		}
		expectedPinnedImage := image + "@" + pin.Digest
		if pin.PinnedImage != expectedPinnedImage {
			issues = append(issues, fmt.Sprintf("container pin %q has inconsistent pinned_image %q (expected %q)", image, pin.PinnedImage, expectedPinnedImage))
		}
	}

	if len(issues) > 0 {
		return fmt.Errorf("actions-lock.json validation failed:\n  - %s", strings.Join(issues, "\n  - "))
	}
	return nil
}
