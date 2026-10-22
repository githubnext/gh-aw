// @ts-check

// Keep header collection dependency-free for isolated MCP redaction deployments.
// Runner workflow-command masking and telemetry export must not be loaded here.

/**
 * Parse OpenTelemetry key=value header pairs, preserving valid pairs when another
 * pair has malformed percent encoding.
 * @param {string} raw
 * @returns {Record<string, string>}
 */
function parseOTLPHeaders(raw) {
  if (!raw || !raw.trim()) return {};
  /** @type {Record<string, string>} */
  const result = {};
  for (const pair of raw.split(",")) {
    const eqIdx = pair.indexOf("=");
    if (eqIdx <= 0) continue;
    try {
      const key = decodeURIComponent(pair.slice(0, eqIdx)).trim();
      const value = decodeURIComponent(pair.slice(eqIdx + 1)).trim();
      if (key) result[key] = value;
    } catch {
      // Ignore only the malformed pair so other configured headers remain usable.
    }
  }
  return result;
}

/**
 * Collect header credentials even when tools print them without their auth scheme
 * or decode a Basic-auth password from a stored, encoded secret.
 * Skip each value shorter than four characters to avoid masking unrelated output,
 * matching the OTLP shell masking minimum in mask_otlp_common.sh.
 * @param {string} value
 * @returns {string[]}
 */
function getAuthorizationSecretValues(value) {
  const minMaskLength = 4;
  const values = [];
  if (value.length >= minMaskLength) values.push(value);
  const match = /^(Basic|Bearer)\s+(.+)$/i.exec(value.trim());
  if (!match) return values;
  const credential = match[2].trim();
  if (credential.length >= minMaskLength) values.push(credential);
  if (match[1].toLowerCase() !== "basic" || !/^[A-Za-z0-9+/]+={0,2}$/.test(credential)) return values;
  const decoded = Buffer.from(credential, "base64").toString("utf8");
  const separator = decoded.indexOf(":");
  if (separator < 0) return values;
  if (decoded.length >= minMaskLength) values.push(decoded);
  const password = decoded.slice(separator + 1);
  if (password.length >= minMaskLength) values.push(password);
  return values;
}

/**
 * Collect OTLP credentials independently of setup log capture or token prefixes.
 * Include headers from disabled endpoints: their credentials still exist in env.
 * @param {boolean} forRunnerMasking - Avoid registering plain placeholder values as runner masks
 * @returns {string[]} Unique secret values, longest first
 */
function collectOTLPSecretValues(forRunnerMasking) {
  const headers = [process.env.OTEL_EXPORTER_OTLP_HEADERS, process.env.GH_AW_OTLP_ALL_HEADERS];
  const endpointsRaw = process.env.GH_AW_OTLP_ENDPOINTS;
  if (endpointsRaw) {
    let endpoints;
    try {
      endpoints = JSON.parse(endpointsRaw);
    } catch {
      global.core?.warning?.("Unable to collect OTLP endpoint credentials: invalid endpoint JSON");
    }
    if (Array.isArray(endpoints)) {
      for (const endpoint of endpoints) {
        if (typeof endpoint?.headers === "string") headers.push(endpoint.headers);
      }
    }
  }
  if (!headers.some(Boolean)) return [];
  /** @type {Set<string>} */
  const secrets = new Set();
  for (const raw of headers) {
    if (forRunnerMasking && raw && raw.length >= 4) secrets.add(raw);
    for (const pair of (raw || "").split(",")) {
      const separator = pair.indexOf("=");
      if (separator <= 0) continue;
      const encoded = pair.slice(separator + 1).trim();
      // Parse individually so duplicate header names never overwrite a secret.
      const values = Object.values(parseOTLPHeaders(pair));
      const value = values[0] ?? encoded;
      const isAuthorization = /^(Basic|Bearer)\s+(.+)$/i.test(value.trim());
      if (forRunnerMasking && !isAuthorization && (value.length < 8 || /^\d+$/.test(value))) continue;
      if (encoded.length >= 4) secrets.add(encoded);
      for (const value of values) {
        for (const secret of getAuthorizationSecretValues(value)) secrets.add(secret);
      }
    }
  }
  for (const secret of [...secrets]) {
    secrets.add(JSON.stringify(secret).slice(1, -1));
  }
  return [...secrets].sort((a, b) => b.length - a.length);
}

/**
 * Collect credentials for artifact and step-summary redaction, including plain
 * header values at the four-character minimum.
 * @returns {string[]}
 */
function getOTLPSecretValues() {
  return collectOTLPSecretValues(false);
}

/**
 * Select runner masks without suppressing outputs matching plain placeholders.
 * Auth schemes retain the four-character credential minimum, including decoded
 * Basic passwords. Full headers remain masked even when a plain value is skipped.
 * @returns {string[]}
 */
function getOTLPRunnerMaskValues() {
  return collectOTLPSecretValues(true);
}

module.exports = { parseOTLPHeaders, getAuthorizationSecretValues, getOTLPSecretValues, getOTLPRunnerMaskValues };
