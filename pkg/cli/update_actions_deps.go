package cli

// This file provides the dependency-injection scaffolding and caching layer shared
// by the action update code paths in update_actions_lockfile.go,
// update_actions_release.go, update_actions_workflow_files.go, and
// update_actions_content_refs.go.

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/github/gh-aw/pkg/gitutil"
	"github.com/github/gh-aw/pkg/workflow"
)

// isCoreAction returns true if the repo is a GitHub-maintained core action (actions/* org).
// Core actions are always updated to the latest major version without requiring --major.
func isCoreAction(repo string) bool {
	return strings.HasPrefix(repo, "actions/")
}

// isGhAwNativeAction returns true if the action repo is part of the gh-aw native ecosystem
// (i.e., maintained in the github/gh-aw or github/gh-aw-actions repository). These actions
// are versioned in lock-step with the CLI and must never be updated beyond the running CLI version.
func isGhAwNativeAction(repo string) bool {
	base := gitutil.ExtractBaseRepo(repo)
	return base == "github/gh-aw" || base == "github/gh-aw-actions"
}

type actionUpdateDeps struct {
	getLatestRelease       func(ctx context.Context, repo, currentVersion string, allowMajor, verbose bool) (string, string, error)
	getLatestReleaseViaGit func(ctx context.Context, repo, currentVersion string, allowMajor, verbose bool) (string, string, error)
	runGHReleasesAPI       func(ctx context.Context, baseRepo string) ([]byte, error)
	getActionSHAForTag     func(ctx context.Context, repo, tag string) (string, error)
	checkCoolDown          func(ctx context.Context, repo, tag string, coolDown time.Duration) coolDownCheckResult
}

type cachedLatestRelease struct {
	version string
	sha     string
	err     error
}

type cachedSHA struct {
	sha string
	err error
}

// newCachedActionUpdateDeps memoizes GitHub reads for the full update command.
// This cache is shared by actions-lock.json updates and Markdown action refs.
func newCachedActionUpdateDeps(base actionUpdateDeps) actionUpdateDeps {
	var mu sync.Mutex
	latestReleases := make(map[string]cachedLatestRelease)
	releaseLists := make(map[string]struct {
		output []byte
		err    error
	})
	shas := make(map[string]cachedSHA)
	cooldowns := make(map[string]coolDownCheckResult)

	cached := base
	cached.getLatestRelease = func(ctx context.Context, repo, currentVersion string, allowMajor, verbose bool) (string, string, error) {
		key := fmt.Sprintf("%s|%s|%t", repo, currentVersion, allowMajor)
		mu.Lock()
		result, ok := latestReleases[key]
		mu.Unlock()
		if ok {
			return result.version, result.sha, result.err
		}
		version, sha, err := base.getLatestRelease(ctx, repo, currentVersion, allowMajor, verbose)
		mu.Lock()
		latestReleases[key] = cachedLatestRelease{version: version, sha: sha, err: err}
		mu.Unlock()
		return version, sha, err
	}
	cached.runGHReleasesAPI = func(ctx context.Context, repo string) ([]byte, error) {
		mu.Lock()
		result, ok := releaseLists[repo]
		mu.Unlock()
		if ok {
			return result.output, result.err
		}
		output, err := base.runGHReleasesAPI(ctx, repo)
		mu.Lock()
		releaseLists[repo] = struct {
			output []byte
			err    error
		}{output: output, err: err}
		mu.Unlock()
		return output, err
	}
	cached.getActionSHAForTag = func(ctx context.Context, repo, tag string) (string, error) {
		key := repo + "|" + tag
		mu.Lock()
		result, ok := shas[key]
		mu.Unlock()
		if ok {
			return result.sha, result.err
		}
		sha, err := base.getActionSHAForTag(ctx, repo, tag)
		mu.Lock()
		shas[key] = cachedSHA{sha: sha, err: err}
		mu.Unlock()
		return sha, err
	}
	cached.checkCoolDown = func(ctx context.Context, repo, tag string, coolDown time.Duration) coolDownCheckResult {
		key := fmt.Sprintf("%s|%s|%s", repo, tag, coolDown)
		mu.Lock()
		result, ok := cooldowns[key]
		mu.Unlock()
		if ok {
			return result
		}
		result = base.checkCoolDown(ctx, repo, tag, coolDown)
		mu.Lock()
		cooldowns[key] = result
		mu.Unlock()
		return result
	}
	return cached
}

func defaultActionUpdateDeps() actionUpdateDeps {
	return actionUpdateDeps{
		getLatestRelease:       getLatestActionRelease,
		getLatestReleaseViaGit: getLatestActionReleaseViaGit,
		checkCoolDown:          checkReleaseCoolDown,
		runGHReleasesAPI: func(ctx context.Context, baseRepo string) ([]byte, error) {
			return workflow.RunGHCombinedContext(ctx, "Fetching releases...", "api", "--paginate", fmt.Sprintf("/repos/%s/releases", baseRepo), "--jq", ".[].tag_name")
		},
		getActionSHAForTag: getActionSHAForTag,
	}
}
