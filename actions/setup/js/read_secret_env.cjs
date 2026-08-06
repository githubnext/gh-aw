// @ts-check
"use strict";

require("./shim.cjs");

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
    core.setSecret(value);
  }
  return value;
}

module.exports = { readSecretEnv };
