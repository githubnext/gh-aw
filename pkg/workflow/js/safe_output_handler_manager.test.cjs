// @ts-check

import { describe, it, expect, beforeEach } from "vitest";
import { SafeOutputHandlerManager } from "./safe_output_handler_manager.cjs";

describe("SafeOutputHandlerManager", () => {
  /** @type {SafeOutputHandlerManager} */
  let manager;
  /** @type {any} */
  let mockContext;

  beforeEach(() => {
    manager = new SafeOutputHandlerManager();
    mockContext = {
      core: {
        info: () => {},
        warning: () => {},
        error: () => {},
      },
      github: {},
      context: {},
      exec: {},
      temporaryIdMap: new Map(),
    };
  });

  describe("registerHandler", () => {
    it("should register a handler for a message type", () => {
      const handler = async () => ({ success: true });
      manager.registerHandler("create_issue", handler);
      
      expect(manager.getRegisteredTypes()).toContain("create_issue");
    });

    it("should throw error when registering duplicate type", () => {
      const handler = async () => ({ success: true });
      manager.registerHandler("create_issue", handler);
      
      expect(() => {
        manager.registerHandler("create_issue", handler);
      }).toThrow("Handler already registered for type: create_issue");
    });

    it("should allow multiple different types", () => {
      manager.registerHandler("create_issue", async () => ({ success: true }));
      manager.registerHandler("create_pull_request", async () => ({ success: true }));
      
      const types = manager.getRegisteredTypes();
      expect(types).toContain("create_issue");
      expect(types).toContain("create_pull_request");
      expect(types.length).toBe(2);
    });
  });

  describe("processAll", () => {
    it("should process items with registered handlers", async () => {
      let called = false;
      manager.registerHandler("create_issue", async (item) => {
        called = true;
        expect(item.title).toBe("Test Issue");
        return { success: true };
      });

      const items = [{ type: "create_issue", title: "Test Issue" }];
      const result = await manager.processAll(items, mockContext);

      expect(called).toBe(true);
      expect(result.success).toBe(true);
      expect(result.errors.length).toBe(0);
    });

    it("should skip items with no registered handler", async () => {
      const items = [{ type: "unknown_type", data: "test" }];
      const result = await manager.processAll(items, mockContext);

      expect(result.success).toBe(true);
      expect(result.errors.length).toBe(0);
    });

    it("should skip items with no type field", async () => {
      const items = [{ data: "test" }];
      const result = await manager.processAll(items, mockContext);

      expect(result.success).toBe(true);
      expect(result.errors.length).toBe(0);
    });

    it("should collect temporary IDs from handlers", async () => {
      manager.registerHandler("create_issue", async () => {
        const tempIds = new Map();
        tempIds.set("temp-1", { repo: "owner/repo", number: 42 });
        return { success: true, temporaryIds: tempIds };
      });

      manager.registerHandler("create_pull_request", async () => {
        const tempIds = new Map();
        tempIds.set("temp-2", { repo: "owner/repo", number: 100 });
        return { success: true, temporaryIds: tempIds };
      });

      const items = [
        { type: "create_issue", title: "Issue" },
        { type: "create_pull_request", title: "PR" },
      ];
      const result = await manager.processAll(items, mockContext);

      expect(result.success).toBe(true);
      expect(result.temporaryIdMap.size).toBe(2);
      expect(result.temporaryIdMap.get("temp-1")).toEqual({ repo: "owner/repo", number: 42 });
      expect(result.temporaryIdMap.get("temp-2")).toEqual({ repo: "owner/repo", number: 100 });
      
      // Also check that context.temporaryIdMap was updated
      expect(mockContext.temporaryIdMap.size).toBe(2);
    });

    it("should pass temporary IDs from earlier handlers to later handlers", async () => {
      manager.registerHandler("create_issue", async () => {
        const tempIds = new Map();
        tempIds.set("temp-1", { repo: "owner/repo", number: 42 });
        return { success: true, temporaryIds: tempIds };
      });

      manager.registerHandler("add_comment", async (item, context) => {
        // This handler should be able to see temp IDs from create_issue
        expect(context.temporaryIdMap.has("temp-1")).toBe(true);
        expect(context.temporaryIdMap.get("temp-1")).toEqual({ repo: "owner/repo", number: 42 });
        return { success: true };
      });

      const items = [
        { type: "create_issue", title: "Issue" },
        { type: "add_comment", body: "Comment", parent: "temp-1" },
      ];
      await manager.processAll(items, mockContext);
    });

    it("should collect errors from failed handlers", async () => {
      manager.registerHandler("create_issue", async () => {
        return { success: false, error: "Failed to create issue" };
      });

      const items = [{ type: "create_issue", title: "Issue" }];
      const result = await manager.processAll(items, mockContext);

      expect(result.success).toBe(false);
      expect(result.errors.length).toBe(1);
      expect(result.errors[0]).toContain("Failed to create issue");
    });

    it("should catch exceptions from handlers", async () => {
      manager.registerHandler("create_issue", async () => {
        throw new Error("Unexpected error");
      });

      const items = [{ type: "create_issue", title: "Issue" }];
      const result = await manager.processAll(items, mockContext);

      expect(result.success).toBe(false);
      expect(result.errors.length).toBe(1);
      expect(result.errors[0]).toContain("Unexpected error");
    });

    it("should process multiple items in sequence", async () => {
      const callOrder = [];
      
      manager.registerHandler("create_issue", async (item) => {
        callOrder.push(`issue:${item.title}`);
        return { success: true };
      });

      manager.registerHandler("create_pull_request", async (item) => {
        callOrder.push(`pr:${item.title}`);
        return { success: true };
      });

      const items = [
        { type: "create_issue", title: "Issue 1" },
        { type: "create_pull_request", title: "PR 1" },
        { type: "create_issue", title: "Issue 2" },
      ];
      await manager.processAll(items, mockContext);

      expect(callOrder).toEqual(["issue:Issue 1", "pr:PR 1", "issue:Issue 2"]);
    });
  });
});
