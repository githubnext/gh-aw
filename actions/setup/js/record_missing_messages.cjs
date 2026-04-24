// @ts-check
/// <reference types="@actions/github-script" />

const { loadAgentOutput } = require("./load_agent_output.cjs");
const { getErrorMessage } = require("./error_helpers.cjs");

/**
 * Process missing_tool, missing_data, and report_incomplete messages from agent output.
 * This merged step replaces the separate "Record missing tool" and "Record incomplete" steps.
 * It loads the agent output artifact directly and processes all three message types in
 * a single step, optionally creating GitHub issues via the missing_issue_handler_registry.
 *
 * Behaviour:
 * 1. Load all items from the agent output artifact.
 * 2. Collect missing_tool, missing_data, and report_incomplete messages.
 * 3. Set step outputs: tools_reported, total_count, incomplete_count.
 * 4. When create-issue is configured (GH_AW_*_CREATE_ISSUE=true), delegate issue creation
 *    to the appropriate handler in missing_issue_handler_registry.cjs.
 */
async function main() {
  try {
    // --- Load agent output ---
    const result = loadAgentOutput();
    if (!result.success) {
      core.info("Could not load agent output, skipping");
      return;
    }

    const messages = result.items || [];
    const workflowName = process.env.GH_AW_WORKFLOW_NAME || "unknown";
    const runUrl = process.env.GH_AW_RUN_URL || "";
    const workflowSource = process.env.GH_AW_WORKFLOW_SOURCE || "";
    const workflowSourceURL = process.env.GH_AW_WORKFLOW_SOURCE_URL || "";

    const ts = () => new Date().toISOString();

    // --- Collect messages by type ---
    const missingTools = messages.filter((/** @type {any} */ m) => m.type === "missing_tool").map((/** @type {any} */ m) => ({ tool: m.tool || null, reason: m.reason, alternatives: m.alternatives || null, timestamp: ts() }));

    const missingData = messages
      .filter((/** @type {any} */ m) => m.type === "missing_data")
      .map((/** @type {any} */ m) => ({ data_type: m.data_type, reason: m.reason, context: m.context || null, alternatives: m.alternatives || null, timestamp: ts() }));

    const incompleteSignals = messages.filter((/** @type {any} */ m) => m.type === "report_incomplete").map((/** @type {any} */ m) => ({ reason: m.reason, details: m.details || null, timestamp: ts() }));

    core.info(`Found ${missingTools.length} missing tool message(s)`);
    core.info(`Found ${missingData.length} missing data message(s)`);
    core.info(`Found ${incompleteSignals.length} incomplete signal(s)`);

    // --- Set step outputs ---
    const toolsReported = missingTools
      .filter((/** @type {any} */ t) => t.tool)
      .map((/** @type {any} */ t) => t.tool)
      .join(", ");
    core.setOutput("tools_reported", toolsReported);
    core.setOutput("total_count", String(missingTools.length));
    core.setOutput("incomplete_count", String(incompleteSignals.length));

    // Build shared message base used by all issue-creation calls
    const baseMessage = {
      workflow_name: workflowName,
      workflow_source: workflowSource,
      workflow_source_url: workflowSourceURL,
      run_url: runUrl,
    };

    const { handlerRegistry } = require("./missing_issue_handler_registry.cjs");

    // --- Create missing-tool issue if configured ---
    if (missingTools.length > 0 && process.env.GH_AW_MISSING_TOOL_CREATE_ISSUE === "true") {
      const factory = handlerRegistry.get("create_missing_tool_issue");
      if (factory) {
        const config = buildHandlerConfig("GH_AW_MISSING_TOOL");
        const handler = await factory(config);
        await handler({ ...baseMessage, missing_tools: missingTools }, {});
      }
    }

    // --- Create missing-data issue if configured ---
    if (missingData.length > 0 && process.env.GH_AW_MISSING_DATA_CREATE_ISSUE === "true") {
      const factory = handlerRegistry.get("create_missing_data_issue");
      if (factory) {
        const config = buildHandlerConfig("GH_AW_MISSING_DATA");
        const handler = await factory(config);
        await handler({ ...baseMessage, missing_data: missingData }, {});
      }
    }

    // --- Create report-incomplete issue if configured ---
    if (incompleteSignals.length > 0 && process.env.GH_AW_REPORT_INCOMPLETE_CREATE_ISSUE === "true") {
      const factory = handlerRegistry.get("create_report_incomplete_issue");
      if (factory) {
        const config = buildHandlerConfig("GH_AW_REPORT_INCOMPLETE");
        const handler = await factory(config);
        await handler({ ...baseMessage, incomplete_signals: incompleteSignals }, {});
      }
    }
  } catch (error) {
    core.warning(`Error in record_missing_messages: ${getErrorMessage(error)}`);
  }
}

/**
 * Build a handler config object from environment variables with the given prefix.
 *
 * Reads:
 *   {prefix}_MAX          → config.max  (default: 1)
 *   {prefix}_TITLE_PREFIX → config.title_prefix (if set)
 *   {prefix}_LABELS       → config.labels (JSON array, if set)
 *
 * @param {string} prefix - Environment variable prefix (e.g. "GH_AW_MISSING_TOOL")
 * @returns {{max: number, title_prefix?: string, labels?: string[]}}
 */
function buildHandlerConfig(prefix) {
  /** @type {{max: number, title_prefix?: string, labels?: string[]}} */
  const config = {
    max: parseInt(process.env[`${prefix}_MAX`] || "1", 10),
  };
  const titlePrefix = process.env[`${prefix}_TITLE_PREFIX`];
  if (titlePrefix) config.title_prefix = titlePrefix;
  const labelsJSON = process.env[`${prefix}_LABELS`];
  if (labelsJSON) {
    try {
      config.labels = JSON.parse(labelsJSON);
    } catch (_) {
      // ignore malformed labels JSON
    }
  }
  return config;
}

module.exports = { main, buildHandlerConfig };
