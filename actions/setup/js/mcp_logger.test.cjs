import { describe, it, expect, beforeEach, afterEach, vi } from "vitest";
import { createLogger } from "./mcp_logger.cjs";
describe("mcp_logger.cjs", () => {
  let stderrSpy;
  (beforeEach(() => {
    vi.stubEnv("DEBUG", "*");
    stderrSpy = vi.spyOn(process.stderr, "write").mockImplementation(() => !0);
  }),
    afterEach(() => {
      stderrSpy.mockRestore();
      vi.unstubAllEnvs();
    }),
    describe("createLogger", () => {
      (it("should create a logger with debug method", () => {
        const logger = createLogger("test-server");
        (expect(logger).toBeDefined(), expect(logger.debug).toBeDefined(), expect(typeof logger.debug).toBe("function"));
      }),
        it("should create a logger with debugError method", () => {
          const logger = createLogger("test-server");
          (expect(logger).toBeDefined(), expect(logger.debugError).toBeDefined(), expect(typeof logger.debugError).toBe("function"));
        }),
        it("should log messages with timestamp and server name", () => {
          (createLogger("test-server").debug("Test message"), expect(stderrSpy).toHaveBeenCalledOnce());
          const output = stderrSpy.mock.calls[0][0];
          (expect(output).toContain("[test-server]"), expect(output).toContain("Test message"), expect(output).toMatch(/\[\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}\.\d{3}Z\]/));
        }),
        it("should log error messages", () => {
          const logger = createLogger("test-server"),
            error = new Error("Test error");
          (logger.debugError("Error: ", error), expect(stderrSpy).toHaveBeenCalled());
          const output = stderrSpy.mock.calls[0][0];
          expect(output).toContain("Error: Test error");
        }),
        it("should log error stack trace if available", () => {
          const logger = createLogger("test-server"),
            error = new Error("Test error");
          (logger.debugError("Error: ", error), expect(stderrSpy).toHaveBeenCalled(), expect(stderrSpy.mock.calls.length).toBeGreaterThanOrEqual(2));
          const stackOutput = stderrSpy.mock.calls[1][0];
          expect(stackOutput).toContain("Stack trace:");
        }),
        it("should handle non-Error objects in debugError", () => {
          (createLogger("test-server").debugError("Error: ", "Simple string error"), expect(stderrSpy).toHaveBeenCalledOnce());
          const output = stderrSpy.mock.calls[0][0];
          expect(output).toContain("Error: Simple string error");
        }),
        it("should use different server names for different loggers", () => {
          const logger1 = createLogger("server-1"),
            logger2 = createLogger("server-2");
          (logger1.debug("Message from server 1"),
            logger2.debug("Message from server 2"),
            expect(stderrSpy).toHaveBeenCalledTimes(2),
            expect(stderrSpy.mock.calls[0][0]).toContain("[server-1]"),
            expect(stderrSpy.mock.calls[1][0]).toContain("[server-2]"));
        }),
        it("should log multiple messages sequentially", () => {
          const logger = createLogger("test-server");
          (logger.debug("First message"),
            logger.debug("Second message"),
            logger.debug("Third message"),
            expect(stderrSpy).toHaveBeenCalledTimes(3),
            expect(stderrSpy.mock.calls[0][0]).toContain("First message"),
            expect(stderrSpy.mock.calls[1][0]).toContain("Second message"),
            expect(stderrSpy.mock.calls[2][0]).toContain("Third message"));
        }),
        it.each([
          ["", false],
          ["unrelated:*", false],
          ["test-server", true],
          ["test-*", true],
          ["*", true],
          ["*,-test-server", false],
          ["-test-*,*", false],
          ["other,test-server", true],
          ["other test-server", true],
          ["test.server", false],
        ])("filters actual stderr with DEBUG=%s", (pattern, enabled) => {
          vi.stubEnv("DEBUG", pattern);
          createLogger("test-server").debug("selected diagnostic");
          expect(stderrSpy).toHaveBeenCalledTimes(enabled ? 1 : 0);
          if (enabled) expect(stderrSpy.mock.calls[0][0]).toContain("selected diagnostic");
        }),
        it("keeps diagnostics and error summaries visible with DEBUG disabled, but not stacks", () => {
          vi.stubEnv("DEBUG", "");
          const logger = createLogger("test-server");
          logger.diagnostic("required audit");
          logger.debugError("Failed: ", new Error("failure summary"));
          expect(stderrSpy).toHaveBeenCalledTimes(2);
          expect(stderrSpy.mock.calls[0][0]).toContain("required audit");
          expect(stderrSpy.mock.calls[1][0]).toContain("Failed: failure summary");
          expect(stderrSpy.mock.calls.join("")).not.toContain("Stack trace");
        }),
        it("supports namespace wildcards and exclusions without runner flags overriding DEBUG", () => {
          vi.stubEnv("DEBUG", "mcp:*,-mcp:hidden");
          vi.stubEnv("RUNNER_DEBUG", "1");
          vi.stubEnv("ACTIONS_RUNNER_DEBUG", "true");
          createLogger("mcp:visible").debug("selected");
          createLogger("mcp:hidden").debug("excluded");
          createLogger("other:server").debug("unrelated");
          expect(stderrSpy).toHaveBeenCalledTimes(1);
          expect(stderrSpy.mock.calls[0][0]).toContain("[mcp:visible] selected");
        }),
        it("redacts diagnostics, errors and enabled debug messages", () => {
          const logger = createLogger("test-server", msg => msg.replaceAll("private-value", "[redacted]"));
          logger.debug("private-value");
          logger.diagnostic("private-value");
          logger.debugError("Failed: ", new Error("private-value"));
          expect(stderrSpy.mock.calls.join("")).not.toContain("private-value");
          expect(stderrSpy.mock.calls.join("")).toContain("[redacted]");
        }));
    }));
});
