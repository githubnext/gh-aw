// @ts-check
/// <reference types="@actions/github-script" />

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
 * @returns {Promise<string>} The base branch name
 */
async function getBaseBranch() {
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
  if (typeof context !== "undefined" && context.eventName === "issue_comment" && context.payload?.issue?.pull_request) {
    try {
      if (typeof github !== "undefined") {
        const { data: pr } = await github.rest.pulls.get({
          owner: context.repo.owner,
          repo: context.repo.repo,
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
