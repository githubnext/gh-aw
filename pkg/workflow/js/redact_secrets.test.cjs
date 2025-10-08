import { describe, it, expect, beforeEach, afterEach, vi } from "vitest";
import fs from "fs";
import path from "path";
import os from "os";

// Mock the global objects that GitHub Actions provides
const mockCore = {
  debug: vi.fn(),
  info: vi.fn(),
  notice: vi.fn(),
  warning: vi.fn(),
  error: vi.fn(),
  setFailed: vi.fn(),
  setOutput: vi.fn(),
  exportVariable: vi.fn(),
  setSecret: vi.fn(),
  getInput: vi.fn(),
  getBooleanInput: vi.fn(),
  getMultilineInput: vi.fn(),
  getState: vi.fn(),
  saveState: vi.fn(),
  startGroup: vi.fn(),
  endGroup: vi.fn(),
  group: vi.fn(),
  addPath: vi.fn(),
  setCommandEcho: vi.fn(),
  isDebug: vi.fn().mockReturnValue(false),
  getIDToken: vi.fn(),
  summary: {
    addRaw: vi.fn().mockReturnThis(),
    write: vi.fn().mockResolvedValue(undefined),
  },
};

global.core = mockCore;

// Read the actual script file
const redactScript = fs.readFileSync(path.join(__dirname, "redact_secrets.cjs"), "utf8");

describe("redact_secrets.cjs", () => {
  let tempDir;

  beforeEach(() => {
    // Create a temporary directory for test files
    tempDir = fs.mkdtempSync(path.join(os.tmpdir(), "redact-test-"));

    // Reset all mocks
    Object.values(mockCore).forEach(fn => {
      if (typeof fn === "function") {
        fn.mockClear();
      }
    });
    if (mockCore.summary.addRaw) mockCore.summary.addRaw.mockClear();
    if (mockCore.summary.write) mockCore.summary.write.mockClear();

    // Clear environment variables
    delete process.env.GITHUB_AW_SECRETS_PATTERN;
  });

  afterEach(() => {
    // Clean up temporary directory
    if (tempDir && fs.existsSync(tempDir)) {
      fs.rmSync(tempDir, { recursive: true, force: true });
    }
  });

  describe("file scanning and redaction", () => {
    it("should find and redact files with target extensions", async () => {
      // Create test files
      fs.writeFileSync(path.join(tempDir, "test1.txt"), "Secret: sk-1234567890");
      fs.writeFileSync(path.join(tempDir, "test2.json"), '{"key": "sk-0987654321"}');
      fs.writeFileSync(path.join(tempDir, "test3.log"), "Log entry: sk-abcdefghij");
      fs.writeFileSync(path.join(tempDir, "test4.md"), "sk-shouldnotberedacted"); // Should be ignored

      process.env.GITHUB_AW_SECRETS_PATTERN = "sk-[a-zA-Z0-9]+";

      const modifiedScript = redactScript.replace(
        'findFiles("/tmp", targetExtensions)',
        `findFiles("${tempDir.replace(/\\/g, "\\\\")}", targetExtensions)`
      );

      await eval(`(async () => { ${modifiedScript} })()`);

      // Check that target files were redacted
      expect(fs.readFileSync(path.join(tempDir, "test1.txt"), "utf8")).toBe("Secret: ***REDACTED***");
      expect(fs.readFileSync(path.join(tempDir, "test2.json"), "utf8")).toBe('{"key": "***REDACTED***"}');
      expect(fs.readFileSync(path.join(tempDir, "test3.log"), "utf8")).toBe("Log entry: ***REDACTED***");

      // Markdown file should not be modified
      expect(fs.readFileSync(path.join(tempDir, "test4.md"), "utf8")).toBe("sk-shouldnotberedacted");
    });

    it("should recursively search subdirectories", async () => {
      // Create nested directory structure
      const subDir = path.join(tempDir, "subdir");
      fs.mkdirSync(subDir);
      fs.writeFileSync(path.join(subDir, "nested.txt"), "Nested secret: api-key-123");
      fs.writeFileSync(path.join(tempDir, "root.log"), "Root secret: api-key-456");

      process.env.GITHUB_AW_SECRETS_PATTERN = "api-key-[0-9]+";

      const modifiedScript = redactScript.replace(
        'findFiles("/tmp", targetExtensions)',
        `findFiles("${tempDir.replace(/\\/g, "\\\\")}", targetExtensions)`
      );

      await eval(`(async () => { ${modifiedScript} })()`);

      // Both files should be redacted
      expect(fs.readFileSync(path.join(subDir, "nested.txt"), "utf8")).toBe("Nested secret: ***REDACTED***");
      expect(fs.readFileSync(path.join(tempDir, "root.log"), "utf8")).toBe("Root secret: ***REDACTED***");
    });
  });

  describe("main function integration", () => {
    it("should skip redaction when GITHUB_AW_SECRETS_PATTERN is not set", async () => {
      // Execute the script without setting the environment variable
      await eval(`(async () => { ${redactScript} })()`);

      expect(mockCore.info).toHaveBeenCalledWith("GITHUB_AW_SECRETS_PATTERN not set, no redaction performed");
    });

    it("should redact secrets from files in /tmp", async () => {
      // Create test files in temp directory (simulating /tmp)
      const testFile = path.join(tempDir, "test.txt");
      fs.writeFileSync(testFile, "Secret: sk-1234567890 and another sk-0987654321");

      // Set environment variable with regex pattern
      process.env.GITHUB_AW_SECRETS_PATTERN = "sk-[a-zA-Z0-9]+";

      // Mock findFiles to return our test directory
      const modifiedScript = redactScript.replace(
        'findFiles("/tmp", targetExtensions)',
        `findFiles("${tempDir.replace(/\\/g, "\\\\")}", targetExtensions)`
      );

      // Execute the modified script
      await eval(`(async () => { ${modifiedScript} })()`);

      // Read the file and check if secrets were redacted
      const redactedContent = fs.readFileSync(testFile, "utf8");
      expect(redactedContent).toBe("Secret: ***REDACTED*** and another ***REDACTED***");

      // Check that info was logged
      expect(mockCore.info).toHaveBeenCalledWith("Starting secret redaction in /tmp directory");
      expect(mockCore.info).toHaveBeenCalledWith(expect.stringContaining("Secret redaction complete"));
    });

    it("should handle multiple file types", async () => {
      // Create test files
      fs.writeFileSync(path.join(tempDir, "test1.txt"), "Secret: api-key-123");
      fs.writeFileSync(path.join(tempDir, "test2.json"), '{"key": "api-key-456"}');
      fs.writeFileSync(path.join(tempDir, "test3.log"), "Log: api-key-789");

      process.env.GITHUB_AW_SECRETS_PATTERN = "api-key-[0-9]+";

      const modifiedScript = redactScript.replace(
        'findFiles("/tmp", targetExtensions)',
        `findFiles("${tempDir.replace(/\\/g, "\\\\")}", targetExtensions)`
      );

      await eval(`(async () => { ${modifiedScript} })()`);

      // Check all files were redacted
      expect(fs.readFileSync(path.join(tempDir, "test1.txt"), "utf8")).toBe("Secret: ***REDACTED***");
      expect(fs.readFileSync(path.join(tempDir, "test2.json"), "utf8")).toBe('{"key": "***REDACTED***"}');
      expect(fs.readFileSync(path.join(tempDir, "test3.log"), "utf8")).toBe("Log: ***REDACTED***");
    });

    it("should use core.debug for logging hits", async () => {
      const testFile = path.join(tempDir, "test.txt");
      fs.writeFileSync(testFile, "Secret: sk-123 and sk-456");

      process.env.GITHUB_AW_SECRETS_PATTERN = "sk-[0-9]+";

      const modifiedScript = redactScript.replace(
        'findFiles("/tmp", targetExtensions)',
        `findFiles("${tempDir.replace(/\\/g, "\\\\")}", targetExtensions)`
      );

      await eval(`(async () => { ${modifiedScript} })()`);

      // Verify core.debug was called for each redaction
      expect(mockCore.debug).toHaveBeenCalledWith("Redacted secret occurrence #1");
      expect(mockCore.debug).toHaveBeenCalledWith("Redacted secret occurrence #2");
      expect(mockCore.debug).toHaveBeenCalledWith(expect.stringContaining("Processed"));
    });

    it("should not log actual secret values", async () => {
      const testFile = path.join(tempDir, "test.txt");
      const secretValue = "sk-very-secret-key-123";
      fs.writeFileSync(testFile, `Secret: ${secretValue}`);

      process.env.GITHUB_AW_SECRETS_PATTERN = "sk-[a-zA-Z0-9-]+";

      const modifiedScript = redactScript.replace(
        'findFiles("/tmp", targetExtensions)',
        `findFiles("${tempDir.replace(/\\/g, "\\\\")}", targetExtensions)`
      );

      await eval(`(async () => { ${modifiedScript} })()`);

      // Check that no mock call contains the actual secret value
      const allCalls = [...mockCore.debug.mock.calls, ...mockCore.info.mock.calls, ...mockCore.warning.mock.calls];

      for (const call of allCalls) {
        const callString = JSON.stringify(call);
        expect(callString).not.toContain(secretValue);
      }
    });
  });
});
