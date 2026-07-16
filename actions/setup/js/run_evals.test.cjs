import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import fs from "fs";
import { createRequire } from "module";

const EVALS_DIR = "/tmp/gh-aw/evals";
const EVALS_LOG_PATH = `${EVALS_DIR}/evals.log`;
const EVALS_OUTPUT_PATH = "/tmp/gh-aw/evals.jsonl";
const require = createRequire(import.meta.url);
const { parseMain, runScriptsMain, extractAssistantTextFromJsonlLog } = require("./run_evals.cjs");

const mockCore = {
  info: vi.fn(),
  warning: vi.fn(),
  setFailed: vi.fn(),
  exportVariable: vi.fn(),
  summary: {
    addDetails: vi.fn().mockReturnThis(),
    write: vi.fn().mockResolvedValue(),
  },
};

global.core = mockCore;

describe("run_evals.cjs", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    fs.mkdirSync(EVALS_DIR, { recursive: true });
    if (fs.existsSync(EVALS_LOG_PATH)) {
      fs.unlinkSync(EVALS_LOG_PATH);
    }
    if (fs.existsSync(EVALS_OUTPUT_PATH)) {
      fs.unlinkSync(EVALS_OUTPUT_PATH);
    }
  });

  afterEach(() => {
    vi.unstubAllEnvs();
    if (fs.existsSync(EVALS_LOG_PATH)) {
      fs.unlinkSync(EVALS_LOG_PATH);
    }
    if (fs.existsSync(EVALS_OUTPUT_PATH)) {
      fs.unlinkSync(EVALS_OUTPUT_PATH);
    }
  });

  it("stores the workflow run id when writing eval records", async () => {
    vi.stubEnv("GH_AW_EVALS_QUESTIONS", JSON.stringify([{ id: "labels-applied", question: "Did labels get applied?" }]));
    vi.stubEnv("GH_AW_EVALS_MODEL", "small");
    vi.stubEnv("GITHUB_RUN_ID", "123456789");
    fs.writeFileSync(EVALS_LOG_PATH, "labels-applied: YES\n", "utf8");

    await parseMain();

    const lines = fs.readFileSync(EVALS_OUTPUT_PATH, "utf8").trim().split("\n");
    expect(lines).toHaveLength(1);
    expect(JSON.parse(lines[0])).toEqual({
      id: "labels-applied",
      question: "Did labels get applied?",
      answer: "YES",
      model: "small",
      timestamp: expect.any(String),
      runid: "123456789",
    });
  });

  it('falls back to "unknown" when the workflow run id is absent', async () => {
    vi.stubEnv("GH_AW_EVALS_QUESTIONS", JSON.stringify([{ id: "labels-applied", question: "Did labels get applied?" }]));
    vi.stubEnv("GH_AW_EVALS_MODEL", "small");
    vi.stubEnv("GITHUB_RUN_ID", "");
    fs.writeFileSync(EVALS_LOG_PATH, "labels-applied: YES\n", "utf8");

    await parseMain();

    const [line] = fs.readFileSync(EVALS_OUTPUT_PATH, "utf8").trim().split("\n");
    expect(JSON.parse(line).runid).toBe("unknown");
  });

  it("parses answers from Pi v3 JSONL turn_end events (positional format)", async () => {
    vi.stubEnv(
      "GH_AW_EVALS_QUESTIONS",
      JSON.stringify([
        { id: "labels-applied", question: "Did labels get applied?" },
        { id: "report-created", question: "Was a summary discussion created?" },
      ])
    );
    vi.stubEnv("GH_AW_EVALS_MODEL", "small");
    vi.stubEnv("GITHUB_RUN_ID", "999");

    // Simulate the Pi v3 JSONL log where the model answers in positional form
    // inside a turn_end event. The \n in the JSON string is a real escape sequence
    // representing a newline in the model's response text.
    const turnEndEvent = JSON.stringify({
      type: "turn_end",
      message: {
        role: "assistant",
        content: [{ type: "text", text: "Q1: YES\nQ2: YES" }],
      },
    });
    fs.writeFileSync(EVALS_LOG_PATH, turnEndEvent + "\n", "utf8");

    await parseMain();

    const lines = fs.readFileSync(EVALS_OUTPUT_PATH, "utf8").trim().split("\n");
    expect(lines).toHaveLength(2);
    expect(JSON.parse(lines[0]).answer).toBe("YES");
    expect(JSON.parse(lines[1]).answer).toBe("YES");
  });

  it("parses answers from Pi v3 JSONL turn_end events (mixed YES/NO)", async () => {
    vi.stubEnv(
      "GH_AW_EVALS_QUESTIONS",
      JSON.stringify([
        { id: "labels-applied", question: "Did labels get applied?" },
        { id: "report-created", question: "Was a summary discussion created?" },
      ])
    );
    vi.stubEnv("GH_AW_EVALS_MODEL", "small");
    vi.stubEnv("GITHUB_RUN_ID", "999");

    const turnEndEvent = JSON.stringify({
      type: "turn_end",
      message: {
        role: "assistant",
        content: [{ type: "text", text: "Q1: YES\nQ2: NO" }],
      },
    });
    fs.writeFileSync(EVALS_LOG_PATH, turnEndEvent + "\n", "utf8");

    await parseMain();

    const lines = fs.readFileSync(EVALS_OUTPUT_PATH, "utf8").trim().split("\n");
    expect(lines).toHaveLength(2);
    expect(JSON.parse(lines[0]).answer).toBe("YES");
    expect(JSON.parse(lines[1]).answer).toBe("NO");
  });

  it("parses answers from Pi v3 JSONL turn_end events (id-based format)", async () => {
    vi.stubEnv("GH_AW_EVALS_QUESTIONS", JSON.stringify([{ id: "labels-applied", question: "Did labels get applied?" }]));
    vi.stubEnv("GH_AW_EVALS_MODEL", "small");
    vi.stubEnv("GITHUB_RUN_ID", "999");

    const turnEndEvent = JSON.stringify({
      type: "turn_end",
      message: {
        role: "assistant",
        content: [{ type: "text", text: "labels-applied: YES" }],
      },
    });
    fs.writeFileSync(EVALS_LOG_PATH, turnEndEvent + "\n", "utf8");

    await parseMain();

    const [line] = fs.readFileSync(EVALS_OUTPUT_PATH, "utf8").trim().split("\n");
    expect(JSON.parse(line).answer).toBe("YES");
  });

  describe("extractAssistantTextFromJsonlLog", () => {
    it("returns empty string for non-JSONL content", () => {
      expect(extractAssistantTextFromJsonlLog("Q1: YES\nQ2: NO\n")).toBe("");
    });

    it("extracts text from v3 turn_end events", () => {
      const log = JSON.stringify({
        type: "turn_end",
        message: { role: "assistant", content: [{ type: "text", text: "Q1: YES\nQ2: NO" }] },
      });
      expect(extractAssistantTextFromJsonlLog(log)).toBe("Q1: YES\nQ2: NO");
    });

    it("extracts text from v1 legacy assistant events", () => {
      const log = JSON.stringify({ type: "assistant", content: "Q1: YES" });
      expect(extractAssistantTextFromJsonlLog(log)).toBe("Q1: YES");
    });

    it("joins multiple assistant messages with newlines", () => {
      const lines = [JSON.stringify({ type: "assistant", content: "Q1: YES" }), JSON.stringify({ type: "assistant", content: "Q2: NO" })].join("\n");
      expect(extractAssistantTextFromJsonlLog(lines)).toBe("Q1: YES\nQ2: NO");
    });

    it("ignores non-assistant JSONL events", () => {
      const log = [JSON.stringify({ type: "turn_start" }), JSON.stringify({ type: "turn_end", message: { role: "assistant", content: [{ type: "text", text: "Q1: YES" }] } }), JSON.stringify({ type: "agent_end" })].join("\n");
      expect(extractAssistantTextFromJsonlLog(log)).toBe("Q1: YES");
    });

    it("handles timestamp-prefixed log lines", () => {
      const jsonPart = JSON.stringify({
        type: "turn_end",
        message: { role: "assistant", content: [{ type: "text", text: "Q1: YES\nQ2: NO" }] },
      });
      const log = `2026-07-16T07:21:45.2085595Z ${jsonPart}`;
      expect(extractAssistantTextFromJsonlLog(log)).toBe("Q1: YES\nQ2: NO");
    });
  });

  describe("runScriptsMain", () => {
    const mockExec = {
      exec: vi.fn(),
    };

    beforeEach(() => {
      global.exec = mockExec;
    });

    afterEach(() => {
      vi.unstubAllEnvs();
      if (fs.existsSync(EVALS_OUTPUT_PATH)) {
        fs.unlinkSync(EVALS_OUTPUT_PATH);
      }
    });

    it("writes YES result when script outputs YES", async () => {
      mockExec.exec.mockImplementation(async (_cmd, _args, opts) => {
        opts.listeners.stdout(Buffer.from("YES\n"));
        return 0;
      });
      vi.stubEnv("GH_AW_EVALS_SCRIPT_DEFS", JSON.stringify([{ id: "s1", run: "echo YES" }]));
      vi.stubEnv("GH_AW_EVALS_MODEL", "small");
      vi.stubEnv("GITHUB_RUN_ID", "42");

      await runScriptsMain();

      const lines = fs.readFileSync(EVALS_OUTPUT_PATH, "utf8").trim().split("\n");
      expect(lines).toHaveLength(1);
      const rec = JSON.parse(lines[0]);
      expect(rec.id).toBe("s1");
      expect(rec.answer).toBe("YES");
      expect(rec.run).toBe("echo YES");
      expect(rec.runid).toBe("42");
    });

    it("writes NO result when script outputs NO", async () => {
      mockExec.exec.mockImplementation(async (_cmd, _args, opts) => {
        opts.listeners.stdout(Buffer.from("NO\n"));
        return 0;
      });
      vi.stubEnv("GH_AW_EVALS_SCRIPT_DEFS", JSON.stringify([{ id: "s1", run: "echo NO" }]));
      vi.stubEnv("GH_AW_EVALS_MODEL", "small");
      vi.stubEnv("GITHUB_RUN_ID", "43");

      await runScriptsMain();

      const [line] = fs.readFileSync(EVALS_OUTPUT_PATH, "utf8").trim().split("\n");
      expect(JSON.parse(line).answer).toBe("NO");
    });

    it("writes UNKNOWN when script output is not YES or NO", async () => {
      mockExec.exec.mockImplementation(async (_cmd, _args, opts) => {
        opts.listeners.stdout(Buffer.from("maybe\n"));
        return 0;
      });
      vi.stubEnv("GH_AW_EVALS_SCRIPT_DEFS", JSON.stringify([{ id: "s1", run: "echo maybe" }]));
      vi.stubEnv("GH_AW_EVALS_MODEL", "small");
      vi.stubEnv("GITHUB_RUN_ID", "44");

      await runScriptsMain();

      const [line] = fs.readFileSync(EVALS_OUTPUT_PATH, "utf8").trim().split("\n");
      expect(JSON.parse(line).answer).toBe("UNKNOWN");
    });

    it("appends to existing evals.jsonl without overwriting LLM results", async () => {
      // Pre-existing LLM eval result
      fs.mkdirSync("/tmp/gh-aw", { recursive: true });
      fs.writeFileSync(EVALS_OUTPUT_PATH, JSON.stringify({ id: "q1", answer: "YES" }) + "\n");

      mockExec.exec.mockImplementation(async (_cmd, _args, opts) => {
        opts.listeners.stdout(Buffer.from("NO\n"));
        return 0;
      });
      vi.stubEnv("GH_AW_EVALS_SCRIPT_DEFS", JSON.stringify([{ id: "s1", run: "echo NO" }]));
      vi.stubEnv("GH_AW_EVALS_MODEL", "small");
      vi.stubEnv("GITHUB_RUN_ID", "45");

      await runScriptsMain();

      const lines = fs.readFileSync(EVALS_OUTPUT_PATH, "utf8").trim().split("\n");
      expect(lines).toHaveLength(2);
      expect(JSON.parse(lines[0]).id).toBe("q1");
      expect(JSON.parse(lines[1]).id).toBe("s1");
    });

    it("handles multiple script evals in order", async () => {
      let callCount = 0;
      mockExec.exec.mockImplementation(async (_cmd, _args, opts) => {
        callCount++;
        opts.listeners.stdout(Buffer.from(callCount === 1 ? "YES\n" : "NO\n"));
        return 0;
      });
      vi.stubEnv(
        "GH_AW_EVALS_SCRIPT_DEFS",
        JSON.stringify([
          { id: "s1", run: "echo YES" },
          { id: "s2", run: "echo NO" },
        ])
      );
      vi.stubEnv("GH_AW_EVALS_MODEL", "small");
      vi.stubEnv("GITHUB_RUN_ID", "46");

      await runScriptsMain();

      const lines = fs.readFileSync(EVALS_OUTPUT_PATH, "utf8").trim().split("\n");
      expect(lines).toHaveLength(2);
      expect(JSON.parse(lines[0]).answer).toBe("YES");
      expect(JSON.parse(lines[1]).answer).toBe("NO");
    });

    it("records UNKNOWN when exec throws", async () => {
      mockExec.exec.mockRejectedValue(new Error("command not found"));
      vi.stubEnv("GH_AW_EVALS_SCRIPT_DEFS", JSON.stringify([{ id: "s1", run: "nonexistent_cmd" }]));
      vi.stubEnv("GH_AW_EVALS_MODEL", "small");
      vi.stubEnv("GITHUB_RUN_ID", "47");

      await runScriptsMain();

      const [line] = fs.readFileSync(EVALS_OUTPUT_PATH, "utf8").trim().split("\n");
      expect(JSON.parse(line).answer).toBe("UNKNOWN");
    });

    it("fails when GH_AW_EVALS_SCRIPT_DEFS is not set", async () => {
      vi.stubEnv("GH_AW_EVALS_SCRIPT_DEFS", "");
      await runScriptsMain();
      expect(mockCore.setFailed).toHaveBeenCalledWith(expect.stringContaining("GH_AW_EVALS_SCRIPT_DEFS"));
    });
  });
});
