// @ts-check
/// <reference types="@actions/github-script" />
"use strict";
// @safe-outputs-exempt SEC-004 — schema generator; does not process user body content. The substring "body:" appears only in the comment referencing the "allow-body" config option.

/**
 * generate_safe_outputs_tools.cjs
 *
 * Generates the safe outputs tools.json at runtime by:
 * 1. Writing tools_meta.json and validation.json from env var payloads (if provided)
 * 2. Loading the full safe_outputs_tools.json from the actions folder
 * 3. Filtering tools based on config.json (which tools are enabled)
 * 4. Applying description suffixes and repo parameters from tools_meta.json
 * 5. Appending dynamic tools (dispatch_workflow, call_workflow, custom jobs) from tools_meta.json
 * 6. Writing the result to the output tools.json path
 *
 * Environment variables:
 *   GH_AW_TOOLS_META_JSON - JSON payload for tools_meta.json (written to disk before processing)
 *   GH_AW_VALIDATION_JSON - JSON payload for validation.json (written to disk if provided)
 *   GH_AW_SAFE_OUTPUTS_TOOLS_SOURCE_PATH - Path to the source safe_outputs_tools.json
 *     Default: ${RUNNER_TEMP}/gh-aw/actions/safe_outputs_tools.json
 *   GH_AW_SAFE_OUTPUTS_CONFIG_PATH - Path to config.json (used to determine enabled tools)
 *     Default: ${RUNNER_TEMP}/gh-aw/safeoutputs/config.json
 *   GH_AW_SAFE_OUTPUTS_TOOLS_META_PATH - Path to tools_meta.json (descriptions, repo params, dynamic tools)
 *     Default: ${RUNNER_TEMP}/gh-aw/safeoutputs/tools_meta.json
 *   GH_AW_SAFE_OUTPUTS_TOOLS_PATH - Output path for the generated tools.json
 *     Default: ${RUNNER_TEMP}/gh-aw/safeoutputs/tools.json
 */

const fs = require("fs");
const path = require("path");
const { ERR_CONFIG } = require("./error_codes.cjs");
const { getErrorMessage } = require("./error_helpers.cjs");
const ADD_COMMENT_DEFAULT_DISCUSSIONS_NOTE =
  "NOTE: By default, this tool does not require discussions:write permission. Set 'discussions: true' in the workflow's safe-outputs.add-comment configuration to enable discussion comments and request this permission.";
const ADD_COMMENT_DISCUSSIONS_ENABLED_NOTE = "NOTE: Discussion comments are enabled for this workflow because discussions:write permission is available.";
const ADD_COMMENT_DISCUSSIONS_DISABLED_NOTE =
  "NOTE: Discussion comments are disabled for this workflow because discussions:write permission is not available. Set 'discussions: true' in the workflow's safe-outputs.add-comment configuration to enable discussion comments and request this permission.";
const ADD_COMMENT_REPLY_SUPPORT_SENTENCE = "Supports reply_to_id for discussion threading.";
const ADD_COMMENT_REPLY_SUPPORT_REGEX = /\s*Supports reply_to_id for discussion threading\./g;
const ISSUE_INTENT_SUFFIX = "INTENT: Include rationale (string, max 280 chars) and confidence (string, exactly one of: LOW, MEDIUM, HIGH) with each call.";
const ISSUE_INTENT_TOOL_NAMES = new Set(["set_issue_type", "set_issue_field", "add_labels", "close_issue", "assign_to_user", "assign_to_agent"]);
const ISSUE_INTENT_SCHEMA_FIELDS = ["rationale", "confidence", "suggest"];

/**
 * Determine whether issue-intent guidance is enabled for a tool.
 * Default is disabled; explicit issue_intent: true enables it.
 *
 * @param {string} toolName
 * @param {unknown} toolConfig
 * @returns {boolean}
 */
function isIssueIntentEnabledForTool(toolName, toolConfig) {
  if (!ISSUE_INTENT_TOOL_NAMES.has(toolName)) {
    return false;
  }
  return !!(toolConfig && typeof toolConfig === "object" && "issue_intent" in toolConfig && toolConfig.issue_intent === true);
}

/**
 * Determine whether issue-intent schema fields should be omitted for a tool.
 * Default is optional (present, not required); explicit issue_intent: false omits them.
 *
 * @param {string} toolName
 * @param {unknown} toolConfig
 * @returns {boolean}
 */
function isIssueIntentDisabledForTool(toolName, toolConfig) {
  if (!ISSUE_INTENT_TOOL_NAMES.has(toolName)) {
    return false;
  }
  return !!(toolConfig && typeof toolConfig === "object" && "issue_intent" in toolConfig && toolConfig.issue_intent === false);
}

/**
 * Remove issue-intent properties from a tool schema.
 * If this removes all required fields, the required array is omitted.
 * @param {{inputSchema?: {properties?: Record<string, unknown>, required?: string[]}}} tool
 */
function stripIssueIntentSchemaFields(tool) {
  if (!tool.inputSchema || !tool.inputSchema.properties) {
    return;
  }
  for (const field of ISSUE_INTENT_SCHEMA_FIELDS) {
    if (field in tool.inputSchema.properties) {
      delete tool.inputSchema.properties[field];
    }
  }
  if (Array.isArray(tool.inputSchema.required)) {
    tool.inputSchema.required = tool.inputSchema.required.filter(f => !ISSUE_INTENT_SCHEMA_FIELDS.includes(f));
    if (tool.inputSchema.required.length === 0) {
      delete tool.inputSchema.required;
    }
  }
}

/**
 * Update add_comment description to match runtime-safe-output permissions.
 * @param {string} description
 * @param {unknown} addCommentConfig
 * @returns {string}
 */
function updateAddCommentDescription(description, addCommentConfig) {
  const discussionCommentsEnabled = typeof addCommentConfig === "object" && addCommentConfig !== null && "discussions" in addCommentConfig && addCommentConfig.discussions === true;

  let updated = description || "";
  const note = discussionCommentsEnabled ? ADD_COMMENT_DISCUSSIONS_ENABLED_NOTE : ADD_COMMENT_DISCUSSIONS_DISABLED_NOTE;
  if (updated.includes(ADD_COMMENT_DEFAULT_DISCUSSIONS_NOTE)) {
    updated = updated.replace(ADD_COMMENT_DEFAULT_DISCUSSIONS_NOTE, note);
  } else if (!updated.includes(ADD_COMMENT_DISCUSSIONS_ENABLED_NOTE) && !updated.includes(ADD_COMMENT_DISCUSSIONS_DISABLED_NOTE)) {
    updated = `${updated} ${note}`.trim();
  }

  if (discussionCommentsEnabled) {
    if (!updated.includes(ADD_COMMENT_REPLY_SUPPORT_SENTENCE)) {
      updated = `${updated} ${ADD_COMMENT_REPLY_SUPPORT_SENTENCE}`.trim();
    }
  } else {
    updated = updated
      .replace(ADD_COMMENT_REPLY_SUPPORT_REGEX, "")
      .replace(/\s{2,}/g, " ")
      .trim();
  }

  return updated;
}

async function main() {
  const toolsSourcePath = process.env.GH_AW_SAFE_OUTPUTS_TOOLS_SOURCE_PATH || `${process.env.RUNNER_TEMP}/gh-aw/actions/safe_outputs_tools.json`;
  const configPath = process.env.GH_AW_SAFE_OUTPUTS_CONFIG_PATH || `${process.env.RUNNER_TEMP}/gh-aw/safeoutputs/config.json`;
  const toolsMetaPath = process.env.GH_AW_SAFE_OUTPUTS_TOOLS_META_PATH || path.join(path.dirname(configPath), "tools_meta.json");
  const outputPath = process.env.GH_AW_SAFE_OUTPUTS_TOOLS_PATH || `${process.env.RUNNER_TEMP}/gh-aw/safeoutputs/tools.json`;

  // Write JSON payloads from env vars if provided (replaces heredoc-based file writing)
  if (process.env.GH_AW_TOOLS_META_JSON) {
    try {
      fs.writeFileSync(toolsMetaPath, process.env.GH_AW_TOOLS_META_JSON);
    } catch (err) {
      throw new Error(`Failed to write file ${toolsMetaPath}: ${String(err)}`, { cause: err });
    }
  }
  if (process.env.GH_AW_VALIDATION_JSON) {
    const validationPath = path.join(path.dirname(configPath), "validation.json");
    try {
      fs.writeFileSync(validationPath, process.env.GH_AW_VALIDATION_JSON);
    } catch (err) {
      throw new Error(`Failed to write file ${validationPath}: ${String(err)}`, { cause: err });
    }
  }

  // Load all source tools from the actions folder
  if (!fs.existsSync(toolsSourcePath)) {
    const msg = `${ERR_CONFIG}: Source tools file not found at: ${toolsSourcePath}`;
    console.error(msg);
    throw new Error(msg);
  }
  /** @type {Array<{name: string, description: string, inputSchema?: {properties?: Record<string, unknown>}}>} */
  let allTools;
  try {
    allTools = JSON.parse(fs.readFileSync(toolsSourcePath, "utf8"));
  } catch (err) {
    throw new Error("Failed to parse tools source file " + toolsSourcePath + ": " + getErrorMessage(err), { cause: err });
  }

  // Load config to determine which tools are enabled
  if (!fs.existsSync(configPath)) {
    const msg = `${ERR_CONFIG}: Config file not found at: ${configPath}`;
    console.error(msg);
    throw new Error(msg);
  }
  /** @type {Record<string, unknown>} */
  let config;
  try {
    config = JSON.parse(fs.readFileSync(configPath, "utf8"));
  } catch (err) {
    throw new Error("Failed to parse config file " + configPath + ": " + getErrorMessage(err), { cause: err });
  }

  // Load tools meta (description suffixes, repo params, dynamic tools)
  /** @type {{description_suffixes?: Record<string, string>, repo_params?: Record<string, {type: string, description: string}>, dynamic_tools?: Array<unknown>, required_field_removals?: Record<string, string[]>, required_field_additions?: Record<string, string[]>}} */
  let toolsMeta = { description_suffixes: {}, repo_params: {}, dynamic_tools: [] };
  if (fs.existsSync(toolsMetaPath)) {
    try {
      toolsMeta = JSON.parse(fs.readFileSync(toolsMetaPath, "utf8"));
    } catch (err) {
      throw new Error("Failed to parse tools meta file " + toolsMetaPath + ": " + getErrorMessage(err), { cause: err });
    }
  }

  // Build set of source tool names (predefined/static tools only)
  const sourceToolNames = new Set(allTools.map(t => t.name));

  // Determine enabled tools: config keys that match source tool names
  // This filters out non-tool config entries like dispatch_workflow, call_workflow,
  // mentions, max_bot_mentions, etc.
  const enabledToolNames = new Set(Object.keys(config).filter(k => sourceToolNames.has(k)));
  // Filter predefined tools to those enabled in config and apply enhancements
  const filteredTools = allTools
    .filter(tool => enabledToolNames.has(tool.name))
    .map(tool => {
      // Deep copy to avoid modifying the original
      let enhancedTool;
      try {
        enhancedTool = JSON.parse(JSON.stringify(tool));
      } catch (err) {
        throw new Error("Failed to deep-copy tool " + tool.name + ": " + getErrorMessage(err), { cause: err });
      }

      // Apply description suffix if available (e.g., " CONSTRAINTS: Maximum 5 issues.")
      const descSuffix = toolsMeta.description_suffixes?.[tool.name];
      if (descSuffix) {
        enhancedTool.description = (enhancedTool.description || "") + descSuffix;
      }
      if (isIssueIntentEnabledForTool(tool.name, config[tool.name])) {
        enhancedTool.description = `${enhancedTool.description || ""} ${ISSUE_INTENT_SUFFIX}`.trim();
      }
      if (isIssueIntentDisabledForTool(tool.name, config[tool.name])) {
        stripIssueIntentSchemaFields(enhancedTool);
      }

      if (tool.name === "add_comment") {
        enhancedTool.description = updateAddCommentDescription(enhancedTool.description, config.add_comment);
      }

      // Add repo parameter to inputSchema if configured
      const repoParam = toolsMeta.repo_params?.[tool.name];
      if (repoParam) {
        if (!enhancedTool.inputSchema) {
          enhancedTool.inputSchema = { type: "object", properties: {} };
        }
        if (!enhancedTool.inputSchema.properties) {
          enhancedTool.inputSchema.properties = {};
        }
        enhancedTool.inputSchema.properties.repo = repoParam;
      }

      // Remove fields from inputSchema.required when configured (e.g. allow-body: false)
      const requiredRemovals = toolsMeta.required_field_removals?.[tool.name];
      if (requiredRemovals && Array.isArray(enhancedTool.inputSchema?.required)) {
        enhancedTool.inputSchema.required = enhancedTool.inputSchema.required.filter(/** @param {string} f */ f => !requiredRemovals.includes(f));
        if (enhancedTool.inputSchema.required.length === 0) {
          delete enhancedTool.inputSchema.required;
        }
      }

      // Add fields to inputSchema.required when configured (e.g. require-temporary-id: true)
      const requiredAdditions = toolsMeta.required_field_additions?.[tool.name];
      if (requiredAdditions && requiredAdditions.length > 0) {
        const existingRequired = Array.isArray(enhancedTool.inputSchema?.required) ? enhancedTool.inputSchema.required : [];
        enhancedTool.inputSchema.required = Array.from(new Set([...existingRequired, ...requiredAdditions]));
      }

      return enhancedTool;
    });

  // Append dynamic tools (custom jobs, dispatch_workflow, call_workflow)
  const dynamicTools = Array.isArray(toolsMeta.dynamic_tools) ? toolsMeta.dynamic_tools : [];
  const allFilteredTools = [...filteredTools, ...dynamicTools];

  // Write the result to the output path
  try {
    fs.writeFileSync(outputPath, JSON.stringify(allFilteredTools, null, 2));
  } catch (err) {
    throw new Error(`Failed to write file ${outputPath}: ${String(err)}`, { cause: err });
  }

  const debugEnabled = process.env.DEBUG === "*" || (process.env.DEBUG || "").includes("safe_outputs");
  if (debugEnabled) {
    const infoMsg = `Generated tools.json with ${allFilteredTools.length} tools (${filteredTools.length} static + ${dynamicTools.length} dynamic)`;
    if (typeof core !== "undefined") {
      core.info(infoMsg);
    } else {
      console.log(infoMsg);
    }
  }
}

module.exports = { main };

// Run when executed directly (e.g. node generate_safe_outputs_tools.cjs)
if (require.main === module) {
  main().catch(err => {
    process.exit(1);
  });
}
