// @ts-check
/// <reference types="@actions/github-script" />

const { getErrorMessage } = require("./error_helpers.cjs");
const { sanitizeContent } = require("./sanitize_content.cjs");
const { getDetectionCautionAlert, getFooterAgentFailureIssueMessage, getFooterAgentFailureCommentMessage, generateXMLMarker } = require("./messages.cjs");
const { renderTemplate, renderTemplateFromFile, getPromptPath } = require("./messages_core.cjs");
const { getCurrentBranch } = require("./get_current_branch.cjs");
const { createExpirationLine, generateFooterWithExpiration } = require("./ephemerals.cjs");
const { MAX_SUB_ISSUES, getSubIssueCount } = require("./sub_issue_helpers.cjs");
const { formatMissingData } = require("./missing_info_formatter.cjs");
const { generateHistoryUrl } = require("./generate_history_link.cjs");
const { AWF_INFRA_LINE_RE } = require("./log_parser_shared.cjs");
const fs = require("fs");
const path = require("path");

const DEFAULT_ACTION_FAILURE_ISSUE_EXPIRES_HOURS = 24 * 7;

/**
 * Parse action failure issue expiration from environment.
 * @returns {number} Expiration in hours (defaults to 168 when unset/invalid)
 */
function getActionFailureIssueExpiresHours() {
  const raw = process.env.GH_AW_ACTION_FAILURE_ISSUE_EXPIRES_HOURS || "";
  const parsed = Number.parseInt(raw, 10);
  if (Number.isInteger(parsed) && parsed > 0) {
    return parsed;
  }
  return DEFAULT_ACTION_FAILURE_ISSUE_EXPIRES_HOURS;
}

/**
 * Attempt to find a pull request for the current branch
 * @returns {Promise<{number: number, html_url: string, head_sha: string, mergeable: boolean | null, mergeable_state: string, updated_at: string} | null>} PR info or null if not found
 */
async function findPullRequestForCurrentBranch() {
  try {
    const { owner, repo } = context.repo;
    const currentBranch = getCurrentBranch();

    core.info(`Searching for pull request from branch: ${currentBranch}`);

    // Search for open PRs with the current branch as head
    const searchQuery = `repo:${owner}/${repo} is:pr is:open head:${currentBranch}`;

    const searchResult = await github.rest.search.issuesAndPullRequests({
      q: searchQuery,
      per_page: 1,
    });

    if (searchResult.data.total_count > 0) {
      const pr = searchResult.data.items[0];
      core.info(`Found pull request #${pr.number}: ${pr.html_url}`);

      // Fetch detailed PR info to get mergeable state and head SHA
      try {
        const detailedPR = await github.rest.pulls.get({
          owner,
          repo,
          pull_number: pr.number,
        });

        core.info(`PR #${pr.number} details - head_sha: ${detailedPR.data.head.sha}, mergeable: ${detailedPR.data.mergeable}, mergeable_state: ${detailedPR.data.mergeable_state}`);

        return {
          number: pr.number,
          html_url: pr.html_url,
          head_sha: detailedPR.data.head.sha,
          mergeable: detailedPR.data.mergeable,
          mergeable_state: detailedPR.data.mergeable_state || "unknown",
          updated_at: detailedPR.data.updated_at,
        };
      } catch (detailsError) {
        core.warning(`Failed to fetch detailed PR info: ${getErrorMessage(detailsError)}`);
        // Fall back to basic info
        return {
          number: pr.number,
          html_url: pr.html_url,
          head_sha: "",
          mergeable: null,
          mergeable_state: "unknown",
          updated_at: "",
        };
      }
    }

    core.info(`No pull request found for branch: ${currentBranch}`);
    return null;
  } catch (error) {
    core.warning(`Failed to find pull request for current branch: ${getErrorMessage(error)}`);
    return null;
  }
}

/**
 * Search for or create the parent issue for all agentic workflow failures
 * @param {number|null} previousParentNumber - Previous parent issue number if creating due to limit
 * @param {string} [ownerOverride] - Repository owner override (from failure-issue-repo config)
 * @param {string} [repoOverride] - Repository name override (from failure-issue-repo config)
 * @param {number} [expiresHours] - Expiration in hours for created parent issue
 * @returns {Promise<{number: number, node_id: string}>} Parent issue number and node ID
 */
async function ensureParentIssue(previousParentNumber = null, ownerOverride, repoOverride, expiresHours = DEFAULT_ACTION_FAILURE_ISSUE_EXPIRES_HOURS) {
  const { owner: contextOwner, repo: contextRepo } = context.repo;
  const owner = ownerOverride || contextOwner;
  const repo = repoOverride || contextRepo;
  const parentTitle = "[aw] Failed runs";
  const parentLabel = "agentic-workflows";

  core.info(`Searching for parent issue: "${parentTitle}"`);

  // Search for existing parent issue
  const searchQuery = `repo:${owner}/${repo} is:issue is:open label:${parentLabel} in:title "${parentTitle}"`;

  try {
    const searchResult = await github.rest.search.issuesAndPullRequests({
      q: searchQuery,
      per_page: 1,
    });

    if (searchResult.data.total_count > 0) {
      const existingIssue = searchResult.data.items[0];
      core.info(`Found existing parent issue #${existingIssue.number}: ${existingIssue.html_url}`);

      // Check the sub-issue count
      const subIssueCount = await getSubIssueCount(owner, repo, existingIssue.number);

      if (subIssueCount !== null && subIssueCount >= MAX_SUB_ISSUES) {
        core.warning(`Parent issue #${existingIssue.number} has ${subIssueCount} sub-issues (max: ${MAX_SUB_ISSUES})`);
        core.info(`Creating a new parent issue (previous parent #${existingIssue.number} is full)`);

        // Fall through to create a new parent issue, passing the previous parent number
        previousParentNumber = existingIssue.number;
      } else {
        // Parent issue is within limits, return it
        if (subIssueCount !== null) {
          core.info(`Parent issue has ${subIssueCount} sub-issues (within limit of ${MAX_SUB_ISSUES})`);
        }
        return {
          number: existingIssue.number,
          node_id: existingIssue.node_id,
        };
      }
    }
  } catch (error) {
    core.warning(`Error searching for parent issue: ${getErrorMessage(error)}`);
  }

  // Create parent issue if it doesn't exist or if previous one is full
  const creationReason = previousParentNumber ? `creating new parent (previous #${previousParentNumber} reached limit)` : "creating first parent";
  core.info(`No suitable parent issue found, ${creationReason}`);

  let parentBodyContent = `This issue tracks all failures from agentic workflows in this repository. Each failed workflow run creates a sub-issue linked here for organization and easy filtering.`;

  // Add reference to previous parent if this is a continuation
  if (previousParentNumber) {
    parentBodyContent += `

> **Note:** This is a continuation parent issue. The previous parent issue #${previousParentNumber} reached the maximum of ${MAX_SUB_ISSUES} sub-issues.`;
  }

  parentBodyContent += `

### Purpose

This parent issue helps you:
- View all workflow failures in one place by checking the sub-issues below
- Filter out failure issues from your main issue list using \`no:parent-issue\`
- Track the health of your agentic workflows over time

### Sub-Issues

All individual workflow failure issues are linked as sub-issues below. Click on any sub-issue to see details about a specific failure.

### Troubleshooting Failed Workflows

#### Using agentic-workflows Agent (Recommended)

**Agent:** \`agentic-workflows\`  
**Purpose:** Debug and fix workflow failures

**Instructions:**

1. Invoke the agent: Type \`/agent\` in GitHub Copilot Chat and select **agentic-workflows**
2. Provide context: Tell the agent to **debug** the workflow failure
3. Supply the workflow run URL for analysis
4. The agent will:
   - Analyze failure logs
   - Identify root causes
   - Propose specific fixes
   - Validate solutions

#### Using gh aw CLI

You can also debug failures using the \`gh aw\` CLI:

\`\`\`bash
# Download and analyze workflow logs
gh aw logs <workflow-run-url>

# Audit a specific workflow run
gh aw audit <run-id>
\`\`\`

#### Manual Investigation

1. Click on a sub-issue to see the failed workflow details
2. Follow the workflow run link in the issue
3. Review the agent job logs for error messages
4. Check the workflow configuration in your repository

### Resources

- [GitHub Agentic Workflows Documentation](https://github.com/github/gh-aw)
- [Troubleshooting Guide](https://github.github.com/gh-aw/troubleshooting/common-issues/)

---

> This issue is automatically managed by GitHub Agentic Workflows. Do not close this issue manually.`;

  // Add expiration marker inside the quoted section using helper
  const footer = generateFooterWithExpiration({
    footerText: parentBodyContent,
    expiresHours,
  });
  const parentBody = footer;

  try {
    const newIssue = await github.rest.issues.create({
      owner,
      repo,
      title: parentTitle,
      body: parentBody,
      labels: [parentLabel],
    });

    core.info(`✓ Created parent issue #${newIssue.data.number}: ${newIssue.data.html_url}`);
    return {
      number: newIssue.data.number,
      node_id: newIssue.data.node_id,
    };
  } catch (error) {
    core.error(`Failed to create parent issue: ${getErrorMessage(error)}`);
    throw error;
  }
}

/**
 * Link an issue as a sub-issue to a parent issue
 * @param {string} parentNodeId - GraphQL node ID of the parent issue
 * @param {string} subIssueNodeId - GraphQL node ID of the sub-issue
 * @param {number} parentNumber - Parent issue number (for logging)
 * @param {number} subIssueNumber - Sub-issue number (for logging)
 */
async function linkSubIssue(parentNodeId, subIssueNodeId, parentNumber, subIssueNumber) {
  core.info(`Linking issue #${subIssueNumber} as sub-issue of #${parentNumber}`);

  try {
    // Use GraphQL to link the sub-issue
    await github.graphql(
      `mutation($parentId: ID!, $subIssueId: ID!) {
        addSubIssue(input: {issueId: $parentId, subIssueId: $subIssueId}) {
          issue {
            id
            number
          }
          subIssue {
            id
            number
          }
        }
      }`,
      {
        parentId: parentNodeId,
        subIssueId: subIssueNodeId,
      }
    );

    core.info(`✓ Successfully linked #${subIssueNumber} as sub-issue of #${parentNumber}`);
  } catch (error) {
    const errorMessage = getErrorMessage(error);
    if (errorMessage.includes("Field 'addSubIssue' doesn't exist") || errorMessage.includes("not yet available")) {
      core.warning(`Sub-issue API not available. Issue #${subIssueNumber} created but not linked to parent.`);
    } else {
      core.warning(`Failed to link sub-issue: ${errorMessage}`);
    }
  }
}

/**
 * Build create_discussion errors context string from error environment variable
 * @param {string} createDiscussionErrors - Newline-separated error strings
 * @returns {string} Formatted error context for display
 */
function buildCreateDiscussionErrorsContext(createDiscussionErrors) {
  if (!createDiscussionErrors) {
    return "";
  }

  let context = "\n**⚠️ Create Discussion Failed**: Failed to create one or more discussions.\n\n**Discussion Errors:**\n";
  const errorLines = createDiscussionErrors.split("\n").filter(line => line.trim());
  for (const errorLine of errorLines) {
    const parts = errorLine.split(":");
    if (parts.length >= 4) {
      // parts[0] is "discussion", parts[1] is index - both unused
      const repo = parts[2];
      const title = parts[3];
      const error = parts.slice(4).join(":"); // Rest is the error message
      context += `- Discussion "${title}" in ${repo}: ${error}\n`;
    }
  }
  context += "\n";
  return context;
}

/**
 * Build a fork context hint string when the repository is a fork.
 * @returns {string} Fork hint string, or empty string if not a fork
 */
function buildForkContextHint() {
  if (context.payload?.repository?.fork) {
    return "\n💡 **This repository is a fork.** If this failure is due to missing API keys or tokens, note that secrets from the parent repository are not inherited. Configure the required secrets directly in your fork's Settings → Secrets and variables → Actions.\n";
  }
  return "";
}

/**
 * Build a context string describing code-push failures for inclusion in failure issue/comment bodies.
 * Manifest file protection refusals are separated from other push failures to give them a dedicated
 * section with clearer remediation instructions.
 * @param {string} codePushFailureErrors - Newline-separated list of "type:error" entries
 * @param {{number: number, html_url: string, head_sha?: string, mergeable?: boolean | null, mergeable_state?: string, updated_at?: string} | null} pullRequest - PR info if available
 * @param {string} [runUrl] - URL of the current workflow run, used to provide patch download instructions
 * @returns {string} Formatted context string, or empty string if no failures
 */
function buildCodePushFailureContext(codePushFailureErrors, pullRequest = null, runUrl = "") {
  if (!codePushFailureErrors) {
    return "";
  }

  // Split errors into protected-file protection refusals, patch size errors, patch apply failures, and other push failures
  const manifestErrors = [];
  const patchSizeErrors = [];
  const patchApplyErrors = [];
  const otherErrors = [];
  const errorLines = codePushFailureErrors.split("\n").filter(line => line.trim());
  for (const errorLine of errorLines) {
    const colonIndex = errorLine.indexOf(":");
    if (colonIndex !== -1) {
      const type = errorLine.substring(0, colonIndex);
      const error = errorLine.substring(colonIndex + 1);
      if (error.includes("manifest files") || error.includes("protected files")) {
        manifestErrors.push({ type, error });
      } else if (error.includes("Patch size") && error.includes("exceeds")) {
        patchSizeErrors.push({ type, error });
      } else if (error.includes("Failed to apply patch")) {
        patchApplyErrors.push({ type, error });
      } else {
        otherErrors.push({ type, error });
      }
    }
  }

  let context = "";

  // Protected file protection section — shown before generic failures
  if (manifestErrors.length > 0) {
    context +=
      "\n**🛡️ Protected Files**: The code push was refused because the patch modifies protected files (package manifests, agent instruction files, or repository security configuration). " +
      "This protection guards against unintended supply chain changes.\n";
    if (pullRequest) {
      context += `\n**Target Pull Request:** [#${pullRequest.number}](${pullRequest.html_url})\n`;
    }
    context += "\n**Blocked Operations:**\n";
    for (const { type, error } of manifestErrors) {
      context += `- \`${type}\`: ${error}\n`;
    }
    // Build a dynamic YAML snippet listing only the safe output types that were actually blocked
    const typeToYamlKey = {
      create_pull_request: "create-pull-request",
      push_to_pull_request_branch: "push-to-pull-request-branch",
    };
    const blockedTypes = [...new Set(manifestErrors.map(e => e.type))];
    let yamlSnippet = "```yaml\nsafe-outputs:\n";
    for (const type of blockedTypes) {
      const yamlKey = typeToYamlKey[type] || type.replace(/_/g, "-");
      yamlSnippet += `  ${yamlKey}:\n    protected-files: fallback-to-issue\n`;
    }
    yamlSnippet += "```\n";
    context += "\n<details>\n<summary>⚙️ Configure <code>protected-files: fallback-to-issue</code></summary>\n\n";
    context += yamlSnippet;
    context += "</details>\n";
  }

  // Patch size exceeded section
  if (patchSizeErrors.length > 0) {
    context += "\n**📦 Patch Size Exceeded**: The code push was rejected because the generated patch is too large.\n";
    if (pullRequest) {
      context += `\n**Target Pull Request:** [#${pullRequest.number}](${pullRequest.html_url})\n`;
    }
    context += "\n**Errors:**\n";
    for (const { type, error } of patchSizeErrors) {
      context += `- \`${type}\`: ${error}\n`;
    }
    // Build a dynamic YAML snippet listing only the safe output types that had patch size errors
    const typeToYamlKey = {
      create_pull_request: "create-pull-request",
      push_to_pull_request_branch: "push-to-pull-request-branch",
    };
    const affectedTypes = [...new Set(patchSizeErrors.map(e => e.type))];
    let yamlSnippet = "```yaml\nsafe-outputs:\n";
    for (const type of affectedTypes) {
      const yamlKey = typeToYamlKey[type] || type.replace(/_/g, "-");
      yamlSnippet += `  ${yamlKey}:\n    max-patch-size: 2048  # Example: double the default limit (in KB, default: 1024)\n`;
    }
    yamlSnippet += "```\n";
    context += "\nTo allow larger patches, increase `max-patch-size` in your workflow's front matter (value in KB):\n";
    context += yamlSnippet;
  }

  // Patch apply failure section — shown when the patch could not be applied (e.g. merge conflict)
  if (patchApplyErrors.length > 0) {
    context += "\n**🔀 Patch Apply Failed**: The patch could not be applied to the current state of the repository. " + "This is typically caused by a merge conflict between the agent's changes and recent commits on the target branch.\n";
    if (pullRequest) {
      context += `\n**Target Pull Request:** [#${pullRequest.number}](${pullRequest.html_url})\n`;
    }
    context += "\n**Failed Operations:**\n";
    for (const { type, error } of patchApplyErrors) {
      context += `- \`${type}\`: ${error}\n`;
    }

    // Extract run ID from runUrl for use in the download command
    let runId = "";
    if (runUrl) {
      const runIdMatch = runUrl.match(/\/actions\/runs\/(\d+)/);
      if (runIdMatch) {
        runId = runIdMatch[1];
      }
    }

    context += "\n<details>\n<summary>📋 Apply the patch manually</summary>\n\n";
    if (runId) {
      context += `\`\`\`sh
# Download the patch artifact from the workflow run
gh run download ${runId} -n agent -D /tmp/agent-${runId}

# List available patches
ls /tmp/agent-${runId}/*.patch

# Create a new branch (adjust as needed)
git checkout -b aw/manual-apply

# Apply the patch (--3way handles cross-repo patches)
git am --3way /tmp/agent-${runId}/YOUR_PATCH_FILE.patch

# If there are conflicts, resolve them and continue (or abort):
# git am --continue
# git am --abort

# Push and create a pull request
git push origin aw/manual-apply
gh pr create --head aw/manual-apply
\`\`\`
${runUrl ? `\nThe patch artifact is available at: [View run and download artifacts](${runUrl})\n` : ""}`;
    } else {
      context += "Download the patch artifact from the workflow run, then apply it with `git am --3way <patch-file>`.\n";
    }
    context += "\n</details>\n";
  }

  // Generic code-push failure section
  if (otherErrors.length > 0) {
    context += "\n**⚠️ Code Push Failed**: A code push safe output failed, and subsequent safe outputs were cancelled.";
    if (pullRequest) {
      context += `\n\n**Target Pull Request:** [#${pullRequest.number}](${pullRequest.html_url})`;

      // Add PR state diagnostics
      const workflowSha = process.env.GITHUB_SHA || "";
      const prDetails = [];

      // Check for merge conflicts
      if (pullRequest.mergeable === false) {
        prDetails.push("❌ **Merge conflicts detected** - the PR has conflicts that need resolution");
      } else if (pullRequest.mergeable_state === "dirty") {
        prDetails.push("❌ **PR is in dirty state** - likely has merge conflicts");
      } else if (pullRequest.mergeable_state === "blocked") {
        prDetails.push("⚠️ **PR is blocked** - required status checks or reviews may be missing");
      } else if (pullRequest.mergeable_state === "behind") {
        prDetails.push("⚠️ **PR is behind base branch** - may need to be updated");
      }

      // Check if branch was updated since workflow started
      if (workflowSha && pullRequest.head_sha && workflowSha !== pullRequest.head_sha) {
        prDetails.push(`⚠️ **Branch was updated** - workflow started at \`${workflowSha.substring(0, 7)}\`, PR head is now \`${pullRequest.head_sha.substring(0, 7)}\``);
      }

      // Add SHA info for debugging
      if (pullRequest.head_sha) {
        prDetails.push(`**PR head SHA:** \`${pullRequest.head_sha.substring(0, 7)}\``);
      }
      if (workflowSha) {
        prDetails.push(`**Workflow SHA:** \`${workflowSha.substring(0, 7)}\``);
      }
      if (pullRequest.mergeable_state && pullRequest.mergeable_state !== "unknown") {
        prDetails.push(`**Mergeable state:** ${pullRequest.mergeable_state}`);
      }

      if (prDetails.length > 0) {
        context += "\n\n**PR State at Push Time:**\n";
        for (const detail of prDetails) {
          context += `- ${detail}\n`;
        }
      }
    }
    context += "\n**Code Push Errors:**\n";
    for (const { type, error } of otherErrors) {
      context += `- \`${type}\`: ${error}\n`;
    }
    context += "\n";
  } else if (manifestErrors.length > 0 || patchSizeErrors.length > 0 || patchApplyErrors.length > 0) {
    // Only manifest, patch size, or patch apply errors — ensure trailing newline
    context += "\n";
  }

  return context;
}

/**
 * Build a context string for push_repo_memory job failures, with a dedicated section for patch size errors.
 * @param {boolean} hasPushRepoMemoryFailure - Whether the push_repo_memory job failed
 * @param {string[]} repoMemoryPatchSizeExceededIDs - Memory IDs that exceeded the patch size limit
 * @param {string} runUrl - URL of the current workflow run
 * @returns {string} Formatted context string, or empty string if no failure
 */
function buildPushRepoMemoryFailureContext(hasPushRepoMemoryFailure, repoMemoryPatchSizeExceededIDs, runUrl) {
  if (!hasPushRepoMemoryFailure) {
    return "";
  }
  if (repoMemoryPatchSizeExceededIDs.length > 0) {
    let context = "\n**📦 Repo-Memory Patch Size Exceeded**: The repo-memory push failed because the memory data is too large.\n";
    context += "\n**Affected memories:** " + repoMemoryPatchSizeExceededIDs.map(id => `\`${id}\``).join(", ") + "\n";
    context += "\nTo allow larger memory snapshots, increase `max-patch-size` in your workflow's `repo-memory` front matter (value in bytes):\n";
    context += "```yaml\nrepo-memory:\n";
    for (const memoryID of repoMemoryPatchSizeExceededIDs) {
      context += `  - id: ${memoryID}\n    max-patch-size: 51200  # Example: 5x the default limit (in bytes, default: 10240, max: 102400)\n`;
    }
    context += "```\n\n";
    return context;
  }
  return (
    "\n**⚠️ Repo-Memory Push Failed**: The push-repo-memory job failed to write memory back to the repository. This may indicate a permission issue, a configuration error, or a network problem. Check the [workflow run](" +
    runUrl +
    ") for details.\n\n"
  );
}

/**
 * Load missing_data messages from agent output
 * @returns {Array<{data_type: string, reason: string, context?: string, alternatives?: string}>} Array of missing data messages
 */
function loadMissingDataMessages() {
  try {
    const { loadAgentOutput } = require("./load_agent_output.cjs");
    const agentOutputResult = loadAgentOutput();

    if (!agentOutputResult.success || !agentOutputResult.items) {
      return [];
    }

    // Extract missing_data messages from agent output
    const missingDataMessages = [];
    for (const item of agentOutputResult.items) {
      if (item.type === "missing_data") {
        // Accept items with at least a reason; data_type may be absent for cache-miss signals
        if (item.reason) {
          missingDataMessages.push({
            data_type: item.data_type || "",
            reason: item.reason,
            context: item.context || null,
            alternatives: item.alternatives || null,
          });
        }
      }
    }

    return missingDataMessages;
  } catch (error) {
    core.warning(`Failed to load missing_data messages: ${getErrorMessage(error)}`);
    return [];
  }
}

/**
 * Build missing_data context string for display in failure issues/comments.
 * When cache-memory is enabled and a cache_miss is detected, appends a
 * configuration-problem warning to the context.
 * @param {boolean} cacheMemoryEnabled - Whether cache-memory is configured for this workflow
 * @returns {string} Formatted missing data context
 */
function buildMissingDataContext(cacheMemoryEnabled) {
  const missingDataMessages = loadMissingDataMessages();

  if (missingDataMessages.length === 0) {
    return "";
  }

  core.info(`Found ${missingDataMessages.length} missing_data message(s)`);

  // Format the missing data using the existing formatter
  const formattedList = formatMissingData(missingDataMessages);

  let context = "\n**⚠️ Missing Data Reported**: The agent reported missing data during execution.\n\n**Missing Data:**\n";
  context += formattedList;
  context += "\n\n";

  // Detect cache_miss: if cache-memory is available and the agent reported a cache miss,
  // this indicates the prompt is referencing an incorrect file path within the cache directory.
  const hasCacheMiss = missingDataMessages.some(m => m.reason === "cache_memory_miss");
  if (cacheMemoryEnabled && hasCacheMiss) {
    core.info("Cache-miss detected despite cache-memory being available — likely a configuration problem");
    const templatePath = getPromptPath("cache_memory_miss.md");
    context += "\n" + renderTemplateFromFile(templatePath, {}) + "\n";
  }

  return context;
}

/**
 * Load report_incomplete messages from agent output
 * @returns {Array<{reason: string, details?: string}>} Array of report_incomplete messages
 */
function loadReportIncompleteMessages() {
  try {
    const { loadAgentOutput } = require("./load_agent_output.cjs");
    const agentOutputResult = loadAgentOutput();

    if (!agentOutputResult.success || !agentOutputResult.items) {
      return [];
    }

    const messages = [];
    for (const item of agentOutputResult.items) {
      if (item.type === "report_incomplete" && item.reason) {
        messages.push({
          reason: item.reason,
          details: item.details || null,
        });
      }
    }

    return messages;
  } catch (error) {
    core.warning(`Failed to load report_incomplete messages: ${getErrorMessage(error)}`);
    return [];
  }
}

/**
 * Build report_incomplete context string for display in failure issues/comments.
 * This surfaces the agent's structured incompletion signal so maintainers can
 * distinguish a tool-failure report from a real task outcome.
 * @returns {string} Formatted report_incomplete context
 */
function buildReportIncompleteContext() {
  const messages = loadReportIncompleteMessages();

  if (messages.length === 0) {
    return "";
  }

  core.info(`Found ${messages.length} report_incomplete signal(s)`);

  let context = "\n**⚠️ Task Could Not Be Completed**: The agent reported that the task could not be performed due to an infrastructure or tool failure.\n\n**Reasons:**\n";
  for (const msg of messages) {
    context += `- ${msg.reason}\n`;
    if (msg.details) {
      context += `  \n  ${msg.details}\n`;
    }
  }
  context +=
    "\nThis is a structured incompletion signal (`report_incomplete`), not a real task outcome. Any other safe outputs emitted alongside this signal (e.g., comments) describe the failure state, not a completed review or action.\n\n";

  return context;
}

/**
 * Build a context string with a frontmatter hint when the agent timed out.
 * @param {boolean} isTimedOut - Whether the agent job timed out
 * @param {string} timeoutMinutes - Current timeout value in minutes (e.g. "20")
 * @returns {string} Formatted timeout context string, or empty string if not timed out
 */
function buildTimeoutContext(isTimedOut, timeoutMinutes) {
  if (!isTimedOut) {
    return "";
  }

  const currentMinutes = parseInt(timeoutMinutes || "20", 10);
  const suggestedMinutes = currentMinutes + 10;

  const templatePath = getPromptPath("agent_timeout.md");
  return "\n" + renderTemplateFromFile(templatePath, { current_minutes: currentMinutes, suggested_minutes: suggestedMinutes });
}

/**
 * Build a context string when the Copilot CLI failed due to the token lacking inference access.
 * @param {boolean} hasInferenceAccessError - Whether an inference access error was detected
 * @returns {string} Formatted context string, or empty string if no error
 */
function buildInferenceAccessErrorContext(hasInferenceAccessError) {
  if (!hasInferenceAccessError) {
    return "";
  }

  const templatePath = getPromptPath("inference_access_error.md");
  const template = fs.readFileSync(templatePath, "utf8");
  return "\n" + template;
}

/**
 * Build a context string when MCP servers were blocked by enterprise/organization policy.
 * This is a persistent configuration error — retrying will not help.
 * @param {boolean} hasMCPPolicyError - Whether an MCP policy error was detected
 * @returns {string} Formatted context string, or empty string if no error
 */
function buildMCPPolicyErrorContext(hasMCPPolicyError) {
  if (!hasMCPPolicyError) {
    return "";
  }

  const templatePath = getPromptPath("mcp_policy_error.md");
  try {
    const template = fs.readFileSync(templatePath, "utf8");
    return "\n" + template;
  } catch {
    // Template not available — return inline message
    return (
      "\n**🔒 MCP Servers Blocked by Policy**: The Copilot CLI blocked MCP server connections due to an organization or enterprise policy.\n\n" +
      'An administrator must enable the **"MCP servers in Copilot"** policy. ' +
      "See: [Configure MCP server access](https://docs.github.com/en/copilot/how-tos/administer-copilot/manage-mcp-usage/configure-mcp-server-access)\n"
    );
  }
}

/**
 * Build a context string when the requested model is not supported for the subscription tier.
 * This is a persistent configuration error — retrying will not help.
 * @param {boolean} hasModelNotSupportedError - Whether a model-not-supported error was detected
 * @returns {string} Formatted context string, or empty string if no error
 */
function buildModelNotSupportedErrorContext(hasModelNotSupportedError) {
  if (!hasModelNotSupportedError) {
    return "";
  }

  const templatePath = getPromptPath("model_not_supported_error.md");
  try {
    const template = fs.readFileSync(templatePath, "utf8");
    return "\n" + template;
  } catch {
    // Template not available — return inline message
    return (
      "\n**🚫 Model Not Supported**: The requested model is not available for your Copilot subscription tier (e.g., Copilot Pro or Education).\n\n" +
      "Specify a supported model in the workflow frontmatter, for example `model: gpt-5-mini`. " +
      "See: [Supported models](https://docs.github.com/en/copilot/using-github-copilot/using-github-copilot-in-the-command-line#supported-models)\n"
    );
  }
}

/**
 * Build a context string when a GitHub App token minting step failed.
 * @param {boolean} hasAppTokenMintingFailed - Whether any GitHub App token minting step failed
 * @returns {string} Formatted context string, or empty string if no error
 */
function buildAppTokenMintingFailedContext(hasAppTokenMintingFailed) {
  if (!hasAppTokenMintingFailed) {
    return "";
  }

  const templatePath = "/opt/gh-aw/prompts/app_token_minting_failed.md";
  return "\n" + renderTemplateFromFile(templatePath, {});
}

/**
 * Build a context string when the lockdown check step failed in the activation job.
 * @param {boolean} hasLockdownCheckFailed - Whether the lockdown check failed
 * @returns {string} Formatted context string, or empty string if no failure
 */
function buildLockdownCheckFailedContext(hasLockdownCheckFailed) {
  if (!hasLockdownCheckFailed) {
    return "";
  }

  const templatePath = getPromptPath("lockdown_check_failed.md");
  const template = fs.readFileSync(templatePath, "utf8");
  return "\n" + template;
}

/**
 * Build a context string when the frontmatter hash (stale lock file) check failed in the
 * activation job. This surfaces remediation guidance — including how to disable the check —
 * directly in the failure issue / comment.
 * @param {boolean} hasStaleLockFileFailed - Whether the stale lock file check failed
 * @returns {string} Formatted context string, or empty string if no failure
 */
function buildStaleLockFileFailedContext(hasStaleLockFileFailed) {
  if (!hasStaleLockFileFailed) {
    return "";
  }

  const templatePath = getPromptPath("stale_lock_file_failed.md");
  const template = fs.readFileSync(templatePath, "utf8");
  return "\n" + template;
}

// Maps engine ID (GH_AW_ENGINE_ID) to credential name for use with GH_AW_ENGINE_API_HOSTS.
const ENGINE_ID_TO_CREDENTIAL = /** @type {Record<string, string>} */ {
  copilot: "`COPILOT_GITHUB_TOKEN`",
  claude: "`ANTHROPIC_API_KEY`",
  codex: "`CODEX_API_KEY` / `OPENAI_API_KEY`",
  gemini: "`GEMINI_API_KEY`",
};

// Maps engine ID to a human-readable provider label.
const ENGINE_ID_TO_LABEL = /** @type {Record<string, string>} */ {
  copilot: "GitHub Copilot",
  claude: "Anthropic Claude",
  codex: "OpenAI Codex",
  gemini: "Google Gemini",
};

// Hardcoded fallback provider hosts for when GH_AW_ENGINE_API_HOSTS is not set.
// The host patterns are matched against the "host" field in audit.jsonl entries.
const FIREWALL_AUTH_PROVIDER_HOSTS = /** @type {Array<{provider: string, pattern: RegExp, credential: string}>} */ [
  { provider: "GitHub Copilot", pattern: /\.githubcopilot\.com/i, credential: "`COPILOT_GITHUB_TOKEN`" },
  { provider: "OpenAI Codex", pattern: /^api\.openai\.com/i, credential: "`CODEX_API_KEY` / `OPENAI_API_KEY`" },
  { provider: "Anthropic Claude", pattern: /^api\.anthropic\.com/i, credential: "`ANTHROPIC_API_KEY`" },
  { provider: "Google Gemini", pattern: /^generativelanguage\.googleapis\.com/i, credential: "`GEMINI_API_KEY`" },
];

/**
 * Build the list of registered provider entries from the GH_AW_ENGINE_API_HOSTS and
 * GH_AW_ENGINE_ID environment variables. Falls back to the hardcoded
 * FIREWALL_AUTH_PROVIDER_HOSTS list when the env var is not set.
 *
 * @returns {Array<{provider: string, credential: string, hosts?: string[], pattern?: RegExp}>} Provider entries
 */
function buildRegisteredProviderEntries() {
  const engineApiHosts = process.env.GH_AW_ENGINE_API_HOSTS;
  const engineId = (process.env.GH_AW_ENGINE_ID || "").toLowerCase();

  if (engineApiHosts) {
    const hosts = engineApiHosts
      .split(",")
      .map(h => h.trim().toLowerCase())
      .filter(Boolean);
    if (hosts.length > 0) {
      const provider = ENGINE_ID_TO_LABEL[engineId] || engineId || "Engine";
      const credential = ENGINE_ID_TO_CREDENTIAL[engineId] || "`API_KEY`";
      return [{ provider, credential, hosts }];
    }
  }

  // Fallback: use hardcoded patterns, converting to the same shape.
  return FIREWALL_AUTH_PROVIDER_HOSTS.map(({ provider, pattern, credential }) => ({
    provider,
    credential,
    hosts: /** @type {string[]} */ [],
    pattern,
  }));
}

/**
 * Parse the firewall audit.jsonl for authentication rejection entries.
 *
 * Performance strategy for large JSONL files (three-pass approach):
 *   1. Quick file-level regex pre-scan: bail early if no 401/403 status codes appear at all.
 *   2. Per-line regex pre-filter: skip lines without a 401/403 status pattern before JSON.parse.
 *   3. Full JSON parse: only for lines that pass both pre-filters.
 *
 * Providers are resolved dynamically from GH_AW_ENGINE_API_HOSTS / GH_AW_ENGINE_ID env vars
 * (set by the workflow compiler for the current engine). Falls back to a hardcoded list of
 * known public provider API hosts when the env var is not available.
 *
 * @param {string} auditJsonlPath - Path to audit.jsonl
 * @returns {Array<{provider: string, credential: string}>} Unique providers with auth rejections
 */
function parseFirewallAuthErrors(auditJsonlPath) {
  try {
    if (!fs.existsSync(auditJsonlPath)) {
      return [];
    }

    const content = fs.readFileSync(auditJsonlPath, "utf8");
    if (!content.trim()) {
      return [];
    }

    // Pass 1 — file-level pre-scan: bail early when no 401 or 403 appears anywhere.
    // The audit.jsonl format uses `"status":401` or `"status": 401` (compact or spaced).
    // `40[13]` matches digits 1 and 3 only, covering status codes 401 and 403.
    if (!/"status"\s*:\s*40[13]/.test(content)) {
      return [];
    }

    // Build the provider list from env vars (or fallback to hardcoded patterns).
    const providerEntries = buildRegisteredProviderEntries();
    const seenProviders = new Set();
    const results = [];

    for (const line of content.split("\n")) {
      const trimmed = line.trim();
      if (!trimmed || trimmed[0] !== "{") continue;

      // Pass 2 — per-line regex pre-filter: skip lines that cannot have a 401/403 status.
      // This avoids the JSON.parse overhead on the majority of non-auth-failure lines.
      if (!/"status"\s*:\s*40[13]/.test(trimmed)) continue;

      let entry;
      try {
        entry = JSON.parse(trimmed);
      } catch {
        continue;
      }

      const status = entry.status;
      if (status !== 401 && status !== 403) continue;

      const host = typeof entry.host === "string" ? entry.host : "";
      if (!host) continue;
      // Strip port from host for matching (e.g. "api.openai.com:443" → "api.openai.com")
      const hostWithoutPort = host.replace(/:\d+$/, "").toLowerCase();

      for (const providerEntry of providerEntries) {
        const { provider, credential } = providerEntry;
        if (seenProviders.has(provider)) continue;

        // Dynamic matching: check if the host is in the registered hosts list.
        let matched = false;
        if (providerEntry.hosts && providerEntry.hosts.length > 0) {
          matched = providerEntry.hosts.some(h => hostWithoutPort === h || hostWithoutPort.endsWith("." + h));
        } else if (providerEntry.pattern) {
          // Fallback: use the pattern from the hardcoded list.
          matched = providerEntry.pattern.test(hostWithoutPort);
        }

        if (matched) {
          seenProviders.add(provider);
          results.push({ provider, credential });
          break;
        }
      }

      // Early exit: all providers have been matched, no need to scan remaining lines.
      if (seenProviders.size === providerEntries.length) break;
    }

    return results;
  } catch {
    return [];
  }
}

/**
 * Build a context string when the firewall audit log shows authentication rejections
 * from AI provider endpoints (expired or missing API credentials).
 *
 * Reads audit.jsonl from the known path relative to GH_AW_AGENT_OUTPUT and surfaces
 * a remediation hint for each affected provider in the failure issue/comment.
 *
 * @param {string} [auditJsonlPathOverride] - Path override for testing
 * @returns {string} Formatted context string, or empty string if no auth errors
 */
function buildCredentialAuthErrorContext(auditJsonlPathOverride) {
  const agentOutputFile = process.env.GH_AW_AGENT_OUTPUT;
  const defaultAuditPath = agentOutputFile ? path.join(path.dirname(agentOutputFile), "sandbox", "firewall", "audit", "audit.jsonl") : "/tmp/gh-aw/sandbox/firewall/audit/audit.jsonl";
  const auditJsonlPath = auditJsonlPathOverride || defaultAuditPath;

  const authErrors = parseFirewallAuthErrors(auditJsonlPath);

  if (authErrors.length === 0) {
    return "";
  }

  core.info(`Firewall audit log: detected ${authErrors.length} credential auth rejection(s) for: ${authErrors.map(e => e.provider).join(", ")}`);

  const providersList = authErrors.map(e => `- ${e.provider} (${e.credential})`).join("\n");

  const templatePath = getPromptPath("credential_auth_error.md");
  return "\n" + renderTemplateFromFile(templatePath, { providers: providersList });
}
/**
 * Build a context string when assigning the Copilot coding agent to created issues failed.
 * @param {boolean} hasAssignCopilotFailures - Whether any copilot assignments failed
 * @param {string} assignCopilotErrors - Newline-separated list of "issue:number:copilot:error" entries
 * @returns {string} Formatted context string, or empty string if no failures
 */
function buildAssignCopilotFailureContext(hasAssignCopilotFailures, assignCopilotErrors) {
  if (!hasAssignCopilotFailures) {
    return "";
  }

  // Build a list of failed issue assignments
  let issueList = "";
  if (assignCopilotErrors) {
    const errorLines = assignCopilotErrors.split("\n").filter(line => line.trim());
    for (const errorLine of errorLines) {
      const parts = errorLine.split(":");
      if (parts.length >= 4) {
        const number = parts[1];
        const error = parts.slice(3).join(":"); // Rest is the error message
        issueList += `- Issue #${number}: ${error}\n`;
      }
    }
  }

  const templatePath = getPromptPath("assign_copilot_to_created_issues_failure.md");
  return "\n" + renderTemplateFromFile(templatePath, { issues: issueList });
}

/**
 * Check whether agent-stdio.log contains a terminal_reason: "completed" result entry,
 * indicating the agent finished its task successfully despite a non-zero job exit code.
 * Log lines may be prefixed with a timestamp (e.g. "2026-04-27T21:45:00.080Z  {JSON}").
 *
 * Lazy strategy: tests a single regex against the entire file content in one pass and
 * returns immediately on the first match, avoiding line splitting and JSON parsing.
 * The pattern is specific enough (`"terminal_reason"` key with `"completed"` value,
 * 0 or 1 literal space around the colon) that false positives from unrelated content
 * are negligible in practice.
 *
 * @returns {boolean} true if terminal_reason: "completed" was found in the log
 */
function hasAgentTerminalReasonCompleted() {
  const agentOutputFile = process.env.GH_AW_AGENT_OUTPUT;
  const stdioLogPath = agentOutputFile ? path.join(path.dirname(agentOutputFile), "agent-stdio.log") : "/tmp/gh-aw/agent-stdio.log";
  try {
    if (!fs.existsSync(stdioLogPath)) {
      return false;
    }
    const logContent = fs.readFileSync(stdioLogPath, "utf8");
    // Single-pass scan: "terminal_reason" key with 0 or 1 literal space around
    // the colon and "completed" value. JSON uses either compact ("key":"val") or
    // single-spaced ("key" : "val") formatting. `[ ]?` matches only a space
    // character (not tabs or newlines). Returns on first match without splitting
    // lines or parsing JSON.
    return /"terminal_reason"[ ]?:[ ]?"completed"/.test(logContent);
  } catch {
    // IO error — assume not completed
  }
  return false;
}

/**
 * Extract terminal error messages from agent-stdio.log to surface engine failures.
 * First tries to match known error patterns (ERROR:, Error:, Fatal:, panic:, Reconnecting...).
 * Falls back to the last non-empty lines of the log when no patterns match, so that
 * even timeout or unexpected-termination failures include the final agent output.
 * The log file is available in the conclusion job after the agent artifact is downloaded.
 * @returns {string} Formatted context string, or empty string if no engine failure found
 */
function buildEngineFailureContext() {
  // Derive agent-stdio.log path from the agent output file path (same directory)
  const agentOutputFile = process.env.GH_AW_AGENT_OUTPUT;
  const stdioLogPath = agentOutputFile ? path.join(path.dirname(agentOutputFile), "agent-stdio.log") : "/tmp/gh-aw/agent-stdio.log";

  // Include engine ID in failure messages when available (e.g. "copilot", "claude", "codex")
  const engineId = process.env.GH_AW_ENGINE_ID || "";
  const engineLabel = engineId ? ` \`${engineId}\`` : " AI";

  try {
    if (!fs.existsSync(stdioLogPath)) {
      core.info(`agent-stdio.log not found at ${stdioLogPath}, skipping engine failure context`);
      return "";
    }

    const logContent = fs.readFileSync(stdioLogPath, "utf8");
    if (!logContent.trim()) {
      return "";
    }

    const lines = logContent.split("\n");

    // Guard: if the agent completed successfully (terminal_reason: "completed"), the job
    // failure was caused by something other than the agent itself (e.g., post-processing
    // or infrastructure). Suppress the engine failure context to avoid false positive labels.
    // Use the already-loaded logContent directly to avoid a redundant file read.
    if (/"terminal_reason"[ ]?:[ ]?"completed"/.test(logContent)) {
      core.info("Agent completed successfully (terminal_reason: completed) — suppressing engine failure context");
      return "";
    }

    const errorMessages = new Set();

    for (const line of lines) {
      // Codex / generic CLI: "ERROR: <message>" at the start of a line
      const errorPrefixMatch = line.match(/^ERROR:\s*(.+)$/);
      if (errorPrefixMatch) {
        errorMessages.add(errorPrefixMatch[1].trim());
        continue;
      }

      // Node.js / generic: "Error: <message>" at the start of a line
      const errorCapMatch = line.match(/^Error:\s*(.+)$/);
      if (errorCapMatch) {
        errorMessages.add(errorCapMatch[1].trim());
        continue;
      }

      // Fatal errors: "Fatal: <message>" or "FATAL: <message>"
      const fatalMatch = line.match(/^(?:FATAL|Fatal):\s*(.+)$/);
      if (fatalMatch) {
        errorMessages.add(fatalMatch[1].trim());
        continue;
      }

      // Go runtime panic: "panic: <message>"
      const panicMatch = line.match(/^panic:\s*(.+)$/);
      if (panicMatch) {
        errorMessages.add(panicMatch[1].trim());
        continue;
      }

      // Reconnect-style lines that embed the error reason: "Reconnecting... N/M (reason)"
      const reconnectMatch = line.match(/^Reconnecting\.\.\.\s+\d+\/\d+\s*\((.+)\)$/);
      if (reconnectMatch) {
        errorMessages.add(reconnectMatch[1].trim());
      }
    }

    if (errorMessages.size > 0) {
      core.info(`Found ${errorMessages.size} engine error message(s) in agent-stdio.log`);

      // Check for cyber_policy_violation specifically and return a dedicated message
      const hasCyberPolicyViolation = Array.from(errorMessages).some(msg => msg.includes("cyber_policy_violation"));
      if (hasCyberPolicyViolation) {
        core.info("Detected cyber_policy_violation error — using dedicated context message");
        const templatePath = getPromptPath("cyber_policy_violation.md");
        try {
          return "\n" + renderTemplateFromFile(templatePath, {});
        } catch {
          // Template not available — fall through to generic engine failure message
          core.info(`cyber_policy_violation template not found at ${templatePath}, using generic message`);
        }
      }

      let context = `\n**⚠️ Engine Failure**: The${engineLabel} engine terminated before producing output.\n\n**Error details:**\n`;
      for (const message of errorMessages) {
        context += `- ${message}\n`;
      }
      context += "\n";
      return context;
    }

    // AWF infrastructure lines written by the firewall/container wrapper — not produced by
    // the engine itself. They must be filtered out of the fallback tail so the failure
    // context surfaces actual agent output rather than container lifecycle noise
    // (e.g. "Container awf-squid  Removed", "[WARN] Command completed with exit code: 1",
    // "Process exiting with code: 1"). Shared constant from log_parser_shared.cjs keeps the
    // pattern in sync with parse_copilot_log.cjs.
    const INFRA_LINE_RE = AWF_INFRA_LINE_RE;

    // Fallback: no known error patterns found — include the last non-empty lines so that
    // failures caused by timeouts or unexpected terminations still surface useful context.
    const TAIL_LINES = 10;
    const nonEmptyLines = lines.filter(l => l.trim());
    if (nonEmptyLines.length === 0) {
      return "";
    }

    // Exclude AWF infrastructure lines so the fallback displays only actual engine output.
    const agentLines = nonEmptyLines.filter(l => !INFRA_LINE_RE.test(l));

    if (agentLines.length === 0) {
      // The log contains only AWF infrastructure lines — the engine exited before producing
      // any substantive output. This pattern is characteristic of a transient startup failure
      // (e.g., API service unavailable, rate-limiting, token not yet provisioned).
      core.info("agent-stdio.log contains only infrastructure lines — engine likely failed at startup (possible transient failure)");
      const recurringFailureGuidance =
        process.env.GH_AW_ENGINE_ID === "copilot"
          ? "If this failure recurs, check the GitHub Copilot status page and review the firewall audit logs.\n\n"
          : "If this failure recurs, check the provider status page (if available) and review the firewall audit logs.\n\n";
      let context = `\n**⚠️ Engine Failure**: The${engineLabel} engine terminated before producing output.\n\n`;
      context += "The engine exited immediately without producing any output. This often indicates a transient infrastructure issue (e.g., service unavailable, API rate limiting). " + recurringFailureGuidance;
      return context;
    }

    const tailLines = agentLines.slice(-TAIL_LINES);
    core.info(`No specific error patterns found; including last ${tailLines.length} line(s) of agent-stdio.log as fallback`);

    let context = `\n**⚠️ Engine Failure**: The${engineLabel} engine terminated unexpectedly.\n\n**Last agent output:**\n\`\`\`\n`;
    context += tailLines.join("\n");
    context += "\n```\n\n";
    return context;
  } catch (error) {
    core.info(`Failed to read agent-stdio.log for engine failure context: ${getErrorMessage(error)}`);
    return "";
  }
}

/**
 * Handle agent job failure by creating or updating a failure tracking issue
 * This script is called from the conclusion job when the agent job has failed
 * or when the agent succeeded but produced no safe outputs
 */
async function main() {
  try {
    // Get workflow context
    const workflowName = process.env.GH_AW_WORKFLOW_NAME || "unknown";
    const workflowID = process.env.GH_AW_WORKFLOW_ID || "unknown";
    const agentConclusion = process.env.GH_AW_AGENT_CONCLUSION || "";
    const runUrl = process.env.GH_AW_RUN_URL || "";
    const workflowSource = process.env.GH_AW_WORKFLOW_SOURCE || "";
    const workflowSourceURL = process.env.GH_AW_WORKFLOW_SOURCE_URL || "";
    const secretVerificationResult = process.env.GH_AW_SECRET_VERIFICATION_RESULT || "";
    const assignmentErrors = process.env.GH_AW_ASSIGNMENT_ERRORS || "";
    const assignmentErrorCount = process.env.GH_AW_ASSIGNMENT_ERROR_COUNT || "0";
    const assignCopilotErrors = process.env.GH_AW_ASSIGN_COPILOT_ERRORS || "";
    const assignCopilotFailureCount = process.env.GH_AW_ASSIGN_COPILOT_FAILURE_COUNT || "0";
    const createDiscussionErrors = process.env.GH_AW_CREATE_DISCUSSION_ERRORS || "";
    const createDiscussionErrorCount = process.env.GH_AW_CREATE_DISCUSSION_ERROR_COUNT || "0";
    const codePushFailureErrors = process.env.GH_AW_CODE_PUSH_FAILURE_ERRORS || "";
    const codePushFailureCount = process.env.GH_AW_CODE_PUSH_FAILURE_COUNT || "0";
    const checkoutPRSuccess = process.env.GH_AW_CHECKOUT_PR_SUCCESS || "";
    const timeoutMinutes = process.env.GH_AW_TIMEOUT_MINUTES || "";
    const inferenceAccessError = process.env.GH_AW_INFERENCE_ACCESS_ERROR === "true";
    const mcpPolicyError = process.env.GH_AW_MCP_POLICY_ERROR === "true";
    const agenticEngineTimeout = process.env.GH_AW_AGENTIC_ENGINE_TIMEOUT === "true";
    const modelNotSupportedError = process.env.GH_AW_MODEL_NOT_SUPPORTED_ERROR === "true";
    const pushRepoMemoryResult = process.env.GH_AW_PUSH_REPO_MEMORY_RESULT || "";
    const reportFailureAsIssue = process.env.GH_AW_FAILURE_REPORT_AS_ISSUE !== "false"; // Default to true
    // GitHub App token minting failures from the safe_outputs job, conclusion job, and activation job.
    // Any of these being "true" indicates a GitHub App authentication configuration error.
    const safeOutputsAppTokenMintingFailed = process.env.GH_AW_SAFE_OUTPUTS_APP_TOKEN_MINTING_FAILED === "true";
    const conclusionAppTokenMintingFailed = process.env.GH_AW_CONCLUSION_APP_TOKEN_MINTING_FAILED === "true";
    const activationAppTokenMintingFailed = process.env.GH_AW_ACTIVATION_APP_TOKEN_MINTING_FAILED === "true";
    const hasAppTokenMintingFailed = safeOutputsAppTokenMintingFailed || conclusionAppTokenMintingFailed || activationAppTokenMintingFailed;
    // Lockdown check failure from the activation job — set when validate_lockdown_requirements fails.
    // The agent is skipped in this case, but the conclusion job still runs to report the failure.
    const hasLockdownCheckFailed = process.env.GH_AW_LOCKDOWN_CHECK_FAILED === "true";
    // Stale lock file check failure from the activation job — set when the frontmatter hash
    // stored in the compiled .lock.yml no longer matches the source .md file.
    // The agent is skipped in this case; the conclusion job runs to surface remediation guidance.
    const hasStaleLockFileFailed = process.env.GH_AW_STALE_LOCK_FILE_FAILED === "true";
    // Cache-memory availability flag — set when cache-memory is configured for the workflow.
    // Used to detect cache-miss misconfigurations reported by the agent.
    const cacheMemoryEnabled = process.env.GH_AW_CACHE_MEMORY_ENABLED === "true";

    // Collect repo-memory validation errors from all memory configurations
    const repoMemoryValidationErrors = [];
    const repoMemoryPatchSizeExceededIDs = [];
    for (const key in process.env) {
      if (key.startsWith("GH_AW_REPO_MEMORY_VALIDATION_FAILED_")) {
        const memoryID = key.replace("GH_AW_REPO_MEMORY_VALIDATION_FAILED_", "");
        const failed = process.env[key] === "true";
        if (failed) {
          const errorKey = `GH_AW_REPO_MEMORY_VALIDATION_ERROR_${memoryID}`;
          const errorMessage = process.env[errorKey] || "Unknown validation error";
          repoMemoryValidationErrors.push({ memoryID, errorMessage });
        }
      }
      if (key.startsWith("GH_AW_REPO_MEMORY_PATCH_SIZE_EXCEEDED_")) {
        const memoryID = key.replace("GH_AW_REPO_MEMORY_PATCH_SIZE_EXCEEDED_", "");
        if (process.env[key] === "true") {
          repoMemoryPatchSizeExceededIDs.push(memoryID);
        }
      }
    }

    core.info(`Agent conclusion: ${agentConclusion}`);
    core.info(`Workflow name: ${workflowName}`);
    core.info(`Workflow ID: ${workflowID}`);
    core.info(`Secret verification result: ${secretVerificationResult}`);
    core.info(`Assignment error count: ${assignmentErrorCount}`);
    core.info(`Assign copilot failure count: ${assignCopilotFailureCount}`);
    core.info(`Create discussion error count: ${createDiscussionErrorCount}`);
    core.info(`Code push failure count: ${codePushFailureCount}`);
    core.info(`Checkout PR success: ${checkoutPRSuccess}`);
    core.info(`Inference access error: ${inferenceAccessError}`);
    core.info(`MCP policy error: ${mcpPolicyError}`);
    core.info(`Agentic engine timeout: ${agenticEngineTimeout}`);
    core.info(`Model not supported error: ${modelNotSupportedError}`);
    core.info(`Push repo-memory result: ${pushRepoMemoryResult}`);
    core.info(`App token minting failed (safe_outputs/conclusion/activation): ${safeOutputsAppTokenMintingFailed}/${conclusionAppTokenMintingFailed}/${activationAppTokenMintingFailed}`);
    core.info(`Lockdown check failed: ${hasLockdownCheckFailed}`);
    core.info(`Stale lock file check failed: ${hasStaleLockFileFailed}`);
    core.info(`Cache memory enabled: ${cacheMemoryEnabled}`);

    // Check if the agent timed out.
    // A job-level timeout sets agentConclusion to "timed_out".
    // A step-level timeout (timeout-minutes on the engine execution step) is detected by
    // the detect-copilot-errors step which checks for SIGTERM/SIGKILL/SIGINT signals
    // in the engine output and sets the agentic_engine_timeout output.
    const isTimedOut = agentConclusion === "timed_out" || agenticEngineTimeout;

    // Check if there are assignment errors (regardless of agent job status)
    const hasAssignmentErrors = parseInt(assignmentErrorCount, 10) > 0;

    // Check if there are copilot assignment failures for created issues (regardless of agent job status)
    const hasAssignCopilotFailures = parseInt(assignCopilotFailureCount, 10) > 0;

    // Check if there are create_discussion errors (regardless of agent job status)
    const hasCreateDiscussionErrors = parseInt(createDiscussionErrorCount, 10) > 0;

    // Check if there are code-push failures (regardless of agent job status)
    const hasCodePushFailures = parseInt(codePushFailureCount, 10) > 0;

    // Check if the push_repo_memory job itself failed (e.g. permission or config errors)
    const hasPushRepoMemoryFailure = pushRepoMemoryResult === "failure";

    // Check if agent succeeded but produced no safe outputs
    let hasMissingSafeOutputs = false;
    let hasOnlyNoopOutputs = false;
    let hasReportIncomplete = false;
    // Tracks the case where agentConclusion is "failure" but the agent completed its work
    // successfully (terminal_reason: completed) and produced valid non-noop safe outputs.
    // This is a false-positive failure caused by a transient error after the agent's task
    // was done (e.g., post-processing step or AI model server returning a spurious error).
    let hasCompletedDespiteJobFailure = false;
    const { loadAgentOutput } = require("./load_agent_output.cjs");
    const agentOutputResult = loadAgentOutput();

    if (agentConclusion === "success") {
      if (!agentOutputResult.success || !agentOutputResult.items || agentOutputResult.items.length === 0) {
        hasMissingSafeOutputs = true;
        core.info("Agent succeeded but produced no safe outputs");
      } else {
        // Check if all outputs are noop types
        const nonNoopItems = agentOutputResult.items.filter(item => item.type !== "noop");
        if (nonNoopItems.length === 0) {
          hasOnlyNoopOutputs = true;
          core.info("Agent succeeded with only noop outputs - this is not a failure");
        }
      }
    } else if (agentConclusion === "failure") {
      // The agent may have called noop successfully but the AI model server subsequently
      // returned a transient error (e.g. "Response was interrupted due to a server error"),
      // causing exit code 1. In that case we should not report a failure issue since the
      // agent completed its intended work.
      if (agentOutputResult.success && agentOutputResult.items && agentOutputResult.items.length > 0) {
        const nonNoopItems = agentOutputResult.items.filter(item => item.type !== "noop");
        if (nonNoopItems.length === 0) {
          hasOnlyNoopOutputs = true;
          core.info("Agent failed with exit code 1 but produced only noop outputs - treating as successful no-action (transient AI model error)");
        } else if (!nonNoopItems.some(item => item.type === "report_incomplete")) {
          // The agent produced valid non-noop safe outputs (e.g. create_discussion) but the
          // job exit code is non-zero. If terminal_reason: completed is present in the log,
          // the failure was a transient error after the agent finished its task — do not report
          // a failure issue.
          if (hasAgentTerminalReasonCompleted()) {
            hasCompletedDespiteJobFailure = true;
            core.info("Agent failed with exit code 1 but completed successfully (terminal_reason: completed) with valid safe outputs — treating as completed (transient AI model error)");
          }
        }
      }
    }

    // Check if the agent emitted report_incomplete — a first-class signal that the task
    // could not be performed (e.g., MCP crash, missing auth, inaccessible repo).
    // This activates failure handling even when the agent exited 0 and emitted other
    // safe outputs such as add_comment, preventing a tool-failure narrative from being
    // classified as a successful review or other completed task.
    if (agentOutputResult.success && agentOutputResult.items && agentOutputResult.items.length > 0) {
      const reportIncompleteItems = agentOutputResult.items.filter(item => item.type === "report_incomplete");
      if (reportIncompleteItems.length > 0) {
        hasReportIncomplete = true;
        core.info(`Agent emitted ${reportIncompleteItems.length} report_incomplete signal(s) - activating failure handling`);
        for (const item of reportIncompleteItems) {
          core.info(`  report_incomplete reason: ${item.reason}`);
        }
      }
    }

    // Detect cache-miss misconfiguration: the agent reported a missing_data with reason
    // "cache_memory_miss" while cache-memory was configured and available.  This indicates the
    // prompt is referencing an incorrect path inside the cache directory.
    // Check for items regardless of agentOutputResult.success so that cache-miss signals
    // emitted alongside other output are not missed when the agent job also fails.
    let hasCacheMissMisconfiguration = false;
    if (cacheMemoryEnabled && agentOutputResult.items) {
      const cacheMissItems = agentOutputResult.items.filter(item => item.type === "missing_data" && item.reason === "cache_memory_miss");
      if (cacheMissItems.length > 0) {
        hasCacheMissMisconfiguration = true;
        core.info(`Cache-miss misconfiguration detected: ${cacheMissItems.length} missing_data item(s) with reason "cache_memory_miss" despite cache-memory being available`);
      }
    }

    // Only proceed if the agent job actually failed OR timed out OR there are assignment errors OR
    // create_discussion errors OR code-push failures OR push_repo_memory failed OR missing safe outputs
    // OR a GitHub App token minting step failed OR the lockdown check failed OR copilot assignment failed
    // OR the stale lock file check failed OR the agent reported task incompletion via report_incomplete
    // OR a cache-miss was detected despite cache-memory being available (configuration problem).
    // BUT skip if we only have noop outputs (that's a successful no-action scenario)
    if (
      agentConclusion !== "failure" &&
      !isTimedOut &&
      !hasAssignmentErrors &&
      !hasAssignCopilotFailures &&
      !hasCreateDiscussionErrors &&
      !hasCodePushFailures &&
      !hasPushRepoMemoryFailure &&
      !hasMissingSafeOutputs &&
      !hasAppTokenMintingFailed &&
      !hasLockdownCheckFailed &&
      !hasStaleLockFileFailed &&
      !hasReportIncomplete &&
      !hasCacheMissMisconfiguration
    ) {
      core.info(
        `Agent job did not fail and no assignment/discussion/code-push/push-repo-memory/app-token/lockdown/stale-lock-file/report-incomplete/cache-miss errors and has safe outputs (conclusion: ${agentConclusion}), skipping failure handling`
      );
      return;
    }

    // If we only have noop outputs (and no report_incomplete or cache-miss), skip failure handling
    if (hasOnlyNoopOutputs && !hasReportIncomplete && !hasCacheMissMisconfiguration) {
      core.info("Agent completed with only noop outputs - skipping failure handling");
      return;
    }

    // If the agent completed its work successfully (terminal_reason: completed) and produced
    // valid non-noop safe outputs despite the job's non-zero exit code, skip failure handling.
    // This prevents false-positive failure issues when a transient AI model error occurs
    // after the agent has already finished its task (e.g., create_discussion produced but
    // the server returned a spurious error on teardown).
    if (hasCompletedDespiteJobFailure && !hasReportIncomplete && !hasCacheMissMisconfiguration) {
      core.info("Agent completed with valid safe outputs despite job failure (terminal_reason: completed) — skipping failure handling");
      return;
    }

    // Check if failure issue reporting is disabled
    if (!reportFailureAsIssue) {
      core.info("Failure issue reporting is disabled (report-failure-as-issue: false), skipping failure issue creation");
      return;
    }

    // Check if the failure was due to PR checkout (e.g., PR was merged and branch deleted)
    // If checkout_pr_success is "false", skip creating an issue as this is expected behavior
    if (agentConclusion === "failure" && checkoutPRSuccess === "false") {
      core.info("Skipping failure handling - failure was due to PR checkout (likely PR merged)");
      return;
    }

    // Determine the failure issue repository destination.
    // SEC-005: GH_AW_FAILURE_ISSUE_REPO is set in the workflow frontmatter at compile time
    // and is therefore a trusted compile-time configuration value. No validateTargetRepo
    // allowlist check is required; the frontmatter trust boundary provides the equivalent
    // security guarantee.
    // If GH_AW_FAILURE_ISSUE_REPO is set, use that repo instead of the current repo
    const failureIssueRepo = process.env.GH_AW_FAILURE_ISSUE_REPO || "";
    let owner, repo;
    if (failureIssueRepo && failureIssueRepo.includes("/")) {
      const parts = failureIssueRepo.split("/");
      owner = parts[0];
      repo = parts[1];
      core.info(`Using configured failure issue repo: ${owner}/${repo}`);
    } else {
      ({ owner, repo } = context.repo);
    }

    // Try to find a pull request for the current branch
    const pullRequest = await findPullRequestForCurrentBranch();

    // Generate history URL for linking to all failure issues created by this workflow
    const historyUrl = generateHistoryUrl({
      owner,
      repo,
      itemType: "issue",
      workflowId: workflowID,
    });
    const actionFailureIssueExpiresHours = getActionFailureIssueExpiresHours();

    // Check if parent issue creation is enabled (defaults to false)
    const groupReports = process.env.GH_AW_GROUP_REPORTS === "true";

    // Ensure parent issue exists first (only if enabled)
    let parentIssue;
    if (groupReports) {
      try {
        parentIssue = await ensureParentIssue(null, owner, repo, actionFailureIssueExpiresHours);
      } catch (error) {
        core.warning(`Could not create parent issue, proceeding without parent: ${getErrorMessage(error)}`);
        // Continue without parent issue
      }
    } else {
      core.info("Parent issue creation is disabled (group-reports: false)");
    }

    // Sanitize workflow name for title
    const sanitizedWorkflowName = sanitizeContent(workflowName, { maxLength: 100 });
    const issueTitle = `[aw] ${sanitizedWorkflowName} failed`;

    core.info(`Checking for existing issue with title: "${issueTitle}"`);

    // Search for existing open issue with this title and label
    const searchQuery = `repo:${owner}/${repo} is:issue is:open label:agentic-workflows in:title "${issueTitle}"`;

    try {
      const searchResult = await github.rest.search.issuesAndPullRequests({
        q: searchQuery,
        per_page: 1,
      });

      if (searchResult.data.total_count > 0) {
        // Issue exists, add a comment
        const existingIssue = searchResult.data.items[0];
        core.info(`Found existing issue #${existingIssue.number}: ${existingIssue.html_url}`);

        // Read comment template
        const commentTemplatePath = getPromptPath("agent_failure_comment.md");
        const commentTemplate = fs.readFileSync(commentTemplatePath, "utf8");

        // Extract run ID from URL (e.g., https://github.com/owner/repo/actions/runs/123 -> "123")
        let runId = "";
        const runIdMatch = runUrl.match(/\/actions\/runs\/(\d+)/);
        if (runIdMatch) {
          runId = runIdMatch[1];
        }

        // Build assignment errors context
        let assignmentErrorsContext = "";
        if (hasAssignmentErrors && assignmentErrors) {
          assignmentErrorsContext = "\n**⚠️ Agent Assignment Failed**: Failed to assign agent to issues due to insufficient permissions or missing token.\n\n**Assignment Errors:**\n";
          const errorLines = assignmentErrors.split("\n").filter(line => line.trim());
          for (const errorLine of errorLines) {
            const parts = errorLine.split(":");
            if (parts.length >= 4) {
              const type = parts[0]; // "issue" or "pr"
              const number = parts[1];
              const agent = parts[2];
              const error = parts.slice(3).join(":"); // Rest is the error message
              assignmentErrorsContext += `- ${type === "issue" ? "Issue" : "PR"} #${number} (agent: ${agent}): ${error}\n`;
            }
          }
          assignmentErrorsContext += "\n";
        }

        // Build create_discussion errors context
        const createDiscussionErrorsContext = hasCreateDiscussionErrors ? buildCreateDiscussionErrorsContext(createDiscussionErrors) : "";

        // Build code-push failure context
        const codePushFailureContext = hasCodePushFailures ? buildCodePushFailureContext(codePushFailureErrors, pullRequest, runUrl) : "";

        // Build repo-memory validation errors context
        let repoMemoryValidationContext = "";
        if (repoMemoryValidationErrors.length > 0) {
          repoMemoryValidationContext = "\n**⚠️ Repo-Memory Validation Failed**: Invalid file types detected in repo-memory.";
          if (pullRequest) {
            repoMemoryValidationContext += `\n\n**Pull Request:** [#${pullRequest.number}](${pullRequest.html_url})`;
          }
          repoMemoryValidationContext += "\n\n**Validation Errors:**\n";
          for (const { memoryID, errorMessage } of repoMemoryValidationErrors) {
            repoMemoryValidationContext += `- Memory "${memoryID}": ${errorMessage}\n`;
          }
          repoMemoryValidationContext += "\n";
        }

        // Build push_repo_memory job failure context
        const pushRepoMemoryFailureContext = buildPushRepoMemoryFailureContext(hasPushRepoMemoryFailure, repoMemoryPatchSizeExceededIDs, runUrl);

        // Build missing_data context
        const missingDataContext = buildMissingDataContext(cacheMemoryEnabled);

        // Build report_incomplete context
        const reportIncompleteContext = buildReportIncompleteContext();

        // Build missing safe outputs context
        let missingSafeOutputsContext = "";
        if (hasMissingSafeOutputs) {
          missingSafeOutputsContext = "\n**⚠️ No Safe Outputs Generated**: The agent job succeeded but did not produce any safe outputs.";
          if (pullRequest) {
            missingSafeOutputsContext += `\n\n**Pull Request:** [#${pullRequest.number}](${pullRequest.html_url})`;
          }
          missingSafeOutputsContext += "\n\nThis typically indicates:\n";
          missingSafeOutputsContext += "- The safe output server failed to run\n";
          missingSafeOutputsContext += "- The prompt failed to generate any meaningful result\n";
          missingSafeOutputsContext += "- The agent should have called `noop` to explicitly indicate no action was taken\n\n";
        }

        // Build fork context hint
        const forkContext = buildForkContextHint();

        // Build engine failure context (surfaces terminal errors from agent-stdio.log)
        const engineFailureContext = agentConclusion === "failure" ? buildEngineFailureContext() : "";

        // Build timeout context
        const timeoutContext = buildTimeoutContext(isTimedOut, timeoutMinutes);

        // Build inference access error context
        const inferenceAccessErrorContext = buildInferenceAccessErrorContext(inferenceAccessError);

        // Build MCP policy error context
        const mcpPolicyErrorContext = buildMCPPolicyErrorContext(mcpPolicyError);

        // Build model not supported error context
        const modelNotSupportedErrorContext = buildModelNotSupportedErrorContext(modelNotSupportedError);

        // Build GitHub App token minting failure context
        const appTokenMintingFailedContext = buildAppTokenMintingFailedContext(hasAppTokenMintingFailed);

        // Build lockdown check failure context
        const lockdownCheckFailedContext = buildLockdownCheckFailedContext(hasLockdownCheckFailed);

        // Build stale lock file failure context
        const staleLockFileFailedContext = buildStaleLockFileFailedContext(hasStaleLockFileFailed);

        // Build copilot assignment failure context for created issues
        const assignCopilotFailureContext = buildAssignCopilotFailureContext(hasAssignCopilotFailures, assignCopilotErrors);

        // Build credential auth error context (firewall audit.jsonl 401/403 from provider endpoints)
        const credentialAuthErrorContext = buildCredentialAuthErrorContext();

        // Create template context
        const templateContext = {
          run_url: runUrl,
          run_id: runId,
          workflow_name: workflowName,
          workflow_source: workflowSource,
          workflow_source_url: workflowSourceURL,
          secret_verification_failed: String(secretVerificationResult === "failed"),
          secret_verification_context:
            secretVerificationResult === "failed"
              ? "\n**⚠️ Secret Verification Failed**: The workflow's secret validation step failed. Please check that the required secrets are configured in your repository settings.\n\nFor more information on configuring tokens, see: https://github.github.com/gh-aw/reference/engines/\n"
              : "",
          credential_auth_error_context: credentialAuthErrorContext,
          assignment_errors_context: assignmentErrorsContext,
          assign_copilot_failure_context: assignCopilotFailureContext,
          create_discussion_errors_context: createDiscussionErrorsContext,
          code_push_failure_context: codePushFailureContext,
          repo_memory_validation_context: repoMemoryValidationContext,
          push_repo_memory_failure_context: pushRepoMemoryFailureContext,
          missing_data_context: missingDataContext,
          report_incomplete_context: reportIncompleteContext,
          missing_safe_outputs_context: missingSafeOutputsContext,
          engine_failure_context: engineFailureContext,
          timeout_context: timeoutContext,
          fork_context: forkContext,
          inference_access_error_context: inferenceAccessErrorContext,
          mcp_policy_error_context: mcpPolicyErrorContext,
          model_not_supported_error_context: modelNotSupportedErrorContext,
          app_token_minting_failed_context: appTokenMintingFailedContext,
          lockdown_check_failed_context: lockdownCheckFailedContext,
          stale_lock_file_failed_context: staleLockFileFailedContext,
        };

        // Render the comment template
        const commentBody = renderTemplate(commentTemplate, templateContext);

        // Generate footer for the comment using templated message
        const ctx = {
          workflowName,
          runUrl,
          workflowSource,
          workflowSourceUrl: workflowSourceURL,
          historyUrl: historyUrl || undefined,
        };
        const footer = getFooterAgentFailureCommentMessage(ctx);

        // Prepend detection caution alert (when present) so it appears first in the comment body
        const detectionCaution = getDetectionCautionAlert(workflowName, runUrl);
        const fullCommentBodyRaw = detectionCaution ? `${detectionCaution}\n\n${commentBody}\n\n${footer}` : `${commentBody}\n\n${footer}`;

        // Combine comment body with footer
        const fullCommentBody = sanitizeContent(fullCommentBodyRaw, { maxLength: 65000 });

        await github.rest.issues.createComment({
          owner,
          repo,
          issue_number: existingIssue.number,
          body: fullCommentBody,
        });

        core.info(`✓ Added comment to existing issue #${existingIssue.number}`);
      } else {
        // No existing issue, create a new one
        core.info("No existing issue found, creating a new one");

        // Read issue template
        const issueTemplatePath = getPromptPath("agent_failure_issue.md");
        const issueTemplate = fs.readFileSync(issueTemplatePath, "utf8");

        // Get current branch information
        const currentBranch = getCurrentBranch();

        // Build assignment errors context
        let assignmentErrorsContext = "";
        if (hasAssignmentErrors && assignmentErrors) {
          assignmentErrorsContext = "\n**⚠️ Agent Assignment Failed**: Failed to assign agent to issues due to insufficient permissions or missing token.\n\n**Assignment Errors:**\n";
          const errorLines = assignmentErrors.split("\n").filter(line => line.trim());
          for (const errorLine of errorLines) {
            const parts = errorLine.split(":");
            if (parts.length >= 4) {
              const type = parts[0]; // "issue" or "pr"
              const number = parts[1];
              const agent = parts[2];
              const error = parts.slice(3).join(":"); // Rest is the error message
              assignmentErrorsContext += `- ${type === "issue" ? "Issue" : "PR"} #${number} (agent: ${agent}): ${error}\n`;
            }
          }
          assignmentErrorsContext += "\n";
        }

        // Build create_discussion errors context
        const createDiscussionErrorsContext = hasCreateDiscussionErrors ? buildCreateDiscussionErrorsContext(createDiscussionErrors) : "";

        // Build code-push failure context
        const codePushFailureContext = hasCodePushFailures ? buildCodePushFailureContext(codePushFailureErrors, pullRequest, runUrl) : "";

        // Build repo-memory validation errors context
        let repoMemoryValidationContext = "";
        if (repoMemoryValidationErrors.length > 0) {
          repoMemoryValidationContext = "\n**⚠️ Repo-Memory Validation Failed**: Invalid file types detected in repo-memory.";
          if (pullRequest) {
            repoMemoryValidationContext += `\n\n**Pull Request:** [#${pullRequest.number}](${pullRequest.html_url})`;
          }
          repoMemoryValidationContext += "\n\n**Validation Errors:**\n";
          for (const { memoryID, errorMessage } of repoMemoryValidationErrors) {
            repoMemoryValidationContext += `- Memory "${memoryID}": ${errorMessage}\n`;
          }
          repoMemoryValidationContext += "\n";
        }

        // Build push_repo_memory job failure context
        const pushRepoMemoryFailureContext = buildPushRepoMemoryFailureContext(hasPushRepoMemoryFailure, repoMemoryPatchSizeExceededIDs, runUrl);

        // Build missing_data context
        const missingDataContext = buildMissingDataContext(cacheMemoryEnabled);

        // Build report_incomplete context
        const reportIncompleteContext = buildReportIncompleteContext();

        // Build missing safe outputs context
        let missingSafeOutputsContext = "";
        if (hasMissingSafeOutputs) {
          missingSafeOutputsContext = "\n**⚠️ No Safe Outputs Generated**: The agent job succeeded but did not produce any safe outputs.";
          if (pullRequest) {
            missingSafeOutputsContext += `\n\n**Pull Request:** [#${pullRequest.number}](${pullRequest.html_url})`;
          }
          missingSafeOutputsContext += "\n\nThis typically indicates:\n";
          missingSafeOutputsContext += "- The safe output server failed to run\n";
          missingSafeOutputsContext += "- The prompt failed to generate any meaningful result\n";
          missingSafeOutputsContext += "- The agent should have called `noop` to explicitly indicate no action was taken\n\n";
        }

        // Build fork context hint
        const forkContext = buildForkContextHint();

        // Build engine failure context (surfaces terminal errors from agent-stdio.log)
        const engineFailureContext = agentConclusion === "failure" ? buildEngineFailureContext() : "";

        // Build timeout context
        const timeoutContext = buildTimeoutContext(isTimedOut, timeoutMinutes);

        // Build inference access error context
        const inferenceAccessErrorContext = buildInferenceAccessErrorContext(inferenceAccessError);

        // Build MCP policy error context
        const mcpPolicyErrorContext = buildMCPPolicyErrorContext(mcpPolicyError);

        // Build model not supported error context
        const modelNotSupportedErrorContext = buildModelNotSupportedErrorContext(modelNotSupportedError);

        // Build GitHub App token minting failure context
        const appTokenMintingFailedContext = buildAppTokenMintingFailedContext(hasAppTokenMintingFailed);

        // Build lockdown check failure context
        const lockdownCheckFailedContext = buildLockdownCheckFailedContext(hasLockdownCheckFailed);

        // Build stale lock file failure context
        const staleLockFileFailedContext = buildStaleLockFileFailedContext(hasStaleLockFileFailed);

        // Build copilot assignment failure context for created issues
        const assignCopilotFailureContext = buildAssignCopilotFailureContext(hasAssignCopilotFailures, assignCopilotErrors);

        // Build credential auth error context (firewall audit.jsonl 401/403 from provider endpoints)
        const credentialAuthErrorContext = buildCredentialAuthErrorContext();

        // Create template context with sanitized workflow name
        const templateContext = {
          workflow_name: sanitizedWorkflowName,
          workflow_id: workflowID,
          run_url: runUrl,
          workflow_source_url: workflowSourceURL || "#",
          branch: currentBranch,
          pull_request_info: pullRequest ? `  \n**Pull Request:** [#${pullRequest.number}](${pullRequest.html_url})` : "",
          secret_verification_failed: String(secretVerificationResult === "failed"),
          secret_verification_context:
            secretVerificationResult === "failed"
              ? "\n**⚠️ Secret Verification Failed**: The workflow's secret validation step failed. Please check that the required secrets are configured in your repository settings.\n\nFor more information on configuring tokens, see: https://github.github.com/gh-aw/reference/engines/\n"
              : "",
          credential_auth_error_context: credentialAuthErrorContext,
          assignment_errors_context: assignmentErrorsContext,
          assign_copilot_failure_context: assignCopilotFailureContext,
          create_discussion_errors_context: createDiscussionErrorsContext,
          code_push_failure_context: codePushFailureContext,
          repo_memory_validation_context: repoMemoryValidationContext,
          push_repo_memory_failure_context: pushRepoMemoryFailureContext,
          missing_data_context: missingDataContext,
          report_incomplete_context: reportIncompleteContext,
          missing_safe_outputs_context: missingSafeOutputsContext,
          engine_failure_context: engineFailureContext,
          timeout_context: timeoutContext,
          fork_context: forkContext,
          inference_access_error_context: inferenceAccessErrorContext,
          mcp_policy_error_context: mcpPolicyErrorContext,
          model_not_supported_error_context: modelNotSupportedErrorContext,
          app_token_minting_failed_context: appTokenMintingFailedContext,
          lockdown_check_failed_context: lockdownCheckFailedContext,
          stale_lock_file_failed_context: staleLockFileFailedContext,
        };

        // Render the issue template
        const issueBodyContent = renderTemplate(issueTemplate, templateContext);

        // Generate footer for the issue using templated message
        const ctx = {
          workflowName,
          runUrl,
          workflowSource,
          workflowSourceUrl: workflowSourceURL,
          historyUrl: historyUrl || undefined,
        };
        const footer = getFooterAgentFailureIssueMessage(ctx);

        // Add expiration marker inside the quoted footer section using helper
        const footerWithExpires = generateFooterWithExpiration({
          footerText: footer,
          expiresHours: actionFailureIssueExpiresHours,
          suffix: `\n\n${generateXMLMarker(workflowName, runUrl)}`,
        });

        // Prepend detection caution alert (when present) so it appears first in the issue body
        const detectionCaution = getDetectionCautionAlert(workflowName, runUrl);

        // Combine issue body with footer
        const bodyLines = detectionCaution ? [detectionCaution, "", issueBodyContent, "", footerWithExpires] : [issueBodyContent, "", footerWithExpires];
        const issueBody = bodyLines.join("\n");

        const newIssue = await github.rest.issues.create({
          owner,
          repo,
          title: issueTitle,
          body: issueBody,
          labels: ["agentic-workflows"],
        });

        core.info(`✓ Created new issue #${newIssue.data.number}: ${newIssue.data.html_url}`);

        // Link as sub-issue to parent if parent issue was created
        if (parentIssue) {
          try {
            await linkSubIssue(parentIssue.node_id, newIssue.data.node_id, parentIssue.number, newIssue.data.number);
          } catch (error) {
            core.warning(`Could not link issue as sub-issue: ${getErrorMessage(error)}`);
            // Continue even if linking fails
          }
        }
      }
    } catch (error) {
      core.warning(`Failed to create or update failure tracking issue: ${getErrorMessage(error)}`);
      // Don't fail the workflow if we can't create the issue
    }
  } catch (error) {
    core.warning(`Error in handle_agent_failure: ${getErrorMessage(error)}`);
    // Don't fail the workflow
  }
}

module.exports = {
  main,
  buildCodePushFailureContext,
  buildPushRepoMemoryFailureContext,
  buildAppTokenMintingFailedContext,
  buildLockdownCheckFailedContext,
  buildStaleLockFileFailedContext,
  buildTimeoutContext,
  buildAssignCopilotFailureContext,
  buildEngineFailureContext,
  buildReportIncompleteContext,
  buildMCPPolicyErrorContext,
  buildModelNotSupportedErrorContext,
  buildMissingDataContext,
  buildCredentialAuthErrorContext,
  parseFirewallAuthErrors,
  getActionFailureIssueExpiresHours,
  hasAgentTerminalReasonCompleted,
};
