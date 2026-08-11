package cli

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/goccy/go-yaml"

	"github.com/github/gh-aw/pkg/console"
	"github.com/github/gh-aw/pkg/constants"
	"github.com/github/gh-aw/pkg/gitutil"
	"github.com/github/gh-aw/pkg/parser"
)

var actionRefPattern = regexp.MustCompile(`(uses:\s+)([a-zA-Z0-9][a-zA-Z0-9_-]*/[a-zA-Z0-9_.-]+(?:/[a-zA-Z0-9_.-]+)*)@([a-fA-F0-9]{40}|[^\s#\n]+?)(\s*#\s*\S+)?(\s*)$`)

// latestReleaseResult caches a resolved version/SHA pair.
type latestReleaseResult struct {
	version string
	sha     string
}

// UpdateActionsInWorkflowFiles scans all workflow .md files under workflowsDir
// (recursively) and updates any "uses: org/repo@version" references to the latest
// major version. Updated files are recompiled. By default all actions are updated to
// the latest major version; pass disableReleaseBump=true to only update core
// (actions/*) references.
func UpdateActionsInWorkflowFiles(ctx context.Context, workflowsDir, engineOverride string, verbose, disableReleaseBump bool, noCompile bool, coolDown time.Duration, approve bool) error {
	return updateActionsInWorkflowFiles(ctx, defaultActionUpdateDeps(), updateActionsOptions{
		workflowsDir:       workflowsDir,
		engineOverride:     engineOverride,
		verbose:            verbose,
		disableReleaseBump: disableReleaseBump,
		noCompile:          noCompile,
		coolDown:           coolDown,
		approve:            approve,
	})
}

// updateActionsOptions bundles the configuration parameters for updateActionsInWorkflowFiles,
// collapsing a long positional parameter list into a struct.
// engineOverride sets a non-default agentic engine for recompiled workflows.
// disableReleaseBump prevents upgrading action/skill references to newer releases.
// noCompile skips recompilation of updated workflow files.
// coolDown is the minimum age a release must have before it is considered for upgrade.
// approve auto-approves any interactive prompts during recompilation.
type updateActionsOptions struct {
	workflowsDir       string
	engineOverride     string
	verbose            bool
	disableReleaseBump bool
	noCompile          bool
	coolDown           time.Duration
	approve            bool
}

func updateActionsInWorkflowFiles(ctx context.Context, deps actionUpdateDeps, opts updateActionsOptions) error {
	if opts.workflowsDir == "" {
		opts.workflowsDir = getWorkflowsDir()
	}

	updateLog.Printf("Updating action references in workflow files: dir=%s", opts.workflowsDir)

	// Per-invocation cache: key = "repo@currentVersion", avoids repeated API calls
	cache := make(map[string]latestReleaseResult)
	// Per-invocation cooldown cache: key = "repo@tag", avoids redundant date API calls
	coolDownCache := make(map[string]coolDownCheckResult)

	var updatedFiles []string

	err := filepath.WalkDir(opts.workflowsDir, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if d.IsDir() || !strings.HasSuffix(d.Name(), ".md") {
			return nil
		}

		content, err := os.ReadFile(path)
		if err != nil {
			if opts.verbose {
				fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to read %s: %v", path, err)))
			}
			return nil
		}

		updatedActions, newContent, err := updateActionRefsInContentWithDeps(ctx, deps, string(content), cache, coolDownCache, !opts.disableReleaseBump, opts.verbose, opts.coolDown)
		if err != nil {
			if opts.verbose {
				fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to update action refs in %s: %v", path, err)))
			}
			return nil
		}
		updatedSkills, newContent, err := updateSkillRefsInContent(ctx, newContent, !opts.disableReleaseBump, opts.verbose, opts.coolDown)
		if err != nil {
			if opts.verbose {
				fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to update skill refs in %s: %v", path, err)))
			}
			return nil
		}

		if !updatedActions && !updatedSkills {
			return nil
		}

		if err := os.WriteFile(path, []byte(newContent), constants.FilePermPublic); err != nil {
			return fmt.Errorf("failed to write updated workflow %s: %w", path, err)
		}

		fmt.Fprintln(os.Stderr, console.FormatSuccessMessage("Updated action/skill references in "+d.Name()))
		updatedFiles = append(updatedFiles, path)
		return nil
	})
	if err != nil {
		return fmt.Errorf("failed to walk workflows directory: %w", err)
	}

	if len(updatedFiles) > 0 && !opts.noCompile {
		if err := compileWorkflowsForUpdate(ctx, updatedFiles, opts.workflowsDir, opts.engineOverride, opts.verbose, opts.approve); err != nil {
			return fmt.Errorf("failed to compile workflows with updated action references: %w", err)
		}
	}

	if len(updatedFiles) == 0 && opts.verbose {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage("No action references needed updating in workflow files"))
	}

	return nil
}

type skillRefUpdateResolver func(ctx context.Context, repo, currentRef string, allowMajor, verbose bool, coolDown time.Duration) (string, error)

func updateSkillRefsInContent(ctx context.Context, content string, allowMajor, verbose bool, coolDown time.Duration) (bool, string, error) {
	return updateSkillRefsInContentWithResolver(ctx, content, allowMajor, verbose, coolDown, resolveLatestRef)
}

func updateSkillRefsInContentWithResolver(
	ctx context.Context,
	content string,
	allowMajor, verbose bool,
	coolDown time.Duration,
	resolver skillRefUpdateResolver,
) (bool, string, error) {
	result, err := parser.ExtractFrontmatterFromContent(content)
	if err != nil {
		if verbose {
			updateLog.Printf("Skipping skill update for content without parseable frontmatter: %v", err)
		}
		return false, content, nil
	}
	if result == nil || result.Frontmatter == nil {
		return false, content, nil
	}

	rawSkills, ok := result.Frontmatter["skills"].([]any)
	if !ok || len(rawSkills) == 0 {
		return false, content, nil
	}

	changed := false
	for i, rawSkill := range rawSkills {
		switch typed := rawSkill.(type) {
		case string:
			updated, updatedRef, err := updateSkillRefValue(ctx, typed, allowMajor, verbose, coolDown, resolver)
			if err != nil {
				return false, content, err
			}
			if updated {
				rawSkills[i] = updatedRef
				changed = true
			}
		case map[string]any:
			skillRef, ok := typed["skill"].(string)
			if !ok {
				continue
			}
			updated, updatedRef, err := updateSkillRefValue(ctx, skillRef, allowMajor, verbose, coolDown, resolver)
			if err != nil {
				return false, content, err
			}
			if updated {
				typed["skill"] = updatedRef
				changed = true
			}
		}
	}
	if !changed {
		return false, content, nil
	}
	result.Frontmatter["skills"] = rawSkills

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
			updateLog.Printf("Skipping skill update for %s@%s: %v", spec, currentRef, err)
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
