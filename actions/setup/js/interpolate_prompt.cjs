// @ts-check
/// <reference types="@actions/github-script" />

// @safe-outputs-exempt SEC-004 — no issue body is read or reflected; the only "body" occurrence is
// a literal log string ("body") used to describe a template branch, not user-controlled content.

// interpolate_prompt.cjs
// Interpolates GitHub Actions expressions and renders template conditionals in the prompt file.
// This combines variable interpolation and template filtering into a single step.

const fs = require("fs");
const { isTruthy } = require("./is_truthy.cjs");
const { processRuntimeImports } = require("./runtime_import.cjs");
const { writeInlineSubAgents } = require("./extract_inline_sub_agents.cjs");
const { getErrorMessage } = require("./error_helpers.cjs");
const { ERR_API, ERR_CONFIG, ERR_VALIDATION } = require("./error_codes.cjs");

/**
 * Interpolates variables in the prompt content
 * @param {string} content - The prompt content with ${GH_AW_EXPR_*} placeholders
 * @param {Record<string, string>} variables - Map of variable names to their values
 * @returns {string} - The interpolated content
 */
function interpolateVariables(content, variables) {
  core.info(`[interpolateVariables] Starting interpolation with ${Object.keys(variables).length} variables`);
  core.info(`[interpolateVariables] Content length: ${content.length} characters`);

  let result = content;
  let totalReplacements = 0;

  // Replace each ${VAR_NAME} with its corresponding value
  for (const [varName, value] of Object.entries(variables)) {
    const pattern = new RegExp(`\\$\\{${varName}\\}`, "g");
    const matches = (content.match(pattern) || []).length;

    if (matches > 0) {
      core.info(`[interpolateVariables] Replacing ${varName} (${matches} occurrence(s))`);
      core.info(`[interpolateVariables]   Value: ${value.substring(0, 100)}${value.length > 100 ? "..." : ""}`);
      result = result.replace(pattern, value);
      totalReplacements += matches;
    } else {
      core.info(`[interpolateVariables] Variable ${varName} not found in content (unused)`);
    }
  }

  core.info(`[interpolateVariables] Completed: ${totalReplacements} total replacement(s)`);
  core.info(`[interpolateVariables] Result length: ${result.length} characters`);
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
  const blockConditionals = (_stripped.match(/(\n?)([ \t]*{{#if\s+([^}]*)}}[ \t]*\n)([\s\S]*?)([ \t]*(?:{{#endif}}|{{\/if}})[ \t]*)(\n?)/g) || []).length;
  const inlineConditionals = (_stripped.match(/{{#if\s+([^}]*)}}([\s\S]*?)(?:{{#endif}}|{{\/if}})/g) || []).length - blockConditionals;

  core.info(`[renderMarkdownTemplate] Found ${blockConditionals} block conditional(s) and ${inlineConditionals} inline conditional(s)`);

  let blockCount = 0;
  let keptBlocks = 0;
  let removedBlocks = 0;

  // First pass: Handle blocks where tags are on their own lines
  // Captures: (leading newline)(opening tag line)(condition)(body)(closing tag line)(trailing newline)
  // Closing tag: {{#endif}} (primary) or {{/if}} (alternate)
  let result = _stripped.replace(/(\n?)([ \t]*{{#if\s+([^}]*)}}[ \t]*\n)([\s\S]*?)([ \t]*(?:{{#endif}}|{{\/if}})[ \t]*)(\n?)/g, (match, leadNL, openLine, cond, body, closeLine, trailNL) => {
    blockCount++;
    const condTrimmed = cond.trim();
    const truthyResult = isTruthy(cond);
    const bodyPreview = body.substring(0, 60).replace(/\n/g, "\\n");

    core.info(`[renderMarkdownTemplate] Block ${blockCount}: condition="${condTrimmed}" -> ${truthyResult ? "KEEP" : "REMOVE"}`);
    core.info(`[renderMarkdownTemplate]   Body preview: "${bodyPreview}${body.length > 60 ? "..." : ""}"`);

    // Split on {{#else}} if present to support two-branch conditionals.
    // e.g. {{#if experiments.prompt_style == "concise"}} ... {{#else}} ... {{#endif}}
    const elseParts = body.split(/[ \t]*\{\{#else\}\}[ \t]*\n?/);
    const trueBranch = elseParts[0];
    const falseBranch = elseParts.length > 1 ? elseParts.slice(1).join("{{#else}}") : null;

    if (truthyResult) {
      // Keep the true branch (before {{#else}}, or full body if no {{#else}})
      keptBlocks++;
      core.info(`[renderMarkdownTemplate]   Action: Keeping ${falseBranch !== null ? "true branch" : "body"} with leading newline=${!!leadNL}`);
      return leadNL + trueBranch;
    } else {
      // Remove the block, or keep the false branch when {{#else}} is present
      removedBlocks++;
      if (falseBranch !== null) {
        core.info(`[renderMarkdownTemplate]   Action: Keeping false branch ({{#else}} branch)`);
        return leadNL + falseBranch;
      }
      core.info(`[renderMarkdownTemplate]   Action: Removing entire block`);
      return "";
    }
  });

  core.info(`[renderMarkdownTemplate] First pass complete: ${keptBlocks} kept, ${removedBlocks} removed`);

  let inlineCount = 0;
  let keptInline = 0;
  let removedInline = 0;

  // Second pass: Handle inline conditionals (tags not on their own lines)
  // Closing tag: {{#endif}} (primary) or {{/if}} (alternate)
  result = result.replace(/{{#if\s+([^}]*)}}([\s\S]*?)(?:{{#endif}}|{{\/if}})/g, (_, cond, body) => {
    inlineCount++;
    const condTrimmed = cond.trim();
    const truthyResult = isTruthy(cond);
    const bodyPreview = body.substring(0, 40).replace(/\n/g, "\\n");

    core.info(`[renderMarkdownTemplate] Inline ${inlineCount}: condition="${condTrimmed}" -> ${truthyResult ? "KEEP" : "REMOVE"}`);
    core.info(`[renderMarkdownTemplate]   Body preview: "${bodyPreview}${body.length > 40 ? "..." : ""}"`);

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
  const beforeCleanup = result.length;
  const excessiveLines = (result.match(/\n{3,}/g) || []).length;
  result = result.replace(/\n{3,}/g, "\n\n");

  if (excessiveLines > 0) {
    core.info(`[renderMarkdownTemplate] Cleaned up ${excessiveLines} excessive blank line sequence(s)`);
    core.info(`[renderMarkdownTemplate] Length change from cleanup: ${beforeCleanup} -> ${result.length} characters`);
  }
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
 * Main function for prompt variable interpolation and template rendering
 */
async function main() {
  try {
    core.info("========================================");
    core.info("[main] Starting interpolate_prompt processing");
    core.info("========================================");

    const promptPath = process.env.GH_AW_PROMPT;
    if (!promptPath) {
      core.setFailed(`${ERR_CONFIG}: GH_AW_PROMPT environment variable is not set`);
      return;
    }
    core.info(`[main] Prompt path: ${promptPath}`);

    // Get the workspace directory for runtime imports
    const workspaceDir = process.env.GITHUB_WORKSPACE;
    if (!workspaceDir) {
      core.setFailed(`${ERR_CONFIG}: GITHUB_WORKSPACE environment variable is not set`);
      return;
    }
    core.info(`[main] Workspace directory: ${workspaceDir}`);

    // Read the prompt file
    core.info(`[main] Reading prompt file...`);
    let content = fs.readFileSync(promptPath, "utf8");
    const originalLength = content.length;
    core.info(`[main] Original content length: ${originalLength} characters`);
    core.info(`[main] First 200 characters: ${content.substring(0, 200).replace(/\n/g, "\\n")}`);

    // Step 1: Process runtime imports (files and URLs)
    core.info("\n========================================");
    core.info("[main] STEP 1: Runtime Imports");
    core.info("========================================");
    const hasRuntimeImports = /{{#runtime-import\??[ \t]+[^\}]+}}/.test(content);
    if (hasRuntimeImports) {
      const importMatches = content.match(/{{#runtime-import\??[ \t]+[^\}]+}}/g) || [];
      core.info(`Processing ${importMatches.length} runtime import macro(s) (files and URLs)`);
      importMatches.forEach((match, i) => {
        core.info(`  Import ${i + 1}: ${match.substring(0, 80)}${match.length > 80 ? "..." : ""}`);
      });

      const beforeImports = content.length;
      content = await processRuntimeImports(content, workspaceDir);
      const afterImports = content.length;

      core.info(`Runtime imports processed successfully`);
      core.info(`Content length change: ${beforeImports} -> ${afterImports} (${afterImports > beforeImports ? "+" : ""}${afterImports - beforeImports})`);
    } else {
      core.info("No runtime import macros found, skipping runtime import processing");
    }

    // Step 1.5: Extract and write inline sub-agents
    // ## agent: name / ## end: name blocks are written to .github/agents/<name>.md.
    // This happens after runtime imports so that any {{#runtime-import}} macros
    // inside an agent block have already been resolved.
    core.info("\n========================================");
    core.info("[main] STEP 1.5: Inline Sub-Agent Extraction");
    core.info("========================================");
    const hasAgentMarkers = /^##[ \t]+agent:[ \t]+`[a-z]/m.test(content);
    if (hasAgentMarkers) {
      const beforeExtraction = content.length;
      // Write agents to /tmp/gh-aw/<engine-dir>/ so the files are included in the
      // activation artifact and available to the downstream agent job.
      const agentsBaseDir = "/tmp/gh-aw";
      const engineId = process.env.GH_AW_ENGINE_ID || "";
      content = writeInlineSubAgents(content, workspaceDir, agentsBaseDir, engineId);
      const afterExtraction = content.length;
      core.info(`Inline sub-agents extracted and written`);
      core.info(`Content length change: ${beforeExtraction} -> ${afterExtraction} (${afterExtraction > beforeExtraction ? "+" : ""}${afterExtraction - beforeExtraction})`);
    } else {
      core.info("No inline sub-agent markers found, skipping");
    }

    // Step 2: Interpolate variables
    core.info("\n========================================");
    core.info("[main] STEP 2: Variable Interpolation");
    core.info("========================================");
    /** @type {Record<string, string>} */
    const variables = {};
    for (const [key, value] of Object.entries(process.env)) {
      if (key.startsWith("GH_AW_EXPR_")) {
        variables[key] = value || "";
      }
    }

    const varCount = Object.keys(variables).length;
    if (varCount > 0) {
      core.info(`Found ${varCount} expression variable(s) to interpolate:`);
      for (const [key, value] of Object.entries(variables)) {
        const preview = value.substring(0, 60);
        core.info(`  ${key}: ${preview}${value.length > 60 ? "..." : ""}`);
      }

      const beforeInterpolation = content.length;
      content = interpolateVariables(content, variables);
      const afterInterpolation = content.length;

      core.info(`Successfully interpolated ${varCount} variable(s) in prompt`);
      core.info(`Content length change: ${beforeInterpolation} -> ${afterInterpolation} (${afterInterpolation > beforeInterpolation ? "+" : ""}${afterInterpolation - beforeInterpolation})`);
    } else {
      core.info("No expression variables found, skipping interpolation");
    }

    // Step 2.5: Substitute experiment placeholders BEFORE template rendering.
    // When the runtime-import step processes {{#if experiments.name}} conditionals,
    // it converts them to __GH_AW_EXPERIMENTS_NAME__ placeholders. These must be
    // resolved with the actual variant value before renderMarkdownTemplate() runs,
    // otherwise the placeholder string is truthy and the block is always kept.
    // The activation job exposes GH_AW_EXPERIMENTS_* env vars (from the pick-experiment
    // step output via the step's env: block), so we can substitute them here.
    // Additionally, {{#if experiments.name == "value"}} conditions use the dot-notation
    // form directly in the condition expression. We substitute experiments.NAME → actual
    // value inside {{#if ...}} condition tags so that isTruthy can evaluate the resulting
    // GitHub Actions script style expression (e.g. concise == "concise").
    core.info("\n========================================");
    core.info("[main] STEP 2.5: Experiment Placeholder Substitution");
    core.info("========================================");
    let experimentSubCount = 0;
    for (const [key, value] of Object.entries(process.env)) {
      if (key.startsWith("GH_AW_EXPERIMENTS_")) {
        const placeholder = `__${key}__`;
        if (content.includes(placeholder)) {
          content = content.split(placeholder).join(value || "");
          experimentSubCount++;
          core.info(`  Substituted ${placeholder} → "${value || ""}"`);
        }
        // Also substitute experiments.name references inside {{#if ...}} conditions.
        // This enables GitHub Actions script style comparisons (e.g. prompt_style == "concise")
        // to resolve correctly — after substitution the condition becomes: concise == "concise".
        const experimentName = key.substring("GH_AW_EXPERIMENTS_".length).toLowerCase();
        const exprForm = `experiments.${experimentName}`;
        const conditionPattern = new RegExp(`(\\{\\{#if[^}]*?)${exprForm.replace(".", "\\.")}`, "gi");
        if (conditionPattern.test(content)) {
          conditionPattern.lastIndex = 0;
          content = content.replace(conditionPattern, `$1${value || ""}`);
          core.info(`  Substituted ${exprForm} in conditions → "${value || ""}"`);
        }
      }
    }
    if (experimentSubCount > 0) {
      core.info(`Substituted ${experimentSubCount} experiment placeholder(s)`);
    } else {
      core.info("No experiment placeholders found in prompt");
    }

    // Step 3: Render template conditionals
    core.info("\n========================================");
    core.info("[main] STEP 3: Template Rendering");
    core.info("========================================");
    const hasConditionals = /{{#if\s+[^}]+}}/.test(content);
    if (hasConditionals) {
      const conditionalMatches = content.match(/{{#if\s+[^}]+}}/g) || [];
      core.info(`Processing ${conditionalMatches.length} conditional template block(s)`);

      const beforeRendering = content.length;
      content = renderMarkdownTemplate(content);
      const afterRendering = content.length;

      core.info(`Template rendered successfully`);
      core.info(`Content length change: ${beforeRendering} -> ${afterRendering} (${afterRendering > beforeRendering ? "+" : ""}${afterRendering - beforeRendering})`);
    } else {
      core.info("No conditional blocks found in prompt, skipping template rendering");
    }

    // Write back to the same file
    core.info("\n========================================");
    core.info("[main] STEP 4: Writing Output");
    core.info("========================================");
    core.info(`Writing processed content back to: ${promptPath}`);
    core.info(`Final content length: ${content.length} characters`);
    core.info(`Total length change: ${originalLength} -> ${content.length} (${content.length > originalLength ? "+" : ""}${content.length - originalLength})`);

    fs.writeFileSync(promptPath, content, "utf8");

    core.info(`Last 200 characters: ${content.substring(Math.max(0, content.length - 200)).replace(/\n/g, "\\n")}`);
    core.info("========================================");
    core.info("[main] Processing complete - SUCCESS");
    core.info("========================================");
  } catch (error) {
    core.info("========================================");
    core.info("[main] Processing failed - ERROR");
    core.info("========================================");
    const err = error instanceof Error ? error : new Error(String(error));
    core.info(`[main] Error type: ${err.constructor.name}`);
    core.info(`[main] Error message: ${err.message}`);
    if (err.stack) {
      core.info(`[main] Stack trace:\n${err.stack}`);
    }
    core.setFailed(`${ERR_API}: ${getErrorMessage(error)}`);
  }
}

module.exports = { main };
