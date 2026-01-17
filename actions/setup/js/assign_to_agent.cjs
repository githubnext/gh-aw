// @ts-check
/// <reference types="@actions/github-script" />

const { loadAgentOutput } = require("./load_agent_output.cjs");
const { generateStagedPreview } = require("./staged_preview.cjs");
const { AGENT_LOGIN_NAMES, getAvailableAgentLogins, findAgent, getIssueDetails, getPullRequestDetails, assignAgentToIssue, generatePermissionErrorSummary } = require("./assign_agent_helpers.cjs");
const { getErrorMessage } = require("./error_helpers.cjs");
const { resolveTarget } = require("./safe_output_helpers.cjs");

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
        let content = "";
        if (item.issue_number) {
          content += `**Issue:** #${item.issue_number}\n`;
        } else if (item.pull_number) {
          content += `**Pull Request:** #${item.pull_number}\n`;
        }
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

  // Get target configuration (defaults to "triggering")
  const targetConfig = process.env.GH_AW_AGENT_TARGET?.trim() || "triggering";
  core.info(`Target configuration: ${targetConfig}`);

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
    const agentName = item.agent ?? defaultAgent;

    // Resolve target number using the same logic as other safe outputs
    // This allows automatic resolution from workflow context when issue_number/pull_number is not explicitly provided
    const targetResult = resolveTarget({
      targetConfig,
      item,
      context,
      itemType: "assign_to_agent",
      supportsPR: true, // Supports both issues and PRs
      supportsIssue: false, // Use supportsPR=true to indicate both are supported
    });

    if (!targetResult.success) {
      if (targetResult.shouldFail) {
        core.error(targetResult.error);
        results.push({
          issue_number: item.issue_number || null,
          pull_number: item.pull_number || null,
          agent: agentName,
          success: false,
          error: targetResult.error,
        });
      } else {
        // Just skip this item (e.g., wrong event type for "triggering" target)
        core.info(targetResult.error);
      }
      continue;
    }

    const number = targetResult.number;
    const type = targetResult.contextType;
    const issueNumber = type === "issue" ? number : null;
    const pullNumber = type === "pull request" ? number : null;

    if (isNaN(number) || number <= 0) {
      core.error(`Invalid ${type} number: ${number}`);
      results.push({
        issue_number: issueNumber,
        pull_number: pullNumber,
        agent: agentName,
        success: false,
        error: `Invalid ${type} number: ${number}`,
      });
      continue;
    }

    // Check if agent is supported
    if (!AGENT_LOGIN_NAMES[agentName]) {
      core.warning(`Agent "${agentName}" is not supported. Supported agents: ${Object.keys(AGENT_LOGIN_NAMES).join(", ")}`);
      results.push({
        issue_number: issueNumber,
        pull_number: pullNumber,
        agent: agentName,
        success: false,
        error: `Unsupported agent: ${agentName}`,
      });
      continue;
    }

    // Assign the agent to the issue or PR using GraphQL
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

      // Get issue or PR details (ID and current assignees) via GraphQL
      core.info(`Getting ${type} details...`);
      let assignableId;
      let currentAssignees;

      if (issueNumber) {
        const issueDetails = await getIssueDetails(targetOwner, targetRepo, issueNumber);
        if (!issueDetails) {
          throw new Error(`Failed to get issue details`);
        }
        assignableId = issueDetails.issueId;
        currentAssignees = issueDetails.currentAssignees;
      } else {
        const prDetails = await getPullRequestDetails(targetOwner, targetRepo, pullNumber);
        if (!prDetails) {
          throw new Error(`Failed to get pull request details`);
        }
        assignableId = prDetails.pullRequestId;
        currentAssignees = prDetails.currentAssignees;
      }

      core.info(`${type} ID: ${assignableId}`);

      // Check if agent is already assigned
      if (currentAssignees.includes(agentId)) {
        core.info(`${agentName} is already assigned to ${type} #${number}`);
        results.push({
          issue_number: issueNumber,
          pull_number: pullNumber,
          agent: agentName,
          success: true,
        });
        continue;
      }

      // Assign agent using GraphQL mutation - uses built-in github object authenticated via github-token
      core.info(`Assigning ${agentName} coding agent to ${type} #${number}...`);
      const success = await assignAgentToIssue(assignableId, agentId, currentAssignees, agentName);

      if (!success) {
        throw new Error(`Failed to assign ${agentName} via GraphQL`);
      }

      core.info(`Successfully assigned ${agentName} coding agent to ${type} #${number}`);
      results.push({
        issue_number: issueNumber,
        pull_number: pullNumber,
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
      core.error(`Failed to assign agent "${agentName}" to ${type} #${number}: ${errorMessage}`);
      results.push({
        issue_number: issueNumber,
        pull_number: pullNumber,
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
      .map(r => {
        const itemType = r.issue_number ? `Issue #${r.issue_number}` : `Pull Request #${r.pull_number}`;
        return `- ${itemType} → Agent: ${r.agent}`;
      })
      .join("\n");
    summaryContent += "\n\n";
  }

  if (failureCount > 0) {
    summaryContent += `❌ Failed to assign ${failureCount} agent(s):\n\n`;
    summaryContent += results
      .filter(r => !r.success)
      .map(r => {
        const itemType = r.issue_number ? `Issue #${r.issue_number}` : `Pull Request #${r.pull_number}`;
        return `- ${itemType} → Agent: ${r.agent}: ${r.error}`;
      })
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
    .map(r => {
      const number = r.issue_number || r.pull_number;
      const prefix = r.issue_number ? "issue" : "pr";
      return `${prefix}:${number}:${r.agent}`;
    })
    .join("\n");
  core.setOutput("assigned_agents", assignedAgents);

  // Fail if any assignments failed
  if (failureCount > 0) {
    core.setFailed(`Failed to assign ${failureCount} agent(s)`);
  }
}

module.exports = { main };
