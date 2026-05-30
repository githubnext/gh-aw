// @ts-check
/**
 * Fuzz test harness for processExpressions in runtime_import.cjs.
 *
 * Tests various combinations of GitHub Actions expression insertions in content,
 * including duplicate expressions and evaluated values that themselves contain
 * expression-like syntax.
 *
 * Used by tests and by Go's fuzzing framework when run as main module via stdin.
 */

/**
 * @typedef {Object} MockContext
 * @property {string} [actor]
 * @property {string} [job]
 * @property {string} [owner]
 * @property {string} [repo]
 * @property {number} [runId]
 * @property {number} [runNumber]
 * @property {string} [workflow]
 * @property {number|undefined} [issueNumber]
 * @property {string|undefined} [issueTitle]
 * @property {number|undefined} [prNumber]
 * @property {string|undefined} [prTitle]
 * @property {string} [serverUrl]
 * @property {string} [workspace]
 */

/**
 * Sets up the global mock context and core logger required by runtime_import.cjs.
 * @param {MockContext} ctx - Optional overrides for context fields
 */
function setupMockContext(ctx) {
  /** @type {any} */
  const g = global;
  g.core = {
    info: () => {},
    warning: () => {},
    setFailed: () => {},
  };

  g.context = {
    actor: ctx.actor ?? "testuser",
    job: ctx.job ?? "test-job",
    repo: { owner: ctx.owner ?? "testorg", repo: ctx.repo ?? "testrepo" },
    runId: ctx.runId ?? 12345,
    runNumber: ctx.runNumber ?? 42,
    workflow: ctx.workflow ?? "test-workflow",
    payload: {
      issue: ctx.issueNumber !== undefined || ctx.issueTitle !== undefined ? { number: ctx.issueNumber ?? 0, title: ctx.issueTitle ?? "", state: "open" } : { number: 123, title: "Test Issue", state: "open" },
      pull_request: ctx.prNumber !== undefined || ctx.prTitle !== undefined ? { number: ctx.prNumber ?? 0, title: ctx.prTitle ?? "", state: "open" } : { number: 456, title: "Test PR", state: "open" },
      sender: { id: 789 },
    },
  };

  process.env.GITHUB_SERVER_URL = ctx.serverUrl ?? "https://github.com";
  process.env.GITHUB_WORKSPACE = ctx.workspace ?? "/workspace";
}

/**
 * Calls processExpressions with a fresh mock context and returns the result.
 * @param {string} content - Markdown content with zero or more `${{ … }}` expressions
 * @param {MockContext} [ctx] - Optional context value overrides
 * @returns {{ result: string | null, error: string | null }}
 */
function testProcessExpressions(content, ctx) {
  setupMockContext(ctx ?? {});

  // Re-require every call so module state is reset between tests
  const { processExpressions } = require("./runtime_import.cjs");
  try {
    const result = processExpressions(content, "fuzz-test.md");
    return { result, error: null };
  } catch (err) {
    return { result: null, error: err instanceof Error ? err.message : String(err) };
  }
}

// Read input from stdin for Go-driven fuzzing
if (require.main === module) {
  let input = "";

  process.stdin.on("data", chunk => {
    input += chunk;
  });

  process.stdin.on("end", () => {
    try {
      // Expected JSON: { content: string, ctx?: MockContext }
      const { content, ctx } = JSON.parse(input);
      const result = testProcessExpressions(content ?? "", ctx ?? {});
      process.stdout.write(JSON.stringify(result));
      process.exit(0);
    } catch (err) {
      const errorMsg = err instanceof Error ? err.message : String(err);
      process.stdout.write(JSON.stringify({ result: null, error: errorMsg }));
      process.exit(1);
    }
  });
}

module.exports = { testProcessExpressions };
