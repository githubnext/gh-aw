// @ts-check
/// <reference types="@actions/github-script" />

const { getErrorMessage } = require("./error_helpers.cjs");
const { resolveTargetRepoConfig, resolveAndValidateRepo } = require("./repo_helpers.cjs");
const { getBaseBranch } = require("./get_base_branch.cjs");
const { isStagedMode } = require("./safe_output_helpers.cjs");
const { generateStagedPreview } = require("./staged_preview.cjs");

const fs = require("fs");
const path = require("path");

/**
 * Module-level state — populated by handleMessage(), read by the exported getters below.
 * Using module-level variables (rather than closure-only state) allows the handler
 * manager to read final output values after all messages have been processed.
 * @type {Array<{number: string, url: string, success: boolean, error?: string}>}
 */
let _allResults = [];

/**
 * Handler factory for create-agent-session safe output.
 *
 * Replaces the standalone create_agent_session step. This function is called once by the
 * safe output handler manager with the handler's configuration. It returns a message
 * processor function that is invoked for each create_agent_session message in the agent output.
 *
 * @param {Object} config - Handler configuration from GH_AW_SAFE_OUTPUTS_HANDLER_CONFIG
 * @returns {Promise<Function>} Message processor function
 */
async function main(config = {}) {
  // Reset module-level state for this run
  _allResults = [];

  // Parse configuration
  const configuredBaseBranch = config.base ? String(config.base).trim() : null;
  const isStaged = isStagedMode(config);
  const { defaultTargetRepo, allowedRepos } = resolveTargetRepoConfig(config);

  if (configuredBaseBranch) core.info(`Configured base branch: ${configuredBaseBranch}`);
  core.info(`Default target repo: ${defaultTargetRepo}`);
  if (allowedRepos.size > 0) core.info(`Allowed repos: ${[...allowedRepos].join(", ")}`);

  /**
   * Process a single create_agent_session message.
   * @param {Object} message - The agent output message
   * @returns {Promise<{success: boolean, number?: string, url?: string, error?: string, skipped?: boolean}>}
   */
  return async function handleMessage(message) {
    const taskDescription = message.body;

    if (!taskDescription || taskDescription.trim() === "") {
      core.warning("Agent task description is empty, skipping");
      _allResults.push({ number: "", url: "", success: false, error: "Empty task description" });
      return { success: false, error: "Empty task description" };
    }

    // Resolve and validate target repository for this message
    const repoResult = resolveAndValidateRepo(message, defaultTargetRepo, allowedRepos, "agent session");
    if (!repoResult.success) {
      const errorMsg = `E004: ${repoResult.error}`;
      core.error(errorMsg);
      _allResults.push({ number: "", url: "", success: false, error: repoResult.error });
      return { success: false, error: repoResult.error };
    }
    const { repo: effectiveRepo, repoParts } = repoResult;

    // Resolve base branch: use custom config if set, otherwise resolve dynamically.
    // Dynamic resolution is needed for issue_comment events on PRs where the base branch
    // is not available in GitHub Actions expressions and requires an API call.
    const baseBranch = configuredBaseBranch || (await getBaseBranch(repoParts));

    if (isStaged) {
      await generateStagedPreview({
        title: "Create Agent Session",
        description: "The following agent sessions would be created if staged mode was disabled:",
        items: [message],
        renderItem: item => {
          const parts = [];
          parts.push(`**Description:**\n${item.body}`);
          parts.push(`**Base Branch:** ${baseBranch}`);
          parts.push(`**Target Repository:** ${effectiveRepo}`);
          return parts.join("\n\n") + "\n\n";
        },
      });
      return { success: true, skipped: true };
    }

    try {
      // Write task description to a temporary file
      const tmpDir = "/tmp/gh-aw";
      if (!fs.existsSync(tmpDir)) {
        fs.mkdirSync(tmpDir, { recursive: true });
      }

      const taskIndex = _allResults.length + 1;
      const taskFile = path.join(tmpDir, `agent-task-description-${taskIndex}.md`);
      fs.writeFileSync(taskFile, taskDescription, "utf8");
      core.info(`Task ${taskIndex}: Task description written to ${taskFile}`);

      // Build gh agent-task create command
      const ghArgs = ["agent-task", "create", "--from-file", taskFile, "--base", baseBranch];

      const contextRepo = `${context.repo.owner}/${context.repo.repo}`;
      if (effectiveRepo !== contextRepo) {
        ghArgs.push("--repo", effectiveRepo);
      }

      core.info(`Task ${taskIndex}: Creating agent session with command: gh ${ghArgs.join(" ")}`);

      // Determine token: prefer per-handler token, fall back to step-level token
      const ghToken = config["github-token"] || process.env.GH_AW_AGENT_SESSION_TOKEN || process.env.GITHUB_TOKEN || "";

      // Execute gh agent-task create command
      let taskOutput;
      try {
        taskOutput = await exec.getExecOutput("gh", ghArgs, {
          silent: false,
          ignoreReturnCode: false,
          env: {
            ...process.env,
            GH_TOKEN: ghToken,
          },
        });
      } catch (execError) {
        const errorMessage = execError instanceof Error ? execError.message : String(execError);

        // Check for authentication/permission errors
        if (errorMessage.includes("authentication") || errorMessage.includes("permission") || errorMessage.includes("forbidden") || errorMessage.includes("401") || errorMessage.includes("403")) {
          core.error(`Task ${taskIndex}: Failed to create agent session due to authentication/permission error.`);
          core.error(`The default GITHUB_TOKEN may not have permission to create agent sessions.`);
          core.error(`Configure a Personal Access Token (PAT) using the handler's github-token setting or GH_AW_AGENT_SESSION_TOKEN.`);
          core.error(`See documentation: https://github.github.com/gh-aw/reference/safe-outputs/#agent-task-creation-create-agent-session`);
        } else {
          core.error(`Task ${taskIndex}: Failed to create agent session: ${errorMessage}`);
        }
        _allResults.push({ number: "", url: "", success: false, error: errorMessage });
        return { success: false, error: errorMessage };
      }

      // Parse the output to extract task number and URL.
      // Expected output format from gh agent-task create is typically:
      // https://github.com/owner/repo/issues/123
      const output = taskOutput.stdout.trim();
      core.info(`Task ${taskIndex}: Agent task created: ${output}`);

      // Extract task number from URL
      const urlMatch = output.match(/github\.com\/[^/]+\/[^/]+\/issues\/(\d+)/);
      if (urlMatch) {
        const taskNumber = urlMatch[1];
        core.info(`✅ Successfully created agent session #${taskNumber}`);
        _allResults.push({ number: taskNumber, url: output, success: true });
        return { success: true, number: taskNumber, url: output };
      } else {
        core.warning(`Task ${taskIndex}: Could not parse task number from output: ${output}`);
        _allResults.push({ number: "", url: output, success: true });
        return { success: true, number: "", url: output };
      }
    } catch (error) {
      const errorMessage = getErrorMessage(error);
      core.error(`Error creating agent session: ${errorMessage}`);
      _allResults.push({ number: "", url: "", success: false, error: errorMessage });
      return { success: false, error: errorMessage };
    }
  };
}

/**
 * Returns the session_number output: the number of the first successfully created session.
 * @returns {string}
 */
function getCreateAgentSessionNumber() {
  const first = _allResults.find(r => r.success && r.number);
  return first ? first.number : "";
}

/**
 * Returns the session_url output: the URL of the first successfully created session.
 * @returns {string}
 */
function getCreateAgentSessionUrl() {
  const first = _allResults.find(r => r.success && r.url);
  return first ? first.url : "";
}

/**
 * Writes a step summary for agent session creation results.
 * Called by the handler manager after all messages have been processed.
 * @returns {Promise<void>}
 */
async function writeCreateAgentSessionSummary() {
  const successResults = _allResults.filter(r => r.success);
  const failedResults = _allResults.filter(r => !r.success);

  if (_allResults.length === 0) return;

  let summaryContent = "## Agent Sessions\n\n";

  if (successResults.length > 0) {
    summaryContent += `✅ Successfully created ${successResults.length} agent session(s):\n\n`;
    summaryContent += successResults
      .map((r, i) => {
        if (r.url && r.number) {
          return `- [#${r.number}](${r.url})`;
        } else if (r.url) {
          return `- [Session ${i + 1}](${r.url})`;
        }
        return `- Session ${i + 1}`;
      })
      .join("\n");
    summaryContent += "\n\n";
  }

  if (failedResults.length > 0) {
    summaryContent += `❌ Failed to create ${failedResults.length} agent session(s):\n\n`;
    summaryContent += failedResults.map(r => `- ${r.error || "Unknown error"}`).join("\n");
    summaryContent += "\n\n";
  }

  try {
    await core.summary.addRaw(summaryContent).write();
  } catch (error) {
    core.warning(`Failed to write agent session summary: ${getErrorMessage(error)}`);
  }
}

module.exports = { main, getCreateAgentSessionNumber, getCreateAgentSessionUrl, writeCreateAgentSessionSummary };
