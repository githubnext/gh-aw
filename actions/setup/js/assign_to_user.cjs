// @ts-check
/// <reference types="@actions/github-script" />

const { processSafeOutput, processItems } = require("./safe_output_processor.cjs");

async function main() {
  // Use shared processor for common steps
  const result = await processSafeOutput(
    {
      itemType: "assign_to_user",
      configKey: "assign_to_user",
      displayName: "Assignees",
      itemTypeName: "user assignment",
      supportsPR: false, // Issue-only: not relevant for PRs
      supportsIssue: true,
      envVars: {
        allowed: "GH_AW_ASSIGNEES_ALLOWED",
        maxCount: "GH_AW_ASSIGNEES_MAX_COUNT",
        target: "GH_AW_ASSIGNEES_TARGET",
      },
    },
    {
      title: "Assign to User",
      description: "The following user assignments would be made if staged mode was disabled:",
      renderItem: item => {
        let content = "";
        if (item.issue_number) {
          content += `**Target Issue:** #${item.issue_number}\n\n`;
        } else {
          content += `**Target:** Current issue\n\n`;
        }
        if (item.assignees && item.assignees.length > 0) {
          content += `**Users to assign:** ${item.assignees.join(", ")}\n\n`;
        } else if (item.assignee) {
          content += `**User to assign:** ${item.assignee}\n\n`;
        }
        return content;
      },
    }
  );

  if (!result.success) {
    return;
  }

  // @ts-ignore - TypeScript doesn't narrow properly after success check
  const { item: assignItem, config, targetResult } = result;
  if (!config || !targetResult || targetResult.number === undefined) {
    core.setFailed("Internal error: config, targetResult, or targetResult.number is undefined");
    return;
  }
  const { allowed: allowedAssignees, maxCount } = config;
  const issueNumber = targetResult.number;

  // Support both singular "assignee" and plural "assignees" for flexibility
  const requestedAssignees = assignItem.assignees && Array.isArray(assignItem.assignees) ? assignItem.assignees : assignItem.assignee ? [assignItem.assignee] : [];

  core.info(`Requested assignees: ${JSON.stringify(requestedAssignees)}`);

  // Use shared helper to filter, sanitize, dedupe, and limit
  const uniqueAssignees = processItems(requestedAssignees, allowedAssignees, maxCount);

  if (uniqueAssignees.length === 0) {
    core.info("No assignees to add");
    core.setOutput("assigned_users", "");
    await core.summary
      .addRaw(
        `
## User Assignment

No users were assigned (no valid assignees found in agent output).
`
      )
      .write();
    return;
  }

  core.info(`Assigning ${uniqueAssignees.length} users to issue #${issueNumber}: ${JSON.stringify(uniqueAssignees)}`);

  // Get target repository from environment or use current
  const targetRepoEnv = process.env.GH_AW_TARGET_REPO_SLUG?.trim();
  const [targetOwner, targetRepo] = targetRepoEnv?.includes("/")
    ? targetRepoEnv.split("/")
    : [context.repo.owner, context.repo.repo];

  if (targetRepoEnv?.includes("/")) {
    core.info(`Using target repository: ${targetOwner}/${targetRepo}`);
  }

  try {
    // Add assignees to the issue
    await github.rest.issues.addAssignees({
      owner: targetOwner,
      repo: targetRepo,
      issue_number: issueNumber,
      assignees: uniqueAssignees,
    });

    core.info(`Successfully assigned ${uniqueAssignees.length} user(s) to issue #${issueNumber}`);
    core.setOutput("assigned_users", uniqueAssignees.join("\n"));

    const assigneesListMarkdown = uniqueAssignees.map(assignee => `- \`${assignee}\``).join("\n");
    await core.summary
      .addRaw(
        `
## User Assignment

Successfully assigned ${uniqueAssignees.length} user(s) to issue #${issueNumber}:

${assigneesListMarkdown}
`
      )
      .write();
  } catch (error) {
    const errorMessage = error?.message ?? String(error);
    core.error(`Failed to assign users: ${errorMessage}`);
    core.setFailed(`Failed to assign users: ${errorMessage}`);
  }
}

module.exports = { main };
