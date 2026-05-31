// @ts-check

/**
 * @param {unknown} value
 * @returns {string}
 */
function parsePositiveEffectiveTokenLimitString(value) {
  if (typeof value === "number" && Number.isFinite(value) && Number.isInteger(value) && value > 0) {
    return String(value);
  }

  if (typeof value !== "string") {
    return "";
  }

  const trimmed = value.trim();
  const match = /^([1-9]\d*)([kKmM])?$/.exec(trimmed);
  if (!match) {
    return "";
  }

  let parsed = BigInt(match[1]);
  const suffix = match[2]?.toLowerCase();
  if (suffix === "k") {
    parsed *= 1000n;
  } else if (suffix === "m") {
    parsed *= 1000000n;
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
