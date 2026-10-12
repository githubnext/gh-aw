// @ts-check

const { collectArtifactSecretValues, redactManifestValue } = require("./safe_output_manifest.cjs");
const { applyAddMaskRedaction } = require("./add_mask_redaction.cjs");
const { sanitizeContent } = require("./sanitize_content.cjs");

/**
 * @typedef {Object} DiagnosticOptions
 * @property {string[]} [secrets]
 * @property {string[]} [maskedValues]
 * @property {number} [maxLength]
 */

/**
 * Exact masks must run before patterns that can consume only a secret's prefix.
 * @param {string} text
 * @param {DiagnosticOptions} [options]
 * @returns {string}
 */
function redactDiagnosticText(text, { secrets = collectArtifactSecretValues(), maskedValues = [] } = {}) {
  const masks = [...secrets, ...maskedValues].sort((a, b) => b.length - a.length);
  return String(redactManifestValue(applyAddMaskRedaction(text, masks), []));
}

/**
 * Bound and fully normalize diagnostics before callers choose Markdown fences.
 * @param {string} text
 * @param {DiagnosticOptions} [options]
 * @returns {string}
 */
function redactAndBoundDiagnostics(text, options = {}) {
  const maxLength = options.maxLength ?? 8000;
  const normalize = value => {
    // Entity decoding and control/comment removal can expose another encoded layer.
    // Stabilize the bounded text so later sanitization cannot create a new fence.
    for (;;) {
      const sanitized = sanitizeContent(value, { maxLength });
      if (sanitized === value) return sanitized;
      value = sanitized;
    }
  };
  const normalized = normalize(redactDiagnosticText(text, options));
  return normalize(redactDiagnosticText(normalized, options));
}

/**
 * Withhold MCP script payloads, including outputs and errors that may echo inputs.
 * Apply the shared secret redactor to the remaining diagnostic metadata.
 * @param {string} message
 * @returns {string}
 */
function redactMCPScriptLogMessage(message) {
  const metadata = message
    .replace(
      /(\b(?:Invoking handler with args|(?:JavaScript|Python|Go|Shell) handler args|Input JSON \(\d+ bytes\)|stdout|stderr|Serialized result|Output file content|Parsed output|Shell handler completed, outputs|Error for notification|recv|send):\s*)[\s\S]*$/i,
      "$1[REDACTED]"
    )
    .replace(/(\bSet env:\s*[^=\r\n]+=)[\s\S]*$/, "$1[REDACTED]")
    .replace(/(\bargs=)[\s\S]*$/, "$1[REDACTED]")
    .replace(/(\b(?:error|threw|Stack trace):\s*)[\s\S]*$/i, "$1[REDACTED]");
  return redactDiagnosticText(metadata);
}

module.exports = { redactDiagnosticText, redactAndBoundDiagnostics, redactMCPScriptLogMessage };
