/**
 * Test Suite: messages.cjs
 *
 * Tests for the safe-output messages module functionality including:
 * - Environment variable parsing (GH_AW_SAFE_OUTPUT_MESSAGES)
 * - Template rendering with placeholder replacement
 * - Footer message generation (default and custom)
 * - Installation instructions generation
 * - Staged mode title and description generation
 * - Caching behavior
 */
import { describe, it, expect, beforeEach, vi } from "vitest";

// Mock core for GitHub Actions environment
const mockCore = {
  debug: vi.fn(),
  info: vi.fn(),
  warning: vi.fn(),
  error: vi.fn(),
};

// Set up global mocks
global.core = mockCore;

describe("messages.cjs", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    // Clear environment variable before each test
    delete process.env.GH_AW_SAFE_OUTPUT_MESSAGES;
    // Clear cache by reimporting
    vi.resetModules();
  });

  describe("getMessages", () => {
    it("should return null when env var is not set", async () => {
      const { getMessages, clearMessagesCache } = await import("./messages.cjs");
      clearMessagesCache();
      const result = getMessages();
      expect(result).toBeNull();
    });

    it("should parse valid JSON config with PascalCase keys (Go struct)", async () => {
      process.env.GH_AW_SAFE_OUTPUT_MESSAGES = JSON.stringify({
        Footer: "> Custom footer by [{workflow_name}]({run_url})",
        FooterInstall: "> Custom install: `gh aw add {workflow_source}`",
        StagedTitle: "## Custom Preview: {operation}",
        StagedDescription: "Preview of {operation}:",
      });

      const { getMessages, clearMessagesCache } = await import("./messages.cjs");
      clearMessagesCache();
      const result = getMessages();

      expect(result).toEqual({
        footer: "> Custom footer by [{workflow_name}]({run_url})",
        footerInstall: "> Custom install: `gh aw add {workflow_source}`",
        stagedTitle: "## Custom Preview: {operation}",
        stagedDescription: "Preview of {operation}:",
      });
    });

    it("should parse valid JSON config with camelCase keys", async () => {
      process.env.GH_AW_SAFE_OUTPUT_MESSAGES = JSON.stringify({
        footer: "> Custom footer",
        footerInstall: "> Custom install",
      });

      const { getMessages, clearMessagesCache } = await import("./messages.cjs");
      clearMessagesCache();
      const result = getMessages();

      expect(result.footer).toBe("> Custom footer");
      expect(result.footerInstall).toBe("> Custom install");
    });

    it("should handle invalid JSON gracefully", async () => {
      process.env.GH_AW_SAFE_OUTPUT_MESSAGES = "not valid json";

      const { getMessages, clearMessagesCache } = await import("./messages.cjs");
      clearMessagesCache();
      const result = getMessages();

      expect(result).toBeNull();
      expect(mockCore.warning).toHaveBeenCalledWith(expect.stringContaining("Failed to parse GH_AW_SAFE_OUTPUT_MESSAGES"));
    });

    it("should cache results after first call", async () => {
      process.env.GH_AW_SAFE_OUTPUT_MESSAGES = JSON.stringify({
        Footer: "cached value",
      });

      const { getMessages, clearMessagesCache } = await import("./messages.cjs");
      clearMessagesCache();

      const result1 = getMessages();
      process.env.GH_AW_SAFE_OUTPUT_MESSAGES = JSON.stringify({
        Footer: "new value",
      });
      const result2 = getMessages();

      expect(result1.footer).toBe("cached value");
      expect(result2.footer).toBe("cached value"); // Should still be cached
    });

    it("should clear cache when clearMessagesCache is called", async () => {
      process.env.GH_AW_SAFE_OUTPUT_MESSAGES = JSON.stringify({
        Footer: "first value",
      });

      const { getMessages, clearMessagesCache } = await import("./messages.cjs");
      clearMessagesCache();

      const result1 = getMessages();
      expect(result1.footer).toBe("first value");

      process.env.GH_AW_SAFE_OUTPUT_MESSAGES = JSON.stringify({
        Footer: "second value",
      });
      clearMessagesCache();

      const result2 = getMessages();
      expect(result2.footer).toBe("second value");
    });
  });

  describe("renderTemplate", () => {
    it("should replace simple placeholders", async () => {
      const { renderTemplate } = await import("./messages.cjs");

      const result = renderTemplate("Hello {name}!", { name: "World" });
      expect(result).toBe("Hello World!");
    });

    it("should replace multiple placeholders", async () => {
      const { renderTemplate } = await import("./messages.cjs");

      const result = renderTemplate("{greeting} {name}, you have {count} messages", {
        greeting: "Hello",
        name: "User",
        count: 5,
      });
      expect(result).toBe("Hello User, you have 5 messages");
    });

    it("should leave unknown placeholders unchanged", async () => {
      const { renderTemplate } = await import("./messages.cjs");

      const result = renderTemplate("Hello {name}, {unknown} placeholder", { name: "User" });
      expect(result).toBe("Hello User, {unknown} placeholder");
    });

    it("should handle snake_case placeholders", async () => {
      const { renderTemplate } = await import("./messages.cjs");

      const result = renderTemplate("{workflow_name} at {run_url}", {
        workflow_name: "My Workflow",
        run_url: "https://example.com",
      });
      expect(result).toBe("My Workflow at https://example.com");
    });

    it("should handle numbers as values", async () => {
      const { renderTemplate } = await import("./messages.cjs");

      const result = renderTemplate("Issue #{issue_number}", { issue_number: 42 });
      expect(result).toBe("Issue #42");
    });

    it("should handle undefined values by keeping placeholder", async () => {
      const { renderTemplate } = await import("./messages.cjs");

      const result = renderTemplate("Value: {value}", { value: undefined });
      expect(result).toBe("Value: {value}");
    });
  });

  describe("getFooterMessage", () => {
    it("should return default footer when no custom config", async () => {
      const { getFooterMessage, clearMessagesCache } = await import("./messages.cjs");
      clearMessagesCache();

      const result = getFooterMessage({
        workflowName: "Test Workflow",
        runUrl: "https://github.com/test/repo/actions/runs/123",
      });

      expect(result).toBe("> AI generated by [Test Workflow](https://github.com/test/repo/actions/runs/123)");
    });

    it("should append triggering number when provided", async () => {
      const { getFooterMessage, clearMessagesCache } = await import("./messages.cjs");
      clearMessagesCache();

      const result = getFooterMessage({
        workflowName: "Test Workflow",
        runUrl: "https://github.com/test/repo/actions/runs/123",
        triggeringNumber: 42,
      });

      expect(result).toBe("> AI generated by [Test Workflow](https://github.com/test/repo/actions/runs/123) for #42");
    });

    it("should use custom footer template", async () => {
      process.env.GH_AW_SAFE_OUTPUT_MESSAGES = JSON.stringify({
        Footer: "> Custom: [{workflow_name}]({run_url})",
      });

      const { getFooterMessage, clearMessagesCache } = await import("./messages.cjs");
      clearMessagesCache();

      const result = getFooterMessage({
        workflowName: "Custom Workflow",
        runUrl: "https://example.com/run/456",
      });

      expect(result).toBe("> Custom: [Custom Workflow](https://example.com/run/456)");
    });

    it("should support both snake_case and camelCase in custom templates", async () => {
      process.env.GH_AW_SAFE_OUTPUT_MESSAGES = JSON.stringify({
        Footer: "> {workflowName} ({workflow_name})",
      });

      const { getFooterMessage, clearMessagesCache } = await import("./messages.cjs");
      clearMessagesCache();

      const result = getFooterMessage({
        workflowName: "Test",
        runUrl: "https://example.com",
      });

      expect(result).toBe("> Test (Test)");
    });
  });

  describe("getFooterInstallMessage", () => {
    it("should return empty string when no workflow source", async () => {
      const { getFooterInstallMessage, clearMessagesCache } = await import("./messages.cjs");
      clearMessagesCache();

      const result = getFooterInstallMessage({
        workflowName: "Test",
        runUrl: "https://example.com",
      });

      expect(result).toBe("");
    });

    it("should return default install message when source is provided", async () => {
      const { getFooterInstallMessage, clearMessagesCache } = await import("./messages.cjs");
      clearMessagesCache();

      const result = getFooterInstallMessage({
        workflowName: "Test",
        runUrl: "https://example.com",
        workflowSource: "owner/repo/workflow.md@main",
        workflowSourceUrl: "https://github.com/owner/repo",
      });

      expect(result).toContain("gh aw add owner/repo/workflow.md@main");
      expect(result).toContain("usage guide");
    });

    it("should use custom install template", async () => {
      process.env.GH_AW_SAFE_OUTPUT_MESSAGES = JSON.stringify({
        FooterInstall: "> Install: `gh aw add {workflow_source}`",
      });

      const { getFooterInstallMessage, clearMessagesCache } = await import("./messages.cjs");
      clearMessagesCache();

      const result = getFooterInstallMessage({
        workflowName: "Test",
        runUrl: "https://example.com",
        workflowSource: "owner/repo/workflow.md@main",
        workflowSourceUrl: "https://github.com/owner/repo",
      });

      expect(result).toBe("> Install: `gh aw add owner/repo/workflow.md@main`");
    });
  });

  describe("generateFooterWithMessages", () => {
    it("should generate complete default footer", async () => {
      const { generateFooterWithMessages, clearMessagesCache } = await import("./messages.cjs");
      clearMessagesCache();

      const result = generateFooterWithMessages(
        "Test Workflow",
        "https://github.com/test/repo/actions/runs/123",
        "",
        "",
        undefined,
        undefined,
        undefined
      );

      expect(result).toContain("> AI generated by [Test Workflow]");
      expect(result).toContain("https://github.com/test/repo/actions/runs/123");
    });

    it("should include triggering issue number", async () => {
      const { generateFooterWithMessages, clearMessagesCache } = await import("./messages.cjs");
      clearMessagesCache();

      const result = generateFooterWithMessages(
        "Test Workflow",
        "https://github.com/test/repo/actions/runs/123",
        "",
        "",
        42,
        undefined,
        undefined
      );

      expect(result).toContain("for #42");
    });

    it("should include triggering PR number when no issue", async () => {
      const { generateFooterWithMessages, clearMessagesCache } = await import("./messages.cjs");
      clearMessagesCache();

      const result = generateFooterWithMessages(
        "Test Workflow",
        "https://github.com/test/repo/actions/runs/123",
        "",
        "",
        undefined,
        99,
        undefined
      );

      expect(result).toContain("for #99");
    });

    it("should include triggering discussion number", async () => {
      const { generateFooterWithMessages, clearMessagesCache } = await import("./messages.cjs");
      clearMessagesCache();

      const result = generateFooterWithMessages(
        "Test Workflow",
        "https://github.com/test/repo/actions/runs/123",
        "",
        "",
        undefined,
        undefined,
        7
      );

      expect(result).toContain("for #discussion #7");
    });

    it("should include installation instructions when source is provided", async () => {
      const { generateFooterWithMessages, clearMessagesCache } = await import("./messages.cjs");
      clearMessagesCache();

      const result = generateFooterWithMessages(
        "Test Workflow",
        "https://github.com/test/repo/actions/runs/123",
        "owner/repo/workflow.md@main",
        "https://github.com/owner/repo",
        undefined,
        undefined,
        undefined
      );

      expect(result).toContain("gh aw add owner/repo/workflow.md@main");
    });
  });

  describe("getStagedTitle", () => {
    it("should return default staged title", async () => {
      const { getStagedTitle, clearMessagesCache } = await import("./messages.cjs");
      clearMessagesCache();

      const result = getStagedTitle({ operation: "Create Issues" });

      expect(result).toBe("## 🎭 Staged Mode: Create Issues Preview");
    });

    it("should use custom staged title template", async () => {
      process.env.GH_AW_SAFE_OUTPUT_MESSAGES = JSON.stringify({
        StagedTitle: "## 🔍 Preview: {operation}",
      });

      const { getStagedTitle, clearMessagesCache } = await import("./messages.cjs");
      clearMessagesCache();

      const result = getStagedTitle({ operation: "Add Comments" });

      expect(result).toBe("## 🔍 Preview: Add Comments");
    });
  });

  describe("getStagedDescription", () => {
    it("should return default staged description", async () => {
      const { getStagedDescription, clearMessagesCache } = await import("./messages.cjs");
      clearMessagesCache();

      const result = getStagedDescription({ operation: "Create Issues" });

      expect(result).toBe("The following would be created if staged mode was disabled:");
    });

    it("should use custom staged description template", async () => {
      process.env.GH_AW_SAFE_OUTPUT_MESSAGES = JSON.stringify({
        StagedDescription: "Preview of {operation} - nothing will be created:",
      });

      const { getStagedDescription, clearMessagesCache } = await import("./messages.cjs");
      clearMessagesCache();

      const result = getStagedDescription({ operation: "pull requests" });

      expect(result).toBe("Preview of pull requests - nothing will be created:");
    });
  });
});
