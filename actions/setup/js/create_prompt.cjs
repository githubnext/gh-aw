// @ts-check
/// <reference types="@actions/github-script" />

const fs = require("fs");
const path = require("path");
const { getErrorMessage } = require("./error_helpers.cjs");
const { ERR_CONFIG } = require("./error_codes.cjs");

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
  const parsed = JSON.parse(value);
  if (!parsed || !Array.isArray(parsed.items)) {
    throw new Error("GH_AW_PROMPT_CONFIG must contain an items array");
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
    throw new Error("Prompt file must be a relative path");
  }
  const root = path.resolve(promptsDir);
  const resolved = path.resolve(root, filename);
  if (resolved !== root && !resolved.startsWith(`${root}${path.sep}`)) {
    throw new Error("Prompt file must stay within the prompt directory");
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
      throw new Error("Prompt render item must be an object");
    }
    if (item.condition_env && env[item.condition_env] !== "true") {
      continue;
    }

    const hasContent = typeof item.content_env === "string";
    const hasFile = typeof item.file === "string";
    if (hasContent === hasFile) {
      throw new Error("Prompt render item must specify exactly one content_env or file");
    }

    if (hasContent) {
      if (!Object.prototype.hasOwnProperty.call(env, item.content_env)) {
        throw new Error(`Prompt content environment variable is missing: ${item.content_env}`);
      }
      result += env[item.content_env] || "";
      continue;
    }

    const promptFile = resolvePromptFile(promptsDir, item.file);
    result += fs.readFileSync(promptFile, "utf8");
  }

  return result;
}

async function main() {
  try {
    const promptPath = process.env.GH_AW_PROMPT;
    const configValue = process.env.GH_AW_PROMPT_CONFIG;
    const runnerTemp = process.env.RUNNER_TEMP;
    if (!promptPath) {
      throw new Error("GH_AW_PROMPT environment variable is not set");
    }
    if (!configValue) {
      throw new Error("GH_AW_PROMPT_CONFIG environment variable is not set");
    }
    if (!runnerTemp) {
      throw new Error("RUNNER_TEMP environment variable is not set");
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
