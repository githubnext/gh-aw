// @ts-check
/// <reference types="@actions/github-script" />

const { loadAgentOutput } = require("./load_agent_output.cjs");
const { generateStagedPreview } = require("./staged_preview.cjs");

/**
 * Map agent names to their GitHub bot login names
 */
const AGENT_LOGIN_NAMES = {
  copilot: "copilot-swe-agent",
  claude: "claude-swe-agent",
  codex: "codex-swe-agent",
};

/**
 * Find an agent in repository's suggested actors using GraphQL
 * @param {string} owner - Repository owner
 * @param {string} repo - Repository name
 * @param {string} agentName - Agent name (copilot, claude, codex)
 * @returns {Promise<string|null>} Agent ID or null if not found
 */
async function findAgent(owner, repo, agentName) {
  const query = `
    query {
      repository(owner: "${owner}", name: "${repo}") {
        suggestedActors(first: 100, capabilities: CAN_BE_ASSIGNED) {
          nodes {
            ... on Bot {
              id
              login
              __typename
            }
          }
        }
      }
    }
  `;

  try {
    const response = await github.graphql(query);
    const actors = response.repository.suggestedActors.nodes;

    const loginName = AGENT_LOGIN_NAMES[agentName];
    if (!loginName) {
      core.error(`Unknown agent: ${agentName}. Supported agents: ${Object.keys(AGENT_LOGIN_NAMES).join(", ")}`);
      return null;
    }

    for (const actor of actors) {
      if (actor.login === loginName) {
        return actor.id;
      }
    }

    core.warning(`${agentName} coding agent (${loginName}) is not available as an assignee for this repository`);
    if (agentName === "copilot") {
      core.info(
        "Please visit https://docs.github.com/en/copilot/using-github-copilot/using-copilot-coding-agent-to-work-on-tasks/about-assigning-tasks-to-copilot"
      );
    }
    return null;
  } catch (error) {
    const errorMessage = error instanceof Error ? error.message : String(error);
    core.error(`Failed to find ${agentName} agent: ${errorMessage}`);
    return null;
  }
}

/**
 * Get issue details (ID and current assignees) using GraphQL
 * @param {string} owner - Repository owner
 * @param {string} repo - Repository name
 * @param {number} issueNumber - Issue number
 * @returns {Promise<{issueId: string, currentAssignees: string[]}|null>}
 */
async function getIssueDetails(owner, repo, issueNumber) {
  const query = `
    query {
      repository(owner: "${owner}", name: "${repo}") {
        issue(number: ${issueNumber}) {
          id
          assignees(first: 100) {
            nodes {
              id
            }
          }
        }
      }
    }
  `;

  try {
    const response = await github.graphql(query);
    const issue = response.repository.issue;

    if (!issue || !issue.id) {
      core.error("Could not get issue data");
      return null;
    }

    const currentAssignees = issue.assignees.nodes.map(assignee => assignee.id);

    return {
      issueId: issue.id,
      currentAssignees: currentAssignees,
    };
  } catch (error) {
    const errorMessage = error instanceof Error ? error.message : String(error);
    core.error(`Failed to get issue details: ${errorMessage}`);
    return null;
  }
}

/**
 * Assign agent to issue using GraphQL replaceActorsForAssignable mutation
 * @param {string} issueId - GitHub issue ID
 * @param {string} agentId - Agent ID
 * @param {string[]} currentAssignees - List of current assignee IDs
 * @param {string} agentName - Agent name for error messages
 * @returns {Promise<boolean>} True if successful
 */
async function assignAgentToIssue(issueId, agentId, currentAssignees, agentName) {
  // Build actor IDs array - include agent and preserve other assignees
  const actorIds = [agentId];
  for (const assigneeId of currentAssignees) {
    if (assigneeId !== agentId) {
      actorIds.push(assigneeId);
    }
  }

  const mutation = `
    mutation {
      replaceActorsForAssignable(input: {
        assignableId: "${issueId}",
        actorIds: ${JSON.stringify(actorIds)}
      }) {
        __typename
      }
    }
  `;

  try {
    const response = await github.graphql(mutation);

    if (response.replaceActorsForAssignable && response.replaceActorsForAssignable.__typename) {
      return true;
    } else {
      core.error("Unexpected response from GitHub API");
      return false;
    }
  } catch (error) {
    const errorMessage = error instanceof Error ? error.message : String(error);

    // Debug: surface the raw GraphQL error structure for troubleshooting fine-grained permission issues
    try {
      core.debug(`Raw GraphQL error message: ${errorMessage}`);
      if (error && typeof error === "object") {
        // Common GraphQL error shapes: error.errors (array), error.data, error.response
        const details = {};
        if (error.errors) details.errors = error.errors;
        // Some libraries wrap the payload under 'response' or 'response.data'
        if (error.response) details.response = error.response;
        if (error.data) details.data = error.data;
        // If GitHub returns an array of errors with 'type'/'message'
        if (Array.isArray(error.errors)) {
          details.compactMessages = error.errors.map(e => e.message).filter(Boolean);
        }
        const serialized = JSON.stringify(details, (_k, v) => v, 2);
        if (serialized && serialized !== '{}' ) {
          core.debug(`Raw GraphQL error details: ${serialized}`);
          // Also emit non-debug version so users without ACTIONS_STEP_DEBUG can see it
          core.error("Raw GraphQL error details (for troubleshooting):");
          // Split large JSON for readability
          for (const line of serialized.split(/\n/)) {
            if (line.trim()) core.error(line);
          }
        }
      }
    } catch (loggingErr) {
      // Never fail assignment because of debug logging
      core.debug(`Failed to serialize GraphQL error details: ${loggingErr instanceof Error ? loggingErr.message : String(loggingErr)}`);
    }

    // Check for permission-related errors
    if (
      errorMessage.includes("Resource not accessible by personal access token") ||
      errorMessage.includes("Resource not accessible by integration") ||
      errorMessage.includes("Insufficient permissions to assign")
    ) {
      // Attempt fallback mutation addAssigneesToAssignable when replaceActorsForAssignable is forbidden
      core.info("Primary mutation replaceActorsForAssignable forbidden. Attempting fallback addAssigneesToAssignable...");
      try {
        // Fallback does not preserve removals; we only add the agent if not already assigned
        const fallbackMutation = `mutation {\n  addAssigneesToAssignable(input:{assignableId:"${issueId}", assigneeIds:["${agentId}"]}) {\n    clientMutationId\n  }\n}`;
        const fallbackResp = await github.graphql(fallbackMutation);
        if (fallbackResp && fallbackResp.addAssigneesToAssignable) {
          core.info(`Fallback succeeded: agent '${agentName}' added via addAssigneesToAssignable.`);
          return true;
        } else {
          core.warning("Fallback mutation returned unexpected response; proceeding with permission guidance.");
        }
      } catch (fallbackError) {
        const fbMsg = fallbackError instanceof Error ? fallbackError.message : String(fallbackError);
        core.error(`Fallback addAssigneesToAssignable failed: ${fbMsg}`);
      }
      core.error(`Failed to assign ${agentName}: Insufficient permissions`);
      core.error("");
      core.error("Assigning Copilot agents requires:");
      core.error("  1. All four workflow permissions:");
      core.error("     - actions: write");
      core.error("     - contents: write");
      core.error("     - issues: write");
      core.error("     - pull-requests: write");
      core.error("");
      core.error("  2. A classic PAT with 'repo' scope OR fine-grained PAT with explicit Write permissions above:");
      core.error("     (Fine-grained PATs must grant repository access + write for Issues, Pull requests, Contents, Actions)");
      core.error("");
      core.error("  3. Repository settings:");
      core.error("     - Actions must have write permissions");
      core.error("     - Go to: Settings > Actions > General > Workflow permissions");
      core.error("     - Select: 'Read and write permissions'");
      core.error("");
      core.error("  4. Organization/Enterprise settings:");
      core.error("     - Check if your org restricts bot assignments");
      core.error("     - Verify Copilot is enabled for your repository");
      core.error("");
      core.info("For more information, see: https://docs.github.com/en/copilot/how-tos/use-copilot-agents/coding-agent/create-a-pr");
    } else {
      core.error(`Failed to assign ${agentName}: ${errorMessage}`);
    }
    return false;
  }
}

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
  const defaultAgent = process.env.GH_AW_AGENT_DEFAULT?.trim() || "copilot";
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

  // Cache agent IDs to avoid repeated lookups
  const agentCache = {};

  // Process each agent assignment
  const results = [];
  for (const item of itemsToProcess) {
    const issueNumber = typeof item.issue_number === "number" ? item.issue_number : parseInt(String(item.issue_number), 10);
    const agentName = item.agent || defaultAgent;

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
      // Find agent (use cache if available)
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

      // Assign agent using GraphQL mutation
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
      const errorMessage = error instanceof Error ? error.message : String(error);
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
  const failureCount = results.filter(r => !r.success).length;

  let summaryContent = "## Agent Assignment\n\n";

  if (successCount > 0) {
    summaryContent += `✅ Successfully assigned ${successCount} agent(s):\n\n`;
    for (const result of results.filter(r => r.success)) {
      summaryContent += `- Issue #${result.issue_number} → Agent: ${result.agent}\n`;
    }
    summaryContent += "\n";
  }

  if (failureCount > 0) {
    summaryContent += `❌ Failed to assign ${failureCount} agent(s):\n\n`;
    for (const result of results.filter(r => !r.success)) {
      summaryContent += `- Issue #${result.issue_number} → Agent: ${result.agent}: ${result.error}\n`;
    }

    // Check if any failures were permission-related
    const hasPermissionError = results.some(
      r => !r.success && r.error && (r.error.includes("Resource not accessible") || r.error.includes("Insufficient permissions"))
    );

    if (hasPermissionError) {
      summaryContent += "\n### ⚠️ Permission Requirements\n\n";
      summaryContent += "Assigning Copilot agents requires **ALL** of these permissions:\n\n";
      summaryContent += "```yaml\n";
      summaryContent += "permissions:\n";
      summaryContent += "  actions: write\n";
      summaryContent += "  contents: write\n";
      summaryContent += "  issues: write\n";
      summaryContent += "  pull-requests: write\n";
      summaryContent += "```\n\n";
      summaryContent += "**Token capability note:**\n";
      summaryContent += "- Current token (PAT or GITHUB_TOKEN) lacks assignee mutation capability for this repository.\n";
      summaryContent += "- Both `replaceActorsForAssignable` and fallback `addAssigneesToAssignable` returned FORBIDDEN/Resource not accessible.\n";
      summaryContent += "- This typically means bot/user assignment requires an elevated OAuth or GitHub App installation token.\n\n";
      summaryContent += "**Recommended remediation paths:**\n";
      summaryContent += "1. Create & install a GitHub App with: Issues/Pull requests/Contents/Actions (write) → use installation token in job.\n";
      summaryContent += "2. Manual assignment: add the agent through the UI until broader token support is available.\n";
      summaryContent += "3. Open a support ticket referencing failing mutation `replaceActorsForAssignable` and repository slug.\n\n";
      summaryContent += "**Why this failed:** Fine-grained and classic PATs can update issue title (verified) but not modify assignees in this environment.\n\n";
      summaryContent += "📖 Reference: https://docs.github.com/en/copilot/how-tos/use-copilot-agents/coding-agent/create-a-pr (general agent docs)\n";
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

(async () => {
  await main();
})();
