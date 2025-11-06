// Tests for interpolate_prompt.cjs
import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import fs from "fs";
import path from "path";
import os from "os";
import { fileURLToPath } from "url";

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);

// Mock the core module
const core = {
  info: vi.fn(),
  setFailed: vi.fn(),
};
global.core = core;

// Import the interpolation script
const interpolatePromptScript = fs.readFileSync(path.join(__dirname, "interpolate_prompt.cjs"), "utf8");

// Extract the interpolateVariables function
const interpolateVariablesMatch = interpolatePromptScript.match(
  /function interpolateVariables\(content, variables\)\s*{[\s\S]*?return result;[\s\S]*?}/
);

if (!interpolateVariablesMatch) {
  throw new Error("Could not extract interpolateVariables function from interpolate_prompt.cjs");
}

// eslint-disable-next-line no-eval
const interpolateVariables = eval(`(${interpolateVariablesMatch[0]})`);

describe("interpolate_prompt", () => {
  describe("interpolateVariables", () => {
    it("should interpolate single variable", () => {
      const content = "Repository: ${GH_AW_EXPR_TEST123}";
      const variables = { GH_AW_EXPR_TEST123: "github/test-repo" };
      const result = interpolateVariables(content, variables);
      expect(result).toBe("Repository: github/test-repo");
    });

    it("should interpolate multiple variables", () => {
      const content = "Repo: ${GH_AW_EXPR_REPO}, Actor: ${GH_AW_EXPR_ACTOR}, Issue: ${GH_AW_EXPR_ISSUE}";
      const variables = {
        GH_AW_EXPR_REPO: "github/test-repo",
        GH_AW_EXPR_ACTOR: "testuser",
        GH_AW_EXPR_ISSUE: "123",
      };
      const result = interpolateVariables(content, variables);
      expect(result).toBe("Repo: github/test-repo, Actor: testuser, Issue: 123");
    });

    it("should handle multiline content", () => {
      const content = `# Test Workflow

Repository: \${GH_AW_EXPR_REPO}
Actor: \${GH_AW_EXPR_ACTOR}

Some other content here.`;
      const variables = {
        GH_AW_EXPR_REPO: "github/test-repo",
        GH_AW_EXPR_ACTOR: "testuser",
      };
      const result = interpolateVariables(content, variables);
      expect(result).toContain("Repository: github/test-repo");
      expect(result).toContain("Actor: testuser");
    });

    it("should handle empty variable values", () => {
      const content = "Value: ${GH_AW_EXPR_EMPTY}";
      const variables = { GH_AW_EXPR_EMPTY: "" };
      const result = interpolateVariables(content, variables);
      expect(result).toBe("Value: ");
    });

    it("should replace all occurrences of the same variable", () => {
      const content = "Repo: ${GH_AW_EXPR_REPO}, Same repo: ${GH_AW_EXPR_REPO}";
      const variables = { GH_AW_EXPR_REPO: "github/test-repo" };
      const result = interpolateVariables(content, variables);
      expect(result).toBe("Repo: github/test-repo, Same repo: github/test-repo");
    });

    it("should not modify content without variables", () => {
      const content = "No variables here";
      const variables = {};
      const result = interpolateVariables(content, variables);
      expect(result).toBe("No variables here");
    });

    it("should handle content with literal dollar signs", () => {
      const content = "Price: $100, Repo: ${GH_AW_EXPR_REPO}";
      const variables = { GH_AW_EXPR_REPO: "github/test-repo" };
      const result = interpolateVariables(content, variables);
      expect(result).toBe("Price: $100, Repo: github/test-repo");
    });
  });

  describe("main function integration", () => {
    let tmpDir;
    let promptPath;
    let originalEnv;

    beforeEach(() => {
      // Save original environment
      originalEnv = { ...process.env };

      // Create a temporary directory
      tmpDir = fs.mkdtempSync(path.join(os.tmpdir(), "interpolate-test-"));
      promptPath = path.join(tmpDir, "prompt.txt");

      // Set up environment
      process.env.GH_AW_PROMPT = promptPath;

      // Clear mocks
      core.info.mockClear();
      core.setFailed.mockClear();
    });

    afterEach(() => {
      // Clean up
      if (tmpDir && fs.existsSync(tmpDir)) {
        fs.rmSync(tmpDir, { recursive: true, force: true });
      }

      // Restore environment
      process.env = originalEnv;
    });

    it("should fail when GH_AW_PROMPT is not set", () => {
      delete process.env.GH_AW_PROMPT;

      // Extract and execute main function
      const mainMatch = interpolatePromptScript.match(/async function main\(\)\s*{[\s\S]*?^}/m);
      if (!mainMatch) {
        throw new Error("Could not extract main function");
      }
      // eslint-disable-next-line no-eval
      const main = eval(`(${mainMatch[0]})`);

      main();

      expect(core.setFailed).toHaveBeenCalledWith("GH_AW_PROMPT environment variable is not set");
    });
  });
});

