import { describe, it as test, expect, beforeEach, vi } from "vitest";
import fs from "fs";
import path from "path";

const mockCore = {
  info: vi.fn(),
  setFailed: vi.fn(),
  summary: {
    addRaw: vi.fn().mockReturnThis(),
    write: vi.fn().mockResolvedValue(),
  },
};

global.core = mockCore;

describe("parse_firewall_logs.cjs", () => {
  let parseFirewallLogLine;
  let isRequestAllowed;
  let generateFirewallSummary;
  let sanitizeWorkflowName;

  beforeEach(() => {
    vi.clearAllMocks();

    const scriptPath = path.join(process.cwd(), "parse_firewall_logs.cjs");
    const scriptContent = fs.readFileSync(scriptPath, "utf8");
    // Update pattern to match the actual main block check in parse_firewall_logs.cjs
    const scriptForTesting = scriptContent
      .replace(/if \(typeof module === "undefined".*?\) \{[\s\S]*?main\(\);[\s\S]*?\}/g, "// main() execution disabled for testing")
      .replace(
        "// Export for testing",
        `global.testParseFirewallLogLine = parseFirewallLogLine;
        global.testIsRequestAllowed = isRequestAllowed;
        global.testGenerateFirewallSummary = generateFirewallSummary;
        global.testSanitizeWorkflowName = sanitizeWorkflowName;
        // Export for testing`
      );

    eval(scriptForTesting);

    parseFirewallLogLine = global.testParseFirewallLogLine;
    isRequestAllowed = global.testIsRequestAllowed;
    generateFirewallSummary = global.testGenerateFirewallSummary;
    sanitizeWorkflowName = global.testSanitizeWorkflowName;
  });

  describe("parseFirewallLogLine", () => {
    test("should parse valid firewall log line", () => {
      const line =
        '1761332530.474 172.30.0.20:35288 api.enterprise.githubcopilot.com:443 140.82.112.22:443 1.1 CONNECT 200 TCP_TUNNEL:HIER_DIRECT api.enterprise.githubcopilot.com:443 "-"';
      const result = parseFirewallLogLine(line);
      expect(result).not.toBeNull();
      expect(result.timestamp).toBe("1761332530.474");
      expect(result.clientIpPort).toBe("172.30.0.20:35288");
      expect(result.domain).toBe("api.enterprise.githubcopilot.com:443");
    });

    test("should return null for empty line", () => {
      expect(parseFirewallLogLine("")).toBeNull();
    });
  });

  describe("isRequestAllowed", () => {
    test("should allow request with status 200", () => {
      expect(isRequestAllowed("TCP_TUNNEL:HIER_DIRECT", "200")).toBe(true);
    });

    test("should deny request with NONE_NONE decision", () => {
      expect(isRequestAllowed("NONE_NONE:HIER_NONE", "0")).toBe(false);
    });
  });

  describe("sanitizeWorkflowName", () => {
    test("should convert to lowercase", () => {
      expect(sanitizeWorkflowName("MyWorkflow")).toBe("myworkflow");
    });
  });

  describe("generateFirewallSummary", () => {
    test("should generate summary with blocked requests only", () => {
      const analysis = {
        totalRequests: 5,
        allowedRequests: 3,
        deniedRequests: 2,
        allowedDomains: ["api.github.com:443", "api.npmjs.org:443"],
        deniedDomains: ["blocked.example.com:443", "denied.test.com:443"],
        requestsByDomain: new Map([
          ["blocked.example.com:443", { allowed: 0, denied: 1 }],
          ["denied.test.com:443", { allowed: 0, denied: 1 }],
        ]),
      };

      const summary = generateFirewallSummary(analysis);

      // Should focus on blocked requests
      expect(summary).toContain("🔥 Firewall Blocked Requests");
      expect(summary).toContain("**2** requests blocked");
      expect(summary).toContain("**2** unique domains");
      expect(summary).toContain("40% of total traffic");

      // Should show blocked domains table
      expect(summary).toContain("🚫 Blocked Domains");
      expect(summary).toContain("blocked.example.com:443");
      expect(summary).toContain("denied.test.com:443");

      // Should NOT show allowed domains section
      expect(summary).not.toContain("✅ Allowed Domains");
      expect(summary).not.toContain("api.github.com:443");
    });

    test("should show success message when no blocked requests", () => {
      const analysis = {
        totalRequests: 3,
        allowedRequests: 3,
        deniedRequests: 0,
        allowedDomains: ["api.github.com:443"],
        deniedDomains: [],
        requestsByDomain: new Map(),
      };

      const summary = generateFirewallSummary(analysis);

      expect(summary).toContain("✅ **No blocked requests detected**");
      expect(summary).toContain("All 3 requests were allowed");
    });
  });
});
