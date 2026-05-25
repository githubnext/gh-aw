// @ts-check
/// <reference types="@actions/github-script" />

const fs = require("fs");
const path = require("path");
const { MODELS_JSON_PATH } = require("./constants.cjs");

/**
 * Actions destination path set by setup.sh — the directory where models.json
 * (pre-computed builtin aliases) was copied during the setup step.
 * Computed lazily at call time so RUNNER_TEMP overrides in tests take effect.
 *
 * Falls back to the GitHub Actions default runner temp path when RUNNER_TEMP is unset.
 */
function getActionsDestination() {
  // RUNNER_TEMP is always set by GitHub Actions; the fallback covers local dev/test environments.
  return `${process.env.RUNNER_TEMP || "/home/runner/work/_temp"}/gh-aw/actions`;
}

/**
 * Load the pre-computed builtin model alias map from the actions destination
 * directory (copied by setup.sh from actions/setup/js/models.json).
 *
 * @returns {Record<string, string[]>} Builtin alias map, or empty object if missing/invalid.
 */
function loadBuiltinAliases() {
  const builtinPath = path.join(getActionsDestination(), "models.json");
  if (!fs.existsSync(builtinPath)) {
    core.warning(`Builtin models.json not found at ${builtinPath}; using empty alias map`);
    return {};
  }
  try {
    const data = JSON.parse(fs.readFileSync(builtinPath, "utf8"));
    if (data && typeof data.aliases === "object" && data.aliases !== null) {
      return data.aliases;
    }
    core.warning(`Builtin models.json missing "aliases" key at ${builtinPath}`);
    return {};
  } catch (e) {
    core.warning(`Failed to parse builtin models.json at ${builtinPath}: ${e}`);
    return {};
  }
}

/**
 * Parse user-defined model alias overrides from the GH_AW_INFO_MODEL_ALIASES
 * environment variable (set at compile time from the workflow frontmatter +
 * imported workflow models fields).
 *
 * @returns {Record<string, string[]>} User alias overrides, or empty object if not set/invalid.
 */
function loadUserAliases() {
  const raw = process.env.GH_AW_INFO_MODEL_ALIASES;
  if (!raw || raw === "{}") {
    return {};
  }
  try {
    const parsed = JSON.parse(raw);
    if (typeof parsed === "object" && parsed !== null) {
      return parsed;
    }
    core.warning("GH_AW_INFO_MODEL_ALIASES is not a JSON object; ignoring user overrides");
    return {};
  } catch (e) {
    core.warning(`Failed to parse GH_AW_INFO_MODEL_ALIASES: ${e}`);
    return {};
  }
}

/**
 * Merge builtin aliases with user overrides.  User overrides take priority (overwrite).
 *
 * @param {Record<string, string[]>} builtins - Pre-computed builtin alias map.
 * @param {Record<string, string[]>} overrides - User-defined alias overrides.
 * @returns {Record<string, string[]>} Merged alias map.
 */
function mergeAliases(builtins, overrides) {
  return Object.assign({}, builtins, overrides);
}

/**
 * Render the merged alias map as a collapsible Markdown table for the step summary.
 *
 * @param {Record<string, string[]>} aliases - Merged alias map.
 * @param {number} userCount - Number of user-defined override entries.
 * @returns {string} Markdown string.
 */
function renderSummary(aliases, userCount) {
  const aliasCount = Object.keys(aliases).length;
  const label = `Model aliases (${aliasCount} total, ${userCount} user-defined)`;

  const rows = Object.entries(aliases)
    .sort(([a], [b]) => a.localeCompare(b))
    .map(([alias, targets]) => {
      const aliasCell = alias === "" ? "_default_" : `\`${alias}\``;
      const targetsCell = targets.map((t) => `\`${t}\``).join(", ");
      return `| ${aliasCell} | ${targetsCell} |`;
    })
    .join("\n");

  return (
    "<details>\n" +
    `<summary>${label}</summary>\n\n` +
    "| Alias | Targets |\n" +
    "| ----- | ------- |\n" +
    rows +
    "\n\n" +
    "</details>"
  );
}

/**
 * Compute the merged model alias map and write it to /tmp/gh-aw/models.json.
 *
 * Merge order (highest priority last):
 *   1. Pre-computed builtin aliases from actions/setup/js/models.json (copied by setup.sh)
 *   2. User-defined overrides from GH_AW_INFO_MODEL_ALIASES (set at compile time from
 *      workflow frontmatter + imported workflow models fields)
 *
 * The resulting models.json is included in the activation artifact so downstream
 * jobs (e.g. AWF firewall) can read it without re-deriving the alias map.
 *
 * @returns {Promise<void>}
 */
async function main() {
  const builtins = loadBuiltinAliases();
  const userOverrides = loadUserAliases();
  const merged = mergeAliases(builtins, userOverrides);

  const userCount = Object.keys(userOverrides).length;
  const totalCount = Object.keys(merged).length;
  core.info(`Merged model aliases: ${totalCount} total (${userCount} user-defined override(s))`);

  // Write merged aliases to /tmp/gh-aw/models.json
  fs.mkdirSync("/tmp/gh-aw", { recursive: true });
  const output = {
    version: "1",
    description:
      "Merged model alias map for this workflow run (builtin aliases + user-defined overrides).",
    aliases: merged,
  };
  fs.writeFileSync(MODELS_JSON_PATH, JSON.stringify(output, null, 2));
  core.info(`Written merged model aliases to ${MODELS_JSON_PATH}`);

  // Log to step summary
  const markdown = renderSummary(merged, userCount);
  await core.summary.addRaw(markdown).write();
}

module.exports = { main, loadBuiltinAliases, loadUserAliases, mergeAliases, renderSummary };
