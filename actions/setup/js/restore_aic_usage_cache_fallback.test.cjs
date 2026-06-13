// @ts-check
import fs from "fs";
import os from "os";
import path from "path";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

let exports;

describe("restore_aic_usage_cache_fallback", () => {
  let tmpDir;
  let cacheFile;

  beforeEach(async () => {
    vi.resetModules();
    tmpDir = fs.mkdtempSync(path.join(os.tmpdir(), "restore-aic-cache-fallback-test-"));
    cacheFile = path.join(tmpDir, "agentic-workflow-usage-cache.jsonl");

    global.core = {
      info: vi.fn(),
      warning: vi.fn(),
      error: vi.fn(),
      setFailed: vi.fn(),
    };
    global.context = {
      repo: { owner: "test-owner", repo: "test-repo" },
      runId: 99999,
    };

    const mod = await import("./restore_aic_usage_cache_fallback.cjs");
    exports = mod.default || mod;
  });

  afterEach(() => {
    fs.rmSync(tmpDir, { recursive: true, force: true });
    delete global.core;
    delete global.context;
    delete global.github;
    delete process.env.GH_AW_GITHUB_TOKEN;
    delete process.env.GITHUB_TOKEN;
  });

  it("is a no-op when cache file already exists", async () => {
    const content = JSON.stringify({ run_id: 1, aic: 5.0, timestamp: new Date().toISOString() }) + "\n";
    fs.writeFileSync(cacheFile, content, "utf8");

    global.github = { rest: { actions: { getWorkflowRun: vi.fn() } } };

    await exports.mainWithPaths(cacheFile);

    expect(global.github.rest.actions.getWorkflowRun).not.toHaveBeenCalled();
    // File should be unchanged
    expect(fs.readFileSync(cacheFile, "utf8")).toBe(content);
  });

  it("is a no-op when no token is available", async () => {
    global.github = { rest: { actions: { getWorkflowRun: vi.fn() } } };

    await exports.mainWithPaths(cacheFile);

    expect(global.github.rest.actions.getWorkflowRun).not.toHaveBeenCalled();
    expect(fs.existsSync(cacheFile)).toBe(false);
  });

  it("skips gracefully when getWorkflowRun fails", async () => {
    process.env.GH_AW_GITHUB_TOKEN = "test-token";
    global.github = {
      rest: {
        actions: {
          getWorkflowRun: vi.fn().mockRejectedValue(new Error("API error")),
        },
      },
    };

    await expect(exports.mainWithPaths(cacheFile)).resolves.toBeUndefined();
    expect(fs.existsSync(cacheFile)).toBe(false);
  });

  it("skips gracefully when getWorkflowRun returns no workflow_id", async () => {
    process.env.GH_AW_GITHUB_TOKEN = "test-token";
    global.github = {
      rest: {
        actions: {
          getWorkflowRun: vi.fn().mockResolvedValue({ data: { workflow_id: null } }),
        },
      },
    };

    await exports.mainWithPaths(cacheFile);
    expect(fs.existsSync(cacheFile)).toBe(false);
  });

  it("downloads aic-usage-cache artifact from the most recent matching run", async () => {
    process.env.GH_AW_GITHUB_TOKEN = "test-token";

    const jsonlContent = JSON.stringify({ run_id: 100, aic: 50.0, timestamp: new Date().toISOString() }) + "\n";

    global.github = {
      rest: {
        actions: {
          getWorkflowRun: vi.fn().mockResolvedValue({ data: { workflow_id: 777 } }),
          listWorkflowRuns: vi.fn().mockResolvedValue({
            data: {
              workflow_runs: [
                { id: 99999 }, // current run – should be skipped
                { id: 11111 }, // prior run – has the artifact
              ],
            },
          }),
        },
      },
    };

    // Simulate the artifact client: listArtifacts and downloadArtifact
    const artifactDownloadDir = fs.mkdtempSync(path.join(os.tmpdir(), "artifact-download-"));
    const jsonlInArtifact = path.join(artifactDownloadDir, "agentic-workflow-usage-cache.jsonl");
    fs.writeFileSync(jsonlInArtifact, jsonlContent, "utf8");

    const mockClient = {
      listArtifacts: vi.fn().mockResolvedValue({
        artifacts: [{ id: 42, name: "aic-usage-cache", expired: false }],
      }),
      downloadArtifact: vi.fn().mockResolvedValue({ downloadPath: artifactDownloadDir }),
    };

    await exports.mainWithPaths(cacheFile, { createArtifactClient: () => mockClient });

    expect(fs.existsSync(cacheFile)).toBe(true);
    expect(fs.readFileSync(cacheFile, "utf8")).toBe(jsonlContent);

    fs.rmSync(artifactDownloadDir, { recursive: true, force: true });
  });

  it("skips expired artifacts and proceeds without cache when none valid", async () => {
    process.env.GH_AW_GITHUB_TOKEN = "test-token";

    global.github = {
      rest: {
        actions: {
          getWorkflowRun: vi.fn().mockResolvedValue({ data: { workflow_id: 888 } }),
          listWorkflowRuns: vi.fn().mockResolvedValue({
            data: {
              workflow_runs: [{ id: 22222 }],
            },
          }),
        },
      },
    };

    const mockClient = {
      listArtifacts: vi.fn().mockResolvedValue({
        artifacts: [{ id: 55, name: "aic-usage-cache", expired: true }],
      }),
      downloadArtifact: vi.fn(),
    };

    await exports.mainWithPaths(cacheFile, { createArtifactClient: () => mockClient });
    expect(fs.existsSync(cacheFile)).toBe(false);
  });

  it("skips runs whose listArtifacts call throws and continues to the next run", async () => {
    process.env.GH_AW_GITHUB_TOKEN = "test-token";

    const jsonlContent = JSON.stringify({ run_id: 300, aic: 9.0 }) + "\n";
    const artifactDownloadDir = fs.mkdtempSync(path.join(os.tmpdir(), "artifact-download2-"));
    fs.writeFileSync(path.join(artifactDownloadDir, "agentic-workflow-usage-cache.jsonl"), jsonlContent, "utf8");

    global.github = {
      rest: {
        actions: {
          getWorkflowRun: vi.fn().mockResolvedValue({ data: { workflow_id: 999 } }),
          listWorkflowRuns: vi.fn().mockResolvedValue({
            data: {
              workflow_runs: [
                { id: 33333 }, // throws
                { id: 44444 }, // succeeds
              ],
            },
          }),
        },
      },
    };

    let callCount = 0;
    const mockClient = {
      listArtifacts: vi.fn().mockImplementation(() => {
        callCount++;
        if (callCount === 1) return Promise.reject(new Error("network error"));
        return Promise.resolve({ artifacts: [{ id: 99, name: "aic-usage-cache", expired: false }] });
      }),
      downloadArtifact: vi.fn().mockResolvedValue({ downloadPath: artifactDownloadDir }),
    };

    await exports.mainWithPaths(cacheFile, { createArtifactClient: () => mockClient });

    expect(fs.existsSync(cacheFile)).toBe(true);
    expect(fs.readFileSync(cacheFile, "utf8")).toBe(jsonlContent);

    fs.rmSync(artifactDownloadDir, { recursive: true, force: true });
  });
});
