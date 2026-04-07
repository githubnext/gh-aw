// @ts-check
import { describe, it, expect, beforeEach, afterEach, vi } from "vitest";
import fs from "fs";
import path from "path";
import os from "os";
import { fileURLToPath } from "url";

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);

// Use RUNNER_TEMP as the base so paths match what upload_artifact.cjs computes at runtime.
const RUNNER_TEMP = "/tmp";
const STAGING_DIR = `${RUNNER_TEMP}/gh-aw/safeoutputs/upload-artifacts/`;
const SLOT_BASE_DIR = `${RUNNER_TEMP}/gh-aw/upload-artifacts/`;
const RESOLVER_FILE = `${RUNNER_TEMP}/gh-aw/artifact-resolver.json`;

describe("upload_artifact.cjs", () => {
  let mockCore;
  let originalEnv;

  /**
   * @param {string} relPath
   * @param {string} content
   */
  function writeStaging(relPath, content = "test content") {
    const fullPath = path.join(STAGING_DIR, relPath);
    fs.mkdirSync(path.dirname(fullPath), { recursive: true });
    fs.writeFileSync(fullPath, content);
  }

  /**
   * Build a config object (replaces ENV vars in the old standalone approach).
   * @param {object} overrides
   */
  function buildConfig(overrides = {}) {
    return {
      "max-uploads": 3,
      "default-retention-days": 7,
      "max-retention-days": 30,
      "max-size-bytes": 104857600,
      ...overrides,
    };
  }

  /**
   * Run the handler against a list of messages using the new per-message pattern.
   * Simulates what the handler manager does.
   * @param {object} config
   * @param {object[]} messages
   * @returns {Promise<object[]>} results from each message handler call
   */
  async function runHandler(config, messages) {
    const scriptText = fs.readFileSync(path.join(__dirname, "upload_artifact.cjs"), "utf8");
    global.core = mockCore;
    let handlerFn;
    await eval(`(async () => { ${scriptText}; handlerFn = await main(config); })()`);
    const results = [];
    for (const msg of messages) {
      const result = await handlerFn(msg, {}, new Map());
      results.push(result);
      // Simulate handler manager calling setFailed on failure
      if (result && result.success === false && !result.skipped) {
        mockCore.setFailed(result.error);
      }
    }
    return results;
  }

  beforeEach(() => {
    vi.clearAllMocks();

    mockCore = {
      info: vi.fn(),
      warning: vi.fn(),
      error: vi.fn(),
      setOutput: vi.fn(),
      setFailed: vi.fn(),
      summary: {
        addHeading: vi.fn().mockReturnThis(),
        addRaw: vi.fn().mockReturnThis(),
        write: vi.fn().mockResolvedValue(undefined),
      },
    };

    originalEnv = { ...process.env };

    // Set RUNNER_TEMP so the script resolves paths to the same directories as the test helpers.
    process.env.RUNNER_TEMP = RUNNER_TEMP;
    delete process.env.GH_AW_SAFE_OUTPUTS_STAGED;

    // Ensure staging dir exists and is clean
    if (fs.existsSync(STAGING_DIR)) {
      fs.rmSync(STAGING_DIR, { recursive: true });
    }
    fs.mkdirSync(STAGING_DIR, { recursive: true });

    // Clean slot dir
    if (fs.existsSync(SLOT_BASE_DIR)) {
      fs.rmSync(SLOT_BASE_DIR, { recursive: true });
    }

    // Clean resolver file
    if (fs.existsSync(RESOLVER_FILE)) {
      fs.unlinkSync(RESOLVER_FILE);
    }
  });

  afterEach(() => {
    process.env = originalEnv;
  });

  describe("path-based upload", () => {
    it("stages a single file and sets slot outputs", async () => {
      writeStaging("report.json", '{"result": "ok"}');

      await runHandler(buildConfig(), [{ type: "upload_artifact", path: "report.json", retention_days: 14 }]);

      expect(mockCore.setFailed).not.toHaveBeenCalled();
      expect(mockCore.setOutput).toHaveBeenCalledWith("slot_0_enabled", "true");
      expect(mockCore.setOutput).toHaveBeenCalledWith("slot_0_retention_days", "14");
      expect(mockCore.setOutput).toHaveBeenCalledWith("upload_artifact_count", "1");

      // Verify the file was staged into slot_0.
      const slotFile = path.join(SLOT_BASE_DIR, "slot_0", "report.json");
      expect(fs.existsSync(slotFile)).toBe(true);
    });

    it("clamps retention days to max-retention-days", async () => {
      writeStaging("report.json");

      await runHandler(buildConfig(), [{ type: "upload_artifact", path: "report.json", retention_days: 999 }]);

      expect(mockCore.setFailed).not.toHaveBeenCalled();
      expect(mockCore.setOutput).toHaveBeenCalledWith("slot_0_retention_days", "30");
    });

    it("uses default retention when retention_days is absent", async () => {
      writeStaging("report.json");

      await runHandler(buildConfig(), [{ type: "upload_artifact", path: "report.json" }]);

      expect(mockCore.setOutput).toHaveBeenCalledWith("slot_0_retention_days", "7");
    });
  });

  describe("validation errors", () => {
    it("fails when both path and filters are present", async () => {
      writeStaging("report.json");

      await runHandler(buildConfig(), [{ type: "upload_artifact", path: "report.json", filters: { include: ["**/*.json"] } }]);

      expect(mockCore.setFailed).toHaveBeenCalledWith(expect.stringContaining("exactly one of 'path' or 'filters'"));
    });

    it("fails when neither path nor filters are present", async () => {
      await runHandler(buildConfig(), [{ type: "upload_artifact", retention_days: 7 }]);
      expect(mockCore.setFailed).toHaveBeenCalledWith(expect.stringContaining("exactly one of 'path' or 'filters'"));
    });

    it("fails when path traverses outside staging dir", async () => {
      await runHandler(buildConfig(), [{ type: "upload_artifact", path: "../etc/passwd" }]);
      expect(mockCore.setFailed).toHaveBeenCalledWith(expect.stringContaining("must not traverse outside staging directory"));
    });

    it("fails when absolute path is provided", async () => {
      await runHandler(buildConfig(), [{ type: "upload_artifact", path: "/etc/passwd" }]);
      expect(mockCore.setFailed).toHaveBeenCalledWith(expect.stringContaining("must be relative"));
    });

    it("fails when path does not exist in staging dir", async () => {
      await runHandler(buildConfig(), [{ type: "upload_artifact", path: "nonexistent.json" }]);
      expect(mockCore.setFailed).toHaveBeenCalledWith(expect.stringContaining("does not exist in staging directory"));
    });

    it("fails when max-uploads is exceeded", async () => {
      writeStaging("a.json");
      writeStaging("b.json");

      await runHandler(buildConfig({ "max-uploads": 1 }), [
        { type: "upload_artifact", path: "a.json" },
        { type: "upload_artifact", path: "b.json" },
      ]);

      expect(mockCore.setFailed).toHaveBeenCalledWith(expect.stringContaining("exceeded max-uploads policy"));
    });

    it("fails when skip_archive is requested but not allowed", async () => {
      writeStaging("app.bin");

      await runHandler(buildConfig(), [{ type: "upload_artifact", path: "app.bin", skip_archive: true }]);

      expect(mockCore.setFailed).toHaveBeenCalledWith(expect.stringContaining("skip_archive=true is not permitted"));
    });

    it("fails when skip_archive=true with multiple files", async () => {
      writeStaging("output/a.json");
      writeStaging("output/b.json");

      await runHandler(buildConfig({ "allow-skip-archive": true }), [{ type: "upload_artifact", filters: { include: ["output/**"] }, skip_archive: true }]);

      expect(mockCore.setFailed).toHaveBeenCalledWith(expect.stringContaining("skip_archive=true requires exactly one selected file"));
    });
  });

  describe("skip_archive allowed", () => {
    it("succeeds with skip_archive=true and a single file", async () => {
      writeStaging("app.bin", "binary data");

      await runHandler(buildConfig({ "allow-skip-archive": true }), [{ type: "upload_artifact", path: "app.bin", skip_archive: true }]);

      expect(mockCore.setFailed).not.toHaveBeenCalled();
      expect(mockCore.setOutput).toHaveBeenCalledWith("slot_0_enabled", "true");
    });
  });

  describe("filter-based upload", () => {
    it("selects files matching include pattern", async () => {
      writeStaging("reports/daily/summary.json", "{}");
      writeStaging("reports/weekly/summary.json", "{}");
      writeStaging("reports/private/secret.json", "{}");

      await runHandler(buildConfig(), [
        {
          type: "upload_artifact",
          filters: { include: ["reports/**/*.json"], exclude: ["reports/private/**"] },
        },
      ]);

      expect(mockCore.setFailed).not.toHaveBeenCalled();
      expect(mockCore.setOutput).toHaveBeenCalledWith("slot_0_enabled", "true");
      expect(mockCore.setOutput).toHaveBeenCalledWith("slot_0_file_count", "2");
    });

    it("handles no-files with if-no-files=ignore", async () => {
      await runHandler(buildConfig({ "default-if-no-files": "ignore" }), [{ type: "upload_artifact", filters: { include: ["nonexistent/**"] } }]);

      expect(mockCore.setFailed).not.toHaveBeenCalled();
      // No slot output set since skipped
      expect(mockCore.setOutput).not.toHaveBeenCalledWith("slot_0_enabled", "true");
    });

    it("fails when no files match and if-no-files=error (default)", async () => {
      await runHandler(buildConfig(), [{ type: "upload_artifact", filters: { include: ["nonexistent/**"] } }]);

      expect(mockCore.setFailed).toHaveBeenCalledWith(expect.stringContaining("no files matched"));
    });
  });

  describe("allowed-paths policy", () => {
    it("filters out files not in allowed-paths", async () => {
      writeStaging("dist/app.js");
      writeStaging("secret.env");

      await runHandler(buildConfig({ "allowed-paths": ["dist/**"] }), [{ type: "upload_artifact", filters: { include: ["**"] } }]);

      expect(mockCore.setFailed).not.toHaveBeenCalled();
      expect(mockCore.setOutput).toHaveBeenCalledWith("slot_0_file_count", "1");
    });
  });

  describe("filters-include / filters-exclude from config", () => {
    it("uses config filters-include as default when request has no filters", async () => {
      writeStaging("dist/app.js");
      writeStaging("secret.env");

      await runHandler(buildConfig({ "filters-include": ["dist/**"] }), [{ type: "upload_artifact", filters: {} }]);

      expect(mockCore.setFailed).not.toHaveBeenCalled();
      expect(mockCore.setOutput).toHaveBeenCalledWith("slot_0_file_count", "1");
    });
  });

  describe("staged mode", () => {
    it("skips file staging but sets outputs in staged mode", async () => {
      process.env.GH_AW_SAFE_OUTPUTS_STAGED = "true";
      writeStaging("report.json");

      await runHandler(buildConfig(), [{ type: "upload_artifact", path: "report.json" }]);

      expect(mockCore.setFailed).not.toHaveBeenCalled();
      expect(mockCore.setOutput).toHaveBeenCalledWith("slot_0_enabled", "true");

      // In staged mode, files are NOT copied to the slot directory.
      const slotFile = path.join(SLOT_BASE_DIR, "slot_0", "report.json");
      expect(fs.existsSync(slotFile)).toBe(false);
    });

    it("skips file staging when staged=true in config", async () => {
      writeStaging("report.json");

      await runHandler(buildConfig({ staged: true }), [{ type: "upload_artifact", path: "report.json" }]);

      expect(mockCore.setFailed).not.toHaveBeenCalled();
      expect(mockCore.setOutput).toHaveBeenCalledWith("slot_0_enabled", "true");

      const slotFile = path.join(SLOT_BASE_DIR, "slot_0", "report.json");
      expect(fs.existsSync(slotFile)).toBe(false);
    });
  });

  describe("resolver file", () => {
    it("writes a resolver mapping with temporary IDs", async () => {
      writeStaging("report.json");

      await runHandler(buildConfig(), [{ type: "upload_artifact", path: "report.json" }]);

      expect(fs.existsSync(RESOLVER_FILE)).toBe(true);
      const resolver = JSON.parse(fs.readFileSync(RESOLVER_FILE, "utf8"));
      const keys = Object.keys(resolver);
      expect(keys.length).toBe(1);
      expect(keys[0]).toMatch(/^tmp_artifact_[A-Z0-9]{26}$/);
    });
  });
});
