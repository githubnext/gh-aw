import { describe, it, expect, beforeEach, vi } from "vitest";
import fs from "fs";
import path from "path";

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
  toPlatformPath: vi.fn(),
  toPosixPath: vi.fn(),
  toWin32Path: vi.fn(),
  summary: {
    addRaw: vi.fn().mockReturnThis(),
    write: vi.fn().mockResolvedValue(),
  },
};

const mockExec = {
  exec: vi.fn(),
  getExecOutput: vi.fn(),
};

const mockContext = {
  eventName: "pull_request",
  repo: {
    owner: "testowner",
    repo: "testrepo",
  },
  payload: {
    repository: {
      default_branch: "main",
    },
  },
};

// Set up global variables
global.core = mockCore;
global.exec = mockExec;
global.context = mockContext;

describe("generate_git_patch.cjs", () => {
  let generateGitPatchModule;
  let tempDir;
  let tempSafeOutputsFile;

  beforeEach(() => {
    // Reset all mocks
    vi.clearAllMocks();

    // Create temp directory for test files
    tempDir = fs.mkdtempSync(path.join("/tmp", "generate-git-patch-test-"));
    tempSafeOutputsFile = path.join(tempDir, "safe-outputs.jsonl");

    // Load the module
    const modulePath = path.resolve(__dirname, "generate_git_patch.cjs");
    delete require.cache[modulePath];
    generateGitPatchModule = require("./generate_git_patch.cjs");
  });

  afterEach(() => {
    // Cleanup temp directory
    if (fs.existsSync(tempDir)) {
      fs.rmSync(tempDir, { recursive: true, force: true });
    }
  });

  describe("extractBranchFromSafeOutputs", () => {
    it("should extract branch from push_to_pull_request_branch entry", () => {
      const content = '{"type":"push_to_pull_request_branch","branch":"feature-branch","message":"test"}\n';
      fs.writeFileSync(tempSafeOutputsFile, content);

      const result = generateGitPatchModule.extractBranchFromSafeOutputs(tempSafeOutputsFile);

      expect(result).toBe("feature-branch");
      expect(mockCore.info).toHaveBeenCalledWith(
        expect.stringContaining("Found push_to_pull_request_branch line with branch: feature-branch")
      );
    });

    it("should extract branch from create_pull_request entry", () => {
      const content = '{"type":"create_pull_request","branch":"my-branch","title":"Test PR"}\n';
      fs.writeFileSync(tempSafeOutputsFile, content);

      const result = generateGitPatchModule.extractBranchFromSafeOutputs(tempSafeOutputsFile);

      expect(result).toBe("my-branch");
      expect(mockCore.info).toHaveBeenCalledWith(expect.stringContaining("Found create_pull_request line with branch: my-branch"));
    });

    it("should return empty string if file does not exist", () => {
      const result = generateGitPatchModule.extractBranchFromSafeOutputs("/nonexistent/path");

      expect(result).toBe("");
    });

    it("should return empty string if no branch found", () => {
      const content = '{"type":"other","message":"test"}\n';
      fs.writeFileSync(tempSafeOutputsFile, content);

      const result = generateGitPatchModule.extractBranchFromSafeOutputs(tempSafeOutputsFile);

      expect(result).toBe("");
    });

    it("should skip invalid JSON lines", () => {
      const content = 'invalid json\n{"type":"push_to_pull_request_branch","branch":"valid-branch"}\n';
      fs.writeFileSync(tempSafeOutputsFile, content);

      const result = generateGitPatchModule.extractBranchFromSafeOutputs(tempSafeOutputsFile);

      expect(result).toBe("valid-branch");
    });

    it("should return first branch found in multiple entries", () => {
      const content =
        '{"type":"push_to_pull_request_branch","branch":"first-branch"}\n' + '{"type":"create_pull_request","branch":"second-branch"}\n';
      fs.writeFileSync(tempSafeOutputsFile, content);

      const result = generateGitPatchModule.extractBranchFromSafeOutputs(tempSafeOutputsFile);

      expect(result).toBe("first-branch");
    });
  });

  describe("determineTargetBranch", () => {
    it("should use branch from safe-outputs if it exists locally", async () => {
      mockExec.exec.mockResolvedValueOnce(0); // git show-ref succeeds

      const result = await generateGitPatchModule.determineTargetBranch("existing-branch");

      expect(result).toBe("existing-branch");
      expect(mockCore.info).toHaveBeenCalledWith("Branch name from safe-outputs: existing-branch");
      expect(mockCore.info).toHaveBeenCalledWith("Branch existing-branch exists locally");
      expect(mockExec.exec).toHaveBeenCalledWith("git", ["show-ref", "--verify", "--quiet", "refs/heads/existing-branch"]);
    });

    it("should fall back to current branch if safe-outputs branch does not exist", async () => {
      mockExec.exec.mockRejectedValueOnce(new Error("Branch not found")); // git show-ref fails
      mockExec.getExecOutput.mockResolvedValueOnce({ stdout: "current-branch\n", stderr: "", exitCode: 0 });

      const result = await generateGitPatchModule.determineTargetBranch("nonexistent-branch");

      expect(result).toBe("current-branch");
      expect(mockCore.info).toHaveBeenCalledWith("Branch nonexistent-branch does not exist locally, falling back to current HEAD");
      expect(mockCore.info).toHaveBeenCalledWith("Using current branch: current-branch");
    });

    it("should use HEAD if in detached HEAD state", async () => {
      mockExec.getExecOutput.mockResolvedValueOnce({ stdout: "HEAD\n", stderr: "", exitCode: 0 });

      const result = await generateGitPatchModule.determineTargetBranch("");

      expect(result).toBe("HEAD");
      expect(mockCore.info).toHaveBeenCalledWith("No branch name found in safe-outputs, using current branch");
      expect(mockCore.warning).toHaveBeenCalledWith("Detached HEAD state, using HEAD directly");
    });

    it("should return empty string if git command fails", async () => {
      mockExec.getExecOutput.mockRejectedValueOnce(new Error("Git command failed"));

      const result = await generateGitPatchModule.determineTargetBranch("");

      expect(result).toBe("");
      expect(mockCore.error).toHaveBeenCalledWith(expect.stringContaining("Failed to get current branch"));
    });
  });

  describe("determineBaseRef", () => {
    it("should use merge-base for HEAD target", async () => {
      mockExec.exec.mockResolvedValueOnce(0); // git fetch
      mockExec.getExecOutput.mockResolvedValueOnce({ stdout: "abc123\n", stderr: "", exitCode: 0 });

      const result = await generateGitPatchModule.determineBaseRef("HEAD", "main");

      expect(result).toBe("abc123");
      expect(mockCore.info).toHaveBeenCalledWith("Default branch: main");
      expect(mockCore.info).toHaveBeenCalledWith("Using merge-base as base: abc123");
      expect(mockExec.exec).toHaveBeenCalledWith("git", ["fetch", "origin", "main"]);
      expect(mockExec.getExecOutput).toHaveBeenCalledWith("git", ["merge-base", "origin/main", "HEAD"]);
    });

    it("should use origin/targetBranch if it exists", async () => {
      mockExec.exec.mockResolvedValueOnce(0); // git show-ref succeeds

      const result = await generateGitPatchModule.determineBaseRef("feature-branch", "main");

      expect(result).toBe("origin/feature-branch");
      expect(mockCore.info).toHaveBeenCalledWith("Using origin/feature-branch as base for patch generation");
    });

    it("should fall back to merge-base if origin/targetBranch does not exist", async () => {
      mockExec.exec.mockRejectedValueOnce(new Error("Branch not found")); // git show-ref fails
      mockExec.exec.mockResolvedValueOnce(0); // git fetch
      mockExec.getExecOutput.mockResolvedValueOnce({ stdout: "def456\n", stderr: "", exitCode: 0 });

      const result = await generateGitPatchModule.determineBaseRef("feature-branch", "main");

      expect(result).toBe("def456");
      expect(mockCore.info).toHaveBeenCalledWith("origin/feature-branch does not exist, using merge-base with default branch");
      expect(mockCore.info).toHaveBeenCalledWith("Using merge-base as base: def456");
    });
  });

  describe("generatePatch", () => {
    it("should generate patch successfully", async () => {
      const patchPath = path.join(tempDir, "test.patch");
      const patchContent = "diff --git a/file.txt b/file.txt\n...";

      mockExec.getExecOutput.mockResolvedValueOnce({ stdout: patchContent, stderr: "", exitCode: 0 });

      const result = await generateGitPatchModule.generatePatch("origin/main", "feature-branch", patchPath);

      expect(result).toBe(true);
      expect(fs.existsSync(patchPath)).toBe(true);
      expect(fs.readFileSync(patchPath, "utf8")).toBe(patchContent);
      expect(mockCore.info).toHaveBeenCalledWith("Patch file created from: feature-branch (base: origin/main)");
    });

    it("should handle patch generation failure", async () => {
      const patchPath = path.join(tempDir, "test.patch");

      mockExec.getExecOutput.mockResolvedValueOnce({ stdout: "", stderr: "error", exitCode: 1 });

      const result = await generateGitPatchModule.generatePatch("origin/main", "feature-branch", patchPath);

      expect(result).toBe(false);
      expect(fs.existsSync(patchPath)).toBe(true);
      expect(fs.readFileSync(patchPath, "utf8")).toBe("Failed to generate patch from branch");
      expect(mockCore.warning).toHaveBeenCalledWith(expect.stringContaining("Failed to generate patch"));
    });

    it("should handle git command exception", async () => {
      const patchPath = path.join(tempDir, "test.patch");

      mockExec.getExecOutput.mockRejectedValueOnce(new Error("Git command failed"));

      const result = await generateGitPatchModule.generatePatch("origin/main", "feature-branch", patchPath);

      expect(result).toBe(false);
      expect(fs.existsSync(patchPath)).toBe(true);
      expect(mockCore.error).toHaveBeenCalledWith(expect.stringContaining("Failed to generate patch"));
    });
  });

  describe("addPatchToSummary", () => {
    it("should add patch to step summary", async () => {
      const patchPath = path.join(tempDir, "test.patch");
      const patchContent = "diff --git a/file.txt b/file.txt\nline1\nline2\nline3";
      fs.writeFileSync(patchPath, patchContent);

      await generateGitPatchModule.addPatchToSummary(patchPath);

      expect(mockCore.info).toHaveBeenCalledWith(expect.stringContaining("Patch file size:"));
      expect(mockCore.summary.addRaw).toHaveBeenCalled();
      expect(mockCore.summary.write).toHaveBeenCalled();
    });

    it("should handle long patches by truncating", async () => {
      const patchPath = path.join(tempDir, "test.patch");
      const lines = Array(600)
        .fill(0)
        .map((_, i) => `line ${i}`);
      const patchContent = lines.join("\n");
      fs.writeFileSync(patchPath, patchContent);

      await generateGitPatchModule.addPatchToSummary(patchPath);

      const summaryCall = mockCore.summary.addRaw.mock.calls[0][0];
      expect(summaryCall).toContain("...");
    });

    it("should do nothing if patch file does not exist", async () => {
      await generateGitPatchModule.addPatchToSummary("/nonexistent/patch");

      expect(mockCore.summary.addRaw).not.toHaveBeenCalled();
    });
  });

  describe("main", () => {
    it("should run complete patch generation workflow", async () => {
      const patchContent = "diff --git a/file.txt b/file.txt\n...";

      // Ensure /tmp/gh-aw directory exists
      const patchDir = "/tmp/gh-aw";
      if (!fs.existsSync(patchDir)) {
        fs.mkdirSync(patchDir, { recursive: true });
      }

      // Setup environment
      process.env.GH_AW_SAFE_OUTPUTS = tempSafeOutputsFile;
      process.env.GH_AW_DEFAULT_BRANCH = "main";

      // Create safe-outputs file
      fs.writeFileSync(tempSafeOutputsFile, '{"type":"push_to_pull_request_branch","branch":"feature-branch"}\n');

      // Mock git operations
      mockExec.exec.mockResolvedValue(0);
      mockExec.getExecOutput.mockResolvedValueOnce({ stdout: patchContent, stderr: "", exitCode: 0 });

      await generateGitPatchModule.main();

      expect(mockExec.exec).toHaveBeenCalledWith("git", ["status"]);
      expect(mockCore.info).toHaveBeenCalledWith(expect.stringContaining("Generating patch for:"));

      // Cleanup
      delete process.env.GH_AW_SAFE_OUTPUTS;
      delete process.env.GH_AW_DEFAULT_BRANCH;
      if (fs.existsSync("/tmp/gh-aw/aw.patch")) {
        fs.unlinkSync("/tmp/gh-aw/aw.patch");
      }
    });

    it("should handle no target branch gracefully", async () => {
      process.env.GH_AW_SAFE_OUTPUTS = "";

      mockExec.exec.mockResolvedValue(0);
      mockExec.getExecOutput.mockRejectedValueOnce(new Error("Git failed"));

      await generateGitPatchModule.main();

      expect(mockCore.info).toHaveBeenCalledWith("No target branch determined, no patch generation");

      delete process.env.GH_AW_SAFE_OUTPUTS;
    });
  });
});
