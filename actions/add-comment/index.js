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

// === Inlined from ./messages_footer.cjs ===
// @ts-check
/// <reference types="@actions/github-script" />

/**
 * Footer Message Module
 *
 * This module provides footer and installation instructions generation
 * for safe-output workflows.
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
 * @typedef {Object} FooterContext
 * @property {string} workflowName - Name of the workflow
 * @property {string} runUrl - URL of the workflow run
 * @property {string} [workflowSource] - Source of the workflow (owner/repo/path@ref)
 * @property {string} [workflowSourceUrl] - GitHub URL for the workflow source
 * @property {number|string} [triggeringNumber] - Issue, PR, or discussion number that triggered this workflow
 */

/**
 * Get the footer message, using custom template if configured.
 * @param {FooterContext} ctx - Context for footer generation
 * @returns {string} Footer message
 */
function getFooterMessage(ctx) {
  const messages = getMessages();

  // Create context with both camelCase and snake_case keys
  const templateContext = toSnakeCase(ctx);

  // Default footer template - pirate themed! 🏴‍☠️
  const defaultFooter = "> Ahoy! This treasure was crafted by [🏴‍☠️ {workflow_name}]({run_url})";

  // Use custom footer if configured
  let footer = messages?.footer ? renderTemplate(messages.footer, templateContext) : renderTemplate(defaultFooter, templateContext);

  // Add triggering reference if available
  if (ctx.triggeringNumber) {
    footer += ` fer issue #{triggering_number} 🗺️`.replace("{triggering_number}", String(ctx.triggeringNumber));
  }

  return footer;
}

/**
 * Get the footer installation instructions, using custom template if configured.
 * @param {FooterContext} ctx - Context for footer generation
 * @returns {string} Footer installation message or empty string if no source
 */
function getFooterInstallMessage(ctx) {
  if (!ctx.workflowSource || !ctx.workflowSourceUrl) {
    return "";
  }

  const messages = getMessages();

  // Create context with both camelCase and snake_case keys
  const templateContext = toSnakeCase(ctx);

  // Default installation template - pirate themed! 🏴‍☠️
  const defaultInstall =
    "> Arr! To plunder this workflow fer yer own ship, run `gh aw add {workflow_source}`. Chart yer course at [🦜 {workflow_source_url}]({workflow_source_url})!";

  // Use custom installation message if configured
  return messages?.footerInstall
    ? renderTemplate(messages.footerInstall, templateContext)
    : renderTemplate(defaultInstall, templateContext);
}

/**
 * Generates an XML comment marker with agentic workflow metadata for traceability.
 * This marker enables searching and tracing back items generated by an agentic workflow.
 *
 * The marker format is:
 * <!-- agentic-workflow: workflow-name, engine: copilot, version: 1.0.0, model: gpt-5, run: https://github.com/... -->
 *
 * @param {string} workflowName - Name of the workflow
 * @param {string} runUrl - URL of the workflow run
 * @returns {string} XML comment marker with workflow metadata
 */
function generateXMLMarker(workflowName, runUrl) {
  // Read engine metadata from environment variables
  const engineId = process.env.GH_AW_ENGINE_ID || "";
  const engineVersion = process.env.GH_AW_ENGINE_VERSION || "";
  const engineModel = process.env.GH_AW_ENGINE_MODEL || "";
  const trackerId = process.env.GH_AW_TRACKER_ID || "";

  // Build the key-value pairs for the marker
  const parts = [];

  // Always include agentic-workflow name
  parts.push(`agentic-workflow: ${workflowName}`);

  // Add tracker-id if available (for searchability and tracing)
  if (trackerId) {
    parts.push(`tracker-id: ${trackerId}`);
  }

  // Add engine ID if available
  if (engineId) {
    parts.push(`engine: ${engineId}`);
  }

  // Add version if available
  if (engineVersion) {
    parts.push(`version: ${engineVersion}`);
  }

  // Add model if available
  if (engineModel) {
    parts.push(`model: ${engineModel}`);
  }

  // Always include run URL
  parts.push(`run: ${runUrl}`);

  // Return the XML comment marker
  return `<!-- ${parts.join(", ")} -->`;
}

/**
 * Generate the complete footer with AI attribution and optional installation instructions.
 * This is a drop-in replacement for the original generateFooter function.
 * @param {string} workflowName - Name of the workflow
 * @param {string} runUrl - URL of the workflow run
 * @param {string} workflowSource - Source of the workflow (owner/repo/path@ref)
 * @param {string} workflowSourceURL - GitHub URL for the workflow source
 * @param {number|undefined} triggeringIssueNumber - Issue number that triggered this workflow
 * @param {number|undefined} triggeringPRNumber - Pull request number that triggered this workflow
 * @param {number|undefined} triggeringDiscussionNumber - Discussion number that triggered this workflow
 * @returns {string} Complete footer text
 */
function generateFooterWithMessages(
  workflowName,
  runUrl,
  workflowSource,
  workflowSourceURL,
  triggeringIssueNumber,
  triggeringPRNumber,
  triggeringDiscussionNumber
) {
  // Determine triggering number (issue takes precedence, then PR, then discussion)
  let triggeringNumber;
  if (triggeringIssueNumber) {
    triggeringNumber = triggeringIssueNumber;
  } else if (triggeringPRNumber) {
    triggeringNumber = triggeringPRNumber;
  } else if (triggeringDiscussionNumber) {
    triggeringNumber = `discussion #${triggeringDiscussionNumber}`;
  }

  const ctx = {
    workflowName,
    runUrl,
    workflowSource,
    workflowSourceUrl: workflowSourceURL,
    triggeringNumber,
  };

  let footer = "\n\n" + getFooterMessage(ctx);

  // Add installation instructions if source is available
  const installMessage = getFooterInstallMessage(ctx);
  if (installMessage) {
    footer += "\n>\n" + installMessage;
  }

  // Add XML comment marker for traceability
  footer += "\n\n" + generateXMLMarker(workflowName, runUrl);

  footer += "\n";
  return footer;
}

// === End of ./messages_footer.cjs ===

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

// === Inlined from ./get_repository_url.cjs ===
// @ts-check
/// <reference types="@actions/github-script" />

/**
 * Get the repository URL for different purposes
 * This helper handles trial mode where target repository URLs are different from execution context
 * @returns {string} Repository URL
 */
function getRepositoryUrl() {
  // For trial mode, use target repository for issue/PR URLs but execution context for action runs
  const targetRepoSlug = process.env.GH_AW_TARGET_REPO_SLUG;

  if (targetRepoSlug) {
    // Use target repository for issue/PR URLs in trial mode
    const githubServer = process.env.GITHUB_SERVER_URL || "https://github.com";
    return `${githubServer}/${targetRepoSlug}`;
  } else if (context.payload.repository?.html_url) {
    // Use execution context repository (default behavior)
    return context.payload.repository.html_url;
  } else {
    // Final fallback for action runs when context repo is not available
    const githubServer = process.env.GITHUB_SERVER_URL || "https://github.com";
    return `${githubServer}/${context.repo.owner}/${context.repo.repo}`;
  }
}

// === End of ./get_repository_url.cjs ===

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


/**
 * Comment on a GitHub Discussion using GraphQL
 * @param {any} github - GitHub REST API instance
 * @param {string} owner - Repository owner
 * @param {string} repo - Repository name
 * @param {number} discussionNumber - Discussion number
 * @param {string} message - Comment body
 * @param {string|undefined} replyToId - Optional comment node ID to reply to (for threaded comments)
 * @returns {Promise<{id: string, html_url: string, discussion_url: string}>} Comment details
 */
async function commentOnDiscussion(github, owner, repo, discussionNumber, message, replyToId) {
  // 1. Retrieve discussion node ID
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

  const discussionId = repository.discussion.id;
  const discussionUrl = repository.discussion.url;

  // 2. Add comment (with optional replyToId for threading)
  let result;
  if (replyToId) {
    // Create a threaded reply to an existing comment
    result = await github.graphql(
      `
      mutation($dId: ID!, $body: String!, $replyToId: ID!) {
        addDiscussionComment(input: { discussionId: $dId, body: $body, replyToId: $replyToId }) {
          comment { 
            id 
            body 
            createdAt 
            url
          }
        }
      }`,
      { dId: discussionId, body: message, replyToId }
    );
  } else {
    // Create a top-level comment on the discussion
    result = await github.graphql(
      `
      mutation($dId: ID!, $body: String!) {
        addDiscussionComment(input: { discussionId: $dId, body: $body }) {
          comment { 
            id 
            body 
            createdAt 
            url
          }
        }
      }`,
      { dId: discussionId, body: message }
    );
  }

  const comment = result.addDiscussionComment.comment;

  return {
    id: comment.id,
    html_url: comment.url,
    discussion_url: discussionUrl,
  };
}

async function main() {
  // Check if we're in staged mode
  const isStaged = process.env.GH_AW_SAFE_OUTPUTS_STAGED === "true";
  const isDiscussionExplicit = process.env.GITHUB_AW_COMMENT_DISCUSSION === "true";

  // Load the temporary ID map from create_issue job
  const temporaryIdMap = loadTemporaryIdMap();
  if (temporaryIdMap.size > 0) {
    core.info(`Loaded temporary ID map with ${temporaryIdMap.size} entries`);
  }

  const result = loadAgentOutput();
  if (!result.success) {
    return;
  }

  // Find all add-comment items
  const commentItems = result.items.filter(/** @param {any} item */ item => item.type === "add_comment");
  if (commentItems.length === 0) {
    core.info("No add-comment items found in agent output");
    return;
  }

  core.info(`Found ${commentItems.length} add-comment item(s)`);

  // Helper function to get the target number (issue, discussion, or pull request)
  function getTargetNumber(item) {
    return item.item_number;
  }

  // Get the target configuration from environment variable
  const commentTarget = process.env.GH_AW_COMMENT_TARGET || "triggering";
  core.info(`Comment target configuration: ${commentTarget}`);

  // Check if we're in an issue, pull request, or discussion context
  const isIssueContext = context.eventName === "issues" || context.eventName === "issue_comment";
  const isPRContext =
    context.eventName === "pull_request" ||
    context.eventName === "pull_request_review" ||
    context.eventName === "pull_request_review_comment";
  const isDiscussionContext = context.eventName === "discussion" || context.eventName === "discussion_comment";
  const isDiscussion = isDiscussionContext || isDiscussionExplicit;

  // If in staged mode, emit step summary instead of creating comments
  if (isStaged) {
    let summaryContent = "## 🎭 Staged Mode: Add Comments Preview\n\n";
    summaryContent += "The following comments would be added if staged mode was disabled:\n\n";

    // Show created items references if available
    const createdIssueUrl = process.env.GH_AW_CREATED_ISSUE_URL;
    const createdIssueNumber = process.env.GH_AW_CREATED_ISSUE_NUMBER;
    const createdDiscussionUrl = process.env.GH_AW_CREATED_DISCUSSION_URL;
    const createdDiscussionNumber = process.env.GH_AW_CREATED_DISCUSSION_NUMBER;
    const createdPullRequestUrl = process.env.GH_AW_CREATED_PULL_REQUEST_URL;
    const createdPullRequestNumber = process.env.GH_AW_CREATED_PULL_REQUEST_NUMBER;

    if (createdIssueUrl || createdDiscussionUrl || createdPullRequestUrl) {
      summaryContent += "#### Related Items\n\n";
      if (createdIssueUrl && createdIssueNumber) {
        summaryContent += `- Issue: [#${createdIssueNumber}](${createdIssueUrl})\n`;
      }
      if (createdDiscussionUrl && createdDiscussionNumber) {
        summaryContent += `- Discussion: [#${createdDiscussionNumber}](${createdDiscussionUrl})\n`;
      }
      if (createdPullRequestUrl && createdPullRequestNumber) {
        summaryContent += `- Pull Request: [#${createdPullRequestNumber}](${createdPullRequestUrl})\n`;
      }
      summaryContent += "\n";
    }

    for (let i = 0; i < commentItems.length; i++) {
      const item = commentItems[i];
      summaryContent += `### Comment ${i + 1}\n`;
      const targetNumber = getTargetNumber(item);
      if (targetNumber) {
        const repoUrl = getRepositoryUrl();
        if (isDiscussion) {
          const discussionUrl = `${repoUrl}/discussions/${targetNumber}`;
          summaryContent += `**Target Discussion:** [#${targetNumber}](${discussionUrl})\n\n`;
        } else {
          const issueUrl = `${repoUrl}/issues/${targetNumber}`;
          summaryContent += `**Target Issue:** [#${targetNumber}](${issueUrl})\n\n`;
        }
      } else {
        if (isDiscussion) {
          summaryContent += `**Target:** Current discussion\n\n`;
        } else {
          summaryContent += `**Target:** Current issue/PR\n\n`;
        }
      }
      summaryContent += `**Body:**\n${item.body || "No content provided"}\n\n`;
      summaryContent += "---\n\n";
    }

    // Write to step summary
    await core.summary.addRaw(summaryContent).write();
    core.info("📝 Comment creation preview written to step summary");
    return;
  }

  // Validate context based on target configuration
  if (commentTarget === "triggering" && !isIssueContext && !isPRContext && !isDiscussionContext) {
    core.info('Target is "triggering" but not running in issue, pull request, or discussion context, skipping comment creation');
    return;
  }

  // Extract triggering context for footer generation
  const triggeringIssueNumber =
    context.payload?.issue?.number && !context.payload?.issue?.pull_request ? context.payload.issue.number : undefined;
  const triggeringPRNumber =
    context.payload?.pull_request?.number || (context.payload?.issue?.pull_request ? context.payload.issue.number : undefined);
  const triggeringDiscussionNumber = context.payload?.discussion?.number;

  const createdComments = [];

  // Process each comment item
  for (let i = 0; i < commentItems.length; i++) {
    const commentItem = commentItems[i];
    core.info(`Processing add-comment item ${i + 1}/${commentItems.length}: bodyLength=${commentItem.body.length}`);

    // Determine the issue/PR number and comment endpoint for this comment
    let itemNumber;
    let commentEndpoint;

    if (commentTarget === "*") {
      // For target "*", we need an explicit number from the comment item
      const targetNumber = getTargetNumber(commentItem);
      if (targetNumber) {
        itemNumber = parseInt(targetNumber, 10);
        if (isNaN(itemNumber) || itemNumber <= 0) {
          core.info(`Invalid target number specified: ${targetNumber}`);
          continue;
        }
        commentEndpoint = isDiscussion ? "discussions" : "issues";
      } else {
        core.info(`Target is "*" but no number specified in comment item`);
        continue;
      }
    } else if (commentTarget && commentTarget !== "triggering") {
      // Explicit number specified in target configuration
      itemNumber = parseInt(commentTarget, 10);
      if (isNaN(itemNumber) || itemNumber <= 0) {
        core.info(`Invalid target number in target configuration: ${commentTarget}`);
        continue;
      }
      commentEndpoint = isDiscussion ? "discussions" : "issues";
    } else {
      // Default behavior: use triggering issue/PR/discussion
      if (isIssueContext) {
        itemNumber = context.payload.issue?.number || context.payload.pull_request?.number || context.payload.discussion?.number;
        if (context.payload.issue) {
          commentEndpoint = "issues";
        } else {
          core.info("Issue context detected but no issue found in payload");
          continue;
        }
      } else if (isPRContext) {
        itemNumber = context.payload.pull_request?.number || context.payload.issue?.number || context.payload.discussion?.number;
        if (context.payload.pull_request) {
          commentEndpoint = "issues"; // PR comments use the issues API endpoint
        } else {
          core.info("Pull request context detected but no pull request found in payload");
          continue;
        }
      } else if (isDiscussionContext) {
        itemNumber = context.payload.discussion?.number || context.payload.issue?.number || context.payload.pull_request?.number;
        if (context.payload.discussion) {
          commentEndpoint = "discussions"; // Discussion comments use GraphQL via commentOnDiscussion
        } else {
          core.info("Discussion context detected but no discussion found in payload");
          continue;
        }
      }
    }

    if (!itemNumber) {
      core.info("Could not determine issue, pull request, or discussion number");
      continue;
    }

    // Extract body from the JSON item and replace temporary ID references
    let body = replaceTemporaryIdReferences(commentItem.body.trim(), temporaryIdMap);

    // Append references to created issues, discussions, and pull requests if they exist
    const createdIssueUrl = process.env.GH_AW_CREATED_ISSUE_URL;
    const createdIssueNumber = process.env.GH_AW_CREATED_ISSUE_NUMBER;
    const createdDiscussionUrl = process.env.GH_AW_CREATED_DISCUSSION_URL;
    const createdDiscussionNumber = process.env.GH_AW_CREATED_DISCUSSION_NUMBER;
    const createdPullRequestUrl = process.env.GH_AW_CREATED_PULL_REQUEST_URL;
    const createdPullRequestNumber = process.env.GH_AW_CREATED_PULL_REQUEST_NUMBER;

    // Add references section if any URLs are available
    let hasReferences = false;
    let referencesSection = "\n\n#### Related Items\n\n";

    if (createdIssueUrl && createdIssueNumber) {
      referencesSection += `- Issue: [#${createdIssueNumber}](${createdIssueUrl})\n`;
      hasReferences = true;
    }
    if (createdDiscussionUrl && createdDiscussionNumber) {
      referencesSection += `- Discussion: [#${createdDiscussionNumber}](${createdDiscussionUrl})\n`;
      hasReferences = true;
    }
    if (createdPullRequestUrl && createdPullRequestNumber) {
      referencesSection += `- Pull Request: [#${createdPullRequestNumber}](${createdPullRequestUrl})\n`;
      hasReferences = true;
    }

    if (hasReferences) {
      body += referencesSection;
    }

    // Add AI disclaimer with workflow name and run url
    const workflowName = process.env.GH_AW_WORKFLOW_NAME || "Workflow";
    const workflowSource = process.env.GH_AW_WORKFLOW_SOURCE || "";
    const workflowSourceURL = process.env.GH_AW_WORKFLOW_SOURCE_URL || "";
    const runId = context.runId;
    const githubServer = process.env.GITHUB_SERVER_URL || "https://github.com";
    const runUrl = context.payload.repository
      ? `${context.payload.repository.html_url}/actions/runs/${runId}`
      : `${githubServer}/${context.repo.owner}/${context.repo.repo}/actions/runs/${runId}`;

    // Add fingerprint comment if present
    body += getTrackerID("markdown");

    body += generateFooterWithMessages(
      workflowName,
      runUrl,
      workflowSource,
      workflowSourceURL,
      triggeringIssueNumber,
      triggeringPRNumber,
      triggeringDiscussionNumber
    );

    try {
      let comment;

      // Use GraphQL API for discussions, REST API for issues/PRs
      if (commentEndpoint === "discussions") {
        core.info(`Creating comment on discussion #${itemNumber}`);
        core.info(`Comment content length: ${body.length}`);

        // For discussion_comment events, extract the comment node_id to create a threaded reply
        let replyToId;
        if (context.eventName === "discussion_comment" && context.payload?.comment?.node_id) {
          replyToId = context.payload.comment.node_id;
          core.info(`Creating threaded reply to comment ${replyToId}`);
        }

        // Create discussion comment using GraphQL
        comment = await commentOnDiscussion(github, context.repo.owner, context.repo.repo, itemNumber, body, replyToId);
        core.info("Created discussion comment #" + comment.id + ": " + comment.html_url);

        // Add discussion_url to the comment object for consistency
        comment.discussion_url = comment.discussion_url;
      } else {
        core.info(`Creating comment on ${commentEndpoint} #${itemNumber}`);
        core.info(`Comment content length: ${body.length}`);

        // Create regular issue/PR comment using REST API
        const { data: restComment } = await github.rest.issues.createComment({
          owner: context.repo.owner,
          repo: context.repo.repo,
          issue_number: itemNumber,
          body: body,
        });

        comment = restComment;
        core.info("Created comment #" + comment.id + ": " + comment.html_url);
      }

      createdComments.push(comment);

      // Set output for the last created comment (for backward compatibility)
      if (i === commentItems.length - 1) {
        core.setOutput("comment_id", comment.id);
        core.setOutput("comment_url", comment.html_url);
      }
    } catch (error) {
      core.error(`✗ Failed to create comment: ${error instanceof Error ? error.message : String(error)}`);
      throw error;
    }
  }

  // Write summary for all created comments
  if (createdComments.length > 0) {
    let summaryContent = "\n\n## GitHub Comments\n";
    for (const comment of createdComments) {
      summaryContent += `- Comment #${comment.id}: [View Comment](${comment.html_url})\n`;
    }
    await core.summary.addRaw(summaryContent).write();
  }

  core.info(`Successfully created ${createdComments.length} comment(s)`);
  return createdComments;
}
await main();
