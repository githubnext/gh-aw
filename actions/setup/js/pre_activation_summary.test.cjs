// @ts-check
import { describe, it, expect, beforeEach, afterEach, vi } from "vitest";
import fs from "fs";
import os from "os";
import path from "path";

describe("pre_activation_summary.cjs", () => {
  let mockCore;
  let originalRunnerTemp;

  beforeEach(() => {
    mockCore = {
      info: vi.fn(),
      warning: vi.fn(),
      summary: {
        addRaw: vi.fn().mockReturnThis(),
        write: vi.fn().mockResolvedValue(undefined),
      },
    };
    global.core = mockCore;
    originalRunnerTemp = process.env.RUNNER_TEMP;
    vi.resetModules();
  });

  afterEach(() => {
    if (originalRunnerTemp !== undefined) {
      process.env.RUNNER_TEMP = originalRunnerTemp;
    } else {
      delete process.env.RUNNER_TEMP;
    }
    delete global.core;
    vi.clearAllMocks();
  });

  describe("writeDenialSummary", () => {
    it("uses the markdown template when template file exists", async () => {
      const tmpDir = fs.mkdtempSync(path.join(os.tmpdir(), "gh-aw-test-"));
      const promptsDir = path.join(tmpDir, "gh-aw", "prompts");
      fs.mkdirSync(promptsDir, { recursive: true });
      fs.writeFileSync(path.join(promptsDir, "pre_activation_skip.md"), "## Skipped\n\n> {reason}\n\n**Fix:** {remediation}\n", "utf8");

      process.env.RUNNER_TEMP = tmpDir;

      try {
        const { writeDenialSummary } = await import("./pre_activation_summary.cjs");
        await writeDenialSummary("Denied: insufficient perms", "Update frontmatter roles");

        expect(mockCore.summary.addRaw).toHaveBeenCalledWith("## Skipped\n\n> Denied: insufficient perms\n\n**Fix:** Update frontmatter roles\n");
        expect(mockCore.summary.write).toHaveBeenCalled();
      } finally {
        fs.rmSync(tmpDir, { recursive: true, force: true });
      }
    });

    it("falls back to hardcoded format when RUNNER_TEMP is not set", async () => {
      delete process.env.RUNNER_TEMP;

      const { writeDenialSummary } = await import("./pre_activation_summary.cjs");
      await writeDenialSummary("Bot not authorized", "Add bot to on.bots:");

      const rawCall = mockCore.summary.addRaw.mock.calls[0][0];
      expect(rawCall).toContain("Bot not authorized");
      expect(rawCall).toContain("Add bot to on.bots:");
      expect(mockCore.summary.write).toHaveBeenCalled();
    });

    it("falls back to hardcoded format when template file does not exist", async () => {
      const tmpDir = fs.mkdtempSync(path.join(os.tmpdir(), "gh-aw-test-"));
      process.env.RUNNER_TEMP = tmpDir;
      // No template file created

      try {
        const { writeDenialSummary } = await import("./pre_activation_summary.cjs");
        await writeDenialSummary("Stop time exceeded", "Update on.stop-after:");

        const rawCall = mockCore.summary.addRaw.mock.calls[0][0];
        expect(rawCall).toContain("Stop time exceeded");
        expect(rawCall).toContain("Update on.stop-after:");
        expect(mockCore.summary.write).toHaveBeenCalled();
      } finally {
        fs.rmSync(tmpDir, { recursive: true, force: true });
      }
    });
  });
});
