// @ts-check
/// <reference types="@actions/github-script" />

/**
 * Assign an issue to a regular user
 * This script handles assigning issues to regular users (not agents) after they are created
 * Agent assignment (e.g., copilot) is handled separately with dedicated steps that use agent tokens
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
    // Note: Agent assignment (e.g., copilot) is handled separately in dedicated steps
    // with proper agent token authentication. This script is only for regular user assignment.
    // Use gh CLI for regular user assignment
    await exec.exec("gh", ["issue", "edit", trimmedIssueNumber, "--add-assignee", trimmedAssignee], {
      env: { ...process.env, GH_TOKEN: ghToken },
    });

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
