// @ts-check
/// <reference types="@actions/github-script" />

/**
 * Shared helper functions for assigning coding agents (like Copilot) to issues
 * Uses the REST API (December 2025) to assign Copilot to issues with advanced options.
 * Falls back to GraphQL when REST fails or for advanced options.
 *
 * Reference: https://github.blog/changelog/2025-12-03-assign-issues-to-copilot-using-the-api/
 *
 * NOTE: All functions use the built-in `github` global object for authentication.
 * The token must be set at the step level via the `github-token` parameter in GitHub Actions.
 * This approach is required for compatibility with actions/github-script@v8.
 */

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
 * @returns {Promise<string[]>}
 */
async function getAvailableAgentLogins(owner, repo) {
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
    const response = await github.graphql(query, { owner, repo });
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
 * Get repository ID from owner/repo format
 * @param {string} owner - Repository owner
 * @param {string} repo - Repository name
 * @returns {Promise<string|null>} Repository ID or null if not found
 */
async function getRepositoryId(owner, repo) {
  const query = `
    query($owner: String!, $repo: String!) {
      repository(owner: $owner, name: $repo) {
        id
      }
    }
  `;

  try {
    const response = await github.graphql(query, { owner, repo });
    return response.repository?.id || null;
  } catch (error) {
    const errorMessage = error instanceof Error ? error.message : String(error);
    core.error(`Failed to get repository ID for ${owner}/${repo}: ${errorMessage}`);
    return null;
  }
}

/**
 * Check if agent is already assigned to an issue using REST API
 * @param {string} owner - Repository owner
 * @param {string} repo - Repository name
 * @param {number} issueNumber - Issue number
 * @param {string} agentName - Agent name
 * @returns {Promise<boolean>}
 */
async function isAgentAlreadyAssigned(owner, repo, issueNumber, agentName) {
  const loginName = AGENT_LOGIN_NAMES[agentName];
  if (!loginName) return false;

  try {
    const response = await github.rest.issues.get({
      owner,
      repo,
      issue_number: issueNumber,
    });

    const assignees = response.data.assignees || [];
    return assignees.some(a => a.login === loginName);
  } catch (error) {
    core.debug(`Failed to check existing assignees: ${error instanceof Error ? error.message : String(error)}`);
    return false;
  }
}

/**
 * Assign Copilot agent to an issue using the REST API
 * Uses POST /repos/{owner}/{repo}/issues/{issue_number}/assignees endpoint
 *
 * @param {string} owner - Repository owner
 * @param {string} repo - Repository name
 * @param {number} issueNumber - Issue number
 * @param {string} agentName - Agent name (e.g., "copilot")
 * @returns {Promise<{success: boolean, error?: string}>}
 */
async function assignAgentViaRest(owner, repo, issueNumber, agentName) {
  const loginName = AGENT_LOGIN_NAMES[agentName];
  if (!loginName) {
    const error = `Unknown agent: ${agentName}. Supported agents: ${Object.keys(AGENT_LOGIN_NAMES).join(", ")}`;
    core.error(error);
    return { success: false, error };
  }

  try {
    core.info(`Assigning ${agentName} (${loginName}) to issue #${issueNumber} via REST API...`);

    // Use the REST API to add assignees
    // POST /repos/{owner}/{repo}/issues/{issue_number}/assignees
    const response = await github.rest.issues.addAssignees({
      owner,
      repo,
      issue_number: issueNumber,
      assignees: [loginName],
    });

    if (response.status === 201 || response.status === 200) {
      core.info(`✅ Successfully assigned ${agentName} to issue #${issueNumber} via REST API`);
      return { success: true };
    } else {
      const error = `Unexpected response status: ${response.status}`;
      core.error(error);
      return { success: false, error };
    }
  } catch (error) {
    const errorMessage = error instanceof Error ? error.message : String(error);

    // Check for common errors
    if (errorMessage.includes("422") || errorMessage.includes("Validation Failed")) {
      // Copilot not available or not enabled
      core.debug(`REST API 422 error: ${errorMessage}`);
      return { success: false, error: `${agentName} coding agent may not be available for this repository` };
    }

    if (errorMessage.includes("Resource not accessible") || errorMessage.includes("403")) {
      core.debug(`REST API permission error: ${errorMessage}`);
      return { success: false, error: "Insufficient permissions to assign agent via REST API" };
    }

    core.debug(`REST API failed: ${errorMessage}`);
    return { success: false, error: errorMessage };
  }
}

/**
 * Find an agent in repository's suggested actors using GraphQL
 * @param {string} owner - Repository owner
 * @param {string} repo - Repository name
 * @param {string} agentName - Agent name (copilot)
 * @returns {Promise<string|null>} Agent ID or null if not found
 */
async function findAgent(owner, repo, agentName) {
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
    const response = await github.graphql(query, { owner, repo });
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
 * @param {object} options - Additional assignment options
 * @param {string} [options.targetRepositoryId] - Target repository ID for the PR
 * @param {string} [options.baseBranch] - Base branch for the PR
 * @param {string} [options.customInstructions] - Custom instructions for the agent
 * @param {string} [options.customAgent] - Custom agent name/path
 * @returns {Promise<boolean>} True if successful
 */
async function assignAgentToIssue(issueId, agentId, currentAssignees, agentName, options = {}) {
  // Build actor IDs array - include agent and preserve other assignees
  const actorIds = [agentId];
  for (const assigneeId of currentAssignees) {
    if (assigneeId !== agentId) {
      actorIds.push(assigneeId);
    }
  }

  // Check if any Copilot-specific options are provided
  const hasCopilotOptions = options.targetRepositoryId || options.baseBranch || options.customInstructions || options.customAgent;

  try {
    core.info("Using built-in github object for mutation");

    let response;

    if (hasCopilotOptions) {
      // Build Copilot assignment options
      const copilotOptions = {};

      if (options.targetRepositoryId) {
        copilotOptions.targetRepositoryId = options.targetRepositoryId;
      }

      if (options.baseBranch) {
        copilotOptions.baseBranch = options.baseBranch;
      }

      if (options.customInstructions) {
        copilotOptions.customInstructions = options.customInstructions;
      }

      if (options.customAgent) {
        copilotOptions.customAgent = options.customAgent;
      }

      // Use extended mutation with Copilot assignment options
      const extendedMutation = `
        mutation($assignableId: ID!, $actorIds: [ID!]!, $copilotAssignmentOptions: CopilotAssignmentOptionsInput) {
          replaceActorsForAssignable(input: {
            assignableId: $assignableId,
            actorIds: $actorIds,
            copilotAssignmentOptions: $copilotAssignmentOptions
          }) {
            __typename
          }
        }
      `;

      const mutationInput = {
        assignableId: issueId,
        actorIds: actorIds,
        copilotAssignmentOptions: copilotOptions,
      };

      core.debug(`GraphQL mutation with Copilot options: ${JSON.stringify(mutationInput)}`);
      response = await github.graphql(extendedMutation, mutationInput, {
        headers: {
          "GraphQL-Features": "issues_copilot_assignment_api_support",
        },
      });
    } else {
      // Use simple mutation for backward compatibility (no Copilot-specific options)
      const simpleMutation = `
        mutation($assignableId: ID!, $actorIds: [ID!]!) {
          replaceActorsForAssignable(input: {
            assignableId: $assignableId,
            actorIds: $actorIds
          }) {
            __typename
          }
        }
      `;

      core.debug(`GraphQL mutation with variables: assignableId=${issueId}, actorIds=${JSON.stringify(actorIds)}`);
      response = await github.graphql(simpleMutation, {
        assignableId: issueId,
        actorIds: actorIds,
      });
    }

    if (response && response.replaceActorsForAssignable && response.replaceActorsForAssignable.__typename) {
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
        core.info("Using built-in github object for fallback mutation");
        core.debug(`Fallback GraphQL mutation with variables: assignableId=${issueId}, assigneeIds=[${agentId}]`);
        const fallbackResp = await github.graphql(fallbackMutation, {
          assignableId: issueId,
          assigneeIds: [agentId],
        });
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
  core.error("Assigning Copilot agents requires a Personal Access Token (PAT) with:");
  core.error("  - 'repo' scope (classic PAT), OR");
  core.error("  - Fine-grained PAT with Issues and Contents write permissions");
  core.error("");
  core.error("The default GITHUB_TOKEN cannot assign Copilot to issues.");
  core.error("");
  core.error("Configure your token:");
  core.error("  1. Create a PAT at: https://github.com/settings/tokens");
  core.error("  2. Store it as COPILOT_GITHUB_TOKEN secret in your repository");
  core.error("");
  core.error("Repository requirements:");
  core.error("  - Copilot coding agent must be enabled");
  core.error("  - Check: Settings > Copilot > Policies > Coding agent");
  core.error("");
  core.info("For more information, see: https://docs.github.com/en/copilot/how-tos/use-copilot-agents/coding-agent/create-a-pr");
}

/**
 * Generate permission error summary content for step summary
 * @returns {string} Markdown content for permission error guidance
 */
function generatePermissionErrorSummary() {
  let content = "\n### ⚠️ Permission Requirements\n\n";
  content += "Assigning Copilot agents requires a Personal Access Token (PAT):\n\n";
  content += "**Token Options:**\n";
  content += "- Classic PAT with `repo` scope\n";
  content += "- Fine-grained PAT with Issues and Contents write permissions\n\n";
  content += "⚠️ The default `GITHUB_TOKEN` cannot assign Copilot to issues.\n\n";
  content += "**Setup:**\n";
  content += "1. Create a PAT at https://github.com/settings/tokens\n";
  content += "2. Store as `COPILOT_GITHUB_TOKEN` secret in your repository\n\n";
  content += "**Repository Requirements:**\n";
  content += "- Copilot coding agent must be enabled\n";
  content += "- Check: Settings → Copilot → Policies → Coding agent\n\n";
  content += "📖 Reference: https://github.blog/changelog/2025-12-03-assign-issues-to-copilot-using-the-api/\n";
  return content;
}

/**
 * Assign an agent to an issue using GraphQL
 * This is the main entry point for assigning agents from other scripts
 * @param {string} owner - Repository owner
 * @param {string} repo - Repository name
 * @param {number} issueNumber - Issue number
 * @param {string} agentName - Agent name (e.g., "copilot")
 * @param {object} options - Optional assignment options
 * @param {string} [options.targetRepository] - Target repository in 'owner/repo' format
 * @param {string} [options.baseBranch] - Base branch for the PR
 * @param {string} [options.customInstructions] - Custom instructions for the agent
 * @param {string} [options.customAgent] - Custom agent name/path
 * @returns {Promise<{success: boolean, error?: string}>}
 */
async function assignAgentToIssueByName(owner, repo, issueNumber, agentName, options = {}) {
  // Check if agent is supported
  if (!AGENT_LOGIN_NAMES[agentName]) {
    const error = `Agent "${agentName}" is not supported. Supported agents: ${Object.keys(AGENT_LOGIN_NAMES).join(", ")}`;
    core.warning(error);
    return { success: false, error };
  }

  try {
    // Find agent using the github object authenticated via step-level github-token
    core.info(`Looking for ${agentName} coding agent...`);
    const agentId = await findAgent(owner, repo, agentName);
    if (!agentId) {
      const error = `${agentName} coding agent is not available for this repository`;
      // Enrich with available agent logins
      const available = await getAvailableAgentLogins(owner, repo);
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

    // Prepare assignment options
    const assignmentOptions = {};

    // Handle target repository if specified
    if (options.targetRepository) {
      const parts = options.targetRepository.split("/");
      if (parts.length === 2) {
        const repoId = await getRepositoryId(parts[0], parts[1]);
        if (repoId) {
          assignmentOptions.targetRepositoryId = repoId;
        }
      }
    }

    if (options.baseBranch) {
      assignmentOptions.baseBranch = options.baseBranch;
    }

    if (options.customInstructions) {
      assignmentOptions.customInstructions = options.customInstructions;
    }

    if (options.customAgent) {
      assignmentOptions.customAgent = options.customAgent;
    }

    // Assign agent using GraphQL mutation
    core.info(`Assigning ${agentName} coding agent to issue #${issueNumber}...`);
    const success = await assignAgentToIssue(issueDetails.issueId, agentId, issueDetails.currentAssignees, agentName, assignmentOptions);

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
  getRepositoryId,
  findAgent,
  getIssueDetails,
  assignAgentToIssue,
  assignAgentViaRest,
  isAgentAlreadyAssigned,
  logPermissionError,
  generatePermissionErrorSummary,
  assignAgentToIssueByName,
};
