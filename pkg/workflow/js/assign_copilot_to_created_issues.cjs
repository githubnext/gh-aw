// @ts-check
/// <reference types="@actions/github-script" />

const { AGENT_LOGIN_NAMES, findAgent, getIssueDetails, assignAgentToIssue, generatePermissionErrorSummary } = require("./assign_agent_helpers.cjs");

/**
 * Assign copilot to issues created by create_issue job.
 * This script reads the issues_to_assign_copilot output and assigns copilot to each issue.
 * It uses the agent token (GH_AW_AGENT_TOKEN) for the GraphQL mutation.
 */

async function main() {
  // Get the issues to assign from step output
  const issuesToAssignStr = "${{ steps.create_issue.outputs.issues_to_assign_copilot }}";

  if (!issuesToAssignStr || issuesToAssignStr.trim() === "") {
    core.info("No issues to assign copilot to");
    return;
  }

  core.info(`Issues to assign copilot: ${issuesToAssignStr}`);

  // Parse the comma-separated list of repo:number entries
  const issueEntries = issuesToAssignStr.split(",").filter(entry => entry.trim() !== "");
  if (issueEntries.length === 0) {
    core.info("No valid issue entries found");
    return;
  }

  core.info(`Processing ${issueEntries.length} issue(s) for copilot assignment`);

  const agentName = "copilot";
  const results = [];
  let agentId = null;

  for (const entry of issueEntries) {
    // Parse repo:number format
    const parts = entry.split(":");
    if (parts.length !== 2) {
      core.warning(`Invalid issue entry format: ${entry}. Expected 'owner/repo:number'`);
      continue;
    }

    const repoSlug = parts[0];
    const issueNumber = parseInt(parts[1], 10);

    if (isNaN(issueNumber) || issueNumber <= 0) {
      core.warning(`Invalid issue number in entry: ${entry}`);
      continue;
    }

    // Parse owner/repo from repo slug
    const repoParts = repoSlug.split("/");
    if (repoParts.length !== 2) {
      core.warning(`Invalid repo format: ${repoSlug}. Expected 'owner/repo'`);
      continue;
    }

    const owner = repoParts[0];
    const repo = repoParts[1];

    try {
      // Find agent (reuse cached ID for same repo)
      if (!agentId) {
        core.info(`Looking for ${agentName} coding agent...`);
        agentId = await findAgent(owner, repo, agentName);
        if (!agentId) {
          throw new Error(`${agentName} coding agent is not available for this repository`);
        }
        core.info(`Found ${agentName} coding agent (ID: ${agentId})`);
      }

      // Get issue details
      core.info(`Getting details for issue #${issueNumber} in ${repoSlug}...`);
      const issueDetails = await getIssueDetails(owner, repo, issueNumber);
      if (!issueDetails) {
        throw new Error("Failed to get issue details");
      }

      core.info(`Issue ID: ${issueDetails.issueId}`);

      // Check if agent is already assigned
      if (issueDetails.currentAssignees.includes(agentId)) {
        core.info(`${agentName} is already assigned to issue #${issueNumber}`);
        results.push({
          repo: repoSlug,
          issue_number: issueNumber,
          success: true,
          already_assigned: true,
        });
        continue;
      }

      // Assign agent using GraphQL mutation
      core.info(`Assigning ${agentName} coding agent to issue #${issueNumber}...`);
      const success = await assignAgentToIssue(issueDetails.issueId, agentId, issueDetails.currentAssignees, agentName);

      if (!success) {
        throw new Error(`Failed to assign ${agentName} via GraphQL`);
      }

      core.info(`Successfully assigned ${agentName} coding agent to issue #${issueNumber}`);
      results.push({
        repo: repoSlug,
        issue_number: issueNumber,
        success: true,
      });
    } catch (error) {
      const errorMessage = error instanceof Error ? error.message : String(error);
      core.error(`Failed to assign ${agentName} to issue #${issueNumber} in ${repoSlug}: ${errorMessage}`);
      results.push({
        repo: repoSlug,
        issue_number: issueNumber,
        success: false,
        error: errorMessage,
      });
    }
  }

  // Generate step summary
  const successCount = results.filter(r => r.success).length;
  const failureCount = results.filter(r => !r.success).length;

  let summaryContent = "## Copilot Assignment for Created Issues\n\n";

  if (successCount > 0) {
    summaryContent += `✅ Successfully assigned copilot to ${successCount} issue(s):\n\n`;
    for (const result of results.filter(r => r.success)) {
      const note = result.already_assigned ? " (already assigned)" : "";
      summaryContent += `- ${result.repo}#${result.issue_number}${note}\n`;
    }
    summaryContent += "\n";
  }

  if (failureCount > 0) {
    summaryContent += `❌ Failed to assign copilot to ${failureCount} issue(s):\n\n`;
    for (const result of results.filter(r => !r.success)) {
      summaryContent += `- ${result.repo}#${result.issue_number}: ${result.error}\n`;
    }

    // Check if any failures were permission-related
    const hasPermissionError = results.some(r => !r.success && r.error && (r.error.includes("Resource not accessible") || r.error.includes("Insufficient permissions")));

    if (hasPermissionError) {
      summaryContent += generatePermissionErrorSummary();
    }
  }

  await core.summary.addRaw(summaryContent).write();

  // Fail if any assignments failed
  if (failureCount > 0) {
    core.setFailed(`Failed to assign copilot to ${failureCount} issue(s)`);
  }
}

// Export for use with require()
if (typeof module !== "undefined" && module.exports) {
  module.exports = { main };
}
