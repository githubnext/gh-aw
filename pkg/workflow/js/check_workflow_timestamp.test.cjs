import { describe, it, expect, beforeEach, afterEach, vi } from "vitest";
import fs from "fs";
import path from "path";
import os from "os";

// Mock the global objects that GitHub Actions provides
const mockCore = {
  // Core logging functions
  debug: vi.fn(),
  info: vi.fn(),
  notice: vi.fn(),
  warning: vi.fn(),
  error: vi.fn(),

  // Core workflow functions
  setFailed: vi.fn(),
  setOutput: vi.fn(),
  exportVariable: vi.fn(),
  setSecret: vi.fn(),
  setCancelled: vi.fn(),
  setError: vi.fn(),

  // Input/state functions
  getInput: vi.fn(),
  getBooleanInput: vi.fn(),
  getMultilineInput: vi.fn(),
  getState: vi.fn(),
  saveState: vi.fn(),

  // Group functions
  startGroup: vi.fn(),
  endGroup: vi.fn(),
  group: vi.fn(),

  // Other utility functions
  addPath: vi.fn(),
  setCommandEcho: vi.fn(),
  isDebug: vi.fn().mockReturnValue(false),
  getIDToken: vi.fn(),
  toPlatformPath: vi.fn(),
  toPosixPath: vi.fn(),
  toWin32Path: vi.fn(),

  // Summary object with chainable methods
  summary: {
    addRaw: vi.fn().mockReturnThis(),
    write: vi.fn().mockResolvedValue(),
  },
};

// Set up global variables
global.core = mockCore;

describe("check_workflow_timestamp.cjs", () => {
  let checkWorkflowTimestampScript;
  let originalEnv;
  let tmpDir;
  let workflowsDir;

  beforeEach(() => {
    // Reset all mocks
    vi.clearAllMocks();

    // Store original environment
    originalEnv = {
      GITHUB_WORKSPACE: process.env.GITHUB_WORKSPACE,
      GITHUB_WORKFLOW: process.env.GITHUB_WORKFLOW,
    };

    // Create a temporary directory for test files
    tmpDir = fs.mkdtempSync(path.join(os.tmpdir(), "workflow-test-"));
    workflowsDir = path.join(tmpDir, ".github", "workflows");
    fs.mkdirSync(workflowsDir, { recursive: true });

    // Set up environment
    process.env.GITHUB_WORKSPACE = tmpDir;

    // Read the script content
    const scriptPath = path.join(process.cwd(), "check_workflow_timestamp.cjs");
    checkWorkflowTimestampScript = fs.readFileSync(scriptPath, "utf8");
  });

  afterEach(() => {
    // Restore original environment
    if (originalEnv.GITHUB_WORKSPACE !== undefined) {
      process.env.GITHUB_WORKSPACE = originalEnv.GITHUB_WORKSPACE;
    } else {
      delete process.env.GITHUB_WORKSPACE;
    }
    if (originalEnv.GITHUB_WORKFLOW !== undefined) {
      process.env.GITHUB_WORKFLOW = originalEnv.GITHUB_WORKFLOW;
    } else {
      delete process.env.GITHUB_WORKFLOW;
    }

    // Clean up temporary directory
    if (tmpDir && fs.existsSync(tmpDir)) {
      fs.rmSync(tmpDir, { recursive: true, force: true });
    }
  });

  describe("when environment variables are missing", () => {
    it("should fail if GITHUB_WORKSPACE is not set", async () => {
      delete process.env.GITHUB_WORKSPACE;
      process.env.GITHUB_WORKFLOW = "test.lock.yml";

      await eval(`(async () => { ${checkWorkflowTimestampScript} })()`);

      expect(mockCore.setFailed).toHaveBeenCalledWith(expect.stringContaining("GITHUB_WORKSPACE not available"));
    });

    it("should fail if GITHUB_WORKFLOW is not set", async () => {
      process.env.GITHUB_WORKSPACE = tmpDir;
      delete process.env.GITHUB_WORKFLOW;

      await eval(`(async () => { ${checkWorkflowTimestampScript} })()`);

      expect(mockCore.setFailed).toHaveBeenCalledWith(expect.stringContaining("GITHUB_WORKFLOW not available"));
    });
  });

  describe("when files do not exist", () => {
    it("should skip check when source file does not exist", async () => {
      process.env.GITHUB_WORKSPACE = tmpDir;
      process.env.GITHUB_WORKFLOW = "test.lock.yml";

      // Create only the lock file
      const lockFile = path.join(workflowsDir, "test.lock.yml");
      fs.writeFileSync(lockFile, "# Lock file content");

      await eval(`(async () => { ${checkWorkflowTimestampScript} })()`);

      expect(mockCore.info).toHaveBeenCalledWith(expect.stringContaining("Source file does not exist"));
      expect(mockCore.info).toHaveBeenCalledWith(expect.stringContaining("Skipping timestamp check"));
      expect(mockCore.setFailed).not.toHaveBeenCalled();
      expect(mockCore.error).not.toHaveBeenCalled();
    });

    it("should skip check when lock file does not exist", async () => {
      process.env.GITHUB_WORKSPACE = tmpDir;
      process.env.GITHUB_WORKFLOW = "test.lock.yml";

      // Create only the source file
      const workflowFile = path.join(workflowsDir, "test.md");
      fs.writeFileSync(workflowFile, "# Workflow content");

      await eval(`(async () => { ${checkWorkflowTimestampScript} })()`);

      expect(mockCore.info).toHaveBeenCalledWith(expect.stringContaining("Lock file does not exist"));
      expect(mockCore.info).toHaveBeenCalledWith(expect.stringContaining("Skipping timestamp check"));
      expect(mockCore.setFailed).not.toHaveBeenCalled();
      expect(mockCore.error).not.toHaveBeenCalled();
    });

    it("should skip check when both files do not exist", async () => {
      process.env.GITHUB_WORKSPACE = tmpDir;
      process.env.GITHUB_WORKFLOW = "test.lock.yml";

      await eval(`(async () => { ${checkWorkflowTimestampScript} })()`);

      expect(mockCore.info).toHaveBeenCalledWith(expect.stringContaining("Skipping timestamp check"));
      expect(mockCore.setFailed).not.toHaveBeenCalled();
      expect(mockCore.error).not.toHaveBeenCalled();
    });
  });

  describe("when lock file is up to date", () => {
    it("should pass when lock file is newer than source file", async () => {
      process.env.GITHUB_WORKSPACE = tmpDir;
      process.env.GITHUB_WORKFLOW = "test.lock.yml";

      const workflowFile = path.join(workflowsDir, "test.md");
      const lockFile = path.join(workflowsDir, "test.lock.yml");

      // Create source file first
      fs.writeFileSync(workflowFile, "# Workflow content");

      // Wait a bit to ensure different timestamps
      await new Promise(resolve => setTimeout(resolve, 10));

      // Create lock file (newer)
      fs.writeFileSync(lockFile, "# Lock file content");

      await eval(`(async () => { ${checkWorkflowTimestampScript} })()`);

      expect(mockCore.info).toHaveBeenCalledWith(expect.stringContaining("Lock file is up to date"));
      expect(mockCore.error).not.toHaveBeenCalled();
      expect(mockCore.setFailed).not.toHaveBeenCalled();
      expect(mockCore.summary.addRaw).not.toHaveBeenCalled();
    });

    it("should pass when lock file has same timestamp as source file", async () => {
      process.env.GITHUB_WORKSPACE = tmpDir;
      process.env.GITHUB_WORKFLOW = "test.lock.yml";

      const workflowFile = path.join(workflowsDir, "test.md");
      const lockFile = path.join(workflowsDir, "test.lock.yml");

      // Create both files at the same time
      const now = new Date();
      fs.writeFileSync(workflowFile, "# Workflow content");
      fs.writeFileSync(lockFile, "# Lock file content");

      // Set both files to have the exact same timestamp
      fs.utimesSync(workflowFile, now, now);
      fs.utimesSync(lockFile, now, now);

      await eval(`(async () => { ${checkWorkflowTimestampScript} })()`);

      expect(mockCore.info).toHaveBeenCalledWith(expect.stringContaining("Lock file is up to date"));
      expect(mockCore.error).not.toHaveBeenCalled();
      expect(mockCore.setFailed).not.toHaveBeenCalled();
      expect(mockCore.summary.addRaw).not.toHaveBeenCalled();
    });
  });

  describe("when lock file is outdated", () => {
    it("should warn when source file is newer than lock file", async () => {
      process.env.GITHUB_WORKSPACE = tmpDir;
      process.env.GITHUB_WORKFLOW = "test.lock.yml";

      const workflowFile = path.join(workflowsDir, "test.md");
      const lockFile = path.join(workflowsDir, "test.lock.yml");

      // Create lock file first
      fs.writeFileSync(lockFile, "# Lock file content");

      // Wait a bit to ensure different timestamps
      await new Promise(resolve => setTimeout(resolve, 10));

      // Create source file (newer)
      fs.writeFileSync(workflowFile, "# Workflow content");

      await eval(`(async () => { ${checkWorkflowTimestampScript} })()`);

      expect(mockCore.error).toHaveBeenCalledWith(expect.stringContaining("WARNING: Lock file"));
      expect(mockCore.error).toHaveBeenCalledWith(expect.stringContaining("is outdated"));
      expect(mockCore.error).toHaveBeenCalledWith(expect.stringContaining("gh aw compile"));
      expect(mockCore.summary.addRaw).toHaveBeenCalled();
      expect(mockCore.summary.write).toHaveBeenCalled();
      expect(mockCore.setFailed).not.toHaveBeenCalled();
    });

    it("should include file paths in warning message", async () => {
      process.env.GITHUB_WORKSPACE = tmpDir;
      process.env.GITHUB_WORKFLOW = "my-workflow.lock.yml";

      const workflowFile = path.join(workflowsDir, "my-workflow.md");
      const lockFile = path.join(workflowsDir, "my-workflow.lock.yml");

      // Create lock file first
      fs.writeFileSync(lockFile, "# Lock file content");

      // Wait a bit to ensure different timestamps
      await new Promise(resolve => setTimeout(resolve, 10));

      // Create source file (newer)
      fs.writeFileSync(workflowFile, "# Workflow content");

      await eval(`(async () => { ${checkWorkflowTimestampScript} })()`);

      expect(mockCore.error).toHaveBeenCalledWith(expect.stringMatching(/my-workflow\.lock\.yml.*outdated/));
      expect(mockCore.error).toHaveBeenCalledWith(expect.stringMatching(/my-workflow\.md.*modified more recently/));
    });

    it("should add step summary with warning", async () => {
      process.env.GITHUB_WORKSPACE = tmpDir;
      process.env.GITHUB_WORKFLOW = "test.lock.yml";

      const workflowFile = path.join(workflowsDir, "test.md");
      const lockFile = path.join(workflowsDir, "test.lock.yml");

      // Create lock file first
      fs.writeFileSync(lockFile, "# Lock file content");

      // Wait a bit to ensure different timestamps
      await new Promise(resolve => setTimeout(resolve, 10));

      // Create source file (newer)
      fs.writeFileSync(workflowFile, "# Workflow content");

      await eval(`(async () => { ${checkWorkflowTimestampScript} })()`);

      expect(mockCore.summary.addRaw).toHaveBeenCalledWith(expect.stringContaining("Workflow Lock File Warning"));
      expect(mockCore.summary.addRaw).toHaveBeenCalledWith(expect.stringContaining("WARNING"));
      expect(mockCore.summary.addRaw).toHaveBeenCalledWith(expect.stringContaining("gh aw compile"));
      expect(mockCore.summary.write).toHaveBeenCalled();
    });
  });

  describe("with different workflow names", () => {
    it("should handle workflow names with hyphens", async () => {
      process.env.GITHUB_WORKSPACE = tmpDir;
      process.env.GITHUB_WORKFLOW = "my-test-workflow.lock.yml";

      const workflowFile = path.join(workflowsDir, "my-test-workflow.md");
      const lockFile = path.join(workflowsDir, "my-test-workflow.lock.yml");

      fs.writeFileSync(workflowFile, "# Workflow content");
      fs.writeFileSync(lockFile, "# Lock file content");

      await eval(`(async () => { ${checkWorkflowTimestampScript} })()`);

      expect(mockCore.info).toHaveBeenCalledWith(expect.stringContaining("my-test-workflow.md"));
      expect(mockCore.setFailed).not.toHaveBeenCalled();
    });

    it("should handle workflow names with underscores", async () => {
      process.env.GITHUB_WORKSPACE = tmpDir;
      process.env.GITHUB_WORKFLOW = "my_test_workflow.lock.yml";

      const workflowFile = path.join(workflowsDir, "my_test_workflow.md");
      const lockFile = path.join(workflowsDir, "my_test_workflow.lock.yml");

      fs.writeFileSync(workflowFile, "# Workflow content");
      fs.writeFileSync(lockFile, "# Lock file content");

      await eval(`(async () => { ${checkWorkflowTimestampScript} })()`);

      expect(mockCore.info).toHaveBeenCalledWith(expect.stringContaining("my_test_workflow.md"));
      expect(mockCore.setFailed).not.toHaveBeenCalled();
    });
  });
});
