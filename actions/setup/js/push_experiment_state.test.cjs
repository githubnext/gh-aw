// @ts-check
import { describe, it, expect, beforeEach, afterEach, vi } from "vitest";
import os from "os";
import path from "path";
import fs from "fs";

// Globals required by push_experiment_state.cjs and its dependencies
const mockCore = {
  info: vi.fn(),
  warning: vi.fn(),
  error: vi.fn(),
  setFailed: vi.fn(),
  debug: vi.fn(),
};

const mockExec = {
  getExecOutput: vi.fn(),
};

const mockContext = {
  repo: { owner: "testowner", repo: "testrepo" },
};

global.core = mockCore;
global.exec = mockExec;
global.context = mockContext;
global.github = {};

const { main } = await import("./push_experiment_state.cjs");

describe("push_experiment_state", () => {
  let tmpDir;

  beforeEach(() => {
    vi.clearAllMocks();
    tmpDir = fs.mkdtempSync(path.join(os.tmpdir(), "push-exp-test-"));
    process.env.GITHUB_WORKSPACE = tmpDir;
    process.env.GITHUB_REPOSITORY = "testowner/testrepo";
  });

  afterEach(() => {
    fs.rmSync(tmpDir, { recursive: true, force: true });
    delete process.env.GH_AW_EXPERIMENT_STATE_DIR;
    delete process.env.GH_AW_EXPERIMENT_BRANCH;
    delete process.env.GH_TOKEN;
    delete process.env.GITHUB_WORKSPACE;
    delete process.env.GITHUB_REPOSITORY;
  });

  it("calls setFailed when GH_AW_EXPERIMENT_BRANCH is not set", async () => {
    process.env.GH_TOKEN = "ghp_test";
    delete process.env.GH_AW_EXPERIMENT_BRANCH;

    await main();

    expect(mockCore.setFailed).toHaveBeenCalledWith(expect.stringContaining("GH_AW_EXPERIMENT_BRANCH"));
  });

  it("calls setFailed when GH_TOKEN is not set", async () => {
    process.env.GH_AW_EXPERIMENT_BRANCH = "experiments/myworkflow";
    delete process.env.GH_TOKEN;
    delete process.env.GITHUB_TOKEN;

    await main();

    expect(mockCore.setFailed).toHaveBeenCalledWith(expect.stringContaining("GH_TOKEN"));
  });

  it("logs info and returns when no state files are present in state dir", async () => {
    process.env.GH_TOKEN = "ghp_test";
    process.env.GH_AW_EXPERIMENT_BRANCH = "experiments/myworkflow";
    process.env.GH_AW_EXPERIMENT_STATE_DIR = tmpDir;
    // tmpDir exists but is empty — no state.json or assignments.json

    await main();

    expect(mockCore.setFailed).not.toHaveBeenCalled();
    expect(mockCore.info).toHaveBeenCalledWith(expect.stringContaining("No experiment state files found"));
  });
});
