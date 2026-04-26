/**
 * Test Suite: messages_core.cjs
 *
 * Tests for the core message utilities module including:
 * - Template rendering with placeholder replacement
 * - Template rendering from file
 * - Snake_case conversion with camelCase compatibility
 * - Messages config parsing from environment variable
 */
import { describe, it, expect, beforeEach, vi } from "vitest";
import fs from "fs";
import os from "os";
import path from "path";

const mockCore = {
  debug: vi.fn(),
  info: vi.fn(),
  warning: vi.fn(),
  error: vi.fn(),
};

global.core = mockCore;

describe("messages_core.cjs", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    vi.resetModules();
    delete process.env.GH_AW_SAFE_OUTPUT_MESSAGES;
    delete process.env.GH_AW_PROMPTS_DIR;
    delete process.env.RUNNER_TEMP;
  });

  describe("renderTemplate", () => {
    it("should replace a single placeholder", async () => {
      const { renderTemplate } = await import("./messages_core.cjs?" + Date.now());
      const result = renderTemplate("Hello, {name}!", { name: "World" });
      expect(result).toBe("Hello, World!");
    });

    it("should replace multiple placeholders", async () => {
      const { renderTemplate } = await import("./messages_core.cjs?" + Date.now());
      const result = renderTemplate("{greeting}, {name}! Run: {run_url}", {
        greeting: "Hello",
        name: "Alice",
        run_url: "https://github.com/actions/runs/123",
      });
      expect(result).toBe("Hello, Alice! Run: https://github.com/actions/runs/123");
    });

    it("should keep placeholder unchanged when key is missing from context", async () => {
      const { renderTemplate } = await import("./messages_core.cjs?" + Date.now());
      const result = renderTemplate("Hello, {name}! {unknown}", { name: "World" });
      expect(result).toBe("Hello, World! {unknown}");
    });

    it("should keep placeholder unchanged when value is undefined", async () => {
      const { renderTemplate } = await import("./messages_core.cjs?" + Date.now());
      const result = renderTemplate("{key}", { key: undefined });
      expect(result).toBe("{key}");
    });

    it("should coerce numeric values to strings", async () => {
      const { renderTemplate } = await import("./messages_core.cjs?" + Date.now());
      const result = renderTemplate("Issue #{number}", { number: 42 });
      expect(result).toBe("Issue #42");
    });

    it("should coerce boolean values to strings", async () => {
      const { renderTemplate } = await import("./messages_core.cjs?" + Date.now());
      const result = renderTemplate("Active: {active}", { active: true });
      expect(result).toBe("Active: true");
    });

    it("should return template unchanged when no placeholders present", async () => {
      const { renderTemplate } = await import("./messages_core.cjs?" + Date.now());
      const result = renderTemplate("No placeholders here.", {});
      expect(result).toBe("No placeholders here.");
    });

    it("should handle empty template", async () => {
      const { renderTemplate } = await import("./messages_core.cjs?" + Date.now());
      const result = renderTemplate("", { key: "value" });
      expect(result).toBe("");
    });
  });

  describe("renderTemplateFromFile", () => {
    it("should read a file and render its contents as a template", async () => {
      const { renderTemplateFromFile } = await import("./messages_core.cjs?" + Date.now());
      const tmpFile = path.join(os.tmpdir(), `msg-core-test-${Date.now()}.md`);
      fs.writeFileSync(tmpFile, "Hello, {name}!", "utf8");
      try {
        const result = renderTemplateFromFile(tmpFile, { name: "World" });
        expect(result).toBe("Hello, World!");
      } finally {
        fs.unlinkSync(tmpFile);
      }
    });

    it("should replace multiple placeholders from a file", async () => {
      const { renderTemplateFromFile } = await import("./messages_core.cjs?" + Date.now());
      const tmpFile = path.join(os.tmpdir(), `msg-core-test-${Date.now()}.md`);
      fs.writeFileSync(tmpFile, "{greeting}, {name}! Run: {run_url}", "utf8");
      try {
        const result = renderTemplateFromFile(tmpFile, {
          greeting: "Hello",
          name: "Alice",
          run_url: "https://github.com/actions/runs/123",
        });
        expect(result).toBe("Hello, Alice! Run: https://github.com/actions/runs/123");
      } finally {
        fs.unlinkSync(tmpFile);
      }
    });

    it("should leave unknown placeholders unchanged", async () => {
      const { renderTemplateFromFile } = await import("./messages_core.cjs?" + Date.now());
      const tmpFile = path.join(os.tmpdir(), `msg-core-test-${Date.now()}.md`);
      fs.writeFileSync(tmpFile, "Hello, {name}! {unknown}", "utf8");
      try {
        const result = renderTemplateFromFile(tmpFile, { name: "World" });
        expect(result).toBe("Hello, World! {unknown}");
      } finally {
        fs.unlinkSync(tmpFile);
      }
    });

    it("should return file contents unchanged when context is empty", async () => {
      const { renderTemplateFromFile } = await import("./messages_core.cjs?" + Date.now());
      const tmpFile = path.join(os.tmpdir(), `msg-core-test-${Date.now()}.md`);
      fs.writeFileSync(tmpFile, "No placeholders here.", "utf8");
      try {
        const result = renderTemplateFromFile(tmpFile, {});
        expect(result).toBe("No placeholders here.");
      } finally {
        fs.unlinkSync(tmpFile);
      }
    });
  });

  describe("toSnakeCase", () => {
    it("should convert camelCase keys to snake_case", async () => {
      const { toSnakeCase } = await import("./messages_core.cjs?" + Date.now());
      const result = toSnakeCase({ workflowName: "test" });
      expect(result.workflow_name).toBe("test");
    });

    it("should preserve original camelCase keys for backwards compatibility", async () => {
      const { toSnakeCase } = await import("./messages_core.cjs?" + Date.now());
      const result = toSnakeCase({ workflowName: "test" });
      expect(result.workflowName).toBe("test");
    });

    it("should not duplicate snake_case keys that are already snake_case", async () => {
      const { toSnakeCase } = await import("./messages_core.cjs?" + Date.now());
      const result = toSnakeCase({ run_url: "https://example.com" });
      // Only one entry for already-snake_case keys
      expect(result.run_url).toBe("https://example.com");
      expect(Object.keys(result).filter(k => k === "run_url")).toHaveLength(1);
    });

    it("should handle multi-word camelCase keys", async () => {
      const { toSnakeCase } = await import("./messages_core.cjs?" + Date.now());
      const result = toSnakeCase({ newDiscussionNumber: 42, newDiscussionUrl: "https://github.com" });
      expect(result.new_discussion_number).toBe(42);
      expect(result.new_discussion_url).toBe("https://github.com");
      // Original keys also preserved
      expect(result.newDiscussionNumber).toBe(42);
    });

    it("should handle empty object", async () => {
      const { toSnakeCase } = await import("./messages_core.cjs?" + Date.now());
      const result = toSnakeCase({});
      expect(result).toEqual({});
    });

    it("should handle multiple fields mixed camelCase and snake_case", async () => {
      const { toSnakeCase } = await import("./messages_core.cjs?" + Date.now());
      const result = toSnakeCase({ workflowName: "my-workflow", run_url: "https://example.com" });
      expect(result.workflow_name).toBe("my-workflow");
      expect(result.workflowName).toBe("my-workflow");
      expect(result.run_url).toBe("https://example.com");
    });
  });

  describe("getPromptPath", () => {
    it("should use RUNNER_TEMP when no override is set", async () => {
      process.env.RUNNER_TEMP = "/tmp/runner";
      const { getPromptPath } = await import("./messages_core.cjs?" + Date.now());
      const result = getPromptPath("agent_timeout.md");
      expect(result).toBe("/tmp/runner/gh-aw/prompts/agent_timeout.md");
    });

    it("should prefer GH_AW_PROMPTS_DIR over RUNNER_TEMP", async () => {
      process.env.RUNNER_TEMP = "/tmp/runner";
      process.env.GH_AW_PROMPTS_DIR = "/custom/prompts";
      const { getPromptPath } = await import("./messages_core.cjs?" + Date.now());
      const result = getPromptPath("agent_timeout.md");
      expect(result).toBe("/custom/prompts/agent_timeout.md");
    });

    it("should use GH_AW_PROMPTS_DIR when RUNNER_TEMP is not set", async () => {
      process.env.GH_AW_PROMPTS_DIR = "/custom/prompts";
      const { getPromptPath } = await import("./messages_core.cjs?" + Date.now());
      const result = getPromptPath("cache_memory_miss.md");
      expect(result).toBe("/custom/prompts/cache_memory_miss.md");
    });

    it("should include the template name in the path", async () => {
      process.env.RUNNER_TEMP = "/tmp/runner";
      const { getPromptPath } = await import("./messages_core.cjs?" + Date.now());
      expect(getPromptPath("foo.md")).toBe("/tmp/runner/gh-aw/prompts/foo.md");
      expect(getPromptPath("bar.md")).toBe("/tmp/runner/gh-aw/prompts/bar.md");
    });

    it("should throw when neither GH_AW_PROMPTS_DIR nor RUNNER_TEMP is set", async () => {
      const { getPromptPath } = await import("./messages_core.cjs?" + Date.now());
      expect(() => getPromptPath("any.md")).toThrow("Cannot resolve prompt path: neither GH_AW_PROMPTS_DIR nor RUNNER_TEMP is set");
    });
  });

  describe("getMessages", () => {
    it("should return null when env var is not set", async () => {
      const { getMessages } = await import("./messages_core.cjs?" + Date.now());
      const result = getMessages();
      expect(result).toBeNull();
    });

    it("should return null when env var is empty", async () => {
      process.env.GH_AW_SAFE_OUTPUT_MESSAGES = "";
      const { getMessages } = await import("./messages_core.cjs?" + Date.now());
      const result = getMessages();
      expect(result).toBeNull();
    });

    it("should parse valid JSON config", async () => {
      process.env.GH_AW_SAFE_OUTPUT_MESSAGES = JSON.stringify({ footer: "Custom footer" });
      const { getMessages } = await import("./messages_core.cjs?" + Date.now());
      const result = getMessages();
      expect(result).toEqual({ footer: "Custom footer" });
    });

    it("should parse config with multiple message fields", async () => {
      const config = {
        footer: "Custom footer",
        runStarted: "Workflow started",
        runSuccess: "Workflow succeeded",
        appendOnlyComments: true,
      };
      process.env.GH_AW_SAFE_OUTPUT_MESSAGES = JSON.stringify(config);
      const { getMessages } = await import("./messages_core.cjs?" + Date.now());
      const result = getMessages();
      expect(result).toEqual(config);
    });

    it("should return null and warn on invalid JSON", async () => {
      process.env.GH_AW_SAFE_OUTPUT_MESSAGES = "not-valid-json";
      const { getMessages } = await import("./messages_core.cjs?" + Date.now());
      const result = getMessages();
      expect(result).toBeNull();
      expect(mockCore.warning).toHaveBeenCalledWith(expect.stringContaining("Failed to parse GH_AW_SAFE_OUTPUT_MESSAGES"));
    });
  });
});
