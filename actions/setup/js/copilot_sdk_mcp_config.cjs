// @ts-check
"use strict";

const fs = require("node:fs");
const path = require("node:path");
const { createHash, timingSafeEqual } = require("node:crypto");
const { resolveCopilotSDKMCPConfigPath } = require("./copilot_sdk_mcp_config_path.cjs");

const COPILOT_SDK_MCP_CONFIG_SHA256 = "GH_AW_COPILOT_SDK_MCP_CONFIG_SHA256";

/**
 * Bind the exact catalog bytes to the job and run attempt, not just its path.
 * @param {string | Buffer} source
 * @param {NodeJS.ProcessEnv} [environment]
 * @returns {string}
 */
function digestCopilotSDKMCPConfig(source, environment = process.env) {
  const identity = [environment.GITHUB_RUN_ID, environment.GITHUB_RUN_ATTEMPT, environment.GITHUB_JOB];
  if (identity.some(value => !value)) {
    throw new Error("GITHUB_RUN_ID, GITHUB_RUN_ATTEMPT, and GITHUB_JOB are required for Copilot SDK MCP configuration integrity");
  }
  return createHash("sha256").update(JSON.stringify(identity)).update("\0").update(source).digest("hex");
}

/**
 * @param {string} source
 * @returns {string} Run-bound digest to publish through the trusted step environment.
 */
function writeCopilotSDKMCPConfig(source) {
  const digest = digestCopilotSDKMCPConfig(source);
  const outputPath = resolveCopilotSDKMCPConfigPath();
  const runnerTemp = path.resolve(process.env.RUNNER_TEMP || "");
  let directory = runnerTemp;
  for (const name of ["gh-aw", "mcp-config"]) {
    directory = path.join(directory, name);
    fs.mkdirSync(directory, { recursive: true, mode: 0o700 });
    const stat = fs.lstatSync(directory);
    if (!stat.isDirectory() || (stat.mode & 0o022) !== 0) {
      throw new Error("Copilot SDK MCP configuration directory must not be a symlink or writable by other users");
    }
  }

  // Rename a fresh private file instead of truncating a possibly linked old file.
  const staging = fs.mkdtempSync(path.join(directory, ".copilot-sdk-"));
  const temporaryPath = path.join(staging, "copilot-sdk.json");
  try {
    fs.writeFileSync(temporaryPath, source, { flag: "wx", mode: 0o600 });
    fs.renameSync(temporaryPath, outputPath);
  } finally {
    fs.rmSync(temporaryPath, { force: true });
    fs.rmdirSync(staging);
  }
  return digest;
}

/**
 * The gateway launches the converter as plain Node, without github-script core.
 * @param {string} source
 */
function publishCopilotSDKMCPConfig(source) {
  const envFile = process.env.GITHUB_ENV;
  if (!envFile) throw new Error("GITHUB_ENV is required to publish Copilot SDK MCP configuration integrity");
  const digest = writeCopilotSDKMCPConfig(source);
  try {
    const descriptor = fs.openSync(envFile, fs.constants.O_WRONLY | fs.constants.O_APPEND | fs.constants.O_NOFOLLOW | fs.constants.O_NONBLOCK);
    try {
      if (!fs.fstatSync(descriptor).isFile()) throw new Error("Invalid GitHub environment file");
      fs.writeFileSync(descriptor, `${COPILOT_SDK_MCP_CONFIG_SHA256}=${digest}\n`, "utf8");
    } finally {
      fs.closeSync(descriptor);
    }
  } catch {
    throw new Error("Cannot publish Copilot SDK MCP configuration integrity");
  }
}

/**
 * @param {string} configPath
 * @returns {string}
 */
function readCopilotSDKMCPConfig(configPath) {
  /** @type {Buffer} */
  let source;
  try {
    if (!fs.lstatSync(path.dirname(configPath)).isDirectory() || fs.constants.O_NOFOLLOW === undefined) {
      throw new Error("Unsafe MCP configuration path");
    }
    const descriptor = fs.openSync(configPath, fs.constants.O_RDONLY | fs.constants.O_NOFOLLOW | fs.constants.O_NONBLOCK);
    try {
      const stat = fs.fstatSync(descriptor);
      if (!stat.isFile() || stat.nlink !== 1) throw new Error("MCP configuration must be a single-link regular file");
      source = fs.readFileSync(descriptor);
    } finally {
      fs.closeSync(descriptor);
    }
  } catch {
    throw new Error("Cannot read generated Copilot SDK MCP configuration");
  }

  const expected = process.env[COPILOT_SDK_MCP_CONFIG_SHA256];
  if (!expected || !/^[a-f0-9]{64}$/.test(expected)) {
    throw new Error("Generated Copilot SDK MCP configuration integrity is required");
  }
  const actual = digestCopilotSDKMCPConfig(source);
  if (!timingSafeEqual(Buffer.from(actual, "hex"), Buffer.from(expected, "hex"))) {
    throw new Error("Copilot SDK MCP configuration integrity check failed");
  }
  return source.toString("utf8");
}

module.exports = { COPILOT_SDK_MCP_CONFIG_SHA256, digestCopilotSDKMCPConfig, readCopilotSDKMCPConfig, writeCopilotSDKMCPConfig, publishCopilotSDKMCPConfig };
