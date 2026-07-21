"use strict";

const MODEL_FALLBACK_ENV_VAR = "GH_AW_MODEL_FALLBACK";

function readTrimmedEnv(env, name) {
  return typeof env?.[name] === "string" ? env[name].trim() : "";
}

function resolveModelWithFallback(env, primaryEnvVar) {
  return readTrimmedEnv(env, primaryEnvVar) || readTrimmedEnv(env, MODEL_FALLBACK_ENV_VAR);
}

/**
 * @param {NodeJS.ProcessEnv} env
 * @param {string} primaryEnvVar
 * @param {(message: string) => void} [logger]
 * @returns {string}
 */
function applyModelFallback(env, primaryEnvVar, logger = () => {}) {
  const primary = readTrimmedEnv(env, primaryEnvVar);
  if (primary) {
    return primary;
  }
  const fallback = readTrimmedEnv(env, MODEL_FALLBACK_ENV_VAR);
  if (fallback) {
    env[primaryEnvVar] = fallback;
    logger(`applied ${MODEL_FALLBACK_ENV_VAR} to ${primaryEnvVar}`);
  }
  return fallback;
}

function injectModelFlagAfterExec(args, model) {
  if (!model || args.includes("--model")) {
    return args;
  }
  const execIndex = args.indexOf("exec");
  if (execIndex === -1) {
    return [...args, "--model", model];
  }
  return [...args.slice(0, execIndex + 1), "--model", model, ...args.slice(execIndex + 1)];
}

module.exports = {
  MODEL_FALLBACK_ENV_VAR,
  resolveModelWithFallback,
  applyModelFallback,
  injectModelFlagAfterExec,
};
