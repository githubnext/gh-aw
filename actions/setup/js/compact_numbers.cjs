// @ts-check

/**
 * Formats a non-negative integer in a compact, human-readable form.
 *
 * Ranges:
 *   < 1,000        → exact integer (e.g. "900")
 *   1,000–999,999  → Xk with one decimal when non-zero (e.g. "1.2K", "450K")
 *   >= 1,000,000   → Xm with one decimal when non-zero (e.g. "1.2M", "3M")
 *
 * @param {number} value
 * @returns {string}
 */
function formatCompactInteger(value) {
  const normalized = Number.isFinite(value) ? Math.max(0, Math.round(value)) : 0;
  if (normalized < 1000) return String(normalized);
  if (normalized < 1_000_000) {
    const k = (normalized / 1000).toFixed(1).replace(/\.0$/, "");
    return k === "1000" ? "1M" : `${k}K`;
  }
  return `${(normalized / 1_000_000).toFixed(1).replace(/\.0$/, "")}M`;
}

module.exports = {
  formatCompactInteger,
};
