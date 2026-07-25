package cli

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/github/gh-aw/pkg/gitutil"
	"github.com/github/gh-aw/pkg/logger"
	"github.com/github/gh-aw/pkg/parser"
	"github.com/github/gh-aw/pkg/workflow"
)

var updateValidationLog = logger.New("cli:update_validation")

var sha256DigestPattern = regexp.MustCompile(`^sha256:[a-f0-9]{64}$`)

// validationResolvers holds injectable resolver functions for testability.
// Using a struct instead of package-level vars allows callers to supply their own
// resolvers inline without touching globals, enabling t.Parallel() in tests.
type validationResolvers struct {
	// verifyActionCommitExists checks that a commit SHA actually exists in the
	// action repository. It must return parser.ErrVerificationSkipped (wrapped)
	// for auth/network failures so callers can treat those as non-fatal.
	verifyActionCommitExists func(ctx context.Context, repo, sha string) error
	// resolveActionVersionToSHA resolves a version tag/ref to its commit SHA.
	// Used to verify that the stored SHA matches the pinned version.
	resolveActionVersionToSHA func(ctx context.Context, repo, ref string) (string, error)
	// resolveContainerDigest resolves the current digest for a mutable container
	// image reference so the stored pin can be checked against the live registry.
	resolveContainerDigest func(ctx context.Context, image string) (string, error)
}

func defaultValidationResolvers() validationResolvers {
	return validationResolvers{
		verifyActionCommitExists: func(ctx context.Context, repo, sha string) error {
			baseRepo := gitutil.ExtractBaseRepo(repo)
			owner, name, ok := strings.Cut(baseRepo, "/")
			if !ok || owner == "" || name == "" {
				return fmt.Errorf("invalid action repository %q", repo)
			}
			return parser.VerifyCommitExists(ctx, owner, name, sha, "")
		},
		resolveActionVersionToSHA: func(ctx context.Context, repo, ref string) (string, error) {
			baseRepo := gitutil.ExtractBaseRepo(repo)
			owner, name, ok := strings.Cut(baseRepo, "/")
			if !ok || owner == "" || name == "" {
				return "", fmt.Errorf("invalid action repository %q", repo)
			}
			return parser.ResolveRefToSHAForHost(ctx, owner, name, ref, "")
		},
		resolveContainerDigest: func(ctx context.Context, image string) (string, error) {
			return fetchContainerDigest(ctx, image, false)
		},
	}
}

// validateUpdateSHAEntries validates the structural and liveness integrity of
// .github/aw/actions-lock.json after a workflow update. Action SHAs and
// container digests are both re-resolved when possible so malformed or stale
// pins are caught before the updated lock file is kept.
func validateUpdateSHAEntries(ctx context.Context, repoRoot string) error {
	return validateUpdateSHAEntriesWithResolvers(ctx, repoRoot, defaultValidationResolvers())
}

// validateUpdateSHAEntriesStructural validates only the structural integrity of
// .github/aw/actions-lock.json (format, non-empty fields, key consistency) without
// performing any live network calls to resolve or verify SHAs. This is suitable for
// a fast pre-gate check during compilation.
func validateUpdateSHAEntriesStructural(ctx context.Context, repoRoot string) error {
	return validateUpdateSHAEntriesWithResolvers(ctx, repoRoot, structuralOnlyResolvers())
}

// structuralOnlyResolvers returns resolvers that skip all live network checks.
// Used for structural-only validation (format, key consistency) without API calls.
func structuralOnlyResolvers() validationResolvers {
	return validationResolvers{
		verifyActionCommitExists: func(_ context.Context, _, _ string) error {
			return nil
		},
		resolveActionVersionToSHA: func(_ context.Context, _, _ string) (string, error) {
			// Return ErrVerificationSkipped so the caller treats this as non-fatal and
			// skips the version→SHA round-trip check entirely. Structural-only mode
			// validates format and key consistency without live API calls.
			return "", fmt.Errorf("%w: structural-only mode", parser.ErrVerificationSkipped)
		},
		resolveContainerDigest: func(_ context.Context, _ string) (string, error) {
			return "", fmt.Errorf("%w: structural-only mode", parser.ErrVerificationSkipped)
		},
	}
}

func validateUpdateSHAEntriesWithResolvers(ctx context.Context, repoRoot string, r validationResolvers) error {
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
		validSHA := false
		if entry.SHA == "" {
			issues = append(issues, fmt.Sprintf("action entry %q has empty SHA", key))
		} else if !IsCommitSHA(entry.SHA) {
			issues = append(issues, fmt.Sprintf("action entry %q has invalid SHA %q (expected 40-character commit SHA)", key, entry.SHA))
		} else {
			validSHA = true
			// Verify the commit SHA actually exists in the repository via the
			// GitHub commits API. Auth/network failures are non-fatal and logged;
			// only a definitive not-found (e.g. HTTP 422/404) is an error.
			if entry.Repo != "" {
				if err := r.verifyActionCommitExists(ctx, entry.Repo, entry.SHA); err != nil {
					if errors.Is(err, parser.ErrVerificationSkipped) {
						updateValidationLog.Printf("action entry %q: skipping commit existence check (auth/network error): %v", key, err)
					} else {
						issues = append(issues, fmt.Sprintf("action entry %q: commit SHA %q not found in %q: %v", key, entry.SHA, entry.Repo, err))
					}
				}
			}
		}
		if entry.Repo != "" && entry.Version != "" {
			expectedKey := entry.Repo + "@" + entry.Version
			if key != expectedKey {
				issues = append(issues, fmt.Sprintf("action entry key/version mismatch: key %q should be %q", key, expectedKey))
			}
			if validSHA {
				// Verify the stored version tag resolves to the stored SHA.
				// Auth/network failures are non-fatal; only a confirmed mismatch is an error.
				resolvedVersionSHA, err := r.resolveActionVersionToSHA(ctx, entry.Repo, entry.Version)
				if err != nil {
					updateValidationLog.Printf("action entry %q: skipping version/SHA check (resolution failed): %v", key, err)
				} else if !strings.EqualFold(resolvedVersionSHA, entry.SHA) {
					issues = append(issues, fmt.Sprintf("action entry %q SHA/version mismatch: version %q resolves to %q but stored SHA is %q", key, entry.Version, resolvedVersionSHA, entry.SHA))
				}
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
		validDigest := sha256DigestPattern.MatchString(pin.Digest)
		if pin.Image != image {
			issues = append(issues, fmt.Sprintf("container pin key/image mismatch: key %q has image %q", image, pin.Image))
		}
		if !validDigest {
			issues = append(issues, fmt.Sprintf("container pin %q has invalid digest %q (expected sha256:<64 lowercase hex chars>)", image, pin.Digest))
		}
		expectedPinnedImage := image + "@" + pin.Digest
		if pin.PinnedImage != expectedPinnedImage {
			issues = append(issues, fmt.Sprintf("container pin %q has inconsistent pinned_image %q (expected %q)", image, pin.PinnedImage, expectedPinnedImage))
		}
		if validDigest && pin.Image == image {
			resolvedDigest, err := r.resolveContainerDigest(ctx, image)
			if err != nil {
				updateValidationLog.Printf("container pin %q: skipping digest check (resolution failed): %v", image, err)
			} else if !strings.EqualFold(resolvedDigest, pin.Digest) {
				issues = append(issues, fmt.Sprintf("container pin %q digest mismatch: image resolves to %q but stored digest is %q", image, resolvedDigest, pin.Digest))
			}
		}
	}

	if len(issues) > 0 {
		return fmt.Errorf("actions-lock.json validation failed:\n  - %s", strings.Join(issues, "\n  - "))
	}
	return nil
}
