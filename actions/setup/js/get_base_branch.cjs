// @ts-check
/// <reference types="@actions/github-script" />

const { validateTargetRepo, parseAllowedRepos, getDefaultTargetRepo } = require("./repo_helpers.cjs");

/**
 * Get the base branch name, resolving dynamically based on event context.
 *
 * Resolution order:
 * 1. Custom base branch from env var (explicitly configured in workflow)
 * 2. github.base_ref env var (set for pull_request/pull_request_target events)
 * 3. Pull request payload base ref (pull_request_review, pull_request_review_comment events)
 * 4. API lookup for issue_comment events on PRs (the PR's base ref is not in the payload)
 * 5. Fallback to DEFAULT_BRANCH env var or "main"
 *
 * @param {{owner: string, repo: string}|null} [targetRepo] - Optional target repository.
 *   If provided, API calls (step 4) use this instead of context.repo,
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
        core.warning(`Failed to fetch PR base branch: ${error.message}`);
      }
    }
  }

  // 5. Fallback to DEFAULT_BRANCH env var or "main"
  return process.env.DEFAULT_BRANCH || "main";
}

module.exports = {
  getBaseBranch,
};
