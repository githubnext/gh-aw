// @ts-check
/// <reference types="@actions/github-script" />

const fs = require("fs");
const path = require("path");

/**
 * Resolve the absolute path to a prompt template file.
 * Prefers GH_AW_PROMPTS_DIR when set, otherwise falls back to
 * ${RUNNER_TEMP}/gh-aw/prompts (the runtime location used in production).
 * When neither is set, falls back to the source md/ directory relative to
 * this file (used in local development and unit tests).
 * @param {string} name - Template filename (e.g. "agent_timeout.md")
 * @returns {string} Absolute path to the prompt template file
 */
function getPromptPath(name) {
  const promptsDir = process.env.GH_AW_PROMPTS_DIR || (process.env.RUNNER_TEMP ? `${process.env.RUNNER_TEMP}/gh-aw/prompts` : null) || path.join(__dirname, "../md");
  return `${promptsDir}/${name}`;
}

/**
 * Replace placeholders in a template string with values from context.
 * Supports {key} syntax for placeholder replacement.
 * @param {string} template - Template string with {key} placeholders
 * @param {Record<string, string|number|boolean|undefined>} context - Key-value pairs for replacement
 * @returns {string} Template with placeholders replaced
 */
function renderTemplate(template, context) {
  return template.replace(/\{(\w+)\}/g, (match, key) => {
    const value = context[key];
    if (value === undefined || value === null) {
      return match;
    }
    return String(value);
  });
}

/**
 * Read a template file and render it with the given context.
 * Combines file loading and template rendering into a single helper.
 * @param {string} templatePath - Absolute path to the template file
 * @param {Record<string, string|number|boolean|undefined>} context - Key-value pairs for replacement
 * @returns {string} Rendered template with placeholders replaced
 */
function renderTemplateFromFile(templatePath, context) {
  let template;
  try {
    template = fs.readFileSync(templatePath, "utf8");
  } catch (err) {
    throw new Error(`Failed to read file ${templatePath}: ${String(err)}`, { cause: err });
  }
  return renderTemplate(template, context);
}

module.exports = {
  getPromptPath,
  renderTemplate,
  renderTemplateFromFile,
};
