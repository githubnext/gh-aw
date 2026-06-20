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
 *   label_to_remove: string,
 *   label_to_add: string,
 *   repo?: string
 * }} ReplaceLabelMessage
 */

/** @type {string} Safe output type handled by this module */
const HANDLER_TYPE = "replace_label";

/**
 * GraphQL mutation that removes one label and adds another in a single request.
 * Root mutations in a single request are executed sequentially (remove first, then add).
 * When $doRemove is false the removeLabels field is skipped entirely, avoiding a
 * GitHub API error that would result from passing an empty labelIds array.
 *
 * Note: this is a sequential operation, not a transaction. If removeLabels succeeds
 * but addLabels fails the removal is not reversed (see RL-046 partial-failure handling).
 *
 * @type {string}
 */
const REPLACE_LABEL_MUTATION = /* GraphQL */ `
  mutation ReplaceLabelMutation($labelableId: ID!, $addLabelIds: [ID!]!, $removeLabelIds: [ID!]!, $doRemove: Boolean!) {
    removeLabels: removeLabelsFromLabelable(input: { labelableId: $labelableId, labelIds: $removeLabelIds }) @include(if: $doRemove) {
      clientMutationId
    }
    addLabels: addLabelsToLabelable(input: { labelableId: $labelableId, labelIds: $addLabelIds }) {
      labelable {
        ... on Issue {
          labels(first: 25) {
            nodes {
              name
            }
          }
        }
        ... on PullRequest {
          labels(first: 25) {
            nodes {
              name
            }
          }
        }
      }
    }
  }
`;

const { matchesSimpleGlob } = require("./glob_pattern_helpers.cjs");
const { getErrorMessage } = require("./error_helpers.cjs");
const { resolveTargetRepoConfig, resolveAndValidateRepo } = require("./repo_helpers.cjs");
const { logStagedPreviewInfo } = require("./staged_preview.cjs");
const { createAuthenticatedGitHubClient } = require("./handler_auth.cjs");
const { resolveSafeOutputIssueTarget } = require("./temporary_id.cjs");
const { attachExecutionState, fetchIssueState, normalizeLabelNames } = require("./safe_output_execution_metadata.cjs");
const { createCountGatedHandler } = require("./handler_scaffold.cjs");
const { withRetry, RATE_LIMIT_RETRY_CONFIG } = require("./error_recovery.cjs");
const { resolveInvocationContext } = require("./invocation_context_helpers.cjs");

/**
 * Resolve a label in the repository, returning its GraphQL node ID.
 * If the label does not exist in the repository, a hard error is thrown.
 *
 * @param {any} githubClient - Authenticated GitHub client with REST
 * @param {string} owner - Repository owner
 * @param {string} repo - Repository name
 * @param {string} labelName - Label name to resolve
 * @param {Map<string, string>} labelNodeIdCache - Cache of label name → node_id
 * @returns {Promise<string>} The GraphQL node ID of the label
 */
async function resolveLabel(githubClient, owner, repo, labelName, labelNodeIdCache) {
  if (labelNodeIdCache.has(labelName)) {
    return /** @type {string} */ labelNodeIdCache.get(labelName);
  }

  try {
    const { data: label } = await githubClient.rest.issues.getLabel({
      owner,
      repo,
      name: labelName,
    });
    labelNodeIdCache.set(labelName, label.node_id);
    return label.node_id;
  } catch (err) {
    if (err?.status !== 404) {
      throw err;
    }
    throw new Error(`Label "${labelName}" does not exist in ${owner}/${repo} (${getErrorMessage(err)}). Create the label in the repository before using it with replace-label.`);
  }
}

/**
 * Validate a single label against blocked and allowed-list patterns.
 * Uses explicit rejection semantics — does not silently filter or truncate the label name.
 * Blocked patterns are evaluated first (security boundary), consistent with safe_output_validator.cjs.
 *
 * @param {string} labelName - Label name to validate
 * @param {string[]} allowedPatterns - Allowlist patterns (empty = all labels allowed)
 * @param {string[]} blockedPatterns - Blocklist patterns
 * @param {string} fieldName - Field name for error messages (e.g. "label_to_add")
 * @returns {{valid: true} | {valid: false, error: string}}
 */
function validateSingleLabel(labelName, allowedPatterns, blockedPatterns, fieldName) {
  if (blockedPatterns.length > 0) {
    const isBlocked = blockedPatterns.some(pattern => matchesSimpleGlob(labelName, pattern));
    if (isBlocked) {
      return { valid: false, error: `${fieldName} "${labelName}" matches a blocked pattern` };
    }
  }
  if (allowedPatterns.length > 0) {
    const isAllowed = allowedPatterns.some(pattern => matchesSimpleGlob(labelName, pattern));
    if (!isAllowed) {
      return { valid: false, error: `${fieldName} "${labelName}" is not in the allowed list` };
    }
  }
  return { valid: true };
}

/**
 * Main handler factory for replace_label.
 * Uses the GraphQL API to remove one label and add another in a single request.
 * @type {HandlerFactoryFunction}
 */
const main = createCountGatedHandler({
  handlerType: HANDLER_TYPE,
  setup: async (config, maxCount, isStaged) => {
    const blockedPatterns = config.blocked || [];
    const requiredLabels = Array.isArray(config.required_labels) ? config.required_labels : [];
    const requiredTitlePrefix = config.required_title_prefix || "";
    const { defaultTargetRepo, allowedRepos } = resolveTargetRepoConfig(config);
    const githubClient = await createAuthenticatedGitHubClient(config);

    // Config keys use snake_case (set by the Go handler config builder)
    const configAllowedAdd = Array.isArray(config.allowed_add) ? config.allowed_add : [];
    const configAllowedRemove = Array.isArray(config.allowed_remove) ? config.allowed_remove : [];

    core.info(`Replace label configuration: max=${maxCount}`);
    if (configAllowedAdd.length > 0) core.info(`Allowed labels to add: ${configAllowedAdd.join(", ")}`);
    if (configAllowedRemove.length > 0) core.info(`Allowed labels to remove: ${configAllowedRemove.join(", ")}`);
    if (blockedPatterns.length > 0) core.info(`Blocked patterns: ${blockedPatterns.join(", ")}`);
    if (requiredLabels.length > 0) core.info(`Required labels (all): ${requiredLabels.join(", ")}`);
    if (requiredTitlePrefix) core.info(`Required title prefix: ${requiredTitlePrefix}`);
    core.info(`Default target repo: ${defaultTargetRepo}`);
    if (allowedRepos.size > 0) core.info(`Allowed repos: ${[...allowedRepos].join(", ")}`);

    /** Cache of repo label name → node_id, keyed per repo to avoid cross-repo conflicts */
    /** @type {Map<string, Map<string, string>>} */
    const repoCaches = new Map();

    /**
     * Message handler function that processes a single replace_label message.
     * @param {ReplaceLabelMessage} message - The replace_label message to process
     * @param {ResolvedTemporaryIds} resolvedTemporaryIds - Map of temporary IDs to {repo, number}
     * @returns {Promise<HandlerResult>} Result with success/error status
     */
    return async function handleReplaceLabel(message, resolvedTemporaryIds) {
      // Resolve and validate target repository
      const repoResult = resolveAndValidateRepo(message, defaultTargetRepo, allowedRepos, "label");
      if (!repoResult.success) {
        core.warning(`Skipping replace_label: ${repoResult.error}`);
        return { success: false, error: repoResult.error };
      }
      const { repo: itemRepo, repoParts } = repoResult;
      core.info(`Target repository: ${itemRepo}`);

      // Determine target issue/PR number
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
      const labelToRemove = String(message.label_to_remove ?? "").trim();
      const labelToAdd = String(message.label_to_add ?? "").trim();

      core.info(`Requested label replacement for ${contextType} #${itemNumber}: "${labelToRemove}" → "${labelToAdd}"`);

      if (!labelToRemove || !labelToAdd) {
        const error = "Both label_to_remove and label_to_add must be provided and non-empty";
        core.warning(error);
        return { success: false, error };
      }

      // Validate label_to_remove against blocked patterns and allowed-remove list
      const removeValidation = validateSingleLabel(labelToRemove, configAllowedRemove, blockedPatterns, "label_to_remove");
      if (!removeValidation.valid) {
        core.warning(`label_to_remove validation failed: ${removeValidation.error}`);
        return { success: false, error: removeValidation.error };
      }

      // Validate label_to_add against blocked patterns and allowed-add list
      const addValidation = validateSingleLabel(labelToAdd, configAllowedAdd, blockedPatterns, "label_to_add");
      if (!addValidation.valid) {
        core.warning(`label_to_add validation failed: ${addValidation.error}`);
        return { success: false, error: addValidation.error };
      }

      // Apply required-labels and required-title-prefix filters
      const { data: item } = await githubClient.rest.issues.get({
        owner: repoParts.owner,
        repo: repoParts.repo,
        issue_number: itemNumber,
      });

      if (requiredLabels.length > 0) {
        const itemLabels = (item.labels || []).map(/** @param {any} l */ l => (typeof l === "string" ? l : l.name || ""));
        if (!requiredLabels.every(r => itemLabels.includes(r))) {
          core.info(`Skipping replace_label for ${contextType} #${itemNumber}: does not match required-labels filter (${requiredLabels.join(", ")})`);
          return { success: false, skipped: true, error: "Item does not match required-labels filter" };
        }
      }
      if (requiredTitlePrefix && !item.title?.startsWith(requiredTitlePrefix)) {
        core.info(`Skipping replace_label for ${contextType} #${itemNumber}: title does not start with required prefix "${requiredTitlePrefix}"`);
        return { success: false, skipped: true, error: "Item title does not start with required prefix" };
      }

      // If in staged mode, preview the replacement without applying it
      if (isStaged) {
        logStagedPreviewInfo(`Would replace label "${labelToRemove}" → "${labelToAdd}" on ${contextType} #${itemNumber} in ${itemRepo}`);
        return {
          success: true,
          staged: true,
          previewInfo: {
            number: itemNumber,
            repo: itemRepo,
            labelToRemove,
            labelToAdd,
            contextType,
          },
        };
      }

      // Get or initialize the per-repo label cache
      if (!repoCaches.has(itemRepo)) {
        repoCaches.set(itemRepo, new Map());
      }
      const labelNodeIdCache = /** @type {Map<string, string>} */ repoCaches.get(itemRepo);

      // Resolve the node ID of label_to_add — fails with hard error if the label does not exist
      let addLabelNodeId;
      try {
        addLabelNodeId = await withRetry(() => resolveLabel(githubClient, repoParts.owner, repoParts.repo, labelToAdd, labelNodeIdCache), RATE_LIMIT_RETRY_CONFIG, `resolve label "${labelToAdd}" in ${itemRepo}`);
      } catch (err) {
        const errorMessage = getErrorMessage(err);
        core.error(`Failed to resolve label "${labelToAdd}": ${errorMessage}`);
        return { success: false, error: `Failed to resolve label "${labelToAdd}": ${errorMessage}` };
      }

      // Find the node ID of label_to_remove from the issue's current labels.
      // If the label is not on the issue we can still proceed (just won't remove anything).
      const currentLabelMap = new Map((item.labels || []).map(/** @param {any} l */ l => [l.name || "", l.node_id || ""]));
      const removeLabelNodeId = currentLabelMap.get(labelToRemove);

      if (!removeLabelNodeId) {
        core.info(`Label "${labelToRemove}" is not present on ${contextType} #${itemNumber} in ${itemRepo} — will only add "${labelToAdd}"`);
      }

      // Issue node_id for the GraphQL mutation
      const labelableId = item.node_id;

      core.info(`Executing combined GraphQL mutation: remove="${labelToRemove}", add="${labelToAdd}" on ${contextType} #${itemNumber} in ${itemRepo}`);

      const beforeState = await fetchIssueState(githubClient, repoParts, itemNumber);

      try {
        const mutationResult = await withRetry(
          () =>
            githubClient.graphql(REPLACE_LABEL_MUTATION, {
              labelableId,
              addLabelIds: [addLabelNodeId],
              removeLabelIds: removeLabelNodeId ? [removeLabelNodeId] : [],
              doRemove: !!removeLabelNodeId,
            }),
          RATE_LIMIT_RETRY_CONFIG,
          `replace_label on ${contextType} #${itemNumber} in ${itemRepo}`
        );

        const updatedLabels = mutationResult?.addLabels?.labelable?.labels?.nodes || [];
        const updatedLabelNames = updatedLabels.map((/** @param {any} l */ l) => l.name || "").filter(Boolean);

        core.info(`Successfully replaced label "${labelToRemove}" → "${labelToAdd}" on ${contextType} #${itemNumber} in ${itemRepo}`);
        core.info(`Updated labels: ${JSON.stringify(updatedLabelNames)}`);

        return attachExecutionState(
          {
            success: true,
            number: itemNumber,
            repo: itemRepo,
            labelRemoved: removeLabelNodeId ? labelToRemove : null,
            labelAdded: labelToAdd,
            contextType,
          },
          beforeState,
          {
            ...beforeState,
            labels: updatedLabelNames.length > 0 ? updatedLabelNames : normalizeLabelNames(item.labels),
          }
        );
      } catch (err) {
        const errorMessage = getErrorMessage(err);
        // RL-046: detect partial mutation success — remove succeeded but add failed.
        // withRetry may wrap the original error via enhanceError, so check both:
        //   err.data          — present on direct @octokit/graphql GraphQLResponseError
        //   err.originalError.data — present when withRetry has wrapped the graphql error
        // The nullish-coalescing order is intentional: prefer err.data (the closest
        // error to the API boundary); fall back to err.originalError.data only when
        // err.data is absent (i.e. the error has been wrapped by enhanceError).
        const errAsAny = /** @type {any} */ err;
        const partialData = errAsAny?.data ?? errAsAny?.originalError?.data;
        if (partialData?.removeLabels && !partialData?.addLabels) {
          core.error(`Partial mutation failure on ${contextType} #${itemNumber} in ${itemRepo}: "${labelToRemove}" was removed but "${labelToAdd}" could not be added: ${errorMessage}`);
        } else {
          core.error(`Failed to replace label: ${errorMessage}`);
        }
        return { success: false, error: errorMessage };
      }
    };
  },
});

module.exports = { main, REPLACE_LABEL_MUTATION };
