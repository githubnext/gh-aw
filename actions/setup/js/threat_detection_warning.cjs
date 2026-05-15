// @ts-check

/**
 * Normalizes one or more threat-kind labels for XML marker use.
 * Accepts comma/space-delimited values and falls back to "unknown".
 *
 * @param {string | undefined | null} reason
 * @returns {string}
 */
function normalizeThreatKinds(reason) {
  const value = String(reason || "").trim();
  if (!value) return "unknown";
  const kinds = value
    .split(/[\s,]+/)
    .map(kind => kind.toLowerCase())
    .filter(Boolean)
    // Marker values are machine-readable tokens; keep a strict safe charset.
    .map(kind => kind.replace(/[^a-z0-9_-]/g, ""))
    .filter(Boolean);
  return kinds.length > 0 ? Array.from(new Set(kinds)).join(",") : "unknown";
}

/**
 * Returns the XML marker used to identify threat-detected output.
 *
 * @param {string | undefined | null} reason
 * @returns {string}
 */
function getThreatDetectedMarker(reason) {
  return "<!-- gh-aw-threat-detected -->";
}

/**
 * Returns the marker template for configured message rendering.
 *
 * @returns {string}
 */
function getThreatDetectedMarkerTemplate() {
  return "<!-- gh-aw-threat-detected -->";
}

/**
 * Returns a human-readable reason text for detection warnings.
 *
 * @param {string | undefined | null} reason
 * @returns {string}
 */
function getDetectionReasonText(reason) {
  const reasonDescriptions = {
    threat_detected: "Potential security threats were detected in the agent output.",
    agent_failure: "The threat detection engine failed to produce results.",
    parse_error: "The threat detection results could not be parsed.",
  };
  const normalizedReason = String(reason || "").trim();
  return reasonDescriptions[normalizedReason] || "The threat detection analysis could not be completed.";
}

module.exports = {
  normalizeThreatKinds,
  getThreatDetectedMarker,
  getThreatDetectedMarkerTemplate,
  getDetectionReasonText,
};
