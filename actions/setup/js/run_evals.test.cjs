import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import fs from "fs";

const EVALS_DIR = "/tmp/gh-aw/evals";
const EVALS_LOG_PATH = `${EVALS_DIR}/evals.log`;
const EVALS_OUTPUT_PATH = "/tmp/gh-aw/evals.jsonl";

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
  let module;

  beforeEach(async () => {
    vi.clearAllMocks();
    fs.mkdirSync(EVALS_DIR, { recursive: true });
    if (fs.existsSync(EVALS_LOG_PATH)) {
      fs.unlinkSync(EVALS_LOG_PATH);
    }
    if (fs.existsSync(EVALS_OUTPUT_PATH)) {
      fs.unlinkSync(EVALS_OUTPUT_PATH);
    }
    delete process.env.GH_AW_EVALS_QUESTIONS;
    delete process.env.GH_AW_EVALS_MODEL;
    delete process.env.GITHUB_RUN_ID;
    module = await import("./run_evals.cjs");
  });

  afterEach(() => {
    if (fs.existsSync(EVALS_LOG_PATH)) {
      fs.unlinkSync(EVALS_LOG_PATH);
    }
    if (fs.existsSync(EVALS_OUTPUT_PATH)) {
      fs.unlinkSync(EVALS_OUTPUT_PATH);
    }
  });

  it("stores the workflow run id when writing eval records", async () => {
    process.env.GH_AW_EVALS_QUESTIONS = JSON.stringify([{ id: "labels-applied", question: "Did labels get applied?" }]);
    process.env.GH_AW_EVALS_MODEL = "small";
    process.env.GITHUB_RUN_ID = "123456789";
    fs.writeFileSync(EVALS_LOG_PATH, "labels-applied: YES\n", "utf8");

    await module.parseMain();

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
});
