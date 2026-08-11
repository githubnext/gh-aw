package cli

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/github/gh-aw/pkg/workflow"
)

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

// UpdateActions updates GitHub Actions versions in .github/aw/actions-lock.json
// It checks each action for newer releases and updates the SHA if a newer version is found.
// By default all actions are updated to the latest major version; pass disableReleaseBump=true
// to revert to the old behaviour where only core (actions/*) actions bypass the --major flag.
//
// coolDown specifies the minimum age a release must have before it is applied. Repos under the
// "actions/" and "github/" namespaces are always exempt from the cooldown.
//
// The ActionCache helpers from pkg/workflow are used so that cached inputs and descriptions
// for safe-outputs.actions entries are preserved when their SHA is unchanged, and cleared
