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

describe("normalize_branch.cjs", () => {
  let normalizeBranchScript;
  let originalEnv;

  beforeEach(() => {
    // Reset all mocks
    vi.clearAllMocks();

    // Store original environment
    originalEnv = {
      GITHUB_AW_ASSETS_BRANCH: process.env.GITHUB_AW_ASSETS_BRANCH,
    };

    // Read the script content
    const scriptPath = path.join(process.cwd(), "normalize_branch.cjs");
    normalizeBranchScript = fs.readFileSync(scriptPath, "utf8");
  });

  afterEach(() => {
    // Restore original environment
    if (originalEnv.GITHUB_AW_ASSETS_BRANCH !== undefined) {
      process.env.GITHUB_AW_ASSETS_BRANCH = originalEnv.GITHUB_AW_ASSETS_BRANCH;
    } else {
      delete process.env.GITHUB_AW_ASSETS_BRANCH;
    }
  });

  describe("when GITHUB_AW_ASSETS_BRANCH is not set", () => {
    it("should warn and skip normalization", async () => {
      delete process.env.GITHUB_AW_ASSETS_BRANCH;

      await eval(`(async () => { ${normalizeBranchScript} })()`);

      expect(mockCore.warning).toHaveBeenCalledWith(expect.stringContaining("not set"));
      expect(mockCore.exportVariable).not.toHaveBeenCalled();
      expect(mockCore.setOutput).not.toHaveBeenCalled();
    });

    it("should warn and skip normalization when empty string", async () => {
      process.env.GITHUB_AW_ASSETS_BRANCH = "";

      await eval(`(async () => { ${normalizeBranchScript} })()`);

      expect(mockCore.warning).toHaveBeenCalledWith(expect.stringContaining("not set"));
      expect(mockCore.exportVariable).not.toHaveBeenCalled();
    });

    it("should warn and skip normalization when only whitespace", async () => {
      process.env.GITHUB_AW_ASSETS_BRANCH = "   ";

      await eval(`(async () => { ${normalizeBranchScript} })()`);

      expect(mockCore.warning).toHaveBeenCalledWith(expect.stringContaining("not set"));
      expect(mockCore.exportVariable).not.toHaveBeenCalled();
    });
  });

  describe("when branch name is valid", () => {
    it("should keep valid branch name unchanged", async () => {
      process.env.GITHUB_AW_ASSETS_BRANCH = "assets/my-workflow";

      await eval(`(async () => { ${normalizeBranchScript} })()`);

      expect(mockCore.exportVariable).toHaveBeenCalledWith("GITHUB_AW_ASSETS_BRANCH", "assets/my-workflow");
      expect(mockCore.setOutput).toHaveBeenCalledWith("normalized_branch", "assets/my-workflow");
    });

    it("should keep alphanumeric and valid special characters", async () => {
      process.env.GITHUB_AW_ASSETS_BRANCH = "feature/ABC-123_test.v1";

      await eval(`(async () => { ${normalizeBranchScript} })()`);

      expect(mockCore.exportVariable).toHaveBeenCalledWith("GITHUB_AW_ASSETS_BRANCH", "feature/ABC-123_test.v1");
    });
  });

  describe("when branch name contains invalid characters", () => {
    it("should replace GitHub expression with dashes", async () => {
      process.env.GITHUB_AW_ASSETS_BRANCH = "assets/${{ github.workflow }}";

      await eval(`(async () => { ${normalizeBranchScript} })()`);

      expect(mockCore.exportVariable).toHaveBeenCalledWith("GITHUB_AW_ASSETS_BRANCH", "assets/-github.workflow");
      expect(mockCore.info).toHaveBeenCalledWith(expect.stringContaining("assets/-github.workflow"));
    });

    it("should replace multiple invalid characters with single dash", async () => {
      process.env.GITHUB_AW_ASSETS_BRANCH = "branch@@##name";

      await eval(`(async () => { ${normalizeBranchScript} })()`);

      expect(mockCore.exportVariable).toHaveBeenCalledWith("GITHUB_AW_ASSETS_BRANCH", "branch-name");
    });

    it("should replace spaces with dashes", async () => {
      process.env.GITHUB_AW_ASSETS_BRANCH = "my branch name";

      await eval(`(async () => { ${normalizeBranchScript} })()`);

      expect(mockCore.exportVariable).toHaveBeenCalledWith("GITHUB_AW_ASSETS_BRANCH", "my-branch-name");
    });

    it("should handle workflow name with spaces", async () => {
      process.env.GITHUB_AW_ASSETS_BRANCH = "assets/Poem Bot - A Creative Agentic Workflow";

      await eval(`(async () => { ${normalizeBranchScript} })()`);

      expect(mockCore.exportVariable).toHaveBeenCalledWith("GITHUB_AW_ASSETS_BRANCH", "assets/Poem-Bot---A-Creative-Agentic-Workflow");
    });
  });

  describe("when branch name has leading or trailing dashes", () => {
    it("should remove leading dashes", async () => {
      process.env.GITHUB_AW_ASSETS_BRANCH = "---branch-name";

      await eval(`(async () => { ${normalizeBranchScript} })()`);

      expect(mockCore.exportVariable).toHaveBeenCalledWith("GITHUB_AW_ASSETS_BRANCH", "branch-name");
    });

    it("should remove trailing dashes", async () => {
      process.env.GITHUB_AW_ASSETS_BRANCH = "branch-name---";

      await eval(`(async () => { ${normalizeBranchScript} })()`);

      expect(mockCore.exportVariable).toHaveBeenCalledWith("GITHUB_AW_ASSETS_BRANCH", "branch-name");
    });

    it("should remove both leading and trailing dashes", async () => {
      process.env.GITHUB_AW_ASSETS_BRANCH = "---branch-name---";

      await eval(`(async () => { ${normalizeBranchScript} })()`);

      expect(mockCore.exportVariable).toHaveBeenCalledWith("GITHUB_AW_ASSETS_BRANCH", "branch-name");
    });
  });

  describe("when branch name exceeds 128 characters", () => {
    it("should truncate to 128 characters", async () => {
      const longName = "a" + "b".repeat(150);
      process.env.GITHUB_AW_ASSETS_BRANCH = longName;

      await eval(`(async () => { ${normalizeBranchScript} })()`);

      const expectedName = "a" + "b".repeat(127); // 128 total
      expect(mockCore.exportVariable).toHaveBeenCalledWith("GITHUB_AW_ASSETS_BRANCH", expectedName);
      expect(expectedName.length).toBe(128);
    });

    it("should remove trailing dashes after truncation", async () => {
      const nameWithDashAtEnd = "a".repeat(127) + "-" + "b".repeat(10);
      process.env.GITHUB_AW_ASSETS_BRANCH = nameWithDashAtEnd;

      await eval(`(async () => { ${normalizeBranchScript} })()`);

      const expectedName = "a".repeat(127); // Truncated at 128, then trailing dash removed
      expect(mockCore.exportVariable).toHaveBeenCalledWith("GITHUB_AW_ASSETS_BRANCH", expectedName);
      expect(expectedName.length).toBe(127);
    });
  });

  describe("edge cases", () => {
    it("should handle only invalid characters", async () => {
      process.env.GITHUB_AW_ASSETS_BRANCH = "@#$%";

      await eval(`(async () => { ${normalizeBranchScript} })()`);

      // All invalid characters replaced with dash, then dashes trimmed = empty string
      expect(mockCore.exportVariable).toHaveBeenCalledWith("GITHUB_AW_ASSETS_BRANCH", "");
    });

    it("should handle mixed valid and invalid characters", async () => {
      process.env.GITHUB_AW_ASSETS_BRANCH = "test/branch-123_ABC@#$xyz";

      await eval(`(async () => { ${normalizeBranchScript} })()`);

      expect(mockCore.exportVariable).toHaveBeenCalledWith("GITHUB_AW_ASSETS_BRANCH", "test/branch-123_ABC-xyz");
    });
  });
});
