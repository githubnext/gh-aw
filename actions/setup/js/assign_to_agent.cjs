// @ts-check
/// <reference types="@actions/github-script" />

const { AGENT_LOGIN_NAMES, getAvailableAgentLogins, findAgent, getIssueDetails, getPullRequestDetails, assignAgentToIssue, generatePermissionErrorSummary } = require("./assign_agent_helpers.cjs");
const { getErrorMessage } = require("./error_helpers.cjs");
const { resolveTarget, isStagedMode } = require("./safe_output_helpers.cjs");
const { generateStagedPreview } = require("./staged_preview.cjs");
const { isTemporaryId, normalizeTemporaryId, resolveRepoIssueTarget } = require("./temporary_id.cjs");
const { sleep } = require("./error_recovery.cjs");
const { parseAllowedRepos, validateRepo, resolveTargetRepoConfig, resolveAndValidateRepo } = require("./repo_helpers.cjs");
const { resolvePullRequestRepo } = require("./pr_helpers.cjs");
const { sanitizeContent } = require("./sanitize_content.cjs");

/**
 * Module-level state — populated by main(), read by the exported getters below.
 * Using module-level variables (rather than closure-only state) allows the handler
 * manager to read final output values after all messages have been processed.
 * @type {Array<{issue_number: number|null, pull_number: number|null, agent: string, owner: string|null, repo: string|null, success: boolean, skipped?: boolean, error?: string}>}
 */
let _allResults = [];

/**
 * Create a dedicated GitHub client for assign-to-agent operations.
 *
 * Token precedence:
 *   1. config["github-token"] — per-handler PAT configured in the workflow frontmatter
 *   2. GH_AW_ASSIGN_TO_AGENT_TOKEN — agent token injected by the compiler as a step env var
 *      (evaluates to: GH_AW_AGENT_TOKEN || GH_AW_GITHUB_TOKEN || GITHUB_TOKEN)
 *   3. global github — step-level token (fallback when no agent token is available)
 *
 * @param {Object} config - Handler configuration
 * @returns {Promise<Object>} Authenticated GitHub client
 */
async function createAssignToAgentGitHubClient(config) {
  const token = config["github-token"] || process.env.GH_AW_ASSIGN_TO_AGENT_TOKEN;
  if (!token) {
    core.debug("No dedicated agent token configured — using step-level github client for assign-to-agent operations");
    return github;
  }
  core.info("Using dedicated github client for assign-to-agent operations");
  return global.getOctokit(token);
}

/**
 * Handler factory for assign-to-agent safe output.
 *
 * Replaces the standalone assign_to_agent step. This function is called once by the
 * safe output handler manager with the handler's configuration. It returns a message
 * processor function that is invoked for each assign_to_agent message in the agent output.
 *
 * @param {Object} config - Handler configuration from GH_AW_SAFE_OUTPUTS_HANDLER_CONFIG
 * @returns {Promise<Function>} Message processor function
 */
async function main(config = {}) {
  // Parse configuration (replaces env vars from the old standalone step)
  const maxCount = parseInt(String(config.max ?? "1"), 10);
  if (isNaN(maxCount) || maxCount < 1) {
    throw new Error(`Invalid max value: ${config.max}. Must be a positive integer`);
  }
  const defaultAgent = String(config.name ?? "copilot").trim();
  const defaultModel = config.model ? String(config.model).trim() : null;
  const defaultCustomAgent = config["custom-agent"] ? String(config["custom-agent"]).trim() : null;
  const defaultCustomInstructions = config["custom-instructions"] ? String(config["custom-instructions"]).trim() : null;
  const configuredBaseBranch = config["base-branch"] ? String(config["base-branch"]).trim() : null;
  const targetConfig = config.target ? String(config.target).trim() : "triggering";
  const ignoreIfError = config["ignore-if-error"] === true || config["ignore-if-error"] === "true";
  const allowedAgents = config.allowed
    ? Array.isArray(config.allowed)
      ? config.allowed.map(a => String(a).trim()).filter(Boolean)
      : String(config.allowed)
          .split(",")
          .map(a => a.trim())
          .filter(Boolean)
    : null;
  const { defaultTargetRepo, allowedRepos } = resolveTargetRepoConfig(config);
  const allowedPullRequestRepos = parseAllowedRepos(config["allowed-pull-request-repos"]);

  // Create a dedicated Octokit instance using the agent token
  const githubClient = await createAssignToAgentGitHubClient(config);

  // Check if we're in staged mode
  const isStaged = isStagedMode(config);

  core.info(`Default agent: ${defaultAgent}`);
  if (defaultModel) core.info(`Default model: ${defaultModel}`);
  if (defaultCustomAgent) core.info(`Default custom agent: ${defaultCustomAgent}`);
  if (configuredBaseBranch) core.info(`Configured base branch: ${configuredBaseBranch}`);
  core.info(`Target configuration: ${targetConfig}`);
  core.info(`Max count: ${maxCount}`);
  if (ignoreIfError) core.info("Ignore-if-error mode enabled: Will not fail if agent assignment encounters auth errors");
  if (allowedAgents) core.info(`Allowed agents: ${allowedAgents.join(", ")}`);
  core.info(`Default target repo: ${defaultTargetRepo}`);
  if (allowedRepos.size > 0) core.info(`Allowed repos: ${[...allowedRepos].join(", ")}`);

  // Resolve pull request repo upfront (if globally configured)
  let pullRequestOwner = null;
  let pullRequestRepo = null;
  let pullRequestRepoId = null;
  let effectiveBaseBranch = configuredBaseBranch;
  const pullRequestRepoConfig = config["pull-request-repo"] ? String(config["pull-request-repo"]).trim() : null;

  if (pullRequestRepoConfig) {
    const parts = pullRequestRepoConfig.split("/");
    if (parts.length === 2) {
      const repoValidation = validateRepo(pullRequestRepoConfig, pullRequestRepoConfig, allowedPullRequestRepos);
      if (!repoValidation.valid) {
        throw new Error(`E004: ${repoValidation.error}`);
      }
      pullRequestOwner = parts[0];
      pullRequestRepo = parts[1];
      core.info(`Using pull request repository: ${pullRequestOwner}/${pullRequestRepo}`);
      try {
        const resolved = await resolvePullRequestRepo(githubClient, pullRequestOwner, pullRequestRepo, configuredBaseBranch);
        pullRequestRepoId = resolved.repoId;
        effectiveBaseBranch = resolved.effectiveBaseBranch;
        core.info(`Pull request repository ID: ${pullRequestRepoId}`);
        if (!configuredBaseBranch && effectiveBaseBranch) {
          core.info(`Resolved pull request repository default branch: ${effectiveBaseBranch}`);
        }
      } catch (error) {
        throw new Error(`Failed to fetch pull request repository ID for ${pullRequestOwner}/${pullRequestRepo}: ${getErrorMessage(error)}`);
      }
    } else {
      core.warning(`Invalid pull-request-repo format: ${pullRequestRepoConfig}. Expected owner/repo. PRs will be created in issue repository.`);
    }
  }

  // Closure-level state
  let processedCount = 0;
  const agentCache = {};

  // Reset module-level results for this handler invocation
  _allResults = [];

  /**
   * Message processor — called once per assign_to_agent message by the handler manager.
   *
   * @param {Object} message - The assign_to_agent message from agent output
   * @param {Object} resolvedTemporaryIds - Plain object of already-resolved temp IDs
   * @param {Map<string, {repo: string, number: number}>} temporaryIdMap - Live temp ID map
   * @returns {Promise<{success: boolean, error?: string, skipped?: boolean, deferred?: boolean}>}
   */
  return async function handleMessage(message, resolvedTemporaryIds, temporaryIdMap) {
    // Handle staged mode — emit preview and skip actual assignment
    if (isStaged) {
      await generateStagedPreview({
        title: "Assign to Agent",
        description: "The following agent assignments would be made if staged mode was disabled:",
        items: [message],
        renderItem: item => {
          const parts = [];
          if (item.issue_number) {
            parts.push(`**Issue:** #${item.issue_number}`);
          } else if (item.pull_number) {
            parts.push(`**Pull Request:** #${item.pull_number}`);
          }
          parts.push(`**Agent:** ${item.agent || defaultAgent}`);
          if (defaultModel) parts.push(`**Model:** ${defaultModel}`);
          if (defaultCustomAgent) parts.push(`**Custom Agent:** ${defaultCustomAgent}`);
          if (defaultCustomInstructions) parts.push(`**Custom Instructions:** ${defaultCustomInstructions}`);
          return parts.join("\n") + "\n\n";
        },
      });
      return { success: true, skipped: true };
    }

    // Enforce max count — track the attempt in _allResults so it appears in the summary
    if (processedCount >= maxCount) {
      core.info(`⏭ Max count (${maxCount}) reached, skipping agent assignment`);
      const agentNameForSkip = message.agent ?? defaultAgent;
      _allResults.push({ issue_number: message.issue_number || null, pull_number: message.pull_number || null, agent: agentNameForSkip, owner: null, repo: null, success: false, skipped: true });
      return { success: false, skipped: true };
    }

    // Add delay between consecutive assignments to avoid spawning too many agents at once
    if (processedCount > 0) {
      core.info("Waiting 10 seconds before processing next agent assignment...");
      await sleep(10000);
    }

    const agentName = message.agent ?? defaultAgent;
    const model = defaultModel;
    const customAgent = defaultCustomAgent;
    const customInstructions = defaultCustomInstructions || null;

    // Validate that both issue_number and pull_number are not specified simultaneously
    if (message.issue_number != null && message.pull_number != null) {
      const error = "Cannot specify both issue_number and pull_number in the same assign_to_agent item";
      core.error(error);
      _allResults.push({ issue_number: message.issue_number, pull_number: message.pull_number, agent: agentName, owner: null, repo: null, success: false, error });
      return { success: false, error };
    }

    // Defer if issue_number is a temporary ID that hasn't been resolved yet
    if (message.issue_number != null && isTemporaryId(message.issue_number)) {
      const normalized = normalizeTemporaryId(String(message.issue_number));
      if (!temporaryIdMap.has(normalized)) {
        core.info(`Deferring assign_to_agent — temporary ID ${message.issue_number} not yet resolved`);
        return { success: false, deferred: true };
      }
    }

    // Resolve and validate target repository
    const repoResult = resolveAndValidateRepo(message, defaultTargetRepo, allowedRepos, "issue/PR");
    if (!repoResult.success) {
      core.error(`E004: ${repoResult.error}`);
      _allResults.push({ issue_number: message.issue_number || null, pull_number: message.pull_number || null, agent: agentName, owner: null, repo: null, success: false, error: repoResult.error });
      return { success: false, error: repoResult.error };
    }
    let effectiveOwner = repoResult.repoParts.owner;
    let effectiveRepo = repoResult.repoParts.repo;
    let itemForTarget = message;

    // Resolve temporary ID in issue_number to real issue number
    if (message.issue_number != null) {
      const resolvedTarget = resolveRepoIssueTarget(message.issue_number, temporaryIdMap, effectiveOwner, effectiveRepo);
      if (!resolvedTarget.resolved) {
        const error = resolvedTarget.errorMessage || `Failed to resolve issue target: ${message.issue_number}`;
        core.error(error);
        _allResults.push({ issue_number: message.issue_number, pull_number: null, agent: agentName, owner: effectiveOwner, repo: effectiveRepo, success: false, error });
        return { success: false, error };
      }
      effectiveOwner = resolvedTarget.resolved.owner;
      effectiveRepo = resolvedTarget.resolved.repo;
      itemForTarget = { ...message, issue_number: resolvedTarget.resolved.number };
      if (resolvedTarget.wasTemporaryId) {
        core.info(`Resolved temporary issue id to ${effectiveOwner}/${effectiveRepo}#${resolvedTarget.resolved.number}`);
      }
    }

    // Determine effective target configuration
    const hasExplicitTarget = itemForTarget.issue_number != null || itemForTarget.pull_number != null;
    const effectiveTarget = hasExplicitTarget ? "*" : targetConfig;

    // Handle per-item pull_request_repo override
    let effectivePullRequestRepoId = pullRequestRepoId;
    if (message.pull_request_repo) {
      const itemPullRequestRepo = String(message.pull_request_repo).trim();
      const pullRequestRepoParts = itemPullRequestRepo.split("/");
      if (pullRequestRepoParts.length === 2) {
        const defaultPullRequestRepo = pullRequestRepoConfig || defaultTargetRepo;
        const pullRequestRepoValidation = validateRepo(itemPullRequestRepo, defaultPullRequestRepo, allowedPullRequestRepos);
        if (!pullRequestRepoValidation.valid) {
          const error = pullRequestRepoValidation.error ?? "Repository validation failed";
          core.error(`E004: ${error}`);
          _allResults.push({ issue_number: message.issue_number || null, pull_number: message.pull_number || null, agent: agentName, owner: effectiveOwner, repo: effectiveRepo, success: false, error });
          return { success: false, error };
        }
        try {
          const itemPullRequestRepoQuery = `
            query($owner: String!, $name: String!) {
              repository(owner: $owner, name: $name) { id }
            }
          `;
          const itemPullRequestRepoResponse = await githubClient.graphql(itemPullRequestRepoQuery, { owner: pullRequestRepoParts[0], name: pullRequestRepoParts[1] });
          effectivePullRequestRepoId = itemPullRequestRepoResponse.repository.id;
          core.info(`Using per-item pull request repository: ${itemPullRequestRepo} (ID: ${effectivePullRequestRepoId})`);
        } catch (error) {
          const errorMsg = `Failed to fetch pull request repository ID for ${itemPullRequestRepo}: ${getErrorMessage(error)}`;
          core.error(errorMsg);
          _allResults.push({ issue_number: message.issue_number || null, pull_number: message.pull_number || null, agent: agentName, owner: effectiveOwner, repo: effectiveRepo, success: false, error: errorMsg });
          return { success: false, error: errorMsg };
        }
      } else {
        core.warning(`Invalid pull_request_repo format: ${itemPullRequestRepo}. Expected owner/repo. Using global pull-request-repo if configured.`);
      }
    }

    // Resolve the target issue or pull request number from context
    const targetResult = resolveTarget({
      targetConfig: effectiveTarget,
      item: itemForTarget,
      context,
      itemType: "assign_to_agent",
      supportsPR: true,
      supportsIssue: false,
    });

    if (!targetResult.success) {
      if (targetResult.shouldFail) {
        core.error(targetResult.error);
        _allResults.push({ issue_number: message.issue_number || null, pull_number: message.pull_number || null, agent: agentName, owner: effectiveOwner, repo: effectiveRepo, success: false, error: targetResult.error });
        return { success: false, error: targetResult.error };
      } else {
        core.info(targetResult.error);
        return { success: false, skipped: true };
      }
    }

    const number = targetResult.number;
    const type = targetResult.contextType;
    const issueNumber = type === "issue" ? number : null;
    const pullNumber = type === "pull request" ? number : null;

    if (isNaN(number) || number <= 0) {
      const error = `Invalid ${type} number: ${number}`;
      core.error(error);
      _allResults.push({ issue_number: issueNumber, pull_number: pullNumber, agent: agentName, owner: effectiveOwner, repo: effectiveRepo, success: false, error });
      return { success: false, error };
    }

    // Validate agent name
    if (!AGENT_LOGIN_NAMES[agentName]) {
      const error = `Unsupported agent: ${agentName}`;
      core.warning(`Agent "${agentName}" is not supported. Supported agents: ${Object.keys(AGENT_LOGIN_NAMES).join(", ")}`);
      _allResults.push({ issue_number: issueNumber, pull_number: pullNumber, agent: agentName, owner: effectiveOwner, repo: effectiveRepo, success: false, error });
      return { success: false, error };
    }

    // Enforce allowed agents list
    if (allowedAgents && !allowedAgents.includes(agentName)) {
      const error = `Agent not allowed: ${agentName}`;
      core.error(`Agent "${agentName}" is not in the allowed list. Allowed agents: ${allowedAgents.join(", ")}`);
      _allResults.push({ issue_number: issueNumber, pull_number: pullNumber, agent: agentName, owner: effectiveOwner, repo: effectiveRepo, success: false, error });
      return { success: false, error };
    }

    // Increment processed count before attempting the assignment
    processedCount++;

    try {
      // Find agent (use cache to avoid repeated lookups)
      let agentId = agentCache[agentName];
      if (!agentId) {
        core.info(`Looking for ${agentName} coding agent...`);
        agentId = await findAgent(effectiveOwner, effectiveRepo, agentName, githubClient);
        if (!agentId) {
          throw new Error(`${agentName} coding agent is not available for this repository`);
        }
        agentCache[agentName] = agentId;
        core.info(`Found ${agentName} coding agent (ID: ${agentId})`);
      }

      // Get issue or PR details
      core.info(`Getting ${type} details...`);
      let assignableId;
      let currentAssignees;
      if (issueNumber) {
        const issueDetails = await getIssueDetails(effectiveOwner, effectiveRepo, issueNumber, githubClient);
        if (!issueDetails) throw new Error(`Failed to get issue details`);
        assignableId = issueDetails.issueId;
        currentAssignees = issueDetails.currentAssignees;
      } else if (pullNumber) {
        const prDetails = await getPullRequestDetails(effectiveOwner, effectiveRepo, pullNumber, githubClient);
        if (!prDetails) throw new Error(`Failed to get pull request details`);
        assignableId = prDetails.pullRequestId;
        currentAssignees = prDetails.currentAssignees;
      } else {
        throw new Error(`No issue or pull request number available`);
      }

      core.info(`${type} ID: ${assignableId}`);

      const hasPerItemPullRequestRepoOverride = !!message.pull_request_repo;

      // Skip if agent is already assigned and no explicit per-item pull_request_repo is specified.
      // When a different pull_request_repo is provided on the message, allow re-assignment
      // so Copilot can be triggered for a different target repository on the same issue.
      if (currentAssignees.some(a => a.id === agentId) && !hasPerItemPullRequestRepoOverride) {
        core.info(`${agentName} is already assigned to ${type} #${number}`);
        _allResults.push({ issue_number: issueNumber, pull_number: pullNumber, agent: agentName, owner: effectiveOwner, repo: effectiveRepo, success: true });
        return { success: true };
      }

      core.info(`Assigning ${agentName} coding agent to ${type} #${number}...`);
      if (model) core.info(`Using model: ${model}`);
      if (customAgent) core.info(`Using custom agent: ${customAgent}`);
      if (customInstructions) core.info(`Using custom instructions: ${customInstructions.substring(0, 100)}${customInstructions.length > 100 ? "..." : ""}`);
      if (effectiveBaseBranch) core.info(`Using base branch: ${effectiveBaseBranch}`);

      const success = await assignAgentToIssue(assignableId, agentId, currentAssignees, agentName, allowedAgents, effectivePullRequestRepoId, model, customAgent, customInstructions, effectiveBaseBranch, githubClient);
      if (!success) throw new Error(`Failed to assign ${agentName} via GraphQL`);

      core.info(`Successfully assigned ${agentName} coding agent to ${type} #${number}`);
      _allResults.push({ issue_number: issueNumber, pull_number: pullNumber, agent: agentName, owner: effectiveOwner, repo: effectiveRepo, success: true });
      return { success: true };
    } catch (error) {
      let errorMessage = getErrorMessage(error);

      const isAuthError = ["Bad credentials", "Not Authenticated", "Resource not accessible", "Insufficient permissions", "requires authentication"].some(msg => errorMessage.includes(msg));

      if (ignoreIfError && isAuthError) {
        core.warning(`Agent assignment failed for ${agentName} on ${type} #${number} due to authentication/permission error. Skipping due to ignore-if-error=true.`);
        core.info(`Error details: ${errorMessage}`);
        _allResults.push({ issue_number: issueNumber, pull_number: pullNumber, agent: agentName, owner: effectiveOwner, repo: effectiveRepo, success: true, skipped: true });
        return { success: true, skipped: true };
      }

      if (errorMessage.includes("coding agent is not available for this repository")) {
        try {
          const available = await getAvailableAgentLogins(effectiveOwner, effectiveRepo, githubClient);
          if (available.length > 0) errorMessage += ` (available agents: ${available.join(", ")})`;
        } catch (e) {
          core.debug("Failed to enrich unavailable agent message with available list");
        }
      }

      core.error(`Failed to assign agent "${agentName}" to ${type} #${number}: ${errorMessage}`);

      // Post failure comment on the issue/PR so the user sees the failure in context
      try {
        await githubClient.rest.issues.createComment({
          owner: effectiveOwner,
          repo: effectiveRepo,
          issue_number: number,
          body: sanitizeContent(`⚠️ **Assignment failed**: Failed to assign ${agentName} coding agent to this ${type}.\n\nError: ${errorMessage}`, { maxLength: 65000 }),
        });
        core.info(`Posted failure comment on ${type} #${number} in ${effectiveOwner}/${effectiveRepo}`);
      } catch (commentError) {
        core.warning(`Failed to post failure comment on ${type} #${number}: ${getErrorMessage(commentError)}`);
      }

      _allResults.push({ issue_number: issueNumber, pull_number: pullNumber, agent: agentName, owner: effectiveOwner, repo: effectiveRepo, success: false, error: errorMessage });
      return { success: false, error: errorMessage };
    }
  };
}

/**
 * Returns the "assigned" output string for step outputs.
 * Format: "issue:N:agent" or "pr:N:agent" per successful assignment, newline-separated.
 * @returns {string}
 */
function getAssignToAgentAssigned() {
  return _allResults
    .filter(r => r.success && !r.skipped)
    .map(r => {
      const number = r.issue_number || r.pull_number;
      const prefix = r.issue_number ? "issue" : "pr";
      return `${prefix}:${number}:${r.agent}`;
    })
    .join("\n");
}

/**
 * Returns the "assignment_errors" output string for step outputs.
 * Format: "issue:N:agent:error" or "pr:N:agent:error" per failure, newline-separated.
 * @returns {string}
 */
function getAssignToAgentErrors() {
  return _allResults
    .filter(r => !r.success && !r.skipped)
    .map(r => {
      const number = r.issue_number || r.pull_number;
      const prefix = r.issue_number ? "issue" : "pr";
      return `${prefix}:${number}:${r.agent}:${r.error}`;
    })
    .join("\n");
}

/**
 * Returns the "assignment_error_count" output value.
 * @returns {number}
 */
function getAssignToAgentErrorCount() {
  return _allResults.filter(r => !r.success && !r.skipped).length;
}

/**
 * Writes a step summary for agent assignment results.
 * Called by the handler manager after all messages have been processed.
 * @returns {Promise<void>}
 */
async function writeAssignToAgentSummary() {
  const successResults = _allResults.filter(r => r.success && !r.skipped);
  const skippedResults = _allResults.filter(r => r.skipped);
  const failedResults = _allResults.filter(r => !r.success && !r.skipped);

  if (_allResults.length === 0) return;

  let summaryContent = "## Agent Assignment\n\n";

  if (successResults.length > 0) {
    summaryContent += `✅ Successfully assigned ${successResults.length} agent(s):\n\n`;
    summaryContent += successResults
      .map(r => {
        const itemType = r.issue_number ? `Issue #${r.issue_number}` : `Pull Request #${r.pull_number}`;
        return `- ${itemType} → Agent: ${r.agent}`;
      })
      .join("\n");
    summaryContent += "\n\n";
  }

  if (skippedResults.length > 0) {
    summaryContent += `⏭️ Skipped ${skippedResults.length} agent assignment(s) (ignore-if-error enabled):\n\n`;
    summaryContent += skippedResults
      .map(r => {
        const itemType = r.issue_number ? `Issue #${r.issue_number}` : `Pull Request #${r.pull_number}`;
        return `- ${itemType} → Agent: ${r.agent} (assignment failed due to error)`;
      })
      .join("\n");
    summaryContent += "\n\n";
  }

  if (failedResults.length > 0) {
    summaryContent += `❌ Failed to assign ${failedResults.length} agent(s):\n\n`;
    summaryContent += failedResults
      .map(r => {
        const itemType = r.issue_number ? `Issue #${r.issue_number}` : `Pull Request #${r.pull_number}`;
        return `- ${itemType} → Agent: ${r.agent}: ${r.error}`;
      })
      .join("\n");

    const hasPermissionError = failedResults.some(r => r.error?.includes("Resource not accessible") || r.error?.includes("Insufficient permissions"));
    if (hasPermissionError) {
      summaryContent += generatePermissionErrorSummary();
    }
    summaryContent += "\n\n";
  }

  try {
    await core.summary.addRaw(summaryContent).write();
  } catch (error) {
    core.warning(`Failed to write agent assignment summary: ${getErrorMessage(error)}`);
  }
}

module.exports = { main, getAssignToAgentAssigned, getAssignToAgentErrors, getAssignToAgentErrorCount, writeAssignToAgentSummary };
