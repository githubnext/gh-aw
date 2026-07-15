import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import fs from "fs";
import { createRequire } from "module";

const EVALS_DIR = "/tmp/gh-aw/evals";
const EVALS_LOG_PATH = `${EVALS_DIR}/evals.log`;
const EVALS_OUTPUT_PATH = "/tmp/gh-aw/evals.jsonl";
const require = createRequire(import.meta.url);
const { parseMain } = require("./run_evals.cjs");

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
});
