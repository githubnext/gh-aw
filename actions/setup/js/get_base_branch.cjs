// @ts-check
/// <reference types="@actions/github-script" />

const { validateTargetRepo, parseAllowedRepos, getDefaultTargetRepo } = require("./repo_helpers.cjs");
const { getErrorMessage } = require("./error_helpers.cjs");

/**
 * Get the base branch name, resolving dynamically based on event context.
 *
 * Resolution order:
 * 1. Custom base branch from env var (explicitly configured in workflow)
 * 2. github.base_ref env var (set for pull_request/pull_request_target events)
 * 3. Pull request payload base ref (pull_request_review, pull_request_review_comment events)
 * 4. API lookup for issue_comment events on PRs (the PR's base ref is not in the payload)
 * 5. context.payload.repository.default_branch (included in most event payloads, no API call)
 * 5b. API lookup via repos.get() when payload doesn't have it (e.g. cross-repo scenarios)
 * 6. Fallback to DEFAULT_BRANCH env var or "main"
 *
 * @param {{owner: string, repo: string}|null} [targetRepo] - Optional target repository.
 *   If provided, API calls (steps 4 and 5) use this instead of context.repo,
 *   which is needed for cross-repo scenarios where the target repo differs
 *   from the workflow repository.
 * @returns {Promise<string>} The base branch name
 */
async function getBaseBranch(targetRepo = null) {
  // 1. Custom base branch from workflow configuration
  if (process.env.GH_AW_CUSTOM_BASE_BRANCH) {
    return process.env.GH_AW_CUSTOM_BASE_BRANCH;
  }

  // 2. github.base_ref - set by GitHub Actions for pull_request/pull_request_target events
  if (process.env.GITHUB_BASE_REF) {
    return process.env.GITHUB_BASE_REF;
  }

  // 3. From pull request payload (pull_request_review, pull_request_review_comment events)
  if (typeof context !== "undefined" && context.payload?.pull_request?.base?.ref) {
    return context.payload.pull_request.base.ref;
  }

  // 4. For issue_comment events on PRs - must call API since base ref not in payload
  // Use targetRepo if provided (cross-repo scenarios), otherwise fall back to context.repo
  if (typeof context !== "undefined" && context.eventName === "issue_comment" && context.payload?.issue?.pull_request) {
    try {
      if (typeof github !== "undefined") {
        const repoOwner = targetRepo?.owner ?? context.repo.owner;
        const repoName = targetRepo?.repo ?? context.repo.repo;

        // Validate target repo against allowlist before any API calls
        const targetRepoSlug = `${repoOwner}/${repoName}`;
        const allowedRepos = parseAllowedRepos(process.env.GH_AW_ALLOWED_REPOS);
        if (allowedRepos.size > 0) {
          const defaultRepo = getDefaultTargetRepo();
          const validation = validateTargetRepo(targetRepoSlug, defaultRepo, allowedRepos);
          if (!validation.valid) {
            if (typeof core !== "undefined") {
              core.warning(`ERR_VALIDATION: ${validation.error}`);
            }
            return process.env.DEFAULT_BRANCH || "main";
          }
        }

        const { data: pr } = await github.rest.pulls.get({
          owner: repoOwner,
          repo: repoName,
          pull_number: context.payload.issue.number,
        });
        return pr.base.ref;
      }
    } catch (/** @type {any} */ error) {
      // Fall through to default if API call fails
      if (typeof core !== "undefined") {
        core.warning(`Failed to fetch PR base branch: ${getErrorMessage(error)}`);
      }
    }
  }

  // 5. Repository default branch - from payload first, then API lookup
  // Many events include context.payload.repository.default_branch, so we check that
  // first to avoid an unnecessary API call and reduce rate-limit risk.
  // Only fall back to repos.get() when the payload doesn't have the value
  // (e.g. cross-repo scenarios where targetRepo differs from the workflow repo).
  {
    const repoOwner = targetRepo?.owner ?? (typeof context !== "undefined" ? context.repo?.owner : null);
    const repoName = targetRepo?.repo ?? (typeof context !== "undefined" ? context.repo?.repo : null);

    // If no targetRepo override, check the payload first (free - no API call)
    if (!targetRepo && typeof context !== "undefined" && context.payload?.repository?.default_branch) {
      return context.payload.repository.default_branch;
    }

    // Otherwise fall back to repos.get() API call
    if (repoOwner && repoName && typeof github !== "undefined") {
      try {
        // SECURITY: Validate target repo against allowlist before any API calls
        const targetRepoSlug = `${repoOwner}/${repoName}`;
        const allowedRepos = parseAllowedRepos(process.env.GH_AW_ALLOWED_REPOS);
        if (allowedRepos.size > 0) {
          const defaultRepo = getDefaultTargetRepo();
          const validation = validateTargetRepo(targetRepoSlug, defaultRepo, allowedRepos);
          if (!validation.valid) {
            if (typeof core !== "undefined") {
              core.warning(`ERR_VALIDATION: ${validation.error}`);
            }
            return process.env.DEFAULT_BRANCH || "main";
          }
        }

        const { data: repoData } = await github.rest.repos.get({
          owner: repoOwner,
          repo: repoName,
        });
        return repoData.default_branch;
      } catch (/** @type {any} */ error) {
        // Fall through to default if API call fails
        if (typeof core !== "undefined") {
          core.warning(`Failed to fetch repository default branch: ${getErrorMessage(error)}`);
        }
      }
    }
  }

  // 6. Fallback to DEFAULT_BRANCH env var or "main"
  return process.env.DEFAULT_BRANCH || "main";
}

module.exports = {
  getBaseBranch,
};
