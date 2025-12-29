// @ts-check

import { describe, it, expect, beforeEach, afterEach, vi } from "vitest";
import { loadConfig, loadHandlers, processMessages } from "./safe_output_handler_manager.cjs";
import fs from "fs";
import path from "path";

describe("Safe Output Handler Manager", () => {
  const testConfigPath = "/tmp/gh-aw/safeoutputs/config.json";

  beforeEach(() => {
    // Ensure test directory exists
    const dir = path.dirname(testConfigPath);
    if (!fs.existsSync(dir)) {
      fs.mkdirSync(dir, { recursive: true });
    }

    // Set environment variable for config path
    process.env.GH_AW_SAFE_OUTPUTS_CONFIG_PATH = testConfigPath;

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
    // Clean up test config file
    if (fs.existsSync(testConfigPath)) {
      fs.unlinkSync(testConfigPath);
    }
    delete process.env.GH_AW_SAFE_OUTPUTS_CONFIG_PATH;
  });

  describe("loadConfig", () => {
    it("should load config from file and normalize keys", () => {
      const config = {
        "create-issue": { enabled: true, max: 5 },
        "add-comment": { enabled: true },
      };

      fs.writeFileSync(testConfigPath, JSON.stringify(config));

      const result = loadConfig();

      expect(result).toHaveProperty("create_issue");
      expect(result).toHaveProperty("add_comment");
      expect(result.create_issue).toEqual({ enabled: true, max: 5 });
      expect(result.add_comment).toEqual({ enabled: true });
    });

    it("should return empty object if config file does not exist", () => {
      const result = loadConfig();
      expect(result).toEqual({});
    });

    it("should return empty object if config file is invalid JSON", () => {
      fs.writeFileSync(testConfigPath, "not json");
      const result = loadConfig();
      expect(result).toEqual({});
    });
  });

  describe("loadHandlers", () => {
    it("should load handlers for enabled safe output types", () => {
      const config = {
        create_issue: { enabled: true },
        add_comment: { enabled: true },
      };

      const handlers = loadHandlers(config);

      expect(handlers.size).toBeGreaterThan(0);
      expect(handlers.has("create_issue")).toBe(true);
      expect(handlers.has("add_comment")).toBe(true);
    });

    it("should not load handlers for disabled safe output types", () => {
      const config = {
        create_issue: { enabled: false },
      };

      const handlers = loadHandlers(config);

      expect(handlers.has("create_issue")).toBe(false);
    });

    it("should handle missing handlers gracefully", () => {
      const config = {
        nonexistent_handler: { enabled: true },
      };

      const handlers = loadHandlers(config);

      expect(handlers.size).toBe(0);
    });
  });

  describe("processMessages", () => {
    it("should process messages in order of appearance", async () => {
      const messages = [
        { type: "add_comment", body: "Comment" },
        { type: "create_issue", title: "Issue" },
      ];

      const mockHandler = {
        main: vi.fn().mockResolvedValue({ success: true }),
      };

      const handlers = new Map([
        ["create_issue", mockHandler],
        ["add_comment", mockHandler],
      ]);

      const result = await processMessages(handlers, messages);

      expect(result.success).toBe(true);
      expect(result.results).toHaveLength(2);

      // Verify messages were processed in order of appearance (add_comment first, then create_issue)
      expect(result.results[0].type).toBe("add_comment");
      expect(result.results[0].messageIndex).toBe(0);
      expect(result.results[1].type).toBe("create_issue");
      expect(result.results[1].messageIndex).toBe(1);
    });

    it("should skip messages without type", async () => {
      const messages = [{ type: "create_issue", title: "Issue" }, { title: "No type" }, { type: "add_comment", body: "Comment" }];

      const mockHandler = {
        main: vi.fn().mockResolvedValue({ success: true }),
      };

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

      const errorHandler = {
        main: vi.fn().mockRejectedValue(new Error("Handler failed")),
      };

      const handlers = new Map([["create_issue", errorHandler]]);

      const result = await processMessages(handlers, messages);

      expect(result.success).toBe(true);
      expect(result.results).toHaveLength(1);
      expect(result.results[0].success).toBe(false);
      expect(result.results[0].error).toBe("Handler failed");
    });
  });
});
