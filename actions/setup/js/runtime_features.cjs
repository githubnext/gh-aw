// @ts-check

/**
 * Parse GH_AW_RUNTIME_FEATURES into a key/value map.
 *
 * Supported line formats:
 *   - key
 *   - key=value
 *
 * Blank lines are ignored. Flags without an explicit value default to true.
 * Empty values (key=) are preserved as the empty string; use hasRuntimeFeature()
 * for presence checks when callers need to distinguish empty from missing.
 *
 * @param {string | undefined | null} raw
 * @returns {Record<string, string | boolean>}
 */
function parseRuntimeFeatures(raw) {
  if (!raw) {
    return {};
  }

  /** @type {Record<string, string | boolean>} */
  const features = {};

  for (const line of raw.split(/\r?\n/)) {
    const trimmed = line.trim();
    if (!trimmed) {
      continue;
    }
    const equalsIndex = trimmed.indexOf("=");
    if (equalsIndex === -1) {
      features[trimmed] = true;
      continue;
    }

    const key = trimmed.slice(0, equalsIndex).trim();
    if (!key) {
      continue;
    }
    features[key] = trimmed.slice(equalsIndex + 1).trim();
  }

  return features;
}

/**
 * @param {Record<string, string | boolean>} features
 * @param {string} key
 * @returns {boolean}
 */
function hasRuntimeFeature(features, key) {
  return Object.prototype.hasOwnProperty.call(features, key);
}

/**
 * @param {Record<string, string | boolean>} features
 * @param {string} key
 * @returns {string | boolean | undefined}
 */
function getRuntimeFeatureValue(features, key) {
  return features[key];
}

module.exports = {
  parseRuntimeFeatures,
  hasRuntimeFeature,
  getRuntimeFeatureValue,
};
