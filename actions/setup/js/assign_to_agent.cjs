// @ts-check
/// <reference types="@actions/github-script" />

const { loadAgentOutput } = require("./load_agent_output.cjs");
const { generateStagedPreview } = require("./staged_preview.cjs");
const { AGENT_LOGIN_NAMES, getAvailableAgentLogins, findAgent, getIssueDetails, assignAgentToIssue, generatePermissionErrorSummary } = require("./assign_agent_helpers.cjs");
const { getErrorMessage } = require("./error_helpers.cjs");

async function main() {
  const result = loadAgentOutput();
  if (!result.success) {
    return;
  }

  const assignItems = result.items.filter(item => item.type === "assign_to_agent");
  if (assignItems.length === 0) {
    core.info("No assign_to_agent items found in agent output");
    return;
  }

  core.info(`Found ${assignItems.length} assign_to_agent item(s)`);

  // Check if we're in staged mode
  if (process.env.GH_AW_SAFE_OUTPUTS_STAGED === "true") {
    await generateStagedPreview({
      title: "Assign to Agent",
      description: "The following agent assignments would be made if staged mode was disabled:",
      items: assignItems,
      renderItem: item => {
        let content = `**Issue:** #${item.issue_number}\n`;
        content += `**Agent:** ${item.agent || "copilot"}\n`;
        content += "\n";
        return content;
      },
    });
    return;
  }

  // Get default agent from configuration
  const defaultAgent = process.env.GH_AW_AGENT_DEFAULT?.trim() ?? "copilot";
  core.info(`Default agent: ${defaultAgent}`);

  // Get max count configuration
  const maxCountEnv = process.env.GH_AW_AGENT_MAX_COUNT;
  const maxCount = maxCountEnv ? parseInt(maxCountEnv, 10) : 1;
  if (isNaN(maxCount) || maxCount < 1) {
    core.setFailed(`Invalid max value: ${maxCountEnv}. Must be a positive integer`);
    return;
  }
  core.info(`Max count: ${maxCount}`);

  // Limit items to max count
  const itemsToProcess = assignItems.slice(0, maxCount);
  if (assignItems.length > maxCount) {
    core.warning(`Found ${assignItems.length} agent assignments, but max is ${maxCount}. Processing first ${maxCount}.`);
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

  // The github-token is set at the step level, so the built-in github object is authenticated
  // with the correct token (GH_AW_AGENT_TOKEN by default)

  // Cache agent IDs to avoid repeated lookups
  const agentCache = {};

  // Process each agent assignment
  const results = [];
  for (const item of itemsToProcess) {
    const issueNumber = typeof item.issue_number === "number" ? item.issue_number : parseInt(String(item.issue_number), 10);
    const agentName = item.agent ?? defaultAgent;

    if (isNaN(issueNumber) || issueNumber <= 0) {
      core.error(`Invalid issue_number: ${item.issue_number}`);
      continue;
    }

    // Check if agent is supported
    if (!AGENT_LOGIN_NAMES[agentName]) {
      core.warning(`Agent "${agentName}" is not supported. Supported agents: ${Object.keys(AGENT_LOGIN_NAMES).join(", ")}`);
      results.push({
        issue_number: issueNumber,
        agent: agentName,
        success: false,
        error: `Unsupported agent: ${agentName}`,
      });
      continue;
    }

    // Assign the agent to the issue using GraphQL
    try {
      // Find agent (use cache if available) - uses built-in github object authenticated via github-token
      let agentId = agentCache[agentName];
      if (!agentId) {
        core.info(`Looking for ${agentName} coding agent...`);
        agentId = await findAgent(targetOwner, targetRepo, agentName);
        if (!agentId) {
          throw new Error(`${agentName} coding agent is not available for this repository`);
        }
        agentCache[agentName] = agentId;
        core.info(`Found ${agentName} coding agent (ID: ${agentId})`);
      }

      // Get issue details (ID and current assignees) via GraphQL
      core.info("Getting issue details...");
      const issueDetails = await getIssueDetails(targetOwner, targetRepo, issueNumber);
      if (!issueDetails) {
        throw new Error("Failed to get issue details");
      }

      core.info(`Issue ID: ${issueDetails.issueId}`);

      // Check if agent is already assigned
      if (issueDetails.currentAssignees.includes(agentId)) {
        core.info(`${agentName} is already assigned to issue #${issueNumber}`);
        results.push({
          issue_number: issueNumber,
          agent: agentName,
          success: true,
        });
        continue;
      }

      // Assign agent using GraphQL mutation - uses built-in github object authenticated via github-token
      core.info(`Assigning ${agentName} coding agent to issue #${issueNumber}...`);
      const success = await assignAgentToIssue(issueDetails.issueId, agentId, issueDetails.currentAssignees, agentName);

      if (!success) {
        throw new Error(`Failed to assign ${agentName} via GraphQL`);
      }

      core.info(`Successfully assigned ${agentName} coding agent to issue #${issueNumber}`);
      results.push({
        issue_number: issueNumber,
        agent: agentName,
        success: true,
      });
    } catch (error) {
      let errorMessage = getErrorMessage(error);
      if (errorMessage.includes("coding agent is not available for this repository")) {
        // Enrich with available agent logins to aid troubleshooting - uses built-in github object
        try {
          const available = await getAvailableAgentLogins(targetOwner, targetRepo);
          if (available.length > 0) {
            errorMessage += ` (available agents: ${available.join(", ")})`;
          }
        } catch (e) {
          core.debug("Failed to enrich unavailable agent message with available list");
        }
      }
      core.error(`Failed to assign agent "${agentName}" to issue #${issueNumber}: ${errorMessage}`);
      results.push({
        issue_number: issueNumber,
        agent: agentName,
        success: false,
        error: errorMessage,
      });
    }
  }

  // Generate step summary
  const successCount = results.filter(r => r.success).length;
  const failureCount = results.length - successCount;

  let summaryContent = "## Agent Assignment\n\n";

  if (successCount > 0) {
    summaryContent += `✅ Successfully assigned ${successCount} agent(s):\n\n`;
    summaryContent += results
      .filter(r => r.success)
      .map(r => `- Issue #${r.issue_number} → Agent: ${r.agent}`)
      .join("\n");
    summaryContent += "\n\n";
  }

  if (failureCount > 0) {
    summaryContent += `❌ Failed to assign ${failureCount} agent(s):\n\n`;
    summaryContent += results
      .filter(r => !r.success)
      .map(r => `- Issue #${r.issue_number} → Agent: ${r.agent}: ${r.error}`)
      .join("\n");

    // Check if any failures were permission-related
    const hasPermissionError = results.some(r => (!r.success && r.error?.includes("Resource not accessible")) || r.error?.includes("Insufficient permissions"));

    if (hasPermissionError) {
      summaryContent += generatePermissionErrorSummary();
    }
  }

  await core.summary.addRaw(summaryContent).write();

  // Set outputs
  const assignedAgents = results
    .filter(r => r.success)
    .map(r => `${r.issue_number}:${r.agent}`)
    .join("\n");
  core.setOutput("assigned_agents", assignedAgents);

  // Fail if any assignments failed
  if (failureCount > 0) {
    core.setFailed(`Failed to assign ${failureCount} agent(s)`);
  }
}

module.exports = { main };
