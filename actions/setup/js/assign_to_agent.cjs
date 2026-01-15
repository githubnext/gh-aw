// @ts-check
/// <reference types="@actions/github-script" />

/**
 * @typedef {import('./types/handler-factory').HandlerFactoryFunction} HandlerFactoryFunction
 */

/** @type {string} Safe output type handled by this module */
const HANDLER_TYPE = "assign_to_agent";

const { AGENT_LOGIN_NAMES, getAvailableAgentLogins, findAgent, getIssueDetails, getPullRequestDetails, assignAgentToIssue, generatePermissionErrorSummary } = require("./assign_agent_helpers.cjs");
const { getErrorMessage } = require("./error_helpers.cjs");

/**
 * Main handler factory for assign_to_agent
 * Returns a message handler function that processes individual assign_to_agent messages
 * @type {HandlerFactoryFunction}
 */
async function main(config = {}) {
  // Extract configuration
  const defaultAgent = config.name || "copilot";
  const maxCount = config.max || 1;
  const targetRepoConfig = config["target-repo"] || "";

  core.info(`Assign to agent configuration: max=${maxCount}, default_agent=${defaultAgent}`);
  if (targetRepoConfig) {
    core.info(`Target repository: ${targetRepoConfig}`);
  }

  // Track how many items we've processed for max limit
  let processedCount = 0;

  // Cache agent IDs to avoid repeated lookups
  const agentCache = {};

  /**
   * Message handler function that processes a single assign_to_agent message
   * @param {Object} message - The assign_to_agent message to process
   * @param {Object} resolvedTemporaryIds - Map of temporary IDs to {repo, number}
   * @returns {Promise<Object>} Result with success/error status
   */
  return async function handleAssignToAgent(message, resolvedTemporaryIds) {
    // Check if we've hit the max limit
    if (processedCount >= maxCount) {
      core.warning(`Skipping assign_to_agent: max count of ${maxCount} reached`);
      return {
        success: false,
        error: `Max count of ${maxCount} reached`,
      };
    }

    processedCount++;

    // Get target repository from config or current repo
    let targetOwner = context.repo.owner;
    let targetRepo = context.repo.repo;

    if (targetRepoConfig) {
      const parts = targetRepoConfig.split("/");
      if (parts.length === 2) {
        targetOwner = parts[0];
        targetRepo = parts[1];
      }
    }

    // Determine if this is an issue or PR assignment
    const issueNumber = message.issue_number ? (typeof message.issue_number === "number" ? message.issue_number : parseInt(String(message.issue_number), 10)) : null;
    const pullNumber = message.pull_number ? (typeof message.pull_number === "number" ? message.pull_number : parseInt(String(message.pull_number), 10)) : null;
    const agentName = message.agent ?? defaultAgent;

    // Validate that we have either issue_number or pull_number
    if (!issueNumber && !pullNumber) {
      core.error("Missing both issue_number and pull_number in assign_to_agent message");
      return {
        success: false,
        error: "Missing both issue_number and pull_number",
      };
    }

    if (issueNumber && pullNumber) {
      core.error("Cannot specify both issue_number and pull_number in the same assign_to_agent message");
      return {
        success: false,
        error: "Cannot specify both issue_number and pull_number",
      };
    }

    const number = issueNumber || pullNumber;
    const type = issueNumber ? "issue" : "pull request";

    if (isNaN(number) || number <= 0) {
      core.error(`Invalid ${type} number: ${number}`);
      return {
        success: false,
        error: `Invalid ${type} number: ${number}`,
      };
    }

    // Check if agent is supported
    if (!AGENT_LOGIN_NAMES[agentName]) {
      core.warning(`Agent "${agentName}" is not supported. Supported agents: ${Object.keys(AGENT_LOGIN_NAMES).join(", ")}`);
      return {
        success: false,
        error: `Unsupported agent: ${agentName}`,
      };
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
        return {
          success: true,
          number: number,
          agent: agentName,
          type: type,
        };
      }

      // Assign agent using GraphQL mutation - uses built-in github object authenticated via github-token
      core.info(`Assigning ${agentName} coding agent to ${type} #${number}...`);
      const success = await assignAgentToIssue(assignableId, agentId, currentAssignees, agentName);

      if (!success) {
        throw new Error(`Failed to assign ${agentName} via GraphQL`);
      }

      core.info(`Successfully assigned ${agentName} coding agent to ${type} #${number}`);
      return {
        success: true,
        number: number,
        agent: agentName,
        type: type,
      };
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
      return {
        success: false,
        error: errorMessage,
      };
    }
  };
}

module.exports = { main };
