// @ts-check
/// <reference types="@actions/github-script" />

const fs = require("fs");
const path = require("path");
const { getErrorMessage } = require("./error_helpers.cjs");
const { ERR_CONFIG, ERR_PARSE, ERR_SYSTEM } = require("./error_codes.cjs");

/**
 * @typedef {Object} FileRenderItem
 * @property {string} path
 * @property {string} content_env
 */

/**
 * @typedef {Object} FileRenderConfig
 * @property {string[]} [directories]
 * @property {FileRenderItem[]} files
 */

/**
 * @param {string} value
 * @returns {FileRenderConfig}
 */
function parseConfig(value) {
  let parsed;
  try {
    parsed = JSON.parse(value);
  } catch (error) {
    throw new Error(`${ERR_PARSE}: Invalid GH_AW_FILE_CONFIG: ${getErrorMessage(error)}`, { cause: error });
  }
  if (!parsed || !Array.isArray(parsed.files)) {
    throw new Error(`${ERR_CONFIG}: GH_AW_FILE_CONFIG must contain a files array`);
  }
  if (parsed.directories !== undefined && !Array.isArray(parsed.directories)) {
    throw new Error(`${ERR_CONFIG}: GH_AW_FILE_CONFIG directories must be an array`);
  }
  return parsed;
}

/**
 * @param {string} root
 * @param {string} candidate
 */
function assertPathWithin(root, candidate) {
  const relative = path.relative(root, candidate);
  if (relative === ".." || relative.startsWith(`..${path.sep}`) || path.isAbsolute(relative)) {
    throw new Error(`${ERR_CONFIG}: Generated file path must stay within its configured root`);
  }
}

/**
 * @param {string} root
 * @param {string} relativePath
 * @returns {string}
 */
function resolveRelativePath(root, relativePath) {
  if (!relativePath || path.isAbsolute(relativePath)) {
    throw new Error(`${ERR_CONFIG}: Generated file path must be relative`);
  }
  const resolved = path.resolve(root, relativePath);
  assertPathWithin(root, resolved);
  return resolved;
}

/**
 * @param {string} filePath
 * @param {string} content
 */
function writeFile(filePath, content) {
  if (fs.constants.O_NOFOLLOW === undefined) {
    throw new Error(`${ERR_SYSTEM}: O_NOFOLLOW is not available on this platform; cannot write generated files safely`);
  }
  const flags = fs.constants.O_WRONLY | fs.constants.O_CREAT | fs.constants.O_TRUNC | fs.constants.O_NOFOLLOW;
  const fd = fs.openSync(filePath, flags, 0o600);
  try {
    fs.fchmodSync(fd, 0o600);
    fs.writeFileSync(fd, content, "utf8");
  } catch (error) {
    throw new Error(`${ERR_SYSTEM}: Failed to write generated file ${filePath}: ${getErrorMessage(error)}`, { cause: error });
  } finally {
    fs.closeSync(fd);
  }
}

/**
 * @param {string} directoryPath
 */
function makeDirectory(directoryPath) {
  try {
    fs.mkdirSync(directoryPath, { recursive: true, mode: 0o700 });
  } catch (error) {
    throw new Error(`${ERR_SYSTEM}: Failed to create generated directory ${directoryPath}: ${getErrorMessage(error)}`, { cause: error });
  }
}

/**
 * @param {FileRenderConfig} config
 * @param {NodeJS.ProcessEnv} env
 * @param {string} root
 */
function renderFiles(config, env, root) {
  makeDirectory(root);
  const resolvedRoot = fs.realpathSync(root);

  for (const directory of config.directories || []) {
    if (typeof directory !== "string") {
      throw new Error(`${ERR_CONFIG}: Generated directory path must be a string`);
    }
    const directoryPath = resolveRelativePath(resolvedRoot, directory);
    makeDirectory(directoryPath);
    assertPathWithin(resolvedRoot, fs.realpathSync(directoryPath));
  }

  for (const item of config.files) {
    if (!item || typeof item !== "object" || typeof item.path !== "string" || typeof item.content_env !== "string") {
      throw new Error(`${ERR_CONFIG}: Generated file item must specify string path and content_env values`);
    }
    if (!Object.prototype.hasOwnProperty.call(env, item.content_env)) {
      throw new Error(`${ERR_CONFIG}: Generated file content environment variable is missing: ${item.content_env}`);
    }

    const filePath = resolveRelativePath(resolvedRoot, item.path);
    const parentPath = path.dirname(filePath);
    const expectedParent = path.resolve(resolvedRoot, path.dirname(item.path));
    assertPathWithin(resolvedRoot, expectedParent);
    makeDirectory(parentPath);
    const resolvedParent = fs.realpathSync(parentPath);
    assertPathWithin(resolvedRoot, resolvedParent);
    writeFile(path.join(resolvedParent, path.basename(filePath)), String(env[item.content_env] ?? ""));
  }
}

/**
 * @param {typeof import('@actions/core')} [core] - GitHub Actions core library
 * @returns {Promise<void>}
 */
async function main(core = global.core) {
  try {
    const root = process.env.GH_AW_FILE_ROOT;
    const configValue = process.env.GH_AW_FILE_CONFIG;
    const runnerTemp = process.env.RUNNER_TEMP;
    if (!root) {
      throw new Error(`${ERR_CONFIG}: GH_AW_FILE_ROOT environment variable is not set`);
    }
    if (!configValue) {
      throw new Error(`${ERR_CONFIG}: GH_AW_FILE_CONFIG environment variable is not set`);
    }
    if (!runnerTemp) {
      throw new Error(`${ERR_CONFIG}: RUNNER_TEMP environment variable is not set`);
    }

    const unresolvedRunnerTemp = path.resolve(runnerTemp);
    const unresolvedRoot = path.resolve(root);
    assertPathWithin(unresolvedRunnerTemp, unresolvedRoot);
    makeDirectory(unresolvedRunnerTemp);
    const resolvedRunnerTemp = fs.realpathSync(unresolvedRunnerTemp);
    makeDirectory(unresolvedRoot);
    const resolvedRoot = fs.realpathSync(unresolvedRoot);
    assertPathWithin(resolvedRunnerTemp, resolvedRoot);

    const config = parseConfig(configValue);
    renderFiles(config, process.env, resolvedRoot);
    core.info(`Created ${config.files.length} generated file(s) under ${resolvedRoot}`);
  } catch (error) {
    core.setFailed(`${ERR_SYSTEM}: ${getErrorMessage(error)}`);
  }
}

module.exports = { assertPathWithin, main, makeDirectory, parseConfig, renderFiles, resolveRelativePath, writeFile };
