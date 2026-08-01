import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import fs from "fs";

const OUTPUT_DIR = "/tmp/gh-aw";
const OUTPUT_PATH = "/tmp/gh-aw/evals.jsonl";

const mockCore = {
  info: vi.fn(),
  summary: {
    addRaw: vi.fn().mockReturnThis(),
    write: vi.fn().mockResolvedValue(),
  },
};

global.core = mockCore;

describe("render_evals_summary.cjs", () => {
  let module;

  beforeEach(async () => {
    vi.clearAllMocks();
    fs.mkdirSync(OUTPUT_DIR, { recursive: true });
    if (fs.existsSync(OUTPUT_PATH)) {
      fs.unlinkSync(OUTPUT_PATH);
    }
    module = await import("./render_evals_summary.cjs");
  });

  afterEach(() => {
    if (fs.existsSync(OUTPUT_PATH)) {
      fs.unlinkSync(OUTPUT_PATH);
    }
  });

  describe("readEvalsResults", () => {
    it("returns an empty array when the output file does not exist", () => {
      expect(module.readEvalsResults()).toEqual([]);
    });

    it("parses JSONL, skips malformed lines, and preserves rationale text", () => {
      fs.writeFileSync(
        OUTPUT_PATH,
        [
          JSON.stringify({ id: "q1", question: "First?", answer: " yes ", rationale: "The output clearly confirms the change.", model: "claude-sonnet-4.6", timestamp: "2026-07-15T00:00:00Z" }),
          "not-json",
          JSON.stringify({ id: "q2", question: "Second?", answer: "No", rationale: "The output does not mention the requested test.", model: "claude-sonnet-4.6" }),
          JSON.stringify({ id: "q3", question: "Third?" }),
        ].join("\n"),
        "utf8"
      );

      expect(module.readEvalsResults()).toEqual([
        {
          id: "q1",
          question: "First?",
          answer: "YES",
          rationale: "The output clearly confirms the change.",
          model: "claude-sonnet-4.6",
          timestamp: "2026-07-15T00:00:00Z",
        },
        {
          id: "q2",
          question: "Second?",
          answer: "NO",
          rationale: "The output does not mention the requested test.",
          model: "claude-sonnet-4.6",
          timestamp: "",
        },
        {
          id: "q3",
          question: "Third?",
          answer: "NO",
          rationale: "",
          model: "",
          timestamp: "",
        },
      ]);
    });
  });

  describe("buildEvalsBody", () => {
    it("renders tallies, escapes table cells, rationale text, and includes the model", () => {
      const markdown = module.buildEvalsBody([
        {
          id: "id|`1`\r",
          question: "Line 1\nLine 2 | `code` <tag> & more",
          answer: "YES",
          rationale: "A short | `reason` with <b>tag</b> & evidence.",
          model: "claude`-4.6` & more",
          timestamp: "",
        },
        {
          id: "id2",
          question: "Question 2",
          answer: "NO",
          rationale: "Missing evidence in the output.",
          model: "claude`-4.6`",
          timestamp: "",
        },
        {
          id: "id3",
          question: "Question 3",
          answer: "NO",
          rationale: "",
          model: "claude`-4.6`",
          timestamp: "",
        },
      ]);

      expect(markdown).toContain("| id\\|\\`1\\`  | Line 1 Line 2 \\| \\`code\\` &lt;tag&gt; &amp; more | ✅ YES | A short \\| \\`reason\\` with &lt;b&gt;tag&lt;/b&gt; &amp; evidence. |");
      expect(markdown).toContain("| id2 | Question 2 | ❌ NO | Missing evidence in the output. |");
      expect(markdown).toContain("| id3 | Question 3 | ❌ NO |  |");
      expect(markdown).toContain("**YES**: 1 | **NO**: 2");
      expect(markdown).toContain("**model**: claude\\`-4.6\\` &amp; more");
    });

    it("returns an empty string for empty results", () => {
      expect(module.buildEvalsBody([])).toBe("");
    });
  });

  describe("main", () => {
    it("logs and returns early when the output file is empty or malformed", async () => {
      fs.writeFileSync(OUTPUT_PATH, "not-json\n", "utf8");

      await module.main();

      expect(mockCore.info).toHaveBeenCalledWith(expect.stringContaining("No evals results found"));
      expect(mockCore.summary.addRaw).not.toHaveBeenCalled();
      expect(mockCore.summary.write).not.toHaveBeenCalled();
    });

    it("writes the evals details section to the step summary", async () => {
      fs.writeFileSync(
        OUTPUT_PATH,
        [
          JSON.stringify({ id: "builds", question: "Does it build?", answer: "YES", rationale: "The output reports a successful build.", model: "claude-sonnet-4.6" }),
          JSON.stringify({ id: "tests", question: "Do tests pass?", answer: "NO", rationale: "The output does not include a passing test result.", model: "claude-sonnet-4.6" }),
        ].join("\n") + "\n",
        "utf8"
      );

      await module.main();

      expect(mockCore.summary.addRaw).toHaveBeenCalledTimes(1);
      const summary = mockCore.summary.addRaw.mock.calls[0][0];
      expect(summary).toContain("<details>");
      expect(summary).toContain("<summary>BinEval Results</summary>");
      expect(summary).toContain("✅ YES");
      expect(summary).toContain("❌ NO");
      expect(summary).toContain("The output reports a successful build.");
      expect(summary).toContain("**model**: claude-sonnet-4.6");
      expect(mockCore.summary.write).toHaveBeenCalledTimes(1);
      expect(mockCore.info).toHaveBeenCalledWith(expect.stringContaining("2 result(s)"));
      expect(mockCore.info).toHaveBeenCalledWith(expect.stringContaining("written to step summary"));
    });
  });
});
