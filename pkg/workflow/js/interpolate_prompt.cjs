// @ts-check
/// <reference types="@actions/github-script" />

// interpolate_prompt.cjs
// Interpolates GitHub Actions expressions and renders template conditionals in the prompt file.
// This combines variable interpolation and template filtering into a single step.

const fs = require("fs");
const { isTruthy } = require("./is_truthy.cjs");

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
 * Renders a Markdown template by processing {{#if}} conditional blocks
 * @param {string} markdown - The markdown content to process
 * @returns {string} - The processed markdown content
 */
function renderMarkdownTemplate(markdown) {
  return markdown.replace(/{{#if\s+([^}]+)}}([\s\S]*?){{\/if}}/g, (_, cond, body) => (isTruthy(cond) ? body : ""));
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

    // Read the prompt file
    let content = fs.readFileSync(promptPath, "utf8");

    // Step 1: Interpolate variables
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

    // Step 2: Render template conditionals
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
    core.setFailed(error instanceof Error ? error.message : String(error));
  }
}

// Execute main function
main();
