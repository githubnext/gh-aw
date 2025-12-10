// @ts-check
/// <reference types="@actions/github-script" />

// === Inlined from ./load_agent_output.cjs ===
// @ts-check
/// <reference types="@actions/github-script" />

const fs = require("fs");
const crypto = require("crypto");

/**
 * Maximum content length to log for debugging purposes
 * @type {number}
 */
const MAX_LOG_CONTENT_LENGTH = 10000;

/**
 * Truncate content for logging if it exceeds the maximum length
 * @param {string} content - Content to potentially truncate
 * @returns {string} Truncated content with indicator if truncated
 */
function truncateForLogging(content) {
  if (content.length <= MAX_LOG_CONTENT_LENGTH) {
    return content;
  }
  return content.substring(0, MAX_LOG_CONTENT_LENGTH) + `\n... (truncated, total length: ${content.length})`;
}

/**
 * Load and parse agent output from the GH_AW_AGENT_OUTPUT file
 *
 * This utility handles the common pattern of:
 * 1. Reading the GH_AW_AGENT_OUTPUT environment variable
 * 2. Loading the file content
 * 3. Validating the JSON structure
 * 4. Returning parsed items array
 *
 * @returns {{
 *   success: true,
 *   items: any[]
 * } | {
 *   success: false,
 *   items?: undefined,
 *   error?: string
 * }} Result object with success flag and items array (if successful) or error message
 */
function loadAgentOutput() {
  const agentOutputFile = process.env.GH_AW_AGENT_OUTPUT;

  // No agent output file specified
  if (!agentOutputFile) {
    core.info("No GH_AW_AGENT_OUTPUT environment variable found");
    return { success: false };
  }

  // Read agent output from file
  let outputContent;
  try {
    outputContent = fs.readFileSync(agentOutputFile, "utf8");
  } catch (error) {
    const errorMessage = `Error reading agent output file: ${error instanceof Error ? error.message : String(error)}`;
    core.error(errorMessage);
    return { success: false, error: errorMessage };
  }

  // Check for empty content
  if (outputContent.trim() === "") {
    core.info("Agent output content is empty");
    return { success: false };
  }

  core.info(`Agent output content length: ${outputContent.length}`);

  // Parse the validated output JSON
  let validatedOutput;
  try {
    validatedOutput = JSON.parse(outputContent);
  } catch (error) {
    const errorMessage = `Error parsing agent output JSON: ${error instanceof Error ? error.message : String(error)}`;
    core.error(errorMessage);
    core.info(`Failed to parse content:\n${truncateForLogging(outputContent)}`);
    return { success: false, error: errorMessage };
  }

  // Validate items array exists
  if (!validatedOutput.items || !Array.isArray(validatedOutput.items)) {
    core.info("No valid items found in agent output");
    core.info(`Parsed content: ${truncateForLogging(JSON.stringify(validatedOutput))}`);
    return { success: false };
  }

  return { success: true, items: validatedOutput.items };
}

// === End of ./load_agent_output.cjs ===

// === Inlined from ./get_tracker_id.cjs ===
// @ts-check
/// <reference types="@actions/github-script" />

/**
 * Get tracker-id from environment variable, log it, and optionally format it
 * @param {string} [format] - Output format: "markdown" for HTML comment, "text" for plain text, or undefined for raw value
 * @returns {string} Tracker ID in requested format or empty string
 */
function getTrackerID(format) {
  const trackerID = process.env.GH_AW_TRACKER_ID || "";
  if (trackerID) {
    core.info(`Tracker ID: ${trackerID}`);
    return format === "markdown" ? `\n\n<!-- tracker-id: ${trackerID} -->` : trackerID;
  }
  return "";
}

// === End of ./get_tracker_id.cjs ===

// === Inlined from ./close_older_discussions.cjs ===
// @ts-check
/// <reference types="@actions/github-script" />

// === Inlined from ./messages_close_discussion.cjs ===
// @ts-check
/// <reference types="@actions/github-script" />

/**
 * Close Discussion Message Module
 *
 * This module provides the message for closing older discussions
 * when a newer one is created.
 */

// === Inlined from ./messages_core.cjs ===
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

// === End of ./messages_core.cjs ===


/**
 * @typedef {Object} CloseOlderDiscussionContext
 * @property {string} newDiscussionUrl - URL of the new discussion that replaced this one
 * @property {number} newDiscussionNumber - Number of the new discussion
 * @property {string} workflowName - Name of the workflow
 * @property {string} runUrl - URL of the workflow run
 */

/**
 * Get the close-older-discussion message, using custom template if configured.
 * @param {CloseOlderDiscussionContext} ctx - Context for message generation
 * @returns {string} Close older discussion message
 */
function getCloseOlderDiscussionMessage(ctx) {
  const messages = getMessages();

  // Create context with both camelCase and snake_case keys
  const templateContext = toSnakeCase(ctx);

  // Default close-older-discussion template - pirate themed! 🏴‍☠️
  const defaultMessage = `⚓ Avast! This discussion be marked as **outdated** by [{workflow_name}]({run_url}).

🗺️ A newer treasure map awaits ye at **[Discussion #{new_discussion_number}]({new_discussion_url})**.

Fair winds, matey! 🏴‍☠️`;

  // Use custom message if configured
  return messages?.closeOlderDiscussion
    ? renderTemplate(messages.closeOlderDiscussion, templateContext)
    : renderTemplate(defaultMessage, templateContext);
}

// === End of ./messages_close_discussion.cjs ===


/**
 * Maximum number of older discussions to close
 */
const MAX_CLOSE_COUNT = 10;

/**
 * Delay between GraphQL API calls in milliseconds to avoid rate limiting
 */
const GRAPHQL_DELAY_MS = 500;

/**
 * Delay execution for a specified number of milliseconds
 * @param {number} ms - Milliseconds to delay
 * @returns {Promise<void>}
 */
function delay(ms) {
  return new Promise(resolve => setTimeout(resolve, ms));
}

/**
 * Search for open discussions with a matching title prefix and/or labels
 * @param {any} github - GitHub GraphQL instance
 * @param {string} owner - Repository owner
 * @param {string} repo - Repository name
 * @param {string} titlePrefix - Title prefix to match (empty string to skip prefix matching)
 * @param {string[]} labels - Labels to match (empty array to skip label matching)
 * @param {string|undefined} categoryId - Optional category ID to filter by
 * @param {number} excludeNumber - Discussion number to exclude (the newly created one)
 * @returns {Promise<Array<{id: string, number: number, title: string, url: string}>>} Matching discussions
 */
async function searchOlderDiscussions(github, owner, repo, titlePrefix, labels, categoryId, excludeNumber) {
  // Build GraphQL search query
  // Search for open discussions, optionally with title prefix or labels
  let searchQuery = `repo:${owner}/${repo} is:open`;

  if (titlePrefix) {
    // Escape quotes in title prefix to prevent query injection
    const escapedPrefix = titlePrefix.replace(/"/g, '\\"');
    searchQuery += ` in:title "${escapedPrefix}"`;
  }

  // Add label filters to the search query
  // Note: GitHub search uses AND logic for multiple labels, so discussions must have ALL labels.
  // We add each label as a separate filter and also validate client-side for extra safety.
  if (labels && labels.length > 0) {
    for (const label of labels) {
      // Escape quotes in label names to prevent query injection
      const escapedLabel = label.replace(/"/g, '\\"');
      searchQuery += ` label:"${escapedLabel}"`;
    }
  }

  const result = await github.graphql(
    `
    query($searchTerms: String!, $first: Int!) {
      search(query: $searchTerms, type: DISCUSSION, first: $first) {
        nodes {
          ... on Discussion {
            id
            number
            title
            url
            category {
              id
            }
            labels(first: 100) {
              nodes {
                name
              }
            }
            closed
          }
        }
      }
    }`,
    { searchTerms: searchQuery, first: 50 }
  );

  if (!result || !result.search || !result.search.nodes) {
    return [];
  }

  // Filter results:
  // 1. Must not be the excluded discussion (newly created one)
  // 2. Must not be already closed
  // 3. If titlePrefix is specified, must have title starting with the prefix
  // 4. If labels are specified, must have ALL specified labels (AND logic, not OR)
  // 5. If categoryId is specified, must match
  return result.search.nodes
    .filter(
      /** @param {any} d */ d => {
        if (!d || d.number === excludeNumber || d.closed) {
          return false;
        }

        // Check title prefix if specified
        if (titlePrefix && d.title && !d.title.startsWith(titlePrefix)) {
          return false;
        }

        // Check labels if specified - requires ALL labels to match (AND logic)
        // This is intentional: we only want to close discussions that have ALL the specified labels
        if (labels && labels.length > 0) {
          const discussionLabels = d.labels?.nodes?.map((/** @type {{name: string}} */ l) => l.name) || [];
          const hasAllLabels = labels.every(label => discussionLabels.includes(label));
          if (!hasAllLabels) {
            return false;
          }
        }

        // Check category if specified
        if (categoryId && (!d.category || d.category.id !== categoryId)) {
          return false;
        }

        return true;
      }
    )
    .map(
      /** @param {any} d */ d => ({
        id: d.id,
        number: d.number,
        title: d.title,
        url: d.url,
      })
    );
}

/**
 * Add comment to a GitHub Discussion using GraphQL
 * @param {any} github - GitHub GraphQL instance
 * @param {string} discussionId - Discussion node ID
 * @param {string} message - Comment body
 * @returns {Promise<{id: string, url: string}>} Comment details
 */
async function addDiscussionComment(github, discussionId, message) {
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
    { dId: discussionId, body: message }
  );

  return result.addDiscussionComment.comment;
}

/**
 * Close a GitHub Discussion as OUTDATED using GraphQL
 * @param {any} github - GitHub GraphQL instance
 * @param {string} discussionId - Discussion node ID
 * @returns {Promise<{id: string, url: string}>} Discussion details
 */
async function closeDiscussionAsOutdated(github, discussionId) {
  const result = await github.graphql(
    `
    mutation($dId: ID!) {
      closeDiscussion(input: { discussionId: $dId, reason: OUTDATED }) {
        discussion { 
          id
          url
        }
      }
    }`,
    { dId: discussionId }
  );

  return result.closeDiscussion.discussion;
}

/**
 * Close older discussions that match the title prefix and/or labels
 * @param {any} github - GitHub GraphQL instance
 * @param {string} owner - Repository owner
 * @param {string} repo - Repository name
 * @param {string} titlePrefix - Title prefix to match (empty string to skip)
 * @param {string[]} labels - Labels to match (empty array to skip)
 * @param {string|undefined} categoryId - Optional category ID to filter by
 * @param {{number: number, url: string}} newDiscussion - The newly created discussion
 * @param {string} workflowName - Name of the workflow
 * @param {string} runUrl - URL of the workflow run
 * @returns {Promise<Array<{number: number, url: string}>>} List of closed discussions
 */
async function closeOlderDiscussions(github, owner, repo, titlePrefix, labels, categoryId, newDiscussion, workflowName, runUrl) {
  // Build search criteria description for logging
  const searchCriteria = [];
  if (titlePrefix) searchCriteria.push(`title prefix: "${titlePrefix}"`);
  if (labels && labels.length > 0) searchCriteria.push(`labels: [${labels.join(", ")}]`);
  core.info(`Searching for older discussions with ${searchCriteria.join(" and ")}`);

  const olderDiscussions = await searchOlderDiscussions(github, owner, repo, titlePrefix, labels, categoryId, newDiscussion.number);

  if (olderDiscussions.length === 0) {
    core.info("No older discussions found to close");
    return [];
  }

  core.info(`Found ${olderDiscussions.length} older discussion(s) to close`);

  // Limit to MAX_CLOSE_COUNT discussions
  const discussionsToClose = olderDiscussions.slice(0, MAX_CLOSE_COUNT);

  if (olderDiscussions.length > MAX_CLOSE_COUNT) {
    core.warning(`Found ${olderDiscussions.length} older discussions, but only closing the first ${MAX_CLOSE_COUNT}`);
  }

  const closedDiscussions = [];

  for (let i = 0; i < discussionsToClose.length; i++) {
    const discussion = discussionsToClose[i];
    try {
      // Generate closing message using the messages module
      const closingMessage = getCloseOlderDiscussionMessage({
        newDiscussionUrl: newDiscussion.url,
        newDiscussionNumber: newDiscussion.number,
        workflowName,
        runUrl,
      });

      // Add comment first
      core.info(`Adding closing comment to discussion #${discussion.number}`);
      await addDiscussionComment(github, discussion.id, closingMessage);

      // Then close the discussion as outdated
      core.info(`Closing discussion #${discussion.number} as outdated`);
      await closeDiscussionAsOutdated(github, discussion.id);

      closedDiscussions.push({
        number: discussion.number,
        url: discussion.url,
      });

      core.info(`✓ Closed discussion #${discussion.number}: ${discussion.url}`);
    } catch (error) {
      core.error(`✗ Failed to close discussion #${discussion.number}: ${error instanceof Error ? error.message : String(error)}`);
      // Continue with other discussions even if one fails
    }

    // Add delay between GraphQL operations to avoid rate limiting (except for the last item)
    if (i < discussionsToClose.length - 1) {
      await delay(GRAPHQL_DELAY_MS);
    }
  }

  return closedDiscussions;
}

// === End of ./close_older_discussions.cjs ===

// === Inlined from ./temporary_id.cjs ===
// @ts-check
/// <reference types="@actions/github-script" />


/**
 * Regex pattern for matching temporary ID references in text
 * Format: #aw_XXXXXXXXXXXX (aw_ prefix + 12 hex characters)
 */
const TEMPORARY_ID_PATTERN = /#(aw_[0-9a-f]{12})/gi;

/**
 * @typedef {Object} RepoIssuePair
 * @property {string} repo - Repository slug in "owner/repo" format
 * @property {number} number - Issue or discussion number
 */

/**
 * Generate a temporary ID with aw_ prefix for temporary issue IDs
 * @returns {string} A temporary ID in format aw_XXXXXXXXXXXX (12 hex characters)
 */
function generateTemporaryId() {
  return "aw_" + crypto.randomBytes(6).toString("hex");
}

/**
 * Check if a value is a valid temporary ID (aw_ prefix + 12-character hex string)
 * @param {any} value - The value to check
 * @returns {boolean} True if the value is a valid temporary ID
 */
function isTemporaryId(value) {
  if (typeof value === "string") {
    return /^aw_[0-9a-f]{12}$/i.test(value);
  }
  return false;
}

/**
 * Normalize a temporary ID to lowercase for consistent map lookups
 * @param {string} tempId - The temporary ID to normalize
 * @returns {string} Lowercase temporary ID
 */
function normalizeTemporaryId(tempId) {
  return String(tempId).toLowerCase();
}

/**
 * Replace temporary ID references in text with actual issue numbers
 * Format: #aw_XXXXXXXXXXXX -> #123 (same repo) or owner/repo#123 (cross-repo)
 * @param {string} text - The text to process
 * @param {Map<string, RepoIssuePair>} tempIdMap - Map of temporary_id to {repo, number}
 * @param {string} [currentRepo] - Current repository slug for same-repo references
 * @returns {string} Text with temporary IDs replaced with issue numbers
 */
function replaceTemporaryIdReferences(text, tempIdMap, currentRepo) {
  return text.replace(TEMPORARY_ID_PATTERN, (match, tempId) => {
    const resolved = tempIdMap.get(normalizeTemporaryId(tempId));
    if (resolved !== undefined) {
      // If we have a currentRepo and the issue is in the same repo, use short format
      if (currentRepo && resolved.repo === currentRepo) {
        return `#${resolved.number}`;
      }
      // Otherwise use full repo#number format for cross-repo references
      return `${resolved.repo}#${resolved.number}`;
    }
    // Return original if not found (it may be created later)
    return match;
  });
}

/**
 * Replace temporary ID references in text with actual issue numbers (legacy format)
 * This is a compatibility function that works with Map<string, number>
 * Format: #aw_XXXXXXXXXXXX -> #123
 * @param {string} text - The text to process
 * @param {Map<string, number>} tempIdMap - Map of temporary_id to issue number
 * @returns {string} Text with temporary IDs replaced with issue numbers
 */
function replaceTemporaryIdReferencesLegacy(text, tempIdMap) {
  return text.replace(TEMPORARY_ID_PATTERN, (match, tempId) => {
    const issueNumber = tempIdMap.get(normalizeTemporaryId(tempId));
    if (issueNumber !== undefined) {
      return `#${issueNumber}`;
    }
    // Return original if not found (it may be created later)
    return match;
  });
}

/**
 * Load the temporary ID map from environment variable
 * Supports both old format (temporary_id -> number) and new format (temporary_id -> {repo, number})
 * @returns {Map<string, RepoIssuePair>} Map of temporary_id to {repo, number}
 */
function loadTemporaryIdMap() {
  const mapJson = process.env.GH_AW_TEMPORARY_ID_MAP;
  if (!mapJson || mapJson === "{}") {
    return new Map();
  }
  try {
    const mapObject = JSON.parse(mapJson);
    /** @type {Map<string, RepoIssuePair>} */
    const result = new Map();

    for (const [key, value] of Object.entries(mapObject)) {
      const normalizedKey = normalizeTemporaryId(key);
      if (typeof value === "number") {
        // Legacy format: number only, use context repo
        const contextRepo = `${context.repo.owner}/${context.repo.repo}`;
        result.set(normalizedKey, { repo: contextRepo, number: value });
      } else if (typeof value === "object" && value !== null && "repo" in value && "number" in value) {
        // New format: {repo, number}
        result.set(normalizedKey, { repo: String(value.repo), number: Number(value.number) });
      }
    }
    return result;
  } catch (error) {
    if (typeof core !== "undefined") {
      core.warning(`Failed to parse temporary ID map: ${error instanceof Error ? error.message : String(error)}`);
    }
    return new Map();
  }
}

/**
 * Resolve an issue number that may be a temporary ID or an actual issue number
 * Returns structured result with the resolved number, repo, and metadata
 * @param {any} value - The value to resolve (can be temporary ID, number, or string)
 * @param {Map<string, RepoIssuePair>} temporaryIdMap - Map of temporary ID to {repo, number}
 * @returns {{resolved: RepoIssuePair|null, wasTemporaryId: boolean, errorMessage: string|null}}
 */
function resolveIssueNumber(value, temporaryIdMap) {
  if (value === undefined || value === null) {
    return { resolved: null, wasTemporaryId: false, errorMessage: "Issue number is missing" };
  }

  // Check if it's a temporary ID
  const valueStr = String(value);
  if (isTemporaryId(valueStr)) {
    const resolvedPair = temporaryIdMap.get(normalizeTemporaryId(valueStr));
    if (resolvedPair !== undefined) {
      return { resolved: resolvedPair, wasTemporaryId: true, errorMessage: null };
    }
    return {
      resolved: null,
      wasTemporaryId: true,
      errorMessage: `Temporary ID '${valueStr}' not found in map. Ensure the issue was created before linking.`,
    };
  }

  // It's a real issue number - use context repo as default
  const issueNumber = typeof value === "number" ? value : parseInt(valueStr, 10);
  if (isNaN(issueNumber) || issueNumber <= 0) {
    return { resolved: null, wasTemporaryId: false, errorMessage: `Invalid issue number: ${value}` };
  }

  const contextRepo = typeof context !== "undefined" ? `${context.repo.owner}/${context.repo.repo}` : "";
  return { resolved: { repo: contextRepo, number: issueNumber }, wasTemporaryId: false, errorMessage: null };
}

/**
 * Serialize the temporary ID map to JSON for output
 * @param {Map<string, RepoIssuePair>} tempIdMap - Map of temporary_id to {repo, number}
 * @returns {string} JSON string of the map
 */
function serializeTemporaryIdMap(tempIdMap) {
  const obj = Object.fromEntries(tempIdMap);
  return JSON.stringify(obj);
}

// === End of ./temporary_id.cjs ===

// === Inlined from ./repo_helpers.cjs ===
// @ts-check
/// <reference types="@actions/github-script" />

/**
 * Repository-related helper functions for safe-output scripts
 * Provides common repository parsing, validation, and resolution logic
 */

/**
 * Parse the allowed repos from environment variable
 * @returns {Set<string>} Set of allowed repository slugs
 */
function parseAllowedRepos() {
  const allowedReposEnv = process.env.GH_AW_ALLOWED_REPOS;
  const set = new Set();
  if (allowedReposEnv) {
    allowedReposEnv
      .split(",")
      .map(repo => repo.trim())
      .filter(repo => repo)
      .forEach(repo => set.add(repo));
  }
  return set;
}

/**
 * Get the default target repository
 * @returns {string} Repository slug in "owner/repo" format
 */
function getDefaultTargetRepo() {
  // First check if there's a target-repo override
  const targetRepoSlug = process.env.GH_AW_TARGET_REPO_SLUG;
  if (targetRepoSlug) {
    return targetRepoSlug;
  }
  // Fall back to context repo
  return `${context.repo.owner}/${context.repo.repo}`;
}

/**
 * Validate that a repo is allowed for operations
 * @param {string} repo - Repository slug to validate
 * @param {string} defaultRepo - Default target repository
 * @param {Set<string>} allowedRepos - Set of explicitly allowed repos
 * @returns {{valid: boolean, error: string|null}}
 */
function validateRepo(repo, defaultRepo, allowedRepos) {
  // Default repo is always allowed
  if (repo === defaultRepo) {
    return { valid: true, error: null };
  }
  // Check if it's in the allowed repos list
  if (allowedRepos.has(repo)) {
    return { valid: true, error: null };
  }
  return {
    valid: false,
    error: `Repository '${repo}' is not in the allowed-repos list. Allowed: ${defaultRepo}${allowedRepos.size > 0 ? ", " + Array.from(allowedRepos).join(", ") : ""}`,
  };
}

/**
 * Parse owner and repo from a repository slug
 * @param {string} repoSlug - Repository slug in "owner/repo" format
 * @returns {{owner: string, repo: string}|null}
 */
function parseRepoSlug(repoSlug) {
  const parts = repoSlug.split("/");
  if (parts.length !== 2 || !parts[0] || !parts[1]) {
    return null;
  }
  return { owner: parts[0], repo: parts[1] };
}

// === End of ./repo_helpers.cjs ===

// === Inlined from ./expiration_helpers.cjs ===
// @ts-check
/// <reference types="@actions/github-script" />

/**
 * Add expiration XML comment to body lines if expires is set
 * @param {string[]} bodyLines - Array of body lines to append to
 * @param {string} envVarName - Name of the environment variable containing expires days (e.g., "GH_AW_DISCUSSION_EXPIRES")
 * @param {string} entityType - Type of entity for logging (e.g., "Discussion", "Issue", "Pull Request")
 * @returns {void}
 */
function addExpirationComment(bodyLines, envVarName, entityType) {
  const expiresEnv = process.env[envVarName];
  if (expiresEnv) {
    const expiresDays = parseInt(expiresEnv, 10);
    if (!isNaN(expiresDays) && expiresDays > 0) {
      const expirationDate = new Date();
      expirationDate.setDate(expirationDate.getDate() + expiresDays);
      const expirationISO = expirationDate.toISOString();
      bodyLines.push(`<!-- gh-aw-expires: ${expirationISO} -->`);
      core.info(`${entityType} will expire on ${expirationISO} (${expiresDays} days)`);
    }
  }
}

// === End of ./expiration_helpers.cjs ===


/**
 * Fetch repository ID and discussion categories for a repository
 * @param {string} owner - Repository owner
 * @param {string} repo - Repository name
 * @returns {Promise<{repositoryId: string, discussionCategories: Array<{id: string, name: string, slug: string, description: string}>}|null>}
 */
async function fetchRepoDiscussionInfo(owner, repo) {
  const repositoryQuery = `
    query($owner: String!, $repo: String!) {
      repository(owner: $owner, name: $repo) {
        id
        discussionCategories(first: 20) {
          nodes {
            id
            name
            slug
            description
          }
        }
      }
    }
  `;
  const queryResult = await github.graphql(repositoryQuery, {
    owner: owner,
    repo: repo,
  });
  if (!queryResult || !queryResult.repository) {
    return null;
  }
  return {
    repositoryId: queryResult.repository.id,
    discussionCategories: queryResult.repository.discussionCategories.nodes || [],
  };
}

/**
 * Resolve category ID for a repository
 * @param {string} categoryConfig - Category ID, name, or slug from config
 * @param {string} itemCategory - Category from agent output item (optional)
 * @param {Array<{id: string, name: string, slug: string}>} categories - Available categories
 * @returns {{id: string, matchType: string, name: string, requestedCategory?: string}|undefined} Resolved category info
 */
function resolveCategoryId(categoryConfig, itemCategory, categories) {
  // Use item category if provided, otherwise use config
  const categoryToMatch = itemCategory || categoryConfig;

  if (categoryToMatch) {
    // Try to match against category IDs first
    const categoryById = categories.find(cat => cat.id === categoryToMatch);
    if (categoryById) {
      return { id: categoryById.id, matchType: "id", name: categoryById.name };
    }
    // Try to match against category names
    const categoryByName = categories.find(cat => cat.name === categoryToMatch);
    if (categoryByName) {
      return { id: categoryByName.id, matchType: "name", name: categoryByName.name };
    }
    // Try to match against category slugs (routes)
    const categoryBySlug = categories.find(cat => cat.slug === categoryToMatch);
    if (categoryBySlug) {
      return { id: categoryBySlug.id, matchType: "slug", name: categoryBySlug.name };
    }
  }

  // Fall back to first category if available
  if (categories.length > 0) {
    return {
      id: categories[0].id,
      matchType: "fallback",
      name: categories[0].name,
      requestedCategory: categoryToMatch,
    };
  }

  return undefined;
}

async function main() {
  // Initialize outputs to empty strings to ensure they're always set
  core.setOutput("discussion_number", "");
  core.setOutput("discussion_url", "");

  // Load the temporary ID map from create_issue job
  const temporaryIdMap = loadTemporaryIdMap();
  if (temporaryIdMap.size > 0) {
    core.info(`Loaded temporary ID map with ${temporaryIdMap.size} entries`);
  }

  const result = loadAgentOutput();
  if (!result.success) {
    return;
  }

  const createDiscussionItems = result.items.filter(item => item.type === "create_discussion");
  if (createDiscussionItems.length === 0) {
    core.warning("No create-discussion items found in agent output");
    return;
  }
  core.info(`Found ${createDiscussionItems.length} create-discussion item(s)`);

  // Parse allowed repos and default target
  const allowedRepos = parseAllowedRepos();
  const defaultTargetRepo = getDefaultTargetRepo();
  core.info(`Default target repo: ${defaultTargetRepo}`);
  if (allowedRepos.size > 0) {
    core.info(`Allowed repos: ${Array.from(allowedRepos).join(", ")}`);
  }

  if (process.env.GH_AW_SAFE_OUTPUTS_STAGED === "true") {
    let summaryContent = "## 🎭 Staged Mode: Create Discussions Preview\n\n";
    summaryContent += "The following discussions would be created if staged mode was disabled:\n\n";
    for (let i = 0; i < createDiscussionItems.length; i++) {
      const item = createDiscussionItems[i];
      summaryContent += `### Discussion ${i + 1}\n`;
      summaryContent += `**Title:** ${item.title || "No title provided"}\n\n`;
      if (item.repo) {
        summaryContent += `**Repository:** ${item.repo}\n\n`;
      }
      if (item.body) {
        summaryContent += `**Body:**\n${item.body}\n\n`;
      }
      if (item.category) {
        summaryContent += `**Category:** ${item.category}\n\n`;
      }
      summaryContent += "---\n\n";
    }
    await core.summary.addRaw(summaryContent).write();
    core.info("📝 Discussion creation preview written to step summary");
    return;
  }

  // Cache for repository info to avoid redundant API calls
  /** @type {Map<string, {repositoryId: string, discussionCategories: Array<{id: string, name: string, slug: string, description: string}>}>} */
  const repoInfoCache = new Map();

  // Get configuration for close-older-discussions
  const closeOlderEnabled = process.env.GH_AW_CLOSE_OLDER_DISCUSSIONS === "true";
  const titlePrefix = process.env.GH_AW_DISCUSSION_TITLE_PREFIX || "";
  const configCategory = process.env.GH_AW_DISCUSSION_CATEGORY || "";
  const labelsEnvVar = process.env.GH_AW_DISCUSSION_LABELS || "";
  const labels = labelsEnvVar
    ? labelsEnvVar
        .split(",")
        .map(l => l.trim())
        .filter(l => l.length > 0)
    : [];
  const workflowName = process.env.GH_AW_WORKFLOW_NAME || "Workflow";
  const runId = context.runId;
  const githubServer = process.env.GITHUB_SERVER_URL || "https://github.com";
  const runUrl = context.payload.repository
    ? `${context.payload.repository.html_url}/actions/runs/${runId}`
    : `${githubServer}/${context.repo.owner}/${context.repo.repo}/actions/runs/${runId}`;

  const createdDiscussions = [];
  const closedDiscussionsSummary = [];

  for (let i = 0; i < createDiscussionItems.length; i++) {
    const createDiscussionItem = createDiscussionItems[i];

    // Determine target repository for this discussion
    const itemRepo = createDiscussionItem.repo ? String(createDiscussionItem.repo).trim() : defaultTargetRepo;

    // Validate the repository is allowed
    const repoValidation = validateRepo(itemRepo, defaultTargetRepo, allowedRepos);
    if (!repoValidation.valid) {
      core.warning(`Skipping discussion: ${repoValidation.error}`);
      continue;
    }

    // Parse the repository slug
    const repoParts = parseRepoSlug(itemRepo);
    if (!repoParts) {
      core.warning(`Skipping discussion: Invalid repository format '${itemRepo}'. Expected 'owner/repo'.`);
      continue;
    }

    // Get repository info (cached)
    let repoInfo = repoInfoCache.get(itemRepo);
    if (!repoInfo) {
      try {
        const fetchedInfo = await fetchRepoDiscussionInfo(repoParts.owner, repoParts.repo);
        if (!fetchedInfo) {
          core.warning(`Skipping discussion: Failed to fetch repository information for '${itemRepo}'`);
          continue;
        }
        repoInfo = fetchedInfo;
        repoInfoCache.set(itemRepo, repoInfo);
        core.info(
          `Fetched discussion categories for ${itemRepo}: ${JSON.stringify(repoInfo.discussionCategories.map(cat => ({ name: cat.name, id: cat.id })))}`
        );
      } catch (error) {
        const errorMessage = error instanceof Error ? error.message : String(error);
        if (
          errorMessage.includes("Not Found") ||
          errorMessage.includes("not found") ||
          errorMessage.includes("Could not resolve to a Repository")
        ) {
          core.warning(`Skipping discussion: Discussions are not enabled for repository '${itemRepo}'`);
          continue;
        }
        core.error(`Failed to get discussion categories for ${itemRepo}: ${errorMessage}`);
        throw error;
      }
    }

    // Resolve category ID for this discussion
    const categoryInfo = resolveCategoryId(configCategory, createDiscussionItem.category, repoInfo.discussionCategories);
    if (!categoryInfo) {
      core.warning(`Skipping discussion in ${itemRepo}: No discussion category available`);
      continue;
    }

    // Log how the category was resolved
    if (categoryInfo.matchType === "name") {
      core.info(`Using category by name: ${categoryInfo.name} (${categoryInfo.id})`);
    } else if (categoryInfo.matchType === "slug") {
      core.info(`Using category by slug: ${categoryInfo.name} (${categoryInfo.id})`);
    } else if (categoryInfo.matchType === "fallback") {
      if (categoryInfo.requestedCategory) {
        const availableCategoryNames = repoInfo.discussionCategories.map(cat => cat.name).join(", ");
        core.warning(
          `Category "${categoryInfo.requestedCategory}" not found by ID, name, or slug. Available categories: ${availableCategoryNames}`
        );
        core.info(`Falling back to default category: ${categoryInfo.name} (${categoryInfo.id})`);
      } else {
        core.info(`Using default first category: ${categoryInfo.name} (${categoryInfo.id})`);
      }
    }

    const categoryId = categoryInfo.id;

    core.info(
      `Processing create-discussion item ${i + 1}/${createDiscussionItems.length}: title=${createDiscussionItem.title}, bodyLength=${createDiscussionItem.body?.length || 0}, repo=${itemRepo}`
    );

    // Replace temporary ID references in title
    let title = createDiscussionItem.title ? replaceTemporaryIdReferences(createDiscussionItem.title.trim(), temporaryIdMap, itemRepo) : "";
    // Replace temporary ID references in body (with defensive null check)
    const bodyText = createDiscussionItem.body || "";
    let bodyLines = replaceTemporaryIdReferences(bodyText, temporaryIdMap, itemRepo).split("\n");
    if (!title) {
      title = replaceTemporaryIdReferences(bodyText, temporaryIdMap, itemRepo) || "Agent Output";
    }
    if (titlePrefix && !title.startsWith(titlePrefix)) {
      title = titlePrefix + title;
    }

    // Add tracker-id comment if present
    const trackerIDComment = getTrackerID("markdown");
    if (trackerIDComment) {
      bodyLines.push(trackerIDComment);
    }

    // Add expiration comment if expires is set
    addExpirationComment(bodyLines, "GH_AW_DISCUSSION_EXPIRES", "Discussion");

    bodyLines.push(``, ``, `> AI generated by [${workflowName}](${runUrl})`, "");
    const body = bodyLines.join("\n").trim();
    core.info(`Creating discussion in ${itemRepo} with title: ${title}`);
    core.info(`Category ID: ${categoryId}`);
    core.info(`Body length: ${body.length}`);
    try {
      const createDiscussionMutation = `
        mutation($repositoryId: ID!, $categoryId: ID!, $title: String!, $body: String!) {
          createDiscussion(input: {
            repositoryId: $repositoryId,
            categoryId: $categoryId,
            title: $title,
            body: $body
          }) {
            discussion {
              id
              number
              title
              url
            }
          }
        }
      `;
      const mutationResult = await github.graphql(createDiscussionMutation, {
        repositoryId: repoInfo.repositoryId,
        categoryId: categoryId,
        title: title,
        body: body,
      });
      const discussion = mutationResult.createDiscussion.discussion;
      if (!discussion) {
        core.error(`Failed to create discussion in ${itemRepo}: No discussion data returned`);
        continue;
      }
      core.info(`Created discussion ${itemRepo}#${discussion.number}: ${discussion.url}`);
      createdDiscussions.push({ ...discussion, _repo: itemRepo });
      if (i === createDiscussionItems.length - 1) {
        core.setOutput("discussion_number", discussion.number);
        core.setOutput("discussion_url", discussion.url);
      }

      // Close older discussions if enabled and title prefix or labels are set
      // Note: close-older-discussions only works within the same repository
      const hasMatchingCriteria = titlePrefix || labels.length > 0;
      if (closeOlderEnabled && hasMatchingCriteria) {
        core.info("close-older-discussions is enabled, searching for older discussions to close...");
        try {
          const closedDiscussions = await closeOlderDiscussions(
            github,
            repoParts.owner,
            repoParts.repo,
            titlePrefix,
            labels,
            categoryId,
            { number: discussion.number, url: discussion.url },
            workflowName,
            runUrl
          );

          if (closedDiscussions.length > 0) {
            closedDiscussionsSummary.push(...closedDiscussions);
            core.info(`Closed ${closedDiscussions.length} older discussion(s) as outdated`);
          }
        } catch (closeError) {
          // Log error but don't fail the workflow - closing older discussions is a nice-to-have
          core.warning(`Failed to close older discussions: ${closeError instanceof Error ? closeError.message : String(closeError)}`);
        }
      } else if (closeOlderEnabled && !hasMatchingCriteria) {
        core.warning("close-older-discussions is enabled but no title-prefix or labels are set - skipping close older discussions");
      }
    } catch (error) {
      core.error(`✗ Failed to create discussion "${title}" in ${itemRepo}: ${error instanceof Error ? error.message : String(error)}`);
      throw error;
    }
  }
  if (createdDiscussions.length > 0) {
    let summaryContent = "\n\n## GitHub Discussions\n";
    for (const discussion of createdDiscussions) {
      const repoLabel = discussion._repo !== defaultTargetRepo ? ` (${discussion._repo})` : "";
      summaryContent += `- Discussion #${discussion.number}${repoLabel}: [${discussion.title}](${discussion.url})\n`;
    }

    // Add closed discussions to summary
    if (closedDiscussionsSummary.length > 0) {
      summaryContent += "\n### Closed Older Discussions\n";
      for (const closed of closedDiscussionsSummary) {
        summaryContent += `- Discussion #${closed.number}: [View](${closed.url}) (marked as outdated)\n`;
      }
    }

    await core.summary.addRaw(summaryContent).write();
  }
  core.info(`Successfully created ${createdDiscussions.length} discussion(s)`);
}
await main();
