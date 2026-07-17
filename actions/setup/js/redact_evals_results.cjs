// @ts-check
/// <reference types="@actions/github-script" />

const fs = require("fs");
const { EVALS_OUTPUT_PATH } = require("./evals_constants.cjs");
const { main: redactWorkspaceSecrets, redactSecrets, redactBuiltInPatterns, extractMCPGatewayTokens, MCP_GATEWAY_CONFIG_PATHS } = require("./redact_secrets.cjs");

function getSecretValues() {
  const secretNames = (process.env.GH_AW_SECRET_NAMES || "")
    .split(",")
    .map(name => name.trim())
    .filter(Boolean);

  /** @type {string[]} */
  const secretValues = [];
  for (const secretName of secretNames) {
    const value = process.env[`SECRET_${secretName}`];
    if (typeof value === "string" && value.trim() !== "") {
      secretValues.push(value.trim());
    }
  }

  secretValues.push(...extractMCPGatewayTokens(MCP_GATEWAY_CONFIG_PATHS));
  return secretValues;
}

function verifyRedaction() {
  if (!fs.existsSync(EVALS_OUTPUT_PATH)) {
    return;
  }

  let content;
  try {
    content = fs.readFileSync(EVALS_OUTPUT_PATH, "utf8");
  } catch (err) {
    throw new Error(`Failed to read file ${EVALS_OUTPUT_PATH}: ${String(err)}`, { cause: err });
  }
  const secretValues = getSecretValues();
  const lingeringRedactions = redactBuiltInPatterns(content).redactionCount + redactSecrets(content, secretValues).redactionCount;

  if (lingeringRedactions > 0) {
    core.setFailed(`Secret redaction verification failed for ${EVALS_OUTPUT_PATH}: ${lingeringRedactions} unredacted value(s) remain`);
  }
}

async function main() {
  await redactWorkspaceSecrets();
  verifyRedaction();
}

module.exports = { main, getSecretValues, verifyRedaction };
