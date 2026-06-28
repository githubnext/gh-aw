import { describe, it, expect, beforeEach, afterEach, vi } from "vitest";
import fs from "fs";
import path from "path";

// ---------------------------------------------------------------------------
// Unit tests for exported parse helpers — no fs needed
// ---------------------------------------------------------------------------

// Use require (not import) for CJS modules that export named functions directly;
// this avoids ESM-interop wrapping and keeps the references stable across tests.
const { parseEvalLog, extractResultFromText, extractFromStreamJson } = require("./parse_eval_results.cjs");

describe("extractResultFromText", () => {
  it("extracts a simple JSON object", () => {
    const text = 'EVAL_RESULT:{"results":[{"id":"builds","passed":true}]}';
    const result = extractResultFromText(text);
    expect(result).toBe('EVAL_RESULT:{"results":[{"id":"builds","passed":true}]}');
  });

  it("stops at the matching closing brace and ignores trailing content", () => {
    const text = 'EVAL_RESULT:{"results":[{"id":"builds","passed":true}]}\nSome trailing text';
    const result = extractResultFromText(text);
    expect(result).toBe('EVAL_RESULT:{"results":[{"id":"builds","passed":true}]}');
    expect(result).not.toContain("trailing");
  });

  it("handles nested objects correctly", () => {
    const text = 'EVAL_RESULT:{"results":[{"id":"q","passed":true,"meta":{"k":1}}]}';
    const result = extractResultFromText(text);
    expect(result).toBe('EVAL_RESULT:{"results":[{"id":"q","passed":true,"meta":{"k":1}}]}');
  });

  it("does not count braces inside JSON string values", () => {
    const text = 'EVAL_RESULT:{"results":[{"id":"q","rationale":"found {injection} here","passed":false}]}';
    const result = extractResultFromText(text);
    expect(result).toBe('EVAL_RESULT:{"results":[{"id":"q","rationale":"found {injection} here","passed":false}]}');
  });

  it("handles escaped quotes inside strings", () => {
    const text = 'EVAL_RESULT:{"results":[{"id":"q","rationale":"he said \\"yes\\"","passed":true}]}';
    const result = extractResultFromText(text);
    expect(result).toBe('EVAL_RESULT:{"results":[{"id":"q","rationale":"he said \\"yes\\"","passed":true}]}');
  });

  it("returns null when no opening brace is found", () => {
    expect(extractResultFromText("EVAL_RESULT:null")).toBeNull();
    expect(extractResultFromText("EVAL_RESULT:[]")).toBeNull();
    expect(extractResultFromText("EVAL_RESULT:")).toBeNull();
  });

  it("returns null when closing brace is missing (truncated JSON)", () => {
    expect(extractResultFromText('EVAL_RESULT:{"results":[')).toBeNull();
    expect(extractResultFromText('EVAL_RESULT:{"results":[{"id":"builds","passed":true')).toBeNull();
  });
});

describe("extractFromStreamJson", () => {
  it("extracts result from a stream-json envelope", () => {
    const inner = 'EVAL_RESULT:{"results":[{"id":"builds","passed":true,"rationale":"yes"}]}';
    const line = JSON.stringify({ type: "result", subtype: "success", result: inner });
    const result = extractFromStreamJson(line);
    expect(result).toContain("EVAL_RESULT:");
  });

  it("returns the original line when it does not contain EVAL_RESULT", () => {
    const line = '{"type":"text","text":"some output"}';
    expect(extractFromStreamJson(line)).toBe(line);
  });

  it("returns the original line when it is not valid JSON", () => {
    const line = "EVAL_RESULT:plaintext";
    expect(extractFromStreamJson(line)).toBe(line);
  });

  it("returns the original line when the outer JSON has no result field with EVAL_RESULT", () => {
    const line = JSON.stringify({ type: "text", text: "EVAL_RESULT: partial" });
    // outer.result is undefined → falls through to returning the original line
    const result = extractFromStreamJson(line);
    expect(result).toBe(line);
  });
});

describe("parseEvalLog", () => {
  it("extracts results from a plain log line", () => {
    const log = 'EVAL_RESULT:{"results":[{"id":"builds","passed":true,"rationale":"ok"},{"id":"tests","passed":false,"rationale":"no"}]}';
    const { results, error } = parseEvalLog(log);
    expect(error).toBeNull();
    expect(results).toHaveLength(2);
    expect(results[0]).toMatchObject({ id: "builds", passed: true });
    expect(results[1]).toMatchObject({ id: "tests", passed: false });
  });

  it("returns error when no EVAL_RESULT is found", () => {
    const log = "Engine output with no result marker\nSome other output";
    const { results, error } = parseEvalLog(log);
    expect(results).toBeNull();
    expect(error).toContain("No EVAL_RESULT found");
  });

  it("returns error when EVAL_RESULT JSON has no results array", () => {
    const log = 'EVAL_RESULT:{"passed":true}';
    const { results, error } = parseEvalLog(log);
    expect(results).toBeNull();
    expect(error).toContain("'results' array");
  });

  it("returns error when EVAL_RESULT JSON is malformed", () => {
    const log = "EVAL_RESULT:{malformed json}";
    const { results, error } = parseEvalLog(log);
    expect(results).toBeNull();
    expect(error).toContain("Failed to parse EVAL_RESULT JSON");
  });

  it("parses the first EVAL_RESULT in a multi-line log", () => {
    const log = [
      "Engine starting up...",
      "Reading prompt file...",
      'EVAL_RESULT:{"results":[{"id":"q1","passed":true,"rationale":"good"}]}',
      "Engine finishing.",
    ].join("\n");
    const { results, error } = parseEvalLog(log);
    expect(error).toBeNull();
    expect(results).toHaveLength(1);
    expect(results[0].id).toBe("q1");
  });

  it("handles EVAL_RESULT embedded in a stream-json wrapper", () => {
    const inner = 'EVAL_RESULT:{"results":[{"id":"focused","passed":false,"rationale":"not focused"}]}';
    const streamLine = JSON.stringify({ type: "result", subtype: "success", result: inner });
    const log = `Engine output\n${streamLine}\n`;
    const { results, error } = parseEvalLog(log);
    expect(error).toBeNull();
    expect(results).toHaveLength(1);
    expect(results[0]).toMatchObject({ id: "focused", passed: false });
  });

  it("returns empty results array when results JSON field is empty", () => {
    const log = 'EVAL_RESULT:{"results":[]}';
    const { results, error } = parseEvalLog(log);
    expect(error).toBeNull();
    expect(results).toEqual([]);
  });
});

// ---------------------------------------------------------------------------
// Integration tests for main() — uses real fs with a temp directory
// ---------------------------------------------------------------------------

// Use a directory that does not overlap with setup_eval.test.cjs (/tmp/gh-aw),
// preventing test-parallelism conflicts when both suites run simultaneously.
const PARSE_TEST_ROOT = "/tmp/gh-aw-parse-eval-test";
const EVAL_DIR = path.join(PARSE_TEST_ROOT, "eval");

// Use require (not dynamic import) so the module is cached once, avoiding
// vi.resetModules() interference with the real fs reference inside the module.
const { main: evalMain } = require("./parse_eval_results.cjs");

function makeCoreMocks() {
  const summary = {
    addRaw: vi.fn().mockReturnThis(),
    write: vi.fn().mockResolvedValue(undefined),
  };
  return {
    info: vi.fn(),
    warning: vi.fn(),
    error: vi.fn(),
    setFailed: vi.fn(),
    setOutput: vi.fn(),
    exportVariable: vi.fn(),
    summary,
  };
}

describe("main (parse_eval_results)", () => {
  beforeEach(() => {
    fs.rmSync(PARSE_TEST_ROOT, { recursive: true, force: true });
    fs.mkdirSync(EVAL_DIR, { recursive: true });
    process.env.GH_AW_EVAL_WORK_DIR = EVAL_DIR;
    global.core = makeCoreMocks();
  });

  afterEach(() => {
    fs.rmSync(PARSE_TEST_ROOT, { recursive: true, force: true });
    delete process.env.GH_AW_EVAL_WORK_DIR;
  });

  it("fails when eval log file does not exist", async () => {
    await evalMain();
    expect(global.core.setFailed).toHaveBeenCalledWith(expect.stringContaining("Eval log not found"));
  });

  it("fails when eval log cannot be parsed (no EVAL_RESULT marker)", async () => {
    fs.writeFileSync(path.join(EVAL_DIR, "eval.log"), "Engine ran but produced no result marker\n");
    await evalMain();
    expect(global.core.setFailed).toHaveBeenCalledWith(expect.stringContaining("No EVAL_RESULT found"));
  });

  it("writes eval_results.json and sets outputs on success", async () => {
    fs.writeFileSync(path.join(EVAL_DIR, "eval.log"), 'EVAL_RESULT:{"results":[{"id":"builds","passed":true,"rationale":"compiles"},{"id":"tests","passed":false,"rationale":"failing"}]}');
    await evalMain();

    expect(global.core.setFailed).not.toHaveBeenCalled();

    const resultsPath = path.join(EVAL_DIR, "eval_results.json");
    expect(fs.existsSync(resultsPath)).toBe(true);
    const summary = JSON.parse(fs.readFileSync(resultsPath, "utf-8"));
    expect(summary.total).toBe(2);
    expect(summary.passed).toBe(1);
    expect(summary.failed).toBe(1);
    expect(summary.pass_rate).toBeCloseTo(0.5);

    expect(global.core.setOutput).toHaveBeenCalledWith("eval_passed", "1");
    expect(global.core.setOutput).toHaveBeenCalledWith("eval_total", "2");
    expect(global.core.setOutput).toHaveBeenCalledWith("eval_pass_rate", "0.5000");
  });

  it("writes step summary with BinEval Results heading", async () => {
    fs.writeFileSync(path.join(EVAL_DIR, "eval.log"), 'EVAL_RESULT:{"results":[{"id":"q1","passed":true,"rationale":"ok"}]}');
    await evalMain();

    expect(global.core.summary.addRaw).toHaveBeenCalledWith(expect.stringContaining("BinEval Results"));
    expect(global.core.summary.write).toHaveBeenCalled();
  });

  it("handles all-passed results correctly", async () => {
    fs.writeFileSync(path.join(EVAL_DIR, "eval.log"), 'EVAL_RESULT:{"results":[{"id":"q1","passed":true},{"id":"q2","passed":true}]}');
    await evalMain();

    expect(global.core.setFailed).not.toHaveBeenCalled();
    expect(global.core.setOutput).toHaveBeenCalledWith("eval_passed", "2");
    expect(global.core.setOutput).toHaveBeenCalledWith("eval_total", "2");
    expect(global.core.setOutput).toHaveBeenCalledWith("eval_pass_rate", "1.0000");
  });

  it("normalizes results: drops entries without id, coerces passed to boolean", async () => {
    // entry 2 has no id (dropped), entry 1 has passed:1 (coerced to true)
    fs.writeFileSync(path.join(EVAL_DIR, "eval.log"), 'EVAL_RESULT:{"results":[{"id":"q1","passed":1},{"passed":true},{"id":"q3","passed":false}]}');
    await evalMain();

    expect(global.core.setFailed).not.toHaveBeenCalled();
    expect(global.core.setOutput).toHaveBeenCalledWith("eval_total", "2");
    expect(global.core.setOutput).toHaveBeenCalledWith("eval_passed", "1");
  });

  it("truncates long rationale to 500 chars in the results JSON", async () => {
    const longRationale = "x".repeat(600);
    fs.writeFileSync(path.join(EVAL_DIR, "eval.log"), `EVAL_RESULT:{"results":[{"id":"q1","passed":true,"rationale":"${longRationale}"}]}`);
    await evalMain();

    const summary = JSON.parse(fs.readFileSync(path.join(EVAL_DIR, "eval_results.json"), "utf-8"));
    expect(summary.results[0].rationale.length).toBeLessThanOrEqual(500);
  });
});
