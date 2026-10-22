// @ts-check

/**
 * Preserve detection masks in runner-private GITHUB_ENV, never artifact files.
 * This runner-only module must not be deployed to isolated MCP bundles.
 *
 * @param {string[]} values
 */
function persistDetectionMasks(values) {
  const savedMasks = JSON.stringify(values);
  // Register raw values and serialized state before later step setup prints it.
  for (const value of values) core.setSecret(value);
  core.setSecret(savedMasks);
  core.exportVariable("GH_AW_DETECTION_MASKED_VALUES", savedMasks);
}

module.exports = { persistDetectionMasks };
