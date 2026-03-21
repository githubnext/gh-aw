// @ts-check
/// <reference types="@actions/github-script" />

// render_template.cjs
// Single-function Markdown → Markdown postprocessor for GitHub Actions.
// Processes only {{#if <expr>}} ... {{/if}} blocks after ${{ }} evaluation.

require("./shim.cjs");

const { getErrorMessage } = require("./error_helpers.cjs");
const fs = require("fs");
const { ERR_API, ERR_CONFIG } = require("./error_codes.cjs");
const { isTruthy } = require("./is_truthy.cjs");

/**
 * Renders a Markdown template by processing {{#if}} conditional blocks.
 * When a conditional block is removed (falsy condition) and the template tags
 * were on their own lines, the empty lines are cleaned up to avoid
 * leaving excessive blank lines in the output.
 * @param {string} markdown - The markdown content to process
 * @returns {string} - The processed markdown content
 */
function renderMarkdownTemplate(markdown) {
  core.info(`[renderMarkdownTemplate] Starting template rendering`);
  core.info(`[renderMarkdownTemplate] Input length: ${markdown.length} characters`);

  // Count conditionals before processing
  const blockConditionals = (markdown.match(/(\n?)([ \t]*{{#if\s+(.*?)\s*}}[ \t]*\n)([\s\S]*?)([ \t]*{{\/if}}[ \t]*)(\n?)/g) || []).length;
  const inlineConditionals = (markdown.match(/{{#if\s+(.*?)\s*}}([\s\S]*?){{\/if}}/g) || []).length - blockConditionals;

  core.info(`[renderMarkdownTemplate] Found ${blockConditionals} block conditional(s) and ${inlineConditionals} inline conditional(s)`);

  let blockCount = 0;
  let keptBlocks = 0;
  let removedBlocks = 0;

  // First pass: Handle blocks where tags are on their own lines
  // Captures: (leading newline)(opening tag line)(condition)(body)(closing tag line)(trailing newline)
  // Uses .*? (non-greedy) with \s* to handle expressions with or without trailing spaces
  let result = markdown.replace(/(\n?)([ \t]*{{#if\s+(.*?)\s*}}[ \t]*\n)([\s\S]*?)([ \t]*{{\/if}}[ \t]*)(\n?)/g, (match, leadNL, openLine, cond, body) => {
    blockCount++;
    const truthyResult = isTruthy(cond);

    core.info(`[renderMarkdownTemplate] Block ${blockCount}: condition="${cond.trim()}" -> ${truthyResult ? "KEEP" : "REMOVE"}`);

    if (truthyResult) {
      // Keep body with leading newline if there was one before the opening tag
      keptBlocks++;
      return leadNL + body;
    } else {
      // Remove entire block completely - the line containing the template is removed
      removedBlocks++;
      return "";
    }
  });

  core.info(`[renderMarkdownTemplate] First pass complete: ${keptBlocks} kept, ${removedBlocks} removed`);

  let inlineCount = 0;
  let keptInline = 0;
  let removedInline = 0;

  // Second pass: Handle inline conditionals (tags not on their own lines)
  // Uses .*? (non-greedy) with \s* to handle expressions with or without trailing spaces
  result = result.replace(/{{#if\s+(.*?)\s*}}([\s\S]*?){{\/if}}/g, (_, cond, body) => {
    inlineCount++;
    const truthyResult = isTruthy(cond);

    core.info(`[renderMarkdownTemplate] Inline ${inlineCount}: condition="${cond.trim()}" -> ${truthyResult ? "KEEP" : "REMOVE"}`);

    if (truthyResult) {
      keptInline++;
      return body;
    } else {
      removedInline++;
      return "";
    }
  });

  core.info(`[renderMarkdownTemplate] Second pass complete: ${keptInline} kept, ${removedInline} removed`);

  // Clean up excessive blank lines (more than one blank line = 2 newlines)
  result = result.replace(/\n{3,}/g, "\n\n");

  core.info(`[renderMarkdownTemplate] Final output length: ${result.length} characters`);

  return result;
}

/**
 * Main function for template rendering in GitHub Actions
 */
function main() {
  try {
    core.info("[render_template] Starting template rendering");

    const promptPath = process.env.GH_AW_PROMPT;
    if (!promptPath) {
      core.setFailed(`${ERR_CONFIG}: GH_AW_PROMPT environment variable is not set`);
      process.exit(1);
    }

    core.info(`[render_template] Prompt path: ${promptPath}`);

    const markdown = fs.readFileSync(promptPath, "utf8");
    core.info(`[render_template] Read ${markdown.length} characters`);

    const hasConditionals = /{{#if\s+[^}]+}}/.test(markdown);
    if (!hasConditionals) {
      core.info("No conditional blocks found in prompt, skipping template rendering");
      process.exit(0);
    }

    const conditionalMatches = markdown.match(/{{#if\s+[^}]+}}/g) || [];
    core.info(`[render_template] Processing ${conditionalMatches.length} conditional template block(s)`);

    const rendered = renderMarkdownTemplate(markdown);

    core.info(`[render_template] Writing back to ${promptPath} (${rendered.length} characters)`);
    fs.writeFileSync(promptPath, rendered, "utf8");

    core.info("[render_template] Processing complete");
  } catch (error) {
    const err = error instanceof Error ? error : new Error(String(error));
    if (err.stack) {
      core.info(`[render_template] Stack trace:\n${err.stack}`);
    }
    core.setFailed(`${ERR_API}: ${getErrorMessage(error)}`);
  }
}

module.exports = { renderMarkdownTemplate, main };
