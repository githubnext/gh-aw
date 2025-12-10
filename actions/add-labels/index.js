// @ts-check
/// <reference types="@actions/github-script" />

// === Inlined from ./safe_output_processor.cjs ===
// @ts-check
/// <reference types="@actions/github-script" />

/**
 * Shared processor for safe-output scripts
 * Provides common pipeline: load agent output, handle staged mode, parse config, resolve target
 */

// === Inlined from ./load_agent_output.cjs ===
// @ts-check
/// <reference types="@actions/github-script" />

const fs = require("fs");

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

// === Inlined from ./safe_output_helpers.cjs ===
// @ts-check
/// <reference types="@actions/github-script" />

/**
 * Shared helper functions for safe-output scripts
 * Provides common validation and target resolution logic
 */

/**
 * Parse a comma-separated list of allowed items from environment variable
 * @param {string|undefined} envValue - Environment variable value
 * @returns {string[]|undefined} Array of allowed items, or undefined if no restrictions
 */
function parseAllowedItems(envValue) {
  const trimmed = envValue?.trim();
  if (!trimmed) {
    return undefined;
  }
  return trimmed
    .split(",")
    .map(item => item.trim())
    .filter(item => item);
}

/**
 * Parse and validate max count from environment variable
 * @param {string|undefined} envValue - Environment variable value
 * @param {number} defaultValue - Default value if not specified
 * @returns {{valid: true, value: number} | {valid: false, error: string}} Validation result
 */
function parseMaxCount(envValue, defaultValue = 3) {
  if (!envValue) {
    return { valid: true, value: defaultValue };
  }

  const parsed = parseInt(envValue, 10);
  if (isNaN(parsed) || parsed < 1) {
    return {
      valid: false,
      error: `Invalid max value: ${envValue}. Must be a positive integer`,
    };
  }

  return { valid: true, value: parsed };
}

/**
 * Resolve the target number (issue/PR) based on configuration and context
 * @param {Object} params - Resolution parameters
 * @param {string} params.targetConfig - Target configuration ("triggering", "*", or explicit number)
 * @param {any} params.item - Safe output item with optional item_number or pull_request_number
 * @param {any} params.context - GitHub Actions context
 * @param {string} params.itemType - Type of item being processed (for error messages)
 * @param {boolean} params.supportsPR - Whether this safe output supports PR context
 * @returns {{success: true, number: number, contextType: string} | {success: false, error: string, shouldFail: boolean}} Resolution result
 */
function resolveTarget(params) {
  const { targetConfig, item, context, itemType, supportsPR = false } = params;

  // Check context type
  const isIssueContext = context.eventName === "issues" || context.eventName === "issue_comment";
  const isPRContext =
    context.eventName === "pull_request" ||
    context.eventName === "pull_request_review" ||
    context.eventName === "pull_request_review_comment";

  // Default target is "triggering"
  const target = targetConfig || "triggering";

  // Validate context for triggering mode
  if (target === "triggering") {
    if (supportsPR) {
      if (!isIssueContext && !isPRContext) {
        return {
          success: false,
          error: `Target is "triggering" but not running in issue or pull request context, skipping ${itemType}`,
          shouldFail: false, // Just skip, don't fail the workflow
        };
      }
    } else {
      if (!isPRContext) {
        return {
          success: false,
          error: `Target is "triggering" but not running in pull request context, skipping ${itemType}`,
          shouldFail: false, // Just skip, don't fail the workflow
        };
      }
    }
  }

  // Resolve target number
  let itemNumber;
  let contextType;

  if (target === "*") {
    // Use item_number, issue_number, or pull_request_number from item
    const numberField = supportsPR ? item.item_number || item.issue_number || item.pull_request_number : item.pull_request_number;

    if (numberField) {
      itemNumber = typeof numberField === "number" ? numberField : parseInt(String(numberField), 10);
      if (isNaN(itemNumber) || itemNumber <= 0) {
        return {
          success: false,
          error: `Invalid ${supportsPR ? "item_number/issue_number/pull_request_number" : "pull_request_number"} specified: ${numberField}`,
          shouldFail: true,
        };
      }
      contextType = supportsPR && (item.item_number || item.issue_number) ? "issue" : "pull request";
    } else {
      return {
        success: false,
        error: `Target is "*" but no ${supportsPR ? "item_number/issue_number" : "pull_request_number"} specified in ${itemType} item`,
        shouldFail: true,
      };
    }
  } else if (target !== "triggering") {
    // Explicit number
    itemNumber = parseInt(target, 10);
    if (isNaN(itemNumber) || itemNumber <= 0) {
      return {
        success: false,
        error: `Invalid ${supportsPR ? "issue" : "pull request"} number in target configuration: ${target}`,
        shouldFail: true,
      };
    }
    contextType = supportsPR ? "issue" : "pull request";
  } else {
    // Use triggering context
    if (isIssueContext) {
      if (context.payload.issue) {
        itemNumber = context.payload.issue.number;
        contextType = "issue";
      } else {
        return {
          success: false,
          error: "Issue context detected but no issue found in payload",
          shouldFail: true,
        };
      }
    } else if (isPRContext) {
      if (context.payload.pull_request) {
        itemNumber = context.payload.pull_request.number;
        contextType = "pull request";
      } else {
        return {
          success: false,
          error: "Pull request context detected but no pull request found in payload",
          shouldFail: true,
        };
      }
    }
  }

  if (!itemNumber) {
    return {
      success: false,
      error: `Could not determine ${supportsPR ? "issue or pull request" : "pull request"} number`,
      shouldFail: true,
    };
  }

  return {
    success: true,
    number: itemNumber,
    contextType: contextType || (supportsPR ? "issue" : "pull request"),
  };
}

// === End of ./safe_output_helpers.cjs ===

// === Inlined from ./safe_output_validator.cjs ===
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


/**
 * Load and parse the safe outputs configuration from config.json
 * @returns {object} The parsed configuration object
 */
function loadSafeOutputsConfig() {
  const configPath = "/tmp/gh-aw/safeoutputs/config.json";
  try {
    if (!fs.existsSync(configPath)) {
      core.warning(`Config file not found at ${configPath}, using defaults`);
      return {};
    }
    const configContent = fs.readFileSync(configPath, "utf8");
    return JSON.parse(configContent);
  } catch (error) {
    core.warning(`Failed to load config: ${error instanceof Error ? error.message : String(error)}`);
    return {};
  }
}

/**
 * Get configuration for a specific safe output type
 * @param {string} outputType - The type of safe output (e.g., "add_labels", "update_issue")
 * @returns {{max?: number, target?: string, allowed?: string[]}} The configuration for this output type
 */
function getSafeOutputConfig(outputType) {
  const config = loadSafeOutputsConfig();
  return config[outputType] || {};
}

/**
 * Validate and sanitize a title string
 * @param {any} title - The title to validate
 * @param {string} fieldName - The name of the field for error messages (default: "title")
 * @returns {{valid: boolean, value?: string, error?: string}} Validation result
 */
function validateTitle(title, fieldName = "title") {
  if (title === undefined || title === null) {
    return { valid: false, error: `${fieldName} is required` };
  }

  if (typeof title !== "string") {
    return { valid: false, error: `${fieldName} must be a string` };
  }

  const trimmed = title.trim();
  if (trimmed.length === 0) {
    return { valid: false, error: `${fieldName} cannot be empty` };
  }

  return { valid: true, value: trimmed };
}

/**
 * Validate and sanitize a body/content string
 * @param {any} body - The body to validate
 * @param {string} fieldName - The name of the field for error messages (default: "body")
 * @param {boolean} required - Whether the body is required (default: false)
 * @returns {{valid: boolean, value?: string, error?: string}} Validation result
 */
function validateBody(body, fieldName = "body", required = false) {
  if (body === undefined || body === null) {
    if (required) {
      return { valid: false, error: `${fieldName} is required` };
    }
    return { valid: true, value: "" };
  }

  if (typeof body !== "string") {
    return { valid: false, error: `${fieldName} must be a string` };
  }

  return { valid: true, value: body };
}

/**
 * Validate and sanitize an array of labels
 * @param {any} labels - The labels to validate
 * @param {string[]|undefined} allowedLabels - Optional list of allowed labels
 * @param {number} maxCount - Maximum number of labels allowed
 * @returns {{valid: boolean, value?: string[], error?: string}} Validation result
 */
function validateLabels(labels, allowedLabels = undefined, maxCount = 3) {
  if (!labels || !Array.isArray(labels)) {
    return { valid: false, error: "labels must be an array" };
  }

  // Check for removal attempts (labels starting with '-')
  for (const label of labels) {
    if (label && typeof label === "string" && label.startsWith("-")) {
      return { valid: false, error: `Label removal is not permitted. Found line starting with '-': ${label}` };
    }
  }

  // Filter labels based on allowed list if provided
  let validLabels = labels;
  if (allowedLabels && allowedLabels.length > 0) {
    validLabels = labels.filter(label => allowedLabels.includes(label));
  }

  // Sanitize and deduplicate labels
  const uniqueLabels = validLabels
    .filter(label => label != null && label !== false && label !== 0)
    .map(label => String(label).trim())
    .filter(label => label)
    .map(label => sanitizeLabelContent(label))
    .filter(label => label)
    .map(label => (label.length > 64 ? label.substring(0, 64) : label))
    .filter((label, index, arr) => arr.indexOf(label) === index);

  // Apply max count limit
  if (uniqueLabels.length > maxCount) {
    core.info(`Too many labels (${uniqueLabels.length}), limiting to ${maxCount}`);
    return { valid: true, value: uniqueLabels.slice(0, maxCount) };
  }

  if (uniqueLabels.length === 0) {
    return { valid: false, error: "No valid labels found after sanitization" };
  }

  return { valid: true, value: uniqueLabels };
}

/**
 * Validate max count from environment variable with config fallback
 * @param {string|undefined} envValue - Environment variable value
 * @param {number|undefined} configDefault - Default from config.json
 * @param {number} [fallbackDefault] - Fallback default for testing (optional, defaults to 1)
 * @returns {{valid: true, value: number} | {valid: false, error: string}} Validation result
 */
function validateMaxCount(envValue, configDefault, fallbackDefault = 1) {
  // Priority: env var > config.json > fallback default
  // In production, config.json should always have the default
  // Fallback is provided for backward compatibility and testing
  const defaultValue = configDefault !== undefined ? configDefault : fallbackDefault;

  if (!envValue) {
    return { valid: true, value: defaultValue };
  }

  const parsed = parseInt(envValue, 10);
  if (isNaN(parsed) || parsed < 1) {
    return {
      valid: false,
      error: `Invalid max value: ${envValue}. Must be a positive integer`,
    };
  }

  return { valid: true, value: parsed };
}

// === End of ./safe_output_validator.cjs ===


/**
 * @typedef {Object} ProcessorConfig
 * @property {string} itemType - The type field value to match in agent output (e.g., "add_labels")
 * @property {string} configKey - The key to use when reading from config.json (e.g., "add_labels")
 * @property {string} displayName - Human-readable name for logging (e.g., "Add Labels")
 * @property {string} itemTypeName - Name used in error messages (e.g., "label addition")
 * @property {boolean} [supportsPR] - When true, allows both issue AND PR contexts; when false, only PR context (default: false)
 * @property {boolean} [supportsIssue] - When true, passes supportsPR=true to resolveTarget to enable both contexts (default: false)
 * @property {boolean} [findMultiple] - Whether to find multiple items instead of just one (default: false)
 * @property {Object} envVars - Environment variable names
 * @property {string} [envVars.allowed] - Env var for allowed items list
 * @property {string} [envVars.maxCount] - Env var for max count
 * @property {string} [envVars.target] - Env var for target configuration
 */

/**
 * @typedef {Object} ProcessorResult
 * @property {boolean} success - Whether processing should continue
 * @property {any} [item] - The found item (when findMultiple is false)
 * @property {any[]} [items] - The found items (when findMultiple is true)
 * @property {Object} [config] - Parsed configuration
 * @property {string[]|undefined} [config.allowed] - Allowed items list
 * @property {number} [config.maxCount] - Maximum count
 * @property {string} [config.target] - Target configuration
 * @property {Object} [targetResult] - Result from resolveTarget (when findMultiple is false)
 * @property {number} [targetResult.number] - Target issue/PR number
 * @property {string} [targetResult.contextType] - Type of context (issue or pull request)
 * @property {string} [reason] - Reason why processing should not continue
 */

/**
 * Process the initial steps common to safe-output scripts:
 * 1. Load agent output
 * 2. Find matching item(s)
 * 3. Handle staged mode
 * 4. Parse configuration
 * 5. Resolve target (for single-item processors)
 *
 * @param {ProcessorConfig} config - Processor configuration
 * @param {Object} stagedPreviewOptions - Options for staged preview
 * @param {string} stagedPreviewOptions.title - Title for staged preview
 * @param {string} stagedPreviewOptions.description - Description for staged preview
 * @param {(item: any, index: number) => string} stagedPreviewOptions.renderItem - Function to render item in preview
 * @returns {Promise<ProcessorResult>} Processing result
 */
async function processSafeOutput(config, stagedPreviewOptions) {
  const {
    itemType,
    configKey,
    displayName,
    itemTypeName,
    supportsPR = false,
    supportsIssue = false,
    findMultiple = false,
    envVars,
  } = config;

  // Step 1: Load agent output
  const result = loadAgentOutput();
  if (!result.success) {
    return { success: false, reason: "Agent output not available" };
  }

  // Step 2: Find matching item(s)
  let items;
  if (findMultiple) {
    items = result.items.filter(item => item.type === itemType);
    if (items.length === 0) {
      core.info(`No ${itemType} items found in agent output`);
      return { success: false, reason: `No ${itemType} items found` };
    }
    core.info(`Found ${items.length} ${itemType} item(s)`);
  } else {
    const item = result.items.find(item => item.type === itemType);
    if (!item) {
      core.warning(`No ${itemType.replace(/_/g, "-")} item found in agent output`);
      return { success: false, reason: `No ${itemType} item found` };
    }
    items = [item];
    // Log item details based on common fields
    const itemDetails = getItemDetails(item);
    if (itemDetails) {
      core.info(`Found ${itemType.replace(/_/g, "-")} item with ${itemDetails}`);
    }
  }

  // Step 3: Handle staged mode
  if (process.env.GH_AW_SAFE_OUTPUTS_STAGED === "true") {
    await generateStagedPreview({
      title: stagedPreviewOptions.title,
      description: stagedPreviewOptions.description,
      items: items,
      renderItem: stagedPreviewOptions.renderItem,
    });
    return { success: false, reason: "Staged mode - preview generated" };
  }

  // Step 4: Parse configuration
  const safeOutputConfig = getSafeOutputConfig(configKey);

  // Parse allowed items (from env or config)
  const allowedEnvValue = envVars.allowed ? process.env[envVars.allowed] : undefined;
  const allowed = parseAllowedItems(allowedEnvValue) || safeOutputConfig.allowed;
  if (allowed) {
    core.info(`Allowed ${itemTypeName}s: ${JSON.stringify(allowed)}`);
  } else {
    core.info(`No ${itemTypeName} restrictions - any ${itemTypeName}s are allowed`);
  }

  // Parse max count (env takes priority, then config)
  const maxCountEnvValue = envVars.maxCount ? process.env[envVars.maxCount] : undefined;
  const maxCountResult = validateMaxCount(maxCountEnvValue, safeOutputConfig.max);
  if (!maxCountResult.valid) {
    core.setFailed(maxCountResult.error);
    return { success: false, reason: "Invalid max count configuration" };
  }
  const maxCount = maxCountResult.value;
  core.info(`Max count: ${maxCount}`);

  // Get target configuration
  const target = envVars.target ? process.env[envVars.target] || "triggering" : "triggering";
  core.info(`${displayName} target configuration: ${target}`);

  // For multiple items, return early without target resolution
  if (findMultiple) {
    return {
      success: true,
      items: items,
      config: {
        allowed,
        maxCount,
        target,
      },
    };
  }

  // Step 5: Resolve target (for single-item processors)
  const item = items[0];
  const targetResult = resolveTarget({
    targetConfig: target,
    item: item,
    context,
    itemType: itemTypeName,
    // supportsPR in resolveTarget: true=both issue and PR contexts, false=PR-only
    // If supportsIssue is true, we pass supportsPR=true to enable both contexts
    supportsPR: supportsPR || supportsIssue,
  });

  if (!targetResult.success) {
    if (targetResult.shouldFail) {
      core.setFailed(targetResult.error);
    } else {
      core.info(targetResult.error);
    }
    return { success: false, reason: targetResult.error };
  }

  return {
    success: true,
    item: item,
    config: {
      allowed,
      maxCount,
      target,
    },
    targetResult: {
      number: targetResult.number,
      contextType: targetResult.contextType,
    },
  };
}

/**
 * Get a description of item details for logging
 * @param {any} item - The safe output item
 * @returns {string|null} Description string or null
 */
function getItemDetails(item) {
  if (item.labels && Array.isArray(item.labels)) {
    return `${item.labels.length} labels`;
  }
  if (item.reviewers && Array.isArray(item.reviewers)) {
    return `${item.reviewers.length} reviewers`;
  }
  return null;
}

/**
 * Sanitize and deduplicate an array of string items
 * @param {any[]} items - Raw items array
 * @returns {string[]} Sanitized and deduplicated array
 */
function sanitizeItems(items) {
  return items
    .filter(item => item != null && item !== false && item !== 0)
    .map(item => String(item).trim())
    .filter(item => item)
    .filter((item, index, arr) => arr.indexOf(item) === index);
}

/**
 * Filter items by allowed list
 * @param {string[]} items - Items to filter
 * @param {string[]|undefined} allowed - Allowed items list (undefined means all allowed)
 * @returns {string[]} Filtered items
 */
function filterByAllowed(items, allowed) {
  if (!allowed || allowed.length === 0) {
    return items;
  }
  return items.filter(item => allowed.includes(item));
}

/**
 * Limit items to max count
 * @param {string[]} items - Items to limit
 * @param {number} maxCount - Maximum number of items
 * @returns {string[]} Limited items
 */
function limitToMaxCount(items, maxCount) {
  if (items.length > maxCount) {
    core.info(`Too many items (${items.length}), limiting to ${maxCount}`);
    return items.slice(0, maxCount);
  }
  return items;
}

/**
 * Process items through the standard pipeline: filter by allowed, sanitize, dedupe, limit
 * @param {any[]} rawItems - Raw items array from agent output
 * @param {string[]|undefined} allowed - Allowed items list
 * @param {number} maxCount - Maximum number of items
 * @returns {string[]} Processed items
 */
function processItems(rawItems, allowed, maxCount) {
  // Filter by allowed list first
  const filtered = filterByAllowed(rawItems, allowed);

  // Sanitize and deduplicate
  const sanitized = sanitizeItems(filtered);

  // Limit to max count
  return limitToMaxCount(sanitized, maxCount);
}

// === End of ./safe_output_processor.cjs ===

// Already inlined: ./safe_output_validator.cjs


async function main() {
  // Use shared processor for common steps
  const result = await processSafeOutput(
    {
      itemType: "add_labels",
      configKey: "add_labels",
      displayName: "Labels",
      itemTypeName: "label addition",
      supportsPR: true,
      supportsIssue: true,
      envVars: {
        allowed: "GH_AW_LABELS_ALLOWED",
        maxCount: "GH_AW_LABELS_MAX_COUNT",
        target: "GH_AW_LABELS_TARGET",
      },
    },
    {
      title: "Add Labels",
      description: "The following labels would be added if staged mode was disabled:",
      renderItem: item => {
        let content = "";
        if (item.item_number) {
          content += `**Target Issue:** #${item.item_number}\n\n`;
        } else {
          content += `**Target:** Current issue/PR\n\n`;
        }
        if (item.labels && item.labels.length > 0) {
          content += `**Labels to add:** ${item.labels.join(", ")}\n\n`;
        }
        return content;
      },
    }
  );

  if (!result.success) {
    return;
  }

  // @ts-ignore - TypeScript doesn't narrow properly after success check
  const { item: labelsItem, config, targetResult } = result;
  if (!config || !targetResult || targetResult.number === undefined) {
    core.setFailed("Internal error: config, targetResult, or targetResult.number is undefined");
    return;
  }
  const { allowed: allowedLabels, maxCount } = config;
  const itemNumber = targetResult.number;
  const { contextType } = targetResult;

  const requestedLabels = labelsItem.labels || [];
  core.info(`Requested labels: ${JSON.stringify(requestedLabels)}`);

  // Use validation helper to sanitize and validate labels
  const labelsResult = validateLabels(requestedLabels, allowedLabels, maxCount);
  if (!labelsResult.valid) {
    // If no valid labels, log info and return gracefully instead of failing
    if (labelsResult.error && labelsResult.error.includes("No valid labels")) {
      core.info("No labels to add");
      core.setOutput("labels_added", "");
      await core.summary
        .addRaw(
          `
## Label Addition

No labels were added (no valid labels found in agent output).
`
        )
        .write();
      return;
    }
    // For other validation errors, fail the workflow
    core.setFailed(labelsResult.error || "Invalid labels");
    return;
  }

  const uniqueLabels = labelsResult.value || [];

  if (uniqueLabels.length === 0) {
    core.info("No labels to add");
    core.setOutput("labels_added", "");
    await core.summary
      .addRaw(
        `
## Label Addition

No labels were added (no valid labels found in agent output).
`
      )
      .write();
    return;
  }
  core.info(`Adding ${uniqueLabels.length} labels to ${contextType} #${itemNumber}: ${JSON.stringify(uniqueLabels)}`);
  try {
    await github.rest.issues.addLabels({
      owner: context.repo.owner,
      repo: context.repo.repo,
      issue_number: itemNumber,
      labels: uniqueLabels,
    });
    core.info(`Successfully added ${uniqueLabels.length} labels to ${contextType} #${itemNumber}`);
    core.setOutput("labels_added", uniqueLabels.join("\n"));
    const labelsListMarkdown = uniqueLabels.map(label => `- \`${label}\``).join("\n");
    await core.summary
      .addRaw(
        `
## Label Addition

Successfully added ${uniqueLabels.length} label(s) to ${contextType} #${itemNumber}:

${labelsListMarkdown}
`
      )
      .write();
  } catch (error) {
    const errorMessage = error instanceof Error ? error.message : String(error);
    core.error(`Failed to add labels: ${errorMessage}`);
    core.setFailed(`Failed to add labels: ${errorMessage}`);
  }
}
await main();
