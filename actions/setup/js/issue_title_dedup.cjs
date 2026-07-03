// @ts-check

const { levenshteinDistance } = require("./levenshtein_distance.cjs");

/**
 * Parse create-issue deduplication config.
 * - true / "true"   => enabled with exact-match distance 0
 * - false / "false" => disabled
 *
 * @param {unknown} value
 * @returns {{ enabled: boolean, maxDistance: number }}
 */
function parseDeduplicateByTitle(value) {
  if (value === undefined || value === null || value === false || value === "false") {
    return { enabled: false, maxDistance: 0 };
  }
  if (value === true || value === "true") {
    return { enabled: true, maxDistance: 0 };
  }
  throw new Error("deduplicate-by-title must be a boolean or a resolved boolean string ('true' or 'false')");
}

/**
 * Normalize a title for deduplication comparisons.
 * @param {string} title
 * @returns {string}
 */
function normalizeTitleForDedup(title) {
  return String(title ?? "")
    .toLowerCase()
    .replace(/\s+/g, " ")
    .trim();
}

/**
 * @typedef {{ title: string, normalizedTitle?: string }} TitleCandidate
 */

/**
 * Find a duplicate candidate by Levenshtein distance threshold.
 *
 * @param {string} normalizedTitle
 * @param {TitleCandidate[]} candidates
 * @param {number} maxDistance
 * @returns {{ title: string, distance: number } | null}
 */
function findDuplicateByTitle(normalizedTitle, candidates, maxDistance) {
  let bestMatch = null;

  for (const candidate of candidates) {
    const candidateTitle = normalizeTitleForDedup(candidate.normalizedTitle || candidate.title);
    const distance = levenshteinDistance(normalizedTitle, candidateTitle);
    if (distance <= maxDistance && (!bestMatch || distance < bestMatch.distance)) {
      bestMatch = { title: candidate.title, distance };
      if (distance === 0) {
        return bestMatch;
      }
    }
  }

  return bestMatch;
}

module.exports = {
  parseDeduplicateByTitle,
  normalizeTitleForDedup,
  findDuplicateByTitle,
};
