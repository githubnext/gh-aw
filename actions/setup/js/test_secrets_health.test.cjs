// @ts-check
import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";

describe("test_secrets_health", () => {
  let mockCore;
  let mockGithub;
  let mockContext;
  let originalGlobals;

  beforeEach(() => {
    // Save original globals
    originalGlobals = {
      core: global.core,
      github: global.github,
      context: global.context,
      fetch: global.fetch,
    };

    // Setup mocks
    mockCore = {
      info: vi.fn(),
      warning: vi.fn(),
      error: vi.fn(),
      summary: {
        addRaw: vi.fn().mockReturnThis(),
        write: vi.fn(),
      },
    };

    mockGithub = {
      rest: {
        users: {
          getAuthenticated: vi.fn(),
        },
      },
    };

    mockContext = {
      repo: {
        owner: "githubnext",
        repo: "gh-aw",
      },
    };

    // Set up globals
    global.core = mockCore;
    global.github = mockGithub;
    global.context = mockContext;
    global.fetch = vi.fn();

    // Clear environment variables
    delete process.env.COPILOT_GITHUB_TOKEN;
    delete process.env.GH_AW_GITHUB_TOKEN;
    delete process.env.GH_AW_GITHUB_MCP_SERVER_TOKEN;
    delete process.env.GH_AW_PROJECT_GITHUB_TOKEN;
    delete process.env.ANTHROPIC_API_KEY;
    delete process.env.OPENAI_API_KEY;
    delete process.env.BRAVE_API_KEY;
    delete process.env.NOTION_API_TOKEN;
    delete process.env.TAVILY_API_KEY;
  });

  afterEach(() => {
    // Restore original globals
    global.core = originalGlobals.core;
    global.github = originalGlobals.github;
    global.context = originalGlobals.context;
    global.fetch = originalGlobals.fetch;
    vi.clearAllMocks();
  });

  it("should report not_configured status when no secrets are set", async () => {
    const { main } = require("./test_secrets_health.cjs");

    await main();

    expect(mockCore.summary.addRaw).toHaveBeenCalledOnce();
    const summaryContent = mockCore.summary.addRaw.mock.calls[0][0];

    // Check for not configured section
    expect(summaryContent).toContain("### ⚠️ Not Configured Secrets");
    expect(summaryContent).toContain("COPILOT_GITHUB_TOKEN");
    expect(summaryContent).toContain("GH_AW_GITHUB_TOKEN");
  });

  it("should validate GitHub tokens successfully", async () => {
    const { main } = require("./test_secrets_health.cjs");

    // Mock successful GitHub API response
    global.fetch.mockResolvedValueOnce({
      ok: true,
      json: async () => ({ login: "testuser" }),
    });

    // Set a valid token
    process.env.COPILOT_GITHUB_TOKEN = "ghp_test_token";

    await main();

    expect(mockCore.summary.addRaw).toHaveBeenCalledOnce();
    const summaryContent = mockCore.summary.addRaw.mock.calls[0][0];

    // Check for valid section
    expect(summaryContent).toContain("### ✅ Valid Secrets");
    expect(summaryContent).toContain("COPILOT_GITHUB_TOKEN");
  });

  it("should handle invalid GitHub tokens", async () => {
    const { main } = require("./test_secrets_health.cjs");

    // Mock failed GitHub API response
    global.fetch.mockResolvedValueOnce({
      ok: false,
      status: 401,
      text: async () => "Unauthorized",
    });

    // Set an invalid token
    process.env.GH_AW_GITHUB_TOKEN = "invalid_token";

    await main();

    expect(mockCore.summary.addRaw).toHaveBeenCalledOnce();
    const summaryContent = mockCore.summary.addRaw.mock.calls[0][0];

    // Check for invalid section
    expect(summaryContent).toContain("### ❌ Invalid Secrets");
    expect(summaryContent).toContain("GH_AW_GITHUB_TOKEN");
  });

  it("should test Anthropic API key with valid response", async () => {
    const { main } = require("./test_secrets_health.cjs");

    // Mock successful API response
    global.fetch.mockResolvedValueOnce({
      ok: true,
      json: async () => ({}),
    });

    process.env.ANTHROPIC_API_KEY = "test_anthropic_key";

    await main();

    expect(global.fetch).toHaveBeenCalledWith(
      "https://api.anthropic.com/v1/messages",
      expect.objectContaining({
        method: "POST",
        headers: expect.objectContaining({
          "x-api-key": "test_anthropic_key",
        }),
      })
    );
  });

  it("should test OpenAI API key with valid response", async () => {
    const { main } = require("./test_secrets_health.cjs");

    // Mock successful API response
    global.fetch.mockResolvedValueOnce({
      ok: true,
      json: async () => ({ data: [] }),
    });

    process.env.OPENAI_API_KEY = "test_openai_key";

    await main();

    expect(global.fetch).toHaveBeenCalledWith(
      "https://api.openai.com/v1/models",
      expect.objectContaining({
        method: "GET",
        headers: expect.objectContaining({
          Authorization: "Bearer test_openai_key",
        }),
      })
    );
  });

  it("should generate comprehensive summary report", async () => {
    const { main } = require("./test_secrets_health.cjs");

    // Mock GitHub API response for valid token
    global.fetch.mockResolvedValue({
      ok: true,
      json: async () => ({ login: "testuser" }),
    });

    process.env.COPILOT_GITHUB_TOKEN = "valid_token";
    // Leave other tokens unset

    await main();

    const summaryContent = mockCore.summary.addRaw.mock.calls[0][0];

    // Check for all sections
    expect(summaryContent).toContain("## Secret Health Report");
    expect(summaryContent).toContain("**Summary**:");
    expect(summaryContent).toContain("📚 **Documentation**:");
    expect(summaryContent).toContain("🔧 **Setup**:");
  });
});
