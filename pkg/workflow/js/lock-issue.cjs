// @ts-check
/// <reference types="@actions/github-script" />

/**
 * Lock a GitHub issue without providing a reason
 * This script is used in the activation job when lock-for-agent is enabled
 * to prevent concurrent modifications during agent workflow execution
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

  try {
    // Check if issue is already locked
    core.info(`Checking if issue #${issueNumber} is already locked`);
    const { data: issue } = await github.rest.issues.get({
      owner,
      repo,
      issue_number: issueNumber,
    });

    if (issue.locked) {
      core.info(`ℹ️ Issue #${issueNumber} is already locked, skipping lock operation`);
      return;
    }

    core.info(`Locking issue #${issueNumber} for agent workflow execution`);

    // Lock the issue without providing a lock_reason parameter
    await github.rest.issues.lock({
      owner,
      repo,
      issue_number: issueNumber,
    });

    core.info(`✅ Successfully locked issue #${issueNumber}`);
  } catch (error) {
    const errorMessage = error instanceof Error ? error.message : String(error);
    core.error(`Failed to lock issue: ${errorMessage}`);
    core.setFailed(`Failed to lock issue #${issueNumber}: ${errorMessage}`);
  }
}

await main();
