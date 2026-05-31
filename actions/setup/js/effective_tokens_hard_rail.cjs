// @ts-check

"use strict";

const MAX_EFFECTIVE_TOKENS_EXCEEDED_PATTERN = /maximum effective tokens exceeded/i;

/**
 * Detect the AWF effective-token hard rail in engine output.
 * This is non-retryable because the firewall will continue refusing inference
 * until the effective-token budget resets.
 *
 * @param {string} output
 * @returns {boolean}
 */
function isMaxEffectiveTokensExceededError(output) {
  return typeof output === "string" && MAX_EFFECTIVE_TOKENS_EXCEEDED_PATTERN.test(output);
}

module.exports = {
  MAX_EFFECTIVE_TOKENS_EXCEEDED_PATTERN,
  isMaxEffectiveTokensExceededError,
};
