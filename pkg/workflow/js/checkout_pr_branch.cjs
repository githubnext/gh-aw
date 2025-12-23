// @ts-check
/// <reference types="@actions/github-script" />

/**
 * Checkout PR branch when PR context is available
 * This script handles both pull_request events and comment events on PRs
 */

const formatError = (error) => error instanceof Error ? error.message : String(error);

async function main() {
  const { eventName, payload: { pull_request: pullRequest } } = context;

  if (!pullRequest) {
    core.info("No pull request context available, skipping checkout");
    return;
  }

  core.info(`Event: ${eventName}`);
  core.info(`Pull Request #${pullRequest.number}`);

  try {
    if (eventName === "pull_request") {
      // For pull_request events, use the head ref directly
      const branchName = pullRequest.head.ref;
      core.info(`Checking out PR branch: ${branchName}`);

      await exec.exec("git", ["fetch", "origin", branchName]);
      await exec.exec("git", ["checkout", branchName]);

      core.info(`✅ Successfully checked out branch: ${branchName}`);
      return;
    }

    // For comment events on PRs, use gh pr checkout with PR number
    const prNumber = pullRequest.number;
    core.info(`Checking out PR #${prNumber} using gh pr checkout`);

    await exec.exec("gh", ["pr", "checkout", prNumber.toString()]);

    core.info(`✅ Successfully checked out PR #${prNumber}`);
  } catch (error) {
    core.setFailed(`Failed to checkout PR branch: ${formatError(error)}`);
  }
}

main().catch(error => {
  core.setFailed(formatError(error));
});
