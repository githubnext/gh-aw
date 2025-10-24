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
    const scriptForTesting = scriptContent.replace("if (require.main === module) {", "if (false) {").replace(
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
});
