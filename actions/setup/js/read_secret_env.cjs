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
    const coreShim = ensureCoreSetSecret();
    coreShim.setSecret(value);
  }
  return value;
}

function isSecretEnvName(name) {
  return /(?:^|_)(?:TOKEN|SECRET|PASSWORD|KEY|CREDENTIAL|AUTH|PAT)(?:_|$)/.test(name);
}

function maskSecretEnvValues(env = process.env) {
  const { ensureCoreSetSecret } = require("./shim.cjs");
  const coreShim = ensureCoreSetSecret();
  let masked = 0;
  for (const [name, value] of Object.entries(env)) {
    if (value && isSecretEnvName(name)) {
      coreShim.setSecret(value);
      masked++;
    }
  }
  return masked;
}

module.exports = { isSecretEnvName, maskSecretEnvValues, readSecretEnv };
