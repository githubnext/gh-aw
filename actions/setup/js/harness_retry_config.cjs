"use strict";

const { getEnvPositiveIntOrDefault, parseStrictPositiveInteger } = require("./copilot_sdk_permissions.cjs");

// Maximum number of retry attempts after the initial run
const DEFAULT_MAX_RETRIES = 3;
// Initial delay in milliseconds before the first retry
const DEFAULT_INITIAL_DELAY_MS = 5000;
// Multiplier applied to delay after each retry
const DEFAULT_BACKOFF_MULTIPLIER = 2;
// Maximum delay cap in milliseconds
const DEFAULT_MAX_DELAY_MS = 60000;
const HARNESS_MAX_RETRIES_ENV = "GH_AW_HARNESS_MAX_RETRIES";
const HARNESS_INITIAL_DELAY_MS_ENV = "GH_AW_HARNESS_INITIAL_DELAY_MS";
const HARNESS_BACKOFF_MULTIPLIER_ENV = "GH_AW_HARNESS_BACKOFF_MULTIPLIER";
const HARNESS_MAX_DELAY_MS_ENV = "GH_AW_HARNESS_MAX_DELAY_MS";

/**
 * @param {((message: string) => void) | undefined} logger
 * @param {string} envVar
 * @param {string | undefined} rawValue
 * @param {number} defaultValue
 */
function logInvalidEnvValue(logger, envVar, rawValue, defaultValue) {
  if (logger) {
    logger(`warning: ignoring invalid ${envVar}=${JSON.stringify(rawValue)}; using default ${defaultValue}`);
  }
}

/**
 * @param {NodeJS.ProcessEnv} env
 * @param {string} envVar
 * @param {number} defaultValue
 * @param {(message: string) => void} [logger]
 * @returns {number}
 */
function readEnvPositiveIntOrDefault(env, envVar, defaultValue, logger) {
  const rawValue = env[envVar];
  if (rawValue == null || rawValue === "") {
    return defaultValue;
  }
  if (parseStrictPositiveInteger(rawValue) !== undefined) {
    return getEnvPositiveIntOrDefault(envVar, defaultValue, env);
  }
  logInvalidEnvValue(logger, envVar, rawValue, defaultValue);
  return defaultValue;
}

/**
 * @param {NodeJS.ProcessEnv} env
 * @param {string} envVar
 * @param {number} defaultValue
 * @param {(message: string) => void} [logger]
 * @returns {number}
 */
function readEnvNonNegativeIntOrDefault(env, envVar, defaultValue, logger) {
  const rawValue = env[envVar];
  if (rawValue == null || rawValue === "") {
    return defaultValue;
  }
  const trimmed = String(rawValue).trim();
  if (/^\d+$/.test(trimmed)) {
    const parsed = Number.parseInt(trimmed, 10);
    if (Number.isSafeInteger(parsed)) {
      return parsed;
    }
  }
  logInvalidEnvValue(logger, envVar, rawValue, defaultValue);
  return defaultValue;
}

/**
 * @param {string | undefined} rawValue
 * @param {{envVar: string, defaultValue: number, minimum: number, allowFloat?: boolean, logger?: (message: string) => void}} options
 * @returns {number}
 */
function parseRetryConfigNumber(rawValue, { envVar, defaultValue, minimum, allowFloat = false, logger }) {
  if (rawValue == null || rawValue === "") {
    return defaultValue;
  }
  const parsed = Number(rawValue);
  const isValidNumber = Number.isFinite(parsed) && parsed >= minimum && (allowFloat || Number.isInteger(parsed));
  if (isValidNumber) {
    return parsed;
  }
  logInvalidEnvValue(logger, envVar, rawValue, defaultValue);
  return defaultValue;
}

/**
 * @param {NodeJS.ProcessEnv} [env]
 * @param {(message: string) => void} [logger]
 * @returns {{maxRetries: number, initialDelayMs: number, backoffMultiplier: number, maxDelayMs: number}}
 */
function resolveRetryConfig(env = process.env, logger = () => {}) {
  const maxRetries = readEnvNonNegativeIntOrDefault(env, HARNESS_MAX_RETRIES_ENV, DEFAULT_MAX_RETRIES, logger);
  const initialDelayMs = readEnvPositiveIntOrDefault(env, HARNESS_INITIAL_DELAY_MS_ENV, DEFAULT_INITIAL_DELAY_MS, logger);
  const backoffMultiplier = parseRetryConfigNumber(env[HARNESS_BACKOFF_MULTIPLIER_ENV], {
    envVar: HARNESS_BACKOFF_MULTIPLIER_ENV,
    defaultValue: DEFAULT_BACKOFF_MULTIPLIER,
    minimum: 1,
    allowFloat: true,
    logger,
  });
  let maxDelayMs = readEnvPositiveIntOrDefault(env, HARNESS_MAX_DELAY_MS_ENV, DEFAULT_MAX_DELAY_MS, logger);
  if (maxDelayMs < initialDelayMs) {
    logger(`warning: ${HARNESS_MAX_DELAY_MS_ENV}=${maxDelayMs} is lower than ${HARNESS_INITIAL_DELAY_MS_ENV}=${initialDelayMs}; clamping max delay to initial delay`);
    maxDelayMs = initialDelayMs;
  }
  return { maxRetries, initialDelayMs, backoffMultiplier, maxDelayMs };
}

module.exports = {
  resolveRetryConfig,
};
