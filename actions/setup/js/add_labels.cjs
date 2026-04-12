// @ts-check
/// <reference types="@actions/github-script" />

/**
 * @typedef {import('./types/handler-factory').HandlerFactoryFunction} HandlerFactoryFunction
 * @typedef {import('./types/handler-factory').ResolvedTemporaryIds} ResolvedTemporaryIds
 * @typedef {import('./types/handler-factory').HandlerResult} HandlerResult
 */

/**
 * @typedef {{ item_number?: number|string, labels?: string[], repo?: string }} AddLabelsMessage
 */

/** @type {string} Safe output type handled by this module */
const HANDLER_TYPE = "add_labels";

const { validateLabels } = require("./safe_output_validator.cjs");
const { getErrorMessage } = require("./error_helpers.cjs");
const { resolveTargetRepoConfig, resolveAndValidateRepo } = require("./repo_helpers.cjs");
const { tryEnforceArrayLimit } = require("./limit_enforcement_helpers.cjs");
const { logStagedPreviewInfo } = require("./staged_preview.cjs");
const { createAuthenticatedGitHubClient } = require("./handler_auth.cjs");
const { resolveRepoIssueTarget, loadTemporaryIdMapFromResolved } = require("./temporary_id.cjs");
const { MAX_LABELS } = require("./constants.cjs");
const { createCountGatedHandler } = require("./handler_scaffold.cjs");

/**
 * Main handler factory for add_labels
 * Uses shared count-gated scaffold for max-limit enforcement.
 * @type {HandlerFactoryFunction}
 */
const main = createCountGatedHandler({
  handlerType: HANDLER_TYPE,
  setup: async (config, maxCount, isStaged) => {
    const { allowed: allowedLabels = [], blocked: blockedPatterns = [] } = config;
    const { defaultTargetRepo, allowedRepos } = resolveTargetRepoConfig(config);
    const githubClient = await createAuthenticatedGitHubClient(config);

    core.info(`Add labels configuration: max=${maxCount}`);
    if (allowedLabels.length > 0) core.info(`Allowed labels: ${allowedLabels.join(", ")}`);
    if (blockedPatterns.length > 0) core.info(`Blocked patterns: ${blockedPatterns.join(", ")}`);
    core.info(`Default target repo: ${defaultTargetRepo}`);
    if (allowedRepos.size > 0) core.info(`Allowed repos: ${[...allowedRepos].join(", ")}`);

    /**
     * Message handler function that processes a single add_labels message
     * @param {AddLabelsMessage} message - The add_labels message to process
     * @param {ResolvedTemporaryIds} resolvedTemporaryIds - Map of temporary IDs to {repo, number}
     * @returns {Promise<HandlerResult>} Result with success/error status
     */
    return async function handleAddLabels(message, resolvedTemporaryIds) {
      // Resolve and validate target repository
      const repoResult = resolveAndValidateRepo(message, defaultTargetRepo, allowedRepos, "label");
      if (!repoResult.success) {
        core.warning(`Skipping add_labels: ${repoResult.error}`);
        return {
          success: false,
          error: repoResult.error,
        };
      }
      const { repo: itemRepo, repoParts } = repoResult;
      core.info(`Target repository: ${itemRepo}`);

      // Determine target issue/PR number
      let itemNumber;
      if (message.item_number !== undefined) {
        // Resolve temporary IDs if present
        const tempIdMap = loadTemporaryIdMapFromResolved(resolvedTemporaryIds);
        const resolvedTarget = resolveRepoIssueTarget(message.item_number, tempIdMap, repoParts.owner, repoParts.repo);

        // Check if this is an unresolved temporary ID
        if (resolvedTarget.wasTemporaryId && !resolvedTarget.resolved) {
          core.info(`Deferring add_labels: unresolved temporary ID (${message.item_number})`);
          return {
            success: false,
            deferred: true,
            error: resolvedTarget.errorMessage || `Unresolved temporary ID: ${message.item_number}`,
          };
        }

        // Check for other resolution errors
        if (resolvedTarget.errorMessage || !resolvedTarget.resolved) {
          const error = `Invalid item number: ${message.item_number}`;
          core.warning(error);
          return { success: false, error };
        }

        itemNumber = resolvedTarget.resolved.number;
      } else {
        itemNumber = context.payload?.issue?.number ?? context.payload?.pull_request?.number;
      }

      if (!itemNumber || isNaN(itemNumber)) {
        const error = "No issue/PR number available";
        core.warning(error);
        return { success: false, error };
      }

      const contextType = context.payload?.pull_request ? "pull request" : "issue";
      const requestedLabels = message.labels ?? [];
      core.info(`Requested labels: ${JSON.stringify(requestedLabels)}`);

      // If no labels provided, return a helpful message with allowed labels if configured
      if (requestedLabels.length === 0) {
        const labelSource = allowedLabels.length > 0 ? `the allowed list: ${JSON.stringify(allowedLabels)}` : "the repository's available labels";
        const error = `No labels provided. Please provide at least one label from ${labelSource}`;
        core.info(error);
        return { success: false, error };
      }

      // Enforce max limits on labels before validation
      const limitResult = tryEnforceArrayLimit(requestedLabels, MAX_LABELS, "labels");
      if (!limitResult.success) {
        core.warning(`Label limit exceeded: ${limitResult.error}`);
        return { success: false, error: limitResult.error };
      }

      // Use validation helper to sanitize and validate labels
      const labelsResult = validateLabels(requestedLabels, allowedLabels, maxCount, blockedPatterns);
      if (!labelsResult.valid) {
        // If no valid labels, log info and return gracefully
        if (labelsResult.error?.includes("No valid labels")) {
          core.info("No labels to add");
          return {
            success: true,
            number: itemNumber,
            labelsAdded: [],
            message: "No valid labels found",
          };
        }
        // For other validation errors, return error
        core.warning(`Label validation failed: ${labelsResult.error}`);
        return {
          success: false,
          error: labelsResult.error ?? "Invalid labels",
        };
      }

      const uniqueLabels = labelsResult.value ?? [];

      // Early return if no labels after validation
      if (uniqueLabels.length === 0) {
        core.info("No labels to add");
        return {
          success: true,
          number: itemNumber,
          labelsAdded: [],
          message: "No labels to add",
        };
      }

      core.info(`Adding ${uniqueLabels.length} labels to ${contextType} #${itemNumber} in ${itemRepo}: ${JSON.stringify(uniqueLabels)}`);

      // If in staged mode, preview the labels without adding them
      if (isStaged) {
        logStagedPreviewInfo(`Would add ${uniqueLabels.length} labels to ${contextType} #${itemNumber} in ${itemRepo}`);
        return {
          success: true,
          staged: true,
          previewInfo: {
            number: itemNumber,
            repo: itemRepo,
            labels: uniqueLabels,
            contextType,
          },
        };
      }

      try {
        await githubClient.rest.issues.addLabels({
          owner: repoParts.owner,
          repo: repoParts.repo,
          issue_number: itemNumber,
          labels: uniqueLabels,
        });

        core.info(`Successfully added ${uniqueLabels.length} labels to ${contextType} #${itemNumber} in ${itemRepo}`);
        return {
          success: true,
          number: itemNumber,
          labelsAdded: uniqueLabels,
          contextType,
        };
      } catch (error) {
        const errorMessage = getErrorMessage(error);
        core.error(`Failed to add labels: ${errorMessage}`);
        return { success: false, error: errorMessage };
      }
    };
  },
});

module.exports = { main };
