// @ts-check
/// <reference types="@actions/github-script" />

/**
 * pick_experiment
 *
 * Selects A/B experiment variants for the current workflow run.
 *
 * Environment variables (set by the compiled workflow step):
 *   GH_AW_EXPERIMENT_SPEC       - JSON object mapping experiment name → variant config.
 *                                  Each value is either a legacy bare array of strings
 *                                  or a new object with a 'variants' field and optional
 *                                  metadata: weight, start_date, end_date, description, metric.
 *                                  e.g. '{"feature1":["A","B"],"style":{"variants":["concise","detailed"],"weight":[70,30]}}'
 *   GH_AW_EXPERIMENT_STATE_FILE - Absolute path to the JSON state file to read/write
 *                                  e.g. /tmp/gh-aw/experiments/state.json
 *   GH_AW_EXPERIMENT_STATE_DIR  - Directory that holds the state file (created if missing)
 *                                  e.g. /tmp/gh-aw/experiments
 *
 * Algorithm:
 *   When weight is provided the variant is chosen by weighted-random selection.
 *   Otherwise the variant with the lowest invocation count is selected next (ties are
 *   broken by random selection, ensuring no variant is systematically favoured on the
 *   first run or whenever counts are equal).
 *   When start_date or end_date is provided and today falls outside that window the
 *   control variant (first variant) is used and no counter is incremented.
 */

const fs = require("fs");
const path = require("path");

/** Maximum number of per-run records retained in state.runs. Older entries are pruned to keep state.json small. */
const MAX_RUN_HISTORY = 512;

/**
 * @typedef {Object} ExperimentRunRecord
 * @property {string} run_id       - GitHub Actions run ID (GITHUB_RUN_ID)
 * @property {string} timestamp    - ISO-8601 UTC timestamp of the run
 * @property {Record<string, string>} assignments - Maps experiment name → selected variant
 */

/**
 * @typedef {Object} ExperimentState
 * @property {Record<string, Record<string, number>>} counts
 *   Maps experiment name → variant → cumulative invocation count.
 * @property {ExperimentRunRecord[]} [runs]
 *   Per-run assignment history appended on each invocation.
 */

/**
 * @typedef {Object} GuardrailMetric
 * @property {string} name      - Metric name (e.g. "success_rate")
 * @property {string} threshold - Comparison expression (e.g. ">=0.95")
 */

/**
 * @typedef {Object} ExperimentConfig
 * @property {string[]} variants                    - Array of variant values (length >= 2)
 * @property {number[]} [weight]                    - Optional per-variant weights (same length as variants)
 * @property {string} [start_date]                  - ISO-8601 date; inactive before this date
 * @property {string} [end_date]                    - ISO-8601 date; inactive after this date
 * @property {string} [description]
 * @property {string} [hypothesis]                  - Null and alternative hypothesis text
 * @property {string} [metric]                      - Primary metric name
 * @property {string[]} [secondary_metrics]         - Additional metrics to track
 * @property {GuardrailMetric[]} [guardrail_metrics] - Thresholds that must not degrade
 * @property {number} [min_samples]                 - Minimum runs per variant for reliable analysis
 * @property {number} [issue]
 * @property {string} [analysis_type]               - Statistical test: t_test | mann_whitney | proportion_test | bayesian_ab
 * @property {string[]} [tags]                      - Free-form labels for dashboard filtering
 * @property {{discussion?: number, issue?: number}} [notify] - Where to post significance alerts
 */

/**
 * Normalize a raw spec entry (either a legacy bare array or the new object form) into
 * an ExperimentConfig object.
 *
 * @param {string[]|ExperimentConfig} raw
 * @returns {ExperimentConfig}
 */
function normalizeConfig(raw) {
  if (Array.isArray(raw)) {
    return { variants: raw };
  }
  return raw;
}

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
      if (!Array.isArray(parsed.runs)) {
        parsed.runs = [];
      }
      return parsed;
    }
  } catch {
    // File missing, unreadable, or invalid JSON – start fresh.
  }
  return { counts: {}, runs: [] };
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
 * Return true when today (UTC) falls within the optional [start_date, end_date] window.
 * A missing date is treated as unbounded (open interval).
 *
 * @param {string|undefined} startDate - YYYY-MM-DD or undefined
 * @param {string|undefined} endDate   - YYYY-MM-DD or undefined
 * @param {string} [todayOverride]     - Override today's date for testing (YYYY-MM-DD)
 * @returns {boolean}
 */
function isWithinDateWindow(startDate, endDate, todayOverride) {
  const today = todayOverride || new Date().toISOString().slice(0, 10);
  if (startDate && today < startDate) {
    return false;
  }
  if (endDate && today > endDate) {
    return false;
  }
  return true;
}

/**
 * Pick the variant for one experiment using a balanced least-used selection.
 * The variant with the lowest cumulative count is chosen; when multiple variants
 * share the lowest count (including the initial empty-cache state where all counts
 * are zero), one is selected at random to avoid systematically favouring the first
 * declared variant.
 *
 * @param {string} name       - Experiment name
 * @param {string[]} variants - Array of variant values (length >= 2)
 * @param {ExperimentState} state
 * @returns {string} The selected variant
 */
function pickVariant(name, variants, state) {
  const counts = state.counts[name] || {};
  let minCount = Infinity;
  let tied = [];
  for (const variant of variants) {
    const c = counts[variant] || 0;
    if (c < minCount) {
      minCount = c;
      tied = [variant];
    } else if (c === minCount) {
      tied.push(variant);
    }
  }
  return tied[Math.floor(Math.random() * tied.length)];
}

/**
 * Pick the variant for one experiment using weighted random selection.
 * Each variant is chosen with probability proportional to its weight.
 * Zero-weight variants are never selected.
 *
 * @param {string[]} variants - Array of variant values (length >= 2)
 * @param {number[]} weight   - Per-variant weights (same length as variants, all >= 0)
 * @returns {string} The selected variant
 */
function pickVariantWeighted(variants, weight) {
  const total = weight.reduce((a, b) => a + b, 0);
  if (total <= 0) {
    // All weights are zero – fall back to first variant (control).
    return variants[0];
  }
  let rnd = Math.random() * total;
  for (let i = 0; i < variants.length; i++) {
    rnd -= weight[i];
    if (rnd <= 0) {
      return variants[i];
    }
  }
  // Floating-point rounding guard: return last non-zero-weight variant.
  for (let i = variants.length - 1; i >= 0; i--) {
    if (weight[i] > 0) return variants[i];
  }
  return variants[0];
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
 * @param {Record<string, ExperimentConfig>} configs - Normalized config per experiment
 * @param {ExperimentState} state               - Updated state (post-selection)
 * @param {any} core                            - @actions/core
 */
async function writeSummary(assignments, configs, state, core) {
  const names = Object.keys(assignments).sort();
  const lines = ["## 🧪 Experiment Assignments", "", "| Experiment | Variant | Counts (current/total) |", "| --- | --- | --- |"];
  for (const name of names) {
    const selected = assignments[name];
    const counts = state.counts[name] || {};
    const thisCount = counts[selected] || 0;
    // Prefer counting actual run records for the total when the runs array is present;
    // fall back to summing incremented counts (which excludes date-window gated runs).
    const runsForExp = state.runs ? state.runs.filter(r => r.assignments && name in r.assignments) : null;
    const totalCount = runsForExp !== null && runsForExp.length > 0 ? runsForExp.length : Object.values(/** @type {number[]} */ counts).reduce((a, b) => a + b, 0);
    lines.push(`| \`${name}\` | **${selected}** | ${thisCount} / ${totalCount} |`);
  }
  lines.push("");

  // Progress bars and ready-for-analysis flags when min_samples is a positive integer.
  const progressNames = names.filter(name => {
    const ms = configs[name]?.min_samples;
    return ms != null && Number.isInteger(ms) && ms > 0;
  });
  if (progressNames.length > 0) {
    lines.push("### 📊 Sampling Progress");
    lines.push("");
    for (const name of progressNames) {
      const cfg = configs[name];
      const minSamples = cfg.min_samples ?? 0;
      const variants = cfg.variants || [];
      const counts = state.counts[name] || {};
      const allReady = variants.every(v => (counts[v] || 0) >= minSamples);
      if (allReady) {
        lines.push(`**${name}** ✅ Ready for analysis`);
      } else {
        lines.push(`**${name}** (target: ${minSamples} per variant)`);
      }
      for (const variant of variants) {
        const n = counts[variant] || 0;
        const pct = Math.min(100, Math.round((n / minSamples) * 100));
        const filled = Math.round(pct / 5); // 20-char bar
        const bar = "█".repeat(filled) + "░".repeat(20 - filled);
        lines.push(`  ${variant}: ${bar} ${n}/${minSamples} (${pct}%)`);
      }
      lines.push("");
    }
  }

  // Append optional description, hypothesis, guardrail metrics, and issue link.
  const repo = process.env.GITHUB_REPOSITORY || "";
  const metadataNames = names.filter(name => configs[name]?.description || configs[name]?.hypothesis || configs[name]?.guardrail_metrics?.length || configs[name]?.issue);
  if (metadataNames.length > 0) {
    lines.push("### Experiment Details");
    lines.push("");
    for (const name of metadataNames) {
      const cfg = configs[name];
      const description = cfg?.description;
      const hypothesis = cfg?.hypothesis;
      const guardrails = cfg?.guardrail_metrics;
      const issue = cfg?.issue;
      lines.push(`**${name}**`);
      if (description) {
        lines.push("");
        lines.push(`> ${description}`);
      }
      if (hypothesis) {
        lines.push("");
        lines.push(`**Hypothesis:** ${hypothesis}`);
      }
      if (guardrails && guardrails.length > 0) {
        lines.push("");
        lines.push("**Guardrail metrics:**");
        for (const g of guardrails) {
          lines.push(`- \`${g.name}\` ${g.threshold}`);
        }
      }
      if (issue) {
        lines.push("");
        if (repo) {
          lines.push(`Tracking issue: [#${issue}](https://github.com/${repo}/issues/${issue})`);
        } else {
          lines.push(`Tracking issue: #${issue}`);
        }
      }
      lines.push("");
    }
  }

  lines.push("_Variants are selected by balanced round-robin (or weighted) to ensure statistical relevance across runs. Ties are broken randomly so no variant is systematically favoured on the first run._");
  await core.summary.addRaw(lines.join("\n")).write();
}

/**
 * Main entry point called by the actions/github-script step.
 */
async function main() {
  const specRaw = process.env.GH_AW_EXPERIMENT_SPEC || "{}";
  const stateFile = process.env.GH_AW_EXPERIMENT_STATE_FILE || "/tmp/gh-aw/experiments/state.json";
  const stateDir = process.env.GH_AW_EXPERIMENT_STATE_DIR || "/tmp/gh-aw/experiments";

  /** @type {Record<string, string[]|ExperimentConfig>} */
  let rawSpec;
  try {
    rawSpec = JSON.parse(specRaw);
  } catch (e) {
    core.setFailed(`Failed to parse GH_AW_EXPERIMENT_SPEC: ${e.message}`);
    return;
  }

  const experimentNames = Object.keys(rawSpec).sort();
  if (experimentNames.length === 0) {
    core.info("No experiments defined – nothing to do.");
    return;
  }

  // Normalize all spec entries to ExperimentConfig objects.
  /** @type {Record<string, ExperimentConfig>} */
  const configs = {};
  for (const name of experimentNames) {
    configs[name] = normalizeConfig(rawSpec[name]);
  }

  // Ensure the state directory exists so that the cache-save step can find it.
  fs.mkdirSync(stateDir, { recursive: true });

  const state = loadState(stateFile);

  /** @type {Record<string, string>} */
  const assignments = {};

  for (const name of experimentNames) {
    const cfg = configs[name];
    const variants = cfg.variants;
    if (!Array.isArray(variants) || variants.length < 2) {
      core.warning(`Experiment "${name}" has fewer than 2 variants – skipping.`);
      continue;
    }

    // Date-window check: use control variant (first variant) when outside the window.
    if (!isWithinDateWindow(cfg.start_date, cfg.end_date)) {
      const control = variants[0];
      assignments[name] = control;
      core.setOutput(name, control);
      core.info(`Experiment "${name}": outside date window – using control variant "${control}"`);
      continue;
    }

    let selected;
    if (cfg.weight && cfg.weight.length === variants.length) {
      selected = pickVariantWeighted(variants, cfg.weight);
    } else {
      selected = pickVariant(name, variants, state);
    }
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

  if (Object.keys(assignments).length > 0) {
    // Append a per-run record to state.runs so each assignment is traceable.
    const runId = process.env.GITHUB_RUN_ID || "";
    const timestamp = new Date().toISOString();
    if (!state.runs) {
      state.runs = [];
    }
    state.runs.push({ run_id: runId, timestamp, assignments: { ...assignments } });
    // Prune run history to avoid state.json growing without bound over many runs.
    if (state.runs.length > MAX_RUN_HISTORY) {
      state.runs = state.runs.slice(-MAX_RUN_HISTORY);
    }
  }

  // Persist updated counts and run history.
  saveState(stateFile, state);
  core.info(`Experiment state written to ${stateFile}`);

  // Persist current-run assignments to a separate file so downstream jobs and
  // OTLP telemetry can read which variant was selected without recomputing it.
  // Only written when at least one experiment was successfully assigned.
  if (Object.keys(assignments).length > 0) {
    const assignmentsFile = path.join(stateDir, "assignments.json");
    fs.writeFileSync(assignmentsFile, JSON.stringify(assignments, null, 2) + "\n", "utf8");
    core.info(`Experiment assignments written to ${assignmentsFile}`);

    // Emit OTEL resource attributes so every span in this run carries the
    // experiment assignments for filtering in Honeycomb/Grafana.
    const otelAttrs = Object.entries(assignments)
      .map(([name, variant]) => `experiment.${name}=${variant}`)
      .join(",");
    const existingAttrs = process.env.OTEL_RESOURCE_ATTRIBUTES || "";
    core.exportVariable("OTEL_RESOURCE_ATTRIBUTES", existingAttrs ? `${existingAttrs},${otelAttrs}` : otelAttrs);
    core.info(`OTEL resource attributes set: ${otelAttrs}`);
  }

  // Write step summary.
  await writeSummary(assignments, configs, state, core);
}

module.exports = { main, pickVariant, pickVariantWeighted, loadState, saveState, recordVariant, isWithinDateWindow, normalizeConfig };
