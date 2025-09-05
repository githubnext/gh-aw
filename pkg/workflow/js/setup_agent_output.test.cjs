import { describe, it, expect, beforeEach, vi } from "vitest";
import fs from "fs";
import path from "path";

// Mock the global objects that GitHub Actions provides
const mockCore = {
  exportVariable: vi.fn(),
  setOutput: vi.fn(),
  warning: vi.fn(),
  error: vi.fn(),
};

// Set up global variables
global.core = mockCore;

describe("setup_agent_output.cjs", () => {
  let setupScript;

  beforeEach(() => {
    // Reset all mocks
    vi.clearAllMocks();

    // Read the script content
    const scriptPath = path.join(
      process.cwd(),
      "pkg/workflow/js/setup_agent_output.cjs"
    );
    setupScript = fs.readFileSync(scriptPath, "utf8");

    // Make fs available globally for the evaluated script
    global.fs = fs;
  });

  afterEach(() => {
    // Clean up any test files
    const files = fs
      .readdirSync("/tmp")
      .filter(file => file.startsWith("aw_output_"));
    files.forEach(file => {
      try {
        fs.unlinkSync(path.join("/tmp", file));
      } catch (e) {
        // Ignore cleanup errors
      }
    });

    // Clean up globals
    delete global.fs;
  });

  describe("main function", () => {
    it("should create output file and set environment variables", async () => {
      const consoleSpy = vi.spyOn(console, "log").mockImplementation(() => {});

      // Execute the script
      await eval(`(async () => { ${setupScript} })()`);

      // Check that exportVariable was called with the correct pattern
      expect(mockCore.exportVariable).toHaveBeenCalledWith(
        "GITHUB_AW_SAFE_OUTPUTS",
        expect.stringMatching(/^\/tmp\/aw_output_[0-9a-f]{16}\.txt$/)
      );

      // Check that setOutput was called with the same file path
      const exportCall = mockCore.exportVariable.mock.calls[0];
      const outputCall = mockCore.setOutput.mock.calls[0];
      expect(outputCall[0]).toBe("output_file");
      expect(outputCall[1]).toBe(exportCall[1]);

      // Check that the file was actually created
      const outputFile = exportCall[1];
      expect(fs.existsSync(outputFile)).toBe(true);

      // Check that console.log was called with the correct message
      expect(consoleSpy).toHaveBeenCalledWith(
        "Created agentic output file:",
        outputFile
      );

      // Check that the file is empty (as expected)
      const content = fs.readFileSync(outputFile, "utf8");
      expect(content).toBe("");

      consoleSpy.mockRestore();
    });

    it("should create unique output file names on multiple runs", async () => {
      const consoleSpy = vi.spyOn(console, "log").mockImplementation(() => {});

      // Execute the script multiple times
      await eval(`(async () => { ${setupScript} })()`);
      const firstFile = mockCore.exportVariable.mock.calls[0][1];

      // Reset mocks for second run
      mockCore.exportVariable.mockClear();
      mockCore.setOutput.mockClear();

      await eval(`(async () => { ${setupScript} })()`);
      const secondFile = mockCore.exportVariable.mock.calls[0][1];

      // Files should be different
      expect(firstFile).not.toBe(secondFile);

      // Both files should exist
      expect(fs.existsSync(firstFile)).toBe(true);
      expect(fs.existsSync(secondFile)).toBe(true);

      consoleSpy.mockRestore();
    });

    it("should handle file creation failure gracefully", async () => {
      // Mock fs.writeFileSync to throw an error
      const originalWriteFileSync = fs.writeFileSync;
      fs.writeFileSync = vi.fn().mockImplementation(() => {
        throw new Error("Permission denied");
      });

      try {
        await eval(`(async () => { ${setupScript} })()`);
        expect.fail("Should have thrown an error");
      } catch (error) {
        expect(error.message).toBe("Permission denied");
      }

      // Restore original function
      fs.writeFileSync = originalWriteFileSync;
    });

    it("should verify file existence and throw error if file creation fails", async () => {
      // Mock fs.existsSync to return false (simulating failed file creation)
      const originalExistsSync = fs.existsSync;
      fs.existsSync = vi.fn().mockReturnValue(false);

      try {
        await eval(`(async () => { ${setupScript} })()`);
        expect.fail("Should have thrown an error");
      } catch (error) {
        expect(error.message).toMatch(
          /^Failed to create output file: \/tmp\/aw_output_[0-9a-f]{16}\.txt$/
        );
      }

      // Restore original function
      fs.existsSync = originalExistsSync;
    });
  });
});
