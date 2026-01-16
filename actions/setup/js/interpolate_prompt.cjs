// @ts-check
/// <reference types="@actions/github-script" />

// interpolate_prompt.cjs
// Interpolates GitHub Actions expressions and renders template conditionals in the prompt file.
// This combines variable interpolation and template filtering into a single step.

const fs = require("fs");
const { isTruthy } = require("./is_truthy.cjs");
const { processRuntimeImports } = require("./runtime_import.cjs");
const { getErrorMessage } = require("./error_helpers.cjs");

/**
 * Interpolates variables in the prompt content
 * @param {string} content - The prompt content with ${GH_AW_EXPR_*} placeholders
 * @param {Record<string, string>} variables - Map of variable names to their values
 * @returns {string} - The interpolated content
 */
function interpolateVariables(content, variables) {
  let result = content;

  // Replace each ${VAR_NAME} with its corresponding value
  for (const [varName, value] of Object.entries(variables)) {
    const pattern = new RegExp(`\\$\\{${varName}\\}`, "g");
    result = result.replace(pattern, value);
  }

  return result;
}

/**
 * Renders a Markdown template by processing {{#if}} conditional blocks.
 * When a conditional block is removed (falsy condition) and the template tags
 * were on their own lines, the empty lines are cleaned up to avoid
 * leaving excessive blank lines in the output.
 * @param {string} markdown - The markdown content to process
 * @returns {string} - The processed markdown content
 */
function renderMarkdownTemplate(markdown) {
  // First pass: Handle blocks where tags are on their own lines
  // Captures: (leading newline)(opening tag line)(condition)(body)(closing tag line)(trailing newline)
  let result = markdown.replace(/(\n?)([ \t]*{{#if\s+([^}]*)}}[ \t]*\n)([\s\S]*?)([ \t]*{{\/if}}[ \t]*)(\n?)/g, (match, leadNL, openLine, cond, body, closeLine, trailNL) => {
    if (isTruthy(cond)) {
      // Keep body with leading newline if there was one before the opening tag
      return leadNL + body;
    } else {
      // Remove entire block completely - the line containing the template is removed
      return "";
    }
  });

  // Second pass: Handle inline conditionals (tags not on their own lines)
  result = result.replace(/{{#if\s+([^}]*)}}([\s\S]*?){{\/if}}/g, (_, cond, body) => (isTruthy(cond) ? body : ""));

  // Clean up excessive blank lines (more than one blank line = 2 newlines)
  result = result.replace(/\n{3,}/g, "\n\n");

  return result;
}

/**
 * Main function for prompt variable interpolation and template rendering
 */
async function main() {
  try {
    const promptPath = process.env.GH_AW_PROMPT;
    if (!promptPath) {
      core.setFailed("GH_AW_PROMPT environment variable is not set");
      return;
    }

    // Get the workspace directory for runtime imports
    const workspaceDir = process.env.GITHUB_WORKSPACE;
    if (!workspaceDir) {
      core.setFailed("GITHUB_WORKSPACE environment variable is not set");
      return;
    }

    // Read the prompt file
    let content = fs.readFileSync(promptPath, "utf8");

    // Step 1: Process runtime imports (files and URLs)
    const hasRuntimeImports = /{{#runtime-import\??[ \t]+[^\}]+}}/.test(content);
    if (hasRuntimeImports) {
      core.info("Processing runtime import macros (files and URLs)");
      content = await processRuntimeImports(content, workspaceDir);
      core.info("Runtime imports processed successfully");
    } else {
      core.info("No runtime import macros found, skipping runtime import processing");
    }

    // Step 2: Interpolate variables
    /** @type {Record<string, string>} */
    const variables = {};
    for (const [key, value] of Object.entries(process.env)) {
      if (key.startsWith("GH_AW_EXPR_")) {
        variables[key] = value || "";
      }
    }

    const varCount = Object.keys(variables).length;
    if (varCount > 0) {
      core.info(`Found ${varCount} expression variable(s) to interpolate`);
      content = interpolateVariables(content, variables);
      core.info(`Successfully interpolated ${varCount} variable(s) in prompt`);
    } else {
      core.info("No expression variables found, skipping interpolation");
    }

    // Step 3: Render template conditionals
    const hasConditionals = /{{#if\s+[^}]+}}/.test(content);
    if (hasConditionals) {
      core.info("Processing conditional template blocks");
      content = renderMarkdownTemplate(content);
      core.info("Template rendered successfully");
    } else {
      core.info("No conditional blocks found in prompt, skipping template rendering");
    }

    // Write back to the same file
    fs.writeFileSync(promptPath, content, "utf8");
  } catch (error) {
    core.setFailed(getErrorMessage(error));
  }
}

module.exports = { main };
