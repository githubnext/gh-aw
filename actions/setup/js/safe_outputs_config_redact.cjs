// @ts-check

/**
 * Returns true when a config key should be treated as sensitive (e.g. tokens).
 * @param {string} key
 * @returns {boolean}
 */
function isSensitiveConfigKey(key) {
  return /token/i.test(key);
}

/**
 * Recursively replaces any config value whose key is sensitive with a redaction marker.
 * Safe to pass to JSON.stringify for debug logging.
 * @param {unknown} value
 * @returns {unknown}
 */
function redactSensitiveConfig(value) {
  if (Array.isArray(value)) {
    return value.map(redactSensitiveConfig);
  }
  if (value && typeof value === "object") {
    return Object.fromEntries(
      Object.entries(/** @type {Record<string, unknown>} */ (value)).map(([key, nestedValue]) => [
        key,
        isSensitiveConfigKey(key) ? "***REDACTED***" : redactSensitiveConfig(nestedValue),
      ]),
    );
  }
  return value;
}

module.exports = { isSensitiveConfigKey, redactSensitiveConfig };
