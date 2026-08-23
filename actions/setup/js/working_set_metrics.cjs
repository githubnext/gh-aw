// @ts-check

/**
 * @typedef {object} WorkingSetMetrics
 * @property {"measured" | "partial" | "unavailable"} measurement_state
 * @property {number} [rebuild_factor]
 * @property {number} cumulative_input_tokens
 * @property {number} peak_input_tokens
 * @property {number} rebuild_excess_tokens
 * @property {number} invocations
 */

/**
 * Compute Working-Set Rebuild Factor from canonical per-request input_tokens.
 * Cache-read and cache-write fields are intentionally not added: token-usage
 * records already expose gh-aw's normalized logical input count.
 *
 * @param {any[]} entries
 * @param {number} [initialIgnoredRecords]
 * @returns {{ workingSet: WorkingSetMetrics, ignoredRecords: number }}
 */
function calculateWorkingSetFromEntries(entries, initialIgnoredRecords = 0) {
  let cumulativeInputTokens = 0n;
  let peakInputTokens = 0n;
  let invocations = 0;
  let ignoredRecords = initialIgnoredRecords;

  for (const entry of entries) {
    const inputTokens = entry?.input_tokens;
    if (typeof inputTokens !== "number" || !Number.isSafeInteger(inputTokens) || inputTokens < 0) {
      ignoredRecords += 1;
      continue;
    }

    const logicalInputTokens = BigInt(inputTokens);
    cumulativeInputTokens += logicalInputTokens;
    if (logicalInputTokens > peakInputTokens) {
      peakInputTokens = logicalInputTokens;
    }
    invocations += 1;
  }

  /** @type {WorkingSetMetrics} */
  const base = {
    measurement_state: "unavailable",
    cumulative_input_tokens: Number(cumulativeInputTokens),
    peak_input_tokens: Number(peakInputTokens),
    rebuild_excess_tokens: Number(cumulativeInputTokens - peakInputTokens),
    invocations,
  };

  if (peakInputTokens === 0n) {
    return { workingSet: base, ignoredRecords };
  }

  const rebuildFactor = Number(cumulativeInputTokens) / Number(peakInputTokens);
  return {
    workingSet: {
      ...base,
      measurement_state: ignoredRecords > 0 ? "partial" : "measured",
      rebuild_factor: Number.isFinite(rebuildFactor) ? Math.max(1, rebuildFactor) : 1,
    },
    ignoredRecords,
  };
}

/**
 * @param {string} content
 * @returns {{ workingSet: WorkingSetMetrics, ignoredRecords: number }}
 */
function calculateWorkingSetFromJSONL(content) {
  const entries = [];
  let ignoredRecords = 0;

  for (const raw of content.split("\n")) {
    const line = raw.trim();
    if (!line) continue;

    try {
      entries.push(JSON.parse(line));
    } catch {
      ignoredRecords += 1;
    }
  }

  return calculateWorkingSetFromEntries(entries, ignoredRecords);
}

module.exports = {
  calculateWorkingSetFromEntries,
  calculateWorkingSetFromJSONL,
};
