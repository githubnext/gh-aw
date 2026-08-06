// @ts-check
"use strict";

/**
 * Read a secret from the environment and register its value for masking in
 * GitHub Actions logs.
 *
 * @param {string} name
 * @returns {string|undefined}
 */
function readSecretEnv(name) {
  const value = process.env[name];
  if (value) {
    const { ensureCoreSetSecret } = require("./shim.cjs");
    ensureCoreSetSecret();
    core.setSecret(value);
  }
  return value;
}

module.exports = { readSecretEnv };
