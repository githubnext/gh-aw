import { describe, it, expect, beforeEach, vi } from "vitest";

// Mock the global objects that GitHub Actions provides
const mockCore = {
  // Core logging functions
  debug: vi.fn(),
  info: vi.fn(),
  notice: vi.fn(),
  warning: vi.fn(),
  error: vi.fn(),
  
  // Core workflow functions
  setFailed: vi.fn(),
  setOutput: vi.fn(),
  getInput: vi.fn(),
  
  // Group functions
  startGroup: vi.fn(),
  endGroup: vi.fn(),
  
  // State and summary
  saveState: vi.fn(),
  getState: vi.fn(),
  
  summary: {
    addRaw: vi.fn().mockReturnThis(),
    write: vi.fn(),
  },
};

// Set up global core mock
global.core = mockCore;

// Import the module after setting up mocks - we need to use dynamic import for CommonJS modules
const { extractDomain, isDomainAllowed, main } = await import("./network_permissions_hook.cjs");

describe("extractDomain", () => {
  it("should return null for empty or undefined input", () => {
    expect(extractDomain("")).toBe(null);
    expect(extractDomain(undefined)).toBe(null);
    expect(extractDomain(null)).toBe(null);
  });

  it("should extract domain from HTTP URLs", () => {
    expect(extractDomain("http://example.com/path")).toBe("example.com");
    expect(extractDomain("http://subdomain.example.com")).toBe("subdomain.example.com");
  });

  it("should extract domain from HTTPS URLs", () => {
    expect(extractDomain("https://example.com/path")).toBe("example.com");
    expect(extractDomain("https://api.github.com/repos")).toBe("api.github.com");
  });

  it("should extract domain from search queries with site: prefix", () => {
    expect(extractDomain("site:example.com query terms")).toBe("example.com");
    expect(extractDomain("some query site:github.com more terms")).toBe("github.com");
  });

  it("should return null for invalid URLs", () => {
    expect(extractDomain("not-a-url")).toBe(null);
    expect(extractDomain("just some text")).toBe(null);
  });

  it("should handle case insensitivity", () => {
    expect(extractDomain("https://EXAMPLE.COM/path")).toBe("example.com");
    expect(extractDomain("site:GITHUB.COM")).toBe("github.com");
  });
});

describe("isDomainAllowed", () => {
  it("should return false for deny-all policy (empty allowed domains)", () => {
    expect(isDomainAllowed("example.com", [])).toBe(false);
    expect(isDomainAllowed(null, [])).toBe(false);
  });

  it("should return true for no domain when allowed domains list is not empty", () => {
    expect(isDomainAllowed(null, ["example.com"])).toBe(true);
  });

  it("should allow exact domain matches", () => {
    const allowedDomains = ["example.com", "github.com"];
    expect(isDomainAllowed("example.com", allowedDomains)).toBe(true);
    expect(isDomainAllowed("github.com", allowedDomains)).toBe(true);
    expect(isDomainAllowed("notallowed.com", allowedDomains)).toBe(false);
  });

  it("should support wildcard patterns", () => {
    const allowedDomains = ["*.trusted.com", "api.*.example.org"];
    expect(isDomainAllowed("subdomain.trusted.com", allowedDomains)).toBe(true);
    expect(isDomainAllowed("deep.nested.trusted.com", allowedDomains)).toBe(true);
    expect(isDomainAllowed("api.v1.example.org", allowedDomains)).toBe(true);
    expect(isDomainAllowed("untrusted.com", allowedDomains)).toBe(false);
  });

  it("should handle mixed exact and wildcard patterns", () => {
    const allowedDomains = ["example.com", "*.trusted.com", "api.service.org"];
    expect(isDomainAllowed("example.com", allowedDomains)).toBe(true);
    expect(isDomainAllowed("sub.trusted.com", allowedDomains)).toBe(true);
    expect(isDomainAllowed("api.service.org", allowedDomains)).toBe(true);
    expect(isDomainAllowed("other.com", allowedDomains)).toBe(false);
  });
});

describe("main", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("should allow non-network tools without restrictions", async () => {
    mockCore.getInput.mockImplementation(name => {
      if (name === "allowed_domains") return '["example.com"]';
      if (name === "tool_data") return JSON.stringify({ tool_name: "SomeOtherTool" });
      return "";
    });

    await main();

    expect(mockCore.info).toHaveBeenCalledWith("Tool SomeOtherTool is not subject to network restrictions, allowing");
    expect(mockCore.setOutput).toHaveBeenCalledWith("allowed", "true");
  });

  it("should block WebFetch for disallowed domains", async () => {
    mockCore.getInput.mockImplementation(name => {
      if (name === "allowed_domains") return '["example.com"]';
      if (name === "tool_data")
        return JSON.stringify({
          tool_name: "WebFetch",
          tool_input: { url: "https://blocked.com/path" },
        });
      return "";
    });

    await main();

    expect(mockCore.error).toHaveBeenCalledWith("Network access blocked for domain: blocked.com");
    expect(mockCore.error).toHaveBeenCalledWith("Allowed domains: example.com");
    expect(mockCore.setOutput).toHaveBeenCalledWith("allowed", "false");
    expect(mockCore.setOutput).toHaveBeenCalledWith("reason", "domain blocked.com not allowed");
  });

  it("should allow WebFetch for allowed domains", async () => {
    mockCore.getInput.mockImplementation(name => {
      if (name === "allowed_domains") return '["example.com", "*.trusted.com"]';
      if (name === "tool_data")
        return JSON.stringify({
          tool_name: "WebFetch",
          tool_input: { url: "https://api.trusted.com/endpoint" },
        });
      return "";
    });

    await main();

    expect(mockCore.info).toHaveBeenCalledWith("Network access allowed for domain: api.trusted.com");
    expect(mockCore.setOutput).toHaveBeenCalledWith("allowed", "true");
  });

  it("should block WebSearch with no domain under deny-all policy", async () => {
    mockCore.getInput.mockImplementation(name => {
      if (name === "allowed_domains") return "[]";
      if (name === "tool_data")
        return JSON.stringify({
          tool_name: "WebSearch",
          tool_input: { query: "general search query" },
        });
      return "";
    });

    await main();

    expect(mockCore.error).toHaveBeenCalledWith("Network access blocked: deny-all policy in effect");
    expect(mockCore.error).toHaveBeenCalledWith("No domains are allowed for WebSearch");
    expect(mockCore.setOutput).toHaveBeenCalledWith("allowed", "false");
    expect(mockCore.setOutput).toHaveBeenCalledWith("reason", "deny-all policy in effect");
  });

  it("should block WebSearch with no domain when domain allowlist is configured", async () => {
    mockCore.getInput.mockImplementation(name => {
      if (name === "allowed_domains") return '["example.com"]';
      if (name === "tool_data")
        return JSON.stringify({
          tool_name: "WebSearch",
          tool_input: { query: "general search query" },
        });
      return "";
    });

    await main();

    expect(mockCore.error).toHaveBeenCalledWith("Network access blocked for web-search: no specific domain detected");
    expect(mockCore.error).toHaveBeenCalledWith("Allowed domains: example.com");
    expect(mockCore.setOutput).toHaveBeenCalledWith("allowed", "false");
    expect(mockCore.setOutput).toHaveBeenCalledWith("reason", "no specific domain detected");
  });

  it("should handle parsing errors gracefully", async () => {
    mockCore.getInput.mockImplementation(name => {
      if (name === "allowed_domains") return "invalid-json";
      if (name === "tool_data") return "{}";
      return "";
    });

    await main();

    expect(mockCore.setFailed).toHaveBeenCalledWith(expect.stringContaining("Error parsing allowed_domains input"));
  });

  it("should handle tool_data parsing errors gracefully", async () => {
    mockCore.getInput.mockImplementation(name => {
      if (name === "allowed_domains") return "[]";
      if (name === "tool_data") return "invalid-json";
      return "";
    });

    await main();

    expect(mockCore.setFailed).toHaveBeenCalledWith(expect.stringContaining("Error parsing tool_data input"));
  });

  it("should handle missing tool_data input", async () => {
    mockCore.getInput.mockImplementation((name, options) => {
      if (name === "allowed_domains") return "[]";
      // tool_data is required, so this should cause an error in getInput
      if (name === "tool_data" && options?.required) {
        throw new Error("Input required and not supplied: tool_data");
      }
      return "";
    });

    await main();

    // The main function should catch the error and set the output accordingly
    expect(mockCore.error).toHaveBeenCalledWith(expect.stringContaining("Network validation error"));
    expect(mockCore.setOutput).toHaveBeenCalledWith("allowed", "false");
    expect(mockCore.setOutput).toHaveBeenCalledWith("reason", "validation error");
  });
});
