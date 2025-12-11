// @ts-check
/// <reference types="@actions/github-script" />

/**
 * Get issue details using REST API
 * @param {any} github - GitHub REST API instance
 * @param {string} owner - Repository owner
 * @param {string} repo - Repository name
 * @param {number} issueNumber - Issue number
 * @returns {Promise<{number: number, title: string, labels: Array<{name: string}>, html_url: string, locked: boolean}>} Issue details
 */
async function getIssueDetails(github, owner, repo, issueNumber) {
  const { data: issue } = await github.rest.issues.get({
    owner,
    repo,
    issue_number: issueNumber,
  });

  if (!issue) {
    throw new Error(`Issue #${issueNumber} not found in ${owner}/${repo}`);
  }

  return issue;
}

/**
 * Add comment to a GitHub Issue using REST API
 * @param {any} github - GitHub REST API instance
 * @param {string} owner - Repository owner
 * @param {string} repo - Repository name
 * @param {number} issueNumber - Issue number
 * @param {string} message - Comment body
 * @returns {Promise<{id: number, html_url: string}>} Comment details
 */
async function addIssueComment(github, owner, repo, issueNumber, message) {
  const { data: comment } = await github.rest.issues.createComment({
    owner,
    repo,
    issue_number: issueNumber,
    body: message,
  });

  return comment;
}

/**
 * Lock a GitHub Issue using REST API
 * @param {any} github - GitHub REST API instance
 * @param {string} owner - Repository owner
 * @param {string} repo - Repository name
 * @param {number} issueNumber - Issue number
 * @param {string} [lockReason] - Optional lock reason
 * @returns {Promise<void>}
 */
async function lockIssue(github, owner, repo, issueNumber, lockReason) {
  const params = {
    owner,
    repo,
    issue_number: issueNumber,
    lock_reason: lockReason,
  };

  await github.rest.issues.lock(params);
}

/**
 * Check if target matches configuration
 * @param {string} target - Target configuration value
 * @returns {boolean} True if target is wildcard
 */
function isTargetWildcard(target) {
  return target === "*";
}

/**
 * Check if issue matches required filters
 * @param {any} issue - Issue details
 * @param {string} [requiredTitlePrefix] - Required title prefix
 * @param {string[]} [requiredLabels] - Required labels
 * @returns {{matches: boolean, reason?: string}} Filter match result
 */
function matchesFilters(issue, requiredTitlePrefix, requiredLabels) {
  // Check title prefix
  if (requiredTitlePrefix && !issue.title.startsWith(requiredTitlePrefix)) {
    return {
      matches: false,
      reason: `Issue #${issue.number} does not have required title prefix "${requiredTitlePrefix}"`,
    };
  }

  // Check required labels
  if (requiredLabels && requiredLabels.length > 0) {
    const issueLabels = issue.labels.map(label => label.name);
    const missingLabels = requiredLabels.filter(label => !issueLabels.includes(label));
    if (missingLabels.length > 0) {
      return {
        matches: false,
        reason: `Issue #${issue.number} does not have required labels: ${missingLabels.join(", ")}`,
      };
    }
  }

  return { matches: true };
}

async function main() {
  // Check if we're in staged mode
  const isStaged = process.env.GH_AW_SAFE_OUTPUTS_STAGED === "true";

  // Load the validated agent output
  const agentOutputPath = process.env.GH_AW_AGENT_OUTPUT;
  if (!agentOutputPath) {
    core.info("No GH_AW_AGENT_OUTPUT environment variable found");
    return;
  }

  let validatedOutput;
  try {
    const fs = require("fs");
    const outputContent = fs.readFileSync(agentOutputPath, "utf8");
    if (outputContent.trim() === "") {
      core.info("Agent output content is empty");
      return;
    }
    validatedOutput = JSON.parse(outputContent);
  } catch (error) {
    core.setFailed(`Error reading agent output: ${error instanceof Error ? error.message : String(error)}`);
    return;
  }

  if (!validatedOutput.items || !Array.isArray(validatedOutput.items)) {
    core.info("No valid items found in agent output");
    return;
  }

  // Find all lock_issue items
  const items = validatedOutput.items.filter(/** @param {any} item */ item => item.type === "lock_issue");
  if (items.length === 0) {
    core.info("No lock-issue items found in agent output");
    return;
  }

  core.info(`Found ${items.length} lock-issue item(s)`);

  // Get configuration from environment
  const target = process.env.GH_AW_LOCK_ISSUE_TARGET || "triggering";
  const requiredTitlePrefix = process.env.GH_AW_LOCK_ISSUE_REQUIRED_TITLE_PREFIX || "";
  const requiredLabelsStr = process.env.GH_AW_LOCK_ISSUE_REQUIRED_LABELS || "";
  const requiredLabels = requiredLabelsStr ? requiredLabelsStr.split(",").map(l => l.trim()) : [];

  const owner = context.repo.owner;
  const repo = context.repo.repo;

  // If in staged mode, emit step summary instead of performing actions
  if (isStaged) {
    let summaryContent = "## 🎭 Staged Mode: Lock Issues Preview\n\n";
    summaryContent += "The following issues would be locked if staged mode was disabled:\n\n";

    for (let i = 0; i < items.length; i++) {
      const item = items[i];
      summaryContent += `### Issue ${i + 1}\n`;
      summaryContent += `**Comment**: ${item.body}\n`;
      if (item.lock_reason) {
        summaryContent += `**Lock Reason**: ${item.lock_reason}\n`;
      }
      if (item.issue_number) {
        summaryContent += `**Issue Number**: ${item.issue_number}\n`;
      } else {
        summaryContent += `**Target**: Triggering issue\n`;
      }
      summaryContent += "\n";
    }

    core.summary.addRaw(summaryContent).write();
    core.info("📝 Issue lock preview written to step summary");
    return;
  }

  // Process each lock_issue item
  let lockedCount = 0;

  for (const item of items) {
    try {
      let issueNumber;

      // Determine which issue to lock
      if (target === "triggering") {
        // Use triggering issue from event context
        if (context.eventName !== "issues" && context.eventName !== "issue_comment") {
          core.info('Target is "triggering" but not running in issue context, skipping issue lock');
          continue;
        }

        // Get issue number from event context
        if (context.payload.issue) {
          issueNumber = context.payload.issue.number;
        } else {
          core.info("No issue found in event context, skipping");
          continue;
        }
      } else if (isTargetWildcard(target)) {
        // Use issue_number from the item
        if (!item.issue_number) {
          core.warning("Target is '*' but no issue_number provided in lock_issue item, skipping");
          continue;
        }
        issueNumber = typeof item.issue_number === "string" ? parseInt(item.issue_number, 10) : item.issue_number;
      } else {
        core.warning(`Unknown target value: ${target}, skipping`);
        continue;
      }

      // Get issue details
      const issue = await getIssueDetails(github, owner, repo, issueNumber);

      // Check if issue is already locked
      if (issue.locked) {
        core.info(`Issue #${issueNumber} is already locked, skipping`);
        continue;
      }

      // Check filters
      const filterResult = matchesFilters(issue, requiredTitlePrefix, requiredLabels);
      if (!filterResult.matches) {
        core.info(filterResult.reason);
        continue;
      }

      // Add comment
      await addIssueComment(github, owner, repo, issueNumber, item.body);
      core.info(`Added comment to issue #${issueNumber}`);

      // Lock the issue
      await lockIssue(github, owner, repo, issueNumber, item.lock_reason);
      core.info(`Successfully locked issue #${issueNumber}`);

      lockedCount++;

      // Set outputs for the last locked issue
      core.setOutput("issue_number", issueNumber);
      core.setOutput("issue_url", issue.html_url);
    } catch (error) {
      const errorMessage = error instanceof Error ? error.message : String(error);
      core.error(`Failed to lock issue: ${errorMessage}`);
      core.setFailed(`Failed to lock issue: ${errorMessage}`);
      return;
    }
  }

  if (lockedCount > 0) {
    core.info(`Successfully locked ${lockedCount} issue(s)`);
  }
}

await main();
