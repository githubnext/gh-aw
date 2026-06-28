import { describe, it, expect, beforeEach, afterEach, vi } from "vitest";
import fs from "fs";
import path from "path";

const TMP_ROOT = "/tmp/gh-aw";
const EVAL_DIR = path.join(TMP_ROOT, "eval");
const TEMPLATE_DIR = "/tmp/gh-aw-test-prompts";

describe("setup_eval", () => {
  beforeEach(() => {
    vi.resetModules();
    fs.rmSync(TMP_ROOT, { recursive: true, force: true });
    fs.rmSync(TEMPLATE_DIR, { recursive: true, force: true });
    fs.mkdirSync(TEMPLATE_DIR, { recursive: true });

    fs.writeFileSync(
      path.join(TEMPLATE_DIR, "eval.md"),
      `prompt={WORKFLOW_PROMPT_FILE}\noutput={AGENT_OUTPUT_FILE}\nquestions={EVAL_QUESTIONS}\n`
    );

    fs.mkdirSync(EVAL_DIR, { recursive: true });
    fs.writeFileSync(path.join(EVAL_DIR, "agent_output.json"), '{"ok":true}');

    process.env.GH_AW_PROMPTS_DIR = TEMPLATE_DIR;
    process.env.GH_AW_EVAL_SPEC = JSON.stringify([{ id: "builds", question: "Does the code compile?" }]);
    process.env.GH_AW_EVAL_WORK_DIR = EVAL_DIR;
  });

  afterEach(() => {
    fs.rmSync(TMP_ROOT, { recursive: true, force: true });
    fs.rmSync(TEMPLATE_DIR, { recursive: true, force: true });
    delete process.env.GH_AW_PROMPTS_DIR;
    delete process.env.GH_AW_EVAL_SPEC;
    delete process.env.GH_AW_EVAL_WORK_DIR;
  });

  function setupCoreMocks() {
    const summary = {
      addRaw: vi.fn().mockReturnThis(),
      write: vi.fn().mockResolvedValue(undefined),
    };
    global.core = {
      info: vi.fn(),
      warning: vi.fn(),
      error: vi.fn(),
      setFailed: vi.fn(),
      exportVariable: vi.fn(),
      summary,
    };
  }

  it("warns and returns early when no eval definitions are found", async () => {
    setupCoreMocks();
    process.env.GH_AW_EVAL_SPEC = "[]";

    const module = await import("./setup_eval.cjs");
    await module.main();

    expect(global.core.setFailed).not.toHaveBeenCalled();
    expect(global.core.warning).toHaveBeenCalledWith(expect.stringContaining("No eval definitions found"));
    expect(global.core.exportVariable).not.toHaveBeenCalled();
  });

  it("fails when eval prompt template is missing", async () => {
    setupCoreMocks();
    fs.rmSync(path.join(TEMPLATE_DIR, "eval.md"));

    const module = await import("./setup_eval.cjs");
    await module.main();

    expect(global.core.setFailed).toHaveBeenCalledWith(expect.stringContaining("Eval prompt template not found"));
  });

  it("writes prompt with all placeholders replaced when all files are present", async () => {
    setupCoreMocks();
    const promptDir = path.join(EVAL_DIR, "aw-prompts");
    fs.mkdirSync(promptDir, { recursive: true });
    fs.writeFileSync(path.join(promptDir, "prompt.txt"), "original workflow prompt");

    const module = await import("./setup_eval.cjs");
    await module.main();

    expect(global.core.setFailed).not.toHaveBeenCalled();
    expect(global.core.exportVariable).toHaveBeenCalledWith("GH_AW_PROMPT", "/tmp/gh-aw/aw-prompts/prompt.txt");

    const generatedPromptPath = "/tmp/gh-aw/aw-prompts/prompt.txt";
    expect(fs.existsSync(generatedPromptPath)).toBe(true);
    const content = fs.readFileSync(generatedPromptPath, "utf-8");
    expect(content).toContain("prompt=");
    expect(content).toContain("output=");
    expect(content).toContain("questions=");
    expect(content).toContain("builds");
    expect(content).toContain("Does the code compile?");
  });

  it("continues with reduced context when workflow prompt is missing", async () => {
    setupCoreMocks();

    const module = await import("./setup_eval.cjs");
    await module.main();

    expect(global.core.setFailed).not.toHaveBeenCalled();
    expect(global.core.warning).toHaveBeenCalledWith(expect.stringContaining("Missing workflow prompt"));
    expect(global.core.exportVariable).toHaveBeenCalledWith("GH_AW_PROMPT", "/tmp/gh-aw/aw-prompts/prompt.txt");

    const content = fs.readFileSync("/tmp/gh-aw/aw-prompts/prompt.txt", "utf-8");
    expect(content).toContain("unavailable");
  });

  it("continues with reduced context when workflow prompt is empty", async () => {
    setupCoreMocks();
    const promptDir = path.join(EVAL_DIR, "aw-prompts");
    fs.mkdirSync(promptDir, { recursive: true });
    fs.writeFileSync(path.join(promptDir, "prompt.txt"), "");

    const module = await import("./setup_eval.cjs");
    await module.main();

    expect(global.core.setFailed).not.toHaveBeenCalled();
    expect(global.core.warning).toHaveBeenCalledWith(expect.stringContaining("is empty"));
    expect(global.core.exportVariable).toHaveBeenCalledWith("GH_AW_PROMPT", "/tmp/gh-aw/aw-prompts/prompt.txt");
  });

  it("continues with reduced context when agent output is missing", async () => {
    setupCoreMocks();
    fs.rmSync(path.join(EVAL_DIR, "agent_output.json"), { force: true });

    const module = await import("./setup_eval.cjs");
    await module.main();

    expect(global.core.setFailed).not.toHaveBeenCalled();
    expect(global.core.warning).toHaveBeenCalledWith(expect.stringContaining("Missing agent output"));
    expect(global.core.exportVariable).toHaveBeenCalledWith("GH_AW_PROMPT", "/tmp/gh-aw/aw-prompts/prompt.txt");
  });

  it("embeds multiple eval questions in order", async () => {
    setupCoreMocks();
    process.env.GH_AW_EVAL_SPEC = JSON.stringify([
      { id: "builds", question: "Does the code compile?" },
      { id: "tests", question: "Are all tests passing?" },
    ]);

    const module = await import("./setup_eval.cjs");
    await module.main();

    const content = fs.readFileSync("/tmp/gh-aw/aw-prompts/prompt.txt", "utf-8");
    expect(content).toContain("**builds**: Does the code compile?");
    expect(content).toContain("**tests**: Are all tests passing?");
    const buildsPos = content.indexOf("builds");
    const testsPos = content.indexOf("tests");
    expect(buildsPos).toBeLessThan(testsPos);
  });

  it("writes step summary with details block", async () => {
    setupCoreMocks();

    const module = await import("./setup_eval.cjs");
    await module.main();

    expect(global.core.summary.addRaw).toHaveBeenCalledWith(expect.stringContaining("BinEval Prompt"));
    expect(global.core.summary.write).toHaveBeenCalled();
  });
});

describe("readEvalSpec (setup_eval)", () => {
  beforeEach(() => {
    vi.resetModules();
    delete process.env.GH_AW_EVAL_SPEC;
  });

  afterEach(() => {
    delete process.env.GH_AW_EVAL_SPEC;
  });

  it("returns empty array when env var is absent", async () => {
    const module = await import("./setup_eval.cjs");
    const result = module.readEvalSpec();
    expect(result).toEqual([]);
  });

  it("parses a valid spec", async () => {
    process.env.GH_AW_EVAL_SPEC = JSON.stringify([{ id: "q1", question: "Question one?" }]);
    const module = await import("./setup_eval.cjs");
    const result = module.readEvalSpec();
    expect(result).toHaveLength(1);
    expect(result[0]).toEqual({ id: "q1", question: "Question one?" });
  });

  it("throws on invalid JSON", async () => {
    process.env.GH_AW_EVAL_SPEC = "not-json";
    const module = await import("./setup_eval.cjs");
    expect(() => module.readEvalSpec()).toThrow(/Failed to parse GH_AW_EVAL_SPEC/);
  });
});

describe("formatEvalQuestions", () => {
  beforeEach(() => {
    vi.resetModules();
  });

  it("formats a single question", async () => {
    const module = await import("./setup_eval.cjs");
    const result = module.formatEvalQuestions([{ id: "builds", question: "Does it compile?" }]);
    expect(result).toBe("1. **builds**: Does it compile?");
  });

  it("formats multiple questions as a numbered list", async () => {
    const module = await import("./setup_eval.cjs");
    const result = module.formatEvalQuestions([
      { id: "builds", question: "Does it compile?" },
      { id: "tests", question: "Do tests pass?" },
    ]);
    expect(result).toBe("1. **builds**: Does it compile?\n2. **tests**: Do tests pass?");
  });

  it("returns empty string for empty array", async () => {
    const module = await import("./setup_eval.cjs");
    expect(module.formatEvalQuestions([])).toBe("");
  });
});
