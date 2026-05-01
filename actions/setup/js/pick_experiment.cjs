// @ts-check
/// <reference types="@actions/github-script" />

/**
 * pick_experiment
 *
 * Selects A/B experiment variants for the current workflow run.
 *
 * Environment variables (set by the compiled workflow step):
 *   GH_AW_EXPERIMENT_SPEC       - JSON object mapping experiment name → array of variant strings
 *                                  e.g. '{"feature1":["A","B"],"style":["concise","detailed"]}'
 *   GH_AW_EXPERIMENT_STATE_FILE - Absolute path to the JSON state file to read/write
 *                                  e.g. /tmp/gh-aw/experiments/state.json
 *   GH_AW_EXPERIMENT_STATE_DIR  - Directory that holds the state file (created if missing)
 *                                  e.g. /tmp/gh-aw/experiments
 *
 * Algorithm:
 *   For each experiment the function maintains a counter per variant in the state file.
 *   The variant with the lowest invocation count is selected next (ties are broken by
 *   variant order, yielding a deterministic round-robin across runs).
 *   This ensures that across N runs every variant is used approximately N/K times where
 *   K is the number of variants, satisfying basic A/B statistical balance.
 *
 * Outputs:
 *   - Sets core.setOutput(name, selected) for each experiment (e.g. caveman=yes).
 *   - Sets core.setOutput('experiments', JSON.stringify(assignments)) for the full map.
 *   - Writes the updated counter state back to GH_AW_EXPERIMENT_STATE_FILE.
 *   - Appends a Markdown step summary with the assignment table and cumulative counts.
 */

const fs = require("fs");
const path = require("path");

/**
 * @typedef {Object} ExperimentState
 * @property {Record<string, Record<string, number>>} counts
 *   Maps experiment name → variant → cumulative invocation count.
 */

/**
 * Load and parse the state JSON file.  Returns an empty state if the file does not exist
 * or cannot be parsed (e.g. first run or corrupted cache).
 *
 * @param {string} stateFile
 * @returns {ExperimentState}
 */
function loadState(stateFile) {
  try {
    const raw = fs.readFileSync(stateFile, "utf8");
    const parsed = JSON.parse(raw);
    if (parsed && typeof parsed.counts === "object") {
      return parsed;
    }
  } catch {
    // File missing, unreadable, or invalid JSON – start fresh.
  }
  return { counts: {} };
}

/**
 * Persist the state JSON file to disk.
 *
 * @param {string} stateFile
 * @param {ExperimentState} state
 */
function saveState(stateFile, state) {
  const dir = path.dirname(stateFile);
  fs.mkdirSync(dir, { recursive: true });
  fs.writeFileSync(stateFile, JSON.stringify(state, null, 2) + "\n", "utf8");
}

/**
 * Pick the variant for one experiment using a balanced least-used selection.
 * The variant with the lowest cumulative count is chosen; ties are broken by
 * the order of the variants array so selection is deterministic.
 *
 * @param {string} name       - Experiment name
 * @param {string[]} variants - Array of variant values (length >= 2)
 * @param {ExperimentState} state
 * @returns {string} The selected variant
 */
function pickVariant(name, variants, state) {
  const counts = state.counts[name] || {};
  let minCount = Infinity;
  let selected = variants[0];
  for (const variant of variants) {
    const c = counts[variant] || 0;
    if (c < minCount) {
      minCount = c;
      selected = variant;
    }
  }
  return selected;
}

/**
 * Increment the counter for the chosen variant.
 *
 * @param {string} name    - Experiment name
 * @param {string} variant - Chosen variant
 * @param {ExperimentState} state
 */
function recordVariant(name, variant, state) {
  if (!state.counts[name]) {
    state.counts[name] = {};
  }
  state.counts[name][variant] = (state.counts[name][variant] || 0) + 1;
}

/**
 * Append a Markdown step summary describing the experiment assignments.
 *
 * @param {Record<string, string>} assignments  - Maps experiment name → selected variant
 * @param {Record<string, string[]>} spec       - Maps experiment name → variants array
 * @param {ExperimentState} state               - Updated state (post-selection)
 * @param {any} core                            - @actions/core
 */
async function writeSummary(assignments, spec, state, core) {
  const names = Object.keys(assignments).sort();
  const lines = ["## 🧪 A/B Experiment Assignments", "", "| Experiment | Selected Variant | All Variants | Cumulative Counts |", "| --- | --- | --- | --- |"];
  for (const name of names) {
    const selected = assignments[name];
    const variants = spec[name] || [];
    const counts = state.counts[name] || {};
    const countsStr = variants.map(v => `${v}: ${counts[v] || 0}`).join(", ");
    lines.push(`| \`${name}\` | **${selected}** | ${variants.join(", ")} | ${countsStr} |`);
  }
  lines.push("");
  lines.push("_Variants are selected by balanced round-robin to ensure statistical relevance across runs._");
  await core.summary.addRaw(lines.join("\n")).write();
}

/**
 * Main entry point called by the actions/github-script step.
 */
async function main() {
  const specRaw = process.env.GH_AW_EXPERIMENT_SPEC || "{}";
  const stateFile = process.env.GH_AW_EXPERIMENT_STATE_FILE || "/tmp/gh-aw/experiments/state.json";
  const stateDir = process.env.GH_AW_EXPERIMENT_STATE_DIR || "/tmp/gh-aw/experiments";

  /** @type {Record<string, string[]>} */
  let spec;
  try {
    spec = JSON.parse(specRaw);
  } catch (e) {
    core.setFailed(`Failed to parse GH_AW_EXPERIMENT_SPEC: ${e.message}`);
    return;
  }

  const experimentNames = Object.keys(spec).sort();
  if (experimentNames.length === 0) {
    core.info("No experiments defined – nothing to do.");
    return;
  }

  // Ensure the state directory exists so that the cache-save step can find it.
  fs.mkdirSync(stateDir, { recursive: true });

  const state = loadState(stateFile);

  /** @type {Record<string, string>} */
  const assignments = {};

  for (const name of experimentNames) {
    const variants = spec[name];
    if (!Array.isArray(variants) || variants.length < 2) {
      core.warning(`Experiment "${name}" has fewer than 2 variants – skipping.`);
      continue;
    }
    const selected = pickVariant(name, variants, state);
    recordVariant(name, selected, state);
    assignments[name] = selected;

    // Expose the selected variant as a step output (individual per experiment).
    // Downstream jobs access this via needs.activation.outputs.<name>.
    core.setOutput(name, selected);
    core.info(`Experiment "${name}": selected variant "${selected}" (output: ${name}=${selected})`);
  }

  // Expose the full assignments map as a serialized JSON step output.
  // Downstream jobs access this via needs.activation.outputs.experiments.
  const experimentsJSON = JSON.stringify(assignments);
  core.setOutput("experiments", experimentsJSON);
  core.info(`Experiment assignments (JSON): ${experimentsJSON}`);

  // Persist updated counts.
  saveState(stateFile, state);
  core.info(`Experiment state written to ${stateFile}`);

  // Persist current-run assignments to a separate file so downstream jobs and
  // OTLP telemetry can read which variant was selected without recomputing it.
  // Only written when at least one experiment was successfully assigned.
  if (Object.keys(assignments).length > 0) {
    const assignmentsFile = path.join(stateDir, "assignments.json");
    fs.writeFileSync(assignmentsFile, JSON.stringify(assignments, null, 2) + "\n", "utf8");
    core.info(`Experiment assignments written to ${assignmentsFile}`);
  }

  // Write step summary.
  await writeSummary(assignments, spec, state, core);
}

module.exports = { main, pickVariant, loadState, saveState, recordVariant };
