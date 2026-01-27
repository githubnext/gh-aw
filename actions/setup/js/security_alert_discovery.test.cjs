// @ts-check
import { describe, it, expect, beforeEach, vi } from "vitest";
import { discoverSecurityAlerts } from "./security_alert_discovery.cjs";

// Mock core
global.core = {
  info: vi.fn(),
  warning: vi.fn(),
  error: vi.fn(),
  setFailed: vi.fn(),
  getInput: vi.fn(),
  setOutput: vi.fn(),
};

describe("security_alert_discovery", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  describe("discoverSecurityAlerts", () => {
    it("should discover code scanning alerts", async () => {
      const octokit = {
        rest: {
          codeScanning: {
            listAlertsForRepo: vi.fn().mockResolvedValue({
              data: [
                {
                  number: 1,
                  html_url: "https://github.com/owner/repo/security/code-scanning/1",
                  state: "open",
                  rule: {
                    id: "go/unsafe-quoting",
                    severity: "critical",
                    description: "Unsafe quoting in code",
                  },
                  created_at: "2025-01-01T00:00:00Z",
                  updated_at: "2025-01-02T00:00:00Z",
                },
                {
                  number: 2,
                  html_url: "https://github.com/owner/repo/security/code-scanning/2",
                  state: "open",
                  rule: {
                    id: "js/xss",
                    severity: "high",
                    description: "Cross-site scripting vulnerability",
                  },
                  created_at: "2025-01-01T00:00:00Z",
                  updated_at: "2025-01-02T00:00:00Z",
                },
              ],
            }),
          },
          secretScanning: {
            listAlertsForRepo: vi.fn().mockResolvedValue({ data: [] }),
          },
          dependabot: {
            listAlertsForRepo: vi.fn().mockResolvedValue({ data: [] }),
          },
        },
      };

      const result = await discoverSecurityAlerts(octokit, ["owner/repo"]);

      expect(result).not.toBeNull();
      expect(result.code_scanning.total).toBe(2);
      expect(result.code_scanning.by_severity.critical).toBe(1);
      expect(result.code_scanning.by_severity.high).toBe(1);
      expect(result.code_scanning.items).toHaveLength(2);
      expect(result.code_scanning.items[0]).toMatchObject({
        type: "code_scanning",
        number: 1,
        severity: "critical",
        rule_id: "go/unsafe-quoting",
      });
    });

    it("should discover secret scanning alerts", async () => {
      const octokit = {
        rest: {
          codeScanning: {
            listAlertsForRepo: vi.fn().mockResolvedValue({ data: [] }),
          },
          secretScanning: {
            listAlertsForRepo: vi.fn().mockResolvedValue({
              data: [
                {
                  number: 10,
                  html_url: "https://github.com/owner/repo/security/secret-scanning/10",
                  state: "open",
                  secret_type: "github_personal_access_token",
                  created_at: "2025-01-01T00:00:00Z",
                  updated_at: "2025-01-02T00:00:00Z",
                },
              ],
            }),
          },
          dependabot: {
            listAlertsForRepo: vi.fn().mockResolvedValue({ data: [] }),
          },
        },
      };

      const result = await discoverSecurityAlerts(octokit, ["owner/repo"]);

      expect(result).not.toBeNull();
      expect(result.secret_scanning.total).toBe(1);
      expect(result.secret_scanning.items).toHaveLength(1);
      expect(result.secret_scanning.items[0]).toMatchObject({
        type: "secret_scanning",
        number: 10,
        secret_type: "github_personal_access_token",
      });
    });

    it("should discover dependabot alerts", async () => {
      const octokit = {
        rest: {
          codeScanning: {
            listAlertsForRepo: vi.fn().mockResolvedValue({ data: [] }),
          },
          secretScanning: {
            listAlertsForRepo: vi.fn().mockResolvedValue({ data: [] }),
          },
          dependabot: {
            listAlertsForRepo: vi.fn().mockResolvedValue({
              data: [
                {
                  number: 20,
                  html_url: "https://github.com/owner/repo/security/dependabot/20",
                  state: "open",
                  security_advisory: {
                    severity: "high",
                    package: { name: "lodash" },
                  },
                  created_at: "2025-01-01T00:00:00Z",
                  updated_at: "2025-01-02T00:00:00Z",
                },
              ],
            }),
          },
        },
      };

      const result = await discoverSecurityAlerts(octokit, ["owner/repo"]);

      expect(result).not.toBeNull();
      expect(result.dependabot.total).toBe(1);
      expect(result.dependabot.items).toHaveLength(1);
      expect(result.dependabot.items[0]).toMatchObject({
        type: "dependabot",
        number: 20,
        severity: "high",
        package_name: "lodash",
      });
    });

    it("should handle API errors gracefully", async () => {
      const octokit = {
        rest: {
          codeScanning: {
            listAlertsForRepo: vi.fn().mockRejectedValue(new Error("API error")),
          },
          secretScanning: {
            listAlertsForRepo: vi.fn().mockRejectedValue(new Error("API error")),
          },
          dependabot: {
            listAlertsForRepo: vi.fn().mockRejectedValue(new Error("API error")),
          },
        },
      };

      const result = await discoverSecurityAlerts(octokit, ["owner/repo"]);

      // Should return empty results instead of throwing
      expect(result).not.toBeNull();
      expect(result.code_scanning.total).toBe(0);
      expect(result.secret_scanning.total).toBe(0);
      expect(result.dependabot.total).toBe(0);
    });

    it("should return null if no repos specified", async () => {
      const octokit = {};
      const result = await discoverSecurityAlerts(octokit, []);
      expect(result).toBeNull();
    });

    it("should handle invalid repo format", async () => {
      const octokit = {
        rest: {
          codeScanning: {
            listAlertsForRepo: vi.fn().mockResolvedValue({ data: [] }),
          },
          secretScanning: {
            listAlertsForRepo: vi.fn().mockResolvedValue({ data: [] }),
          },
          dependabot: {
            listAlertsForRepo: vi.fn().mockResolvedValue({ data: [] }),
          },
        },
      };

      const result = await discoverSecurityAlerts(octokit, ["invalid-repo-format"]);

      // Should skip invalid repos but still return results
      expect(result).not.toBeNull();
      expect(result.code_scanning.total).toBe(0);
    });

    it("should aggregate alerts from multiple repos", async () => {
      const octokit = {
        rest: {
          codeScanning: {
            listAlertsForRepo: vi.fn().mockResolvedValue({
              data: [
                {
                  number: 1,
                  html_url: "https://github.com/owner/repo/security/code-scanning/1",
                  state: "open",
                  rule: { id: "test", severity: "high", description: "Test" },
                  created_at: "2025-01-01T00:00:00Z",
                  updated_at: "2025-01-02T00:00:00Z",
                },
              ],
            }),
          },
          secretScanning: {
            listAlertsForRepo: vi.fn().mockResolvedValue({ data: [] }),
          },
          dependabot: {
            listAlertsForRepo: vi.fn().mockResolvedValue({ data: [] }),
          },
        },
      };

      const result = await discoverSecurityAlerts(octokit, ["owner/repo1", "owner/repo2"]);

      expect(result).not.toBeNull();
      // Should have called the API for each repo
      expect(octokit.rest.codeScanning.listAlertsForRepo).toHaveBeenCalledTimes(2);
      // Should aggregate totals from both repos
      expect(result.code_scanning.total).toBe(2);
    });
  });
});
