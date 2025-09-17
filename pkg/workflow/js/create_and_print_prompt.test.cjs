import { describe, it, expect, beforeEach, afterEach, vi } from "vitest";
import fs from "fs";
import path from "path";

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

  // Input/state functions (less commonly used but included for completeness)
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

describe("create_and_print_prompt.cjs", () => {
  let promptScript;

  beforeEach(() => {
    // Reset all mocks
    vi.clearAllMocks();

    // Reset environment variables
    delete process.env.GITHUB_AW_PROMPT;
    delete process.env.GITHUB_AW_PROMPT_CONTENT;

    // Read the script content
    const scriptPath = path.join(
      process.cwd(),
      "pkg/workflow/js/create_and_print_prompt.cjs"
    );
    promptScript = fs.readFileSync(scriptPath, "utf8");

    // Make fs available globally for the evaluated script
    global.fs = fs;
    global.require = require;
  });

  afterEach(() => {
    // Clean up any test files
    const testPromptFile = "/tmp/test-prompt/prompt.txt";
    if (fs.existsSync(testPromptFile)) {
      fs.unlinkSync(testPromptFile);
    }
    if (fs.existsSync("/tmp/test-prompt")) {
      fs.rmdirSync("/tmp/test-prompt");
    }

    // Clean up globals
    delete global.fs;
    delete global.require;
  });

  describe("main function", () => {
    it("should handle missing GITHUB_AW_PROMPT environment variable", async () => {
      delete process.env.GITHUB_AW_PROMPT;
      process.env.GITHUB_AW_PROMPT_CONTENT = "test content";

      // Execute the script
      await eval(`(async () => { ${promptScript} })()`);

      expect(mockCore.setFailed).toHaveBeenCalledWith(
        "GITHUB_AW_PROMPT environment variable is required"
      );
    });

    it("should handle missing GITHUB_AW_PROMPT_CONTENT environment variable", async () => {
      process.env.GITHUB_AW_PROMPT = "/tmp/test-prompt/prompt.txt";
      delete process.env.GITHUB_AW_PROMPT_CONTENT;

      // Execute the script
      await eval(`(async () => { ${promptScript} })()`);

      expect(mockCore.setFailed).toHaveBeenCalledWith(
        "GITHUB_AW_PROMPT_CONTENT environment variable is required"
      );
    });

    it("should create prompt file and set outputs correctly", async () => {
      const promptPath = "/tmp/test-prompt/prompt.txt";
      const promptContent = "This is a test prompt\nWith multiple lines\nAnd special characters: \"quotes\" and 'apostrophes'";

      process.env.GITHUB_AW_PROMPT = promptPath;
      process.env.GITHUB_AW_PROMPT_CONTENT = promptContent;

      // Execute the script
      await eval(`(async () => { ${promptScript} })()`);

      // Check that the file was created
      expect(fs.existsSync(promptPath)).toBe(true);

      // Check the file content
      const fileContent = fs.readFileSync(promptPath, "utf8");
      expect(fileContent).toBe(promptContent);

      // Check that exportVariable was called with JSON stringified content
      expect(mockCore.exportVariable).not.toHaveBeenCalled();

      // Check that setOutput was called correctly
      expect(mockCore.setOutput).toHaveBeenCalledWith("prompt_file", promptPath);
      expect(mockCore.setOutput).not.toHaveBeenCalledWith(
        "prompt_content_json",
        expect.anything()
      );

      // Check that summary was written
      expect(mockCore.summary.addRaw).toHaveBeenCalledWith("## Generated Prompt\n\n");
      expect(mockCore.summary.addRaw).toHaveBeenCalledWith("``````markdown\n");
      expect(mockCore.summary.addRaw).toHaveBeenCalledWith(promptContent);
      expect(mockCore.summary.addRaw).toHaveBeenCalledWith("\n``````");
      expect(mockCore.summary.write).toHaveBeenCalled();

      // Check success logging
      expect(mockCore.info).toHaveBeenCalledWith(`Prompt written to: ${promptPath}`);
      expect(mockCore.info).toHaveBeenCalledWith("Prompt successfully written to step summary");
    });

    it("should create directory structure if it doesn't exist", async () => {
      const promptPath = "/tmp/deep/nested/path/prompt.txt";
      const promptContent = "Test content for nested directory";

      process.env.GITHUB_AW_PROMPT = promptPath;
      process.env.GITHUB_AW_PROMPT_CONTENT = promptContent;

      // Ensure the directory doesn't exist initially
      if (fs.existsSync("/tmp/deep")) {
        fs.rmSync("/tmp/deep", { recursive: true });
      }

      // Execute the script
      await eval(`(async () => { ${promptScript} })()`);

      // Check that the directory was created
      expect(fs.existsSync("/tmp/deep/nested/path")).toBe(true);

      // Check that the file was created
      expect(fs.existsSync(promptPath)).toBe(true);

      // Check the file content
      const fileContent = fs.readFileSync(promptPath, "utf8");
      expect(fileContent).toBe(promptContent);

      // Clean up
      fs.rmSync("/tmp/deep", { recursive: true });
    });

    it("should handle special characters and newlines in prompt content", async () => {
      const promptPath = "/tmp/test-prompt/special.txt";
      const promptContent = "Content with:\n- Newlines\n- \"Double quotes\"\n- 'Single quotes'\n- Special chars: @#$%^&*()";

      process.env.GITHUB_AW_PROMPT = promptPath;
      process.env.GITHUB_AW_PROMPT_CONTENT = promptContent;

      // Execute the script
      await eval(`(async () => { ${promptScript} })()`);

      // Check that the JSON export was not called (removed)
      expect(mockCore.exportVariable).not.toHaveBeenCalled();

      // Check that the file content is preserved exactly
      const fileContent = fs.readFileSync(promptPath, "utf8");
      expect(fileContent).toBe(promptContent);

      // Check that no errors were logged
      expect(mockCore.setFailed).not.toHaveBeenCalled();
    });

    it("should handle errors during file operations", async () => {
      const promptPath = "/invalid/path/that/cannot/be/created/prompt.txt";
      const promptContent = "Test content";

      process.env.GITHUB_AW_PROMPT = promptPath;
      process.env.GITHUB_AW_PROMPT_CONTENT = promptContent;

      // Execute the script
      await eval(`(async () => { ${promptScript} })()`);

      // Check that setFailed was called with an error
      expect(mockCore.setFailed).toHaveBeenCalled();
      const failureMessage = mockCore.setFailed.mock.calls[0][0];
      expect(failureMessage).toMatch(/ENOENT|permission denied|no such file or directory/i);
    });

    it("should log debug information", async () => {
      const promptPath = "/tmp/test-prompt/prompt.txt";
      const promptContent = "Debug test content";

      process.env.GITHUB_AW_PROMPT = promptPath;
      process.env.GITHUB_AW_PROMPT_CONTENT = promptContent;

      // Execute the script
      await eval(`(async () => { ${promptScript} })()`);

      // Check that debug information was logged
      expect(mockCore.debug).toHaveBeenCalledWith(
        `Prompt content length: ${promptContent.length} characters`
      );
    });
  });
});