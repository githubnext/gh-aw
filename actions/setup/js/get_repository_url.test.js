import { describe, it, expect, beforeEach, vi } from "vitest";

// Mock the global objects that GitHub Actions provides
const mockCore = {
  debug: vi.fn(),
  info: vi.fn(),
  warning: vi.fn(),
  error: vi.fn(),
  setFailed: vi.fn(),
  setOutput: vi.fn(),
  summary: {
    addRaw: vi.fn().mockReturnThis(),
    write: vi.fn().mockResolvedValue(),
  },
};

const mockContext = {
  runId: 12345,
  repo: {
    owner: "testowner",
    repo: "testrepo",
  },
  payload: {
    repository: {
      html_url: "https://github.com/testowner/testrepo",
    },
  },
};

// Set up global mocks before importing the module
global.core = mockCore;
global.context = mockContext;

describe("get_repository_url.cjs", () => {
  let getRepositoryUrl;

  beforeEach(async () => {
    // Reset mocks
    vi.clearAllMocks();

    // Reset environment variables
    delete process.env.GH_AW_TARGET_REPO_SLUG;
    delete process.env.GITHUB_SERVER_URL;

    // Reset context
    global.context = {
      runId: 12345,
      repo: {
        owner: "testowner",
        repo: "testrepo",
      },
      payload: {
        repository: {
          html_url: "https://github.com/testowner/testrepo",
        },
      },
    };

    // Dynamic import to get fresh module state
    const module = await import("./get_repository_url.cjs");
    getRepositoryUrl = module.getRepositoryUrl;
  });

  describe("getRepositoryUrl", () => {
    it("should return repository URL from context payload", () => {
      const result = getRepositoryUrl();

      expect(result).toBe("https://github.com/testowner/testrepo");
    });

    it("should return target repository URL in trial mode", () => {
      process.env.GH_AW_TARGET_REPO_SLUG = "targetowner/targetrepo";

      const result = getRepositoryUrl();

      expect(result).toBe("https://github.com/targetowner/targetrepo");
    });

    it("should use custom GitHub server URL in trial mode", () => {
      process.env.GH_AW_TARGET_REPO_SLUG = "targetowner/targetrepo";
      process.env.GITHUB_SERVER_URL = "https://github.enterprise.com";

      const result = getRepositoryUrl();

      expect(result).toBe("https://github.enterprise.com/targetowner/targetrepo");
    });

    it("should fallback to context repo when payload is missing", () => {
      global.context.payload = {};

      const result = getRepositoryUrl();

      expect(result).toBe("https://github.com/testowner/testrepo");
    });

    it("should use custom GitHub server URL in fallback", () => {
      global.context.payload = {};
      process.env.GITHUB_SERVER_URL = "https://github.enterprise.com";

      const result = getRepositoryUrl();

      expect(result).toBe("https://github.enterprise.com/testowner/testrepo");
    });

    it("should prioritize target repo over payload in trial mode", () => {
      process.env.GH_AW_TARGET_REPO_SLUG = "targetowner/targetrepo";
      global.context.payload = {
        repository: {
          html_url: "https://github.com/originalowner/originalrepo",
        },
      };

      const result = getRepositoryUrl();

      expect(result).toBe("https://github.com/targetowner/targetrepo");
    });
  });
});
