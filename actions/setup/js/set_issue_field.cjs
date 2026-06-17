// @ts-check
/// <reference types="@actions/github-script" />

/**
 * @typedef {import('./types/handler-factory').HandlerFactoryFunction} HandlerFactoryFunction
 */

const { getErrorMessage } = require("./error_helpers.cjs");
const { resolveTargetRepoConfig, resolveAndValidateRepo } = require("./repo_helpers.cjs");
const { logStagedPreviewInfo } = require("./staged_preview.cjs");
const { isStagedMode, checkRequiredFilter } = require("./safe_output_helpers.cjs");
const { createAuthenticatedGitHubClient } = require("./handler_auth.cjs");
const { parseAllowedIssueFields, validateAllowedIssueFieldName } = require("./allowed_issue_fields.cjs");
const { resolveSafeOutputIssueTarget } = require("./temporary_id.cjs");

/** @type {string} Safe output type handled by this module */
const HANDLER_TYPE = "set_issue_field";

/**
 * Fetches the node ID of an issue for use in GraphQL mutations.
 * @param {Object} githubClient - Authenticated GitHub client
 * @param {string} owner - Repository owner
 * @param {string} repo - Repository name
 * @param {number} issueNumber - Issue number
 * @returns {Promise<string>} Issue node ID
 */
async function getIssueNodeId(githubClient, owner, repo, issueNumber) {
  const { data } = await githubClient.rest.issues.get({
    owner,
    repo,
    issue_number: issueNumber,
  });
  return data.node_id;
}

/**
 * Fetches available issue fields for the repository/owner.
 * @param {Object} githubClient - Authenticated GitHub client
 * @param {string} owner - Repository owner
 * @param {string} repo - Repository name
 * @returns {Promise<Array<{id: string, name: string, __typename?: string, options?: Array<{id: string, name: string}>}>>}
 */
async function fetchIssueFields(githubClient, owner, repo) {
  const result = await githubClient.graphql(
    `query($owner: String!, $repo: String!) {
      repository(owner: $owner, name: $repo) {
        issueFields(first: 100) {
          nodes {
            __typename
            ... on IssueFieldText { id name }
            ... on IssueFieldNumber { id name }
            ... on IssueFieldDate { id name }
            ... on IssueFieldSingleSelect { id name options { id name } }
            ... on IssueFieldMultiSelect { id name options { id name } }
          }
        }
        owner {
          __typename
          ... on Organization {
            issueFields(first: 100) {
              nodes {
                __typename
                ... on IssueFieldText { id name }
                ... on IssueFieldNumber { id name }
                ... on IssueFieldDate { id name }
                ... on IssueFieldSingleSelect { id name options { id name } }
                ... on IssueFieldMultiSelect { id name options { id name } }
              }
            }
          }
        }
      }
    }`,
    { owner, repo }
  );

  const isValidNode = node => typeof node?.id === "string" && typeof node?.name === "string";

  const repoFields = (result?.repository?.issueFields?.nodes ?? []).filter(isValidNode);
  if (repoFields.length > 0) {
    return repoFields;
  }

  const ownerFields = (result?.repository?.owner?.issueFields?.nodes ?? []).filter(isValidNode);
  return ownerFields;
}

/**
 * Builds a field update payload based on field type and value.
 * @param {{__typename?: string, name?: string, options?: Array<{id: string, name: string}>}|null} field
 * @param {string} rawValue
 * @returns {{success: true, update: Record<string, any>} | {success: false, error: string}}
 */
function buildFieldUpdatePayload(field, rawValue) {
  const fieldType = field?.__typename || "IssueFieldText";

  if (fieldType === "IssueFieldSingleSelect") {
    const options = field?.options ?? [];
    const selected = options.find(option => option.name.toLowerCase() === rawValue.toLowerCase());
    if (!selected) {
      const availableOptions = options.map(option => option.name).join(", ");
      return {
        success: false,
        error: `Invalid value ${JSON.stringify(rawValue)} for issue field ${JSON.stringify(field?.name || "(unknown)")}. Available options: ${availableOptions}. Use the exact option name or pass field_node_id to bypass name discovery.`,
      };
    }
    return {
      success: true,
      update: {
        singleSelectOptionId: selected.id,
      },
    };
  }

  if (fieldType === "IssueFieldNumber") {
    const parsed = Number(rawValue);
    if (!Number.isFinite(parsed)) {
      return {
        success: false,
        error: `Invalid value ${JSON.stringify(rawValue)} for numeric issue field ${JSON.stringify(field?.name || "(unknown)")}. Provide a numeric value (example: "3.14").`,
      };
    }
    return {
      success: true,
      update: {
        numberValue: parsed,
      },
    };
  }

  if (fieldType === "IssueFieldDate") {
    if (!/^\d{4}-\d{2}-\d{2}$/.test(rawValue)) {
      return {
        success: false,
        error: `Invalid value ${JSON.stringify(rawValue)} for date issue field ${JSON.stringify(field?.name || "(unknown)")}. Use YYYY-MM-DD format.`,
      };
    }
    return {
      success: true,
      update: {
        dateValue: rawValue,
      },
    };
  }

  return {
    success: true,
    update: {
      textValue: rawValue,
    },
  };
}

/**
 * Sets one issue field via GraphQL mutation.
 * @param {Object} githubClient - Authenticated GitHub client
 * @param {string} issueNodeId - GraphQL node ID of the issue
 * @param {{fieldId: string, singleSelectOptionId?: string, numberValue?: number, dateValue?: string, textValue?: string}} fieldUpdate
 * @returns {Promise<void>}
 */
async function setIssueFieldValue(githubClient, issueNodeId, fieldUpdate) {
  await githubClient.graphql(
    `mutation($issueId: ID!, $issueFields: [IssueFieldCreateOrUpdateInput!]!) {
      setIssueFieldValue(input: { issueId: $issueId, issueFields: $issueFields }) {
        issue {
          id
        }
      }
    }`,
    {
      issueId: issueNodeId,
      issueFields: [fieldUpdate],
    }
  );
}

/**
 * Main handler factory for set_issue_field.
 * @type {HandlerFactoryFunction}
 */
async function main(config = {}) {
  const maxCount = config.max || 5;
  const allowedIssueFields = parseAllowedIssueFields(config.allowed_fields);
  const { defaultTargetRepo, allowedRepos } = resolveTargetRepoConfig(config);
  const githubClient = await createAuthenticatedGitHubClient(config);
  const isStaged = isStagedMode(config);

  core.info(`Set issue field configuration: max=${maxCount}`);
  const requiredLabels = Array.isArray(config.required_labels) ? config.required_labels : [];
  const requiredTitlePrefix = config.required_title_prefix || "";
  if (requiredLabels.length > 0) core.info(`Required labels (all): ${requiredLabels.join(", ")}`);
  if (requiredTitlePrefix) core.info(`Required title prefix: ${requiredTitlePrefix}`);
  core.info(`Default target repo: ${defaultTargetRepo}`);
  if (allowedRepos.size > 0) {
    core.info(`Allowed repos: ${Array.from(allowedRepos).join(", ")}`);
  }
  if (allowedIssueFields.length > 0 && !allowedIssueFields.includes("*")) {
    core.info(`Allowed issue fields: ${allowedIssueFields.join(", ")}`);
  }

  let processedCount = 0;

  return async function handleSetIssueField(message, resolvedTemporaryIds) {
    if (processedCount >= maxCount) {
      core.warning(`Skipping set_issue_field: max count of ${maxCount} reached`);
      return {
        success: false,
        error: `Max count of ${maxCount} reached`,
      };
    }

    processedCount++;

    const item = message;

    const repoResult = resolveAndValidateRepo(item, defaultTargetRepo, allowedRepos, "issue");
    if (!repoResult.success) {
      core.warning(`Skipping set_issue_field: ${repoResult.error}`);
      return {
        success: false,
        error: repoResult.error,
      };
    }
    const { repo: itemRepo, repoParts } = repoResult;
    core.info(`Target repository: ${itemRepo}`);

    const targetResult = resolveSafeOutputIssueTarget({ message: item, resolvedTemporaryIds, repoParts, handlerType: "set_issue_field", aliases: ["issue_number"] });
    if (!targetResult.success) return targetResult;
    let issueNumber;
    if (targetResult.number !== null) {
      issueNumber = targetResult.number;
      core.info(`Resolved issue number: #${issueNumber}`);
    } else {
      const contextIssueNumber = context.payload?.issue?.number;
      if (!contextIssueNumber) {
        core.warning("No issue_number provided and not in issue context");
        return {
          success: false,
          error: "No issue number available",
        };
      }
      issueNumber = contextIssueNumber;
    }

    const filterResult = await checkRequiredFilter(githubClient, repoParts, issueNumber, requiredLabels, requiredTitlePrefix, "set_issue_field");
    if (filterResult) return filterResult;

    if (item.value === undefined || item.value === null) {
      return {
        success: false,
        error: "Missing required value. Provide the issue field value as a string.",
      };
    }

    const fieldName = typeof item.field_name === "string" ? item.field_name.trim() : "";
    let fieldNodeId = typeof item.field_node_id === "string" ? item.field_node_id.trim() : "";
    const value = String(item.value);

    if (!fieldName && !fieldNodeId) {
      return {
        success: false,
        error: "Missing field identifier. Provide field_name or field_node_id.",
      };
    }

    if (isStaged) {
      const description = `Would set issue field ${JSON.stringify(fieldName || fieldNodeId)} to ${JSON.stringify(value)} on issue #${issueNumber} in ${itemRepo}`;
      logStagedPreviewInfo(description);
      return {
        success: true,
        staged: true,
        previewInfo: {
          issue_number: issueNumber,
          field_name: fieldName,
          field_node_id: fieldNodeId,
          value,
          repo: itemRepo,
        },
      };
    }

    try {
      const { owner, repo } = repoParts;
      const issueNodeId = await getIssueNodeId(githubClient, owner, repo, issueNumber);

      /** @type {{id: string, name: string, __typename?: string, options?: Array<{id: string, name: string}>}|null} */
      let resolvedField = null;

      const availableFields = await fetchIssueFields(githubClient, owner, repo);

      if (availableFields.length === 0) {
        const warning = "No issue fields were discovered for this repository. Verify issue fields are enabled and visible to this token.";
        core.warning(warning);
        return { success: false, skipped: true, error: warning };
      }

      let resolvedFieldByName = null;
      if (fieldName) {
        resolvedFieldByName = availableFields.find(field => field.name.toLowerCase() === fieldName.toLowerCase()) || null;
        if (!resolvedFieldByName) {
          const availableNames = availableFields.map(field => field.name).join(", ");
          const error = `Issue field ${JSON.stringify(fieldName)} not found. Available fields: ${availableNames}. Use a listed field_name or provide field_node_id to bypass discovery.`;
          core.error(error);
          return { success: false, error };
        }
      }

      if (fieldNodeId) {
        resolvedField = availableFields.find(field => field.id === fieldNodeId) || null;
      }

      if (!fieldNodeId && resolvedFieldByName) {
        fieldNodeId = resolvedFieldByName.id;
        resolvedField = resolvedFieldByName;
      }

      if (fieldNodeId && !resolvedField) {
        const availableFieldsSummary = availableFields.map(field => `${field.name} (${field.id})`).join(", ");
        const error = `Issue field ID ${JSON.stringify(fieldNodeId)} not found. Available fields: ${availableFieldsSummary}. Use a valid field_node_id or provide field_name.`;
        core.error(error);
        return { success: false, error };
      }

      const hasConflictingFieldIdentifiers = Boolean(fieldNodeId && fieldName && resolvedFieldByName && resolvedField && resolvedFieldByName.id !== resolvedField.id);
      if (hasConflictingFieldIdentifiers) {
        const error = `field_name ${JSON.stringify(fieldName)} resolves to ${JSON.stringify(resolvedFieldByName?.id)}, but field_node_id was ${JSON.stringify(fieldNodeId)}. Provide only one identifier or make them match.`;
        core.error(error);
        return { success: false, error };
      }

      if (!fieldNodeId) {
        const error = "Could not resolve field_node_id. Provide a valid field_name or explicit field_node_id.";
        core.error(error);
        return { success: false, error };
      }

      const resolvedFieldName = resolvedField?.name || fieldName;
      validateAllowedIssueFieldName(resolvedFieldName, allowedIssueFields);

      const fieldUpdateResult = buildFieldUpdatePayload(resolvedField, value);
      if (!fieldUpdateResult.success) {
        core.error(fieldUpdateResult.error);
        return { success: false, error: fieldUpdateResult.error };
      }

      const fieldUpdate = {
        fieldId: fieldNodeId,
        ...fieldUpdateResult.update,
      };

      await setIssueFieldValue(githubClient, issueNodeId, fieldUpdate);

      core.info(`Successfully set issue field ${JSON.stringify(fieldName || fieldNodeId)} to ${JSON.stringify(value)} on issue #${issueNumber}`);

      return {
        success: true,
        issue_number: issueNumber,
        field_name: fieldName,
        field_node_id: fieldNodeId,
        value,
        repo: itemRepo,
      };
    } catch (error) {
      const errorMessage = getErrorMessage(error);
      core.error(`Failed to set issue field on issue #${issueNumber}: ${errorMessage}`);
      return { success: false, error: errorMessage };
    }
  };
}

module.exports = { main };
