// @ts-check
/// <reference types="@actions/github-script" />

const fs = require("fs");

const AWF_REFLECT_PATH = "/tmp/gh-aw/sandbox/firewall/awf-reflect.json";

/**
 * Read the AWF reflect payload that was persisted to disk by copilot_harness.cjs.
 * Returns null when the file is absent or unparseable (AWF not running / not enabled).
 * @returns {object|null}
 */
function readReflectData() {
  if (!fs.existsSync(AWF_REFLECT_PATH)) {
    return null;
  }
  try {
    return JSON.parse(fs.readFileSync(AWF_REFLECT_PATH, "utf8"));
  } catch {
    return null;
  }
}

/**
 * Format a list of model IDs into a compact comma-separated string, capping the output
 * at `maxModels` entries and appending "… +N more" when the list is longer.
 * @param {string[]|null|undefined} models
 * @param {number} maxModels
 * @returns {string}
 */
function formatModelList(models, maxModels) {
  if (!Array.isArray(models) || models.length === 0) {
    return "—";
  }
  if (models.length <= maxModels) {
    return models.join(", ");
  }
  const shown = models.slice(0, maxModels);
  const remaining = models.length - maxModels;
  return `${shown.join(", ")} … +${remaining} more`;
}

/**
 * Build a markdown step summary from AWF /reflect response data.
 *
 * The summary is wrapped in a <details>/<summary> block so it stays collapsed by
 * default and does not dominate the step output. Each row of the table shows:
 *   - Provider name
 *   - Port the endpoint listens on
 *   - Whether a key/token is configured
 *   - Available models (first `maxModels` entries, with overflow indicator)
 *
 * @param {object} reflectData - Parsed /reflect JSON response
 * @param {{ maxModels?: number }} options
 * @returns {string}
 */
function buildReflectSummary(reflectData, options) {
  const maxModels = options && options.maxModels != null ? options.maxModels : 5;
  const endpoints = Array.isArray(reflectData.endpoints) ? reflectData.endpoints : [];
  const fetchComplete = reflectData.models_fetch_complete === true;

  const lines = [];
  lines.push("<details>");

  const configuredCount = endpoints.filter(ep => ep.configured).length;
  lines.push(`<summary>AWF API proxy: ${configuredCount} of ${endpoints.length} provider${endpoints.length !== 1 ? "s" : ""} configured</summary>`);
  lines.push("");

  if (endpoints.length === 0) {
    lines.push("No endpoint information available.");
  } else {
    const fetchNote = fetchComplete ? "" : " *(model list may be incomplete — fetch in progress)*";
    lines.push(`| Provider | Port | Configured | Available models${fetchNote} |`);
    lines.push("|----------|------|:----------:|-----------------|");

    for (const ep of endpoints) {
      const provider = String(ep.provider || "unknown");
      const port = ep.port != null ? String(ep.port) : "—";
      const configured = ep.configured ? "✅" : "❌";
      const modelStr = formatModelList(ep.models, maxModels);
      lines.push(`| ${provider} | ${port} | ${configured} | ${modelStr} |`);
    }
  }

  lines.push("");
  lines.push("</details>");
  lines.push("");

  return lines.join("\n");
}

async function main() {
  const reflectData = readReflectData();

  if (!reflectData) {
    core.info("AWF reflect data not available (AWF not enabled or /reflect not reachable), skipping summary");
    return;
  }

  const markdown = buildReflectSummary(reflectData, {});
  await core.summary.addRaw(markdown).write();
  core.info("AWF reflect summary written to step summary");
}

if (typeof module !== "undefined" && module.exports) {
  module.exports = {
    AWF_REFLECT_PATH,
    buildReflectSummary,
    formatModelList,
    main,
    readReflectData,
  };
}
