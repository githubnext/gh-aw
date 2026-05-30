// @ts-check
/**
 * Fuzz tests for processExpressions – various combinations of expression inserts.
 *
 * Covers:
 * - Every individual safe expression
 * - Pairs and triples of different safe expressions
 * - Duplicate expressions (same expression N times in the same content)
 * - Expression values that themselves contain expression-like syntax
 * - Expressions at different positions (start, middle, end, adjacent)
 * - Empty or expression-free content
 * - Unsafe expressions (must throw)
 * - Interleaved safe + unsafe expressions (must throw)
 */

const { testProcessExpressions } = require("./fuzz_runtime_import_expressions_harness.cjs");

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

/**
 * All safe single-level expressions (with expected values from the default mock context).
 * @type {Array<{ expr: string, value: string }>}
 */
const SAFE_EXPRESSIONS = [
  { expr: "github.actor", value: "testuser" },
  { expr: "github.job", value: "test-job" },
  { expr: "github.repository", value: "testorg/testrepo" },
  { expr: "github.repository_owner", value: "testorg" },
  { expr: "github.run_id", value: "12345" },
  { expr: "github.run_number", value: "42" },
  { expr: "github.server_url", value: "https://github.com" },
  { expr: "github.workflow", value: "test-workflow" },
  { expr: "github.workspace", value: "/workspace" },
  { expr: "github.event.issue.number", value: "123" },
  { expr: "github.event.issue.title", value: "Test Issue" },
  { expr: "github.event.pull_request.number", value: "456" },
  { expr: "github.event.pull_request.title", value: "Test PR" },
];

/** Builds the interpolation token for an expression. */
const token = expr => `\${{ ${expr} }}`;

// ---------------------------------------------------------------------------
// Individual safe expressions
// ---------------------------------------------------------------------------

describe("fuzz_runtime_import_expressions_harness – individual expressions", () => {
  for (const { expr, value } of SAFE_EXPRESSIONS) {
    it(`should replace $\{{ ${expr} }} with its evaluated value`, () => {
      const content = `Value: ${token(expr)}`;
      const { result, error } = testProcessExpressions(content);
      expect(error).toBeNull();
      expect(result).toBe(`Value: ${value}`);
    });
  }
});

// ---------------------------------------------------------------------------
// Duplicate (repeated) expressions
// ---------------------------------------------------------------------------

describe("fuzz_runtime_import_expressions_harness – duplicate expressions", () => {
  for (const { expr, value } of SAFE_EXPRESSIONS) {
    it(`should replace all three occurrences of $\{{ ${expr} }}`, () => {
      const content = `A: ${token(expr)}, B: ${token(expr)}, C: ${token(expr)}`;
      const { result, error } = testProcessExpressions(content);
      expect(error).toBeNull();
      expect(result).toBe(`A: ${value}, B: ${value}, C: ${value}`);
    });
  }

  it("should replace 10 occurrences of the same expression", () => {
    const DUPLICATE_COUNT = 10;
    const expr = "github.actor";
    const value = "testuser";
    const content = Array.from({ length: DUPLICATE_COUNT }, (_, i) => `${i + 1}: ${token(expr)}`).join(" | ");
    const expected = Array.from({ length: DUPLICATE_COUNT }, (_, i) => `${i + 1}: ${value}`).join(" | ");
    const { result, error } = testProcessExpressions(content);
    expect(error).toBeNull();
    expect(result).toBe(expected);
  });

  it("should replace duplicate expression embedded in a markdown table", () => {
    const content = `| Field | Value |\n` + `| ----- | ----- |\n` + `| Actor | ${token("github.actor")} |\n` + `| Also  | ${token("github.actor")} |\n`;
    const { result, error } = testProcessExpressions(content);
    expect(error).toBeNull();
    expect(result).toContain("| Actor | testuser |");
    expect(result).toContain("| Also  | testuser |");
  });
});

// ---------------------------------------------------------------------------
// All pairs of distinct safe expressions
// ---------------------------------------------------------------------------

describe("fuzz_runtime_import_expressions_harness – pairs of distinct expressions", () => {
  for (let i = 0; i < SAFE_EXPRESSIONS.length; i++) {
    for (let j = i + 1; j < SAFE_EXPRESSIONS.length; j++) {
      const a = SAFE_EXPRESSIONS[i];
      const b = SAFE_EXPRESSIONS[j];
      it(`should replace $\{{ ${a.expr} }} and $\{{ ${b.expr} }} together`, () => {
        const content = `${token(a.expr)} | ${token(b.expr)}`;
        const { result, error } = testProcessExpressions(content);
        expect(error).toBeNull();
        expect(result).toBe(`${a.value} | ${b.value}`);
      });
    }
  }
});

// ---------------------------------------------------------------------------
// No double-substitution: evaluated values must not be re-processed
// ---------------------------------------------------------------------------

describe("fuzz_runtime_import_expressions_harness – no double-substitution", () => {
  it("should not re-replace ${{ github.actor }} found inside an evaluated issue title", () => {
    // The issue title itself contains expression-like text.
    const { result, error } = testProcessExpressions(`Title: ${token("github.event.issue.title")}, User: ${token("github.actor")}`, { issueTitle: "Use ${{ github.actor }} here" });
    expect(error).toBeNull();
    // The literal "${{ github.actor }}" in the title must survive unchanged.
    expect(result).toBe("Title: Use ${{ github.actor }} here, User: testuser");
  });

  it("should not re-replace ${{ github.repository }} found inside an evaluated actor value", () => {
    // Simulate an actor name that looks like an expression.
    const { result, error } = testProcessExpressions(`Actor: ${token("github.actor")}, Repo: ${token("github.repository")}`, { actor: "${{ github.repository }}" });
    expect(error).toBeNull();
    expect(result).toBe("Actor: ${{ github.repository }}, Repo: testorg/testrepo");
  });

  it("should not re-replace when duplicate expression value is expression-like", () => {
    const exprLikeTitle = "${{ github.run_id }} and ${{ github.actor }}";
    const { result, error } = testProcessExpressions(`${token("github.event.issue.title")} – ${token("github.event.issue.title")} – run: ${token("github.run_id")}`, { issueTitle: exprLikeTitle });
    expect(error).toBeNull();
    // Both copies of the title are preserved literally; run_id is replaced correctly.
    expect(result).toBe(`${exprLikeTitle} – ${exprLikeTitle} – run: 12345`);
  });
});

// ---------------------------------------------------------------------------
// Expression positions and adjacency
// ---------------------------------------------------------------------------

describe("fuzz_runtime_import_expressions_harness – expression positions", () => {
  it("should handle expression at the very start of content", () => {
    const { result, error } = testProcessExpressions(`${token("github.actor")} was here`);
    expect(error).toBeNull();
    expect(result).toBe("testuser was here");
  });

  it("should handle expression at the very end of content", () => {
    const { result, error } = testProcessExpressions(`Actor is ${token("github.actor")}`);
    expect(error).toBeNull();
    expect(result).toBe("Actor is testuser");
  });

  it("should handle two expressions with no surrounding text", () => {
    const { result, error } = testProcessExpressions(`${token("github.actor")}${token("github.run_id")}`);
    expect(error).toBeNull();
    expect(result).toBe("testuser12345");
  });

  it("should handle three adjacent expressions", () => {
    const content = `${token("github.actor")}/${token("github.repository")}#${token("github.run_id")}`;
    const { result, error } = testProcessExpressions(content);
    expect(error).toBeNull();
    expect(result).toBe("testuser/testorg/testrepo#12345");
  });

  it("should handle expression-only content (no surrounding text)", () => {
    const { result, error } = testProcessExpressions(token("github.actor"));
    expect(error).toBeNull();
    expect(result).toBe("testuser");
  });

  it("should handle expression inside a URL", () => {
    const content = `${process.env.GITHUB_SERVER_URL || "https://github.com"}/${token("github.repository")}/actions/runs/${token("github.run_id")}`;
    const { result, error } = testProcessExpressions(content);
    expect(error).toBeNull();
    expect(result).toContain("testorg/testrepo");
    expect(result).toContain("12345");
  });
});

// ---------------------------------------------------------------------------
// Expression-free content must pass through unchanged
// ---------------------------------------------------------------------------

describe("fuzz_runtime_import_expressions_harness – no expressions", () => {
  it("should return empty string unchanged", () => {
    const { result, error } = testProcessExpressions("");
    expect(error).toBeNull();
    expect(result).toBe("");
  });

  it("should return plain text unchanged", () => {
    const text = "No expressions here, just plain text.";
    const { result, error } = testProcessExpressions(text);
    expect(error).toBeNull();
    expect(result).toBe(text);
  });

  it("should return content with template conditionals unchanged", () => {
    const text = "{{#if something}}value{{/if}}";
    const { result, error } = testProcessExpressions(text);
    expect(error).toBeNull();
    expect(result).toBe(text);
  });
});

// ---------------------------------------------------------------------------
// Unsafe expressions must throw
// ---------------------------------------------------------------------------

describe("fuzz_runtime_import_expressions_harness – unsafe expressions", () => {
  const UNSAFE = ["secrets.GITHUB_TOKEN", "secrets.MY_SECRET", "runner.os", "runner.temp", "github.token", "vars.MY_VAR"];

  for (const expr of UNSAFE) {
    it(`should reject unsafe expression $\{{ ${expr} }}`, () => {
      const { error } = testProcessExpressions(`Value: ${token(expr)}`);
      expect(error).not.toBeNull();
      expect(error).toMatch(/unauthorized/i);
    });
  }

  it("should reject content mixing safe and unsafe expressions", () => {
    const content = `${token("github.actor")} and ${token("secrets.TOKEN")}`;
    const { error } = testProcessExpressions(content);
    expect(error).not.toBeNull();
    expect(error).toMatch(/unauthorized/i);
  });
});

// ---------------------------------------------------------------------------
// Large / stress combinations
// ---------------------------------------------------------------------------

describe("fuzz_runtime_import_expressions_harness – stress combinations", () => {
  it("should process all safe expressions in a single content string", () => {
    const content = SAFE_EXPRESSIONS.map(({ expr }) => `${expr}: ${token(expr)}`).join("\n");
    const { result, error } = testProcessExpressions(content);
    expect(error).toBeNull();
    for (const { expr, value } of SAFE_EXPRESSIONS) {
      expect(result).toContain(`${expr}: ${value}`);
    }
  });

  it("should process all safe expressions each repeated 3 times", () => {
    const content = SAFE_EXPRESSIONS.flatMap(({ expr }) => [token(expr), token(expr), token(expr)]).join(" ");
    const { result, error } = testProcessExpressions(content);
    expect(error).toBeNull();
    // Ensure no raw expression tokens remain
    expect(result).not.toMatch(/\$\{\{/);
  });
});
