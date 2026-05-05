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
 * Selects the appropriate branch from a conditional block that may contain
 * {{#elseif}}, {{#else-if}}, {{#else_if}}, {{elseif}}, {{else-if}}, {{else_if}}
 * branches in addition to the optional {{#else}} fallback.
 *
 * @param {string} ifCondition - The condition from the opening {{#if ...}} tag
 * @param {string} body        - Everything between the opening tag and {{/if}}
 * @returns {string|null}      - Content of the first truthy branch, or null
 */
function selectBranch(ifCondition, body) {
  // Split on all elseif variants.  The capturing group ensures that the condition
  // text appears between the content pieces in the resulting array.
  // Supported: {{#elseif}}, {{#else-if}}, {{#else_if}}, {{elseif}}, {{else-if}}, {{else_if}}
  const parts = body.split(/[ \t]*\{\{#?else[-_]?if\s+([^}]*)\}\}[ \t]*\n?/);

  // parts alternates: [content0, cond1, content1, cond2, content2, ...]
  const branches = [{ condition: ifCondition, content: parts[0] }];
  for (let i = 1; i < parts.length; i += 2) {
    branches.push({ condition: parts[i].trim(), content: parts[i + 1] || "" });
  }

  // Check whether the last branch's content contains a {{#else}} tail
  const lastBranch = branches[branches.length - 1];
  const elseParts = lastBranch.content.split(/[ \t]*\{\{#else\}\}[ \t]*\n?/);
  if (elseParts.length > 1) {
    lastBranch.content = elseParts[0];
    branches.push({ condition: null, content: elseParts.slice(1).join("{{#else}}") });
  }

  // Return content of the first truthy branch
  for (const branch of branches) {
    if (branch.condition === null || isTruthy(branch.condition)) {
      return branch.content;
    }
  }
  return null;
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
  core.info(`[renderMarkdownTemplate] Starting template rendering`);
  core.info(`[renderMarkdownTemplate] Input length: ${markdown.length} characters`);

  // Preserve fenced code blocks to avoid processing {{#if}} markers inside them
  const _codeBlocks = [];
  const _FENCE_PH = "\x00FENCE\x00";
  const _stripped = markdown.replace(/`{3,}[^\n]*\n[\s\S]*?\n`{3,}[ \t]*/g, m => {
    _codeBlocks.push(m);
    return `${_FENCE_PH}${_codeBlocks.length - 1}${_FENCE_PH}`;
  });
  if (_codeBlocks.length > 0) {
    core.info(`[renderMarkdownTemplate] Preserved ${_codeBlocks.length} fenced code block(s) from template processing`);
  }

  // Count conditionals before processing
  const blockConditionals = (_stripped.match(/(\n?)([ \t]*{{#if\s+(.*?)\s*}}[ \t]*\n)([\s\S]*?)([ \t]*{{\/if}}[ \t]*)(\n?)/g) || []).length;
  const inlineConditionals = (_stripped.match(/{{#if\s+(.*?)\s*}}([\s\S]*?){{\/if}}/g) || []).length - blockConditionals;

  core.info(`[renderMarkdownTemplate] Found ${blockConditionals} block conditional(s) and ${inlineConditionals} inline conditional(s)`);

  let blockCount = 0;
  let keptBlocks = 0;
  let removedBlocks = 0;

  // First pass: Handle blocks where tags are on their own lines
  // Captures: (leading newline)(opening tag line)(condition)(body)(closing tag line)(trailing newline)
  // Uses .*? (non-greedy) with \s* to handle expressions with or without trailing spaces
  let result = _stripped.replace(/(\n?)([ \t]*{{#if\s+(.*?)\s*}}[ \t]*\n)([\s\S]*?)([ \t]*{{\/if}}[ \t]*)(\n?)/g, (match, leadNL, openLine, cond, body) => {
    blockCount++;

    core.info(`[renderMarkdownTemplate] Block ${blockCount}: condition="${cond.trim()}" -> evaluating branches`);

    const selectedContent = selectBranch(cond, body);

    if (selectedContent !== null) {
      keptBlocks++;
      return leadNL + selectedContent;
    } else {
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
    const selectedContent = selectBranch(cond, body);

    core.info(`[renderMarkdownTemplate] Inline ${inlineCount}: condition="${cond.trim()}" -> ${selectedContent !== null ? "KEEP" : "REMOVE"}`);

    if (selectedContent !== null) {
      keptInline++;
      return selectedContent;
    } else {
      removedInline++;
      return "";
    }
  });

  core.info(`[renderMarkdownTemplate] Second pass complete: ${keptInline} kept, ${removedInline} removed`);

  // Clean up excessive blank lines (more than one blank line = 2 newlines)
  result = result.replace(/\n{3,}/g, "\n\n");

  // Restore fenced code blocks
  if (_codeBlocks.length > 0) {
    result = result.replace(/\x00FENCE\x00(\d+)\x00FENCE\x00/g, (_, i) => _codeBlocks[+i]);
  }

  // Runtime assertion: number of fence markers must be the same before and after processing
  const _inputFenceCount = (markdown.match(/`{3,}/g) || []).length;
  const _outputFenceCount = (result.match(/`{3,}/g) || []).length;
  if (_inputFenceCount !== _outputFenceCount) {
    core.warning(`[renderMarkdownTemplate] Fence count mismatch: input had ${_inputFenceCount} fence marker(s), output has ${_outputFenceCount}`);
  }

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

module.exports = { renderMarkdownTemplate, main, selectBranch };
