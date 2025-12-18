const { getAgentName, getIssueDetails, findAgent, assignAgentToIssue } = require("./assign_agent_helpers.cjs");
async function main() {
  const ghToken = process.env.GH_TOKEN,
    assignee = process.env.ASSIGNEE,
    issueNumber = process.env.ISSUE_NUMBER;
  if (!ghToken || "" === ghToken.trim()) {
    const docsUrl = "https://githubnext.github.io/gh-aw/reference/safe-outputs/#assigning-issues-to-copilot";
    return void core.setFailed(`GH_TOKEN environment variable is required but not set. This token is needed to assign issues. For more information on configuring Copilot tokens, see: ${docsUrl}`);
  }
  if (!assignee || "" === assignee.trim()) return void core.setFailed("ASSIGNEE environment variable is required but not set");
  if (!issueNumber || "" === issueNumber.trim()) return void core.setFailed("ISSUE_NUMBER environment variable is required but not set");
  const trimmedAssignee = assignee.trim(),
    trimmedIssueNumber = issueNumber.trim(),
    issueNum = parseInt(trimmedIssueNumber, 10);
  core.info(`Assigning issue #${trimmedIssueNumber} to ${trimmedAssignee}`);
  try {
    const agentName = getAgentName(trimmedAssignee);
    if (agentName) {
      core.info(`Detected coding agent: ${agentName}. Using GraphQL API for assignment.`);
      const owner = context.repo.owner,
        repo = context.repo.repo,
        agentId = await findAgent(owner, repo, agentName);
      if (!agentId) throw new Error(`${agentName} coding agent is not available for this repository`);
      core.info(`Found ${agentName} coding agent (ID: ${agentId})`);
      const issueDetails = await getIssueDetails(owner, repo, issueNum);
      if (!issueDetails) throw new Error("Failed to get issue details");
      if (issueDetails.currentAssignees.includes(agentId)) core.info(`${agentName} is already assigned to issue #${trimmedIssueNumber}`);
      else if (!(await assignAgentToIssue(issueDetails.issueId, agentId, issueDetails.currentAssignees, agentName))) throw new Error(`Failed to assign ${agentName} via GraphQL`);
    } else await exec.exec("gh", ["issue", "edit", trimmedIssueNumber, "--add-assignee", trimmedAssignee], { env: { ...process.env, GH_TOKEN: ghToken } });
    (core.info(`✅ Successfully assigned issue #${trimmedIssueNumber} to ${trimmedAssignee}`), await core.summary.addRaw(`\n## Issue Assignment\n\nSuccessfully assigned issue #${trimmedIssueNumber} to \`${trimmedAssignee}\`.\n`).write());
  } catch (error) {
    const errorMessage = error instanceof Error ? error.message : String(error);
    (core.error(`Failed to assign issue: ${errorMessage}`), core.setFailed(`Failed to assign issue #${trimmedIssueNumber} to ${trimmedAssignee}: ${errorMessage}`));
  }
}
main().catch(error => {
  core.setFailed(error instanceof Error ? error.message : String(error));
});
