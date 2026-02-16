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
  const result = !(v === "" || v === "false" || v === "0" || v === "null" || v === "undefined");
  if (typeof core !== "undefined") {
    core.info(`[isTruthy] Evaluating "${expr}" (trimmed: "${v}") -> ${result}`);
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
  if (typeof core !== "undefined") {
    core.info(`[renderMarkdownTemplate] Starting template rendering`);
    core.info(`[renderMarkdownTemplate] Input length: ${markdown.length} characters`);
  }

  // Count conditionals before processing
  const blockConditionals = (markdown.match(/(\n?)([ \t]*{{#if\s+(.*?)\s*}}[ \t]*\n)([\s\S]*?)([ \t]*{{\/if}}[ \t]*)(\n?)/g) || []).length;
  const inlineConditionals = (markdown.match(/{{#if\s+(.*?)\s*}}([\s\S]*?){{\/if}}/g) || []).length - blockConditionals;

  if (typeof core !== "undefined") {
    core.info(`[renderMarkdownTemplate] Found ${blockConditionals} block conditional(s) and ${inlineConditionals} inline conditional(s)`);
  }

  let blockCount = 0;
  let keptBlocks = 0;
  let removedBlocks = 0;

  // First pass: Handle blocks where tags are on their own lines
  // Captures: (leading newline)(opening tag line)(condition)(body)(closing tag line)(trailing newline)
  // Uses .*? (non-greedy) with \s* to handle expressions with or without trailing spaces
  let result = markdown.replace(/(\n?)([ \t]*{{#if\s+(.*?)\s*}}[ \t]*\n)([\s\S]*?)([ \t]*{{\/if}}[ \t]*)(\n?)/g, (match, leadNL, openLine, cond, body, closeLine, trailNL) => {
    blockCount++;
    const condTrimmed = cond.trim();
    const truthyResult = isTruthy(cond);
    const bodyPreview = body.substring(0, 60).replace(/\n/g, "\\n");

    if (typeof core !== "undefined") {
      core.info(`[renderMarkdownTemplate] Block ${blockCount}: condition="${condTrimmed}" -> ${truthyResult ? "KEEP" : "REMOVE"}`);
      core.info(`[renderMarkdownTemplate]   Body preview: "${bodyPreview}${body.length > 60 ? "..." : ""}"`);
    }

    if (truthyResult) {
      // Keep body with leading newline if there was one before the opening tag
      keptBlocks++;
      if (typeof core !== "undefined") {
        core.info(`[renderMarkdownTemplate]   Action: Keeping body with leading newline=${!!leadNL}`);
      }
      return leadNL + body;
    } else {
      // Remove entire block completely - the line containing the template is removed
      removedBlocks++;
      if (typeof core !== "undefined") {
        core.info(`[renderMarkdownTemplate]   Action: Removing entire block`);
      }
      return "";
    }
  });

  if (typeof core !== "undefined") {
    core.info(`[renderMarkdownTemplate] First pass complete: ${keptBlocks} kept, ${removedBlocks} removed`);
  }

  let inlineCount = 0;
  let keptInline = 0;
  let removedInline = 0;

  // Second pass: Handle inline conditionals (tags not on their own lines)
  // Uses .*? (non-greedy) with \s* to handle expressions with or without trailing spaces
  result = result.replace(/{{#if\s+(.*?)\s*}}([\s\S]*?){{\/if}}/g, (_, cond, body) => {
    inlineCount++;
    const condTrimmed = cond.trim();
    const truthyResult = isTruthy(cond);
    const bodyPreview = body.substring(0, 40).replace(/\n/g, "\\n");

    if (typeof core !== "undefined") {
      core.info(`[renderMarkdownTemplate] Inline ${inlineCount}: condition="${condTrimmed}" -> ${truthyResult ? "KEEP" : "REMOVE"}`);
      core.info(`[renderMarkdownTemplate]   Body preview: "${bodyPreview}${body.length > 40 ? "..." : ""}"`);
    }

    if (truthyResult) {
      keptInline++;
      return body;
    } else {
      removedInline++;
      return "";
    }
  });

  if (typeof core !== "undefined") {
    core.info(`[renderMarkdownTemplate] Second pass complete: ${keptInline} kept, ${removedInline} removed`);
  }

  // Clean up excessive blank lines (more than one blank line = 2 newlines)
  const beforeCleanup = result.length;
  const excessiveLines = (result.match(/\n{3,}/g) || []).length;
  result = result.replace(/\n{3,}/g, "\n\n");

  if (typeof core !== "undefined") {
    if (excessiveLines > 0) {
      core.info(`[renderMarkdownTemplate] Cleaned up ${excessiveLines} excessive blank line sequence(s)`);
      core.info(`[renderMarkdownTemplate] Length change from cleanup: ${beforeCleanup} -> ${result.length} characters`);
    }
    core.info(`[renderMarkdownTemplate] Final output length: ${result.length} characters`);
  }

  return result;
}

/**
 * Main function for template rendering in GitHub Actions
 */
function main() {
  try {
    if (typeof core !== "undefined") {
      core.info("========================================");
      core.info("[main] Starting render_template processing");
      core.info("========================================");
    }

    const promptPath = process.env.GH_AW_PROMPT;
    if (!promptPath) {
      if (typeof core !== "undefined") {
        core.setFailed("GH_AW_PROMPT environment variable is not set");
      }
      process.exit(1);
    }

    if (typeof core !== "undefined") {
      core.info(`[main] Prompt path: ${promptPath}`);
    }

    // Read the prompt file
    if (typeof core !== "undefined") {
      core.info(`[main] Reading prompt file...`);
    }
    const markdown = fs.readFileSync(promptPath, "utf8");
    const originalLength = markdown.length;

    if (typeof core !== "undefined") {
      core.info(`[main] Original content length: ${originalLength} characters`);
      core.info(`[main] First 200 characters: ${markdown.substring(0, 200).replace(/\n/g, "\\n")}`);
    }

    // Check if there are any conditional blocks
    const hasConditionals = /{{#if\s+[^}]+}}/.test(markdown);
    if (!hasConditionals) {
      if (typeof core !== "undefined") {
        core.info("No conditional blocks found in prompt, skipping template rendering");
        core.info("========================================");
        core.info("[main] Processing complete - SKIPPED");
        core.info("========================================");
      }
      process.exit(0);
    }

    const conditionalMatches = markdown.match(/{{#if\s+[^}]+}}/g) || [];
    if (typeof core !== "undefined") {
      core.info(`[main] Processing ${conditionalMatches.length} conditional template block(s)`);
    }

    // Render the template
    const beforeRendering = markdown.length;
    const rendered = renderMarkdownTemplate(markdown);
    const afterRendering = rendered.length;

    // Write back to the same file
    if (typeof core !== "undefined") {
      core.info("\n========================================");
      core.info("[main] Writing Output");
      core.info("========================================");
      core.info(`[main] Writing processed content back to: ${promptPath}`);
      core.info(`[main] Final content length: ${afterRendering} characters`);
      core.info(`[main] Total length change: ${beforeRendering} -> ${afterRendering} (${afterRendering > beforeRendering ? "+" : ""}${afterRendering - beforeRendering})`);
    }

    fs.writeFileSync(promptPath, rendered, "utf8");

    if (typeof core !== "undefined") {
      core.info(`[main] Last 200 characters: ${rendered.substring(Math.max(0, rendered.length - 200)).replace(/\n/g, "\\n")}`);
      core.info("========================================");
      core.info("[main] Processing complete - SUCCESS");
      core.info("========================================");
    }
  } catch (error) {
    if (typeof core !== "undefined") {
      core.info("========================================");
      core.info("[main] Processing failed - ERROR");
      core.info("========================================");
      const err = error instanceof Error ? error : new Error(String(error));
      core.info(`[main] Error type: ${err.constructor.name}`);
      core.info(`[main] Error message: ${err.message}`);
      if (err.stack) {
        core.info(`[main] Stack trace:\n${err.stack}`);
      }
      core.setFailed(getErrorMessage(error));
    } else {
      throw error;
    }
  }
}

module.exports = { renderMarkdownTemplate, main };
