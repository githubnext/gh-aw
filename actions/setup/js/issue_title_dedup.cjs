// @ts-check

const { levenshteinDistance } = require("./levenshtein_distance.cjs");
const MAX_DEDUPLICATE_BY_TITLE_DISTANCE = 100;
const DEFAULT_CREATE_ISSUE_TITLE_DEDUP_ROLLOUT_PERCENT = 50;
const FNV1A_OFFSET_BASIS_32 = 2166136261;
const FNV1A_PRIME_32 = 16777619;

/**
 * Parse create-issue deduplication config.
 * - true  => enabled with exact-match distance 0
 * - false => disabled
 * - N     => enabled with Levenshtein max distance N
 *
 * @param {unknown} value
 * @returns {{ enabled: boolean, maxDistance: number }}
 */
function parseDeduplicateByTitle(value) {
  if (value === undefined || value === null || value === false) {
    return { enabled: false, maxDistance: 0 };
  }
  if (value === true) {
    return { enabled: true, maxDistance: 0 };
  }
  if (typeof value === "number" && Number.isFinite(value) && Number.isInteger(value) && value >= 0 && value <= MAX_DEDUPLICATE_BY_TITLE_DISTANCE) {
    return { enabled: true, maxDistance: value };
  }
  throw new Error(`deduplicate-by-title must be a boolean or a non-negative integer (0-${MAX_DEDUPLICATE_BY_TITLE_DISTANCE})`);
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

/**
 * Parse rollout percentage for create-issue title deduplication.
 *
 * @param {unknown} value
 * @returns {number}
 */
function parseCreateIssueTitleDedupRolloutPercent(value) {
  if (value === undefined || value === null || value === "") {
    return DEFAULT_CREATE_ISSUE_TITLE_DEDUP_ROLLOUT_PERCENT;
  }
  const parsed = Number(value);
  if (Number.isFinite(parsed) && Number.isInteger(parsed) && parsed >= 0 && parsed <= 100) {
    return parsed;
  }
  return DEFAULT_CREATE_ISSUE_TITLE_DEDUP_ROLLOUT_PERCENT;
}

/**
 * Deterministically map a seed string to a bucket in [0, 99].
 * Returns 100 for empty seeds (out-of-rollout).
 *
 * @param {string} seed
 * @returns {number}
 */
function dedupRolloutBucket(seed) {
  if (!seed) {
    return 100;
  }
  let hash = FNV1A_OFFSET_BASIS_32;
  for (let i = 0; i < seed.length; i += 1) {
    hash ^= seed.charCodeAt(i);
    hash = Math.imul(hash, FNV1A_PRIME_32);
  }
  return (hash >>> 0) % 100;
}

/**
 * Resolve effective create-issue title dedup configuration.
 * Explicit config wins; otherwise a deterministic rollout is applied.
 *
 * @param {unknown} value
 * @param {string} rolloutSeed
 * @param {number} rolloutPercent
 * @returns {{ enabled: boolean, maxDistance: number }}
 */
function resolveDeduplicateByTitle(value, rolloutSeed, rolloutPercent) {
  if (value !== undefined) {
    return parseDeduplicateByTitle(value);
  }

  if (rolloutPercent <= 0) {
    return { enabled: false, maxDistance: 0 };
  }

  const normalizedSeed = String(rolloutSeed || "").trim();
  if (!normalizedSeed) {
    return { enabled: false, maxDistance: 0 };
  }

  if (rolloutPercent >= 100) {
    return { enabled: true, maxDistance: 0 };
  }

  const bucket = dedupRolloutBucket(normalizedSeed);
  return {
    enabled: bucket < rolloutPercent,
    maxDistance: 0,
  };
}

module.exports = {
  parseDeduplicateByTitle,
  normalizeTitleForDedup,
  findDuplicateByTitle,
  parseCreateIssueTitleDedupRolloutPercent,
  resolveDeduplicateByTitle,
  dedupRolloutBucket,
};
