// @ts-check
/// <reference types="@actions/github-script" />

/**
 * Shared helper functions for assigning coding agents (like Copilot) to issues
 * These functions use GraphQL to properly assign bot actors that cannot be assigned via gh CLI
 */

const { getOctokitClient, setGetOctokitFactory } = require("./get_octokit_client.cjs");

/**
 * Map agent names to their GitHub bot login names
 * @type {Record<string, string>}
 */
const AGENT_LOGIN_NAMES = {
  copilot: "copilot-swe-agent",
};

/**
 * Check if an assignee is a known coding agent (bot)
 * @param {string} assignee - Assignee name (may include @ prefix)
 * @returns {string|null} Agent name if it's a known agent, null otherwise
 */
function getAgentName(assignee) {
  // Normalize: remove @ prefix if present
  const normalized = assignee.startsWith("@") ? assignee.slice(1) : assignee;

  // Check if it's a known agent
  if (AGENT_LOGIN_NAMES[normalized]) {
    return normalized;
  }

  return null;
}

/**
 * Return list of coding agent bot login names that are currently available as assignable actors
 * (intersection of suggestedActors and known AGENT_LOGIN_NAMES values)
 * @param {string} owner
 * @param {string} repo
 * @param {string} [ghToken] - GitHub token for the query (optional, uses default github object if not provided)
 * @returns {Promise<string[]>}
 */
async function getAvailableAgentLogins(owner, repo, ghToken) {
  const query = `
    query($owner: String!, $repo: String!) {
      repository(owner: $owner, name: $repo) {
        suggestedActors(first: 100, capabilities: CAN_BE_ASSIGNED) {
          nodes { ... on Bot { login __typename } }
        }
      }
    }
  `;
  try {
    // Use Octokit client with custom token if provided, otherwise use default github object
    const client = ghToken ? getOctokitClient(ghToken) : github;
    const response = await client.graphql(query, { owner, repo });
    const actors = response.repository?.suggestedActors?.nodes || [];
    const knownValues = Object.values(AGENT_LOGIN_NAMES);
    const available = [];
    for (const actor of actors) {
      if (actor && actor.login && knownValues.includes(actor.login)) {
        available.push(actor.login);
      }
    }
    return available.sort();
  } catch (e) {
    const msg = e instanceof Error ? e.message : String(e);
    core.debug(`Failed to list available agent logins: ${msg}`);
    return [];
  }
}

/**
 * Find an agent in repository's suggested actors using GraphQL
 * @param {string} owner - Repository owner
 * @param {string} repo - Repository name
 * @param {string} agentName - Agent name (copilot)
 * @param {string} [ghToken] - GitHub token for the query (optional, uses default github object if not provided)
 * @returns {Promise<string|null>} Agent ID or null if not found
 */
async function findAgent(owner, repo, agentName, ghToken) {
  const query = `
    query($owner: String!, $repo: String!) {
      repository(owner: $owner, name: $repo) {
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
    // Use Octokit client with custom token if provided, otherwise use default github object
    const client = ghToken ? getOctokitClient(ghToken) : github;
    const response = await client.graphql(query, { owner, repo });
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

    const available = actors.filter(a => a && a.login && Object.values(AGENT_LOGIN_NAMES).includes(a.login)).map(a => a.login);

    core.warning(`${agentName} coding agent (${loginName}) is not available as an assignee for this repository`);
    if (available.length > 0) {
      core.info(`Available assignable coding agents: ${available.join(", ")}`);
    } else {
      core.info("No coding agents are currently assignable in this repository.");
    }
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
    query($owner: String!, $repo: String!, $issueNumber: Int!) {
      repository(owner: $owner, name: $repo) {
        issue(number: $issueNumber) {
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
    const response = await github.graphql(query, { owner, repo, issueNumber });
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
 * @param {string} ghToken - GitHub token for the mutation. Must have:
 *   - Write actions/contents/issues/pull-requests permissions
 *   - A classic PAT with 'repo' scope OR fine-grained PAT with explicit Write permissions
 *   - Note: The token source varies by caller:
 *     - assign_to_agent.cjs uses GH_AW_AGENT_TOKEN (agent-specific token)
 *     - assign_issue.cjs uses GH_TOKEN (general issue assignment token)
 * @returns {Promise<boolean>} True if successful
 */
async function assignAgentToIssue(issueId, agentId, currentAssignees, agentName, ghToken) {
  // Build actor IDs array - include agent and preserve other assignees
  const actorIds = [agentId];
  for (const assigneeId of currentAssignees) {
    if (assigneeId !== agentId) {
      actorIds.push(assigneeId);
    }
  }

  const mutation = `
    mutation($assignableId: ID!, $actorIds: [ID!]!) {
      replaceActorsForAssignable(input: {
        assignableId: $assignableId,
        actorIds: $actorIds
      }) {
        __typename
      }
    }
  `;

  try {
    // SECURITY: Use provided token for the mutation
    // The mutation requires: Write actions/contents/issues/pull-requests
    if (!ghToken) {
      core.error("GitHub token is not set. Cannot perform assignment mutation.");
      return false;
    }
    core.info("Using provided GitHub token for mutation");

    // Make raw GraphQL request with custom token using variables
    core.debug(`GraphQL mutation with variables: assignableId=${issueId}, actorIds=${JSON.stringify(actorIds)}`);
    const response = await fetch("https://api.github.com/graphql", {
      method: "POST",
      headers: {
        Authorization: `Bearer ${ghToken}`,
        "Content-Type": "application/json",
      },
      body: JSON.stringify({
        query: mutation,
        variables: {
          assignableId: issueId,
          actorIds: actorIds,
        },
      }),
    }).then(res => res.json());

    if (response.errors && response.errors.length > 0) {
      throw new Error(response.errors[0].message);
    }

    if (response.data && response.data.replaceActorsForAssignable && response.data.replaceActorsForAssignable.__typename) {
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
        if (serialized && serialized !== "{}") {
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
        // SECURITY: Use same token for fallback mutation with GraphQL variables
        const fallbackMutation = `
          mutation($assignableId: ID!, $assigneeIds: [ID!]!) {
            addAssigneesToAssignable(input: {
              assignableId: $assignableId,
              assigneeIds: $assigneeIds
            }) {
              clientMutationId
            }
          }
        `;
        if (!ghToken) {
          core.error("GitHub token is not set. Cannot perform fallback mutation.");
        } else {
          core.info("Using provided GitHub token for fallback mutation");
          core.debug(`Fallback GraphQL mutation with variables: assignableId=${issueId}, assigneeIds=[${agentId}]`);
          const fallbackResp = await fetch("https://api.github.com/graphql", {
            method: "POST",
            headers: {
              Authorization: `Bearer ${ghToken}`,
              "Content-Type": "application/json",
            },
            body: JSON.stringify({
              query: fallbackMutation,
              variables: {
                assignableId: issueId,
                assigneeIds: [agentId],
              },
            }),
          }).then(res => res.json());
          if (fallbackResp.data && fallbackResp.data.addAssigneesToAssignable) {
            core.info(`Fallback succeeded: agent '${agentName}' added via addAssigneesToAssignable.`);
            return true;
          } else {
            core.warning("Fallback mutation returned unexpected response; proceeding with permission guidance.");
          }
        }
      } catch (fallbackError) {
        const fbMsg = fallbackError instanceof Error ? fallbackError.message : String(fallbackError);
        core.error(`Fallback addAssigneesToAssignable failed: ${fbMsg}`);
      }
      logPermissionError(agentName);
    } else {
      core.error(`Failed to assign ${agentName}: ${errorMessage}`);
    }
    return false;
  }
}

/**
 * Log detailed permission error guidance
 * @param {string} agentName - Agent name for error messages
 */
function logPermissionError(agentName) {
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
}

/**
 * Generate permission error summary content for step summary
 * @returns {string} Markdown content for permission error guidance
 */
function generatePermissionErrorSummary() {
  let content = "\n### ⚠️ Permission Requirements\n\n";
  content += "Assigning Copilot agents requires **ALL** of these permissions:\n\n";
  content += "```yaml\n";
  content += "permissions:\n";
  content += "  actions: write\n";
  content += "  contents: write\n";
  content += "  issues: write\n";
  content += "  pull-requests: write\n";
  content += "```\n\n";
  content += "**Token capability note:**\n";
  content += "- Current token (PAT or GITHUB_TOKEN) lacks assignee mutation capability for this repository.\n";
  content += "- Both `replaceActorsForAssignable` and fallback `addAssigneesToAssignable` returned FORBIDDEN/Resource not accessible.\n";
  content += "- This typically means bot/user assignment requires an elevated OAuth or GitHub App installation token.\n\n";
  content += "**Recommended remediation paths:**\n";
  content += "1. Create & install a GitHub App with: Issues/Pull requests/Contents/Actions (write) → use installation token in job.\n";
  content += "2. Manual assignment: add the agent through the UI until broader token support is available.\n";
  content += "3. Open a support ticket referencing failing mutation `replaceActorsForAssignable` and repository slug.\n\n";
  content +=
    "**Why this failed:** Fine-grained and classic PATs can update issue title (verified) but not modify assignees in this environment.\n\n";
  content += "📖 Reference: https://docs.github.com/en/copilot/how-tos/use-copilot-agents/coding-agent/create-a-pr (general agent docs)\n";
  return content;
}

/**
 * Assign an agent to an issue using GraphQL
 * This is the main entry point for assigning agents from other scripts
 * @param {string} owner - Repository owner
 * @param {string} repo - Repository name
 * @param {number} issueNumber - Issue number
 * @param {string} agentName - Agent name (e.g., "copilot")
 * @param {string} ghToken - GitHub token for the mutation. Must have sufficient permissions
 *   to assign agents. See assignAgentToIssue() for token requirements.
 * @returns {Promise<{success: boolean, error?: string}>}
 */
async function assignAgentToIssueByName(owner, repo, issueNumber, agentName, ghToken) {
  // Check if agent is supported
  if (!AGENT_LOGIN_NAMES[agentName]) {
    const error = `Agent "${agentName}" is not supported. Supported agents: ${Object.keys(AGENT_LOGIN_NAMES).join(", ")}`;
    core.warning(error);
    return { success: false, error };
  }

  try {
    // Find agent - use the provided token for the GraphQL query
    core.info(`Looking for ${agentName} coding agent...`);
    const agentId = await findAgent(owner, repo, agentName, ghToken);
    if (!agentId) {
      const error = `${agentName} coding agent is not available for this repository`;
      // Enrich with available agent logins - also use the provided token
      const available = await getAvailableAgentLogins(owner, repo, ghToken);
      const enrichedError = available.length > 0 ? `${error} (available agents: ${available.join(", ")})` : error;
      return { success: false, error: enrichedError };
    }
    core.info(`Found ${agentName} coding agent (ID: ${agentId})`);

    // Get issue details (ID and current assignees) via GraphQL
    core.info("Getting issue details...");
    const issueDetails = await getIssueDetails(owner, repo, issueNumber);
    if (!issueDetails) {
      return { success: false, error: "Failed to get issue details" };
    }

    core.info(`Issue ID: ${issueDetails.issueId}`);

    // Check if agent is already assigned
    if (issueDetails.currentAssignees.includes(agentId)) {
      core.info(`${agentName} is already assigned to issue #${issueNumber}`);
      return { success: true };
    }

    // Assign agent using GraphQL mutation
    core.info(`Assigning ${agentName} coding agent to issue #${issueNumber}...`);
    const success = await assignAgentToIssue(issueDetails.issueId, agentId, issueDetails.currentAssignees, agentName, ghToken);

    if (!success) {
      return { success: false, error: `Failed to assign ${agentName} via GraphQL` };
    }

    core.info(`Successfully assigned ${agentName} coding agent to issue #${issueNumber}`);
    return { success: true };
  } catch (error) {
    const errorMessage = error instanceof Error ? error.message : String(error);
    return { success: false, error: errorMessage };
  }
}

module.exports = {
  AGENT_LOGIN_NAMES,
  getAgentName,
  getAvailableAgentLogins,
  findAgent,
  getIssueDetails,
  assignAgentToIssue,
  logPermissionError,
  generatePermissionErrorSummary,
  assignAgentToIssueByName,
  setGetOctokitFactory, // Exposed for testing (re-exported from get_octokit_client.cjs)
};
