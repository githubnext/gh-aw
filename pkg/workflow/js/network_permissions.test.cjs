import { describe, it, expect, beforeEach, vi } from "vitest";
import fs from "fs";
import path from "path";

// Mock the global objects that GitHub Actions provides
const mockCore = {
  setFailed: vi.fn(),
  error: vi.fn(),
  warning: vi.fn(),
};

// Set up global variables
global.core = mockCore;

describe("network_permissions.cjs", () => {
  let networkScript;
  let extractDomainFunc;
  let isDomainAllowedFunc;

  beforeEach(() => {
    // Reset all mocks
    vi.clearAllMocks();

    // Reset environment variables
    delete process.env.GITHUB_AW_NETWORK_DOMAINS;
    delete process.env.GITHUB_AW_TOOL_NAME;
    delete process.env.GITHUB_AW_TOOL_INPUT;

    // Read the script content
    const scriptPath = path.join(
      process.cwd(),
      "pkg/workflow/js/network_permissions.cjs"
    );
    networkScript = fs.readFileSync(scriptPath, "utf8");

    // Extract functions for unit testing
    const scriptForTesting = networkScript.replace(
      "await main();",
      `
      global.testExtractDomain = extractDomain;
      global.testIsDomainAllowed = isDomainAllowed;
      global.testMain = main;
      `
    );
    eval(scriptForTesting);

    extractDomainFunc = global.testExtractDomain;
    isDomainAllowedFunc = global.testIsDomainAllowed;
  });

  afterEach(() => {
    // Clean up globals
    delete global.testExtractDomain;
    delete global.testIsDomainAllowed;
    delete global.testMain;
  });

  describe("extractDomain function", () => {
    it("should extract domain from HTTP URLs", () => {
      expect(extractDomainFunc("http://example.com/path")).toBe("example.com");
      expect(extractDomainFunc("http://api.github.com/repos")).toBe(
        "api.github.com"
      );
    });

    it("should extract domain from HTTPS URLs", () => {
      expect(extractDomainFunc("https://example.com/path")).toBe("example.com");
      expect(extractDomainFunc("https://subdomain.example.com/path")).toBe(
        "subdomain.example.com"
      );
    });

    it("should extract domain from search queries with site: syntax", () => {
      expect(extractDomainFunc("site:example.com search terms")).toBe(
        "example.com"
      );
      expect(extractDomainFunc("some query site:github.com more text")).toBe(
        "github.com"
      );
    });

    it("should handle URLs with ports", () => {
      expect(extractDomainFunc("https://example.com:8080/path")).toBe(
        "example.com"
      );
    });

    it("should handle URLs with query parameters", () => {
      expect(extractDomainFunc("https://example.com/path?query=value")).toBe(
        "example.com"
      );
    });

    it("should return null for invalid inputs", () => {
      expect(extractDomainFunc(null)).toBe(null);
      expect(extractDomainFunc(undefined)).toBe(null);
      expect(extractDomainFunc("")).toBe(null);
      expect(extractDomainFunc("not a url")).toBe(null);
    });

    it("should return null for invalid URLs", () => {
      expect(extractDomainFunc("http://")).toBe(null);
      expect(extractDomainFunc("https://")).toBe(null);
    });

    it("should be case insensitive", () => {
      expect(extractDomainFunc("HTTPS://EXAMPLE.COM/path")).toBe("example.com");
      expect(extractDomainFunc("site:GITHUB.COM")).toBe("github.com");
    });
  });

  describe("isDomainAllowed function", () => {
    it("should allow exact domain matches", () => {
      const allowed = ["example.com", "github.com"];
      expect(isDomainAllowedFunc("example.com", allowed)).toBe(true);
      expect(isDomainAllowedFunc("github.com", allowed)).toBe(true);
      expect(isDomainAllowedFunc("other.com", allowed)).toBe(false);
    });

    it("should support wildcard patterns", () => {
      const allowed = ["*.example.com", "github.com"];
      expect(isDomainAllowedFunc("api.example.com", allowed)).toBe(true);
      expect(isDomainAllowedFunc("subdomain.example.com", allowed)).toBe(true);
      expect(isDomainAllowedFunc("example.com", allowed)).toBe(false); // wildcard doesn't match root
      expect(isDomainAllowedFunc("github.com", allowed)).toBe(true);
    });

    it("should handle empty allowed domains (deny-all)", () => {
      expect(isDomainAllowedFunc("example.com", [])).toBe(false);
      expect(isDomainAllowedFunc("github.com", [])).toBe(false);
    });

    it("should handle null/undefined domain with allowed domains", () => {
      const allowed = ["example.com"];
      expect(isDomainAllowedFunc(null, allowed)).toBe(true); // Has allowed domains, so permit
      expect(isDomainAllowedFunc(undefined, allowed)).toBe(true);
      expect(isDomainAllowedFunc("", allowed)).toBe(true);
    });

    it("should handle null/undefined domain with empty allowed domains", () => {
      expect(isDomainAllowedFunc(null, [])).toBe(false); // No allowed domains, so deny
      expect(isDomainAllowedFunc(undefined, [])).toBe(false);
      expect(isDomainAllowedFunc("", [])).toBe(false);
    });

    it("should be case insensitive", () => {
      const allowed = ["Example.COM", "GITHUB.com"];
      expect(isDomainAllowedFunc("example.com", allowed)).toBe(true);
      expect(isDomainAllowedFunc("github.com", allowed)).toBe(true);
    });

    it("should handle complex wildcard patterns", () => {
      const allowed = ["*.github.com", "*.example.*"];
      expect(isDomainAllowedFunc("api.github.com", allowed)).toBe(true);
      expect(isDomainAllowedFunc("raw.githubusercontent.com", allowed)).toBe(
        false
      );
      expect(isDomainAllowedFunc("test.example.org", allowed)).toBe(true);
      expect(isDomainAllowedFunc("sub.test.example.com", allowed)).toBe(true);
    });
  });

  describe("main function integration", () => {
    let mainFunc;

    beforeEach(() => {
      mainFunc = global.testMain;
    });

    it("should allow non-network tools", async () => {
      process.env.GITHUB_AW_TOOL_NAME = "SomeOtherTool";
      process.env.GITHUB_AW_NETWORK_DOMAINS = JSON.stringify([]);

      const consoleSpy = vi.spyOn(console, "log").mockImplementation(() => {});

      await mainFunc();

      expect(mockCore.setFailed).not.toHaveBeenCalled();
      expect(consoleSpy).toHaveBeenCalledWith(
        "Tool SomeOtherTool is not subject to network restrictions"
      );

      consoleSpy.mockRestore();
    });

    it("should allow WebFetch requests to allowed domains", async () => {
      process.env.GITHUB_AW_TOOL_NAME = "WebFetch";
      process.env.GITHUB_AW_NETWORK_DOMAINS = JSON.stringify([
        "example.com",
        "github.com",
      ]);
      process.env.GITHUB_AW_TOOL_INPUT = JSON.stringify({
        url: "https://example.com/page",
      });

      const consoleSpy = vi.spyOn(console, "log").mockImplementation(() => {});

      await mainFunc();

      expect(mockCore.setFailed).not.toHaveBeenCalled();
      expect(consoleSpy).toHaveBeenCalledWith(
        "Network access allowed for domain: example.com"
      );

      consoleSpy.mockRestore();
    });

    it("should block WebFetch requests to disallowed domains", async () => {
      process.env.GITHUB_AW_TOOL_NAME = "WebFetch";
      process.env.GITHUB_AW_NETWORK_DOMAINS = JSON.stringify(["example.com"]);
      process.env.GITHUB_AW_TOOL_INPUT = JSON.stringify({
        url: "https://malicious.com/page",
      });

      const consoleSpy = vi.spyOn(console, "log").mockImplementation(() => {});

      await mainFunc();

      expect(mockCore.setFailed).toHaveBeenCalledWith(
        "Network access blocked for domain: malicious.com. Allowed domains: example.com"
      );

      consoleSpy.mockRestore();
    });

    it("should block WebSearch with no domain under deny-all policy", async () => {
      process.env.GITHUB_AW_TOOL_NAME = "WebSearch";
      process.env.GITHUB_AW_NETWORK_DOMAINS = JSON.stringify([]);
      process.env.GITHUB_AW_TOOL_INPUT = JSON.stringify({
        query: "general search",
      });

      const consoleSpy = vi.spyOn(console, "log").mockImplementation(() => {});

      await mainFunc();

      expect(mockCore.setFailed).toHaveBeenCalledWith(
        "Network access blocked: deny-all policy in effect. No domains are allowed for WebSearch"
      );

      consoleSpy.mockRestore();
    });

    it("should block WebSearch with no domain when allowlist is configured", async () => {
      process.env.GITHUB_AW_TOOL_NAME = "WebSearch";
      process.env.GITHUB_AW_NETWORK_DOMAINS = JSON.stringify(["example.com"]);
      process.env.GITHUB_AW_TOOL_INPUT = JSON.stringify({
        query: "general search",
      });

      const consoleSpy = vi.spyOn(console, "log").mockImplementation(() => {});

      await mainFunc();

      expect(mockCore.setFailed).toHaveBeenCalledWith(
        "Network access blocked for web-search: no specific domain detected. Allowed domains: example.com"
      );

      consoleSpy.mockRestore();
    });

    it("should allow WebSearch with site-specific queries", async () => {
      process.env.GITHUB_AW_TOOL_NAME = "WebSearch";
      process.env.GITHUB_AW_NETWORK_DOMAINS = JSON.stringify(["github.com"]);
      process.env.GITHUB_AW_TOOL_INPUT = JSON.stringify({
        query: "site:github.com repository search",
      });

      const consoleSpy = vi.spyOn(console, "log").mockImplementation(() => {});

      await mainFunc();

      expect(mockCore.setFailed).not.toHaveBeenCalled();
      expect(consoleSpy).toHaveBeenCalledWith(
        "Network access allowed for domain: github.com"
      );

      consoleSpy.mockRestore();
    });

    it("should handle wildcard domains correctly", async () => {
      process.env.GITHUB_AW_TOOL_NAME = "WebFetch";
      process.env.GITHUB_AW_NETWORK_DOMAINS = JSON.stringify(["*.github.com"]);
      process.env.GITHUB_AW_TOOL_INPUT = JSON.stringify({
        url: "https://api.github.com/repos",
      });

      const consoleSpy = vi.spyOn(console, "log").mockImplementation(() => {});

      await mainFunc();

      expect(mockCore.setFailed).not.toHaveBeenCalled();
      expect(consoleSpy).toHaveBeenCalledWith(
        "Network access allowed for domain: api.github.com"
      );

      consoleSpy.mockRestore();
    });

    it("should handle invalid JSON in environment variables gracefully", async () => {
      process.env.GITHUB_AW_TOOL_NAME = "WebFetch";
      process.env.GITHUB_AW_NETWORK_DOMAINS = "invalid json";
      process.env.GITHUB_AW_TOOL_INPUT = JSON.stringify({
        url: "https://example.com",
      });

      const consoleSpy = vi.spyOn(console, "log").mockImplementation(() => {});

      await mainFunc();

      expect(mockCore.error).toHaveBeenCalledWith(
        expect.stringContaining("Failed to parse GITHUB_AW_NETWORK_DOMAINS")
      );
      // Should default to empty allowed domains (deny-all)
      expect(mockCore.setFailed).toHaveBeenCalledWith(
        "Network access blocked for domain: example.com. Allowed domains: "
      );

      consoleSpy.mockRestore();
    });

    it("should handle missing environment variables", async () => {
      // No environment variables set
      const consoleSpy = vi.spyOn(console, "log").mockImplementation(() => {});

      await mainFunc();

      expect(mockCore.setFailed).not.toHaveBeenCalled();
      expect(consoleSpy).toHaveBeenCalledWith(
        "Tool  is not subject to network restrictions"
      );

      consoleSpy.mockRestore();
    });

    it("should handle errors in main function gracefully", async () => {
      // Force an error by providing invalid tool input JSON
      process.env.GITHUB_AW_TOOL_NAME = "WebFetch";
      process.env.GITHUB_AW_NETWORK_DOMAINS = JSON.stringify(["example.com"]);
      process.env.GITHUB_AW_TOOL_INPUT = "invalid json";

      const consoleSpy = vi.spyOn(console, "log").mockImplementation(() => {});

      await mainFunc();

      expect(mockCore.error).toHaveBeenCalledWith(
        expect.stringContaining("Failed to parse GITHUB_AW_TOOL_INPUT")
      );

      consoleSpy.mockRestore();
    });
  });

  describe("edge cases and error handling", () => {
    it("should handle empty strings appropriately", () => {
      expect(extractDomainFunc("")).toBe(null);
      expect(isDomainAllowedFunc("", ["example.com"])).toBe(true); // Empty domain with allowlist
      expect(isDomainAllowedFunc("", [])).toBe(false); // Empty domain with deny-all
    });

    it("should handle URLs with unusual schemes", () => {
      expect(extractDomainFunc("ftp://example.com")).toBe(null); // Only http/https supported
      expect(extractDomainFunc("mailto:user@example.com")).toBe(null);
    });

    it("should handle malformed URLs gracefully", () => {
      expect(extractDomainFunc("https://")).toBe(null);
      expect(extractDomainFunc("https:///path")).toBe("path"); // URL constructor behavior
      expect(extractDomainFunc("https://[invalid")).toBe(null);
    });

    it("should handle special characters in domain patterns", () => {
      const allowed = ["test.example.com", "*.sub-domain.com"];
      expect(isDomainAllowedFunc("test.example.com", allowed)).toBe(true);
      expect(isDomainAllowedFunc("api.sub-domain.com", allowed)).toBe(true);
    });
  });
});
