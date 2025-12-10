// @ts-check
/// <reference types="@actions/github-script" />

// === Inlined from ./sanitize_label_content.cjs ===
// @ts-check
/**
 * Sanitize label content for GitHub API
 * Removes control characters, ANSI codes, and neutralizes @mentions
 * @module sanitize_label_content
 */

/**
 * Sanitizes label content by removing control characters, ANSI escape codes,
 * and neutralizing @mentions to prevent unintended notifications.
 *
 * @param {string} content - The label content to sanitize
 * @returns {string} The sanitized label content
 */
function sanitizeLabelContent(content) {
  if (!content || typeof content !== "string") {
    return "";
  }
  let sanitized = content.trim();
  // Remove ANSI escape sequences FIRST (before removing control chars)
  sanitized = sanitized.replace(/\x1b\[[0-9;]*[mGKH]/g, "");
  // Then remove control characters (except newlines and tabs)
  sanitized = sanitized.replace(/[\x00-\x08\x0B\x0C\x0E-\x1F\x7F]/g, "");
  sanitized = sanitized.replace(
    /(^|[^\w`])@([A-Za-z0-9](?:[A-Za-z0-9-]{0,37}[A-Za-z0-9])?(?:\/[A-Za-z0-9._-]+)?)/g,
    (_m, p1, p2) => `${p1}\`@${p2}\``
  );
  sanitized = sanitized.replace(/[<>&'"]/g, "");
  return sanitized.trim();
}

// === End of ./sanitize_label_content.cjs ===

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

// === Inlined from ./staged_preview.cjs ===
// @ts-check
/// <reference types="@actions/github-script" />

/**
 * Generate a staged mode preview summary and write it to the step summary.
 *
 * @param {Object} options - Configuration options for the preview
 * @param {string} options.title - The main title for the preview (e.g., "Create Issues")
 * @param {string} options.description - Description of what would happen if staged mode was disabled
 * @param {Array<any>} options.items - Array of items to preview
 * @param {(item: any, index: number) => string} options.renderItem - Function to render each item as markdown
 * @returns {Promise<void>}
 */
async function generateStagedPreview(options) {
  const { title, description, items, renderItem } = options;

  let summaryContent = `## 🎭 Staged Mode: ${title} Preview\n\n`;
  summaryContent += `${description}\n\n`;

  for (let i = 0; i < items.length; i++) {
    const item = items[i];
    summaryContent += renderItem(item, i);
    summaryContent += "---\n\n";
  }

  try {
    await core.summary.addRaw(summaryContent).write();
    core.info(summaryContent);
    core.info(`📝 ${title} preview written to step summary`);
  } catch (error) {
    core.setFailed(error instanceof Error ? error : String(error));
  }
}

// === End of ./staged_preview.cjs ===

// === Inlined from ./generate_footer.cjs ===
// @ts-check
/// <reference types="@actions/github-script" />

/**
 * Generates an XML comment marker with agentic workflow metadata for traceability.
 * This marker enables searching and tracing back items generated by an agentic workflow.
 *
 * Note: This function is duplicated in messages_footer.cjs. While normally we would
 * consolidate to a shared module, importing messages_footer.cjs here would cause the
 * bundler to inline messages_core.cjs which contains 'GH_AW_SAFE_OUTPUT_MESSAGES:' in
 * a warning message, breaking tests that check for env var declarations.
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
 * Generate footer with AI attribution and workflow installation instructions
 * @param {string} workflowName - Name of the workflow
 * @param {string} runUrl - URL of the workflow run
 * @param {string} workflowSource - Source of the workflow (owner/repo/path@ref)
 * @param {string} workflowSourceURL - GitHub URL for the workflow source
 * @param {number|undefined} triggeringIssueNumber - Issue number that triggered this workflow
 * @param {number|undefined} triggeringPRNumber - Pull request number that triggered this workflow
 * @param {number|undefined} triggeringDiscussionNumber - Discussion number that triggered this workflow
 * @returns {string} Footer text
 */
function generateFooter(
  workflowName,
  runUrl,
  workflowSource,
  workflowSourceURL,
  triggeringIssueNumber,
  triggeringPRNumber,
  triggeringDiscussionNumber
) {
  let footer = `\n\n> AI generated by [${workflowName}](${runUrl})`;

  // Add reference to triggering issue/PR/discussion if available
  if (triggeringIssueNumber) {
    footer += ` for #${triggeringIssueNumber}`;
  } else if (triggeringPRNumber) {
    footer += ` for #${triggeringPRNumber}`;
  } else if (triggeringDiscussionNumber) {
    footer += ` for discussion #${triggeringDiscussionNumber}`;
  }

  if (workflowSource && workflowSourceURL) {
    footer += `\n>\n> To add this workflow in your repository, run \`gh aw add ${workflowSource}\`. See [usage guide](https://githubnext.github.io/gh-aw/tools/cli/).`;
  }

  // Add XML comment marker for traceability
  footer += "\n\n" + generateXMLMarker(workflowName, runUrl);

  footer += "\n";
  return footer;
}

// === End of ./generate_footer.cjs ===

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


async function main() {
  // Initialize outputs to empty strings to ensure they're always set
  core.setOutput("issue_number", "");
  core.setOutput("issue_url", "");
  core.setOutput("temporary_id_map", "{}");
  core.setOutput("issues_to_assign_copilot", "");

  const isStaged = process.env.GH_AW_SAFE_OUTPUTS_STAGED === "true";

  const result = loadAgentOutput();
  if (!result.success) {
    return;
  }

  const createIssueItems = result.items.filter(item => item.type === "create_issue");
  if (createIssueItems.length === 0) {
    core.info("No create-issue items found in agent output");
    return;
  }
  core.info(`Found ${createIssueItems.length} create-issue item(s)`);

  // Parse allowed repos and default target
  const allowedRepos = parseAllowedRepos();
  const defaultTargetRepo = getDefaultTargetRepo();
  core.info(`Default target repo: ${defaultTargetRepo}`);
  if (allowedRepos.size > 0) {
    core.info(`Allowed repos: ${Array.from(allowedRepos).join(", ")}`);
  }

  if (isStaged) {
    await generateStagedPreview({
      title: "Create Issues",
      description: "The following issues would be created if staged mode was disabled:",
      items: createIssueItems,
      renderItem: (item, index) => {
        let content = `### Issue ${index + 1}\n`;
        content += `**Title:** ${item.title || "No title provided"}\n\n`;
        if (item.temporary_id) {
          content += `**Temporary ID:** ${item.temporary_id}\n\n`;
        }
        if (item.repo) {
          content += `**Repository:** ${item.repo}\n\n`;
        }
        if (item.body) {
          content += `**Body:**\n${item.body}\n\n`;
        }
        if (item.labels && item.labels.length > 0) {
          content += `**Labels:** ${item.labels.join(", ")}\n\n`;
        }
        if (item.parent) {
          content += `**Parent:** ${item.parent}\n\n`;
        }
        return content;
      },
    });
    return;
  }
  const parentIssueNumber = context.payload?.issue?.number;

  // Map to track temporary_id -> {repo, number} relationships
  /** @type {Map<string, {repo: string, number: number}>} */
  const temporaryIdMap = new Map();

  // Extract triggering context for footer generation
  const triggeringIssueNumber =
    context.payload?.issue?.number && !context.payload?.issue?.pull_request ? context.payload.issue.number : undefined;
  const triggeringPRNumber =
    context.payload?.pull_request?.number || (context.payload?.issue?.pull_request ? context.payload.issue.number : undefined);
  const triggeringDiscussionNumber = context.payload?.discussion?.number;

  const labelsEnv = process.env.GH_AW_ISSUE_LABELS;
  let envLabels = labelsEnv
    ? labelsEnv
        .split(",")
        .map(label => label.trim())
        .filter(label => label)
    : [];
  const createdIssues = [];
  for (let i = 0; i < createIssueItems.length; i++) {
    const createIssueItem = createIssueItems[i];

    // Determine target repository for this issue
    const itemRepo = createIssueItem.repo ? String(createIssueItem.repo).trim() : defaultTargetRepo;

    // Validate the repository is allowed
    const repoValidation = validateRepo(itemRepo, defaultTargetRepo, allowedRepos);
    if (!repoValidation.valid) {
      core.warning(`Skipping issue: ${repoValidation.error}`);
      continue;
    }

    // Parse the repository slug
    const repoParts = parseRepoSlug(itemRepo);
    if (!repoParts) {
      core.warning(`Skipping issue: Invalid repository format '${itemRepo}'. Expected 'owner/repo'.`);
      continue;
    }

    // Get or generate the temporary ID for this issue
    const temporaryId = createIssueItem.temporary_id || generateTemporaryId();
    core.info(
      `Processing create-issue item ${i + 1}/${createIssueItems.length}: title=${createIssueItem.title}, bodyLength=${createIssueItem.body.length}, temporaryId=${temporaryId}, repo=${itemRepo}`
    );

    // Debug logging for parent field
    core.info(`Debug: createIssueItem.parent = ${JSON.stringify(createIssueItem.parent)}`);
    core.info(`Debug: parentIssueNumber from context = ${JSON.stringify(parentIssueNumber)}`);

    // Resolve parent: check if it's a temporary ID reference
    let effectiveParentIssueNumber;
    let effectiveParentRepo = itemRepo; // Default to same repo
    if (createIssueItem.parent !== undefined) {
      if (isTemporaryId(createIssueItem.parent)) {
        // It's a temporary ID, look it up in the map
        const resolvedParent = temporaryIdMap.get(normalizeTemporaryId(createIssueItem.parent));
        if (resolvedParent !== undefined) {
          effectiveParentIssueNumber = resolvedParent.number;
          effectiveParentRepo = resolvedParent.repo;
          core.info(`Resolved parent temporary ID '${createIssueItem.parent}' to ${effectiveParentRepo}#${effectiveParentIssueNumber}`);
        } else {
          core.warning(
            `Parent temporary ID '${createIssueItem.parent}' not found in map. Ensure parent issue is created before sub-issues.`
          );
          effectiveParentIssueNumber = undefined;
        }
      } else {
        // It's a real issue number
        effectiveParentIssueNumber = parseInt(String(createIssueItem.parent), 10);
        if (isNaN(effectiveParentIssueNumber)) {
          core.warning(`Invalid parent value: ${createIssueItem.parent}`);
          effectiveParentIssueNumber = undefined;
        }
      }
    } else {
      // Only use context parent if we're in the same repo as context
      const contextRepo = `${context.repo.owner}/${context.repo.repo}`;
      if (itemRepo === contextRepo) {
        effectiveParentIssueNumber = parentIssueNumber;
      }
    }
    core.info(
      `Debug: effectiveParentIssueNumber = ${JSON.stringify(effectiveParentIssueNumber)}, effectiveParentRepo = ${effectiveParentRepo}`
    );

    if (effectiveParentIssueNumber && createIssueItem.parent !== undefined) {
      core.info(`Using explicit parent issue number from item: ${effectiveParentRepo}#${effectiveParentIssueNumber}`);
    }
    let labels = [...envLabels];
    if (createIssueItem.labels && Array.isArray(createIssueItem.labels)) {
      labels = [...labels, ...createIssueItem.labels];
    }
    labels = labels
      .filter(label => !!label)
      .map(label => String(label).trim())
      .filter(label => label)
      .map(label => sanitizeLabelContent(label))
      .filter(label => label)
      .map(label => (label.length > 64 ? label.substring(0, 64) : label))
      .filter((label, index, arr) => arr.indexOf(label) === index);
    let title = createIssueItem.title ? createIssueItem.title.trim() : "";

    // Replace temporary ID references in the body using already-created issues
    let processedBody = replaceTemporaryIdReferences(createIssueItem.body, temporaryIdMap, itemRepo);
    let bodyLines = processedBody.split("\n");

    if (!title) {
      title = createIssueItem.body || "Agent Output";
    }
    const titlePrefix = process.env.GH_AW_ISSUE_TITLE_PREFIX;
    if (titlePrefix && !title.startsWith(titlePrefix)) {
      title = titlePrefix + title;
    }
    if (effectiveParentIssueNumber) {
      core.info("Detected issue context, parent issue " + effectiveParentRepo + "#" + effectiveParentIssueNumber);
      // Use full repo reference if cross-repo, short reference if same repo
      if (effectiveParentRepo === itemRepo) {
        bodyLines.push(`Related to #${effectiveParentIssueNumber}`);
      } else {
        bodyLines.push(`Related to ${effectiveParentRepo}#${effectiveParentIssueNumber}`);
      }
    }
    const workflowName = process.env.GH_AW_WORKFLOW_NAME || "Workflow";
    const workflowSource = process.env.GH_AW_WORKFLOW_SOURCE || "";
    const workflowSourceURL = process.env.GH_AW_WORKFLOW_SOURCE_URL || "";
    const runId = context.runId;
    const githubServer = process.env.GITHUB_SERVER_URL || "https://github.com";
    const runUrl = context.payload.repository
      ? `${context.payload.repository.html_url}/actions/runs/${runId}`
      : `${githubServer}/${context.repo.owner}/${context.repo.repo}/actions/runs/${runId}`;

    // Add tracker-id comment if present
    const trackerIDComment = getTrackerID("markdown");
    if (trackerIDComment) {
      bodyLines.push(trackerIDComment);
    }

    // Add expiration comment if expires is set
    addExpirationComment(bodyLines, "GH_AW_ISSUE_EXPIRES", "Issue");

    bodyLines.push(
      ``,
      ``,
      generateFooter(
        workflowName,
        runUrl,
        workflowSource,
        workflowSourceURL,
        triggeringIssueNumber,
        triggeringPRNumber,
        triggeringDiscussionNumber
      ).trimEnd(),
      ""
    );
    const body = bodyLines.join("\n").trim();
    core.info(`Creating issue in ${itemRepo} with title: ${title}`);
    core.info(`Labels: ${labels}`);
    core.info(`Body length: ${body.length}`);
    try {
      const { data: issue } = await github.rest.issues.create({
        owner: repoParts.owner,
        repo: repoParts.repo,
        title: title,
        body: body,
        labels: labels,
      });
      core.info(`Created issue ${itemRepo}#${issue.number}: ${issue.html_url}`);
      createdIssues.push({ ...issue, _repo: itemRepo });

      // Store the mapping of temporary_id -> {repo, number}
      temporaryIdMap.set(normalizeTemporaryId(temporaryId), { repo: itemRepo, number: issue.number });
      core.info(`Stored temporary ID mapping: ${temporaryId} -> ${itemRepo}#${issue.number}`);

      // Debug logging for sub-issue linking
      core.info(`Debug: About to check if sub-issue linking is needed. effectiveParentIssueNumber = ${effectiveParentIssueNumber}`);

      // Sub-issue linking only works within the same repository
      if (effectiveParentIssueNumber && effectiveParentRepo === itemRepo) {
        core.info(`Attempting to link issue #${issue.number} as sub-issue of #${effectiveParentIssueNumber}`);
        try {
          // First, get the node IDs for both parent and child issues
          core.info(`Fetching node ID for parent issue #${effectiveParentIssueNumber}...`);
          const getIssueNodeIdQuery = `
            query($owner: String!, $repo: String!, $issueNumber: Int!) {
              repository(owner: $owner, name: $repo) {
                issue(number: $issueNumber) {
                  id
                }
              }
            }
          `;

          // Get parent issue node ID
          const parentResult = await github.graphql(getIssueNodeIdQuery, {
            owner: repoParts.owner,
            repo: repoParts.repo,
            issueNumber: effectiveParentIssueNumber,
          });
          const parentNodeId = parentResult.repository.issue.id;
          core.info(`Parent issue node ID: ${parentNodeId}`);

          // Get child issue node ID
          core.info(`Fetching node ID for child issue #${issue.number}...`);
          const childResult = await github.graphql(getIssueNodeIdQuery, {
            owner: repoParts.owner,
            repo: repoParts.repo,
            issueNumber: issue.number,
          });
          const childNodeId = childResult.repository.issue.id;
          core.info(`Child issue node ID: ${childNodeId}`);

          // Link the child issue as a sub-issue of the parent
          core.info(`Executing addSubIssue mutation...`);
          const addSubIssueMutation = `
            mutation($issueId: ID!, $subIssueId: ID!) {
              addSubIssue(input: {
                issueId: $issueId,
                subIssueId: $subIssueId
              }) {
                subIssue {
                  id
                  number
                }
              }
            }
          `;

          await github.graphql(addSubIssueMutation, {
            issueId: parentNodeId,
            subIssueId: childNodeId,
          });

          core.info("✓ Successfully linked issue #" + issue.number + " as sub-issue of #" + effectiveParentIssueNumber);
        } catch (error) {
          core.info(`Warning: Could not link sub-issue to parent: ${error instanceof Error ? error.message : String(error)}`);
          core.info(`Error details: ${error instanceof Error ? error.stack : String(error)}`);
          // Fallback: add a comment if sub-issue linking fails
          try {
            core.info(`Attempting fallback: adding comment to parent issue #${effectiveParentIssueNumber}...`);
            await github.rest.issues.createComment({
              owner: repoParts.owner,
              repo: repoParts.repo,
              issue_number: effectiveParentIssueNumber,
              body: `Created related issue: #${issue.number}`,
            });
            core.info("✓ Added comment to parent issue #" + effectiveParentIssueNumber + " (sub-issue linking not available)");
          } catch (commentError) {
            core.info(
              `Warning: Could not add comment to parent issue: ${commentError instanceof Error ? commentError.message : String(commentError)}`
            );
          }
        }
      } else if (effectiveParentIssueNumber && effectiveParentRepo !== itemRepo) {
        core.info(`Skipping sub-issue linking: parent is in different repository (${effectiveParentRepo})`);
      } else {
        core.info(`Debug: No parent issue number set, skipping sub-issue linking`);
      }
      if (i === createIssueItems.length - 1) {
        core.setOutput("issue_number", issue.number);
        core.setOutput("issue_url", issue.html_url);
      }
    } catch (error) {
      const errorMessage = error instanceof Error ? error.message : String(error);
      if (errorMessage.includes("Issues has been disabled in this repository")) {
        core.info(`⚠ Cannot create issue "${title}" in ${itemRepo}: Issues are disabled for this repository`);
        core.info("Consider enabling issues in repository settings if you want to create issues automatically");
        continue;
      }
      core.error(`✗ Failed to create issue "${title}" in ${itemRepo}: ${errorMessage}`);
      throw error;
    }
  }
  if (createdIssues.length > 0) {
    let summaryContent = "\n\n## GitHub Issues\n";
    for (const issue of createdIssues) {
      const repoLabel = issue._repo !== defaultTargetRepo ? ` (${issue._repo})` : "";
      summaryContent += `- Issue #${issue.number}${repoLabel}: [${issue.title}](${issue.html_url})\n`;
    }
    await core.summary.addRaw(summaryContent).write();
  }

  // Output the temporary ID map as JSON for use by downstream jobs
  const tempIdMapOutput = serializeTemporaryIdMap(temporaryIdMap);
  core.setOutput("temporary_id_map", tempIdMapOutput);
  core.info(`Temporary ID map: ${tempIdMapOutput}`);

  // Output issues that need copilot assignment for assign_to_agent job
  // This is used when create-issue has assignees: [copilot]
  const assignCopilot = process.env.GH_AW_ASSIGN_COPILOT === "true";
  if (assignCopilot && createdIssues.length > 0) {
    // Format: repo:number for each issue (for cross-repo support)
    const issuesToAssign = createdIssues.map(issue => `${issue._repo}:${issue.number}`).join(",");
    core.setOutput("issues_to_assign_copilot", issuesToAssign);
    core.info(`Issues to assign copilot: ${issuesToAssign}`);
  }

  core.info(`Successfully created ${createdIssues.length} issue(s)`);
}
(async () => {
  await main();
})();
