// @ts-check
/// <reference types="@actions/github-script" />

/**
 * Assign an issue to a user or bot (including copilot)
 * This script handles assigning issues after they are created
 */

async function main() {
  // Validate required environment variables
  const ghToken = process.env.GH_TOKEN;
  const assignee = process.env.ASSIGNEE;
  const issueNumber = process.env.ISSUE_NUMBER;

  // Check if GH_TOKEN is present
  if (!ghToken || ghToken.trim() === "") {
    const docsUrl = "https://githubnext.github.io/gh-aw/";
    core.setFailed(
      `GH_TOKEN environment variable is required but not set. ` +
        `This token is needed to assign issues. ` +
        `For more information, see: ${docsUrl}`
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

  core.info(`Assigning issue #${trimmedIssueNumber} to ${trimmedAssignee}`);

  try {
    // Use exec to run gh CLI command
    // The GH_TOKEN environment variable is already set and will be used by gh CLI
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
