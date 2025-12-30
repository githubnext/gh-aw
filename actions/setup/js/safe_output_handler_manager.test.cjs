// @ts-check

import { describe, it, expect, beforeEach, afterEach, vi } from "vitest";
import { loadConfig, loadHandlers, processMessages } from "./safe_output_handler_manager.cjs";

describe("Safe Output Handler Manager", () => {
  beforeEach(() => {
    // Mock global core
    global.core = {
      info: vi.fn(),
      debug: vi.fn(),
      warning: vi.fn(),
      error: vi.fn(),
      setOutput: vi.fn(),
      setFailed: vi.fn(),
    };
  });

  afterEach(() => {
    // Clean up environment variables
    delete process.env.GH_AW_SAFE_OUTPUTS_HANDLER_CONFIG;
  });

  describe("loadConfig", () => {
    it("should load config from environment variable and normalize keys", () => {
      const config = {
        "create-issue": { max: 5 },
        "add-comment": { max: 1 },
      };

      process.env.GH_AW_SAFE_OUTPUTS_HANDLER_CONFIG = JSON.stringify(config);

      const result = loadConfig();

      expect(result).toHaveProperty("create_issue");
      expect(result).toHaveProperty("add_comment");
      expect(result.create_issue).toEqual({ max: 5 });
      expect(result.add_comment).toEqual({ max: 1 });
    });

    it("should throw error if environment variable is not set", () => {
      expect(() => loadConfig()).toThrow("GH_AW_SAFE_OUTPUTS_HANDLER_CONFIG environment variable is required but not set");
    });

    it("should throw error if environment variable contains invalid JSON", () => {
      process.env.GH_AW_SAFE_OUTPUTS_HANDLER_CONFIG = "not json";
      expect(() => loadConfig()).toThrow("Failed to parse GH_AW_SAFE_OUTPUTS_HANDLER_CONFIG");
    });
  });

  describe("loadHandlers", () => {
    // These tests are skipped because they require actual handler modules to exist
    // In a real environment, handlers are loaded dynamically via require()
    it.skip("should load handlers for enabled safe output types", async () => {
      const config = {
        create_issue: { max: 1 },
        add_comment: { max: 1 },
      };

      const handlers = await loadHandlers(config);

      expect(handlers.size).toBeGreaterThan(0);
      expect(handlers.has("create_issue")).toBe(true);
      expect(handlers.has("add_comment")).toBe(true);
    });

    it.skip("should not load handlers when config entry is missing", async () => {
      const config = {
        create_issue: { max: 1 },
        // add_comment is not in config
      };

      const handlers = await loadHandlers(config);

      expect(handlers.has("create_issue")).toBe(true);
      expect(handlers.has("add_comment")).toBe(false);
    });

    it.skip("should handle missing handlers gracefully", async () => {
      const config = {
        nonexistent_handler: { max: 1 },
      };

      const handlers = await loadHandlers(config);

      expect(handlers.size).toBe(0);
    });
  });

  describe("processMessages", () => {
    it("should process messages in order of appearance", async () => {
      const messages = [
        { type: "add_comment", body: "Comment" },
        { type: "create_issue", title: "Issue" },
      ];

      const mockHandler = vi.fn().mockResolvedValue({ success: true });

      const handlers = new Map([
        ["create_issue", mockHandler],
        ["add_comment", mockHandler],
      ]);

      const result = await processMessages(handlers, messages);

      expect(result.success).toBe(true);
      expect(result.results).toHaveLength(2);

      // Verify handlers were called
      expect(mockHandler).toHaveBeenCalledTimes(2);

      // Verify messages were processed in order of appearance (add_comment first, then create_issue)
      expect(result.results[0].type).toBe("add_comment");
      expect(result.results[0].messageIndex).toBe(0);
      expect(result.results[1].type).toBe("create_issue");
      expect(result.results[1].messageIndex).toBe(1);
    });

    it("should skip messages without type", async () => {
      const messages = [{ type: "create_issue", title: "Issue" }, { title: "No type" }, { type: "add_comment", body: "Comment" }];

      const mockHandler = vi.fn().mockResolvedValue({ success: true });

      const handlers = new Map([
        ["create_issue", mockHandler],
        ["add_comment", mockHandler],
      ]);

      const result = await processMessages(handlers, messages);

      expect(result.success).toBe(true);
      expect(result.results).toHaveLength(2);
      expect(core.warning).toHaveBeenCalledWith("Skipping message 2 without type");
    });

    it("should handle handler errors gracefully", async () => {
      const messages = [{ type: "create_issue", title: "Issue" }];

      const errorHandler = vi.fn().mockRejectedValue(new Error("Handler failed"));

      const handlers = new Map([["create_issue", errorHandler]]);

      const result = await processMessages(handlers, messages);

      expect(result.success).toBe(true);
      expect(result.results).toHaveLength(1);
      expect(result.results[0].success).toBe(false);
      expect(result.results[0].error).toBe("Handler failed");
    });

    it("should track outputs with unresolved temporary IDs", async () => {
      const messages = [
        { 
          type: "create_issue", 
          body: "See #aw_abc123def456 for context",
          title: "Test Issue" 
        },
      ];

      const mockCreateIssueHandler = vi.fn().mockResolvedValue({
        repo: "owner/repo",
        number: 100,
      });

      const handlers = new Map([["create_issue", mockCreateIssueHandler]]);

      const result = await processMessages(handlers, messages);

      expect(result.success).toBe(true);
      expect(result.pendingUpdates).toBeDefined();
      // No synthetic updates should be generated because the temp ID is never resolved
      expect(result.pendingUpdates.length).toBe(0);
    });

    it("should generate synthetic update when temporary ID is resolved", async () => {
      const messages = [
        { 
          type: "create_issue", 
          body: "See #aw_abc123def456 for context",
          title: "First Issue" 
        },
        { 
          type: "create_issue",
          temporary_id: "aw_abc123def456",
          body: "Second issue body",
          title: "Second Issue" 
        },
      ];

      const mockCreateIssueHandler = vi.fn()
        .mockResolvedValueOnce({
          repo: "owner/repo",
          number: 100,
        })
        .mockResolvedValueOnce({
          repo: "owner/repo",
          number: 101,
          temporaryId: "aw_abc123def456",
        });

      const handlers = new Map([["create_issue", mockCreateIssueHandler]]);

      const result = await processMessages(handlers, messages);

      expect(result.success).toBe(true);
      expect(result.pendingUpdates).toBeDefined();
      // Should generate synthetic update for first issue
      expect(result.pendingUpdates.length).toBe(1);
      expect(result.pendingUpdates[0].type).toBe("update_issue");
      expect(result.pendingUpdates[0].issue_number).toBe(100);
      expect(result.pendingUpdates[0]._synthetic).toBe(true);
      // Body should have temp ID replaced
      expect(result.pendingUpdates[0].body).toContain("#101");
      expect(result.pendingUpdates[0].body).not.toContain("aw_abc123def456");
    });

    it("should not generate synthetic update if temporary IDs remain unresolved", async () => {
      const messages = [
        { 
          type: "create_issue", 
          body: "See #aw_abc123def456 and #aw_unresolved99 for context",
          title: "Test Issue" 
        },
      ];

      const mockCreateIssueHandler = vi.fn().mockResolvedValue({
        repo: "owner/repo",
        number: 100,
      });

      const handlers = new Map([["create_issue", mockCreateIssueHandler]]);

      const result = await processMessages(handlers, messages);

      expect(result.success).toBe(true);
      expect(result.pendingUpdates).toBeDefined();
      // Should not generate synthetic update because IDs are still unresolved
      expect(result.pendingUpdates.length).toBe(0);
    });

    it("should handle multiple outputs needing synthetic updates", async () => {
      const messages = [
        { 
          type: "create_issue", 
          body: "Related to #aw_aabbcc111111",
          title: "First Issue" 
        },
        { 
          type: "create_discussion", 
          body: "See #aw_aabbcc111111 for details",
          title: "Discussion" 
        },
        { 
          type: "create_issue",
          temporary_id: "aw_aabbcc111111",
          body: "The referenced issue",
          title: "Referenced Issue" 
        },
      ];

      const mockCreateIssueHandler = vi.fn()
        .mockResolvedValueOnce({
          repo: "owner/repo",
          number: 100,
        })
        .mockResolvedValueOnce({
          repo: "owner/repo",
          number: 102,
          temporaryId: "aw_aabbcc111111",
        });

      const mockCreateDiscussionHandler = vi.fn().mockResolvedValue({
        repo: "owner/repo",
        number: 101,
      });

      const handlers = new Map([
        ["create_issue", mockCreateIssueHandler],
        ["create_discussion", mockCreateDiscussionHandler],
      ]);

      const result = await processMessages(handlers, messages);

      expect(result.success).toBe(true);
      expect(result.pendingUpdates.length).toBe(2);
      // Should generate update_issue for first issue
      expect(result.pendingUpdates.find(u => u.type === "update_issue")).toBeDefined();
      // Should generate update_discussion for discussion
      expect(result.pendingUpdates.find(u => u.type === "update_discussion")).toBeDefined();
    });
  });
});
