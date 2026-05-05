// @ts-check
/**
 * Fuzz harness for {{#if / #elseif / #else}} template branch selection and rendering.
 *
 * Reads a JSON test case from stdin:
 *   { ifCondition: string, body: string }   — for selectBranch
 *   { markdown: string }                     — for renderMarkdownTemplate
 *
 * Writes the result as JSON to stdout:
 *   { result: string|null, error: string|null }
 *
 * Used by the Go fuzz driver in template_conditional_js_fuzz_test.go.
 */

const { selectBranch } = require("./template_branch.cjs");
const { isTruthy } = require("./is_truthy.cjs");

// Minimal shim so renderMarkdownTemplate can call core.info
if (!global.core) {
  global.core = {
    info: () => {},
    warning: () => {},
    setFailed: () => {},
  };
}

const renderTemplateScript = require("fs").readFileSync(require("path").join(__dirname, "render_template.cjs"), "utf8");
const renderMarkdownTemplateMatch = renderTemplateScript.match(/function renderMarkdownTemplate\(markdown\)\s*{[\s\S]*?return result;[\s\S]*?}/);
if (!renderMarkdownTemplateMatch) throw new Error("Could not extract renderMarkdownTemplate");
const renderMarkdownTemplate = eval(`(${renderMarkdownTemplateMatch[0]})`);

if (require.main === module) {
  let input = "";
  process.stdin.on("data", chunk => {
    input += chunk;
  });
  process.stdin.on("end", () => {
    try {
      const parsed = JSON.parse(input);

      let result;
      if (Object.prototype.hasOwnProperty.call(parsed, "ifCondition")) {
        // selectBranch test
        result = { result: selectBranch(parsed.ifCondition, parsed.body || ""), error: null };
      } else if (Object.prototype.hasOwnProperty.call(parsed, "markdown")) {
        // renderMarkdownTemplate test
        result = { result: renderMarkdownTemplate(parsed.markdown || ""), error: null };
      } else {
        result = { result: null, error: "Unknown test type: expected 'ifCondition' or 'markdown' key" };
      }

      process.stdout.write(JSON.stringify(result));
      process.exit(0);
    } catch (err) {
      process.stdout.write(JSON.stringify({ result: null, error: err instanceof Error ? err.message : String(err) }));
      process.exit(1);
    }
  });
}

module.exports = { selectBranch, renderMarkdownTemplate };
