package cli

// This file rewrites "uses:" action refs and skill refs found inside Markdown
// workflow content. It is invoked by update_actions_workflow_files.go while
// walking workflow files.

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/github/gh-aw/pkg/console"
	"github.com/github/gh-aw/pkg/gitutil"
	"github.com/github/gh-aw/pkg/parser"
	"github.com/goccy/go-yaml"
)

type skillRefUpdateResolver func(ctx context.Context, repo, currentRef string, allowMajor, verbose bool, coolDown time.Duration) (string, error)

// noObjectKey signals to updateFrontmatterRepoRefsInContentWithResolver that the field
// being updated (e.g. "plugins") does not support the map[string]any object form with a
// nested ref key, so object-form entries are left untouched.
const noObjectKey = ""

func updateSkillRefsInContent(ctx context.Context, content string, allowMajor, verbose bool, coolDown time.Duration) (bool, string, error) {
	return updateSkillRefsInContentWithResolver(ctx, content, allowMajor, verbose, coolDown, resolveLatestRef)
}

func updatePluginRefsInContent(ctx context.Context, content string, allowMajor, verbose bool, coolDown time.Duration) (bool, string, error) {
	return updatePluginRefsInContentWithResolver(ctx, content, allowMajor, verbose, coolDown, resolveLatestRef)
}

func updateSkillRefsInContentWithResolver(
	ctx context.Context,
	content string,
	allowMajor, verbose bool,
	coolDown time.Duration,
	resolver skillRefUpdateResolver,
) (bool, string, error) {
	return updateFrontmatterRepoRefsInContentWithResolver(ctx, content, "skills", "skill", allowMajor, verbose, coolDown, resolver)
}

func updatePluginRefsInContentWithResolver(
	ctx context.Context,
	content string,
	allowMajor, verbose bool,
	coolDown time.Duration,
	resolver skillRefUpdateResolver,
) (bool, string, error) {
	return updateFrontmatterRepoRefsInContentWithResolver(ctx, content, "plugins", noObjectKey, allowMajor, verbose, coolDown, resolver)
}

func updateFrontmatterRepoRefsInContentWithResolver(
	ctx context.Context,
	content string,
	fieldName string,
	objectKey string,
	allowMajor, verbose bool,
	coolDown time.Duration,
	resolver skillRefUpdateResolver,
) (bool, string, error) {
	result, err := parser.ExtractFrontmatterFromContent(content)
	if err != nil {
		if verbose {
			updateLog.Printf("Skipping %s update for content without parseable frontmatter: %v", fieldName, err)
		}
		return false, content, nil
	}
	if result == nil || result.Frontmatter == nil {
		return false, content, nil
	}

	rawRefs, ok := result.Frontmatter[fieldName].([]any)
	if !ok || len(rawRefs) == 0 {
		return false, content, nil
	}

	changed := false
	for i, rawRef := range rawRefs {
		switch typed := rawRef.(type) {
		case string:
			updated, updatedRef, err := updateSkillRefValue(ctx, fieldName, typed, allowMajor, verbose, coolDown, resolver)
			if err != nil {
				return false, content, err
			}
			if updated {
				rawRefs[i] = updatedRef
				changed = true
			}
		case map[string]any:
			if objectKey == noObjectKey {
				continue
			}
			skillRef, ok := typed[objectKey].(string)
			if !ok {
				continue
			}
			updated, updatedRef, err := updateSkillRefValue(ctx, fieldName, skillRef, allowMajor, verbose, coolDown, resolver)
			if err != nil {
				return false, content, err
			}
			if updated {
				typed[objectKey] = updatedRef
				changed = true
			}
		}
	}
	if !changed {
		return false, content, nil
	}
	result.Frontmatter[fieldName] = rawRefs

	updatedFrontmatter, err := yaml.Marshal(result.Frontmatter)
	if err != nil {
		return false, content, fmt.Errorf("failed to marshal updated frontmatter: %w", err)
	}
	updatedContent, err := parser.ReconstructWorkflowFile(parser.QuoteCronExpressions(string(updatedFrontmatter)), result.Markdown)
	if err != nil {
		return false, content, fmt.Errorf("failed to reconstruct workflow file: %w", err)
	}
	return true, updatedContent, nil
}

func updateSkillRefValue(
	ctx context.Context,
	fieldName string,
	skillRef string,
	allowMajor, verbose bool,
	coolDown time.Duration,
	resolver skillRefUpdateResolver,
) (bool, string, error) {
	trimmedSkillRef := strings.TrimSpace(skillRef)
	if trimmedSkillRef == "" || strings.Contains(trimmedSkillRef, "${{") {
		return false, skillRef, nil
	}
	spec, currentRef, ok := strings.Cut(trimmedSkillRef, "@")
	spec = strings.TrimSpace(spec)
	currentRef = strings.TrimSpace(currentRef)
	if !ok || spec == "" || currentRef == "" {
		return false, skillRef, nil
	}

	repo := gitutil.ExtractBaseRepo(spec)
	if repo == "" {
		return false, skillRef, nil
	}
	latestRef, err := resolver(ctx, repo, currentRef, allowMajor, verbose, coolDown)
	if err != nil {
		if verbose {
			updateLog.Printf("Skipping %s update for %s@%s: %v", fieldName, spec, currentRef, err)
		}
		return false, skillRef, nil
	}
	if latestRef == "" || latestRef == currentRef {
		return false, skillRef, nil
	}
	return true, spec + "@" + latestRef, nil
}

func updateActionRefsInContentWithDeps(ctx context.Context, deps actionUpdateDeps, content string, cache map[string]latestReleaseResult, coolDownCache map[string]coolDownCheckResult, allowMajor, verbose bool, coolDown time.Duration) (bool, string, error) {
	changed := false
	lines := strings.Split(content, "\n")

	for i, line := range lines {
		match := actionRefPattern.FindStringSubmatchIndex(line)
		if match == nil {
			continue
		}

		// Extract matched groups
		prefix := line[match[2]:match[3]] // "uses: "
		repo := line[match[4]:match[5]]   // e.g. "actions/checkout"
		ref := line[match[6]:match[7]]    // SHA or version tag
		comment := ""
		if match[8] >= 0 {
			comment = line[match[8]:match[9]] // e.g. " # v6.0.2"
		}
		trailing := ""
		if match[10] >= 0 {
			trailing = line[match[10]:match[11]]
		}

		// When release bumps are disabled, skip non-core (non actions/*) action refs.
		effectiveAllowMajor := allowMajor || isCoreAction(repo)
		if !effectiveAllowMajor {
			continue
		}

		// Determine the "current version" to pass to the latest-release resolver.
		isSHA := IsCommitSHA(ref)
		currentVersion := ref
		if isSHA {
			// Extract version from comment (e.g., " # v6.0.2" -> "v6.0.2")
			if comment != "" {
				commentVersion := strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(comment), "#"))
				if commentVersion != "" {
					currentVersion = commentVersion
				} else {
					currentVersion = ""
				}
			} else {
				currentVersion = ""
			}
		}

		// Resolve latest version/SHA, using the cache to avoid redundant API calls.
		// Use "|" as separator since GitHub repo names cannot contain "|".
		cacheKey := repo + "|" + currentVersion
		result, cached := cache[cacheKey]
		if !cached {
			latestVersion, latestSHA, err := deps.getLatestRelease(ctx, repo, currentVersion, effectiveAllowMajor, verbose)
			if err != nil {
				updateLog.Printf("Failed to get latest release for %s: %v", repo, err)
				continue
			}
			result = latestReleaseResult{version: latestVersion, sha: latestSHA}
			cache[cacheKey] = result
		}
		latestVersion := result.version
		latestSHA := result.sha

		if isSHA {
			if latestSHA == ref {
				continue // SHA unchanged
			}
		} else {
			if latestVersion == ref {
				continue // Version tag unchanged
			}
			// Prevent downgrades: if the proposed version is older than the current, skip.
			currentVer := parseVersion(ref)
			proposedVer := parseVersion(latestVersion)
			if currentVer != nil && proposedVer != nil && currentVer.IsNewer(proposedVer) {
				updateLog.Printf("Skipping %s in workflow file: proposed version %s is older than current %s (would be a downgrade)", repo, latestVersion, ref)
				continue
			}
		}

		// Apply cooldown: if the repo is not exempt and the release is too recent, try
		// progressively older releases (still newer than current) until finding one that
		// has passed the cooldown period.
		if !isExemptFromCoolDown(repo) {
			coolDownKey := repo + "@" + latestVersion
			coolDownResult, coolDownCached := coolDownCache[coolDownKey]
			if !coolDownCached {
				coolDownResult = deps.checkCoolDown(ctx, repo, latestVersion, coolDown)
				coolDownCache[coolDownKey] = coolDownResult
			}
			if coolDownResult.InCoolDown {
				cooldownLog.Printf("Action ref %s in workflow: %s", repo, coolDownResult.Message)

				// Try to find an older release that has passed the cooldown period.
				olderVersion, olderSHA, findErr := findCooledDownActionVersion(ctx, deps, repo, currentVersion, effectiveAllowMajor, verbose, coolDown, latestVersion)
				if findErr != nil || olderVersion == "" || olderSHA == "" {
					if verbose {
						fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Skipping release candidate %s@%s: %s", repo, latestVersion, coolDownResult.Message)))
					}
					continue
				}
				if verbose {
					fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Falling back to %s for %s (latest release candidate is still in cooldown)", olderVersion, repo)))
				}
				// Use the older, cooled-down release and update the per-invocation cache.
				result = latestReleaseResult{version: olderVersion, sha: olderSHA}
				cache[cacheKey] = result
				latestVersion = olderVersion
				latestSHA = olderSHA
			}
		}

		// Build the new uses line
		var newRef string
		if isSHA {
			// SHA-pinned references stay SHA-pinned, updated to latest SHA + version comment
			newRef = fmt.Sprintf("%s%s%s@%s  # %s%s", line[:match[2]], prefix, repo, latestSHA, latestVersion, trailing)
		} else {
			// Version tag references just get the new version tag
			newRef = fmt.Sprintf("%s%s%s@%s%s%s", line[:match[2]], prefix, repo, latestVersion, comment, trailing)
		}

		updateLog.Printf("Updating %s from %s to %s in line %d", repo, ref, latestVersion, i+1)
		lines[i] = newRef
		changed = true
	}

	return changed, strings.Join(lines, "\n"), nil
}
