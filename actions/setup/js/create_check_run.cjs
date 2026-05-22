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
const { sanitizeContent } = require("./sanitize_content.cjs");

/** @type {string} Safe output type handled by this module */
const HANDLER_TYPE = "create_check_run";

/** @type {Set<string>} Valid conclusion values for GitHub Check Runs */
const VALID_CONCLUSIONS = new Set(["success", "failure", "neutral", "cancelled", "skipped", "timed_out", "action_required"]);

/** @type {number} Maximum length for summary and text fields (GitHub API limit) */
const MAX_CONTENT_LENGTH = 65535;

/** @type {number} Maximum length for the title field */
const MAX_TITLE_LENGTH = 256;

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

  // Optional config-level output defaults (sanitized at startup so we pay the cost once)
  const configOutputTitle = config.output_title ? sanitizeContent(String(config.output_title), MAX_TITLE_LENGTH) : "";
  const configOutputSummary = config.output_summary ? sanitizeContent(String(config.output_summary), MAX_CONTENT_LENGTH) : "";

  // Resolve the check run name: config > workflow name env var > fallback.
  // Auto-deduplicate: if the resolved name equals the workflow name, GitHub's UI
  // may collapse the programmatic check run into the workflow's own check suite
  // entry, hiding it in compact/mobile views. Appending "(Result)" ensures a
  // distinct name so the check run remains visible on all GitHub UI surfaces.
  const workflowName = process.env.GITHUB_WORKFLOW || "";
  let defaultName = configuredName || workflowName || "Agent Check";
  if (defaultName === workflowName && workflowName) {
    defaultName = `${defaultName} (Result)`;
  }

  core.info(`Create check run configuration: name="${defaultName}", max=${maxCount}`);
  if (configOutputTitle) core.info(`Config output.title fallback set (${configOutputTitle.length} chars)`);
  if (configOutputSummary) core.info(`Config output.summary fallback set (${configOutputSummary.length} chars)`);

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

    // Resolve title: agent value (sanitized) > config fallback > error
    const rawTitle = (message.title || "").trim();
    const resolvedTitle = rawTitle ? sanitizeContent(rawTitle, MAX_TITLE_LENGTH) : configOutputTitle;
    if (!resolvedTitle) {
      const msg = configOutputTitle ? "create_check_run: title resolved to empty after sanitization" : "create_check_run requires a non-empty 'title' field (or config output.title fallback)";
      core.error(msg);
      return { success: false, error: msg };
    }

    // Resolve summary: agent value (sanitized + truncated) > config fallback > error
    const rawSummary = (message.summary || "").trim();
    const resolvedSummary = rawSummary ? sanitizeContent(rawSummary, MAX_CONTENT_LENGTH) : configOutputSummary;
    if (!resolvedSummary) {
      const msg = configOutputSummary ? "create_check_run: summary resolved to empty after sanitization" : "create_check_run requires a non-empty 'summary' field (or config output.summary fallback)";
      core.error(msg);
      return { success: false, error: msg };
    }

    // Sanitize optional text field
    const rawText = (message.text || "").trim();
    const resolvedText = rawText ? sanitizeContent(rawText, MAX_CONTENT_LENGTH) : "";

    const owner = context.repo.owner;
    const repo = context.repo.repo;

    // For pull_request events, GITHUB_SHA is the ephemeral merge commit SHA which is
    // not visible in the PR checks UI or the GitHub mobile app. Use the actual PR head
    // SHA from the event payload instead so the check run appears on the PR.
    const prHeadSha = context.payload?.pull_request?.head?.sha;
    const headSha = prHeadSha || process.env.GITHUB_SHA || context.sha;

    if (!headSha) {
      const msg = "create_check_run: cannot determine commit SHA for check run";
      core.error(msg);
      return { success: false, error: msg };
    }

    if (prHeadSha) {
      core.info(`Using PR head SHA ${prHeadSha} (pull_request event)`);
    }

    const checkRunName = defaultName;

    core.info(`Creating check run "${checkRunName}" on ${owner}/${repo}@${headSha} with conclusion=${conclusion}`);

    // If in staged mode, preview without executing
    if (isStaged) {
      logStagedPreviewInfo(`Would create check run "${checkRunName}" with conclusion=${conclusion}, title="${resolvedTitle}"`);
      processedCount++;
      return {
        success: true,
        staged: true,
        previewInfo: {
          name: checkRunName,
          conclusion,
          title: resolvedTitle,
        },
      };
    }

    try {
      const output = {
        title: resolvedTitle,
        summary: resolvedSummary,
        ...(resolvedText ? { text: resolvedText } : {}),
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
        RATE_LIMIT_RETRY_CONFIG
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
