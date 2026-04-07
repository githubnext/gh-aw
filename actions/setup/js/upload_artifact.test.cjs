// @ts-check
import { describe, it, expect, beforeEach, afterEach, vi } from "vitest";
import fs from "fs";
import path from "path";
import os from "os";
import { fileURLToPath } from "url";

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);

const STAGING_DIR = "/tmp/gh-aw/safeoutputs/upload-artifacts/";
const SLOT_BASE_DIR = "/tmp/gh-aw/upload-artifacts/";
const RESOLVER_FILE = "/tmp/gh-aw/artifact-resolver.json";

describe("upload_artifact.cjs", () => {
  let mockCore;
  let agentOutputPath;
  let originalEnv;

  /**
   * @param {object} data
   */
  function writeAgentOutput(data) {
    agentOutputPath = path.join(os.tmpdir(), `test_upload_artifact_${Date.now()}_${Math.random().toString(36).slice(2)}.json`);
    fs.writeFileSync(agentOutputPath, JSON.stringify(data));
    process.env.GH_AW_AGENT_OUTPUT = agentOutputPath;
  }

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
   * @returns {Promise<void>}
   */
  async function runMain() {
    const scriptText = fs.readFileSync(path.join(__dirname, "upload_artifact.cjs"), "utf8");
    global.core = mockCore;
    await eval(`(async () => { ${scriptText}; await main(); })()`);
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

    // Set reasonable defaults
    process.env.GH_AW_ARTIFACT_MAX_UPLOADS = "3";
    process.env.GH_AW_ARTIFACT_DEFAULT_RETENTION_DAYS = "7";
    process.env.GH_AW_ARTIFACT_MAX_RETENTION_DAYS = "30";
    process.env.GH_AW_ARTIFACT_MAX_SIZE_BYTES = "104857600";
    delete process.env.GH_AW_ARTIFACT_ALLOWED_PATHS;
    delete process.env.GH_AW_ARTIFACT_ALLOW_SKIP_ARCHIVE;
    delete process.env.GH_AW_ARTIFACT_DEFAULT_SKIP_ARCHIVE;
    delete process.env.GH_AW_ARTIFACT_DEFAULT_IF_NO_FILES;
    delete process.env.GH_AW_ARTIFACT_FILTERS_INCLUDE;
    delete process.env.GH_AW_ARTIFACT_FILTERS_EXCLUDE;
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
    // Restore env
    process.env = originalEnv;

    if (agentOutputPath && fs.existsSync(agentOutputPath)) {
      fs.unlinkSync(agentOutputPath);
    }
  });

  describe("no agent output", () => {
    it("sets artifact_count to 0 when no agent output is present", async () => {
      delete process.env.GH_AW_AGENT_OUTPUT;
      await runMain();
      expect(mockCore.setOutput).toHaveBeenCalledWith("artifact_count", "0");
      expect(mockCore.setFailed).not.toHaveBeenCalled();
    });
  });

  describe("no upload_artifact records", () => {
    it("sets artifact_count to 0 when output has no upload_artifact items", async () => {
      writeAgentOutput({ items: [{ type: "create_issue", title: "test" }] });
      await runMain();
      expect(mockCore.setOutput).toHaveBeenCalledWith("artifact_count", "0");
      expect(mockCore.setFailed).not.toHaveBeenCalled();
    });
  });

  describe("path-based upload", () => {
    it("stages a single file and sets slot outputs", async () => {
      writeStaging("report.json", '{"result": "ok"}');
      writeAgentOutput({
        items: [{ type: "upload_artifact", path: "report.json", retention_days: 14 }],
      });

      await runMain();

      expect(mockCore.setFailed).not.toHaveBeenCalled();
      expect(mockCore.setOutput).toHaveBeenCalledWith("slot_0_enabled", "true");
      expect(mockCore.setOutput).toHaveBeenCalledWith("slot_0_retention_days", "14");
      expect(mockCore.setOutput).toHaveBeenCalledWith("artifact_count", "1");

      // Verify the file was staged into slot_0.
      const slotFile = path.join(SLOT_BASE_DIR, "slot_0", "report.json");
      expect(fs.existsSync(slotFile)).toBe(true);
    });

    it("clamps retention days to max-retention-days", async () => {
      writeStaging("report.json");
      writeAgentOutput({
        items: [{ type: "upload_artifact", path: "report.json", retention_days: 999 }],
      });

      await runMain();

      expect(mockCore.setFailed).not.toHaveBeenCalled();
      expect(mockCore.setOutput).toHaveBeenCalledWith("slot_0_retention_days", "30");
    });

    it("uses default retention when retention_days is absent", async () => {
      writeStaging("report.json");
      writeAgentOutput({ items: [{ type: "upload_artifact", path: "report.json" }] });

      await runMain();

      expect(mockCore.setOutput).toHaveBeenCalledWith("slot_0_retention_days", "7");
    });
  });

  describe("validation errors", () => {
    it("fails when both path and filters are present", async () => {
      writeStaging("report.json");
      writeAgentOutput({
        items: [
          {
            type: "upload_artifact",
            path: "report.json",
            filters: { include: ["**/*.json"] },
          },
        ],
      });

      await runMain();
      expect(mockCore.setFailed).toHaveBeenCalledWith(expect.stringContaining("exactly one of 'path' or 'filters'"));
    });

    it("fails when neither path nor filters are present", async () => {
      writeAgentOutput({ items: [{ type: "upload_artifact", retention_days: 7 }] });

      await runMain();
      expect(mockCore.setFailed).toHaveBeenCalledWith(expect.stringContaining("exactly one of 'path' or 'filters'"));
    });

    it("fails when path traverses outside staging dir", async () => {
      writeAgentOutput({ items: [{ type: "upload_artifact", path: "../etc/passwd" }] });

      await runMain();
      expect(mockCore.setFailed).toHaveBeenCalledWith(expect.stringContaining("must not traverse outside staging directory"));
    });

    it("fails when absolute path is provided", async () => {
      writeAgentOutput({ items: [{ type: "upload_artifact", path: "/etc/passwd" }] });

      await runMain();
      expect(mockCore.setFailed).toHaveBeenCalledWith(expect.stringContaining("must be relative"));
    });

    it("fails when path does not exist in staging dir", async () => {
      writeAgentOutput({ items: [{ type: "upload_artifact", path: "nonexistent.json" }] });

      await runMain();
      expect(mockCore.setFailed).toHaveBeenCalledWith(expect.stringContaining("does not exist in staging directory"));
    });

    it("fails when max-uploads is exceeded", async () => {
      process.env.GH_AW_ARTIFACT_MAX_UPLOADS = "1";
      writeStaging("a.json");
      writeStaging("b.json");
      writeAgentOutput({
        items: [
          { type: "upload_artifact", path: "a.json" },
          { type: "upload_artifact", path: "b.json" },
        ],
      });

      await runMain();
      expect(mockCore.setFailed).toHaveBeenCalledWith(expect.stringContaining("exceed max-uploads policy"));
    });

    it("fails when skip_archive is requested but not allowed", async () => {
      writeStaging("app.bin");
      writeAgentOutput({ items: [{ type: "upload_artifact", path: "app.bin", skip_archive: true }] });

      await runMain();
      expect(mockCore.setFailed).toHaveBeenCalledWith(expect.stringContaining("skip_archive=true is not permitted"));
    });

    it("fails when skip_archive=true with multiple files", async () => {
      process.env.GH_AW_ARTIFACT_ALLOW_SKIP_ARCHIVE = "true";
      writeStaging("output/a.json");
      writeStaging("output/b.json");
      writeAgentOutput({
        items: [
          {
            type: "upload_artifact",
            // Use "output/**" which matches output/a.json and output/b.json
            filters: { include: ["output/**"] },
            skip_archive: true,
          },
        ],
      });

      await runMain();
      expect(mockCore.setFailed).toHaveBeenCalledWith(expect.stringContaining("skip_archive=true requires exactly one selected file"));
    });
  });

  describe("skip_archive allowed", () => {
    it("succeeds with skip_archive=true and a single file", async () => {
      process.env.GH_AW_ARTIFACT_ALLOW_SKIP_ARCHIVE = "true";
      writeStaging("app.bin", "binary data");
      writeAgentOutput({ items: [{ type: "upload_artifact", path: "app.bin", skip_archive: true }] });

      await runMain();

      expect(mockCore.setFailed).not.toHaveBeenCalled();
      expect(mockCore.setOutput).toHaveBeenCalledWith("slot_0_enabled", "true");
    });
  });

  describe("filter-based upload", () => {
    it("selects files matching include pattern", async () => {
      writeStaging("reports/daily/summary.json", "{}");
      writeStaging("reports/weekly/summary.json", "{}");
      writeStaging("reports/private/secret.json", "{}");
      writeAgentOutput({
        items: [
          {
            type: "upload_artifact",
            filters: {
              include: ["reports/**/*.json"],
              exclude: ["reports/private/**"],
            },
          },
        ],
      });

      await runMain();

      expect(mockCore.setFailed).not.toHaveBeenCalled();
      expect(mockCore.setOutput).toHaveBeenCalledWith("slot_0_enabled", "true");
      expect(mockCore.setOutput).toHaveBeenCalledWith("slot_0_file_count", "2");
    });

    it("handles no-files with if-no-files=ignore", async () => {
      process.env.GH_AW_ARTIFACT_DEFAULT_IF_NO_FILES = "ignore";
      writeAgentOutput({
        items: [
          {
            type: "upload_artifact",
            filters: { include: ["nonexistent/**"] },
          },
        ],
      });

      await runMain();

      expect(mockCore.setFailed).not.toHaveBeenCalled();
      expect(mockCore.setOutput).toHaveBeenCalledWith("artifact_count", "0");
    });

    it("fails when no files match and if-no-files=error (default)", async () => {
      writeAgentOutput({
        items: [
          {
            type: "upload_artifact",
            filters: { include: ["nonexistent/**"] },
          },
        ],
      });

      await runMain();

      expect(mockCore.setFailed).toHaveBeenCalledWith(expect.stringContaining("no files matched"));
    });
  });

  describe("allowed-paths policy", () => {
    it("filters out files not in allowed-paths", async () => {
      process.env.GH_AW_ARTIFACT_ALLOWED_PATHS = JSON.stringify(["dist/**"]);
      writeStaging("dist/app.js");
      writeStaging("secret.env");
      writeAgentOutput({
        items: [
          {
            type: "upload_artifact",
            filters: { include: ["**"] },
          },
        ],
      });

      await runMain();

      expect(mockCore.setFailed).not.toHaveBeenCalled();
      expect(mockCore.setOutput).toHaveBeenCalledWith("slot_0_file_count", "1");
    });
  });

  describe("staged mode", () => {
    it("skips file staging but sets outputs in staged mode", async () => {
      process.env.GH_AW_SAFE_OUTPUTS_STAGED = "true";
      writeStaging("report.json");
      writeAgentOutput({ items: [{ type: "upload_artifact", path: "report.json" }] });

      await runMain();

      expect(mockCore.setFailed).not.toHaveBeenCalled();
      expect(mockCore.setOutput).toHaveBeenCalledWith("slot_0_enabled", "true");

      // In staged mode, files are NOT copied to the slot directory.
      const slotFile = path.join(SLOT_BASE_DIR, "slot_0", "report.json");
      expect(fs.existsSync(slotFile)).toBe(false);
    });
  });

  describe("resolver file", () => {
    it("writes a resolver mapping with temporary IDs", async () => {
      writeStaging("report.json");
      writeAgentOutput({ items: [{ type: "upload_artifact", path: "report.json" }] });

      await runMain();

      expect(fs.existsSync(RESOLVER_FILE)).toBe(true);
      const resolver = JSON.parse(fs.readFileSync(RESOLVER_FILE, "utf8"));
      const keys = Object.keys(resolver);
      expect(keys.length).toBe(1);
      expect(keys[0]).toMatch(/^tmp_artifact_[A-Z0-9]{26}$/);
    });
  });
});
