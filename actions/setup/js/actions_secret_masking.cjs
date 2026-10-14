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
 * Mask header credentials even when tools print them without their auth scheme
 * or decode a Basic-auth password from a stored, encoded secret.
 * Skip each value shorter than four characters to avoid masking unrelated output,
 * matching the OTLP shell masking minimum in mask_otlp_common.sh.
 * @param {string} value
 */
function maskAuthorizationValue(value) {
  const minMaskLength = 4;
  if (value.length >= minMaskLength) maskSecret(value);
  const match = /^(Basic|Bearer)\s+(.+)$/i.exec(value.trim());
  if (!match) return;
  const credential = match[2].trim();
  if (credential.length >= minMaskLength) maskSecret(credential);
  if (match[1].toLowerCase() !== "basic" || !/^[A-Za-z0-9+/]+={0,2}$/.test(credential)) return;
  const decoded = Buffer.from(credential, "base64").toString("utf8");
  const separator = decoded.indexOf(":");
  if (separator < 0) return;
  if (decoded.length >= minMaskLength) maskSecret(decoded);
  const password = decoded.slice(separator + 1);
  if (password.length >= minMaskLength) maskSecret(password);
}

module.exports = { escapeWorkflowCommandValue, maskSecret, maskAuthorizationValue };
