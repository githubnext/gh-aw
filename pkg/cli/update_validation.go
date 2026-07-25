package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/github/gh-aw/pkg/workflow"
)

var sha256DigestPattern = regexp.MustCompile(`^sha256:[a-f0-9]{64}$`)

func validateUpdateSHAEntries(repoRoot string) error {
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
		}
		if entry.Repo != "" && entry.Version != "" {
			expectedKey := entry.Repo + "@" + entry.Version
			if key != expectedKey {
				issues = append(issues, fmt.Sprintf("action entry key/version mismatch: key %q should be %q", key, expectedKey))
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
