// @ts-check

const { withRetry, RATE_LIMIT_RETRY_CONFIG } = require("./error_recovery.cjs");

async function createOrUpdatePullRequest(options) {
  const { githubClient, repoParts, title, body, branchName, baseBranch, draft, preCreatedPullRequestNumber, preCreatedBranch } = options;
  if (!(preCreatedPullRequestNumber > 0)) {
    return withRetry(
      () =>
        githubClient.rest.pulls.create({
          owner: repoParts.owner,
          repo: repoParts.repo,
          title,
          body,
          head: branchName,
          base: baseBranch,
          draft,
        }),
      RATE_LIMIT_RETRY_CONFIG,
      `create pull request in ${repoParts.owner}/${repoParts.repo}`
    );
  }

  return withRetry(
    async () => {
      const { data: existingPullRequest } = await githubClient.rest.pulls.get({
        owner: repoParts.owner,
        repo: repoParts.repo,
        pull_number: preCreatedPullRequestNumber,
      });
      if (existingPullRequest.state !== "open" || existingPullRequest.head.ref !== preCreatedBranch) {
        throw new Error(`Pre-created pull request #${preCreatedPullRequestNumber} is not open on branch ${preCreatedBranch}`);
      }
      return githubClient.rest.pulls.update({
        owner: repoParts.owner,
        repo: repoParts.repo,
        pull_number: preCreatedPullRequestNumber,
        title,
        body,
        base: baseBranch,
      });
    },
    RATE_LIMIT_RETRY_CONFIG,
    `update pre-created pull request #${preCreatedPullRequestNumber}`
  );
}

module.exports = { createOrUpdatePullRequest };
