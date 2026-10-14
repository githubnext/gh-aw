// @ts-check

/**
 * Escape a value for a GitHub Actions workflow command payload.
 *
 * @param {string} value
 * @returns {string}
 */
function escapeWorkflowCommandValue(value) {
  return value.replace(/%/g, "%25").replace(/\r/g, "%0D").replace(/\n/g, "%0A");
}

/**
 * Mask a secret in the surrounding GitHub Actions step when masking is available.
 *
 * Plain Node entry points such as setup.sh-loaded scripts do not have the real
 * @actions/core object, but GitHub Actions still processes add-mask workflow
 * commands emitted by the process.
 *
 * @param {unknown} value
 */
function maskSecret(value) {
  if (value === undefined || value === null) return;
  const secret = String(value);
  if (!secret) return;

  const setSecret = global.core?.setSecret;
  if (typeof setSecret === "function" && !("__ghAwUnavailable" in setSecret && setSecret.__ghAwUnavailable)) {
    setSecret.call(global.core, secret);
    return;
  }

  // shim.cjs marks its throwing placeholder so Actions-side plain Node callers
  // can still mask via workflow commands without enabling masking in MCP shims.
  if (process.env.GITHUB_ACTIONS === "true") {
    process.stdout.write(`::add-mask::${escapeWorkflowCommandValue(secret)}\n`);
  }
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

/** @param {string} value */
function maskAuthorizationValue(value) {
  for (const secret of getAuthorizationSecretValues(value)) maskSecret(secret);
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
  const { parseOTLPHeaders } = require("./send_otlp_span.cjs");
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

module.exports = { escapeWorkflowCommandValue, maskSecret, maskAuthorizationValue, getOTLPSecretValues };
