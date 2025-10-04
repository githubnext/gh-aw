// render_template.cjs
// Single-function Markdown → Markdown postprocessor for GitHub Actions.
// Processes only {{#if <expr>}} ... {{/if}} blocks after ${{ }} evaluation.

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
 * Renders a Markdown template by processing {{#if}} conditional blocks
 * @param {string} markdown - The markdown content to process
 * @returns {string} - The processed markdown content
 */
function renderMarkdownTemplate(markdown) {
  return markdown.replace(/{{#if\s+([^}]+)}}([\s\S]*?){{\/if}}/g, (_, cond, body) => (isTruthy(cond) ? body : ""));
}

// Main execution for GitHub Actions
try {
  const promptPath = process.env.GITHUB_AW_PROMPT;
  if (!promptPath) {
    core.setFailed("GITHUB_AW_PROMPT environment variable is not set");
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
  core.summary.addHeading("Template Rendering", 3).addRaw("\n").addRaw("Processed conditional blocks in prompt\n").write();
} catch (error) {
  core.setFailed(error instanceof Error ? error.message : String(error));
}
