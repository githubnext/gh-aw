// @ts-check
/// <reference types="@actions/github-script" />

/**
 * @param {string | undefined} raw
 * @returns {boolean}
 */
function resolveDetectionContinueOnError(raw) {
  const value = (raw ?? "").trim().toLowerCase();
  if (value === "" || value === "true") return true;
  if (value === "false") return false;
  throw new Error('GH_AW_DEFAULT_DETECTION_CONTINUE_ON_ERROR must be "true" or "false"; unset or empty uses warning mode. Correct the effective Actions variable or override safe-outputs.threat-detection.continue-on-error.');
}

function main() {
  core.setOutput("continue_on_error", String(resolveDetectionContinueOnError(process.env.GH_AW_DEFAULT_DETECTION_CONTINUE_ON_ERROR)));
}

module.exports = { main, resolveDetectionContinueOnError };
