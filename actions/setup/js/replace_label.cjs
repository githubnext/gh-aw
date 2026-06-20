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
 * Root mutations in a single request are executed sequentially (remove first, then add),
 * providing an atomic state transition for label-based state machines.
 *
 * @type {string}
 */
const REPLACE_LABEL_MUTATION = /* GraphQL */ `
  mutation ReplaceLabelMutation($labelableId: ID!, $addLabelIds: [ID!]!, $removeLabelIds: [ID!]!) {
    removeLabels: removeLabelsFromLabelable(input: { labelableId: $labelableId, labelIds: $removeLabelIds }) {
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

/**
 * GraphQL query to resolve label node IDs from a repository by name.
 * Searches for labels matching the given names so we can get their node IDs
 * for the mutation.
 *
 * @type {string}
 */
const RESOLVE_LABEL_QUERY = /* GraphQL */ `
  query ResolveLabelNodeIds($owner: String!, $repo: String!) {
    repository(owner: $owner, name: $repo) {
      labels(first: 100) {
        nodes {
          id
          name
        }
      }
    }
  }
`;

const { validateLabels } = require("./safe_output_validator.cjs");
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
 * Resolve or create a label in the repository, returning its GraphQL node ID.
 * If the label does not exist, it is created with a deterministic pastel color.
 *
 * @param {any} githubClient - Authenticated GitHub client with REST and graphql
 * @param {string} owner - Repository owner
 * @param {string} repo - Repository name
 * @param {string} labelName - Label name to resolve or create
 * @param {Map<string, string>} labelNodeIdCache - Cache of label name → node_id
 * @returns {Promise<string>} The GraphQL node ID of the label
 */
async function resolveOrCreateLabel(githubClient, owner, repo, labelName, labelNodeIdCache) {
  if (labelNodeIdCache.has(labelName)) {
    return /** @type {string} */ labelNodeIdCache.get(labelName);
  }

  // Try to get the label from the repo
  try {
    const { data: label } = await githubClient.rest.issues.getLabel({
      owner,
      repo,
      name: labelName,
    });
    labelNodeIdCache.set(labelName, label.node_id);
    return label.node_id;
  } catch (err) {
    const msg = getErrorMessage(err);
    if (!msg.includes("404") && !msg.toLowerCase().includes("not found")) {
      throw err;
    }
  }

  // Label does not exist — create it with a deterministic color
  core.info(`Label "${labelName}" not found in ${owner}/${repo}, creating it`);
  const color = deterministicLabelColor(labelName);
  const { data: created } = await githubClient.rest.issues.createLabel({
    owner,
    repo,
    name: labelName,
    color,
  });
  core.info(`Created label "${labelName}" with color #${color}`);
  labelNodeIdCache.set(labelName, created.node_id);
  return created.node_id;
}

/**
 * Generate a deterministic pastel hex color from a label name.
 * Produces colors in the pastel range (128–191 per channel) for readability.
 *
 * @param {string} name
 * @returns {string} Six-character hex color (no leading #)
 */
function deterministicLabelColor(name) {
  let hash = 0;
  for (let i = 0; i < name.length; i++) {
    hash = (hash * 31 + name.charCodeAt(i)) >>> 0;
  }
  const r = 128 + (hash & 0x3f);
  const g = 128 + ((hash >> 6) & 0x3f);
  const b = 128 + ((hash >> 12) & 0x3f);
  return ((r << 16) | (g << 8) | b).toString(16).padStart(6, "0");
}

/**
 * Main handler factory for replace_label.
 * Uses the GraphQL API to remove one label and add another in a single atomic request.
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

      // Validate label_to_remove against allowed-remove and blocked patterns
      const removeValidation = validateLabels([labelToRemove], configAllowedRemove, 1, blockedPatterns);
      if (!removeValidation.valid) {
        core.warning(`label_to_remove validation failed: ${removeValidation.error}`);
        return { success: false, error: removeValidation.error ?? "Invalid label_to_remove" };
      }

      // Validate label_to_add against allowed-add and blocked patterns
      const addValidation = validateLabels([labelToAdd], configAllowedAdd, 1, blockedPatterns);
      if (!addValidation.valid) {
        core.warning(`label_to_add validation failed: ${addValidation.error}`);
        return { success: false, error: addValidation.error ?? "Invalid label_to_add" };
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

      // Resolve the node ID of label_to_add (create if it doesn't exist in the repo)
      let addLabelNodeId;
      try {
        addLabelNodeId = await withRetry(() => resolveOrCreateLabel(githubClient, repoParts.owner, repoParts.repo, labelToAdd, labelNodeIdCache), RATE_LIMIT_RETRY_CONFIG, `resolve/create label "${labelToAdd}" in ${itemRepo}`);
      } catch (err) {
        const errorMessage = getErrorMessage(err);
        core.error(`Failed to resolve/create label "${labelToAdd}": ${errorMessage}`);
        return { success: false, error: `Failed to resolve/create label "${labelToAdd}": ${errorMessage}` };
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
        core.error(`Failed to replace label: ${errorMessage}`);
        return { success: false, error: errorMessage };
      }
    };
  },
});

module.exports = { main, deterministicLabelColor, REPLACE_LABEL_MUTATION, RESOLVE_LABEL_QUERY };
