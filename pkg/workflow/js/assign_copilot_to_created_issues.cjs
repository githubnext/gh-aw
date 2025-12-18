const { AGENT_LOGIN_NAMES, findAgent, getIssueDetails, assignAgentToIssue, generatePermissionErrorSummary } = require("./assign_agent_helpers.cjs");
async function main() {
  const issuesToAssignStr = "${{ steps.create_issue.outputs.issues_to_assign_copilot }}";
  if ("" === issuesToAssignStr.trim()) return void core.info("No issues to assign copilot to");
  core.info(`Issues to assign copilot: ${issuesToAssignStr}`);
  const issueEntries = issuesToAssignStr.split(",").filter(entry => "" !== entry.trim());
  if (0 === issueEntries.length) return void core.info("No valid issue entries found");
  core.info(`Processing ${issueEntries.length} issue(s) for copilot assignment`);
  const results = [];
  let agentId = null;
  for (const entry of issueEntries) {
    const parts = entry.split(":");
    if (2 !== parts.length) {
      core.warning(`Invalid issue entry format: ${entry}. Expected 'owner/repo:number'`);
      continue;
    }
    const repoSlug = parts[0],
      issueNumber = parseInt(parts[1], 10);
    if (isNaN(issueNumber) || issueNumber <= 0) {
      core.warning(`Invalid issue number in entry: ${entry}`);
      continue;
    }
    const repoParts = repoSlug.split("/");
    if (2 !== repoParts.length) {
      core.warning(`Invalid repo format: ${repoSlug}. Expected 'owner/repo'`);
      continue;
    }
    const owner = repoParts[0],
      repo = repoParts[1];
    try {
      if (!agentId) {
        if ((core.info("Looking for copilot coding agent..."), (agentId = await findAgent(owner, repo, "copilot")), !agentId)) throw new Error("copilot coding agent is not available for this repository");
        core.info(`Found copilot coding agent (ID: ${agentId})`);
      }
      core.info(`Getting details for issue #${issueNumber} in ${repoSlug}...`);
      const issueDetails = await getIssueDetails(owner, repo, issueNumber);
      if (!issueDetails) throw new Error("Failed to get issue details");
      if ((core.info(`Issue ID: ${issueDetails.issueId}`), issueDetails.currentAssignees.includes(agentId))) {
        (core.info(`copilot is already assigned to issue #${issueNumber}`), results.push({ repo: repoSlug, issue_number: issueNumber, success: !0, already_assigned: !0 }));
        continue;
      }
      if ((core.info(`Assigning copilot coding agent to issue #${issueNumber}...`), !(await assignAgentToIssue(issueDetails.issueId, agentId, issueDetails.currentAssignees, "copilot"))))
        throw new Error("Failed to assign copilot via GraphQL");
      (core.info(`Successfully assigned copilot coding agent to issue #${issueNumber}`), results.push({ repo: repoSlug, issue_number: issueNumber, success: !0 }));
    } catch (error) {
      const errorMessage = error instanceof Error ? error.message : String(error);
      (core.error(`Failed to assign copilot to issue #${issueNumber} in ${repoSlug}: ${errorMessage}`), results.push({ repo: repoSlug, issue_number: issueNumber, success: !1, error: errorMessage }));
    }
  }
  const successCount = results.filter(r => r.success).length,
    failureCount = results.filter(r => !r.success).length;
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
    for (const result of results.filter(r => !r.success)) summaryContent += `- ${result.repo}#${result.issue_number}: ${result.error}\n`;
    results.some(r => !r.success && r.error && (r.error.includes("Resource not accessible") || r.error.includes("Insufficient permissions"))) && (summaryContent += generatePermissionErrorSummary());
  }
  (await core.summary.addRaw(summaryContent).write(), failureCount > 0 && core.setFailed(`Failed to assign copilot to ${failureCount} issue(s)`));
}
(async () => {
  await main();
})();
