// @ts-check

/**
 * Returns true when a config key should be treated as sensitive.
 * @param {string} key
 * @returns {boolean}
 */
function isSensitiveConfigKey(key) {
  const normalizedKey = key.replace(/[^a-z0-9]/gi, "").toLowerCase();
  return /(token|apikey|authorization|password|passwd|privatekey|cookie|secret|credential|headers)/.test(normalizedKey);
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
    return Object.fromEntries(Object.entries(/** @type {Record<string, unknown>} */ value).map(([key, nestedValue]) => [key, isSensitiveConfigKey(key) ? "***REDACTED***" : redactSensitiveConfig(nestedValue)]));
  }
  return value;
}

module.exports = { isSensitiveConfigKey, redactSensitiveConfig };
