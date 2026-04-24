// @ts-check
/// <reference types="@actions/github-script" />

import { describe, it, expect, beforeEach, vi } from "vitest";
import { createRequire } from "module";

const req = createRequire(import.meta.url);
const { main, deterministicLabelColor } = req("./create_labels.cjs");

// ─── global mocks ────────────────────────────────────────────────────────────

const mockCore = {
  info: vi.fn(),
  warning: vi.fn(),
  error: vi.fn(),
  setFailed: vi.fn(),
};

const mockExec = {
  getExecOutput: vi.fn(),
};

const mockGithub = {
  paginate: vi.fn(),
  rest: {
    issues: {
      listLabelsForRepo: vi.fn(),
      createLabel: vi.fn(),
    },
  },
};

const mockContext = {
  repo: { owner: "test-owner", repo: "test-repo" },
};

global.core = mockCore;
global.exec = mockExec;
global.github = mockGithub;
global.context = mockContext;

// ─── deterministicLabelColor ─────────────────────────────────────────────────

describe("deterministicLabelColor", () => {
  it("returns a 6-character hex string", () => {
    const color = deterministicLabelColor("bug");
    expect(color).toMatch(/^[0-9a-f]{6}$/);
  });

  it("returns the same color for the same name (deterministic)", () => {
    expect(deterministicLabelColor("enhancement")).toBe(deterministicLabelColor("enhancement"));
  });

  it("returns different colors for different names", () => {
    expect(deterministicLabelColor("bug")).not.toBe(deterministicLabelColor("feature"));
  });

  it("all channels are in the pastel range 128–191 (0x80–0xbf)", () => {
    for (const name of ["bug", "feature", "docs", "test", "ci"]) {
      const hex = deterministicLabelColor(name);
      const r = parseInt(hex.slice(0, 2), 16);
      const g = parseInt(hex.slice(2, 4), 16);
      const b = parseInt(hex.slice(4, 6), 16);
      expect(r).toBeGreaterThanOrEqual(128);
      expect(r).toBeLessThanOrEqual(191);
      expect(g).toBeGreaterThanOrEqual(128);
      expect(g).toBeLessThanOrEqual(191);
      expect(b).toBeGreaterThanOrEqual(128);
      expect(b).toBeLessThanOrEqual(191);
    }
  });

  it("handles an empty string without throwing", () => {
    expect(() => deterministicLabelColor("")).not.toThrow();
    expect(deterministicLabelColor("")).toMatch(/^[0-9a-f]{6}$/);
  });
});

// ─── main ────────────────────────────────────────────────────────────────────

describe("main", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    process.env.GH_AW_CMD_PREFIX = "gh aw";
    delete process.env.GH_AW_TARGET_REPO_SLUG;

    // Default: compile succeeds and returns two labels
    mockExec.getExecOutput.mockResolvedValue({
      exitCode: 0,
      stdout: JSON.stringify([{ labels: ["bug", "enhancement"] }, { labels: ["bug", "docs"] }]),
      stderr: "",
    });

    // Default: repo has one existing label
    mockGithub.paginate.mockResolvedValue([{ name: "bug" }]);
    mockGithub.rest.issues.createLabel.mockResolvedValue({});
  });

  it("creates labels that are missing from the repository", async () => {
    await main();

    expect(mockGithub.rest.issues.createLabel).toHaveBeenCalledTimes(2);
    const names = mockGithub.rest.issues.createLabel.mock.calls.map(c => c[0].name);
    expect(names).toContain("enhancement");
    expect(names).toContain("docs");
  });

  it("skips labels that already exist", async () => {
    await main();

    const names = mockGithub.rest.issues.createLabel.mock.calls.map(c => c[0].name);
    expect(names).not.toContain("bug");
    expect(mockCore.info).toHaveBeenCalledWith(expect.stringContaining("already exists: bug"));
  });

  it("deduplicates labels from multiple workflows", async () => {
    await main();

    // 'bug' appears in both workflows but should only be counted once
    const allNames = mockGithub.rest.issues.createLabel.mock.calls.map(c => c[0].name);
    const unique = new Set(allNames);
    expect(allNames.length).toBe(unique.size);
  });

  it("uses a deterministic pastel color when creating labels", async () => {
    await main();

    for (const [args] of mockGithub.rest.issues.createLabel.mock.calls) {
      expect(args.color).toMatch(/^[0-9a-f]{6}$/);
    }
  });

  it("logs a summary of created and skipped labels", async () => {
    await main();

    expect(mockCore.info).toHaveBeenCalledWith(expect.stringContaining("created"));
    expect(mockCore.info).toHaveBeenCalledWith(expect.stringContaining("already existed"));
  });

  it("does nothing when no labels are found", async () => {
    mockExec.getExecOutput.mockResolvedValue({
      exitCode: 0,
      stdout: JSON.stringify([{ labels: [] }, {}]),
      stderr: "",
    });

    await main();

    expect(mockGithub.rest.issues.createLabel).not.toHaveBeenCalled();
    expect(mockCore.info).toHaveBeenCalledWith(expect.stringContaining("No labels found"));
  });

  it("ignores non-string or empty label values", async () => {
    mockExec.getExecOutput.mockResolvedValue({
      exitCode: 0,
      stdout: JSON.stringify([{ labels: ["valid", 42, null, "", "  ", "  trimmed  "] }]),
      stderr: "",
    });
    mockGithub.paginate.mockResolvedValue([]);

    await main();

    const names = mockGithub.rest.issues.createLabel.mock.calls.map(c => c[0].name);
    expect(names).toContain("valid");
    expect(names).toContain("trimmed");
    expect(names).not.toContain("");
    expect(names).not.toContain("  ");
  });

  it("calls setFailed when compile exits non-zero with no output", async () => {
    mockExec.getExecOutput.mockResolvedValue({ exitCode: 1, stdout: "", stderr: "compile error" });

    await main();

    expect(mockCore.setFailed).toHaveBeenCalledWith(expect.stringContaining("Failed to run compile"));
  });

  it("continues processing when compile exits non-zero but still produces JSON", async () => {
    mockExec.getExecOutput.mockResolvedValue({
      exitCode: 1,
      stdout: JSON.stringify([{ labels: ["bug"] }]),
      stderr: "some workflow had errors",
    });
    mockGithub.paginate.mockResolvedValue([]);

    await main();

    // Should proceed to create labels even though compile exited non-zero
    expect(mockCore.setFailed).not.toHaveBeenCalled();
  });

  it("calls setFailed when compile output is not valid JSON", async () => {
    mockExec.getExecOutput.mockResolvedValue({ exitCode: 0, stdout: "not json", stderr: "" });

    await main();

    expect(mockCore.setFailed).toHaveBeenCalledWith(expect.stringContaining("Failed to parse compile JSON output"));
  });

  it("treats a 422 error from createLabel as already-existing (race condition)", async () => {
    mockGithub.paginate.mockResolvedValue([]);
    const err = Object.assign(new Error("Unprocessable Entity"), { status: 422 });
    mockGithub.rest.issues.createLabel.mockRejectedValue(err);
    mockExec.getExecOutput.mockResolvedValue({
      exitCode: 0,
      stdout: JSON.stringify([{ labels: ["new-label"] }]),
      stderr: "",
    });

    await main();

    expect(mockCore.setFailed).not.toHaveBeenCalled();
    expect(mockCore.info).toHaveBeenCalledWith(expect.stringContaining("already exists (concurrent): new-label"));
  });

  it("emits a warning on non-422 createLabel errors but continues", async () => {
    mockGithub.paginate.mockResolvedValue([]);
    const err = Object.assign(new Error("Internal Server Error"), { status: 500 });
    mockGithub.rest.issues.createLabel.mockRejectedValueOnce(err);
    mockExec.getExecOutput.mockResolvedValue({
      exitCode: 0,
      stdout: JSON.stringify([{ labels: ["label-a", "label-b"] }]),
      stderr: "",
    });

    await main();

    expect(mockCore.warning).toHaveBeenCalledWith(expect.stringContaining("Failed to create label"));
    expect(mockCore.setFailed).not.toHaveBeenCalled();
  });
});
