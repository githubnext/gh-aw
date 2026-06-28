// @ts-check

/**
 * Tests for eval_harness.cjs
 */

import { describe, it, expect } from "vitest";
import { readEvalSpec, buildEvalPrompt, aggregateResults, renderMarkdownSummary, sanitizeEvalError } from "./eval_harness.cjs";

// ---------------------------------------------------------------------------
// readEvalSpec
// ---------------------------------------------------------------------------

describe("readEvalSpec", () => {
  it("returns empty array when GH_AW_EVAL_SPEC is absent", () => {
    delete process.env.GH_AW_EVAL_SPEC;
    const result = readEvalSpec();
    expect(result).toEqual([]);
  });

  it("returns empty array when GH_AW_EVAL_SPEC is empty array JSON", () => {
    process.env.GH_AW_EVAL_SPEC = "[]";
    expect(readEvalSpec()).toEqual([]);
  });

  it("parses a valid eval spec", () => {
    process.env.GH_AW_EVAL_SPEC = JSON.stringify([
      { id: "builds", question: "Does the code compile?" },
      { id: "tests", question: "Are all tests passing?" },
    ]);
    const result = readEvalSpec();
    expect(result).toHaveLength(2);
    expect(result[0]).toEqual({ id: "builds", question: "Does the code compile?" });
    expect(result[1]).toEqual({ id: "tests", question: "Are all tests passing?" });
  });

  it("filters out entries with missing id or question", () => {
    const spec = [{ id: "valid", question: "Is it good?" }, { id: "", question: "Missing id" }, { id: "no-question", question: "" }, { question: "No id field" }];
    process.env.GH_AW_EVAL_SPEC = JSON.stringify(spec);
    const result = readEvalSpec();
    expect(result).toHaveLength(1);
    expect(result[0].id).toBe("valid");
  });

  it("throws when GH_AW_EVAL_SPEC is invalid JSON", () => {
    process.env.GH_AW_EVAL_SPEC = "not-json";
    expect(() => readEvalSpec()).toThrow(/Failed to parse GH_AW_EVAL_SPEC/);
  });

  it("throws when GH_AW_EVAL_SPEC is not an array", () => {
    process.env.GH_AW_EVAL_SPEC = JSON.stringify({ id: "x" });
    expect(() => readEvalSpec()).toThrow(/must be a JSON array/);
  });
});

// ---------------------------------------------------------------------------
// buildEvalPrompt
// ---------------------------------------------------------------------------

describe("buildEvalPrompt", () => {
  it("includes the question in the prompt", () => {
    const prompt = buildEvalPrompt("Does the code compile?", "");
    expect(prompt).toContain("Does the code compile?");
  });

  it("includes agent context when provided", () => {
    const prompt = buildEvalPrompt("Is it good?", "Agent output: success");
    expect(prompt).toContain("Agent output: success");
    expect(prompt).toContain("Agent Output Context");
  });

  it("omits context section when agent context is empty", () => {
    const prompt = buildEvalPrompt("Is it good?", "");
    expect(prompt).not.toContain("Agent Output Context");
  });

  it("requires binary yes/no answer in JSON format", () => {
    const prompt = buildEvalPrompt("Question?", "Context");
    expect(prompt).toContain('"passed"');
    expect(prompt).toContain('"rationale"');
    expect(prompt).toContain('"confidence"');
    expect(prompt).toContain("JSON object");
  });

  it("instructs to respond only with JSON", () => {
    const prompt = buildEvalPrompt("Question?", "Context");
    expect(prompt).toContain("Respond only with the JSON object");
  });
});

// ---------------------------------------------------------------------------
// aggregateResults
// ---------------------------------------------------------------------------

describe("aggregateResults", () => {
  it("returns zero counts for empty results", () => {
    const summary = aggregateResults([]);
    expect(summary.total).toBe(0);
    expect(summary.passed).toBe(0);
    expect(summary.failed).toBe(0);
    expect(summary.pass_rate).toBe(0);
    expect(summary.results).toEqual([]);
  });

  it("counts all passed when all pass", () => {
    const results = [
      { id: "a", passed: true, rationale: "yes" },
      { id: "b", passed: true, rationale: "yes" },
    ];
    const summary = aggregateResults(results);
    expect(summary.total).toBe(2);
    expect(summary.passed).toBe(2);
    expect(summary.failed).toBe(0);
    expect(summary.pass_rate).toBe(1);
  });

  it("counts all failed when all fail", () => {
    const results = [
      { id: "a", passed: false, rationale: "no" },
      { id: "b", passed: false, rationale: "no" },
    ];
    const summary = aggregateResults(results);
    expect(summary.total).toBe(2);
    expect(summary.passed).toBe(0);
    expect(summary.failed).toBe(2);
    expect(summary.pass_rate).toBe(0);
  });

  it("computes correct pass rate for mixed results", () => {
    const results = [
      { id: "a", passed: true },
      { id: "b", passed: false },
      { id: "c", passed: true },
      { id: "d", passed: false },
    ];
    const summary = aggregateResults(results);
    expect(summary.total).toBe(4);
    expect(summary.passed).toBe(2);
    expect(summary.failed).toBe(2);
    expect(summary.pass_rate).toBe(0.5);
  });

  it("preserves result order", () => {
    const results = [
      { id: "c", passed: true },
      { id: "a", passed: false },
      { id: "b", passed: true },
    ];
    const summary = aggregateResults(results);
    expect(summary.results.map(r => r.id)).toEqual(["c", "a", "b"]);
  });

  it("aggregation is deterministic (pass_rate = passed / total)", () => {
    const results = Array.from({ length: 10 }, (_, i) => ({
      id: `q${i}`,
      passed: i < 7,
    }));
    const summary = aggregateResults(results);
    expect(summary.passed).toBe(7);
    expect(summary.failed).toBe(3);
    expect(summary.pass_rate).toBeCloseTo(0.7, 10);
  });
});

// ---------------------------------------------------------------------------
// renderMarkdownSummary
// ---------------------------------------------------------------------------

describe("renderMarkdownSummary", () => {
  it("includes BinEval heading", () => {
    const summary = aggregateResults([]);
    const md = renderMarkdownSummary(summary);
    expect(md).toContain("BinEval Results");
  });

  it("shows pass count and total", () => {
    const summary = aggregateResults([
      { id: "a", passed: true, rationale: "ok" },
      { id: "b", passed: false, rationale: "nope" },
    ]);
    const md = renderMarkdownSummary(summary);
    expect(md).toContain("1/2 passed");
  });

  it("shows pass rate as percentage", () => {
    const summary = aggregateResults([
      { id: "a", passed: true },
      { id: "b", passed: true },
      { id: "c", passed: false },
      { id: "d", passed: false },
    ]);
    const md = renderMarkdownSummary(summary);
    expect(md).toContain("50.0%");
  });

  it("includes pass/fail icons for each result", () => {
    const summary = aggregateResults([
      { id: "pass-q", passed: true, rationale: "good" },
      { id: "fail-q", passed: false, rationale: "bad" },
    ]);
    const md = renderMarkdownSummary(summary);
    expect(md).toContain("✅");
    expect(md).toContain("❌");
  });

  it("includes question IDs in the table", () => {
    const summary = aggregateResults([{ id: "my-eval-question", passed: true }]);
    const md = renderMarkdownSummary(summary);
    expect(md).toContain("my-eval-question");
  });

  it("escapes pipe characters in rationale to avoid breaking markdown table", () => {
    const summary = aggregateResults([{ id: "a", passed: true, rationale: "a | b | c" }]);
    const md = renderMarkdownSummary(summary);
    // Pipes within the rationale cell should be escaped
    expect(md).toContain("a \\| b \\| c");
  });

  it("renders a markdown table with correct columns", () => {
    const summary = aggregateResults([{ id: "q1", passed: true, rationale: "looks good" }]);
    const md = renderMarkdownSummary(summary);
    expect(md).toContain("| Question ID |");
    expect(md).toContain("| Result |");
    expect(md).toContain("| Rationale |");
  });
});

// ---------------------------------------------------------------------------
// sanitizeEvalError
// ---------------------------------------------------------------------------

describe("sanitizeEvalError", () => {
  it("returns the error message from an Error object", () => {
    const result = sanitizeEvalError(new Error("something went wrong"));
    expect(result).toContain("something went wrong");
  });

  it("redacts bearer tokens", () => {
    const result = sanitizeEvalError(new Error("****** was rejected"));
    expect(result).not.toContain("secret-token");
    expect(result).toContain("[REDACTED_TOKEN]");
  });

  it("redacts GitHub personal access tokens", () => {
    const result = sanitizeEvalError(new Error("token ghp_ABCDEFGH1234567890 expired"));
    expect(result).not.toContain("ghp_ABCDEFGH1234567890");
    expect(result).toContain("[REDACTED_TOKEN]");
  });

  it("redacts URLs", () => {
    const result = sanitizeEvalError(new Error("failed to fetch https://api.example.com/secret"));
    expect(result).not.toContain("https://api.example.com/secret");
    expect(result).toContain("[REDACTED_URL]");
  });

  it("handles non-Error values", () => {
    expect(sanitizeEvalError("plain string error")).toContain("plain string error");
    expect(sanitizeEvalError(null)).toBe("unknown error");
    expect(sanitizeEvalError(undefined)).toBe("unknown error");
  });

  it("truncates very long error messages to 200 chars", () => {
    const longMessage = "x".repeat(500);
    const result = sanitizeEvalError(new Error(longMessage));
    expect(result.length).toBeLessThanOrEqual(200);
  });
});
