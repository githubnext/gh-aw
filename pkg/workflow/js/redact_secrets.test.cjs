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
    delete process.env.GH_AW_SECRET_NAMES;
  });

  afterEach(() => {
    // Clean up temporary directory
    if (tempDir && fs.existsSync(tempDir)) {
      fs.rmSync(tempDir, { recursive: true, force: true });
    }

    // Clean up environment variables
    for (const key of Object.keys(process.env)) {
      if (key.startsWith("SECRET_")) {
        delete process.env[key];
      }
    }
  });

  describe("main function integration", () => {
    it("should skip redaction when GH_AW_SECRET_NAMES is not set", async () => {
      // Execute the script without setting the environment variable
      await eval(`(async () => { ${redactScript} })()`);

      expect(mockCore.info).toHaveBeenCalledWith("GH_AW_SECRET_NAMES not set, no redaction performed");
    });

    it("should redact secrets from files in /tmp using exact matching", async () => {
      // Create test files in temp directory (simulating /tmp)
      const testFile = path.join(tempDir, "test.txt");
      const secretValue = "ghp_1234567890abcdefghijklmnopqrstuvwxyz";
      fs.writeFileSync(testFile, `Secret: ${secretValue} and another ${secretValue}`);

      // Set environment variables
      process.env.GH_AW_SECRET_NAMES = "GITHUB_TOKEN";
      process.env.SECRET_GITHUB_TOKEN = secretValue;

      // Mock findFiles to return our test directory
      const modifiedScript = redactScript.replace(
        'findFiles("/tmp/gh-aw", targetExtensions)',
        `findFiles("${tempDir.replace(/\\/g, "\\\\")}", targetExtensions)`
      );

      // Execute the modified script
      await eval(`(async () => { ${modifiedScript} })()`);

      // Read the file and check if secrets were redacted
      const redactedContent = fs.readFileSync(testFile, "utf8");
      // ghp_1234567890abcdefghijklmnopqrstuvwxyz (40 chars) -> ghp + 37 asterisks
      expect(redactedContent).toBe("Secret: ghp************************************* and another ghp*************************************");

      // Check that info was logged
      expect(mockCore.info).toHaveBeenCalledWith("Starting secret redaction in /tmp/gh-aw directory");
      expect(mockCore.info).toHaveBeenCalledWith(expect.stringContaining("Secret redaction complete"));
    });

    it("should handle multiple file types", async () => {
      // Create test files
      fs.writeFileSync(path.join(tempDir, "test1.txt"), "Secret: api-key-123");
      fs.writeFileSync(path.join(tempDir, "test2.json"), '{"key": "api-key-456"}');
      fs.writeFileSync(path.join(tempDir, "test3.log"), "Log: api-key-789");

      process.env.GH_AW_SECRET_NAMES = "API_KEY1,API_KEY2,API_KEY3";
      process.env.SECRET_API_KEY1 = "api-key-123";
      process.env.SECRET_API_KEY2 = "api-key-456";
      process.env.SECRET_API_KEY3 = "api-key-789";

      const modifiedScript = redactScript.replace(
        'findFiles("/tmp/gh-aw", targetExtensions)',
        `findFiles("${tempDir.replace(/\\/g, "\\\\")}", targetExtensions)`
      );

      await eval(`(async () => { ${modifiedScript} })()`);

      // Check all files were redacted
      // api-key-123 (11 chars) -> api + 8 asterisks
      expect(fs.readFileSync(path.join(tempDir, "test1.txt"), "utf8")).toBe("Secret: api********");
      expect(fs.readFileSync(path.join(tempDir, "test2.json"), "utf8")).toBe('{"key": "api********"}');
      expect(fs.readFileSync(path.join(tempDir, "test3.log"), "utf8")).toBe("Log: api********");
    });

    it("should use core.info for logging hits", async () => {
      const testFile = path.join(tempDir, "test.txt");
      const secretValue = "sk-1234567890";
      fs.writeFileSync(testFile, `Secret: ${secretValue} and ${secretValue}`);

      process.env.GH_AW_SECRET_NAMES = "API_KEY";
      process.env.SECRET_API_KEY = secretValue;

      const modifiedScript = redactScript.replace(
        'findFiles("/tmp/gh-aw", targetExtensions)',
        `findFiles("${tempDir.replace(/\\/g, "\\\\")}", targetExtensions)`
      );

      await eval(`(async () => { ${modifiedScript} })()`);

      // Verify core.info was called for redaction
      expect(mockCore.info).toHaveBeenCalledWith(expect.stringContaining("occurrence(s) of a secret"));
      expect(mockCore.info).toHaveBeenCalledWith(expect.stringContaining("Processed"));
    });

    it("should not log actual secret values", async () => {
      const testFile = path.join(tempDir, "test.txt");
      const secretValue = "sk-very-secret-key-123";
      fs.writeFileSync(testFile, `Secret: ${secretValue}`);

      process.env.GH_AW_SECRET_NAMES = "SECRET_KEY";
      process.env.SECRET_SECRET_KEY = secretValue;

      const modifiedScript = redactScript.replace(
        'findFiles("/tmp/gh-aw", targetExtensions)',
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

    it("should skip very short values", async () => {
      const testFile = path.join(tempDir, "test.txt");
      fs.writeFileSync(testFile, "Short: abc123");

      process.env.GH_AW_SECRET_NAMES = "SHORT_SECRET";
      process.env.SECRET_SHORT_SECRET = "abc"; // Only 3 chars, should be skipped

      const modifiedScript = redactScript.replace(
        'findFiles("/tmp/gh-aw", targetExtensions)',
        `findFiles("${tempDir.replace(/\\/g, "\\\\")}", targetExtensions)`
      );

      await eval(`(async () => { ${modifiedScript} })()`);

      // File should not be modified since secret is too short
      expect(fs.readFileSync(testFile, "utf8")).toBe("Short: abc123");
    });

    it("should handle multiple secrets in same file", async () => {
      const testFile = path.join(tempDir, "test.txt");
      const secret1 = "ghp_1234567890abcdefghijklmnopqrstuvwxyz";
      const secret2 = "sk-proj-abcdef1234567890";
      fs.writeFileSync(testFile, `Token1: ${secret1}\nToken2: ${secret2}\nToken1 again: ${secret1}`);

      process.env.GH_AW_SECRET_NAMES = "TOKEN1,TOKEN2";
      process.env.SECRET_TOKEN1 = secret1;
      process.env.SECRET_TOKEN2 = secret2;

      const modifiedScript = redactScript.replace(
        'findFiles("/tmp/gh-aw", targetExtensions)',
        `findFiles("${tempDir.replace(/\\/g, "\\\\")}", targetExtensions)`
      );

      await eval(`(async () => { ${modifiedScript} })()`);

      const redacted = fs.readFileSync(testFile, "utf8");
      // ghp_1234567890abcdefghijklmnopqrstuvwxyz (40 chars) -> ghp + 37 asterisks
      // sk-proj-abcdef1234567890 (24 chars) -> sk- + 21 asterisks
      expect(redacted).toBe(
        "Token1: ghp*************************************\nToken2: sk-*********************\nToken1 again: ghp*************************************"
      );
    });

    it("should handle empty secret values gracefully", async () => {
      const testFile = path.join(tempDir, "test.txt");
      fs.writeFileSync(testFile, "No secrets here");

      process.env.GH_AW_SECRET_NAMES = "EMPTY_SECRET";
      process.env.SECRET_EMPTY_SECRET = "";

      const modifiedScript = redactScript.replace(
        'findFiles("/tmp/gh-aw", targetExtensions)',
        `findFiles("${tempDir.replace(/\\/g, "\\\\")}", targetExtensions)`
      );

      await eval(`(async () => { ${modifiedScript} })()`);

      // Should complete without error
      expect(mockCore.info).toHaveBeenCalledWith(expect.stringContaining("No secret values found to redact"));
    });
  });
});
