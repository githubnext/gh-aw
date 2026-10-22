// @ts-check
/// <reference types="@actions/github-script" />

// Ensures global.core is available when running outside github-script context
require("./shim.cjs");
const fs = require("fs");
const path = require("path");
const { getErrorMessage } = require("./error_helpers.cjs");
const { ERR_SYSTEM, ERR_VALIDATION } = require("./error_codes.cjs");

/**
 * Substitutes `__KEY__` placeholders in a file with values from the substitutions map.
 * Undefined/null values are treated as empty strings.
 *
 * @param {{ file: string, substitutions: Record<string, string | null | undefined>, inlineArtifacts?: Array<{directory: string, extension: string}> }} params
 * @returns {Promise<string>}
 */
const substitutePlaceholders = async ({ file, substitutions, inlineArtifacts = [] }) => {
  // Validate parameters
  if (!file) {
    throw new Error(`${ERR_VALIDATION}: ` + "file parameter is required");
  }
  if (!substitutions || typeof substitutions !== "object") {
    throw new Error(`${ERR_VALIDATION}: ` + "substitutions parameter must be an object");
  }

  core.info(`[substitutePlaceholders] ${file} (${Object.keys(substitutions).length} substitution(s))`);
  const replacements = new Map(Object.entries(substitutions).map(([key, value]) => [`__${key}__`, value == null ? "" : value]));
  const placeholderPattern = replacements.size ? new RegExp([...replacements.keys()].map(placeholder => placeholder.replace(/[.*+?^${}()|[\]\\]/g, "\\$&")).join("|"), "g") : null;

  const splitPaths = ["system.txt", "user.txt"].map(name => path.join(path.dirname(file), name));
  const hasSplit = !splitPaths.some(splitPath => path.resolve(splitPath) === path.resolve(file)) && splitPaths.every(splitPath => fs.existsSync(splitPath));
  const files = hasSplit ? [file, ...splitPaths] : [file];
  // Only compiler-selected staging directories and extractor-shaped Markdown files are
  // eligible. Never recurse through arbitrary artifact contents or follow symlinks.
  const artifactRoot = path.dirname(path.dirname(path.resolve(file)));
  for (const { directory, extension } of inlineArtifacts) {
    if (!/^[.a-zA-Z0-9_-]+(?:\/[.a-zA-Z0-9_-]+)*$/.test(directory) || directory.split("/").some(part => part === "." || part === "..") || ![".md", ".agent.md", "/SKILL.md"].includes(extension)) {
      throw new Error(`${ERR_VALIDATION}: Invalid inline artifact substitution target`);
    }
    let targetDir = artifactRoot;
    let exists = true;
    for (const part of directory.split("/")) {
      targetDir = path.join(targetDir, part);
      if (!fs.existsSync(targetDir)) {
        exists = false;
        break;
      }
      const stat = fs.lstatSync(targetDir);
      if (stat.isSymbolicLink() || !stat.isDirectory()) throw new Error(`${ERR_VALIDATION}: Inline artifact directory must not be a symlink: ${targetDir}`);
    }
    if (!exists) continue;
    let entries;
    try {
      entries = fs.readdirSync(targetDir, { withFileTypes: true });
    } catch (error) {
      throw new Error(`${ERR_SYSTEM}: Failed to list inline artifacts in ${targetDir}: ${getErrorMessage(error)}`, { cause: error });
    }
    for (const entry of entries) {
      const name = extension === "/SKILL.md" ? entry.name : entry.name.slice(0, -extension.length);
      if (!/^[a-z][a-z0-9_-]*$/.test(name)) continue;
      if (extension === "/SKILL.md" ? !entry.isDirectory() : !entry.isFile() || !entry.name.endsWith(extension)) continue;
      const target = path.join(targetDir, name + extension);
      if (!fs.existsSync(target)) continue;
      const stat = fs.lstatSync(target);
      if (stat.isSymbolicLink() || !stat.isFile()) throw new Error(`${ERR_VALIDATION}: Inline artifact must be a regular file: ${target}`);
      files.push(target);
    }
  }

  // Read the files
  const contents = [];
  for (const target of files) {
    let content;
    try {
      content = fs.readFileSync(target, "utf8");
    } catch (error) {
      const errorMessage = getErrorMessage(error);
      throw new Error(`${ERR_SYSTEM}: Failed to read file ${target}: ${errorMessage}`);
    }

    // A single pass preserves literal placeholder-like text in resolved output values.
    if (placeholderPattern) content = content.replace(placeholderPattern, placeholder => replacements.get(placeholder) ?? "");

    contents.push(content);
  }
  if (hasSplit && contents[0] !== contents[1] + contents[2]) {
    throw new Error(`${ERR_VALIDATION}: Split prompt files do not match the combined prompt after substitution`);
  }

  // Write back to the files
  for (const [index, target] of files.entries()) {
    try {
      fs.writeFileSync(target, contents[index], "utf8");
    } catch (error) {
      const errorMessage = getErrorMessage(error);
      throw new Error(`${ERR_SYSTEM}: Failed to write file ${target}: ${errorMessage}`);
    }
  }

  return `Successfully substituted ${Object.keys(substitutions).length} placeholder(s) in ${file}`;
};

module.exports = substitutePlaceholders;
