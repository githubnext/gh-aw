// @ts-check
/// <reference types="@actions/github-script" />

/**
 * @typedef {import('./types/handler-factory').HandlerFactoryFunction} HandlerFactoryFunction
 * @typedef {import('./types/handler-factory').ResolvedTemporaryIds} ResolvedTemporaryIds
 * @typedef {import('./types/handler-factory').HandlerResult} HandlerResult
 */

/**
 * @typedef {{
 *   item_number?: number|string,
 *   issue_number?: number|string,
 *   pr_number?: number|string,
 *   pull_number?: number|string,
 *   labels?: Array<string|{name: string, rationale?: string, confidence?: "LOW"|"MEDIUM"|"HIGH", suggest?: boolean}>,
 *   repo?: string
 * }} AddLabelsMessage
 */

/** @type {string} Safe output type handled by this module */
const HANDLER_TYPE = "add_labels";

const { validateLabels } = require("./safe_output_validator.cjs");
const { getErrorMessage } = require("./error_helpers.cjs");
const { resolveTargetRepoConfig, resolveAndValidateRepo } = require("./repo_helpers.cjs");
const { tryEnforceArrayLimit } = require("./limit_enforcement_helpers.cjs");
const { logStagedPreviewInfo } = require("./staged_preview.cjs");
const { createAuthenticatedGitHubClient } = require("./handler_auth.cjs");
const { resolveSafeOutputIssueTarget } = require("./temporary_id.cjs");
const { attachExecutionState, fetchIssueState, normalizeLabelNames } = require("./safe_output_execution_metadata.cjs");
const { MAX_LABELS } = require("./constants.cjs");
const { createCountGatedHandler } = require("./handler_scaffold.cjs");
const { withRetry, RATE_LIMIT_RETRY_CONFIG } = require("./error_recovery.cjs");
const { resolveInvocationContext } = require("./invocation_context_helpers.cjs");
const { normalizeIssueIntentLabelInputs } = require("./issue_intents.cjs");

/**
 * Main handler factory for add_labels
 * Uses shared count-gated scaffold for max-limit enforcement.
 * @type {HandlerFactoryFunction}
 */
const main = createCountGatedHandler({
  handlerType: HANDLER_TYPE,
  setup: async (config, maxCount, isStaged) => {
    const { allowed: allowedLabels = [], blocked: blockedPatterns = [] } = config;
    const issueIntentEnabled = config.issue_intent === true;
    const requiredLabels = Array.isArray(config.required_labels) ? config.required_labels : [];
    const requiredTitlePrefix = config.required_title_prefix || "";
    const { defaultTargetRepo, allowedRepos } = resolveTargetRepoConfig(config);
    const githubClient = await createAuthenticatedGitHubClient(config);

    core.info(`Add labels configuration: max=${maxCount}`);
    if (allowedLabels.length > 0) core.info(`Allowed labels: ${allowedLabels.join(", ")}`);
    if (blockedPatterns.length > 0) core.info(`Blocked patterns: ${blockedPatterns.join(", ")}`);
    if (requiredLabels.length > 0) core.info(`Required labels (all): ${requiredLabels.join(", ")}`);
    if (requiredTitlePrefix) core.info(`Required title prefix: ${requiredTitlePrefix}`);
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
      // Accept common aliases: issue_number, pr_number, and pull_number are normalised to item_number
      const targetResult = resolveSafeOutputIssueTarget({ message, resolvedTemporaryIds, repoParts, handlerType: HANDLER_TYPE });
      if (!targetResult.success) return targetResult;
      const effectiveContext = resolveInvocationContext(context);
      const itemNumber = targetResult.number ?? effectiveContext.eventPayload?.issue?.number ?? effectiveContext.eventPayload?.pull_request?.number;

      if (!itemNumber || Number.isNaN(Number(itemNumber))) {
        const error = "No issue/PR number available";
        core.warning(error);
        return { success: false, error };
      }

      const contextType = effectiveContext.eventPayload?.pull_request ? "pull request" : "issue";
      const requestedLabels = message.labels ?? [];
      core.info(`Requested labels: ${JSON.stringify(requestedLabels)}`);
      /** @type {Map<string, {name: string, rationale?: string, confidence?: "LOW"|"MEDIUM"|"HIGH", suggest?: boolean}>} */
      const requestedLabelSpecByLowerName = new Map();
      let requestedLabelNames;
      try {
        const requestedLabelInputs = normalizeIssueIntentLabelInputs(requestedLabels);
        requestedLabelNames = requestedLabelInputs.map(label => {
          if (typeof label === "string") {
            return label;
          }
          const key = label.name.toLowerCase();
          const existing = requestedLabelSpecByLowerName.get(key);
          const newHasMetadata = Boolean(label.rationale || label.confidence || label.suggest);
          const existingHasMetadata = existing && Boolean(existing.rationale || existing.confidence || existing.suggest);
          if (!existing || (!existingHasMetadata && newHasMetadata)) {
            requestedLabelSpecByLowerName.set(key, label);
          }
          return label.name;
        });
      } catch (error) {
        const errorMessage = getErrorMessage(error);
        core.warning(`Invalid add_labels payload: ${errorMessage}`);
        return { success: false, error: errorMessage };
      }

      // Apply required-labels and required-title-prefix filters
      if (requiredLabels.length > 0 || requiredTitlePrefix) {
        const { data: item } = await githubClient.rest.issues.get({
          owner: repoParts.owner,
          repo: repoParts.repo,
          issue_number: itemNumber,
        });
        if (requiredLabels.length > 0) {
          const itemLabels = (item.labels || []).map(/** @param {any} l */ l => (typeof l === "string" ? l : l.name || ""));
          if (!requiredLabels.every(r => itemLabels.includes(r))) {
            core.info(`Skipping add_labels for ${contextType} #${itemNumber}: does not match required-labels filter (${requiredLabels.join(", ")})`);
            return { success: false, skipped: true, error: `Item does not match required-labels filter` };
          }
        }
        if (requiredTitlePrefix && !item.title?.startsWith(requiredTitlePrefix)) {
          core.info(`Skipping add_labels for ${contextType} #${itemNumber}: title does not start with required prefix "${requiredTitlePrefix}"`);
          return { success: false, skipped: true, error: `Item title does not start with required prefix` };
        }
      }

      // If no labels provided, return a helpful message with allowed labels if configured
      if (requestedLabelNames.length === 0) {
        const labelSource = allowedLabels.length > 0 ? `the allowed list: ${JSON.stringify(allowedLabels)}` : "the repository's available labels";
        const error = `No labels provided. Please provide at least one label from ${labelSource}`;
        core.info(error);
        return { success: false, error };
      }

      // Enforce max limits on labels before validation
      const limitResult = tryEnforceArrayLimit(requestedLabelNames, MAX_LABELS, "labels");
      if (!limitResult.success) {
        core.warning(`Label limit exceeded: ${limitResult.error}`);
        return { success: false, error: limitResult.error };
      }

      // Use validation helper to sanitize and validate labels
      const labelsResult = validateLabels(requestedLabelNames, allowedLabels, maxCount, blockedPatterns);

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

      const labelsRequestPayload = uniqueLabels.map(name => {
        const labelSpec = requestedLabelSpecByLowerName.get(name.toLowerCase()) ?? { name };
        const hasIntentMetadata = Boolean(labelSpec.rationale || labelSpec.confidence || labelSpec.suggest);
        return issueIntentEnabled && hasIntentMetadata ? labelSpec : labelSpec.name;
      });

      core.info(`Adding ${uniqueLabels.length} labels to ${contextType} #${itemNumber} in ${itemRepo}: ${JSON.stringify(labelsRequestPayload)}`);

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
        const beforeState = await fetchIssueState(githubClient, repoParts, itemNumber);
        const { data: labels } = await withRetry(
          () =>
            githubClient.rest.issues.addLabels({
              owner: repoParts.owner,
              repo: repoParts.repo,
              issue_number: itemNumber,
              labels: labelsRequestPayload,
            }),
          RATE_LIMIT_RETRY_CONFIG,
          `add_labels to ${contextType} #${itemNumber} in ${itemRepo}`
        );

        core.info(`Successfully added ${uniqueLabels.length} labels to ${contextType} #${itemNumber} in ${itemRepo}`);
        return attachExecutionState(
          {
            success: true,
            number: itemNumber,
            repo: itemRepo,
            labelsAdded: uniqueLabels,
            contextType,
          },
          beforeState,
          {
            ...beforeState,
            labels: normalizeLabelNames(labels),
          }
        );
      } catch (error) {
        const errorMessage = getErrorMessage(error);
        core.error(`Failed to add labels: ${errorMessage}`);
        return { success: false, error: errorMessage };
      }
    };
  },
});

module.exports = { main };
