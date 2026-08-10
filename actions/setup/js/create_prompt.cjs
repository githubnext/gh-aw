// @ts-check
/// <reference types="@actions/github-script" />

const fs = require("fs");
const path = require("path");
const { getErrorMessage } = require("./error_helpers.cjs");
const { ERR_CONFIG, ERR_PARSE, ERR_SYSTEM } = require("./error_codes.cjs");

/**
 * @typedef {Object} PromptRenderItem
 * @property {string} [content_env]
 * @property {string} [file]
 * @property {string} [condition_env]
 */

/**
 * @typedef {Object} PromptRenderConfig
 * @property {PromptRenderItem[]} items
 */

/**
 * Parse and validate the compiler-generated prompt render configuration.
 * @param {string} value
 * @returns {PromptRenderConfig}
 */
function parseConfig(value) {
  let parsed;
  try {
    parsed = JSON.parse(value);
  } catch (error) {
    throw new Error(`${ERR_PARSE}: Invalid GH_AW_PROMPT_CONFIG: ${getErrorMessage(error)}`);
  }
  if (!parsed || !Array.isArray(parsed.items)) {
    throw new Error(`${ERR_CONFIG}: GH_AW_PROMPT_CONFIG must contain an items array`);
  }
  return parsed;
}

/**
 * Resolve a compiler-owned prompt file without allowing traversal outside the
 * setup action's prompt directory.
 * @param {string} promptsDir
 * @param {string} filename
 * @returns {string}
 */
function resolvePromptFile(promptsDir, filename) {
  if (!filename || path.isAbsolute(filename)) {
    throw new Error(`${ERR_CONFIG}: Prompt file must be a relative path`);
  }
  const root = path.resolve(promptsDir);
  const resolved = path.resolve(root, filename);
  if (resolved !== root && !resolved.startsWith(`${root}${path.sep}`)) {
    throw new Error(`${ERR_CONFIG}: Prompt file must stay within the prompt directory`);
  }
  return resolved;
}

/**
 * Render prompt items. All workflow-authored content and expression results are
 * read from environment variables and appended as data.
 * @param {PromptRenderConfig} config
 * @param {NodeJS.ProcessEnv} env
 * @param {string} promptsDir
 * @returns {string}
 */
function renderPrompt(config, env, promptsDir) {
  let result = "";

  for (const item of config.items) {
    if (!item || typeof item !== "object") {
      throw new Error(`${ERR_CONFIG}: Prompt render item must be an object`);
    }
    if (item.condition_env && env[item.condition_env] !== "true") {
      continue;
    }

    const contentEnv = item.content_env;
    const filename = item.file;
    const hasContent = typeof contentEnv === "string";
    const hasFile = typeof filename === "string";
    if (hasContent === hasFile) {
      throw new Error(`${ERR_CONFIG}: Prompt render item must specify exactly one content_env or file`);
    }

    if (typeof contentEnv === "string") {
      if (!Object.prototype.hasOwnProperty.call(env, contentEnv)) {
        throw new Error(`${ERR_CONFIG}: Prompt content environment variable is missing: ${contentEnv}`);
      }
      result += env[contentEnv] || "";
      continue;
    }

    if (typeof filename !== "string") {
      throw new Error(`${ERR_CONFIG}: Prompt render item file must be a string`);
    }
    const promptFile = resolvePromptFile(promptsDir, filename);
    try {
      result += fs.readFileSync(promptFile, "utf8");
    } catch (error) {
      throw new Error(`${ERR_SYSTEM}: Failed to read prompt file: ${getErrorMessage(error)}`);
    }
  }

  return result;
}

/**
 * @param {typeof import('@actions/core')} core - GitHub Actions core library
 * @returns {Promise<void>}
 */
async function main(core) {
  try {
    const promptPath = process.env.GH_AW_PROMPT;
    const configValue = process.env.GH_AW_PROMPT_CONFIG;
    const runnerTemp = process.env.RUNNER_TEMP;
    if (!promptPath) {
      throw new Error(`${ERR_CONFIG}: GH_AW_PROMPT environment variable is not set`);
    }
    if (!configValue) {
      throw new Error(`${ERR_CONFIG}: GH_AW_PROMPT_CONFIG environment variable is not set`);
    }
    if (!runnerTemp) {
      throw new Error(`${ERR_CONFIG}: RUNNER_TEMP environment variable is not set`);
    }

    const config = parseConfig(configValue);
    const promptsDir = path.join(runnerTemp, "gh-aw", "prompts");
    const content = renderPrompt(config, process.env, promptsDir);

    fs.mkdirSync(path.dirname(promptPath), { recursive: true, mode: 0o700 });
    fs.writeFileSync(promptPath, content, { encoding: "utf8", mode: 0o600 });
    core.info(`Created prompt at ${promptPath} (${Buffer.byteLength(content, "utf8")} bytes)`);
  } catch (error) {
    core.setFailed(`${ERR_CONFIG}: ${getErrorMessage(error)}`);
  }
}

module.exports = { main, parseConfig, renderPrompt, resolvePromptFile };
