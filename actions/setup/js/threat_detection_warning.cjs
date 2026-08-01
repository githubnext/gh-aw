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

/**
 * Returns true when the reason indicates a tooling failure rather than an actual
 * security finding. Tooling failures (agent_failure, parse_error) mean the
 * detection engine itself crashed or could not produce a verdict — they should be
 * surfaced as a distinct infrastructure error, not as a security threat.
 *
 * @param {string | undefined | null} reason
 * @returns {boolean}
 */
function isToolingFailureReason(reason) {
  const normalized = String(reason || "").trim();
  return normalized === "agent_failure" || normalized === "parse_error";
}

/**
 * Returns the XML marker used to identify threat-engine-error output.
 * This marker is distinct from the real-threat marker so that automated tools
 * can distinguish a tooling failure from an actual security finding.
 *
 * @returns {string}
 */
function getThreatEngineErrorMarker() {
  return "<!-- gh-aw-threat-engine-error -->";
}

/**
 * Returns the marker template for configured engine-error message rendering.
 *
 * @returns {string}
 */
function getThreatEngineErrorMarkerTemplate() {
  return "<!-- gh-aw-threat-engine-error -->";
}

/**
 * Returns the review-warning presentation associated with a detection reason.
 * Centralizing these fields keeps admonition copy and marker routing in sync
 * across status messages, footers, and fallback pull request bodies.
 *
 * @param {string | undefined | null} reason
 * @returns {{admonition: string, title: string, summary: string, marker: string}}
 */
function getThreatWarningPresentation(reason) {
  if (isToolingFailureReason(reason)) {
    return {
      admonition: "WARNING",
      title: "threat detection engine error",
      summary: "The threat detection engine encountered an error and could not complete analysis. This is a tooling failure, not a security finding.",
      marker: getThreatEngineErrorMarker(),
    };
  }
  return {
    admonition: "CAUTION",
    title: "agentic threat detected",
    summary: "Threat detection flagged this output in warn mode. Manual review is REQUIRED before any follow-up automation.",
    marker: getThreatDetectedMarker(reason),
  };
}

/**
 * Returns the XML marker used to identify threat-detected output.
 * When the reason indicates a tooling failure (agent_failure, parse_error) a
 * distinct engine-error marker is returned so automated tools can distinguish
 * "detection engine crashed" from "detection engine found something".
 *
 * @param {string | undefined | null} reason
 * @returns {string}
 */
function getThreatDetectedMarker(reason) {
  if (isToolingFailureReason(reason)) {
    return getThreatEngineErrorMarker();
  }
  return "<!-- gh-aw-threat-detected -->";
}

/**
 * Returns the marker template for configured message rendering.
 * Always returns the real-threat marker; use getThreatEngineErrorMarkerTemplate()
 * for tooling-failure templates where the reason is known at template-build time.
 *
 * @returns {string}
 */
function getThreatDetectedMarkerTemplate() {
  return "<!-- gh-aw-threat-detected -->";
}

module.exports = {
  normalizeThreatKinds,
  getThreatWarningPresentation,
  getThreatDetectedMarker,
  getThreatDetectedMarkerTemplate,
  getThreatEngineErrorMarker,
  getThreatEngineErrorMarkerTemplate,
  getDetectionReasonText,
  isToolingFailureReason,
};
