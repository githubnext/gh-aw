// @ts-check
/// <reference types="@actions/github-script" />

const { assignIssue } = require("./assign_agent_helpers.cjs");

/**
 * Assign an issue to a user or bot (including copilot)
 * This script handles assigning issues after they are created
 * Uses the unified assignIssue helper that works for both users and agents via GraphQL
 */

async function main() {
  // Validate required environment variables
  const ghToken = process.env.GH_TOKEN;
  const assignee = process.env.ASSIGNEE;
  const issueNumber = process.env.ISSUE_NUMBER;

  // Check if GH_TOKEN is present
  if (!ghToken || ghToken.trim() === "") {
    const docsUrl = "https://githubnext.github.io/gh-aw/reference/safe-outputs/#assigning-issues-to-copilot";
    core.setFailed(
      `GH_TOKEN environment variable is required but not set. ` +
        `This token is needed to assign issues. ` +
        `For more information on configuring Copilot tokens, see: ${docsUrl}`
    );
    return;
  }

  // Validate assignee
  if (!assignee || assignee.trim() === "") {
    core.setFailed("ASSIGNEE environment variable is required but not set");
    return;
  }

  // Validate issue number
  if (!issueNumber || issueNumber.trim() === "") {
    core.setFailed("ISSUE_NUMBER environment variable is required but not set");
    return;
  }

  const trimmedAssignee = assignee.trim();
  const trimmedIssueNumber = issueNumber.trim();
  const issueNum = parseInt(trimmedIssueNumber, 10);

  core.info(`Assigning issue #${trimmedIssueNumber} to ${trimmedAssignee}`);

  try {
    // Get repository owner and repo from context
    const owner = context.repo.owner;
    const repo = context.repo.repo;

    // Use the unified assignIssue helper that works for both users and agents via GraphQL
    // The token is set at the step level via github-token parameter
    const result = await assignIssue(owner, repo, issueNum, trimmedAssignee);

    if (!result.success) {
      throw new Error(result.error || "Failed to assign issue");
    }

    core.info(`✅ Successfully assigned issue #${trimmedIssueNumber} to ${trimmedAssignee}`);

    // Write summary
    await core.summary
      .addRaw(
        `
## Issue Assignment

Successfully assigned issue #${trimmedIssueNumber} to \`${trimmedAssignee}\`.
`
      )
      .write();
  } catch (error) {
    const errorMessage = error instanceof Error ? error.message : String(error);
    core.error(`Failed to assign issue: ${errorMessage}`);
    core.setFailed(`Failed to assign issue #${trimmedIssueNumber} to ${trimmedAssignee}: ${errorMessage}`);
  }
}

main().catch(error => {
  core.setFailed(error instanceof Error ? error.message : String(error));
});
