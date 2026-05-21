// @ts-check
/// <reference types="@actions/github-script" />

/**
 * @typedef {import('./types/handler-factory').HandlerFactoryFunction} HandlerFactoryFunction
 */

const { getErrorMessage } = require("./error_helpers.cjs");
const { logStagedPreviewInfo } = require("./staged_preview.cjs");
const { isStagedMode } = require("./safe_output_helpers.cjs");
const { createAuthenticatedGitHubClient } = require("./handler_auth.cjs");
const { withRetry, RATE_LIMIT_RETRY_CONFIG } = require("./error_recovery.cjs");

/** @type {string} Safe output type handled by this module */
const HANDLER_TYPE = "create_check_run";

/** @type {Set<string>} Valid conclusion values for GitHub Check Runs */
const VALID_CONCLUSIONS = new Set(["success", "failure", "neutral", "cancelled", "skipped", "timed_out", "action_required"]);

/** @type {number} Maximum length for summary and text fields (GitHub API limit) */
const MAX_CONTENT_LENGTH = 65535;

/**
 * Main handler factory for create_check_run
 * Returns a message handler function that processes individual create_check_run messages
 * @type {HandlerFactoryFunction}
 */
async function main(config = {}) {
  // Extract configuration
  const configuredName = config.name || "";
  const maxCount = config.max != null ? Number(config.max) : 1;
  const githubClient = await createAuthenticatedGitHubClient(config);
  const isStaged = isStagedMode(config);

  // Resolve the check run name: config > workflow name env var > fallback
  const defaultName = configuredName || process.env.GITHUB_WORKFLOW || "Agent Check";

  core.info(`Create check run configuration: name="${defaultName}", max=${maxCount}`);

  // Track how many check runs we've created for max limit enforcement
  let processedCount = 0;

  /**
   * Message handler function that processes a single create_check_run message
   * @param {Object} message - The create_check_run message to process
   * @param {Object} _resolvedTemporaryIds - Map of temporary IDs (unused for check runs)
   * @returns {Promise<Object>} Result with success/error status
   */
  return async function handleCreateCheckRun(message, _resolvedTemporaryIds) {
    // Check if we've hit the max limit
    if (processedCount >= maxCount) {
      core.warning(`Skipping create_check_run: max count of ${maxCount} reached`);
      return {
        success: false,
        error: `Max count of ${maxCount} reached`,
      };
    }

    // Validate required fields
    const conclusion = message.conclusion;
    if (!conclusion) {
      const msg = "create_check_run requires a 'conclusion' field";
      core.error(msg);
      return { success: false, error: msg };
    }
    if (!VALID_CONCLUSIONS.has(conclusion)) {
      const msg = `create_check_run: invalid conclusion '${conclusion}'. Must be one of: ${[...VALID_CONCLUSIONS].join(", ")}`;
      core.error(msg);
      return { success: false, error: msg };
    }

    const title = (message.title || "").trim();
    if (!title) {
      const msg = "create_check_run requires a non-empty 'title' field";
      core.error(msg);
      return { success: false, error: msg };
    }

    const summary = (message.summary || "").trim();
    if (!summary) {
      const msg = "create_check_run requires a non-empty 'summary' field";
      core.error(msg);
      return { success: false, error: msg };
    }

    // Truncate content if needed
    const truncatedSummary = summary.length > MAX_CONTENT_LENGTH ? summary.slice(0, MAX_CONTENT_LENGTH) : summary;
    const rawText = (message.text || "").trim();
    const truncatedText = rawText.length > MAX_CONTENT_LENGTH ? rawText.slice(0, MAX_CONTENT_LENGTH) : rawText;

    if (summary.length > MAX_CONTENT_LENGTH) {
      core.warning(`create_check_run: summary truncated from ${summary.length} to ${MAX_CONTENT_LENGTH} characters`);
    }
    if (rawText.length > MAX_CONTENT_LENGTH) {
      core.warning(`create_check_run: text truncated from ${rawText.length} to ${MAX_CONTENT_LENGTH} characters`);
    }

    const owner = context.repo.owner;
    const repo = context.repo.repo;
    const headSha = process.env.GITHUB_SHA || context.sha;

    if (!headSha) {
      const msg = "create_check_run: GITHUB_SHA is not set, cannot determine commit SHA for check run";
      core.error(msg);
      return { success: false, error: msg };
    }

    const checkRunName = defaultName;

    core.info(`Creating check run "${checkRunName}" on ${owner}/${repo}@${headSha} with conclusion=${conclusion}`);

    // If in staged mode, preview without executing
    if (isStaged) {
      logStagedPreviewInfo(`Would create check run "${checkRunName}" with conclusion=${conclusion}, title="${title}"`);
      processedCount++;
      return {
        success: true,
        staged: true,
        previewInfo: {
          name: checkRunName,
          conclusion,
          title,
        },
      };
    }

    try {
      const output = {
        title,
        summary: truncatedSummary,
        ...(truncatedText ? { text: truncatedText } : {}),
      };

      const response = await withRetry(
        () =>
          githubClient.rest.checks.create({
            owner,
            repo,
            name: checkRunName,
            head_sha: headSha,
            status: "completed",
            conclusion,
            completed_at: new Date().toISOString(),
            output,
          }),
        RATE_LIMIT_RETRY_CONFIG,
      );

      const checkRunId = response.data.id;
      const checkRunUrl = response.data.html_url;

      core.info(`✓ Created check run "${checkRunName}" #${checkRunId}: ${checkRunUrl}`);
      processedCount++;

      return {
        success: true,
        check_run_id: checkRunId,
        check_run_url: checkRunUrl,
        conclusion,
        name: checkRunName,
      };
    } catch (error) {
      const errorMessage = getErrorMessage(error);
      core.error(`Failed to create check run "${checkRunName}": ${errorMessage}`);
      return {
        success: false,
        error: errorMessage,
      };
    }
  };
}

module.exports = { main };
