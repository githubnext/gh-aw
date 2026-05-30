import { describe, it as test, expect, beforeEach, vi } from "vitest";
import fs from "fs";
import path from "path";
const mockCore = { info: vi.fn(), setFailed: vi.fn(), summary: { addRaw: vi.fn().mockReturnThis(), write: vi.fn().mockResolvedValue() } };
((global.core = mockCore),
  describe("parse_firewall_logs.cjs", () => {
    let parseFirewallLogLine, isRequestAllowed, analyzeFirewallLogLines, generateFirewallSummary;
    (beforeEach(() => {
      vi.clearAllMocks();
      const scriptPath = path.join(process.cwd(), "parse_firewall_logs.cjs"),
        scriptContent = fs.readFileSync(scriptPath, "utf8"),
        scriptForTesting = scriptContent
          .replace(/if \(typeof module === "undefined".*?\) \{[\s\S]*?main\(\);[\s\S]*?\}/g, "// main() execution disabled for testing")
          .replace(
            "// Export for testing",
            "global.testParseFirewallLogLine = parseFirewallLogLine;\n        global.testIsRequestAllowed = isRequestAllowed;\n        global.testAnalyzeFirewallLogLines = analyzeFirewallLogLines;\n        global.testGenerateFirewallSummary = generateFirewallSummary;\n        // Export for testing"
          );
      (eval(scriptForTesting),
        (parseFirewallLogLine = global.testParseFirewallLogLine),
        (isRequestAllowed = global.testIsRequestAllowed),
        (analyzeFirewallLogLines = global.testAnalyzeFirewallLogLines),
        (generateFirewallSummary = global.testGenerateFirewallSummary));
    }),
      describe("parseFirewallLogLine", () => {
        (test("should parse valid firewall log line", () => {
          const result = parseFirewallLogLine('1761332530.474 172.30.0.20:35288 api.enterprise.githubcopilot.com:443 140.82.112.22:443 1.1 CONNECT 200 TCP_TUNNEL:HIER_DIRECT api.enterprise.githubcopilot.com:443 "-"');
          (expect(result).not.toBeNull(), expect(result.timestamp).toBe("1761332530.474"), expect(result.clientIpPort).toBe("172.30.0.20:35288"), expect(result.domain).toBe("api.enterprise.githubcopilot.com:443"));
        }),
          test("should parse log line with placeholder values", () => {
            const result = parseFirewallLogLine('1761332530.500 - - - - - 0 NONE_NONE:HIER_NONE - "-"');
            (expect(result).not.toBeNull(), expect(result.domain).toBe("-"));
          }),
          test("should return null for empty line", () => {
            expect(parseFirewallLogLine("")).toBeNull();
          }),
          test("should return null for invalid timestamp", () => {
            expect(parseFirewallLogLine('WARNING: 172.30.0.20:35288 api.github.com:443 140.82.112.22:443 1.1 CONNECT 200 TCP_TUNNEL:HIER_DIRECT api.github.com:443 "-"')).toBeNull();
          }),
          test("should parse line with non-standard client IP:port format", () => {
            const result = parseFirewallLogLine('1761332530.474 Accepting api.github.com:443 140.82.112.22:443 1.1 CONNECT 200 TCP_TUNNEL:HIER_DIRECT api.github.com:443 "-"');
            (expect(result).not.toBeNull(), expect(result.clientIpPort).toBe("Accepting"), expect(result.domain).toBe("api.github.com:443"));
          }),
          test("should parse line with non-standard domain format", () => {
            const result = parseFirewallLogLine('1761332530.474 172.30.0.20:35288 DNS 140.82.112.22:443 1.1 CONNECT 200 TCP_TUNNEL:HIER_DIRECT api.github.com:443 "-"');
            (expect(result).not.toBeNull(), expect(result.domain).toBe("DNS"), expect(result.clientIpPort).toBe("172.30.0.20:35288"));
          }),
          test("should return null for lines with fewer than 10 fields", () => {
            (expect(parseFirewallLogLine("WARNING: Something went wrong")).toBeNull(), expect(parseFirewallLogLine("DNS lookup failed")).toBeNull(), expect(parseFirewallLogLine("Accepting connection")).toBeNull());
          }),
          test("should parse line with non-standard dest IP:port format", () => {
            const result = parseFirewallLogLine('1761332530.474 172.30.0.20:35288 api.github.com:443 Local 1.1 CONNECT 200 TCP_TUNNEL:HIER_DIRECT api.github.com:443 "-"');
            (expect(result).not.toBeNull(), expect(result.destIpPort).toBe("Local"), expect(result.domain).toBe("api.github.com:443"));
          }),
          test("should parse line with non-numeric status code", () => {
            const result = parseFirewallLogLine('1761332530.474 172.30.0.20:35288 api.github.com:443 140.82.112.22:443 1.1 CONNECT Swap TCP_TUNNEL:HIER_DIRECT api.github.com:443 "-"');
            (expect(result).not.toBeNull(), expect(result.status).toBe("Swap"), expect(result.decision).toBe("TCP_TUNNEL:HIER_DIRECT"));
          }),
          test("should parse line with non-standard decision format", () => {
            const result = parseFirewallLogLine('1761332530.474 172.30.0.20:35288 api.github.com:443 140.82.112.22:443 1.1 CONNECT 200 Waiting api.github.com:443 "-"');
            (expect(result).not.toBeNull(), expect(result.decision).toBe("Waiting"), expect(result.status).toBe("200"));
          }),
          test("should parse line with pipe character in domain position", () => {
            const result = parseFirewallLogLine('1761332530.474 172.30.0.20:35288 pinger|test 140.82.112.22:443 1.1 CONNECT 200 TCP_TUNNEL:HIER_DIRECT api.github.com:443 "-"');
            (expect(result).not.toBeNull(), expect(result.domain).toBe("pinger|test"), expect(result.decision).toBe("TCP_TUNNEL:HIER_DIRECT"));
          }));
      }),
      describe("isRequestAllowed", () => {
        (test("should allow request with status 200", () => {
          expect(isRequestAllowed("TCP_TUNNEL:HIER_DIRECT", "200")).toBe(!0);
        }),
          test("should deny request with NONE_NONE decision", () => {
            expect(isRequestAllowed("NONE_NONE:HIER_NONE", "0")).toBe(!1);
          }));
      }),
      describe("analyzeFirewallLogLines", () => {
        (test("should skip internal Squid error entries with -:- destination and count only real traffic", () => {
          const lines = [
            '1773003472.027 ::1:52010 - -:- 0.0 - 0 NONE_NONE:HIER_NONE error:transaction-end-before-headers "-"',
            '1773003475.167 172.30.0.30:50232 api.anthropic.com:443 18.64.224.91:443 1.1 CONNECT 200 TCP_TUNNEL:HIER_DIRECT api.anthropic.com:443 "-"',
            '1773003477.068 ::1:35712 - -:- 0.0 - 0 NONE_NONE:HIER_NONE error:transaction-end-before-headers "-"',
            '1773003480.123 172.30.0.30:50235 api.anthropic.com:443 18.64.224.91:443 1.1 CONNECT 200 TCP_TUNNEL:HIER_DIRECT api.anthropic.com:443 "-"',
            '1773003481.456 ::1:41200 - - 0.0 - 0 NONE_NONE:HIER_NONE error:transaction-end-before-headers "-"',
          ];
          const result = analyzeFirewallLogLines(lines);
          (expect(result.totalRequests).toBe(2),
            expect(result.allowedRequests).toBe(2),
            expect(result.blockedRequests).toBe(0),
            expect(result.requestsByDomain.has("-:-")).toBe(!1),
            expect(result.requestsByDomain.get("api.anthropic.com:443").allowed).toBe(2));
        }),
          test("should not inflate blocked count with internal Squid error entries", () => {
            // Reproduces the scenario described in the bug report (run 22831150866)
            const lines = [
              '1773003472.027 ::1:52010 - -:- 0.0 - 0 NONE_NONE:HIER_NONE error:transaction-end-before-headers "-"',
              '1773003472.028 ::1:52011 - -:- 0.0 - 0 NONE_NONE:HIER_NONE error:transaction-end-before-headers "-"',
              '1773003475.167 172.30.0.30:50232 api.anthropic.com:443 18.64.224.91:443 1.1 CONNECT 200 TCP_TUNNEL:HIER_DIRECT api.anthropic.com:443 "-"',
              '1773003475.168 172.30.0.30:50233 blocked.example.com:443 1.2.3.4:443 1.1 CONNECT 403 NONE_NONE:HIER_NONE blocked.example.com:443 "-"',
            ];
            const result = analyzeFirewallLogLines(lines);
            (expect(result.totalRequests).toBe(2),
              expect(result.blockedRequests).toBe(1),
              expect(result.allowedRequests).toBe(1),
              expect(result.blockedDomains.has("-:-")).toBe(!1),
              expect(result.blockedDomains.has("blocked.example.com:443")).toBe(!0));
          }),
          test("should not treat -:- as a real blocked destination (domain fallback fix)", () => {
            // When domain="-" and destIpPort="-:-", should not fall back to "-:-" as domain key
            const lines = ['1773003472.027 172.30.0.20:50000 - -:- 0.0 - 0 NONE_NONE:HIER_NONE - "-"'];
            const result = analyzeFirewallLogLines(lines);
            (expect(result.totalRequests).toBe(1), expect(result.requestsByDomain.has("-:-")).toBe(!1), expect(result.requestsByDomain.has("-")).toBe(!0));
          }),
          test("should still count iptables-dropped traffic with real destIpPort", () => {
            // When domain="-" but destIpPort is a real IP:port (iptables-dropped), use destIpPort as key
            const lines = ['1761332531.123 172.30.0.20:35289 - 8.8.8.8:53 - - 0 NONE_NONE:HIER_NONE - "-"', '1761332532.456 172.30.0.20:35290 - 1.2.3.4:443 - - 0 NONE_NONE:HIER_NONE - "-"'];
            const result = analyzeFirewallLogLines(lines);
            (expect(result.totalRequests).toBe(2), expect(result.blockedRequests).toBe(2), expect(result.requestsByDomain.get("8.8.8.8:53").blocked).toBe(1), expect(result.requestsByDomain.get("1.2.3.4:443").blocked).toBe(1));
          }));
      }),
      describe("generateFirewallSummary", () => {
        (test("should generate summary with details/summary structure", () => {
          const analysis = {
              totalRequests: 5,
              allowedRequests: 3,
              blockedRequests: 2,
              allowedDomains: ["api.github.com:443", "api.npmjs.org:443"],
              blockedDomains: ["blocked.example.com:443", "denied.test.com:443"],
              requestsByDomain: new Map([
                ["api.github.com:443", { allowed: 2, blocked: 0 }],
                ["api.npmjs.org:443", { allowed: 1, blocked: 0 }],
                ["blocked.example.com:443", { allowed: 0, blocked: 1 }],
                ["denied.test.com:443", { allowed: 0, blocked: 1 }],
              ]),
            },
            summary = generateFirewallSummary(analysis);
          (expect(summary).toContain("sandbox agent:"),
            expect(summary).toContain("<details>"),
            expect(summary).toContain("</details>"),
            expect(summary).toContain("<summary>sandbox agent: 5 requests"),
            expect(summary).toContain("3 allowed"),
            expect(summary).toContain("2 blocked"),
            expect(summary).toContain("4 unique domains</summary>"),
            expect(summary).toContain("| Domain | Allowed | Blocked |"),
            expect(summary).toContain("| api.github.com:443 | 2 | 0 |"),
            expect(summary).toContain("| api.npmjs.org:443 | 1 | 0 |"),
            expect(summary).toContain("| blocked.example.com:443 | 0 | 1 |"),
            expect(summary).toContain("| denied.test.com:443 | 0 | 1 |"));
        }),
          test("should filter out placeholder domains", () => {
            const analysis = {
                totalRequests: 5,
                allowedRequests: 2,
                blockedRequests: 3,
                allowedDomains: ["api.github.com:443"],
                blockedDomains: ["-", "example.com:443"],
                requestsByDomain: new Map([
                  ["-", { allowed: 0, blocked: 2 }],
                  ["api.github.com:443", { allowed: 2, blocked: 0 }],
                  ["example.com:443", { allowed: 0, blocked: 1 }],
                ]),
              },
              summary = generateFirewallSummary(analysis);
            (expect(summary).toContain("2 unique domains"),
              expect(summary).toContain("2 allowed"),
              expect(summary).toContain("1 blocked"),
              expect(summary).toContain("| api.github.com:443 | 2 | 0 |"),
              expect(summary).toContain("| example.com:443 | 0 | 1 |"),
              expect(summary).not.toContain("| - |"));
          }),
          test("should show appropriate message when no firewall activity", () => {
            const analysis = { totalRequests: 3, allowedRequests: 3, blockedRequests: 0, allowedDomains: ["api.github.com:443"], blockedDomains: [], requestsByDomain: new Map([["api.github.com:443", { allowed: 3, blocked: 0 }]]) },
              summary = generateFirewallSummary(analysis);
            (expect(summary).toContain("sandbox agent:"),
              expect(summary).toContain("3 requests"),
              expect(summary).toContain("3 allowed"),
              expect(summary).toContain("0 blocked"),
              expect(summary).toContain("1 unique domain"),
              expect(summary).toContain("| api.github.com:443 | 3 | 0 |"));
          }),
          test("should show appropriate message when only placeholder domains are blocked", () => {
            const analysis = {
                totalRequests: 3,
                allowedRequests: 2,
                blockedRequests: 1,
                allowedDomains: ["api.github.com:443"],
                blockedDomains: ["-"],
                requestsByDomain: new Map([
                  ["-", { allowed: 0, blocked: 1 }],
                  ["api.github.com:443", { allowed: 2, blocked: 0 }],
                ]),
              },
              summary = generateFirewallSummary(analysis);
            (expect(summary).toContain("1 unique domain"), expect(summary).toContain("2 allowed"), expect(summary).toContain("0 blocked"), expect(summary).toContain("| api.github.com:443 | 2 | 0 |"));
          }),
          test("should show appropriate message when no valid domains", () => {
            const analysis = { totalRequests: 0, allowedRequests: 0, blockedRequests: 0, allowedDomains: [], blockedDomains: [], requestsByDomain: new Map() },
              summary = generateFirewallSummary(analysis);
            (expect(summary).toContain("sandbox agent:"),
              expect(summary).toContain("0 requests"),
              expect(summary).toContain("0 allowed"),
              expect(summary).toContain("0 blocked"),
              expect(summary).toContain("0 unique domains"),
              expect(summary).toContain("No firewall activity detected."));
          }),
          test("should filter out Envoy error codes", () => {
            const analysis = {
                totalRequests: 10,
                allowedRequests: 5,
                blockedRequests: 5,
                allowedDomains: ["api.github.com:443"],
                blockedDomains: ["error:transaction-end-before-headers", "blocked.example.com:443"],
                requestsByDomain: new Map([
                  ["error:transaction-end-before-headers", { allowed: 0, blocked: 5 }],
                  ["api.github.com:443", { allowed: 5, blocked: 0 }],
                  ["blocked.example.com:443", { allowed: 0, blocked: 1 }],
                ]),
              },
              summary = generateFirewallSummary(analysis);
            (expect(summary).toContain("2 unique domains"),
              expect(summary).toContain("5 allowed"),
              expect(summary).toContain("1 blocked"),
              expect(summary).toContain("| api.github.com:443 | 5 | 0 |"),
              expect(summary).toContain("| blocked.example.com:443 | 0 | 1 |"),
              expect(summary).not.toContain("error:transaction-end-before-headers"));
          }),
          test("should filter out all error codes starting with error:", () => {
            const analysis = {
                totalRequests: 15,
                allowedRequests: 10,
                blockedRequests: 5,
                allowedDomains: ["api.github.com:443"],
                blockedDomains: ["error:transaction-end-before-headers", "error:timeout", "error:connection-refused"],
                requestsByDomain: new Map([
                  ["error:transaction-end-before-headers", { allowed: 0, blocked: 2 }],
                  ["error:timeout", { allowed: 0, blocked: 2 }],
                  ["error:connection-refused", { allowed: 0, blocked: 1 }],
                  ["api.github.com:443", { allowed: 10, blocked: 0 }],
                ]),
              },
              summary = generateFirewallSummary(analysis);
            (expect(summary).toContain("1 unique domain"),
              expect(summary).toContain("10 allowed"),
              expect(summary).toContain("0 blocked"),
              expect(summary).toContain("| api.github.com:443 | 10 | 0 |"),
              expect(summary).not.toContain("error:"));
          }));
      }));
  }));
