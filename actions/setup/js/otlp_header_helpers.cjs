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
 * @returns {string[]} Unique secret values, longest first
 */
function getOTLPSecretValues() {
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
    for (const pair of (raw || "").split(",")) {
      const separator = pair.indexOf("=");
      if (separator <= 0) continue;
      const encoded = pair.slice(separator + 1).trim();
      if (encoded.length >= 4) secrets.add(encoded);
      // Parse individually so duplicate header names never overwrite a secret.
      for (const value of Object.values(parseOTLPHeaders(pair))) {
        for (const secret of getAuthorizationSecretValues(value)) secrets.add(secret);
      }
    }
  }
  for (const secret of [...secrets]) {
    secrets.add(JSON.stringify(secret).slice(1, -1));
  }
  return [...secrets].sort((a, b) => b.length - a.length);
}

module.exports = { parseOTLPHeaders, getAuthorizationSecretValues, getOTLPSecretValues };
