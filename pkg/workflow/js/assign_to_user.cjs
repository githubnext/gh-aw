// @ts-check
/// <reference types="@actions/github-script" />

const { loadAgentOutput } = require("./load_agent_output.cjs");
const { generateStagedPreview } = require("./staged_preview.cjs");

async function main() {
  const result = loadAgentOutput();
  if (!result.success) {
    return;
  }

  const assignItems = result.items.filter(item => item.type === "assign_to_user");
  if (assignItems.length === 0) {
    core.info("No assign_to_user items found in agent output");
    return;
  }

  core.info(`Found ${assignItems.length} assign_to_user item(s)`);

  // Check if we're in staged mode
  if (process.env.GH_AW_SAFE_OUTPUTS_STAGED === "true") {
    await generateStagedPreview({
      title: "Assign to User",
      description: "The following user assignments would be made if staged mode was disabled:",
      items: assignItems,
      renderItem: item => {
        let content = `**Issue:** #${item.issue_number}\n`;
        content += `**User:** ${item.username}\n`;
        content += "\n";
        return content;
      },
    });
    return;
  }

  // Get max count configuration
  const maxCountEnv = process.env.GH_AW_USER_MAX_COUNT;
  const maxCount = maxCountEnv ? parseInt(maxCountEnv, 10) : 1;
  if (isNaN(maxCount) || maxCount < 1) {
    core.setFailed(`Invalid max value: ${maxCountEnv}. Must be a positive integer`);
    return;
  }
  core.info(`Max count: ${maxCount}`);

  // Get allowed users configuration
  const allowedUsersEnv = process.env.GH_AW_ALLOWED_USERS;
  let allowedUsers = null;
  if (allowedUsersEnv) {
    try {
      allowedUsers = JSON.parse(allowedUsersEnv);
      if (!Array.isArray(allowedUsers)) {
        allowedUsers = null;
      }
    } catch {
      core.warning(`Failed to parse allowed users: ${allowedUsersEnv}`);
    }
  }
  if (allowedUsers) {
    core.info(`Allowed users: ${allowedUsers.join(", ")}`);
  }

  // Limit items to max count
  const itemsToProcess = assignItems.slice(0, maxCount);
  if (assignItems.length > maxCount) {
    core.warning(`Found ${assignItems.length} user assignments, but max is ${maxCount}. Processing first ${maxCount}.`);
  }

  // Get target repository configuration
  const targetRepoEnv = process.env.GH_AW_TARGET_REPO?.trim();
  let targetOwner = context.repo.owner;
  let targetRepo = context.repo.repo;

  if (targetRepoEnv) {
    const parts = targetRepoEnv.split("/");
    if (parts.length === 2) {
      targetOwner = parts[0];
      targetRepo = parts[1];
      core.info(`Using target repository: ${targetOwner}/${targetRepo}`);
    } else {
      core.warning(`Invalid target-repo format: ${targetRepoEnv}. Expected owner/repo. Using current repository.`);
    }
  }

  // Process each user assignment
  const results = [];
  for (const item of itemsToProcess) {
    const issueNumber = typeof item.issue_number === "number" ? item.issue_number : parseInt(String(item.issue_number), 10);
    const username = item.username?.trim();

    if (isNaN(issueNumber) || issueNumber <= 0) {
      core.error(`Invalid issue_number: ${item.issue_number}`);
      results.push({
        issue_number: item.issue_number,
        username: username || "unknown",
        success: false,
        error: "Invalid issue number",
      });
      continue;
    }

    if (!username) {
      core.error(`Missing username for issue #${issueNumber}`);
      results.push({
        issue_number: issueNumber,
        username: "unknown",
        success: false,
        error: "Missing username",
      });
      continue;
    }

    // Check if user is in allowed list (if configured)
    if (allowedUsers && !allowedUsers.includes(username)) {
      core.warning(`User "${username}" is not in the allowed list`);
      results.push({
        issue_number: issueNumber,
        username: username,
        success: false,
        error: `User "${username}" is not in the allowed list`,
      });
      continue;
    }

    // Assign the user to the issue using the REST API
    try {
      core.info(`Assigning user "${username}" to issue #${issueNumber}...`);

      await github.rest.issues.addAssignees({
        owner: targetOwner,
        repo: targetRepo,
        issue_number: issueNumber,
        assignees: [username],
      });

      core.info(`Successfully assigned user "${username}" to issue #${issueNumber}`);
      results.push({
        issue_number: issueNumber,
        username: username,
        success: true,
      });
    } catch (error) {
      const errorMessage = error instanceof Error ? error.message : String(error);
      core.error(`Failed to assign user "${username}" to issue #${issueNumber}: ${errorMessage}`);
      results.push({
        issue_number: issueNumber,
        username: username,
        success: false,
        error: errorMessage,
      });
    }
  }

  // Generate step summary
  const successCount = results.filter(r => r.success).length;
  const failureCount = results.filter(r => !r.success).length;

  let summaryContent = "## User Assignment\n\n";

  if (successCount > 0) {
    summaryContent += `Successfully assigned ${successCount} user(s):\n\n`;
    for (const result of results.filter(r => r.success)) {
      summaryContent += `- Issue #${result.issue_number} -> User: ${result.username}\n`;
    }
    summaryContent += "\n";
  }

  if (failureCount > 0) {
    summaryContent += `Failed to assign ${failureCount} user(s):\n\n`;
    for (const result of results.filter(r => !r.success)) {
      summaryContent += `- Issue #${result.issue_number} -> User: ${result.username}: ${result.error}\n`;
    }
  }

  await core.summary.addRaw(summaryContent).write();

  // Set outputs
  const assignedUsers = results
    .filter(r => r.success)
    .map(r => `${r.issue_number}:${r.username}`)
    .join("\n");
  core.setOutput("assigned_users", assignedUsers);

  // Fail if any assignments failed
  if (failureCount > 0) {
    core.setFailed(`Failed to assign ${failureCount} user(s)`);
  }
}

(async () => {
  await main();
})();
