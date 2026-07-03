"use strict";

const { getEnvPositiveIntOrDefault } = require("./copilot_sdk_permissions.cjs");

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
const INVALID_POSITIVE_INT_FALLBACK = Number.NaN;

/**
 * @param {((message: string) => void) | undefined} logger
 * @param {string} envVar
 * @param {string | undefined} rawValue
 * @param {number} defaultValue
 */
function logInvalidEnvValue(logger, envVar, rawValue, defaultValue) {
  if (typeof logger === "function") {
    logger(`warning: ignoring invalid ${envVar}=${JSON.stringify(rawValue)}; using default ${defaultValue}`);
  }
}

/**
 * @param {NodeJS.ProcessEnv} env
 * @param {{envVar: string, defaultValue: number, minimum: number, allowFloat?: boolean, logger?: (message: string) => void}} options
 * @returns {number}
 */
function parseRetryConfigNumber(env, { envVar, defaultValue, minimum, allowFloat = false, logger }) {
  const rawValue = env[envVar];
  if (rawValue == null || rawValue === "") {
    return defaultValue;
  }
  if (!allowFloat && minimum >= 1) {
    const parsedPositiveInt = getEnvPositiveIntOrDefault(envVar, INVALID_POSITIVE_INT_FALLBACK, env);
    if (Number.isSafeInteger(parsedPositiveInt) && parsedPositiveInt >= minimum) {
      return parsedPositiveInt;
    }
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
  const maxRetries = parseRetryConfigNumber(env, {
    envVar: HARNESS_MAX_RETRIES_ENV,
    defaultValue: DEFAULT_MAX_RETRIES,
    minimum: 0,
    logger,
  });
  const initialDelayMs = parseRetryConfigNumber(env, {
    envVar: HARNESS_INITIAL_DELAY_MS_ENV,
    defaultValue: DEFAULT_INITIAL_DELAY_MS,
    minimum: 1,
    logger,
  });
  const backoffMultiplier = parseRetryConfigNumber(env, {
    envVar: HARNESS_BACKOFF_MULTIPLIER_ENV,
    defaultValue: DEFAULT_BACKOFF_MULTIPLIER,
    minimum: 1,
    allowFloat: true,
    logger,
  });
  let maxDelayMs = parseRetryConfigNumber(env, {
    envVar: HARNESS_MAX_DELAY_MS_ENV,
    defaultValue: DEFAULT_MAX_DELAY_MS,
    minimum: 1,
    logger,
  });
  if (maxDelayMs < initialDelayMs) {
    if (typeof logger === "function") {
      logger(`warning: ${HARNESS_MAX_DELAY_MS_ENV}=${maxDelayMs} is lower than ${HARNESS_INITIAL_DELAY_MS_ENV}=${initialDelayMs}; clamping max delay to initial delay`);
    }
    maxDelayMs = initialDelayMs;
  }
  return { maxRetries, initialDelayMs, backoffMultiplier, maxDelayMs };
}

module.exports = {
  resolveRetryConfig,
};
