// @ts-check
/// <reference types="@actions/github-script" />
"use strict";

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

async function main() {
  const toolsSourcePath = process.env.GH_AW_SAFE_OUTPUTS_TOOLS_SOURCE_PATH || `${process.env.RUNNER_TEMP}/gh-aw/actions/safe_outputs_tools.json`;
  const configPath = process.env.GH_AW_SAFE_OUTPUTS_CONFIG_PATH || `${process.env.RUNNER_TEMP}/gh-aw/safeoutputs/config.json`;
  const toolsMetaPath = process.env.GH_AW_SAFE_OUTPUTS_TOOLS_META_PATH || path.join(path.dirname(configPath), "tools_meta.json");
  const outputPath = process.env.GH_AW_SAFE_OUTPUTS_TOOLS_PATH || `${process.env.RUNNER_TEMP}/gh-aw/safeoutputs/tools.json`;

  // Write JSON payloads from env vars if provided (replaces heredoc-based file writing)
  if (process.env.GH_AW_TOOLS_META_JSON) {
    fs.writeFileSync(toolsMetaPath, process.env.GH_AW_TOOLS_META_JSON);
  }
  if (process.env.GH_AW_VALIDATION_JSON) {
    const validationPath = path.join(path.dirname(configPath), "validation.json");
    fs.writeFileSync(validationPath, process.env.GH_AW_VALIDATION_JSON);
  }

  // Load all source tools from the actions folder
  if (!fs.existsSync(toolsSourcePath)) {
    const msg = `${ERR_CONFIG}: Source tools file not found at: ${toolsSourcePath}`;
    console.error(msg);
    throw new Error(msg);
  }
  /** @type {Array<{name: string, description: string, inputSchema?: {properties?: Record<string, unknown>}}>} */
  const allTools = JSON.parse(fs.readFileSync(toolsSourcePath, "utf8"));

  // Load config to determine which tools are enabled
  if (!fs.existsSync(configPath)) {
    const msg = `${ERR_CONFIG}: Config file not found at: ${configPath}`;
    console.error(msg);
    throw new Error(msg);
  }
  /** @type {Record<string, unknown>} */
  const config = JSON.parse(fs.readFileSync(configPath, "utf8"));

  // Load tools meta (description suffixes, repo params, dynamic tools)
  /** @type {{description_suffixes?: Record<string, string>, repo_params?: Record<string, {type: string, description: string}>, dynamic_tools?: Array<unknown>}} */
  let toolsMeta = { description_suffixes: {}, repo_params: {}, dynamic_tools: [] };
  if (fs.existsSync(toolsMetaPath)) {
    toolsMeta = JSON.parse(fs.readFileSync(toolsMetaPath, "utf8"));
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
      const enhancedTool = JSON.parse(JSON.stringify(tool));

      // Apply description suffix if available (e.g., " CONSTRAINTS: Maximum 5 issues.")
      const descSuffix = toolsMeta.description_suffixes?.[tool.name];
      if (descSuffix) {
        enhancedTool.description = (enhancedTool.description || "") + descSuffix;
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

      return enhancedTool;
    });

  // Append dynamic tools (custom jobs, dispatch_workflow, call_workflow)
  const dynamicTools = Array.isArray(toolsMeta.dynamic_tools) ? toolsMeta.dynamic_tools : [];
  const allFilteredTools = [...filteredTools, ...dynamicTools];

  // Write the result to the output path
  fs.writeFileSync(outputPath, JSON.stringify(allFilteredTools, null, 2));

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
