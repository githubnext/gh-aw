// @ts-check
/// <reference types="@actions/github-script" />
// @safe-outputs-exempt SEC-004 — body fields are read-only API context, never written back

const { getErrorMessage } = require("./error_helpers.cjs");

/**
 * Shared helper functions for assigning coding agents (like Copilot) to issues.
 * These functions use GitHub REST APIs.
 */

/**
 * Map agent names to their GitHub bot login aliases.
 * Keep the most common/current alias first so logs have a stable primary name.
 * @type {Record<string, string[]>}
 */
const AGENT_LOGIN_NAMES = {
  copilot: ["copilot-swe-agent", "github-copilot-enterprise", "github-copilot-enterprise[bot]", "github-copilot", "github-copilot[bot]"],
};

/**
 * Normalize a GitHub login for internal matching.
 * @param {string} login
 * @returns {string}
 */
function normalizeLogin(login) {
  return login.startsWith("@") ? login.slice(1) : login;
}

/**
 * Reverse lookup of assignee aliases to canonical agent names.
 * @type {Record<string, string>}
 */
const AGENT_NAME_BY_LOGIN = Object.fromEntries(Object.entries(AGENT_LOGIN_NAMES).flatMap(([agentName, logins]) => logins.map(login => [normalizeLogin(login), agentName])));

/**
 * GitHub can surface bots either via type="Bot" or a [bot] login suffix.
 * Check both because assignee responses are not always consistent across endpoints.
 * @param {{login?: string, type?: string}|null|undefined} assignee
 * @returns {boolean}
 */
function isBotAssignee(assignee) {
  return assignee?.type === "Bot" || Boolean(assignee?.login?.endsWith("[bot]"));
}

/**
 * Return the known GitHub login aliases for an agent.
 * @param {string} agentName
 * @returns {string[]}
 */
function getAgentLogins(agentName) {
  const logins = AGENT_LOGIN_NAMES[agentName];
  if (!logins) return [];
  return logins;
}

/**
 * Check if an assignee is a known coding agent (bot)
 * @param {string} assignee - Assignee name (may include @ prefix)
 * @returns {string|null} Agent name if it's a known agent, null otherwise
 */
function getAgentName(assignee) {
  // Normalize: remove @ prefix if present
  const normalized = normalizeLogin(assignee);

  // Check if it's a known agent
  if (AGENT_LOGIN_NAMES[normalized]) {
    return normalized;
  }
  return AGENT_NAME_BY_LOGIN[normalized] || null;
}

/**
 * Return list of coding agent bot login names that are currently available as assignable actors
 * in this repository, as determined by checkUserCanBeAssigned.
 * @param {string} owner
 * @param {string} repo
 * @param {Object} [githubClient] - Authenticated GitHub client (defaults to global github)
 * @returns {Promise<string[]>}
 */
async function getAvailableAgentLogins(owner, repo, githubClient = github) {
  // Deduplicate defensively so future alias additions across agents do not duplicate REST lookups.
  const knownValues = [...new Set(Object.values(AGENT_LOGIN_NAMES).flat())];
  const available = [];
  for (const login of knownValues) {
    try {
      await githubClient.rest.issues.checkUserCanBeAssigned({
        owner,
        repo,
        assignee: login,
      });
      available.push(login);
    } catch (e) {
      const status = e && typeof e === "object" && "status" in e ? e.status : undefined;
      if (status !== 404) {
        core.debug(`Failed to check assignability for ${login}: ${getErrorMessage(e)}`);
      }
    }
  }
  return available.sort();
}

/**
 * Return assignable bot logins from the repository assignee list.
 * @param {string} owner
 * @param {string} repo
 * @param {Object} [githubClient]
 * @returns {Promise<string[]>}
 */
async function getAssignableBots(owner, repo, githubClient = github) {
  try {
    const assignees = [];
    let page = 1;
    let pageData = [];
    const MAX_PAGES = 5; // Limit to 5 pages (500 assignees) to bound API calls on large repositories

    do {
      const response = await githubClient.rest.issues.listAssignees({
        owner,
        repo,
        per_page: 100,
        page,
      });
      pageData = Array.isArray(response.data) ? response.data : [];
      assignees.push(...pageData);
      page++;
    } while (pageData.length === 100 && page <= MAX_PAGES);

    return [
      ...new Set(
        assignees
          .filter(isBotAssignee)
          .map(assignee => assignee.login)
          .filter(Boolean)
      ),
    ].sort();
  } catch (error) {
    core.debug(`Failed to list assignable bots for ${owner}/${repo}: ${getErrorMessage(error)}`);
    return [];
  }
}

/**
 * Find an agent that can be assigned in the repository using REST
 * @param {string} owner - Repository owner
 * @param {string} repo - Repository name
 * @param {string} agentName - Agent name (copilot)
 * @param {Object} [githubClient] - Authenticated GitHub client (defaults to global github)
 * @returns {Promise<string|null>} Agent ID or null if not found
 */
async function findAgent(owner, repo, agentName, githubClient = github) {
  const loginNames = getAgentLogins(agentName);
  if (loginNames.length === 0) {
    core.error(`Unknown agent: ${agentName}. Supported agents: ${Object.keys(AGENT_LOGIN_NAMES).join(", ")}`);
    return null;
  }

  core.info(`Trying ${loginNames.length} ${agentName} assignee aliases: ${loginNames.join(", ")}`);

  const aliasFailures = [];
  for (const loginName of loginNames) {
    try {
      core.info(`Checking assignee alias: ${loginName}`);
      await githubClient.rest.issues.checkUserCanBeAssigned({
        owner,
        repo,
        assignee: loginName,
      });
    } catch (checkError) {
      const errorMessage = getErrorMessage(checkError);
      const status = checkError?.status;
      const statusLabel = status ? ` (${status})` : "";
      aliasFailures.push(`${loginName}${statusLabel}: ${errorMessage}`);
      if (
        errorMessage.includes("Bad credentials") ||
        errorMessage.includes("Not Authenticated") ||
        errorMessage.includes("Resource not accessible") ||
        errorMessage.includes("Insufficient permissions") ||
        errorMessage.includes("requires authentication")
      ) {
        core.error(`Failed to check assignee alias ${loginName} for ${agentName}: ${errorMessage}`);
        throw checkError;
      }
      core.info(`Assignee alias ${loginName} was not assignable: ${errorMessage}`);
      continue;
    }
    // Alias confirmed assignable — resolve the user ID separately
    try {
      const { data: agentUser } = await githubClient.rest.users.getByUsername({ username: loginName });
      core.info(`Resolved ${agentName} agent via assignee alias ${loginName}`);
      return String(agentUser.id);
    } catch (lookupError) {
      core.warning(`Alias ${loginName} is assignable but user lookup failed: ${getErrorMessage(lookupError)}`);
    }
  }

  const bots = await getAssignableBots(owner, repo, githubClient);
  core.warning(`${agentName} coding agent aliases are not available as assignees for this repository`);
  core.info(`Assignee aliases tried: ${loginNames.join(", ")}`);
  if (aliasFailures.length > 0) {
    core.info(`Alias lookup results: ${aliasFailures.join(" | ")}`);
  }
  if (bots.length > 0) {
    core.info(`Assignable bots in this repository: ${bots.join(", ")}`);
  } else {
    core.info("No assignable bots found in this repository.");
  }
  if (agentName === "copilot") {
    core.info("Please visit https://docs.github.com/en/copilot/using-github-copilot/using-copilot-coding-agent-to-work-on-tasks/about-assigning-tasks-to-copilot");
  }
  return null;
}

/**
 * Get issue details (context and current assignees) using REST
 * @param {string} owner - Repository owner
 * @param {string} repo - Repository name
 * @param {number} issueNumber - Issue number
 * @param {Object} [githubClient] - Authenticated GitHub client (defaults to global github)
 * @returns {Promise<{issueId: string, currentAssignees: Array<{id: string, login: string}>, htmlUrl: string, title: string, body: string, taskContext: {owner: string, repo: string, type: "issue", number: number}}|null>}
 */
async function getIssueDetails(owner, repo, issueNumber, githubClient = github) {
  try {
    const { data: issue } = await githubClient.rest.issues.get({ owner, repo, issue_number: issueNumber });
    if (!issue || !issue.number) {
      core.error("Could not get issue data");
      return null;
    }
    const currentAssignees = (issue.assignees || []).map(assignee => ({
      id: String(assignee.id),
      login: assignee.login,
    }));

    return {
      issueId: String(issue.id),
      currentAssignees,
      htmlUrl: issue.html_url || "",
      title: issue.title || "",
      body: issue.body || "",
      taskContext: { owner, repo, type: "issue", number: issue.number },
    };
  } catch (error) {
    const errorMessage = getErrorMessage(error);
    core.error(`Failed to get issue details: ${errorMessage}`);
    throw error;
  }
}

/**
 * Get pull request details (context and current assignees) using REST
 * @param {string} owner - Repository owner
 * @param {string} repo - Repository name
 * @param {number} pullNumber - Pull request number
 * @param {Object} [githubClient] - Authenticated GitHub client (defaults to global github)
 * @returns {Promise<{pullRequestId: string, currentAssignees: Array<{id: string, login: string}>, htmlUrl: string, title: string, body: string, taskContext: {owner: string, repo: string, type: "pull", number: number}}|null>}
 */
async function getPullRequestDetails(owner, repo, pullNumber, githubClient = github) {
  try {
    const { data: pullRequest } = await githubClient.rest.pulls.get({ owner, repo, pull_number: pullNumber });
    if (!pullRequest || !pullRequest.number) {
      core.error("Could not get pull request data");
      return null;
    }
    const currentAssignees = (pullRequest.assignees || []).map(assignee => ({
      id: String(assignee.id),
      login: assignee.login,
    }));

    return {
      pullRequestId: String(pullRequest.id),
      currentAssignees,
      htmlUrl: pullRequest.html_url || "",
      title: pullRequest.title || "",
      body: pullRequest.body || "",
      taskContext: { owner, repo, type: "pull", number: pullRequest.number },
    };
  } catch (error) {
    const errorMessage = getErrorMessage(error);
    core.error(`Failed to get pull request details: ${errorMessage}`);
    throw error;
  }
}

/**
 * Start an agent task for issue or pull request context using REST
 * @param {string} assignableId - Synthetic target ID in format owner/repo#issue:N or owner/repo#pull:N
 * @param {string} agentId - Agent login name
 * @param {Array<{id: string, login: string}>} currentAssignees - List of current assignees with id and login
 * @param {string} agentName - Agent name for error messages
 * @param {string[]|null} allowedAgents - Optional list of allowed agent names. If provided, filters out non-allowed agents from current assignees.
 * @param {string|null} model - Optional AI model to use (e.g., "claude-opus-4.6", "auto")
 * @param {string|null} customAgent - Optional custom agent ID for custom agents
 * @param {string|null} customInstructions - Optional custom instructions for the agent
 * @param {string|null} baseBranch - Optional base branch for the PR (REST base_ref field)
 * @param {Object} [githubClient] - Authenticated GitHub client (defaults to global github)
 * @param {{owner: string, repo: string, type: "issue"|"pull", number: number}|null} [taskContext] - Source issue/PR context for REST path
 * @param {string|null} [pullRequestRepoSlug] - Optional pull request repository slug (owner/repo) for REST path
 * @returns {Promise<boolean>} True if successful
 */
async function assignAgentToIssue(
  assignableId,
  agentId,
  currentAssignees,
  agentName,
  allowedAgents = null,
  model = null,
  customAgent = null,
  customInstructions = null,
  baseBranch = null,
  githubClient = github,
  taskContext = null,
  pullRequestRepoSlug = null
) {
  // SECURITY: pullRequestRepoSlug specifies a cross-repo target repository slug.
  // Callers MUST validate the corresponding repository slug against allowedRepos using
  // validateTargetRepo (from repo_helpers.cjs) before invoking this function.
  // Filter current assignees based on allowed list (if configured)
  let filteredAssignees = currentAssignees;
  if (allowedAgents && allowedAgents.length > 0) {
    filteredAssignees = currentAssignees.filter(assignee => {
      // Check if this assignee is a known agent
      const assigneeAgentName = getAgentName(assignee.login);
      if (assigneeAgentName) {
        // It's an agent - only keep if in allowed list
        const isAllowed = allowedAgents.includes(assigneeAgentName);
        if (!isAllowed) {
          core.info(`Filtering out agent "${assignee.login}" (not in allowed list)`);
        }
        return isAllowed;
      }
      // Not an agent - keep it (regular user assignee)
      return true;
    });
  }

  if (!githubClient?.request) {
    core.error(`GitHub client does not support REST requests; cannot create agent task`);
    return false;
  }

  if (!taskContext) {
    core.error(`Invalid assignment context: ${assignableId}`);
    return false;
  }
  const sourceOwner = taskContext.owner;
  const sourceRepo = taskContext.repo;
  const itemType = taskContext.type === "pull" ? "pull request" : "issue";
  const itemNumber = String(taskContext.number);
  const sourceUrl = `https://github.com/${sourceOwner}/${sourceRepo}/${itemType === "pull request" ? "pull" : "issues"}/${itemNumber}`;
  const targetRepoSlug = pullRequestRepoSlug || `${sourceOwner}/${sourceRepo}`;
  const targetParts = targetRepoSlug.split("/");
  if (targetParts.length !== 2) {
    core.error(`Invalid target repository slug: ${targetRepoSlug}`);
    return false;
  }
  const targetOwner = targetParts[0];
  const targetRepo = targetParts[1];
  const promptParts = [`Start work for ${itemType} ${sourceOwner}/${sourceRepo}#${itemNumber}.`, `Use this as the primary context: ${sourceUrl}`];
  if (targetRepoSlug !== `${sourceOwner}/${sourceRepo}`) promptParts.push(`Create the branch and pull request in ${targetRepoSlug}.`);
  if (customAgent) {
    core.warning(`customAgent is not a dedicated REST parameter; it will be included as prompt context. If the agent runner does not parse this field, the custom agent selection may be ignored.`);
    promptParts.push(`Custom agent: ${customAgent}`);
  }
  if (customInstructions) promptParts.push(`Additional instructions:\n${customInstructions}`);
  const prompt = promptParts.join("\n\n");

  try {
    core.info("Starting agent task via REST API");
    const response = await githubClient.request("POST /agents/repos/{owner}/{repo}/tasks", {
      owner: targetOwner,
      repo: targetRepo,
      prompt,
      create_pull_request: true,
      ...(model ? { model } : {}),
      ...(baseBranch ? { base_ref: baseBranch } : {}),
      headers: { "X-GitHub-Api-Version": "2026-03-10" },
    });
    if (response?.data?.id) return true;
    core.error("Unexpected response from GitHub API");
    return false;
  } catch (error) {
    const errorMessage = getErrorMessage(error);

    const err = /** @type {any} */ error;
    const is502Error = err?.response?.status === 502 || errorMessage.includes("502 Bad Gateway");

    if (is502Error) {
      core.warning(`Received 502 error from cloud gateway during agent task creation, but task may have been created`);
      core.info(`502 error details logged for troubleshooting`);

      try {
        if (error && typeof error === "object") {
          const details = {
            ...(err.errors && { errors: err.errors }),
            ...(err.response && { response: err.response }),
            ...(err.data && { data: err.data }),
          };
          const serialized = JSON.stringify(details, null, 2);
          if (serialized !== "{}") {
            core.info("502 error details (for troubleshooting):");
            serialized
              .split("\n")
              .filter(line => line.trim())
              .forEach(line => core.info(line));
          }
        }
      } catch (loggingErr) {
        const loggingErrMsg = loggingErr instanceof Error ? loggingErr.message : String(loggingErr);
        core.debug(`Failed to serialize 502 error details: ${loggingErrMsg}`);
      }

      core.info(`Treating 502 error as success - agent task likely created`);
      return true;
    }

    // Debug: surface the raw REST error structure for troubleshooting fine-grained permission issues
    try {
      core.debug(`Raw REST error message: ${errorMessage}`);
      if (error && typeof error === "object") {
        const details = {
          ...(err.errors && { errors: err.errors }),
          ...(err.response && { response: err.response }),
          ...(err.data && { data: err.data }),
        };
        if (Array.isArray(err.errors)) {
          details.compactMessages = err.errors.map(e => e.message).filter(Boolean);
        }
        const serialized = JSON.stringify(details, null, 2);
        if (serialized !== "{}") {
          core.debug(`Raw REST error details: ${serialized}`);
          core.error("Raw REST error details (for troubleshooting):");
          serialized
            .split("\n")
            .filter(line => line.trim())
            .forEach(line => core.error(line));
        }
      }
    } catch (loggingErr) {
      const loggingErrMsg = loggingErr instanceof Error ? loggingErr.message : String(loggingErr);
      core.debug(`Failed to serialize REST error details: ${loggingErrMsg}`);
    }

    if (
      errorMessage.includes("Bad credentials") ||
      errorMessage.includes("Not Authenticated") ||
      errorMessage.includes("Resource not accessible") ||
      errorMessage.includes("Insufficient permissions") ||
      errorMessage.includes("requires authentication")
    ) {
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
  core.error("Assigning Copilot coding agent requires:");
  core.error("  1. Repository permissions:");
  core.error("     - actions: write");
  core.error("     - contents: write");
  core.error("     - agent-tasks: write");
  core.error("");
  core.error("  2. A fine-grained PAT or GitHub App user token with agent-tasks: write");
  core.error("     (Installation tokens are not supported for agent task creation)");
  core.error("");
  core.error("  3. Repository settings:");
  core.error("     - Actions must have write permissions");
  core.error("     - Go to: Settings > Actions > General > Workflow permissions");
  core.error("     - Select: 'Read and write permissions'");
  core.error("");
  core.error("  4. Organization/Enterprise settings and Copilot policy:");
  core.error("     - Check if your org restricts bot assignments");
  core.error("     - Verify Copilot is enabled for your repository");
  core.error("");
  core.info("For more information, see: https://docs.github.com/en/rest/agent-tasks/agent-tasks?apiVersion=2026-03-10#start-a-task");
}

/**
 * Generate permission error summary content for step summary
 * @returns {string} Markdown content for permission error guidance
 */
function generatePermissionErrorSummary() {
  return `
### ⚠️ Permission Requirements

Assigning Copilot coding agent requires **ALL** of these permissions:

\`\`\`yaml
permissions:
  actions: write
  contents: write
  agent-tasks: write
\`\`\`

**Token capability note:**
- Current token lacks permission for \`POST /agents/repos/{owner}/{repo}/tasks\`.
- Agent task creation requires a fine-grained PAT or GitHub App user token with **Agent tasks: read and write**.
- GitHub App installation access tokens are not supported for this endpoint.

**Recommended remediation paths:**
1. Use a fine-grained PAT with repository access and **Agent tasks (read/write)**.
2. Use a GitHub App **user access token** (not installation token) with Agent tasks permission.
3. Verify Copilot coding agent is enabled for the repository and organization policy allows task creation.

**Why this failed:** The token could not create an agent task via the REST API.

📖 Reference: https://docs.github.com/en/rest/agent-tasks/agent-tasks?apiVersion=2026-03-10#start-a-task
`;
}

/**
 * Assign an agent to an issue by starting an agent task using REST
 * This is the main entry point for assigning agents from other scripts
 * @param {string} owner - Repository owner
 * @param {string} repo - Repository name
 * @param {number} issueNumber - Issue number
 * @param {string} agentName - Agent name (e.g., "copilot")
 * @returns {Promise<{success: boolean, error?: string}>}
 */
async function assignAgentToIssueByName(owner, repo, issueNumber, agentName) {
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
      return { success: false, error: `${agentName} coding agent is not available for this repository` };
    }
    core.info(`Found ${agentName} coding agent (ID: ${agentId})`);

    // Get issue details and current assignees via REST
    core.info("Getting issue details...");
    const issueDetails = await getIssueDetails(owner, repo, issueNumber);
    if (!issueDetails) {
      return { success: false, error: "Failed to get issue details" };
    }

    core.info(`Issue context: ${issueDetails.issueId}`);

    // Check if agent is already assigned
    const knownLogins = getAgentLogins(agentName);
    if (issueDetails.currentAssignees.some(a => a.id === agentId || knownLogins.includes(a.login))) {
      core.info(`${agentName} is already assigned to issue #${issueNumber}`);
      return { success: true };
    }

    // Assign agent by starting a REST task (no allowed list filtering in this helper)
    core.info(`Assigning ${agentName} coding agent to issue #${issueNumber}...`);
    const success = await assignAgentToIssue(issueDetails.issueId, agentId, issueDetails.currentAssignees, agentName, null, null, null, null, null, github, issueDetails.taskContext);

    if (!success) {
      return { success: false, error: `Failed to assign ${agentName} via REST` };
    }

    core.info(`Successfully assigned ${agentName} coding agent to issue #${issueNumber}`);
    return { success: true };
  } catch (error) {
    const errorMessage = getErrorMessage(error);
    return { success: false, error: errorMessage };
  }
}

module.exports = {
  AGENT_LOGIN_NAMES,
  getAgentName,
  getAgentLogins,
  getAvailableAgentLogins,
  getAssignableBots,
  findAgent,
  getIssueDetails,
  getPullRequestDetails,
  assignAgentToIssue,
  logPermissionError,
  generatePermissionErrorSummary,
  assignAgentToIssueByName,
};
