// @ts-check
import { describe, it, expect, beforeEach, afterEach, vi } from "vitest";
import fs from "fs";
import path from "path";
import { fileURLToPath } from "url";

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);

// Paths match what upload_artifact.cjs computes at runtime (uses /tmp/gh-aw/ base).
const STAGING_DIR = "/tmp/gh-aw/safeoutputs/upload-artifacts/";
const RESOLVER_FILE = "/tmp/gh-aw/artifact-resolver.json";

describe("upload_artifact.cjs", () => {
  let mockCore;
  let mockArtifactClient;
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
   * Build a config object.
   * @param {object} overrides
   */
  function buildConfig(overrides = {}) {
    return {
      "max-uploads": 3,
      "retention-days": 30,
      "max-size-bytes": 104857600,
      ...overrides,
    };
  }

  /**
   * Run the handler against a list of messages using the per-message handler pattern.
   * Injects global.__createArtifactClient so tests never hit the real REST API.
   * @param {object} config
   * @param {object[]} messages
   * @returns {Promise<object[]>}
   */
  async function runHandler(config, messages) {
    const scriptText = fs.readFileSync(path.join(__dirname, "upload_artifact.cjs"), "utf8");
    global.core = mockCore;
    global.__createArtifactClient = () => mockArtifactClient;
    let handlerFn;
    await eval(`(async () => { ${scriptText}; handlerFn = await main(config); })()`);
    const results = [];
    for (const msg of messages) {
      const result = await handlerFn(msg, {}, new Map());
      results.push(result);
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

    mockArtifactClient = {
      uploadArtifact: vi.fn().mockResolvedValue({ id: 42, size: 100 }),
    };

    originalEnv = { ...process.env };

    delete process.env.GH_AW_SAFE_OUTPUTS_STAGED;

    // Ensure staging dir exists and is clean
    if (fs.existsSync(STAGING_DIR)) {
      fs.rmSync(STAGING_DIR, { recursive: true });
    }
    fs.mkdirSync(STAGING_DIR, { recursive: true });

    // Clean resolver file
    if (fs.existsSync(RESOLVER_FILE)) {
      fs.unlinkSync(RESOLVER_FILE);
    }
  });

  afterEach(() => {
    process.env = originalEnv;
    delete global.__createArtifactClient;
  });

  describe("path-based upload", () => {
    it("uploads a single file using config retention days", async () => {
      writeStaging("report.json", '{"result": "ok"}');

      const results = await runHandler(buildConfig({ "retention-days": 14 }), [{ type: "upload_artifact", path: "report.json" }]);

      expect(mockCore.setFailed).not.toHaveBeenCalled();
      expect(results[0].success).toBe(true);
      expect(mockArtifactClient.uploadArtifact).toHaveBeenCalledOnce();
      const [name, files, rootDir, opts] = mockArtifactClient.uploadArtifact.mock.calls[0];
      expect(name).toBe("report.json");
      expect(files).toContain(path.join(STAGING_DIR, "report.json"));
      expect(rootDir).toBe(STAGING_DIR);
      expect(opts.retentionDays).toBe(14);
      expect(mockCore.setOutput).toHaveBeenCalledWith("upload_artifact_count", "1");
    });

    it("uses default retention of 30 when retention-days not in config", async () => {
      writeStaging("report.json");

      // Omit retention-days from config to test default
      await runHandler({ "max-uploads": 1, "max-size-bytes": 104857600 }, [{ type: "upload_artifact", path: "report.json" }]);

      const [, , , opts] = mockArtifactClient.uploadArtifact.mock.calls[0];
      expect(opts.retentionDays).toBe(30);
    });

    it("ignores retention_days in the message (agent cannot override)", async () => {
      writeStaging("report.json");

      // Even if the agent sends retention_days: 999, the config value (14) should be used.
      await runHandler(buildConfig({ "retention-days": 14 }), [{ type: "upload_artifact", path: "report.json", retention_days: 999 }]);

      const [, , , opts] = mockArtifactClient.uploadArtifact.mock.calls[0];
      expect(opts.retentionDays).toBe(14);
    });
  });

  describe("validation errors", () => {
    it("fails when both path and filters are present", async () => {
      writeStaging("report.json");

      await runHandler(buildConfig(), [{ type: "upload_artifact", path: "report.json", filters: { include: ["**/*.json"] } }]);

      expect(mockCore.setFailed).toHaveBeenCalledWith(expect.stringContaining("exactly one of 'path' or 'filters'"));
      expect(mockArtifactClient.uploadArtifact).not.toHaveBeenCalled();
    });

    it("fails when neither path nor filters are present", async () => {
      await runHandler(buildConfig(), [{ type: "upload_artifact" }]);
      expect(mockCore.setFailed).toHaveBeenCalledWith(expect.stringContaining("exactly one of 'path' or 'filters'"));
      expect(mockArtifactClient.uploadArtifact).not.toHaveBeenCalled();
    });

    it("fails when path traverses outside staging dir", async () => {
      await runHandler(buildConfig(), [{ type: "upload_artifact", path: "../etc/passwd" }]);
      expect(mockCore.setFailed).toHaveBeenCalledWith(expect.stringContaining("must not traverse outside staging directory"));
      expect(mockArtifactClient.uploadArtifact).not.toHaveBeenCalled();
    });

    it("fails when absolute path is provided", async () => {
      await runHandler(buildConfig(), [{ type: "upload_artifact", path: "/etc/passwd" }]);
      expect(mockCore.setFailed).toHaveBeenCalledWith(expect.stringContaining("must be relative"));
      expect(mockArtifactClient.uploadArtifact).not.toHaveBeenCalled();
    });

    it("fails when path does not exist in staging dir", async () => {
      await runHandler(buildConfig(), [{ type: "upload_artifact", path: "nonexistent.json" }]);
      expect(mockCore.setFailed).toHaveBeenCalledWith(expect.stringContaining("does not exist in staging directory"));
      expect(mockArtifactClient.uploadArtifact).not.toHaveBeenCalled();
    });

    it("fails when max-uploads is exceeded", async () => {
      writeStaging("a.json");
      writeStaging("b.json");

      const results = await runHandler(buildConfig({ "max-uploads": 1 }), [
        { type: "upload_artifact", path: "a.json" },
        { type: "upload_artifact", path: "b.json" },
      ]);

      expect(results[0].success).toBe(true);
      expect(results[1].success).toBe(false);
      expect(mockCore.setFailed).toHaveBeenCalledWith(expect.stringContaining("exceeded max-uploads policy"));
      expect(mockArtifactClient.uploadArtifact).toHaveBeenCalledOnce();
    });

    it("fails when skip-archive=true in config with multiple files", async () => {
      writeStaging("output/a.json");
      writeStaging("output/b.json");

      await runHandler(buildConfig({ "skip-archive": true }), [{ type: "upload_artifact", filters: { include: ["output/**"] } }]);

      expect(mockCore.setFailed).toHaveBeenCalledWith(expect.stringContaining("skip-archive requires exactly one selected file"));
      expect(mockArtifactClient.uploadArtifact).not.toHaveBeenCalled();
    });

    it("fails when upload client throws", async () => {
      writeStaging("report.json");
      mockArtifactClient.uploadArtifact.mockRejectedValue(new Error("network failure"));

      const results = await runHandler(buildConfig(), [{ type: "upload_artifact", path: "report.json" }]);

      expect(results[0].success).toBe(false);
      expect(mockCore.setFailed).toHaveBeenCalledWith(expect.stringContaining("network failure"));
    });
  });

  describe("skip-archive from config", () => {
    it("succeeds with skip-archive=true in config and a single file", async () => {
      writeStaging("app.bin", "binary data");

      const results = await runHandler(buildConfig({ "skip-archive": true }), [{ type: "upload_artifact", path: "app.bin" }]);

      expect(mockCore.setFailed).not.toHaveBeenCalled();
      expect(results[0].success).toBe(true);
      expect(mockArtifactClient.uploadArtifact).toHaveBeenCalledOnce();
    });

    it("ignores skip_archive in the message (agent cannot override)", async () => {
      writeStaging("app.bin", "binary data");

      // Config has skip-archive: false; agent sends skip_archive: true — config wins
      const results = await runHandler(buildConfig({ "skip-archive": false }), [{ type: "upload_artifact", path: "app.bin", skip_archive: true }]);

      expect(mockCore.setFailed).not.toHaveBeenCalled();
      expect(results[0].success).toBe(true);
      // No skip-archive error since config says false (so no single-file constraint check triggers)
      expect(mockArtifactClient.uploadArtifact).toHaveBeenCalledOnce();
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
      expect(mockArtifactClient.uploadArtifact).toHaveBeenCalledOnce();
      const [, files] = mockArtifactClient.uploadArtifact.mock.calls[0];
      expect(files).toHaveLength(2);
      expect(mockCore.setOutput).toHaveBeenCalledWith("slot_0_file_count", "2");
    });

    it("handles no-files with if-no-files=ignore", async () => {
      await runHandler(buildConfig({ "default-if-no-files": "ignore" }), [{ type: "upload_artifact", filters: { include: ["nonexistent/**"] } }]);

      expect(mockCore.setFailed).not.toHaveBeenCalled();
      expect(mockArtifactClient.uploadArtifact).not.toHaveBeenCalled();
    });

    it("fails when no files match and if-no-files=error (default)", async () => {
      await runHandler(buildConfig(), [{ type: "upload_artifact", filters: { include: ["nonexistent/**"] } }]);

      expect(mockCore.setFailed).toHaveBeenCalledWith(expect.stringContaining("no files matched"));
      expect(mockArtifactClient.uploadArtifact).not.toHaveBeenCalled();
    });
  });

  describe("allowed-paths policy", () => {
    it("filters out files not in allowed-paths", async () => {
      writeStaging("dist/app.js");
      writeStaging("secret.env");

      await runHandler(buildConfig({ "allowed-paths": ["dist/**"] }), [{ type: "upload_artifact", filters: { include: ["**"] } }]);

      expect(mockCore.setFailed).not.toHaveBeenCalled();
      const [, files] = mockArtifactClient.uploadArtifact.mock.calls[0];
      expect(files).toHaveLength(1);
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
    it("skips upload client call in staged mode (env var)", async () => {
      process.env.GH_AW_SAFE_OUTPUTS_STAGED = "true";
      writeStaging("report.json");

      const results = await runHandler(buildConfig(), [{ type: "upload_artifact", path: "report.json" }]);

      expect(mockCore.setFailed).not.toHaveBeenCalled();
      expect(results[0].success).toBe(true);
      expect(mockArtifactClient.uploadArtifact).not.toHaveBeenCalled();
      expect(mockCore.setOutput).toHaveBeenCalledWith("slot_0_tmp_id", expect.stringMatching(/^aw_[A-Za-z0-9]{8}$/));
    });

    it("skips upload client call when staged=true in config", async () => {
      writeStaging("report.json");

      const results = await runHandler(buildConfig({ staged: true }), [{ type: "upload_artifact", path: "report.json" }]);

      expect(mockCore.setFailed).not.toHaveBeenCalled();
      expect(results[0].success).toBe(true);
      expect(mockArtifactClient.uploadArtifact).not.toHaveBeenCalled();
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
      expect(keys[0]).toMatch(/^aw_[A-Za-z0-9]{8}$/);
    });
  });
});
