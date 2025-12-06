// @ts-check

/**
 * Tests for logger utilities
 */

import { describe, it, expect, beforeEach, afterEach, vi } from "vitest";

describe("logger.cjs", () => {
  describe("createLogger", () => {
    let stderrWriteSpy;
    let originalStderrWrite;
    let createLogger;

    beforeEach(async () => {
      // Import the module
      const module = await import("./logger.cjs");
      createLogger = module.createLogger;

      // Mock process.stderr.write
      originalStderrWrite = process.stderr.write;
      stderrWriteSpy = vi.fn();
      process.stderr.write = stderrWriteSpy;
    });

    afterEach(() => {
      // Restore original stderr.write
      process.stderr.write = originalStderrWrite;
    });

    it("should create a logger with debug method", () => {
      const logger = createLogger("test");
      expect(logger).toBeDefined();
      expect(logger.debug).toBeDefined();
      expect(typeof logger.debug).toBe("function");
    });

    it("should create a logger with debugError method", () => {
      const logger = createLogger("test");
      expect(logger.debugError).toBeDefined();
      expect(typeof logger.debugError).toBe("function");
    });

    it("should write debug messages with timestamp and name prefix", () => {
      const logger = createLogger("test-server");
      logger.debug("test message");

      expect(stderrWriteSpy).toHaveBeenCalledTimes(1);
      const output = stderrWriteSpy.mock.calls[0][0];
      expect(output).toMatch(/^\[\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}\.\d{3}Z\] \[test-server\] test message\n$/);
    });

    it("should handle multiple debug calls", () => {
      const logger = createLogger("multi-test");
      logger.debug("first message");
      logger.debug("second message");
      logger.debug("third message");

      expect(stderrWriteSpy).toHaveBeenCalledTimes(3);
      expect(stderrWriteSpy.mock.calls[0][0]).toContain("first message");
      expect(stderrWriteSpy.mock.calls[1][0]).toContain("second message");
      expect(stderrWriteSpy.mock.calls[2][0]).toContain("third message");
    });

    it("should use different prefixes for different loggers", () => {
      const logger1 = createLogger("server-1");
      const logger2 = createLogger("server-2");

      logger1.debug("from server 1");
      logger2.debug("from server 2");

      expect(stderrWriteSpy).toHaveBeenCalledTimes(2);
      expect(stderrWriteSpy.mock.calls[0][0]).toContain("[server-1]");
      expect(stderrWriteSpy.mock.calls[1][0]).toContain("[server-2]");
    });

    it("should handle debugError with Error object", () => {
      const logger = createLogger("error-test");
      const error = new Error("Test error message");
      error.stack = "Error: Test error message\n    at test.js:1:1";

      logger.debugError("Error occurred: ", error);

      expect(stderrWriteSpy).toHaveBeenCalledTimes(2);
      expect(stderrWriteSpy.mock.calls[0][0]).toContain("Error occurred: Test error message");
      expect(stderrWriteSpy.mock.calls[1][0]).toContain("Error occurred: Stack trace:");
      expect(stderrWriteSpy.mock.calls[1][0]).toContain("at test.js:1:1");
    });

    it("should handle debugError with string", () => {
      const logger = createLogger("error-test");
      logger.debugError("Error: ", "simple string error");

      expect(stderrWriteSpy).toHaveBeenCalledTimes(1);
      expect(stderrWriteSpy.mock.calls[0][0]).toContain("Error: simple string error");
    });

    it("should handle debugError with Error object without stack trace", () => {
      const logger = createLogger("error-test");
      const error = new Error("Test error");
      delete error.stack;

      logger.debugError("Error: ", error);

      expect(stderrWriteSpy).toHaveBeenCalledTimes(1);
      expect(stderrWriteSpy.mock.calls[0][0]).toContain("Error: Test error");
    });

    it("should handle empty messages", () => {
      const logger = createLogger("empty-test");
      logger.debug("");

      expect(stderrWriteSpy).toHaveBeenCalledTimes(1);
      const output = stderrWriteSpy.mock.calls[0][0];
      expect(output).toMatch(/^\[\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}\.\d{3}Z\] \[empty-test\] \n$/);
    });

    it("should handle special characters in messages", () => {
      const logger = createLogger("special-test");
      logger.debug("Message with: newlines\ntabs\tand special chars!@#$%");

      expect(stderrWriteSpy).toHaveBeenCalledTimes(1);
      expect(stderrWriteSpy.mock.calls[0][0]).toContain("Message with: newlines\ntabs\tand special chars!@#$%");
    });

    it("should handle long messages", () => {
      const logger = createLogger("long-test");
      const longMessage = "a".repeat(10000);
      logger.debug(longMessage);

      expect(stderrWriteSpy).toHaveBeenCalledTimes(1);
      expect(stderrWriteSpy.mock.calls[0][0]).toContain(longMessage);
    });

    it("should handle unicode characters", () => {
      const logger = createLogger("unicode-test");
      logger.debug("Hello 世界 🌍 émojis");

      expect(stderrWriteSpy).toHaveBeenCalledTimes(1);
      expect(stderrWriteSpy.mock.calls[0][0]).toContain("Hello 世界 🌍 émojis");
    });

    it("should format timestamps consistently", () => {
      const logger = createLogger("timestamp-test");
      logger.debug("first");
      logger.debug("second");

      expect(stderrWriteSpy).toHaveBeenCalledTimes(2);

      const output1 = stderrWriteSpy.mock.calls[0][0];
      const output2 = stderrWriteSpy.mock.calls[1][0];

      const timestampRegex = /^\[(\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}\.\d{3}Z)\]/;
      expect(output1).toMatch(timestampRegex);
      expect(output2).toMatch(timestampRegex);
    });

    it("should create independent logger instances", () => {
      const logger1 = createLogger("instance-1");
      const logger2 = createLogger("instance-2");

      logger1.debug("message 1");
      logger2.debug("message 2");

      expect(stderrWriteSpy).toHaveBeenCalledTimes(2);
      expect(stderrWriteSpy.mock.calls[0][0]).toContain("instance-1");
      expect(stderrWriteSpy.mock.calls[0][0]).toContain("message 1");
      expect(stderrWriteSpy.mock.calls[1][0]).toContain("instance-2");
      expect(stderrWriteSpy.mock.calls[1][0]).toContain("message 2");
    });

    it("should handle debugError prefix without trailing space", () => {
      const logger = createLogger("prefix-test");
      const error = new Error("test");

      logger.debugError("ERROR", error);

      expect(stderrWriteSpy).toHaveBeenCalled();
      expect(stderrWriteSpy.mock.calls[0][0]).toContain("ERRORtest");
    });

    it("should handle debugError prefix with trailing space", () => {
      const logger = createLogger("prefix-test");
      const error = new Error("test");

      logger.debugError("ERROR: ", error);

      expect(stderrWriteSpy).toHaveBeenCalled();
      expect(stderrWriteSpy.mock.calls[0][0]).toContain("ERROR: test");
    });
  });
});
