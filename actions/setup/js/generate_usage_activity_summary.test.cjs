import { afterEach, beforeEach, describe, expect, it } from "vitest";
import fs from "fs";
import os from "os";
import path from "path";
import { createRequire } from "module";
import { fileURLToPath } from "url";

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);

const req = createRequire(import.meta.url);
const { parseFirewallLogs, parseSafeOutputsManifest } = req("./generate_usage_activity_summary.cjs");

describe("generate_usage_activity_summary.cjs", () => {
  /** Unique directory for each test to avoid cross-test interference */
  let squidLogDir;

  beforeEach(() => {
    squidLogDir = path.join("/tmp/gh-aw", `squid-logs-unit-test-${Date.now()}`);
    fs.mkdirSync(squidLogDir, { recursive: true });
  });

  afterEach(() => {
    if (fs.existsSync(squidLogDir)) {
      fs.rmSync(squidLogDir, { recursive: true, force: true });
    }
  });

  describe("parseFirewallLogs", () => {
    it("skips Squid diagnostic lines (WARNING:, DNS, Accepting) and does not treat them as domain names", () => {
      const logContent = [
        // Squid startup/diagnostic messages that should be skipped
        'WARNING: 172.30.0.20:35288 api.github.com:443 140.82.112.22:443 1.1 CONNECT 200 TCP_TUNNEL:HIER_DIRECT api.github.com:443 "-"',
        'DNS 172.30.0.20:35288 api.github.com:443 140.82.112.22:443 1.1 CONNECT 200 TCP_TUNNEL:HIER_DIRECT api.github.com:443 "-"',
        'Accepting 172.30.0.20:35288 api.github.com:443 140.82.112.22:443 1.1 CONNECT 200 TCP_TUNNEL:HIER_DIRECT api.github.com:443 "-"',
        // A valid access log entry that should be counted
        '1761332530.474 172.30.0.20:35288 api.github.com:443 140.82.112.22:443 1.1 CONNECT 200 TCP_TUNNEL:HIER_DIRECT api.github.com:443 "-"',
      ].join("\n");

      fs.writeFileSync(path.join(squidLogDir, "access.log"), logContent);

      const result = parseFirewallLogs();

      expect(result).not.toBeNull();
      expect(result.total_requests).toBe(1);
      expect(result.allowed_domains).toContain("api.github.com:443");
      // Diagnostic keywords must not appear as domain names
      expect(result.allowed_domains).not.toContain("WARNING:");
      expect(result.allowed_domains).not.toContain("DNS");
      expect(result.allowed_domains).not.toContain("Accepting");
    });

    it("returns null when only non-Squid diagnostic lines are present", () => {
      const logContent = [
        'WARNING: 172.30.0.20:35288 api.github.com:443 140.82.112.22:443 1.1 CONNECT 200 TCP_TUNNEL:HIER_DIRECT api.github.com:443 "-"',
        "DNS resolver ready - some extra fields here to pass length check x y z",
        "Accepting connections on port 3128 x y z",
      ].join("\n");

      fs.writeFileSync(path.join(squidLogDir, "access.log"), logContent);

      const result = parseFirewallLogs();

      expect(result).toBeNull();
    });

    it("counts valid Squid access log entries correctly", () => {
      const logContent = [
        '1761332530.474 172.30.0.20:35288 api.github.com:443 140.82.112.22:443 1.1 CONNECT 200 TCP_TUNNEL:HIER_DIRECT api.github.com:443 "-"',
        '1761332531.000 172.30.0.20:35289 blocked.example.com:443 1.2.3.4:443 1.1 CONNECT 403 NONE_NONE:HIER_NONE blocked.example.com:443 "-"',
      ].join("\n");

      fs.writeFileSync(path.join(squidLogDir, "access.log"), logContent);

      const result = parseFirewallLogs();

      expect(result).not.toBeNull();
      expect(result.total_requests).toBe(2);
      expect(result.allowed_requests).toBe(1);
      expect(result.blocked_requests).toBe(1);
      expect(result.allowed_domains).toContain("api.github.com:443");
      expect(result.blocked_domains).toContain("blocked.example.com:443");
    });
  });

  describe("parseSafeOutputsManifest", () => {
    /** Unique manifest file path per test to avoid cross-test interference */
    let manifestPath;

    beforeEach(() => {
      const testTmpDir = fs.mkdtempSync(path.join(os.tmpdir(), "safe-outputs-test-"));
      manifestPath = path.join(testTmpDir, "safe-output-items.jsonl");
    });

    afterEach(() => {
      const dir = path.dirname(manifestPath);
      if (fs.existsSync(dir)) {
        fs.rmSync(dir, { recursive: true, force: true });
      }
    });

    it("returns null when the manifest file does not exist", () => {
      const result = parseSafeOutputsManifest(manifestPath);
      expect(result).toBeNull();
    });

    it("returns null when the manifest file is empty", () => {
      fs.writeFileSync(manifestPath, "");
      const result = parseSafeOutputsManifest(manifestPath);
      expect(result).toBeNull();
    });

    it("returns null when the manifest contains only blank lines", () => {
      fs.writeFileSync(manifestPath, "\n\n\n");
      const result = parseSafeOutputsManifest(manifestPath);
      expect(result).toBeNull();
    });

    it("counts items by type from a valid manifest", () => {
      const lines = [
        JSON.stringify({ type: "create_issue", url: "https://github.com/owner/repo/issues/1" }),
        JSON.stringify({ type: "create_issue", url: "https://github.com/owner/repo/issues/2" }),
        JSON.stringify({ type: "add_comment", url: "https://github.com/owner/repo/issues/1#issuecomment-1" }),
      ].join("\n");
      fs.writeFileSync(manifestPath, lines);

      const result = parseSafeOutputsManifest(manifestPath);

      expect(result).not.toBeNull();
      expect(result.total_items).toBe(3);
      expect(result.items_by_type).toEqual({ create_issue: 2, add_comment: 1 });
    });

    it("skips lines with missing or empty type field", () => {
      const lines = [
        JSON.stringify({ type: "create_issue", url: "https://github.com/owner/repo/issues/1" }),
        JSON.stringify({ url: "https://example.com" }), // no type field
        JSON.stringify({ type: "", url: "https://example.com" }), // empty type
        "not json at all",
      ].join("\n");
      fs.writeFileSync(manifestPath, lines);

      const result = parseSafeOutputsManifest(manifestPath);

      expect(result).not.toBeNull();
      expect(result.total_items).toBe(1);
      expect(result.items_by_type).toEqual({ create_issue: 1 });
    });
  });
});
