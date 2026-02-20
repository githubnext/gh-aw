// @ts-check
/// <reference types="@actions/github-script" />

const { loadAgentOutput } = require("./load_agent_output.cjs");
const { generateStagedPreview } = require("./staged_preview.cjs");
const { AGENT_LOGIN_NAMES, getAvailableAgentLogins, findAgent, getIssueDetails, getPullRequestDetails, assignAgentToIssue, generatePermissionErrorSummary } = require("./assign_agent_helpers.cjs");
const { getErrorMessage } = require("./error_helpers.cjs");
const { resolveTarget } = require("./safe_output_helpers.cjs");
const { loadTemporaryIdMap, resolveRepoIssueTarget } = require("./temporary_id.cjs");
const { sleep } = require("./error_recovery.cjs");
const { parseAllowedRepos, validateRepo, resolveTargetRepoConfig, resolveAndValidateRepo } = require("./repo_helpers.cjs");
const { resolvePullRequestRepo, buildBranchInstruction } = require("./pr_helpers.cjs");

async function main() {
  const result = loadAgentOutput();
  if (!result.success) {
    return;
  }

  // Load temporary ID map once (used to resolve aw_... IDs to real issue numbers)
  const temporaryIdMap = loadTemporaryIdMap();

  const assignItems = result.items.filter(item => item.type === "assign_to_agent");
  if (assignItems.length === 0) {
    core.info("No assign_to_agent items found in agent output");
    return;
  }

  core.info(`Found ${assignItems.length} assign_to_agent item(s)`);

  // Check if we're in staged mode — if so, emit 🎭 Staged Mode Preview via generateStagedPreview
  if (process.env.GH_AW_SAFE_OUTPUTS_STAGED === "true") {
    // Get defaults for preview
    const previewDefaultAgent = process.env.GH_AW_AGENT_DEFAULT?.trim() ?? "copilot";
    const previewDefaultModel = process.env.GH_AW_AGENT_DEFAULT_MODEL?.trim();
    const previewDefaultCustomAgent = process.env.GH_AW_AGENT_DEFAULT_CUSTOM_AGENT?.trim();
    const previewDefaultCustomInstructions = process.env.GH_AW_AGENT_DEFAULT_CUSTOM_INSTRUCTIONS?.trim();

    await generateStagedPreview({
      title: "Assign to Agent",
      description: "The following agent assignments would be made if staged mode was disabled:",
      items: assignItems,
      renderItem: item => {
        const parts = [];
        if (item.issue_number) {
          parts.push(`**Issue:** #${item.issue_number}`);
        } else if (item.pull_number) {
          parts.push(`**Pull Request:** #${item.pull_number}`);
        }
        parts.push(`**Agent:** ${item.agent || previewDefaultAgent}`);
        if (previewDefaultModel) {
          parts.push(`**Model:** ${previewDefaultModel}`);
        }
        if (previewDefaultCustomAgent) {
          parts.push(`**Custom Agent:** ${previewDefaultCustomAgent}`);
        }
        if (previewDefaultCustomInstructions) {
          parts.push(`**Custom Instructions:** ${previewDefaultCustomInstructions}`);
        }
        return parts.join("\n") + "\n\n";
      },
    });
    return;
  }

  // Get default agent from configuration
  const defaultAgent = process.env.GH_AW_AGENT_DEFAULT?.trim() ?? "copilot";
  core.info(`Default agent: ${defaultAgent}`);

  // Get default model from configuration
  const defaultModel = process.env.GH_AW_AGENT_DEFAULT_MODEL?.trim();
  if (defaultModel) {
    core.info(`Default model: ${defaultModel}`);
  }

  // Get default custom agent from configuration
  const defaultCustomAgent = process.env.GH_AW_AGENT_DEFAULT_CUSTOM_AGENT?.trim();
  if (defaultCustomAgent) {
    core.info(`Default custom agent: ${defaultCustomAgent}`);
  }

  // Get default custom instructions from configuration
  const defaultCustomInstructions = process.env.GH_AW_AGENT_DEFAULT_CUSTOM_INSTRUCTIONS?.trim();
  if (defaultCustomInstructions) {
    core.info(`Default custom instructions: ${defaultCustomInstructions}`);
  }

  // Get base branch configuration for PR creation in target repo
  const configuredBaseBranch = process.env.GH_AW_AGENT_BASE_BRANCH?.trim();
  if (configuredBaseBranch) {
    core.info(`Configured base branch: ${configuredBaseBranch}`);
  }

  // Get target configuration (defaults to "triggering")
  const targetConfig = process.env.GH_AW_AGENT_TARGET?.trim() || "triggering";
  core.info(`Target configuration: ${targetConfig}`);

  // Get ignore-if-error flag (defaults to false)
  const ignoreIfError = process.env.GH_AW_AGENT_IGNORE_IF_ERROR === "true";
  if (ignoreIfError) {
    core.info("Ignore-if-error mode enabled: Will not fail if agent assignment encounters errors");
  }

  // Get allowed agents list (comma-separated)
  const allowedAgentsEnv = process.env.GH_AW_AGENT_ALLOWED?.trim();
  const allowedAgents = allowedAgentsEnv
    ? allowedAgentsEnv
        .split(",")
        .map(a => a.trim())
        .filter(a => a)
    : null;
  if (allowedAgents) {
    core.info(`Allowed agents: ${allowedAgents.join(", ")}`);
  }

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

  // Get default target repository and allowed repos using standardized helpers
  const { defaultTargetRepo, allowedRepos } = resolveTargetRepoConfig({
    allowed_repos: process.env.GH_AW_ALLOWED_REPOS,
  });
  core.info(`Default target repo: ${defaultTargetRepo}`);
  if (allowedRepos.size > 0) {
    core.info(`Allowed repos: ${[...allowedRepos].join(", ")}`);
  }

  // The github-token is set at the step level, so the built-in github object is authenticated
  // with the correct token (GH_AW_AGENT_TOKEN by default)

  // Get PR repository configuration (where the PR should be created, may differ from issue repo)
  const pullRequestRepoEnv = process.env.GH_AW_AGENT_PULL_REQUEST_REPO?.trim();
  let pullRequestOwner = null;
  let pullRequestRepo = null;
  let pullRequestRepoId = null;
  // Effective base branch: explicit config > fetched default branch from PR repo
  let effectiveBaseBranch = configuredBaseBranch || null;
  // Resolved default branch fetched from the target PR repo (used in NOT clause of branch instructions)
  let resolvedDefaultBranch = null;

  // Get allowed PR repos configuration for cross-repo validation
  const allowedPullRequestReposEnv = process.env.GH_AW_AGENT_ALLOWED_PULL_REQUEST_REPOS?.trim();
  const allowedPullRequestRepos = parseAllowedRepos(allowedPullRequestReposEnv);

  if (pullRequestRepoEnv) {
    const parts = pullRequestRepoEnv.split("/");
    if (parts.length === 2) {
      // Validate PR repository against allowlist
      // The configured pull-request-repo is treated as the default (always allowed)
      // allowed-pull-request-repos contains additional repositories beyond pull-request-repo
      const repoValidation = validateRepo(pullRequestRepoEnv, pullRequestRepoEnv, allowedPullRequestRepos);
      if (!repoValidation.valid) {
        core.setFailed(`E004: ${repoValidation.error}`);
        return;
      }

      pullRequestOwner = parts[0];
      pullRequestRepo = parts[1];
      core.info(`Using pull request repository: ${pullRequestOwner}/${pullRequestRepo}`);

      // Fetch the repository ID and default branch for the PR repo
      try {
        const resolved = await resolvePullRequestRepo(github, pullRequestOwner, pullRequestRepo, configuredBaseBranch);
        pullRequestRepoId = resolved.repoId;
        effectiveBaseBranch = resolved.effectiveBaseBranch;
        resolvedDefaultBranch = resolved.resolvedDefaultBranch;
        core.info(`Pull request repository ID: ${pullRequestRepoId}`);
        if (!configuredBaseBranch && effectiveBaseBranch) {
          core.info(`Resolved pull request repository default branch: ${effectiveBaseBranch}`);
        }
      } catch (error) {
        core.setFailed(`Failed to fetch pull request repository ID for ${pullRequestOwner}/${pullRequestRepo}: ${getErrorMessage(error)}`);
        return;
      }
    } else {
      core.warning(`Invalid pull-request-repo format: ${pullRequestRepoEnv}. Expected owner/repo. PRs will be created in issue repository.`);
    }
  }

  // Cache agent IDs to avoid repeated lookups
  const agentCache = {};

  // Process each agent assignment
  const results = [];
  for (let i = 0; i < itemsToProcess.length; i++) {
    const item = itemsToProcess[i];
    const agentName = item.agent ?? defaultAgent;
    // Model, custom agent, and custom instructions are only configurable via frontmatter defaults
    // They are NOT available as per-item overrides in the tool call
    const model = defaultModel;
    const customAgent = defaultCustomAgent;
    // Build effective custom instructions: prepend base-branch instruction when needed
    let customInstructions = defaultCustomInstructions;
    if (effectiveBaseBranch) {
      const branchInstruction = buildBranchInstruction(effectiveBaseBranch, resolvedDefaultBranch);
      customInstructions = customInstructions ? `${branchInstruction}\n\n${customInstructions}` : branchInstruction;
    }

    // Use these variables to allow temporary IDs to override target repo per-item.
    // Default to the per-item resolved repo (from item.repo or defaultTargetRepo).
    let effectiveOwner;
    let effectiveRepo;

    // Use a copy for target resolution so we never mutate the original item.
    let itemForTarget = item;

    // Validate that both issue_number and pull_number are not specified simultaneously
    if (item.issue_number != null && item.pull_number != null) {
      core.error("Cannot specify both issue_number and pull_number in the same assign_to_agent item");
      results.push({
        issue_number: item.issue_number,
        pull_number: item.pull_number,
        agent: agentName,
        success: false,
        error: "Cannot specify both issue_number and pull_number",
      });
      continue;
    }

    // Resolve and validate target repository for this item using the standardized helper.
    // item.repo field (if present) overrides the default target repo.
    const repoResult = resolveAndValidateRepo(item, defaultTargetRepo, allowedRepos, "issue/PR");
    if (!repoResult.success) {
      core.error(`E004: ${repoResult.error}`);
      results.push({
        issue_number: item.issue_number || null,
        pull_number: item.pull_number || null,
        agent: agentName,
        success: false,
        error: repoResult.error,
      });
      continue;
    }
    effectiveOwner = repoResult.repoParts.owner;
    effectiveRepo = repoResult.repoParts.repo;

    // If issue_number is a temporary ID (aw_...), resolve it to a real issue number before calling resolveTarget.
    // resolveTarget parses issue_number as a number, so we must resolve temporary IDs first.
    // Note: We only support temporary IDs for issues, not PRs.
    if (item.issue_number != null) {
      const resolvedTarget = resolveRepoIssueTarget(item.issue_number, temporaryIdMap, effectiveOwner, effectiveRepo);
      if (!resolvedTarget.resolved) {
        core.error(resolvedTarget.errorMessage || `Failed to resolve issue target: ${item.issue_number}`);
        results.push({
          issue_number: item.issue_number,
          pull_number: item.pull_number ?? null,
          agent: agentName,
          success: false,
          error: resolvedTarget.errorMessage || `Failed to resolve issue target: ${item.issue_number}`,
        });
        continue;
      }

      effectiveOwner = resolvedTarget.resolved.owner;
      effectiveRepo = resolvedTarget.resolved.repo;
      itemForTarget = { ...item, issue_number: resolvedTarget.resolved.number };
      if (resolvedTarget.wasTemporaryId) {
        core.info(`Resolved temporary issue id to ${effectiveOwner}/${effectiveRepo}#${resolvedTarget.resolved.number}`);
      }
    }

    // Determine the effective target configuration:
    // - If issue_number or pull_number is explicitly provided, use "*" (explicit mode)
    // - Otherwise use the configured target (defaults to "triggering")
    const hasExplicitTarget = itemForTarget.issue_number != null || itemForTarget.pull_number != null;
    const effectiveTarget = hasExplicitTarget ? "*" : targetConfig;

    // Handle per-item pull_request_repo parameter (where the PR should be created)
    // This overrides the global pull-request-repo configuration if specified
    let effectivePullRequestRepoId = pullRequestRepoId;
    if (item.pull_request_repo) {
      const itemPullRequestRepo = item.pull_request_repo.trim();
      const pullRequestRepoParts = itemPullRequestRepo.split("/");
      if (pullRequestRepoParts.length === 2) {
        // Validate PR repository against allowlist
        // The global pull-request-repo (if set) is treated as the default (always allowed)
        // allowed-pull-request-repos contains additional allowed repositories
        const defaultPullRequestRepo = pullRequestRepoEnv || defaultTargetRepo;
        const pullRequestRepoValidation = validateRepo(itemPullRequestRepo, defaultPullRequestRepo, allowedPullRequestRepos);
        if (!pullRequestRepoValidation.valid) {
          core.error(`E004: ${pullRequestRepoValidation.error}`);
          results.push({
            issue_number: item.issue_number || null,
            pull_number: item.pull_number || null,
            agent: agentName,
            success: false,
            error: pullRequestRepoValidation.error,
          });
          continue;
        }

        // Fetch the repository ID for the item's PR repo
        try {
          const itemPullRequestRepoQuery = `
            query($owner: String!, $name: String!) {
              repository(owner: $owner, name: $name) {
                id
              }
            }
          `;
          const itemPullRequestRepoResponse = await github.graphql(itemPullRequestRepoQuery, { owner: pullRequestRepoParts[0], name: pullRequestRepoParts[1] });
          effectivePullRequestRepoId = itemPullRequestRepoResponse.repository.id;
          core.info(`Using per-item pull request repository: ${itemPullRequestRepo} (ID: ${effectivePullRequestRepoId})`);
        } catch (error) {
          core.error(`Failed to fetch pull request repository ID for ${itemPullRequestRepo}: ${getErrorMessage(error)}`);
          results.push({
            issue_number: item.issue_number || null,
            pull_number: item.pull_number || null,
            agent: agentName,
            success: false,
            error: `Failed to fetch pull request repository ID for ${itemPullRequestRepo}`,
          });
          continue;
        }
      } else {
        core.warning(`Invalid pull_request_repo format: ${itemPullRequestRepo}. Expected owner/repo. Using global pull-request-repo if configured.`);
      }
    }

    // Resolve target number using the same logic as other safe outputs
    // This allows automatic resolution from workflow context when issue_number/pull_number is not explicitly provided
    const targetResult = resolveTarget({
      targetConfig: effectiveTarget,
      item: itemForTarget,
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

    // Check if agent is in allowed list (if configured)
    if (allowedAgents && !allowedAgents.includes(agentName)) {
      core.error(`Agent "${agentName}" is not in the allowed list. Allowed agents: ${allowedAgents.join(", ")}`);
      results.push({
        issue_number: issueNumber,
        pull_number: pullNumber,
        agent: agentName,
        success: false,
        error: `Agent not allowed: ${agentName}`,
      });
      continue;
    }

    // Assign the agent to the issue or PR using GraphQL
    try {
      // Find agent (use cache if available) - uses built-in github object authenticated via github-token
      let agentId = agentCache[agentName];
      if (!agentId) {
        core.info(`Looking for ${agentName} coding agent...`);
        agentId = await findAgent(effectiveOwner, effectiveRepo, agentName);
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
        const issueDetails = await getIssueDetails(effectiveOwner, effectiveRepo, issueNumber);
        if (!issueDetails) {
          throw new Error(`Failed to get issue details`);
        }
        assignableId = issueDetails.issueId;
        currentAssignees = issueDetails.currentAssignees;
      } else if (pullNumber) {
        const prDetails = await getPullRequestDetails(effectiveOwner, effectiveRepo, pullNumber);
        if (!prDetails) {
          throw new Error(`Failed to get pull request details`);
        }
        assignableId = prDetails.pullRequestId;
        currentAssignees = prDetails.currentAssignees;
      } else {
        // This should never happen due to resolveTarget logic, but TypeScript needs it
        throw new Error(`No issue or pull request number available`);
      }

      core.info(`${type} ID: ${assignableId}`);

      // Check if agent is already assigned
      if (currentAssignees.some(a => a.id === agentId)) {
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
      // Pass the allowed list so existing assignees are filtered before calling replaceActorsForAssignable
      // Pass the PR repo ID if configured (to specify where the PR should be created)
      // Pass model, customAgent, and customInstructions if specified
      core.info(`Assigning ${agentName} coding agent to ${type} #${number}...`);
      if (model) {
        core.info(`Using model: ${model}`);
      }
      if (customAgent) {
        core.info(`Using custom agent: ${customAgent}`);
      }
      if (customInstructions) {
        core.info(`Using custom instructions: ${customInstructions.substring(0, 100)}${customInstructions.length > 100 ? "..." : ""}`);
      }
      const success = await assignAgentToIssue(assignableId, agentId, currentAssignees, agentName, allowedAgents, effectivePullRequestRepoId, model, customAgent, customInstructions);

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

      // Check if this is a token authentication error
      const isAuthError =
        errorMessage.includes("Bad credentials") ||
        errorMessage.includes("Not Authenticated") ||
        errorMessage.includes("Resource not accessible") ||
        errorMessage.includes("Insufficient permissions") ||
        errorMessage.includes("requires authentication");

      // If ignore-if-error is enabled and this is an auth error, log warning and skip
      if (ignoreIfError && isAuthError) {
        core.warning(`Agent assignment failed for ${agentName} on ${type} #${number} due to authentication/permission error. Skipping due to ignore-if-error=true.`);
        core.info(`Error details: ${errorMessage}`);
        results.push({
          issue_number: issueNumber,
          pull_number: pullNumber,
          agent: agentName,
          success: true, // Treat as success when ignored
          skipped: true,
        });
        continue;
      }

      if (errorMessage.includes("coding agent is not available for this repository")) {
        // Enrich with available agent logins to aid troubleshooting - uses built-in github object
        try {
          const available = await getAvailableAgentLogins(effectiveOwner, effectiveRepo);
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

    // Add 10-second delay between agent assignments to avoid spawning too many agents at once
    // Skip delay after the last item
    if (i < itemsToProcess.length - 1) {
      core.info("Waiting 10 seconds before processing next agent assignment...");
      await sleep(10000);
    }
  }

  // Generate step summary
  const successCount = results.filter(r => r.success && !r.skipped).length;
  const skippedCount = results.filter(r => r.skipped).length;
  const failureCount = results.length - successCount - skippedCount;

  let summaryContent = "## Agent Assignment\n\n";

  if (successCount > 0) {
    summaryContent += `✅ Successfully assigned ${successCount} agent(s):\n\n`;
    summaryContent += results
      .filter(r => r.success && !r.skipped)
      .map(r => {
        const itemType = r.issue_number ? `Issue #${r.issue_number}` : `Pull Request #${r.pull_number}`;
        return `- ${itemType} → Agent: ${r.agent}`;
      })
      .join("\n");
    summaryContent += "\n\n";
  }

  if (skippedCount > 0) {
    summaryContent += `⏭️ Skipped ${skippedCount} agent assignment(s) (ignore-if-error enabled):\n\n`;
    summaryContent += results
      .filter(r => r.skipped)
      .map(r => {
        const itemType = r.issue_number ? `Issue #${r.issue_number}` : `Pull Request #${r.pull_number}`;
        return `- ${itemType} → Agent: ${r.agent} (assignment failed due to error)`;
      })
      .join("\n");
    summaryContent += "\n\n";
  }

  if (failureCount > 0) {
    summaryContent += `❌ Failed to assign ${failureCount} agent(s):\n\n`;
    summaryContent += results
      .filter(r => !r.success && !r.skipped)
      .map(r => {
        const itemType = r.issue_number ? `Issue #${r.issue_number}` : `Pull Request #${r.pull_number}`;
        return `- ${itemType} → Agent: ${r.agent}: ${r.error}`;
      })
      .join("\n");

    // Check if any failures were permission-related
    const hasPermissionError = results.some(r => (!r.success && !r.skipped && r.error?.includes("Resource not accessible")) || r.error?.includes("Insufficient permissions"));

    if (hasPermissionError) {
      summaryContent += generatePermissionErrorSummary();
    }
  }

  await core.summary.addRaw(summaryContent).write();

  // Set outputs
  const assignedAgents = results
    .filter(r => r.success && !r.skipped)
    .map(r => {
      const number = r.issue_number || r.pull_number;
      const prefix = r.issue_number ? "issue" : "pr";
      return `${prefix}:${number}:${r.agent}`;
    })
    .join("\n");
  core.setOutput("assigned_agents", assignedAgents);

  // Set assignment error output for failed assignments
  const assignmentErrors = results
    .filter(r => !r.success && !r.skipped)
    .map(r => {
      const number = r.issue_number || r.pull_number;
      const prefix = r.issue_number ? "issue" : "pr";
      return `${prefix}:${number}:${r.agent}:${r.error}`;
    })
    .join("\n");
  core.setOutput("assignment_errors", assignmentErrors);
  core.setOutput("assignment_error_count", failureCount.toString());

  // Log assignment failures but don't fail the job
  // The conclusion job will report these failures in the agent failure issue/comment
  if (failureCount > 0) {
    core.warning(`Failed to assign ${failureCount} agent(s) - errors will be reported in conclusion job`);
  }
}

module.exports = { main };
