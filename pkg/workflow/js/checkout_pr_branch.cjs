/**
 * Checkout PR branch when PR context is available
 * This script handles both pull_request events and comment events on PRs
 */

const { execSync } = require("child_process");

async function main() {
  const eventName = context.eventName;
  const pullRequest = context.payload.pull_request;

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

      execSync(`git fetch origin ${branchName}`, { stdio: "inherit" });
      execSync(`git checkout ${branchName}`, { stdio: "inherit" });

      core.info(`✅ Successfully checked out branch: ${branchName}`);
    } else {
      // For comment events on PRs, use gh pr checkout with PR number
      const prNumber = pullRequest.number;
      core.info(`Checking out PR #${prNumber} using gh pr checkout`);

      execSync(`gh pr checkout ${prNumber}`, {
        stdio: "inherit",
        env: { ...process.env, GH_TOKEN: process.env.GITHUB_TOKEN },
      });

      core.info(`✅ Successfully checked out PR #${prNumber}`);
    }
  } catch (error) {
    core.setFailed(`Failed to checkout PR branch: ${error instanceof Error ? error.message : String(error)}`);
  }
}

main().catch(error => {
  core.setFailed(error instanceof Error ? error.message : String(error));
});
