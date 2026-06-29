import { describe, expect, it, vi } from "vitest";

import { createGhAwRunnerWithStatus } from "../src/dashboard-cli.js";

describe("dashboard cli runner", () => {
  it("detects gh aw version from the extension and sets CI=1", async () => {
    const execFileFn = vi.fn((bin, args, options, callback) => {
      expect(bin).toBe("gh");
      expect(args).toEqual(["aw", "version"]);
      expect(options.env.CI).toBe("1");
      callback(null, "", "gh aw version v1.2.3\n");
    });

    const runGhAw = createGhAwRunnerWithStatus({
      getWorkspacePath: () => "/workspace",
      accessFn: vi.fn(async () => {
        throw new Error("missing");
      }),
      execFileFn,
      env: { PATH: "/bin" },
    });

    await expect(runGhAw.getStatus()).resolves.toMatchObject({
      available: true,
      source: "gh-extension",
      version: "v1.2.3",
      command: "gh aw version",
    });
  });

  it("returns install instructions when the gh aw extension is missing", async () => {
    const execFileFn = vi.fn((bin, args, options, callback) => {
      callback(Object.assign(new Error("missing"), { code: 1 }), "", "extension not found: aw");
    });

    const runGhAw = createGhAwRunnerWithStatus({
      getWorkspacePath: () => "/workspace",
      accessFn: vi.fn(async () => {
        throw new Error("missing");
      }),
      execFileFn,
    });

    await expect(runGhAw.getStatus()).resolves.toMatchObject({
      available: false,
      source: "missing",
      command: "gh aw version",
      installCommand: "gh extension install github/gh-aw",
    });
  });

  it("returns gh install instructions when gh itself is not found", async () => {
    const execFileFn = vi.fn((bin, args, options, callback) => {
      callback(Object.assign(new Error("spawn gh ENOENT"), { code: "ENOENT", syscall: "spawn", path: "gh" }), "", "");
    });

    const runGhAw = createGhAwRunnerWithStatus({
      getWorkspacePath: () => "/workspace",
      accessFn: vi.fn(async () => {
        throw new Error("missing");
      }),
      execFileFn,
    });

    await expect(runGhAw.getStatus()).resolves.toMatchObject({
      available: false,
      source: "gh-not-found",
      command: "gh aw version",
      message: "Install the GitHub CLI to use this dashboard.",
      installUrl: "https://cli.github.com",
      installCommand: "gh extension install github/gh-aw",
    });
  });
});
