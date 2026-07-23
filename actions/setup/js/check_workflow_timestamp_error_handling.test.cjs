import { describe, it, expect, beforeEach, afterEach, vi } from "vitest";
import fs from "fs";
import os from "os";
import path from "path";

const mockCore = {
  debug: vi.fn(),
  info: vi.fn(),
  notice: vi.fn(),
  warning: vi.fn(),
  error: vi.fn(),
  setFailed: vi.fn(),
  setOutput: vi.fn(),
  exportVariable: vi.fn(),
  summary: {
    addRaw: vi.fn().mockReturnThis(),
    write: vi.fn().mockResolvedValue(),
  },
};

global.core = mockCore;

const { main } = await import("./check_workflow_timestamp.cjs");

describe("check_workflow_timestamp error handling", () => {
  let tmpDir;
  let workflowsDir;

  beforeEach(() => {
    vi.clearAllMocks();
    tmpDir = fs.mkdtempSync(path.join(os.tmpdir(), "workflow-stat-test-"));
    workflowsDir = path.join(tmpDir, ".github", "workflows");
    fs.mkdirSync(workflowsDir, { recursive: true });
    process.env.GITHUB_WORKSPACE = tmpDir;
    process.env.GH_AW_WORKFLOW_FILE = "test.lock.yml";
  });

  afterEach(() => {
    vi.restoreAllMocks();
    fs.rmSync(tmpDir, { recursive: true, force: true });
    delete process.env.GITHUB_WORKSPACE;
    delete process.env.GH_AW_WORKFLOW_FILE;
  });

  it("reports the workflow source path when source stat inspection fails", async () => {
    const workflowFile = path.join(workflowsDir, "test.md");
    const lockFile = path.join(workflowsDir, "test.lock.yml");
    fs.writeFileSync(workflowFile, "# Workflow content");
    fs.writeFileSync(lockFile, "# Lock content");

    vi.spyOn(fs, "statSync").mockImplementation(targetPath => {
      if (targetPath === workflowFile) {
        throw new Error("source unreadable");
      }
      return {
        mtime: new Date(),
        isFile: () => true,
      };
    });

    await main();

    expect(mockCore.setFailed).toHaveBeenCalledWith(`Failed to inspect workflow source ${workflowFile}: source unreadable`);
  });

  it("reports the lock file path when lock stat inspection fails", async () => {
    const workflowFile = path.join(workflowsDir, "test.md");
    const lockFile = path.join(workflowsDir, "test.lock.yml");
    fs.writeFileSync(workflowFile, "# Workflow content");
    fs.writeFileSync(lockFile, "# Lock content");

    vi.spyOn(fs, "statSync").mockImplementation(targetPath => {
      if (targetPath === lockFile) {
        throw new Error("lock unreadable");
      }
      return {
        mtime: new Date(),
        isFile: () => true,
      };
    });

    await main();

    expect(mockCore.setFailed).toHaveBeenCalledWith(`Failed to inspect lock file ${lockFile}: lock unreadable`);
  });
});
