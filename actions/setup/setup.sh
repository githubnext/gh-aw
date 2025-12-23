#!/usr/bin/env bash
# Setup Activation Action
# Copies activation job files to the agent environment

set -e

# Get destination from input or use default
DESTINATION="${INPUT_DESTINATION:-/tmp/gh-aw/actions/activation}"

echo "::notice::Copying activation files to ${DESTINATION}"

# Create destination directory if it doesn't exist
mkdir -p "${DESTINATION}"
echo "::notice::Created directory: ${DESTINATION}"

# File count
FILE_COUNT=0

# Embedded activation files will be written below
# Each file is written using a here-document

# === FILE: check_stop_time.cjs ===
cat > "${DESTINATION}/check_stop_time.cjs" << 'EOF_CHECK_STOP_TIME'
// @ts-check
/// <reference types="@actions/github-script" />

async function main() {
  const stopTime = process.env.GH_AW_STOP_TIME;
  const workflowName = process.env.GH_AW_WORKFLOW_NAME;

  if (!stopTime) {
    core.setFailed("Configuration error: GH_AW_STOP_TIME not specified.");
    return;
  }

  if (!workflowName) {
    core.setFailed("Configuration error: GH_AW_WORKFLOW_NAME not specified.");
    return;
  }

  core.info(`Checking stop-time limit: ${stopTime}`);

  // Parse the stop time (format: "YYYY-MM-DD HH:MM:SS")
  const stopTimeDate = new Date(stopTime);

  if (isNaN(stopTimeDate.getTime())) {
    core.setFailed(`Invalid stop-time format: ${stopTime}. Expected format: YYYY-MM-DD HH:MM:SS`);
    return;
  }

  const currentTime = new Date();
  core.info(`Current time: ${currentTime.toISOString()}`);
  core.info(`Stop time: ${stopTimeDate.toISOString()}`);

  if (currentTime >= stopTimeDate) {
    core.warning(`⏰ Stop time reached. Workflow execution will be prevented by activation job.`);
    core.setOutput("stop_time_ok", "false");
    return;
  }

  core.setOutput("stop_time_ok", "true");
}
await main();

EOF_CHECK_STOP_TIME
echo "::notice::Copied: check_stop_time.cjs"
FILE_COUNT=$((FILE_COUNT + 1))

# === FILE: check_skip_if_match.cjs ===
cat > "${DESTINATION}/check_skip_if_match.cjs" << 'EOF_CHECK_SKIP_IF_MATCH'
// @ts-check
/// <reference types="@actions/github-script" />

async function main() {
  const skipQuery = process.env.GH_AW_SKIP_QUERY;
  const workflowName = process.env.GH_AW_WORKFLOW_NAME;
  const maxMatchesStr = process.env.GH_AW_SKIP_MAX_MATCHES || "1";

  if (!skipQuery) {
    core.setFailed("Configuration error: GH_AW_SKIP_QUERY not specified.");
    return;
  }

  if (!workflowName) {
    core.setFailed("Configuration error: GH_AW_WORKFLOW_NAME not specified.");
    return;
  }

  const maxMatches = parseInt(maxMatchesStr, 10);
  if (isNaN(maxMatches) || maxMatches < 1) {
    core.setFailed(`Configuration error: GH_AW_SKIP_MAX_MATCHES must be a positive integer, got "${maxMatchesStr}".`);
    return;
  }

  core.info(`Checking skip-if-match query: ${skipQuery}`);
  core.info(`Maximum matches threshold: ${maxMatches}`);

  // Get repository information from context
  const { owner, repo } = context.repo;

  // Scope the query to the current repository
  const scopedQuery = `${skipQuery} repo:${owner}/${repo}`;

  core.info(`Scoped query: ${scopedQuery}`);

  try {
    // Search for issues and pull requests using the GitHub API
    // We only need to know if the count reaches the threshold
    const response = await github.rest.search.issuesAndPullRequests({
      q: scopedQuery,
      per_page: 1, // We only need the count, not the items
    });

    const totalCount = response.data.total_count;
    core.info(`Search found ${totalCount} matching items`);

    if (totalCount >= maxMatches) {
      core.warning(`🔍 Skip condition matched (${totalCount} items found, threshold: ${maxMatches}). Workflow execution will be prevented by activation job.`);
      core.setOutput("skip_check_ok", "false");
      return;
    }

    core.info(`✓ Found ${totalCount} matches (below threshold of ${maxMatches}), workflow can proceed`);
    core.setOutput("skip_check_ok", "true");
  } catch (error) {
    core.setFailed(`Failed to execute search query: ${error instanceof Error ? error.message : String(error)}`);
    return;
  }
}
await main();

EOF_CHECK_SKIP_IF_MATCH
echo "::notice::Copied: check_skip_if_match.cjs"
FILE_COUNT=$((FILE_COUNT + 1))

# === FILE: check_command_position.cjs ===
cat > "${DESTINATION}/check_command_position.cjs" << 'EOF_CHECK_COMMAND_POSITION'
// @ts-check
/// <reference types="@actions/github-script" />

/**
 * Check if command is the first word in the triggering text
 * This prevents accidental command triggers from words appearing later in content
 */
async function main() {
  const command = process.env.GH_AW_COMMAND;

  if (!command) {
    core.setFailed("Configuration error: GH_AW_COMMAND not specified.");
    return;
  }

  // Get the triggering text based on event type
  let text = "";
  const eventName = context.eventName;

  try {
    if (eventName === "issues") {
      text = context.payload.issue?.body || "";
    } else if (eventName === "pull_request") {
      text = context.payload.pull_request?.body || "";
    } else if (eventName === "issue_comment") {
      text = context.payload.comment?.body || "";
    } else if (eventName === "pull_request_review_comment") {
      text = context.payload.comment?.body || "";
    } else if (eventName === "discussion") {
      text = context.payload.discussion?.body || "";
    } else if (eventName === "discussion_comment") {
      text = context.payload.comment?.body || "";
    } else {
      // For non-comment events, pass the check
      core.info(`Event ${eventName} does not require command position check`);
      core.setOutput("command_position_ok", "true");
      return;
    }

    // Expected command format: /command
    const expectedCommand = `/${command}`;

    // If text is empty or doesn't contain the command at all, pass the check
    if (!text || !text.includes(expectedCommand)) {
      core.info(`No command '${expectedCommand}' found in text, passing check`);
      core.setOutput("command_position_ok", "true");
      return;
    }

    // Normalize whitespace and get the first word
    const trimmedText = text.trim();
    const firstWord = trimmedText.split(/\s+/)[0];

    core.info(`Checking command position for: ${expectedCommand}`);
    core.info(`First word in text: ${firstWord}`);

    if (firstWord === expectedCommand) {
      core.info(`✓ Command '${expectedCommand}' is at the start of the text`);
      core.setOutput("command_position_ok", "true");
    } else {
      core.warning(`⚠️ Command '${expectedCommand}' is not the first word (found: '${firstWord}'). Workflow will be skipped.`);
      core.setOutput("command_position_ok", "false");
    }
  } catch (error) {
    core.setFailed(error instanceof Error ? error.message : String(error));
  }
}

await main();

EOF_CHECK_COMMAND_POSITION
echo "::notice::Copied: check_command_position.cjs"
FILE_COUNT=$((FILE_COUNT + 1))

# === FILE: check_workflow_timestamp_api.cjs ===
cat > "${DESTINATION}/check_workflow_timestamp_api.cjs" << 'EOF_CHECK_WORKFLOW_TIMESTAMP_API'
// @ts-check
/// <reference types="@actions/github-script" />

/**
 * Check workflow file timestamps using GitHub API to detect outdated lock files
 * This script compares the last commit time of the source .md file
 * with the compiled .lock.yml file and warns if recompilation is needed
 */

async function main() {
  const workflowFile = process.env.GH_AW_WORKFLOW_FILE;

  if (!workflowFile) {
    core.setFailed("Configuration error: GH_AW_WORKFLOW_FILE not available.");
    return;
  }

  // Construct file paths
  const workflowBasename = workflowFile.replace(".lock.yml", "");
  const workflowMdPath = `.github/workflows/${workflowBasename}.md`;
  const lockFilePath = `.github/workflows/${workflowFile}`;

  core.info(`Checking workflow timestamps using GitHub API:`);
  core.info(`  Source: ${workflowMdPath}`);
  core.info(`  Lock file: ${lockFilePath}`);

  const { owner, repo } = context.repo;
  const ref = context.sha;

  // Helper function to get the last commit for a file
  async function getLastCommitForFile(path) {
    try {
      const response = await github.rest.repos.listCommits({
        owner,
        repo,
        path,
        per_page: 1,
        sha: ref,
      });

      if (response.data && response.data.length > 0) {
        const commit = response.data[0];
        return {
          sha: commit.sha,
          date: commit.commit.committer.date,
          message: commit.commit.message,
        };
      }
      return null;
    } catch (error) {
      core.info(`Could not fetch commit for ${path}: ${error.message}`);
      return null;
    }
  }

  // Fetch last commits for both files
  const workflowCommit = await getLastCommitForFile(workflowMdPath);
  const lockCommit = await getLastCommitForFile(lockFilePath);

  // Handle cases where files don't exist
  if (!workflowCommit) {
    core.info(`Source file does not exist: ${workflowMdPath}`);
  }

  if (!lockCommit) {
    core.info(`Lock file does not exist: ${lockFilePath}`);
  }

  if (!workflowCommit || !lockCommit) {
    core.info("Skipping timestamp check - one or both files not found");
    return;
  }

  // Parse dates for comparison
  const workflowDate = new Date(workflowCommit.date);
  const lockDate = new Date(lockCommit.date);

  core.info(`  Source last commit: ${workflowDate.toISOString()} (${workflowCommit.sha.substring(0, 7)})`);
  core.info(`  Lock last commit: ${lockDate.toISOString()} (${lockCommit.sha.substring(0, 7)})`);

  // Check if workflow file is newer than lock file
  if (workflowDate > lockDate) {
    const warningMessage = `WARNING: Lock file '${lockFilePath}' is outdated! The workflow file '${workflowMdPath}' has been modified more recently. Run 'gh aw compile' to regenerate the lock file.`;

    core.error(warningMessage);

    // Format timestamps and commits for display
    const workflowTimestamp = workflowDate.toISOString();
    const lockTimestamp = lockDate.toISOString();

    // Add summary to GitHub Step Summary
    let summary = core.summary
      .addRaw("### ⚠️ Workflow Lock File Warning\n\n")
      .addRaw("**WARNING**: Lock file is outdated and needs to be regenerated.\n\n")
      .addRaw("**Files:**\n")
      .addRaw(`- Source: \`${workflowMdPath}\`\n`)
      .addRaw(`  - Last commit: ${workflowTimestamp}\n`)
      .addRaw(`  - Commit SHA: [\`${workflowCommit.sha.substring(0, 7)}\`](https://github.com/${owner}/${repo}/commit/${workflowCommit.sha})\n`)
      .addRaw(`- Lock: \`${lockFilePath}\`\n`)
      .addRaw(`  - Last commit: ${lockTimestamp}\n`)
      .addRaw(`  - Commit SHA: [\`${lockCommit.sha.substring(0, 7)}\`](https://github.com/${owner}/${repo}/commit/${lockCommit.sha})\n\n`)
      .addRaw("**Action Required:** Run `gh aw compile` to regenerate the lock file.\n\n");

    await summary.write();
  } else if (workflowCommit.sha === lockCommit.sha) {
    core.info("✅ Lock file is up to date (same commit)");
  } else {
    core.info("✅ Lock file is up to date");
  }
}

main().catch(error => {
  core.setFailed(error instanceof Error ? error.message : String(error));
});

EOF_CHECK_WORKFLOW_TIMESTAMP_API
echo "::notice::Copied: check_workflow_timestamp_api.cjs"
FILE_COUNT=$((FILE_COUNT + 1))

# === FILE: lock-issue.cjs ===
cat > "${DESTINATION}/lock-issue.cjs" << 'EOF_LOCK_ISSUE'
// @ts-check
/// <reference types="@actions/github-script" />

/**
 * Lock a GitHub issue without providing a reason
 * This script is used in the activation job when lock-for-agent is enabled
 * to prevent concurrent modifications during agent workflow execution
 */

async function main() {
  // Log actor and event information for debugging
  core.info(`Lock-issue debug: actor=${context.actor}, eventName=${context.eventName}`);

  // Get issue number from context
  const issueNumber = context.issue.number;

  if (!issueNumber) {
    core.setFailed("Issue number not found in context");
    return;
  }

  const owner = context.repo.owner;
  const repo = context.repo.repo;

  core.info(`Lock-issue debug: owner=${owner}, repo=${repo}, issueNumber=${issueNumber}`);

  try {
    // Check if issue is already locked
    core.info(`Checking if issue #${issueNumber} is already locked`);
    const { data: issue } = await github.rest.issues.get({
      owner,
      repo,
      issue_number: issueNumber,
    });

    if (issue.locked) {
      core.info(`ℹ️ Issue #${issueNumber} is already locked, skipping lock operation`);
      core.setOutput("locked", "false");
      return;
    }

    core.info(`Locking issue #${issueNumber} for agent workflow execution`);

    // Lock the issue without providing a lock_reason parameter
    await github.rest.issues.lock({
      owner,
      repo,
      issue_number: issueNumber,
    });

    core.info(`✅ Successfully locked issue #${issueNumber}`);
    // Set output to indicate the issue was locked and needs to be unlocked
    core.setOutput("locked", "true");
  } catch (error) {
    const errorMessage = error instanceof Error ? error.message : String(error);
    core.error(`Failed to lock issue: ${errorMessage}`);
    core.setFailed(`Failed to lock issue #${issueNumber}: ${errorMessage}`);
    core.setOutput("locked", "false");
  }
}

await main();

EOF_LOCK_ISSUE
echo "::notice::Copied: lock-issue.cjs"
FILE_COUNT=$((FILE_COUNT + 1))

# === FILE: noop.cjs ===
cat > "${DESTINATION}/noop.cjs" << 'EOF_NOOP'
// @ts-check
/// <reference types="@actions/github-script" />

const { loadAgentOutput } = require("./load_agent_output.cjs");

/**
 * Main function to handle noop safe output
 * No-op is a fallback output type that logs messages for transparency
 * without taking any GitHub API actions
 */
async function main() {
  // Check if we're in staged mode
  const isStaged = process.env.GH_AW_SAFE_OUTPUTS_STAGED === "true";

  const result = loadAgentOutput();
  if (!result.success) {
    return;
  }

  // Find all noop items
  const noopItems = result.items.filter(/** @param {any} item */ item => item.type === "noop");
  if (noopItems.length === 0) {
    core.info("No noop items found in agent output");
    return;
  }

  core.info(`Found ${noopItems.length} noop item(s)`);

  // If in staged mode, emit step summary instead of logging
  if (isStaged) {
    let summaryContent = "## 🎭 Staged Mode: No-Op Messages Preview\n\n";
    summaryContent += "The following messages would be logged if staged mode was disabled:\n\n";

    for (let i = 0; i < noopItems.length; i++) {
      const item = noopItems[i];
      summaryContent += `### Message ${i + 1}\n`;
      summaryContent += `${item.message}\n\n`;
      summaryContent += "---\n\n";
    }

    await core.summary.addRaw(summaryContent).write();
    core.info("📝 No-op message preview written to step summary");
    return;
  }

  // Process each noop item - just log the messages for transparency
  let summaryContent = "\n\n## No-Op Messages\n\n";
  summaryContent += "The following messages were logged for transparency:\n\n";

  for (let i = 0; i < noopItems.length; i++) {
    const item = noopItems[i];
    core.info(`No-op message ${i + 1}: ${item.message}`);
    summaryContent += `- ${item.message}\n`;
  }

  // Write summary for all noop messages
  await core.summary.addRaw(summaryContent).write();

  // Export the first noop message for use in add-comment default reporting
  if (noopItems.length > 0) {
    core.setOutput("noop_message", noopItems[0].message);
    core.exportVariable("GH_AW_NOOP_MESSAGE", noopItems[0].message);
  }

  core.info(`Successfully processed ${noopItems.length} noop message(s)`);
}

await main();

EOF_NOOP
echo "::notice::Copied: noop.cjs"
FILE_COUNT=$((FILE_COUNT + 1))

# === FILE: unlock-issue.cjs ===
cat > "${DESTINATION}/unlock-issue.cjs" << 'EOF_UNLOCK_ISSUE'
// @ts-check
/// <reference types="@actions/github-script" />

/**
 * Unlock a GitHub issue
 * This script is used in the conclusion job to ensure the issue is unlocked
 * after agent workflow execution completes or fails
 */

async function main() {
  // Log actor and event information for debugging
  core.info(\`Unlock-issue debug: actor=\${context.actor}, eventName=\${context.eventName}\`);

  // Get issue number from context
  const issueNumber = context.issue.number;

  if (!issueNumber) {
    core.setFailed("Issue number not found in context");
    return;
  }

  const owner = context.repo.owner;
  const repo = context.repo.repo;

  core.info(\`Unlock-issue debug: owner=\${owner}, repo=\${repo}, issueNumber=\${issueNumber}\`);

  try {
    // Check if issue is locked
    core.info(\`Checking if issue #\${issueNumber} is locked\`);
    const { data: issue } = await github.rest.issues.get({
      owner,
      repo,
      issue_number: issueNumber,
    });

    // Skip unlocking if this is a pull request (PRs cannot be unlocked via issues API)
    if (issue.pull_request) {
      core.info(\`ℹ️ Issue #\${issueNumber} is a pull request, skipping unlock operation\`);
      return;
    }

    if (!issue.locked) {
      core.info(\`ℹ️ Issue #\${issueNumber} is not locked, skipping unlock operation\`);
      return;
    }

    core.info(\`Unlocking issue #\${issueNumber} after agent workflow execution\`);

    // Unlock the issue
    await github.rest.issues.unlock({
      owner,
      repo,
      issue_number: issueNumber,
    });

    core.info(\`✅ Successfully unlocked issue #\${issueNumber}\`);
  } catch (error) {
    const errorMessage = error instanceof Error ? error.message : String(error);
    core.error(\`Failed to unlock issue: \${errorMessage}\`);
    core.setFailed(\`Failed to unlock issue #\${issueNumber}: \${errorMessage}\`);
  }
}

await main();

EOF_UNLOCK_ISSUE
echo "::notice::Copied: unlock-issue.cjs"
FILE_COUNT=$((FILE_COUNT + 1))

# === FILE: missing_tool.cjs ===
cat > "${DESTINATION}/missing_tool.cjs" << 'EOF_MISSING_TOOL'
// @ts-check
/// <reference types="@actions/github-script" />

async function main() {
  const fs = require("fs");

  // Get environment variables
  const agentOutputFile = process.env.GH_AW_AGENT_OUTPUT || "";
  const maxReports = process.env.GH_AW_MISSING_TOOL_MAX ? parseInt(process.env.GH_AW_MISSING_TOOL_MAX) : null;

  core.info("Processing missing-tool reports...");
  if (maxReports) {
    core.info(\`Maximum reports allowed: \${maxReports}\`);
  }

  /** @type {any[]} */
  const missingTools = [];

  // Return early if no agent output
  if (!agentOutputFile.trim()) {
    core.info("No agent output to process");
    core.setOutput("tools_reported", JSON.stringify(missingTools));
    core.setOutput("total_count", missingTools.length.toString());
    return;
  }

  // Read agent output from file
  let agentOutput;
  try {
    agentOutput = fs.readFileSync(agentOutputFile, "utf8");
  } catch (error) {
    core.info(\`Agent output file not found or unreadable: \${error instanceof Error ? error.message : String(error)}\`);
    core.setOutput("tools_reported", JSON.stringify(missingTools));
    core.setOutput("total_count", missingTools.length.toString());
    return;
  }

  if (agentOutput.trim() === "") {
    core.info("No agent output to process");
    core.setOutput("tools_reported", JSON.stringify(missingTools));
    core.setOutput("total_count", missingTools.length.toString());
    return;
  }

  core.info(\`Agent output length: \${agentOutput.length}\`);

  // Parse the validated output JSON
  let validatedOutput;
  try {
    validatedOutput = JSON.parse(agentOutput);
  } catch (error) {
    core.setFailed(\`Error parsing agent output JSON: \${error instanceof Error ? error.message : String(error)}\`);
    return;
  }

  if (!validatedOutput.items || !Array.isArray(validatedOutput.items)) {
    core.info("No valid items found in agent output");
    core.setOutput("tools_reported", JSON.stringify(missingTools));
    core.setOutput("total_count", missingTools.length.toString());
    return;
  }

  core.info(\`Parsed agent output with \${validatedOutput.items.length} entries\`);

  // Process all parsed entries
  for (const entry of validatedOutput.items) {
    if (entry.type === "missing_tool") {
      // Validate required fields
      if (!entry.tool) {
        core.warning(\`missing-tool entry missing 'tool' field: \${JSON.stringify(entry)}\`);
        continue;
      }
      if (!entry.reason) {
        core.warning(\`missing-tool entry missing 'reason' field: \${JSON.stringify(entry)}\`);
        continue;
      }

      const missingTool = {
        tool: entry.tool,
        reason: entry.reason,
        alternatives: entry.alternatives || null,
        timestamp: new Date().toISOString(),
      };

      missingTools.push(missingTool);
      core.info(\`Recorded missing tool: \${missingTool.tool}\`);

      // Check max limit
      if (maxReports && missingTools.length >= maxReports) {
        core.info(\`Reached maximum number of missing tool reports (\${maxReports})\`);
        break;
      }
    }
  }

  core.info(\`Total missing tools reported: \${missingTools.length}\`);

  // Output results
  core.setOutput("tools_reported", JSON.stringify(missingTools));
  core.setOutput("total_count", missingTools.length.toString());

  // Log details for debugging and create step summary
  if (missingTools.length > 0) {
    core.info("Missing tools summary:");

    // Create structured summary for GitHub Actions step summary
    core.summary.addHeading("Missing Tools Report", 3).addRaw(\`Found **\${missingTools.length}** missing tool\${missingTools.length > 1 ? "s" : ""} in this workflow execution.\\n\\n\`);

    missingTools.forEach((tool, index) => {
      core.info(\`\${index + 1}. Tool: \${tool.tool}\`);
      core.info(\`   Reason: \${tool.reason}\`);
      if (tool.alternatives) {
        core.info(\`   Alternatives: \${tool.alternatives}\`);
      }
      core.info(\`   Reported at: \${tool.timestamp}\`);
      core.info("");

      // Add to summary with structured formatting
      core.summary.addRaw(\`#### \${index + 1}. \\\`\${tool.tool}\\\`\\n\\n\`).addRaw(\`**Reason:** \${tool.reason}\\n\\n\`);

      if (tool.alternatives) {
        core.summary.addRaw(\`**Alternatives:** \${tool.alternatives}\\n\\n\`);
      }

      core.summary.addRaw(\`**Reported at:** \${tool.timestamp}\\n\\n---\\n\\n\`);
    });

    core.summary.write();
  } else {
    core.info("No missing tools reported in this workflow execution.");
    core.summary.addHeading("Missing Tools Report", 3).addRaw("✅ No missing tools reported in this workflow execution.").write();
  }
}

main().catch(error => {
  core.error(\`Error processing missing-tool reports: \${error}\`);
  core.setFailed(\`Error processing missing-tool reports: \${error}\`);
});

EOF_MISSING_TOOL

# === FILE: notify_comment_error.cjs ===
cat > "${DESTINATION}/notify_comment_error.cjs" << 'EOF_NOTIFY_COMMENT_ERROR'
// @ts-check
/// <reference types="@actions/github-script" />

// This script updates an existing comment created by the activation job
// to notify about the workflow completion status (success or failure).
// It also processes noop messages and adds them to the activation comment.

const { loadAgentOutput } = require("./load_agent_output.cjs");
const { getRunSuccessMessage, getRunFailureMessage, getDetectionFailureMessage } = require("./messages_run_status.cjs");

/**
 * Collect generated asset URLs from safe output jobs
 * @returns {Array<string>} Array of generated asset URLs
 */
function collectGeneratedAssets() {
  const assets = [];

  // Get the safe output jobs mapping from environment
  const safeOutputJobsEnv = process.env.GH_AW_SAFE_OUTPUT_JOBS;
  if (!safeOutputJobsEnv) {
    return assets;
  }

  let jobOutputMapping;
  try {
    jobOutputMapping = JSON.parse(safeOutputJobsEnv);
  } catch (error) {
    core.warning(\`Failed to parse GH_AW_SAFE_OUTPUT_JOBS: \${error instanceof Error ? error.message : String(error)}\`);
    return assets;
  }

  // Iterate through each job and collect its URL output
  for (const [jobName, urlKey] of Object.entries(jobOutputMapping)) {
    // Access the job output using the GitHub Actions context
    // The value will be set as an environment variable in the format GH_AW_OUTPUT_<JOB>_<KEY>
    const envVarName = \`GH_AW_OUTPUT_\${jobName.toUpperCase()}_\${urlKey.toUpperCase()}\`;
    const url = process.env[envVarName];

    if (url && url.trim() !== "") {
      assets.push(url);
      core.info(\`Collected asset URL: \${url}\`);
    }
  }

  return assets;
}

async function main() {
  const commentId = process.env.GH_AW_COMMENT_ID;
  const commentRepo = process.env.GH_AW_COMMENT_REPO;
  const runUrl = process.env.GH_AW_RUN_URL;
  const workflowName = process.env.GH_AW_WORKFLOW_NAME || "Workflow";
  const agentConclusion = process.env.GH_AW_AGENT_CONCLUSION || "failure";
  const detectionConclusion = process.env.GH_AW_DETECTION_CONCLUSION;

  core.info(\`Comment ID: \${commentId}\`);
  core.info(\`Comment Repo: \${commentRepo}\`);
  core.info(\`Run URL: \${runUrl}\`);
  core.info(\`Workflow Name: \${workflowName}\`);
  core.info(\`Agent Conclusion: \${agentConclusion}\`);
  if (detectionConclusion) {
    core.info(\`Detection Conclusion: \${detectionConclusion}\`);
  }

  // Load agent output to check for noop messages
  let noopMessages = [];
  const agentOutputResult = loadAgentOutput();
  if (agentOutputResult.success && agentOutputResult.data) {
    const noopItems = agentOutputResult.data.items.filter(item => item.type === "noop");
    if (noopItems.length > 0) {
      core.info(\`Found \${noopItems.length} noop message(s)\`);
      noopMessages = noopItems.map(item => item.message);
    }
  }

  // If there's no comment to update but we have noop messages, write to step summary
  if (!commentId && noopMessages.length > 0) {
    core.info("No comment ID found, writing noop messages to step summary");

    let summaryContent = "## No-Op Messages\\n\\n";
    summaryContent += "The following messages were logged for transparency:\\n\\n";

    if (noopMessages.length === 1) {
      summaryContent += noopMessages[0];
    } else {
      summaryContent += noopMessages.map((msg, idx) => \`\${idx + 1}. \${msg}\`).join("\\n");
    }

    await core.summary.addRaw(summaryContent).write();
    core.info(\`Successfully wrote \${noopMessages.length} noop message(s) to step summary\`);
    return;
  }

  if (!commentId) {
    core.info("No comment ID found and no noop messages to process, skipping comment update");
    return;
  }

  // At this point, we have a comment to update
  if (!runUrl) {
    core.setFailed("Run URL is required");
    return;
  }

  // Parse comment repo (format: "owner/repo")
  const repoOwner = commentRepo ? commentRepo.split("/")[0] : context.repo.owner;
  const repoName = commentRepo ? commentRepo.split("/")[1] : context.repo.repo;

  core.info(\`Updating comment in \${repoOwner}/\${repoName}\`);

  // Determine the message based on agent conclusion using custom messages if configured
  let message;

  // Check if detection job failed (if detection job exists)
  if (detectionConclusion && detectionConclusion === "failure") {
    // Detection job failed - report this prominently
    message = getDetectionFailureMessage({
      workflowName,
      runUrl,
    });
  } else if (agentConclusion === "success") {
    message = getRunSuccessMessage({
      workflowName,
      runUrl,
    });
  } else {
    // Determine status text based on conclusion type
    let statusText;
    if (agentConclusion === "cancelled") {
      statusText = "was cancelled";
    } else if (agentConclusion === "skipped") {
      statusText = "was skipped";
    } else if (agentConclusion === "timed_out") {
      statusText = "timed out";
    } else {
      statusText = "failed";
    }

    message = getRunFailureMessage({
      workflowName,
      runUrl,
      status: statusText,
    });
  }

  // Add noop messages to the comment if any
  if (noopMessages.length > 0) {
    message += "\\n\\n";
    if (noopMessages.length === 1) {
      message += noopMessages[0];
    } else {
      message += noopMessages.map((msg, idx) => \`\${idx + 1}. \${msg}\`).join("\\n");
    }
  }

  // Collect generated asset URLs from safe output jobs
  const generatedAssets = collectGeneratedAssets();
  if (generatedAssets.length > 0) {
    message += "\\n\\n";
    generatedAssets.forEach(url => {
      message += \`\${url}\\n\`;
    });
  }

  // Check if this is a discussion comment (GraphQL node ID format)
  const isDiscussionComment = commentId.startsWith("DC_");

  try {
    if (isDiscussionComment) {
      // Update discussion comment using GraphQL
      const result = await github.graphql(
        \`
        mutation($commentId: ID!, $body: String!) {
          updateDiscussionComment(input: { commentId: $commentId, body: $body }) {
            comment {
              id
              url
            }
          }
        }\`,
        { commentId: commentId, body: message }
      );

      const comment = result.updateDiscussionComment.comment;
      core.info(\`Successfully updated discussion comment\`);
      core.info(\`Comment ID: \${comment.id}\`);
      core.info(\`Comment URL: \${comment.url}\`);
    } else {
      // Update issue/PR comment using REST API
      const response = await github.request("PATCH /repos/{owner}/{repo}/issues/comments/{comment_id}", {
        owner: repoOwner,
        repo: repoName,
        comment_id: parseInt(commentId, 10),
        body: message,
        headers: {
          Accept: "application/vnd.github+json",
        },
      });

      core.info(\`Successfully updated comment\`);
      core.info(\`Comment ID: \${response.data.id}\`);
      core.info(\`Comment URL: \${response.data.html_url}\`);
    }
  } catch (error) {
    // Don't fail the workflow if we can't update the comment
    core.warning(\`Failed to update comment: \${error instanceof Error ? error.message : String(error)}\`);
  }
}

main().catch(error => {
  core.setFailed(error instanceof Error ? error.message : String(error));
});

EOF_NOTIFY_COMMENT_ERROR
echo "::notice::Copied: notify_comment_error.cjs"
FILE_COUNT=$((FILE_COUNT + 1))

echo "::notice::Copied: missing_tool.cjs"
FILE_COUNT=$((FILE_COUNT + 1))

# === FILE: compute_text.cjs ===
cat > "${DESTINATION}/compute_text.cjs" << 'EOF_COMPUTE_TEXT'
// @ts-check
/// <reference types="@actions/github-script" />

/**
 * Sanitizes content for safe output in GitHub Actions
 * @param {string} content - The content to sanitize
 * @returns {string} The sanitized content
 */
const { sanitizeIncomingText, writeRedactedDomainsLog } = require("./sanitize_incoming_text.cjs");

async function main() {
  let text = "";

  const actor = context.actor;
  const { owner, repo } = context.repo;

  // Check if the actor has repository access (admin, maintain permissions)
  const repoPermission = await github.rest.repos.getCollaboratorPermissionLevel({
    owner: owner,
    repo: repo,
    username: actor,
  });

  const permission = repoPermission.data.permission;
  core.info(`Repository permission level: ${permission}`);

  if (permission !== "admin" && permission !== "maintain") {
    core.setOutput("text", "");
    return;
  }

  // Determine current body text based on event context
  switch (context.eventName) {
    case "issues":
      // For issues: title + body
      if (context.payload.issue) {
        const title = context.payload.issue.title || "";
        const body = context.payload.issue.body || "";
        text = `${title}\n\n${body}`;
      }
      break;

    case "pull_request":
      // For pull requests: title + body
      if (context.payload.pull_request) {
        const title = context.payload.pull_request.title || "";
        const body = context.payload.pull_request.body || "";
        text = `${title}\n\n${body}`;
      }
      break;

    case "pull_request_target":
      // For pull request target events: title + body
      if (context.payload.pull_request) {
        const title = context.payload.pull_request.title || "";
        const body = context.payload.pull_request.body || "";
        text = `${title}\n\n${body}`;
      }
      break;

    case "issue_comment":
      // For issue comments: comment body
      if (context.payload.comment) {
        text = context.payload.comment.body || "";
      }
      break;

    case "pull_request_review_comment":
      // For PR review comments: comment body
      if (context.payload.comment) {
        text = context.payload.comment.body || "";
      }
      break;

    case "pull_request_review":
      // For PR reviews: review body
      if (context.payload.review) {
        text = context.payload.review.body || "";
      }
      break;

    case "discussion":
      // For discussions: title + body
      if (context.payload.discussion) {
        const title = context.payload.discussion.title || "";
        const body = context.payload.discussion.body || "";
        text = `${title}\n\n${body}`;
      }
      break;

    case "discussion_comment":
      // For discussion comments: comment body
      if (context.payload.comment) {
        text = context.payload.comment.body || "";
      }
      break;

    case "release":
      // For releases: name + body
      if (context.payload.release) {
        const name = context.payload.release.name || context.payload.release.tag_name || "";
        const body = context.payload.release.body || "";
        text = `${name}\n\n${body}`;
      }
      break;

    case "workflow_dispatch":
      // For workflow dispatch: check for release_url or release_id in inputs
      if (context.payload.inputs) {
        const releaseUrl = context.payload.inputs.release_url;
        const releaseId = context.payload.inputs.release_id;

        // If release_url is provided, extract owner/repo/tag
        if (releaseUrl) {
          const urlMatch = releaseUrl.match(/github\.com\/([^\/]+)\/([^\/]+)\/releases\/tag\/([^\/]+)/);
          if (urlMatch) {
            const [, urlOwner, urlRepo, tag] = urlMatch;
            try {
              const { data: release } = await github.rest.repos.getReleaseByTag({
                owner: urlOwner,
                repo: urlRepo,
                tag: tag,
              });
              const name = release.name || release.tag_name || "";
              const body = release.body || "";
              text = `${name}\n\n${body}`;
            } catch (error) {
              core.warning(`Failed to fetch release from URL: ${error instanceof Error ? error.message : String(error)}`);
            }
          }
        } else if (releaseId) {
          // If release_id is provided, fetch the release
          try {
            const { data: release } = await github.rest.repos.getRelease({
              owner: owner,
              repo: repo,
              release_id: parseInt(releaseId, 10),
            });
            const name = release.name || release.tag_name || "";
            const body = release.body || "";
            text = `${name}\n\n${body}`;
          } catch (error) {
            core.warning(`Failed to fetch release by ID: ${error instanceof Error ? error.message : String(error)}`);
          }
        }
      }
      break;

    default:
      // Default: empty text
      text = "";
      break;
  }

  // Sanitize the text before output
  // All mentions are escaped (wrapped in backticks) to prevent unintended notifications
  // Mention filtering will be applied by the agent output collector
  const sanitizedText = sanitizeIncomingText(text);

  // Display sanitized text in logs
  core.info(`text: ${sanitizedText}`);

  // Set the sanitized text as output
  core.setOutput("text", sanitizedText);

  // Write redacted URL domains to log file if any were collected
  const logPath = writeRedactedDomainsLog();
  if (logPath) {
    core.info(`Redacted URL domains written to: ${logPath}`);
  }
}

await main();

EOF_COMPUTE_TEXT
echo "::notice::Copied: compute_text.cjs"
FILE_COUNT=$((FILE_COUNT + 1))

# === FILE: add_reaction_and_edit_comment.cjs ===
cat > "${DESTINATION}/add_reaction_and_edit_comment.cjs" << 'EOF_ADD_REACTION_AND_EDIT_COMMENT'
// @ts-check
/// <reference types="@actions/github-script" />

const { getRunStartedMessage } = require("./messages_run_status.cjs");

async function main() {
  // Read inputs from environment variables
  const reaction = process.env.GH_AW_REACTION || "eyes";
  const command = process.env.GH_AW_COMMAND; // Only present for command workflows
  const runId = context.runId;
  const githubServer = process.env.GITHUB_SERVER_URL || "https://github.com";
  const runUrl = context.payload.repository ? `${context.payload.repository.html_url}/actions/runs/${runId}` : `${githubServer}/${context.repo.owner}/${context.repo.repo}/actions/runs/${runId}`;

  core.info(`Reaction type: ${reaction}`);
  core.info(`Command name: ${command || "none"}`);
  core.info(`Run ID: ${runId}`);
  core.info(`Run URL: ${runUrl}`);

  // Validate reaction type
  const validReactions = ["+1", "-1", "laugh", "confused", "heart", "hooray", "rocket", "eyes"];
  if (!validReactions.includes(reaction)) {
    core.setFailed(`Invalid reaction type: ${reaction}. Valid reactions are: ${validReactions.join(", ")}`);
    return;
  }

  // Determine the API endpoint based on the event type
  let reactionEndpoint;
  let commentUpdateEndpoint;
  let shouldCreateComment = false;
  const eventName = context.eventName;
  const owner = context.repo.owner;
  const repo = context.repo.repo;

  try {
    switch (eventName) {
      case "issues":
        const issueNumber = context.payload?.issue?.number;
        if (!issueNumber) {
          core.setFailed("Issue number not found in event payload");
          return;
        }
        reactionEndpoint = `/repos/${owner}/${repo}/issues/${issueNumber}/reactions`;
        commentUpdateEndpoint = `/repos/${owner}/${repo}/issues/${issueNumber}/comments`;
        // Create comments for all workflows using reactions
        shouldCreateComment = true;
        break;

      case "issue_comment":
        const commentId = context.payload?.comment?.id;
        const issueNumberForComment = context.payload?.issue?.number;
        if (!commentId) {
          core.setFailed("Comment ID not found in event payload");
          return;
        }
        if (!issueNumberForComment) {
          core.setFailed("Issue number not found in event payload");
          return;
        }
        reactionEndpoint = `/repos/${owner}/${repo}/issues/comments/${commentId}/reactions`;
        // Create new comment on the issue itself, not on the comment
        commentUpdateEndpoint = `/repos/${owner}/${repo}/issues/${issueNumberForComment}/comments`;
        // Create comments for all workflows using reactions
        shouldCreateComment = true;
        break;

      case "pull_request":
        const prNumber = context.payload?.pull_request?.number;
        if (!prNumber) {
          core.setFailed("Pull request number not found in event payload");
          return;
        }
        // PRs are "issues" for the reactions endpoint
        reactionEndpoint = `/repos/${owner}/${repo}/issues/${prNumber}/reactions`;
        commentUpdateEndpoint = `/repos/${owner}/${repo}/issues/${prNumber}/comments`;
        // Create comments for all workflows using reactions
        shouldCreateComment = true;
        break;

      case "pull_request_review_comment":
        const reviewCommentId = context.payload?.comment?.id;
        const prNumberForReviewComment = context.payload?.pull_request?.number;
        if (!reviewCommentId) {
          core.setFailed("Review comment ID not found in event payload");
          return;
        }
        if (!prNumberForReviewComment) {
          core.setFailed("Pull request number not found in event payload");
          return;
        }
        reactionEndpoint = `/repos/${owner}/${repo}/pulls/comments/${reviewCommentId}/reactions`;
        // Create new comment on the PR itself (using issues endpoint since PRs are issues)
        commentUpdateEndpoint = `/repos/${owner}/${repo}/issues/${prNumberForReviewComment}/comments`;
        // Create comments for all workflows using reactions
        shouldCreateComment = true;
        break;

      case "discussion":
        const discussionNumber = context.payload?.discussion?.number;
        if (!discussionNumber) {
          core.setFailed("Discussion number not found in event payload");
          return;
        }
        // Discussions use GraphQL API - get the node ID
        const discussion = await getDiscussionId(owner, repo, discussionNumber);
        reactionEndpoint = discussion.id; // Store node ID for GraphQL
        commentUpdateEndpoint = `discussion:${discussionNumber}`; // Special format to indicate discussion
        // Create comments for all workflows using reactions
        shouldCreateComment = true;
        break;

      case "discussion_comment":
        const discussionCommentNumber = context.payload?.discussion?.number;
        const discussionCommentId = context.payload?.comment?.id;
        if (!discussionCommentNumber || !discussionCommentId) {
          core.setFailed("Discussion or comment information not found in event payload");
          return;
        }
        // Get the comment node ID from the payload
        const commentNodeId = context.payload?.comment?.node_id;
        if (!commentNodeId) {
          core.setFailed("Discussion comment node ID not found in event payload");
          return;
        }
        reactionEndpoint = commentNodeId; // Store node ID for GraphQL
        commentUpdateEndpoint = `discussion_comment:${discussionCommentNumber}:${discussionCommentId}`; // Special format
        // Create comments for all workflows using reactions
        shouldCreateComment = true;
        break;

      default:
        core.setFailed(`Unsupported event type: ${eventName}`);
        return;
    }

    core.info(`Reaction API endpoint: ${reactionEndpoint}`);

    // Add reaction first
    // For discussions, reactionEndpoint is a node ID (GraphQL), otherwise it's a REST API path
    const isDiscussionEvent = eventName === "discussion" || eventName === "discussion_comment";
    if (isDiscussionEvent) {
      await addDiscussionReaction(reactionEndpoint, reaction);
    } else {
      await addReaction(reactionEndpoint, reaction);
    }

    // Then add comment if applicable
    if (shouldCreateComment && commentUpdateEndpoint) {
      core.info(`Comment endpoint: ${commentUpdateEndpoint}`);
      await addCommentWithWorkflowLink(commentUpdateEndpoint, runUrl, eventName);
    } else {
      core.info(`Skipping comment for event type: ${eventName}`);
    }
  } catch (error) {
    const errorMessage = error instanceof Error ? error.message : String(error);
    core.error(`Failed to process reaction and comment creation: ${errorMessage}`);
    core.setFailed(`Failed to process reaction and comment creation: ${errorMessage}`);
  }
}

/**
 * Add a reaction to a GitHub issue, PR, or comment using REST API
 * @param {string} endpoint - The GitHub API endpoint to add the reaction to
 * @param {string} reaction - The reaction type to add
 */
async function addReaction(endpoint, reaction) {
  const response = await github.request("POST " + endpoint, {
    content: reaction,
    headers: {
      Accept: "application/vnd.github+json",
    },
  });

  const reactionId = response.data?.id;
  if (reactionId) {
    core.info(`Successfully added reaction: ${reaction} (id: ${reactionId})`);
    core.setOutput("reaction-id", reactionId.toString());
  } else {
    core.info(`Successfully added reaction: ${reaction}`);
    core.setOutput("reaction-id", "");
  }
}

/**
 * Add a reaction to a GitHub discussion or discussion comment using GraphQL
 * @param {string} subjectId - The node ID of the discussion or comment
 * @param {string} reaction - The reaction type to add (mapped to GitHub's ReactionContent enum)
 */
async function addDiscussionReaction(subjectId, reaction) {
  // Map reaction names to GitHub's GraphQL ReactionContent enum
  const reactionMap = {
    "+1": "THUMBS_UP",
    "-1": "THUMBS_DOWN",
    laugh: "LAUGH",
    confused: "CONFUSED",
    heart: "HEART",
    hooray: "HOORAY",
    rocket: "ROCKET",
    eyes: "EYES",
  };

  const reactionContent = reactionMap[reaction];
  if (!reactionContent) {
    throw new Error(`Invalid reaction type for GraphQL: ${reaction}`);
  }

  const result = await github.graphql(
    `
    mutation($subjectId: ID!, $content: ReactionContent!) {
      addReaction(input: { subjectId: $subjectId, content: $content }) {
        reaction {
          id
          content
        }
      }
    }`,
    { subjectId, content: reactionContent }
  );

  const reactionId = result.addReaction.reaction.id;
  core.info(`Successfully added reaction: ${reaction} (id: ${reactionId})`);
  core.setOutput("reaction-id", reactionId);
}

/**
 * Get the node ID for a discussion
 * @param {string} owner - Repository owner
 * @param {string} repo - Repository name
 * @param {number} discussionNumber - Discussion number
 * @returns {Promise<{id: string, url: string}>} Discussion details
 */
async function getDiscussionId(owner, repo, discussionNumber) {
  const { repository } = await github.graphql(
    `
    query($owner: String!, $repo: String!, $num: Int!) {
      repository(owner: $owner, name: $repo) {
        discussion(number: $num) { 
          id 
          url
        }
      }
    }`,
    { owner, repo, num: discussionNumber }
  );

  if (!repository || !repository.discussion) {
    throw new Error(`Discussion #${discussionNumber} not found in ${owner}/${repo}`);
  }

  return {
    id: repository.discussion.id,
    url: repository.discussion.url,
  };
}

/**
 * Get the node ID for a discussion comment
 * @param {string} owner - Repository owner
 * @param {string} repo - Repository name
 * @param {number} discussionNumber - Discussion number
 * @param {number} commentId - Comment ID (database ID, not node ID)
 * @returns {Promise<{id: string, url: string}>} Comment details
 */
async function getDiscussionCommentId(owner, repo, discussionNumber, commentId) {
  // First, get the discussion ID
  const discussion = await getDiscussionId(owner, repo, discussionNumber);
  if (!discussion) throw new Error(`Discussion #${discussionNumber} not found in ${owner}/${repo}`);

  // Then fetch the comment by traversing discussion comments
  // Note: GitHub's GraphQL API doesn't provide a direct way to query comment by database ID
  // We need to use the comment's node ID from the event payload if available
  // For now, we'll use a simplified approach - the commentId from context.payload.comment.node_id

  // If the event payload provides node_id, we can use it directly
  // Otherwise, this would need to fetch all comments and find the matching one
  const nodeId = context.payload?.comment?.node_id;
  if (nodeId) {
    return {
      id: nodeId,
      url: context.payload.comment?.html_url || discussion?.url,
    };
  }

  throw new Error(`Discussion comment node ID not found in event payload for comment ${commentId}`);
}

/**
 * Add a comment with a workflow run link
 * @param {string} endpoint - The GitHub API endpoint to create the comment (or special format for discussions)
 * @param {string} runUrl - The URL of the workflow run
 * @param {string} eventName - The event type (to determine the comment text)
 */
async function addCommentWithWorkflowLink(endpoint, runUrl, eventName) {
  try {
    // Get workflow name from environment variable
    const workflowName = process.env.GH_AW_WORKFLOW_NAME || "Workflow";

    // Determine the event type description
    let eventTypeDescription;
    switch (eventName) {
      case "issues":
        eventTypeDescription = "issue";
        break;
      case "pull_request":
        eventTypeDescription = "pull request";
        break;
      case "issue_comment":
        eventTypeDescription = "issue comment";
        break;
      case "pull_request_review_comment":
        eventTypeDescription = "pull request review comment";
        break;
      case "discussion":
        eventTypeDescription = "discussion";
        break;
      case "discussion_comment":
        eventTypeDescription = "discussion comment";
        break;
      default:
        eventTypeDescription = "event";
    }

    // Use getRunStartedMessage for the workflow link text (supports custom messages)
    const workflowLinkText = getRunStartedMessage({
      workflowName: workflowName,
      runUrl: runUrl,
      eventType: eventTypeDescription,
    });

    // Add workflow-id and tracker-id markers for hide-older-comments feature
    const workflowId = process.env.GITHUB_WORKFLOW || "";
    const trackerId = process.env.GH_AW_TRACKER_ID || "";

    let commentBody = workflowLinkText;

    // Add lock notice if lock-for-agent is enabled for issues or issue_comment
    const lockForAgent = process.env.GH_AW_LOCK_FOR_AGENT === "true";
    if (lockForAgent && (eventName === "issues" || eventName === "issue_comment")) {
      commentBody += "\n\n🔒 This issue has been locked while the workflow is running to prevent concurrent modifications.";
    }

    // Add workflow-id marker if available
    if (workflowId) {
      commentBody += `\n\n<!-- workflow-id: ${workflowId} -->`;
    }

    // Add tracker-id marker if available (for backwards compatibility)
    if (trackerId) {
      commentBody += `\n\n<!-- tracker-id: ${trackerId} -->`;
    }

    // Add comment type marker to identify this as a reaction comment
    // This prevents it from being hidden by hide-older-comments
    commentBody += `\n\n<!-- comment-type: reaction -->`;

    // Handle discussion events specially
    if (eventName === "discussion") {
      // Parse discussion number from special format: "discussion:NUMBER"
      const discussionNumber = parseInt(endpoint.split(":")[1], 10);

      // Create a new comment on the discussion using GraphQL
      const { repository } = await github.graphql(
        `
        query($owner: String!, $repo: String!, $num: Int!) {
          repository(owner: $owner, name: $repo) {
            discussion(number: $num) { 
              id 
            }
          }
        }`,
        { owner: context.repo.owner, repo: context.repo.repo, num: discussionNumber }
      );

      const discussionId = repository.discussion.id;

      const result = await github.graphql(
        `
        mutation($dId: ID!, $body: String!) {
          addDiscussionComment(input: { discussionId: $dId, body: $body }) {
            comment { 
              id 
              url
            }
          }
        }`,
        { dId: discussionId, body: commentBody }
      );

      const comment = result.addDiscussionComment.comment;
      core.info(`Successfully created discussion comment with workflow link`);
      core.info(`Comment ID: ${comment.id}`);
      core.info(`Comment URL: ${comment.url}`);
      core.info(`Comment Repo: ${context.repo.owner}/${context.repo.repo}`);
      core.setOutput("comment-id", comment.id);
      core.setOutput("comment-url", comment.url);
      core.setOutput("comment-repo", `${context.repo.owner}/${context.repo.repo}`);
      return;
    } else if (eventName === "discussion_comment") {
      // Parse discussion number from special format: "discussion_comment:NUMBER:COMMENT_ID"
      const discussionNumber = parseInt(endpoint.split(":")[1], 10);

      // Create a new comment on the discussion using GraphQL
      const { repository } = await github.graphql(
        `
        query($owner: String!, $repo: String!, $num: Int!) {
          repository(owner: $owner, name: $repo) {
            discussion(number: $num) { 
              id 
            }
          }
        }`,
        { owner: context.repo.owner, repo: context.repo.repo, num: discussionNumber }
      );

      const discussionId = repository.discussion.id;

      // Get the comment node ID to use as the parent for threading
      const commentNodeId = context.payload?.comment?.node_id;

      const result = await github.graphql(
        `
        mutation($dId: ID!, $body: String!, $replyToId: ID!) {
          addDiscussionComment(input: { discussionId: $dId, body: $body, replyToId: $replyToId }) {
            comment { 
              id 
              url
            }
          }
        }`,
        { dId: discussionId, body: commentBody, replyToId: commentNodeId }
      );

      const comment = result.addDiscussionComment.comment;
      core.info(`Successfully created discussion comment with workflow link`);
      core.info(`Comment ID: ${comment.id}`);
      core.info(`Comment URL: ${comment.url}`);
      core.info(`Comment Repo: ${context.repo.owner}/${context.repo.repo}`);
      core.setOutput("comment-id", comment.id);
      core.setOutput("comment-url", comment.url);
      core.setOutput("comment-repo", `${context.repo.owner}/${context.repo.repo}`);
      return;
    }

    // Create a new comment for non-discussion events
    const createResponse = await github.request("POST " + endpoint, {
      body: commentBody,
      headers: {
        Accept: "application/vnd.github+json",
      },
    });

    core.info(`Successfully created comment with workflow link`);
    core.info(`Comment ID: ${createResponse.data.id}`);
    core.info(`Comment URL: ${createResponse.data.html_url}`);
    core.info(`Comment Repo: ${context.repo.owner}/${context.repo.repo}`);
    core.setOutput("comment-id", createResponse.data.id.toString());
    core.setOutput("comment-url", createResponse.data.html_url);
    core.setOutput("comment-repo", `${context.repo.owner}/${context.repo.repo}`);
  } catch (error) {
    // Don't fail the entire job if comment creation fails - just log it
    const errorMessage = error instanceof Error ? error.message : String(error);
    core.warning("Failed to create comment with workflow link (This is not critical - the reaction was still added successfully): " + errorMessage);
  }
}

await main();

EOF_ADD_REACTION_AND_EDIT_COMMENT
echo "::notice::Copied: add_reaction_and_edit_comment.cjs"
FILE_COUNT=$((FILE_COUNT + 1))

# === FILE: sanitize_incoming_text.cjs ===
cat > "${DESTINATION}/sanitize_incoming_text.cjs" << 'EOF_SANITIZE_INCOMING_TEXT'
// @ts-check
/**
 * Slimmed-down sanitization for incoming text (compute_text)
 * This version does NOT include mention filtering - all @mentions are escaped
 */

const { sanitizeContentCore, writeRedactedDomainsLog } = require("./sanitize_content_core.cjs");

/**
 * Sanitizes incoming text content without selective mention filtering
 * All @mentions are escaped to prevent unintended notifications
 *
 * Uses the core sanitization functions directly to minimize bundle size.
 *
 * @param {string} content - The content to sanitize
 * @param {number} [maxLength] - Maximum length of content (default: 524288)
 * @returns {string} The sanitized content with all mentions escaped
 */
function sanitizeIncomingText(content, maxLength) {
  // Call core sanitization which neutralizes all mentions
  return sanitizeContentCore(content, maxLength);
}

module.exports = {
  sanitizeIncomingText,
  writeRedactedDomainsLog,
};

EOF_SANITIZE_INCOMING_TEXT
echo "::notice::Copied: sanitize_incoming_text.cjs"
FILE_COUNT=$((FILE_COUNT + 1))

# === FILE: messages_run_status.cjs ===
cat > "${DESTINATION}/messages_run_status.cjs" << 'EOF_MESSAGES_RUN_STATUS'
// @ts-check
/// <reference types="@actions/github-script" />

/**
 * Run Status Message Module
 *
 * This module provides run status messages (started, success, failure)
 * for workflow execution notifications.
 */

const { getMessages, renderTemplate, toSnakeCase } = require("./messages_core.cjs");

/**
 * @typedef {Object} RunStartedContext
 * @property {string} workflowName - Name of the workflow
 * @property {string} runUrl - URL of the workflow run
 * @property {string} eventType - Event type description (e.g., "issue", "pull request", "discussion")
 */

/**
 * Get the run-started message, using custom template if configured.
 * @param {RunStartedContext} ctx - Context for run-started message generation
 * @returns {string} Run-started message
 */
function getRunStartedMessage(ctx) {
  const messages = getMessages();

  // Create context with both camelCase and snake_case keys
  const templateContext = toSnakeCase(ctx);

  // Default run-started template - pirate themed! 🏴‍☠️
  const defaultMessage = "⚓ Avast! [{workflow_name}]({run_url}) be settin' sail on this {event_type}! 🏴‍☠️";

  // Use custom message if configured
  return messages?.runStarted ? renderTemplate(messages.runStarted, templateContext) : renderTemplate(defaultMessage, templateContext);
}

/**
 * @typedef {Object} RunSuccessContext
 * @property {string} workflowName - Name of the workflow
 * @property {string} runUrl - URL of the workflow run
 */

/**
 * Get the run-success message, using custom template if configured.
 * @param {RunSuccessContext} ctx - Context for run-success message generation
 * @returns {string} Run-success message
 */
function getRunSuccessMessage(ctx) {
  const messages = getMessages();

  // Create context with both camelCase and snake_case keys
  const templateContext = toSnakeCase(ctx);

  // Default run-success template - pirate themed! 🏴‍☠️
  const defaultMessage = "🎉 Yo ho ho! [{workflow_name}]({run_url}) found the treasure and completed successfully! ⚓💰";

  // Use custom message if configured
  return messages?.runSuccess ? renderTemplate(messages.runSuccess, templateContext) : renderTemplate(defaultMessage, templateContext);
}

/**
 * @typedef {Object} RunFailureContext
 * @property {string} workflowName - Name of the workflow
 * @property {string} runUrl - URL of the workflow run
 * @property {string} status - Status text (e.g., "failed", "was cancelled", "timed out")
 */

/**
 * Get the run-failure message, using custom template if configured.
 * @param {RunFailureContext} ctx - Context for run-failure message generation
 * @returns {string} Run-failure message
 */
function getRunFailureMessage(ctx) {
  const messages = getMessages();

  // Create context with both camelCase and snake_case keys
  const templateContext = toSnakeCase(ctx);

  // Default run-failure template - pirate themed! 🏴‍☠️
  const defaultMessage = "💀 Blimey! [{workflow_name}]({run_url}) {status} and walked the plank! No treasure today, matey! ☠️";

  // Use custom message if configured
  return messages?.runFailure ? renderTemplate(messages.runFailure, templateContext) : renderTemplate(defaultMessage, templateContext);
}

/**
 * @typedef {Object} DetectionFailureContext
 * @property {string} workflowName - Name of the workflow
 * @property {string} runUrl - URL of the workflow run
 */

/**
 * Get the detection-failure message, using custom template if configured.
 * @param {DetectionFailureContext} ctx - Context for detection-failure message generation
 * @returns {string} Detection-failure message
 */
function getDetectionFailureMessage(ctx) {
  const messages = getMessages();

  // Create context with both camelCase and snake_case keys
  const templateContext = toSnakeCase(ctx);

  // Default detection-failure template
  const defaultMessage = "⚠️ Security scanning failed for [{workflow_name}]({run_url}). Review the logs for details.";

  // Use custom message if configured
  return messages?.detectionFailure ? renderTemplate(messages.detectionFailure, templateContext) : renderTemplate(defaultMessage, templateContext);
}

module.exports = {
  getRunStartedMessage,
  getRunSuccessMessage,
  getRunFailureMessage,
  getDetectionFailureMessage,
};

EOF_MESSAGES_RUN_STATUS
echo "::notice::Copied: messages_run_status.cjs"
FILE_COUNT=$((FILE_COUNT + 1))

# === FILE: messages_core.cjs ===
cat > "${DESTINATION}/messages_core.cjs" << 'EOF_MESSAGES_CORE'
// @ts-check
/// <reference types="@actions/github-script" />

/**
 * Core Message Utilities Module
 *
 * This module provides shared utilities for message template processing.
 * It includes configuration parsing and template rendering functions.
 *
 * Supported placeholders:
 * - {workflow_name} - Name of the workflow
 * - {run_url} - URL to the workflow run
 * - {workflow_source} - Source specification (owner/repo/path@ref)
 * - {workflow_source_url} - GitHub URL for the workflow source
 * - {triggering_number} - Issue/PR/Discussion number that triggered this workflow
 * - {operation} - Operation name (for staged mode titles/descriptions)
 * - {event_type} - Event type description (for run-started messages)
 * - {status} - Workflow status text (for run-failure messages)
 *
 * Both camelCase and snake_case placeholder formats are supported.
 */

/**
 * @typedef {Object} SafeOutputMessages
 * @property {string} [footer] - Custom footer message template
 * @property {string} [footerInstall] - Custom installation instructions template
 * @property {string} [stagedTitle] - Custom staged mode title template
 * @property {string} [stagedDescription] - Custom staged mode description template
 * @property {string} [runStarted] - Custom workflow activation message template
 * @property {string} [runSuccess] - Custom workflow success message template
 * @property {string} [runFailure] - Custom workflow failure message template
 * @property {string} [detectionFailure] - Custom detection job failure message template
 * @property {string} [closeOlderDiscussion] - Custom message for closing older discussions as outdated
 */

/**
 * Get the safe-output messages configuration from environment variable.
 * @returns {SafeOutputMessages|null} Parsed messages config or null if not set
 */
function getMessages() {
  const messagesEnv = process.env.GH_AW_SAFE_OUTPUT_MESSAGES;
  if (!messagesEnv) {
    return null;
  }

  try {
    // Parse JSON with camelCase keys from Go struct (using json struct tags)
    return JSON.parse(messagesEnv);
  } catch (error) {
    core.warning(`Failed to parse GH_AW_SAFE_OUTPUT_MESSAGES: ${error instanceof Error ? error.message : String(error)}`);
    return null;
  }
}

/**
 * Replace placeholders in a template string with values from context.
 * Supports {key} syntax for placeholder replacement.
 * @param {string} template - Template string with {key} placeholders
 * @param {Record<string, string|number|undefined>} context - Key-value pairs for replacement
 * @returns {string} Template with placeholders replaced
 */
function renderTemplate(template, context) {
  return template.replace(/\{(\w+)\}/g, (match, key) => {
    const value = context[key];
    return value !== undefined && value !== null ? String(value) : match;
  });
}

/**
 * Convert context object keys to snake_case for template rendering
 * @param {Record<string, any>} obj - Object with camelCase keys
 * @returns {Record<string, any>} Object with snake_case keys
 */
function toSnakeCase(obj) {
  /** @type {Record<string, any>} */
  const result = {};
  for (const [key, value] of Object.entries(obj)) {
    // Convert camelCase to snake_case
    const snakeKey = key.replace(/([A-Z])/g, "_$1").toLowerCase();
    result[snakeKey] = value;
    // Also keep original key for backwards compatibility
    result[key] = value;
  }
  return result;
}

module.exports = {
  getMessages,
  renderTemplate,
  toSnakeCase,
};

EOF_MESSAGES_CORE
echo "::notice::Copied: messages_core.cjs"
FILE_COUNT=$((FILE_COUNT + 1))

# === FILE: sanitize_content_core.cjs ===
cat > "${DESTINATION}/sanitize_content_core.cjs" << 'EOF_SANITIZE_CONTENT_CORE'
// @ts-check
/**
 * Core sanitization utilities without mention filtering
 * This module provides the base sanitization functions that don't require
 * mention resolution or filtering. It's designed to be imported by both
 * sanitize_content.cjs (full version) and sanitize_incoming_text.cjs (minimal version).
 */

/**
 * Module-level set to collect redacted URL domains across sanitization calls.
 * @type {string[]}
 */
const redactedDomains = [];

/**
 * Gets the list of redacted URL domains collected during sanitization.
 * @returns {string[]} Array of redacted domain strings
 */
function getRedactedDomains() {
  return [...redactedDomains];
}

/**
 * Adds a domain to the redacted domains list
 * @param {string} domain - Domain to add
 */
function addRedactedDomain(domain) {
  redactedDomains.push(domain);
}

/**
 * Clears the list of redacted URL domains.
 * Useful for testing or resetting state between operations.
 */
function clearRedactedDomains() {
  redactedDomains.length = 0;
}

/**
 * Writes the collected redacted URL domains to a log file.
 * Only creates the file if there are redacted domains.
 * @param {string} [filePath] - Path to write the log file. Defaults to /tmp/gh-aw/redacted-urls.log
 * @returns {string|null} The file path if written, null if no domains to write
 */
function writeRedactedDomainsLog(filePath) {
  if (redactedDomains.length === 0) {
    return null;
  }

  const fs = require("fs");
  const path = require("path");
  const targetPath = filePath || "/tmp/gh-aw/redacted-urls.log";

  // Ensure directory exists
  const dir = path.dirname(targetPath);
  if (!fs.existsSync(dir)) {
    fs.mkdirSync(dir, { recursive: true });
  }

  // Write domains to file, one per line
  fs.writeFileSync(targetPath, redactedDomains.join("\n") + "\n");

  return targetPath;
}

/**
 * Extract domains from a URL and return an array of domain variations
 * @param {string} url - The URL to extract domains from
 * @returns {string[]} Array of domain variations
 */
function extractDomainsFromUrl(url) {
  if (!url || typeof url !== "string") {
    return [];
  }

  try {
    // Parse the URL
    const urlObj = new URL(url);
    const hostname = urlObj.hostname.toLowerCase();

    // Return both the exact hostname and common variations
    const domains = [hostname];

    // For github.com, add api and raw content domain variations
    if (hostname === "github.com") {
      domains.push("api.github.com");
      domains.push("raw.githubusercontent.com");
      domains.push("*.githubusercontent.com");
    }
    // For custom GitHub Enterprise domains, add api. prefix and raw content variations
    else if (!hostname.startsWith("api.")) {
      domains.push("api." + hostname);
      // For GitHub Enterprise, raw content is typically served from raw.hostname
      domains.push("raw." + hostname);
    }

    return domains;
  } catch (e) {
    // Invalid URL, return empty array
    return [];
  }
}

/**
 * Build the list of allowed domains from environment variables and GitHub context
 * @returns {string[]} Array of allowed domains
 */
function buildAllowedDomains() {
  const allowedDomainsEnv = process.env.GH_AW_ALLOWED_DOMAINS;
  const defaultAllowedDomains = ["github.com", "github.io", "githubusercontent.com", "githubassets.com", "github.dev", "codespaces.new"];

  let allowedDomains = allowedDomainsEnv
    ? allowedDomainsEnv
        .split(",")
        .map(d => d.trim())
        .filter(d => d)
    : defaultAllowedDomains;

  // Extract and add GitHub domains from GitHub context URLs
  const githubServerUrl = process.env.GITHUB_SERVER_URL;
  const githubApiUrl = process.env.GITHUB_API_URL;

  if (githubServerUrl) {
    const serverDomains = extractDomainsFromUrl(githubServerUrl);
    allowedDomains = allowedDomains.concat(serverDomains);
  }

  if (githubApiUrl) {
    const apiDomains = extractDomainsFromUrl(githubApiUrl);
    allowedDomains = allowedDomains.concat(apiDomains);
  }

  // Remove duplicates
  return [...new Set(allowedDomains)];
}

/**
 * Sanitize URL protocols - replace non-https with (redacted)
 * @param {string} s - The string to process
 * @returns {string} The string with non-https protocols redacted
 */
function sanitizeUrlProtocols(s) {
  // Match common non-https protocols
  // This regex matches: protocol://domain or protocol:path or incomplete protocol://
  // Examples: http://, ftp://, file://, data:, javascript:, mailto:, tel:, ssh://, git://
  // The regex also matches incomplete protocols like "http://" or "ftp://" without a domain
  // Note: No word boundary check to catch protocols even when preceded by word characters
  return s.replace(/((?:http|ftp|file|ssh|git):\/\/([\w.-]*)(?:[^\s]*)|(?:data|javascript|vbscript|about|mailto|tel):[^\s]+)/gi, (match, _fullMatch, domain) => {
    // Extract domain for http/ftp/file/ssh/git protocols
    if (domain) {
      const domainLower = domain.toLowerCase();
      const truncated = domainLower.length > 12 ? domainLower.substring(0, 12) + "..." : domainLower;
      if (typeof core !== "undefined" && core.info) {
        core.info(`Redacted URL: ${truncated}`);
      }
      if (typeof core !== "undefined" && core.debug) {
        core.debug(`Redacted URL (full): ${match}`);
      }
      addRedactedDomain(domainLower);
    } else {
      // For other protocols (data:, javascript:, etc.), track the protocol itself
      const protocolMatch = match.match(/^([^:]+):/);
      if (protocolMatch) {
        const protocol = protocolMatch[1] + ":";
        // Truncate the matched URL for logging (keep first 12 chars + "...")
        const truncated = match.length > 12 ? match.substring(0, 12) + "..." : match;
        if (typeof core !== "undefined" && core.info) {
          core.info(`Redacted URL: ${truncated}`);
        }
        if (typeof core !== "undefined" && core.debug) {
          core.debug(`Redacted URL (full): ${match}`);
        }
        addRedactedDomain(protocol);
      }
    }
    return "(redacted)";
  });
}

/**
 * Remove unknown domains
 * @param {string} s - The string to process
 * @param {string[]} allowed - List of allowed domains
 * @returns {string} The string with unknown domains redacted
 */
function sanitizeUrlDomains(s, allowed) {
  // Match HTTPS URLs with optional port and path
  // This regex is designed to:
  // 1. Match https:// URIs with explicit protocol
  // 2. Capture the hostname/domain
  // 3. Allow optional port (:8080)
  // 4. Allow optional path and query string (but not trailing commas/periods)
  // 5. Stop before another https:// URL in query params (using negative lookahead)
  const httpsUrlRegex = /https:\/\/([\w.-]+(?::\d+)?)(\/(?:(?!https:\/\/)[^\s,])*)?/gi;

  return s.replace(httpsUrlRegex, (match, hostnameWithPort, pathPart) => {
    // Extract just the hostname (remove port if present)
    const hostname = hostnameWithPort.split(":")[0].toLowerCase();
    pathPart = pathPart || "";

    // Check if domain is in the allowed list or is a subdomain of an allowed domain
    const isAllowed = allowed.some(allowedDomain => {
      const normalizedAllowed = allowedDomain.toLowerCase();

      // Exact match
      if (hostname === normalizedAllowed) {
        return true;
      }

      // Wildcard match (*.example.com matches subdomain.example.com)
      if (normalizedAllowed.startsWith("*.")) {
        const baseDomain = normalizedAllowed.substring(2); // Remove *.
        return hostname.endsWith("." + baseDomain) || hostname === baseDomain;
      }

      // Subdomain match (example.com matches subdomain.example.com)
      return hostname.endsWith("." + normalizedAllowed);
    });

    if (isAllowed) {
      return match; // Keep the full URL as-is
    } else {
      // Redact the domain but preserve the protocol and structure for debugging
      const truncated = hostname.length > 12 ? hostname.substring(0, 12) + "..." : hostname;
      if (typeof core !== "undefined" && core.info) {
        core.info(`Redacted URL: ${truncated}`);
      }
      if (typeof core !== "undefined" && core.debug) {
        core.debug(`Redacted URL (full): ${match}`);
      }
      addRedactedDomain(hostname);
      return "(redacted)";
    }
  });
}

/**
 * Neutralizes commands at the start of text by wrapping them in backticks
 * @param {string} s - The string to process
 * @returns {string} The string with neutralized commands
 */
function neutralizeCommands(s) {
  const commandName = process.env.GH_AW_COMMAND;
  if (!commandName) {
    return s;
  }

  // Escape special regex characters in command name
  const escapedCommand = commandName.replace(/[.*+?^${}()|[\]\\]/g, "\\$&");

  // Neutralize /command at the start of text (with optional leading whitespace)
  // Only match at the start of the string or after leading whitespace
  return s.replace(new RegExp(`^(\\s*)/(${escapedCommand})\\b`, "i"), "$1`/$2`");
}

/**
 * Neutralizes ALL @mentions by wrapping them in backticks
 * This is the core version without any filtering
 * @param {string} s - The string to process
 * @returns {string} The string with neutralized mentions
 */
function neutralizeAllMentions(s) {
  // Replace @name or @org/team outside code with `@name`
  // No filtering - all mentions are neutralized
  return s.replace(/(^|[^\w`])@([A-Za-z0-9](?:[A-Za-z0-9-]{0,37}[A-Za-z0-9])?(?:\/[A-Za-z0-9._-]+)?)/g, (m, p1, p2) => {
    // Log when a mention is escaped to help debug issues
    if (typeof core !== "undefined" && core.info) {
      core.info(`Escaped mention: @${p2} (not in allowed list)`);
    }
    return `${p1}\`@${p2}\``;
  });
}

/**
 * Removes XML comments from content
 * @param {string} s - The string to process
 * @returns {string} The string with XML comments removed
 */
function removeXmlComments(s) {
  // Remove <!-- comment --> and malformed <!--! comment --!>
  return s.replace(/<!--[\s\S]*?-->/g, "").replace(/<!--[\s\S]*?--!>/g, "");
}

/**
 * Converts XML/HTML tags to parentheses format to prevent injection
 * @param {string} s - The string to process
 * @returns {string} The string with XML tags converted to parentheses
 */
function convertXmlTags(s) {
  // Allow safe HTML tags: b, blockquote, br, code, details, em, h1–h6, hr, i, li, ol, p, pre, strong, sub, summary, sup, table, tbody, td, th, thead, tr, ul
  const allowedTags = ["b", "blockquote", "br", "code", "details", "em", "h1", "h2", "h3", "h4", "h5", "h6", "hr", "i", "li", "ol", "p", "pre", "strong", "sub", "summary", "sup", "table", "tbody", "td", "th", "thead", "tr", "ul"];

  // First, process CDATA sections specially - convert tags inside them and the CDATA markers
  s = s.replace(/<!\[CDATA\[([\s\S]*?)\]\]>/g, (match, content) => {
    // Convert tags inside CDATA content
    const convertedContent = content.replace(/<(\/?[A-Za-z][A-Za-z0-9]*(?:[^>]*?))>/g, "($1)");
    // Return with CDATA markers also converted to parentheses
    return `(![CDATA[${convertedContent}]])`;
  });

  // Convert opening tags: <tag> or <tag attr="value"> to (tag) or (tag attr="value")
  // Convert closing tags: </tag> to (/tag)
  // Convert self-closing tags: <tag/> or <tag /> to (tag/) or (tag /)
  // But preserve allowed safe tags
  return s.replace(/<(\/?[A-Za-z!][^>]*?)>/g, (match, tagContent) => {
    // Extract tag name from the content (handle closing tags and attributes)
    const tagNameMatch = tagContent.match(/^\/?\s*([A-Za-z][A-Za-z0-9]*)/);
    if (tagNameMatch) {
      const tagName = tagNameMatch[1].toLowerCase();
      if (allowedTags.includes(tagName)) {
        return match; // Preserve allowed tags
      }
    }
    return `(${tagContent})`; // Convert other tags to parentheses
  });
}

/**
 * Neutralizes bot trigger phrases by wrapping them in backticks
 * @param {string} s - The string to process
 * @returns {string} The string with neutralized bot triggers
 */
function neutralizeBotTriggers(s) {
  // Neutralize common bot trigger phrases like "fixes #123", "closes #asdfs", etc.
  return s.replace(/\b(fixes?|closes?|resolves?|fix|close|resolve)\s+#(\w+)/gi, (match, action, ref) => `\`${action} #${ref}\``);
}

/**
 * Apply truncation limits to content
 * @param {string} content - The content to truncate
 * @param {number} [maxLength] - Maximum length of content (default: 524288)
 * @returns {string} The truncated content
 */
function applyTruncation(content, maxLength) {
  maxLength = maxLength || 524288;
  const lines = content.split("\n");
  const maxLines = 65000;

  // If content has too many lines, truncate by lines (primary limit)
  if (lines.length > maxLines) {
    const truncationMsg = "\n[Content truncated due to line count]";
    const truncatedLines = lines.slice(0, maxLines).join("\n") + truncationMsg;

    // If still too long after line truncation, shorten but keep the line count message
    if (truncatedLines.length > maxLength) {
      return truncatedLines.substring(0, maxLength - truncationMsg.length) + truncationMsg;
    } else {
      return truncatedLines;
    }
  } else if (content.length > maxLength) {
    return content.substring(0, maxLength) + "\n[Content truncated due to length]";
  }

  return content;
}

/**
 * Core sanitization function without mention filtering
 * @param {string} content - The content to sanitize
 * @param {number} [maxLength] - Maximum length of content (default: 524288)
 * @returns {string} The sanitized content
 */
function sanitizeContentCore(content, maxLength) {
  if (!content || typeof content !== "string") {
    return "";
  }

  // Build list of allowed domains from environment and GitHub context
  const allowedDomains = buildAllowedDomains();

  let sanitized = content;

  // Remove ANSI escape sequences and control characters early
  // This must happen before mention neutralization to avoid creating bare mentions
  // when control characters are removed between @ and username
  sanitized = sanitized.replace(/\x1b\[[0-9;]*[mGKH]/g, "");
  // Remove control characters except newlines (\n), tabs (\t), and carriage returns (\r)
  sanitized = sanitized.replace(/[\x00-\x08\x0B\x0C\x0E-\x1F\x7F]/g, "");

  // Neutralize commands at the start of text (e.g., /bot-name)
  sanitized = neutralizeCommands(sanitized);

  // Neutralize ALL @mentions (no filtering in core version)
  sanitized = neutralizeAllMentions(sanitized);

  // Remove XML comments first
  sanitized = removeXmlComments(sanitized);

  // Convert XML tags to parentheses format to prevent injection
  sanitized = convertXmlTags(sanitized);

  // URI filtering - replace non-https protocols with "(redacted)"
  sanitized = sanitizeUrlProtocols(sanitized);

  // Domain filtering for HTTPS URIs
  sanitized = sanitizeUrlDomains(sanitized, allowedDomains);

  // Apply truncation limits
  sanitized = applyTruncation(sanitized, maxLength);

  // Neutralize common bot trigger phrases
  sanitized = neutralizeBotTriggers(sanitized);

  // Trim excessive whitespace
  return sanitized.trim();
}

module.exports = {
  sanitizeContentCore,
  getRedactedDomains,
  addRedactedDomain,
  clearRedactedDomains,
  writeRedactedDomainsLog,
  extractDomainsFromUrl,
  buildAllowedDomains,
  sanitizeUrlProtocols,
  sanitizeUrlDomains,
  neutralizeCommands,
  removeXmlComments,
  convertXmlTags,
  neutralizeBotTriggers,
  applyTruncation,
};

EOF_SANITIZE_CONTENT_CORE
echo "::notice::Copied: sanitize_content_core.cjs"
FILE_COUNT=$((FILE_COUNT + 1))

# Set output
echo "files-copied=${FILE_COUNT}" >> "${GITHUB_OUTPUT:-/dev/null}"
echo "::notice::✓ Successfully copied ${FILE_COUNT} files"
