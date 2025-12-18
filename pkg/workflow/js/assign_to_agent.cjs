const { loadAgentOutput } = require("./load_agent_output.cjs"),
  { generateStagedPreview } = require("./staged_preview.cjs"),
  { AGENT_LOGIN_NAMES, getAvailableAgentLogins, findAgent, getIssueDetails, assignAgentToIssue, generatePermissionErrorSummary } = require("./assign_agent_helpers.cjs");
async function main() {
  const result = loadAgentOutput();
  if (!result.success) return;
  const assignItems = result.items.filter(item => "assign_to_agent" === item.type);
  if (0 === assignItems.length) return void core.info("No assign_to_agent items found in agent output");
  if ((core.info(`Found ${assignItems.length} assign_to_agent item(s)`), "true" === process.env.GH_AW_SAFE_OUTPUTS_STAGED))
    return void (await generateStagedPreview({
      title: "Assign to Agent",
      description: "The following agent assignments would be made if staged mode was disabled:",
      items: assignItems,
      renderItem: item => {
        let content = `**Issue:** #${item.issue_number}\n`;
        return ((content += `**Agent:** ${item.agent || "copilot"}\n`), (content += "\n"), content);
      },
    }));
  const defaultAgent = process.env.GH_AW_AGENT_DEFAULT?.trim() || "copilot";
  core.info(`Default agent: ${defaultAgent}`);
  const maxCountEnv = process.env.GH_AW_AGENT_MAX_COUNT,
    maxCount = maxCountEnv ? parseInt(maxCountEnv, 10) : 1;
  if (isNaN(maxCount) || maxCount < 1) return void core.setFailed(`Invalid max value: ${maxCountEnv}. Must be a positive integer`);
  core.info(`Max count: ${maxCount}`);
  const itemsToProcess = assignItems.slice(0, maxCount);
  assignItems.length > maxCount && core.warning(`Found ${assignItems.length} agent assignments, but max is ${maxCount}. Processing first ${maxCount}.`);
  const targetRepoEnv = process.env.GH_AW_TARGET_REPO?.trim();
  let targetOwner = context.repo.owner,
    targetRepo = context.repo.repo;
  if (targetRepoEnv) {
    const parts = targetRepoEnv.split("/");
    2 === parts.length
      ? ((targetOwner = parts[0]), (targetRepo = parts[1]), core.info(`Using target repository: ${targetOwner}/${targetRepo}`))
      : core.warning(`Invalid target-repo format: ${targetRepoEnv}. Expected owner/repo. Using current repository.`);
  }
  const agentCache = {},
    results = [];
  for (const item of itemsToProcess) {
    const issueNumber = "number" == typeof item.issue_number ? item.issue_number : parseInt(String(item.issue_number), 10),
      agentName = item.agent || defaultAgent;
    if (isNaN(issueNumber) || issueNumber <= 0) core.error(`Invalid issue_number: ${item.issue_number}`);
    else if (AGENT_LOGIN_NAMES[agentName])
      try {
        let agentId = agentCache[agentName];
        if (!agentId) {
          if ((core.info(`Looking for ${agentName} coding agent...`), (agentId = await findAgent(targetOwner, targetRepo, agentName)), !agentId)) throw new Error(`${agentName} coding agent is not available for this repository`);
          ((agentCache[agentName] = agentId), core.info(`Found ${agentName} coding agent (ID: ${agentId})`));
        }
        core.info("Getting issue details...");
        const issueDetails = await getIssueDetails(targetOwner, targetRepo, issueNumber);
        if (!issueDetails) throw new Error("Failed to get issue details");
        if ((core.info(`Issue ID: ${issueDetails.issueId}`), issueDetails.currentAssignees.includes(agentId))) {
          (core.info(`${agentName} is already assigned to issue #${issueNumber}`), results.push({ issue_number: issueNumber, agent: agentName, success: !0 }));
          continue;
        }
        if ((core.info(`Assigning ${agentName} coding agent to issue #${issueNumber}...`), !(await assignAgentToIssue(issueDetails.issueId, agentId, issueDetails.currentAssignees, agentName))))
          throw new Error(`Failed to assign ${agentName} via GraphQL`);
        (core.info(`Successfully assigned ${agentName} coding agent to issue #${issueNumber}`), results.push({ issue_number: issueNumber, agent: agentName, success: !0 }));
      } catch (error) {
        let errorMessage = error instanceof Error ? error.message : String(error);
        if (errorMessage.includes("coding agent is not available for this repository"))
          try {
            const available = await getAvailableAgentLogins(targetOwner, targetRepo);
            available.length > 0 && (errorMessage += ` (available agents: ${available.join(", ")})`);
          } catch (e) {
            core.debug("Failed to enrich unavailable agent message with available list");
          }
        (core.error(`Failed to assign agent "${agentName}" to issue #${issueNumber}: ${errorMessage}`), results.push({ issue_number: issueNumber, agent: agentName, success: !1, error: errorMessage }));
      }
    else
      (core.warning(`Agent "${agentName}" is not supported. Supported agents: ${Object.keys(AGENT_LOGIN_NAMES).join(", ")}`),
        results.push({ issue_number: issueNumber, agent: agentName, success: !1, error: `Unsupported agent: ${agentName}` }));
  }
  const successCount = results.filter(r => r.success).length,
    failureCount = results.filter(r => !r.success).length;
  let summaryContent = "## Agent Assignment\n\n";
  if (successCount > 0) {
    summaryContent += `✅ Successfully assigned ${successCount} agent(s):\n\n`;
    for (const result of results.filter(r => r.success)) summaryContent += `- Issue #${result.issue_number} → Agent: ${result.agent}\n`;
    summaryContent += "\n";
  }
  if (failureCount > 0) {
    summaryContent += `❌ Failed to assign ${failureCount} agent(s):\n\n`;
    for (const result of results.filter(r => !r.success)) summaryContent += `- Issue #${result.issue_number} → Agent: ${result.agent}: ${result.error}\n`;
    results.some(r => !r.success && r.error && (r.error.includes("Resource not accessible") || r.error.includes("Insufficient permissions"))) && (summaryContent += generatePermissionErrorSummary());
  }
  await core.summary.addRaw(summaryContent).write();
  const assignedAgents = results
    .filter(r => r.success)
    .map(r => `${r.issue_number}:${r.agent}`)
    .join("\n");
  (core.setOutput("assigned_agents", assignedAgents), failureCount > 0 && core.setFailed(`Failed to assign ${failureCount} agent(s)`));
}
(async () => {
  await main();
})();
