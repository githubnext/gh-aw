// @ts-check
/// <reference types="@actions/github-script" />

/**
 * Unlock a GitHub issue
 * This script is used in the conclusion job to ensure the issue is unlocked
 * after agent workflow execution completes or fails
 */

async function main() {
  // Get issue number from context
  const issueNumber = context.issue.number;

  if (!issueNumber) {
    core.setFailed("Issue number not found in context");
    return;
  }

  const owner = context.repo.owner;
  const repo = context.repo.repo;

  core.info(`Unlocking issue #${issueNumber} after agent workflow execution`);

  try {
    // Unlock the issue
    await github.rest.issues.unlock({
      owner,
      repo,
      issue_number: issueNumber,
    });

    core.info(`✅ Successfully unlocked issue #${issueNumber}`);
  } catch (error) {
    const errorMessage = error instanceof Error ? error.message : String(error);
    core.error(`Failed to unlock issue: ${errorMessage}`);
    core.setFailed(`Failed to unlock issue #${issueNumber}: ${errorMessage}`);
  }
}

await main();
