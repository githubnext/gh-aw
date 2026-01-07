// @ts-check
/// <reference types="@actions/github-script" />

// render_template.cjs
// Single-function Markdown → Markdown postprocessor for GitHub Actions.
// Processes only {{#if <expr>}} ... {{/if}} blocks after ${{ }} evaluation.

const { getErrorMessage } = require("./error_helpers.cjs");

const fs = require("fs");

/**
 * Determines if a value is truthy according to template logic
 * @param {string} expr - The expression to evaluate
 * @returns {boolean} - Whether the expression is truthy
 */
function isTruthy(expr) {
  const v = expr.trim().toLowerCase();
  return !(v === "" || v === "false" || v === "0" || v === "null" || v === "undefined");
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
  // Uses .*? (non-greedy) with \s* to handle expressions with or without trailing spaces
  let result = markdown.replace(/(\n?)([ \t]*{{#if\s+(.*?)\s*}}[ \t]*\n)([\s\S]*?)([ \t]*{{\/if}}[ \t]*)(\n?)/g, (match, leadNL, openLine, cond, body, closeLine, trailNL) => {
    if (isTruthy(cond)) {
      // Keep body with leading newline if there was one before the opening tag
      return leadNL + body;
    } else {
      // Remove entire block completely - the line containing the template is removed
      return "";
    }
  });

  // Second pass: Handle inline conditionals (tags not on their own lines)
  // Uses .*? (non-greedy) with \s* to handle expressions with or without trailing spaces
  result = result.replace(/{{#if\s+(.*?)\s*}}([\s\S]*?){{\/if}}/g, (_, cond, body) => (isTruthy(cond) ? body : ""));

  // Clean up excessive blank lines (more than one blank line = 2 newlines)
  result = result.replace(/\n{3,}/g, "\n\n");

  return result;
}

/**
 * Main function for template rendering in GitHub Actions
 */
function main() {
  try {
    const promptPath = process.env.GH_AW_PROMPT;
    if (!promptPath) {
      core.setFailed("GH_AW_PROMPT environment variable is not set");
      process.exit(1);
    }

    // Read the prompt file
    const markdown = fs.readFileSync(promptPath, "utf8");

    // Check if there are any conditional blocks
    const hasConditionals = /{{#if\s+[^}]+}}/.test(markdown);
    if (!hasConditionals) {
      core.info("No conditional blocks found in prompt, skipping template rendering");
      process.exit(0);
    }

    // Render the template
    const rendered = renderMarkdownTemplate(markdown);

    // Write back to the same file
    fs.writeFileSync(promptPath, rendered, "utf8");

    core.info("Template rendered successfully");
    // core.summary.addHeading("Template Rendering", 3).addRaw("\n").addRaw("Processed conditional blocks in prompt\n").write();
  } catch (error) {
    core.setFailed(getErrorMessage(error));
  }
}

module.exports = { renderMarkdownTemplate, main };
