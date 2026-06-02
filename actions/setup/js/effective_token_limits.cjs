// @ts-check

const POSITIVE_TOKEN_LIMIT_WITH_SUFFIX_REGEX = /^([1-9]\d*)([kKmM])?$/;

/**
 * @param {unknown} value
 * @returns {bigint|null}
 */
function parsePositiveEffectiveTokenLimitBigInt(value) {
  if (typeof value === "number" && Number.isFinite(value) && Number.isSafeInteger(value) && value > 0) {
    return BigInt(value);
  }

  if (typeof value !== "string") {
    return null;
  }

  const trimmed = value.trim();
  const match = POSITIVE_TOKEN_LIMIT_WITH_SUFFIX_REGEX.exec(trimmed);
  if (!match) {
    return null;
  }

  let parsed = BigInt(match[1]);
  const suffix = match[2]?.toLowerCase();
  if (suffix === "k") {
    parsed *= 1000n;
  } else if (suffix === "m") {
    parsed *= 1000000n;
  }

  return parsed;
}

/**
 * @param {unknown} value
 * @returns {string}
 */
function parsePositiveEffectiveTokenLimitString(value) {
  const parsed = parsePositiveEffectiveTokenLimitBigInt(value);
  if (parsed == null) {
    return "";
  }
  return parsed.toString();
}

/**
 * @param {unknown} value
 * @returns {number}
 */
function parsePositiveEffectiveTokenLimitNumber(value) {
  const normalized = parsePositiveEffectiveTokenLimitString(value);
  if (!normalized) {
    return 0;
  }

  const parsed = Number(normalized);
  return Number.isSafeInteger(parsed) && parsed > 0 ? parsed : 0;
}

module.exports = {
  parsePositiveEffectiveTokenLimitString,
  parsePositiveEffectiveTokenLimitNumber,
};
