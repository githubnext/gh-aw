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

// Extract the functions
const interpolateVariablesMatch = interpolatePromptScript.match(
  /function interpolateVariables\(content, variables\)\s*{[\s\S]*?return result;[\s\S]*?}/
);

const isTruthyMatch = interpolatePromptScript.match(/function isTruthy\(expr\)\s*{[\s\S]*?return[\s\S]*?;[\s\S]*?}/);

const renderMarkdownTemplateMatch = interpolatePromptScript.match(
  /function renderMarkdownTemplate\(markdown\)\s*{[\s\S]*?return[\s\S]*?;[\s\S]*?}/
);

if (!interpolateVariablesMatch) {
  throw new Error("Could not extract interpolateVariables function from interpolate_prompt.cjs");
}

if (!isTruthyMatch) {
  throw new Error("Could not extract isTruthy function from interpolate_prompt.cjs");
}

if (!renderMarkdownTemplateMatch) {
  throw new Error("Could not extract renderMarkdownTemplate function from interpolate_prompt.cjs");
}

// eslint-disable-next-line no-eval
const interpolateVariables = eval(`(${interpolateVariablesMatch[0]})`);
// eslint-disable-next-line no-eval
const isTruthy = eval(`(${isTruthyMatch[0]})`);
// eslint-disable-next-line no-eval
const renderMarkdownTemplate = eval(`(${renderMarkdownTemplateMatch[0]})`);

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

  describe("isTruthy", () => {
    it("should return false for empty string", () => {
      expect(isTruthy("")).toBe(false);
    });

    it('should return false for "false"', () => {
      expect(isTruthy("false")).toBe(false);
      expect(isTruthy("FALSE")).toBe(false);
      expect(isTruthy("False")).toBe(false);
    });

    it('should return false for "0"', () => {
      expect(isTruthy("0")).toBe(false);
    });

    it('should return false for "null"', () => {
      expect(isTruthy("null")).toBe(false);
      expect(isTruthy("NULL")).toBe(false);
    });

    it('should return false for "undefined"', () => {
      expect(isTruthy("undefined")).toBe(false);
      expect(isTruthy("UNDEFINED")).toBe(false);
    });

    it('should return true for "true"', () => {
      expect(isTruthy("true")).toBe(true);
      expect(isTruthy("TRUE")).toBe(true);
    });

    it("should return true for any non-falsy string", () => {
      expect(isTruthy("yes")).toBe(true);
      expect(isTruthy("1")).toBe(true);
      expect(isTruthy("hello")).toBe(true);
    });

    it("should trim whitespace", () => {
      expect(isTruthy("  false  ")).toBe(false);
      expect(isTruthy("  true  ")).toBe(true);
    });
  });

  describe("renderMarkdownTemplate", () => {
    it("should keep content in truthy blocks", () => {
      const input = "{{#if true}}\nHello\n{{/if}}";
      const output = renderMarkdownTemplate(input);
      expect(output).toBe("\nHello\n");
    });

    it("should remove content in falsy blocks", () => {
      const input = "{{#if false}}\nHello\n{{/if}}";
      const output = renderMarkdownTemplate(input);
      expect(output).toBe("");
    });

    it("should process multiple blocks", () => {
      const input = "{{#if true}}\nKeep this\n{{/if}}\n{{#if false}}\nRemove this\n{{/if}}";
      const output = renderMarkdownTemplate(input);
      expect(output).toBe("\nKeep this\n\n");
    });

    it("should handle nested content", () => {
      const input = `# Title

{{#if true}}
## Section 1
This should be kept.
{{/if}}

{{#if false}}
## Section 2
This should be removed.
{{/if}}

## Section 3
This is always visible.`;

      const expected = `# Title


## Section 1
This should be kept.




## Section 3
This is always visible.`;

      const output = renderMarkdownTemplate(input);
      expect(output).toBe(expected);
    });

    it("should leave content without conditionals unchanged", () => {
      const input = "# Normal Markdown\n\nNo conditionals here.";
      const output = renderMarkdownTemplate(input);
      expect(output).toBe(input);
    });

    it("should handle conditionals with various expressions", () => {
      const input1 = "{{#if 1}}\nKeep\n{{/if}}";
      expect(renderMarkdownTemplate(input1)).toBe("\nKeep\n");

      const input2 = "{{#if 0}}\nRemove\n{{/if}}";
      expect(renderMarkdownTemplate(input2)).toBe("");

      const input3 = "{{#if null}}\nRemove\n{{/if}}";
      expect(renderMarkdownTemplate(input3)).toBe("");

      const input4 = "{{#if undefined}}\nRemove\n{{/if}}";
      expect(renderMarkdownTemplate(input4)).toBe("");
    });

    it("should preserve markdown formatting inside blocks", () => {
      const input = `{{#if true}}
## Header
- List item 1
- List item 2

\`\`\`javascript
const x = 1;
\`\`\`
{{/if}}`;

      const expected = `
## Header
- List item 1
- List item 2

\`\`\`javascript
const x = 1;
\`\`\`
`;

      const output = renderMarkdownTemplate(input);
      expect(output).toBe(expected);
    });

    it("should handle whitespace in conditionals", () => {
      const input1 = "{{#if   true  }}\nKeep\n{{/if}}";
      expect(renderMarkdownTemplate(input1)).toBe("\nKeep\n");

      const input2 = "{{#if\ttrue\t}}\nKeep\n{{/if}}";
      expect(renderMarkdownTemplate(input2)).toBe("\nKeep\n");
    });
  });

  describe("combined interpolation and template rendering", () => {
    it("should interpolate variables and then render templates", () => {
      const content = "Repo: ${GH_AW_EXPR_REPO}\n{{#if true}}\nShow this\n{{/if}}";
      const variables = { GH_AW_EXPR_REPO: "github/test-repo" };

      // First interpolate
      let result = interpolateVariables(content, variables);
      expect(result).toBe("Repo: github/test-repo\n{{#if true}}\nShow this\n{{/if}}");

      // Then render template
      result = renderMarkdownTemplate(result);
      expect(result).toBe("Repo: github/test-repo\n\nShow this\n");
    });

    it("should handle template conditionals that depend on interpolated values", () => {
      const content = "${GH_AW_EXPR_CONDITION}\n{{#if ${GH_AW_EXPR_CONDITION}}}\nShow this\n{{/if}}";
      const variables = { GH_AW_EXPR_CONDITION: "true" };

      // First interpolate
      let result = interpolateVariables(content, variables);
      expect(result).toBe("true\n{{#if true}}\nShow this\n{{/if}}");

      // Then render template
      result = renderMarkdownTemplate(result);
      expect(result).toBe("true\n\nShow this\n");
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
