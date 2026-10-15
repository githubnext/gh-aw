// @ts-check

const { withRetry, RATE_LIMIT_RETRY_CONFIG } = require("./error_recovery.cjs");

async function createOrUpdatePullRequest(options) {
  const { githubClient, repoParts, title, body, branchName, headRepo, baseBranch, draft, assertTransportUnchanged } = options;
  return withRetry(
    async () => {
      assertTransportUnchanged?.();
      const response = await githubClient.rest.pulls.create({
        owner: repoParts.owner,
        repo: repoParts.repo,
        title,
        body,
        head: branchName,
        ...(headRepo ? { head_repo: headRepo } : {}),
        base: baseBranch,
        draft,
      });
      assertTransportUnchanged?.();
      return response;
    },
    RATE_LIMIT_RETRY_CONFIG,
    `create pull request in ${repoParts.owner}/${repoParts.repo}`
  );
}

module.exports = { createOrUpdatePullRequest };
