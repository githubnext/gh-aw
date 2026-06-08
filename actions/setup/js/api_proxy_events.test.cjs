import { describe, it, beforeEach, afterEach, expect } from "vitest";
import { createRequire } from "module";
import fs from "fs";
import os from "os";
import path from "path";

const require = createRequire(import.meta.url);
const {
  findAPIProxyEventsFile,
  parseAPIProxyEvents,
  checkAPIProxyErrors,
} = require("./api_proxy_events.cjs");

describe("API Proxy Event Log Helpers", () => {
  let tmpDir;

  beforeEach(() => {
    tmpDir = fs.mkdtempSync(path.join(os.tmpdir(), "api-proxy-events-test-"));
  });

  afterEach(() => {
    if (tmpDir && fs.existsSync(tmpDir)) {
      fs.rmSync(tmpDir, { recursive: true, force: true });
    }
  });

  describe("findAPIProxyEventsFile", () => {
    it("should find events.jsonl at primary location", () => {
      const logsDir = path.join(tmpDir, "logs");
      const eventsDir = path.join(logsDir, "api-proxy-logs");
      fs.mkdirSync(eventsDir, { recursive: true });
      const eventsPath = path.join(eventsDir, "events.jsonl");
      fs.writeFileSync(eventsPath, "");

      const found = findAPIProxyEventsFile(logsDir);
      expect(found).toBe(eventsPath);
    });

    it("should return empty string when file does not exist", () => {
      const logsDir = path.join(tmpDir, "logs");
      fs.mkdirSync(logsDir, { recursive: true });

      const found = findAPIProxyEventsFile(logsDir);
      expect(found).toBe("");
    });
  });

  describe("parseAPIProxyEvents", () => {
    it("should detect rate-limit error from status code 429", () => {
      const eventsPath = path.join(tmpDir, "events.jsonl");
      const events = [
        { event: "api_error", status_code: 429, message: "Rate limited" },
      ];
      fs.writeFileSync(eventsPath, events.map(e => JSON.stringify(e)).join("\n"));

      const result = parseAPIProxyEvents(eventsPath);
      expect(result.hasRateLimitError).toBe(true);
      expect(result.totalErrors).toBe(1);
      expect(result.errorTypes).toContain("rate_limit");
    });

    it("should detect max-runs-exceeded error", () => {
      const eventsPath = path.join(tmpDir, "events.jsonl");
      const events = [
        { event: "max_runs_exceeded", message: "Maximum LLM invocations exceeded (50/50)" },
      ];
      fs.writeFileSync(eventsPath, events.map(e => JSON.stringify(e)).join("\n"));

      const result = parseAPIProxyEvents(eventsPath);
      expect(result.hasMaxRunsExceeded).toBe(true);
      expect(result.errorTypes).toContain("max_runs_exceeded");
    });

    it("should return empty classification for non-existent file", () => {
      const result = parseAPIProxyEvents("/non/existent/path");
      expect(result.hasRateLimitError).toBe(false);
      expect(result.hasMaxRunsExceeded).toBe(false);
      expect(result.hasOverloadError).toBe(false);
      expect(result.totalErrors).toBe(0);
      expect(result.errorTypes).toEqual([]);
    });
  });

  describe("checkAPIProxyErrors", () => {
    it("should find and parse events file", () => {
      const logsDir = path.join(tmpDir, "logs");
      const eventsDir = path.join(logsDir, "api-proxy-logs");
      fs.mkdirSync(eventsDir, { recursive: true });
      const eventsPath = path.join(eventsDir, "events.jsonl");
      const events = [
        { event: "rate_limit_error", status_code: 429, message: "Rate limited" },
      ];
      fs.writeFileSync(eventsPath, events.map(e => JSON.stringify(e)).join("\n"));

      const result = checkAPIProxyErrors(logsDir);
      expect(result.hasRateLimitError).toBe(true);
    });
  });
});
