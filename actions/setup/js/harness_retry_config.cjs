// @ts-check
"use strict";

// Maximum number of retry attempts after the initial run
const DEFAULT_MAX_RETRIES = 3;
// Hard upper bound to prevent accidental multi-day retry loops
const MAX_RETRIES_CAP = 100;
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
  if (typeof logger === "function") {
    logger(`warning: ignoring invalid ${envVar}=${JSON.stringify(rawValue)}; using default ${defaultValue}`);
  }
}

const DECIMAL_INT_PATTERN = /^\d+$/;
const DECIMAL_FLOAT_PATTERN = /^\d+(?:\.\d+)?$/;

/**
 * Parse a retry config number from an environment variable.
 * Only accepts decimal-digit strings (e.g. "42" or "1.5" when allowFloat is true).
 * Non-decimal formats such as "1e3" or "0x10" are rejected to avoid surprising timer values.
 *
 * @param {NodeJS.ProcessEnv} env
 * @param {{envVar: string, defaultValue: number, minimum: number, allowFloat?: boolean, logger?: (message: string) => void}} options
 * @returns {number}
 */
function parseRetryConfigNumber(env, { envVar, defaultValue, minimum, allowFloat = false, logger }) {
  const rawValue = env[envVar];
  if (rawValue == null || rawValue === "") {
    return defaultValue;
  }
  const trimmed = String(rawValue).trim();
  const isValid = allowFloat ? DECIMAL_FLOAT_PATTERN.test(trimmed) : DECIMAL_INT_PATTERN.test(trimmed);
  if (!isValid) {
    logInvalidEnvValue(logger, envVar, rawValue, defaultValue);
    return defaultValue;
  }
  const parsed = allowFloat ? parseFloat(trimmed) : parseInt(trimmed, 10);
  if (Number.isFinite(parsed) && parsed >= minimum && (allowFloat || Number.isSafeInteger(parsed))) {
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
  let maxRetries = parseRetryConfigNumber(env, {
    envVar: HARNESS_MAX_RETRIES_ENV,
    defaultValue: DEFAULT_MAX_RETRIES,
    minimum: 0,
    logger,
  });
  if (maxRetries > MAX_RETRIES_CAP) {
    if (typeof logger === "function") {
      logger(`warning: ${HARNESS_MAX_RETRIES_ENV}=${maxRetries} exceeds maximum ${MAX_RETRIES_CAP}; clamping`);
    }
    maxRetries = MAX_RETRIES_CAP;
  }
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
  parseRetryConfigNumber,
};
