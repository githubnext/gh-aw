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

const mockGithub = {
  rest: {
    repos: {
      getCollaboratorPermissionLevel: vi.fn(),
    },
  },
};

// Set up global mocks before importing the module
global.core = mockCore;
global.github = mockGithub;

describe("check_permissions_utils", () => {
  let parseRequiredPermissions;
  let parseAllowedBots;
  let checkRepositoryPermission;
  let checkBotStatus;
  let originalEnv;

  beforeEach(async () => {
    // Reset all mocks
    vi.clearAllMocks();

    // Store original environment
    originalEnv = {
      GH_AW_REQUIRED_ROLES: process.env.GH_AW_REQUIRED_ROLES,
      GH_AW_ALLOWED_BOTS: process.env.GH_AW_ALLOWED_BOTS,
    };

    // Import the module functions
    const module = await import("./check_permissions_utils.cjs");
    parseRequiredPermissions = module.parseRequiredPermissions;
    parseAllowedBots = module.parseAllowedBots;
    checkRepositoryPermission = module.checkRepositoryPermission;
    checkBotStatus = module.checkBotStatus;
  });

  afterEach(() => {
    // Restore original environment
    Object.keys(originalEnv).forEach(key => {
      if (originalEnv[key] !== undefined) {
        process.env[key] = originalEnv[key];
      } else {
        delete process.env[key];
      }
    });
  });

  describe("parseAllowedBots", () => {
    it("should parse comma-separated bot identifiers", () => {
      process.env.GH_AW_ALLOWED_BOTS = "dependabot[bot],renovate[bot],github-actions[bot]";
      const result = parseAllowedBots();
      expect(result).toEqual(["dependabot[bot]", "renovate[bot]", "github-actions[bot]"]);
    });

    it("should filter out empty strings", () => {
      process.env.GH_AW_ALLOWED_BOTS = "dependabot[bot],,renovate[bot],";
      const result = parseAllowedBots();
      expect(result).toEqual(["dependabot[bot]", "renovate[bot]"]);
    });

    it("should filter out whitespace-only entries", () => {
      process.env.GH_AW_ALLOWED_BOTS = "dependabot[bot], ,renovate[bot]";
      const result = parseAllowedBots();
      expect(result).toEqual(["dependabot[bot]", "renovate[bot]"]);
    });

    it("should return empty array when env var is not set", () => {
      delete process.env.GH_AW_ALLOWED_BOTS;
      const result = parseAllowedBots();
      expect(result).toEqual([]);
    });

    it("should return empty array when env var is empty string", () => {
      process.env.GH_AW_ALLOWED_BOTS = "";
      const result = parseAllowedBots();
      expect(result).toEqual([]);
    });

    it("should handle single bot identifier", () => {
      process.env.GH_AW_ALLOWED_BOTS = "dependabot[bot]";
      const result = parseAllowedBots();
      expect(result).toEqual(["dependabot[bot]"]);
    });
  });

  describe("parseRequiredPermissions", () => {
    it("should parse comma-separated permissions", () => {
      process.env.GH_AW_REQUIRED_ROLES = "admin,write,read";
      const result = parseRequiredPermissions();
      expect(result).toEqual(["admin", "write", "read"]);
    });

    it("should filter out empty strings", () => {
      process.env.GH_AW_REQUIRED_ROLES = "admin,,write,";
      const result = parseRequiredPermissions();
      expect(result).toEqual(["admin", "write"]);
    });

    it("should filter out whitespace-only entries", () => {
      process.env.GH_AW_REQUIRED_ROLES = "admin, ,write";
      const result = parseRequiredPermissions();
      expect(result).toEqual(["admin", "write"]);
    });

    it("should return empty array when env var is not set", () => {
      delete process.env.GH_AW_REQUIRED_ROLES;
      const result = parseRequiredPermissions();
      expect(result).toEqual([]);
    });

    it("should return empty array when env var is empty string", () => {
      process.env.GH_AW_REQUIRED_ROLES = "";
      const result = parseRequiredPermissions();
      expect(result).toEqual([]);
    });

    it("should handle single permission", () => {
      process.env.GH_AW_REQUIRED_ROLES = "admin";
      const result = parseRequiredPermissions();
      expect(result).toEqual(["admin"]);
    });

    it("should preserve original values without trimming", () => {
      process.env.GH_AW_REQUIRED_ROLES = "admin,write";
      const result = parseRequiredPermissions();
      expect(result).toEqual(["admin", "write"]);
    });
  });

  describe("checkRepositoryPermission", () => {
    it("should return authorized when user has exact permission match", async () => {
      mockGithub.rest.repos.getCollaboratorPermissionLevel.mockResolvedValue({
        data: { permission: "admin" },
      });

      const result = await checkRepositoryPermission("testuser", "testowner", "testrepo", ["admin", "write"]);

      expect(result).toEqual({
        authorized: true,
        permission: "admin",
      });

      expect(mockGithub.rest.repos.getCollaboratorPermissionLevel).toHaveBeenCalledWith({
        owner: "testowner",
        repo: "testrepo",
        username: "testuser",
      });

      expect(mockCore.info).toHaveBeenCalledWith("Checking if user 'testuser' has required permissions for testowner/testrepo");
      expect(mockCore.info).toHaveBeenCalledWith("Required permissions: admin, write");
      expect(mockCore.info).toHaveBeenCalledWith("Repository permission level: admin");
      expect(mockCore.info).toHaveBeenCalledWith("✅ User has admin access to repository");
    });

    it("should return authorized for maintain when maintainer is required", async () => {
      mockGithub.rest.repos.getCollaboratorPermissionLevel.mockResolvedValue({
        data: { permission: "maintain" },
      });

      const result = await checkRepositoryPermission("testuser", "testowner", "testrepo", ["maintainer"]);

      expect(result).toEqual({
        authorized: true,
        permission: "maintain",
      });

      expect(mockCore.info).toHaveBeenCalledWith("✅ User has maintain access to repository");
    });

    it("should return unauthorized when user has insufficient permissions", async () => {
      mockGithub.rest.repos.getCollaboratorPermissionLevel.mockResolvedValue({
        data: { permission: "read" },
      });

      const result = await checkRepositoryPermission("testuser", "testowner", "testrepo", ["admin", "write"]);

      expect(result).toEqual({
        authorized: false,
        permission: "read",
      });

      expect(mockCore.warning).toHaveBeenCalledWith("User permission 'read' does not meet requirements: admin, write");
    });

    it("should return error on API failure", async () => {
      const apiError = new Error("API Error: Not Found");
      mockGithub.rest.repos.getCollaboratorPermissionLevel.mockRejectedValue(apiError);

      const result = await checkRepositoryPermission("testuser", "testowner", "testrepo", ["admin"]);

      expect(result).toEqual({
        authorized: false,
        error: "API Error: Not Found",
      });

      expect(mockCore.warning).toHaveBeenCalledWith("Repository permission check failed: API Error: Not Found");
    });

    it("should handle non-Error API failures", async () => {
      mockGithub.rest.repos.getCollaboratorPermissionLevel.mockRejectedValue("String error");

      const result = await checkRepositoryPermission("testuser", "testowner", "testrepo", ["admin"]);

      expect(result).toEqual({
        authorized: false,
        error: "String error",
      });

      expect(mockCore.warning).toHaveBeenCalledWith("Repository permission check failed: String error");
    });

    it("should check multiple permissions and return true for any match", async () => {
      mockGithub.rest.repos.getCollaboratorPermissionLevel.mockResolvedValue({
        data: { permission: "write" },
      });

      const result = await checkRepositoryPermission("testuser", "testowner", "testrepo", ["admin", "write", "triage"]);

      expect(result).toEqual({
        authorized: true,
        permission: "write",
      });

      expect(mockCore.info).toHaveBeenCalledWith("✅ User has write access to repository");
    });

    it("should handle triage permission", async () => {
      mockGithub.rest.repos.getCollaboratorPermissionLevel.mockResolvedValue({
        data: { permission: "triage" },
      });

      const result = await checkRepositoryPermission("testuser", "testowner", "testrepo", ["triage"]);

      expect(result).toEqual({
        authorized: true,
        permission: "triage",
      });
    });

    it("should check permissions in order and stop at first match", async () => {
      mockGithub.rest.repos.getCollaboratorPermissionLevel.mockResolvedValue({
        data: { permission: "write" },
      });

      const result = await checkRepositoryPermission("testuser", "testowner", "testrepo", ["admin", "write", "read"]);

      expect(result.authorized).toBe(true);
      expect(result.permission).toBe("write");

      // Should log success for write, not check read
      const successLog = mockCore.info.mock.calls.find(call => call[0].includes("✅"));
      expect(successLog[0]).toContain("write");
    });
  });

  describe("checkBotStatus", () => {
    it("should identify bot by [bot] suffix", async () => {
      mockGithub.rest.repos.getCollaboratorPermissionLevel.mockResolvedValue({
        data: { permission: "write" },
      });

      const result = await checkBotStatus("dependabot[bot]", "testowner", "testrepo");

      expect(result).toEqual({
        isBot: true,
        isActive: true,
      });

      expect(mockCore.info).toHaveBeenCalledWith("Checking if bot 'dependabot[bot]' is active on testowner/testrepo");
      expect(mockCore.info).toHaveBeenCalledWith("Bot 'dependabot[bot]' is active with permission level: write");
    });

    it("should return false for non-bot users", async () => {
      const result = await checkBotStatus("regularuser", "testowner", "testrepo");

      expect(result).toEqual({
        isBot: false,
        isActive: false,
      });

      expect(mockGithub.rest.repos.getCollaboratorPermissionLevel).not.toHaveBeenCalled();
    });

    it("should handle 404 error for inactive bot", async () => {
      const apiError = { status: 404, message: "Not Found" };
      mockGithub.rest.repos.getCollaboratorPermissionLevel.mockRejectedValue(apiError);

      const result = await checkBotStatus("renovate[bot]", "testowner", "testrepo");

      expect(result).toEqual({
        isBot: true,
        isActive: false,
      });

      expect(mockCore.warning).toHaveBeenCalledWith("Bot 'renovate[bot]' is not active/installed on testowner/testrepo");
    });

    it("should handle other API errors", async () => {
      const apiError = new Error("API rate limit exceeded");
      mockGithub.rest.repos.getCollaboratorPermissionLevel.mockRejectedValue(apiError);

      const result = await checkBotStatus("github-actions[bot]", "testowner", "testrepo");

      expect(result).toEqual({
        isBot: true,
        isActive: false,
        error: "API rate limit exceeded",
      });

      expect(mockCore.warning).toHaveBeenCalledWith("Failed to check bot status: API rate limit exceeded");
    });

    it("should handle non-Error API failures", async () => {
      mockGithub.rest.repos.getCollaboratorPermissionLevel.mockRejectedValue("String error");

      const result = await checkBotStatus("bot[bot]", "testowner", "testrepo");

      expect(result).toEqual({
        isBot: true,
        isActive: false,
        error: "String error",
      });

      expect(mockCore.warning).toHaveBeenCalledWith("Failed to check bot status: String error");
    });

    it("should handle unexpected errors gracefully", async () => {
      // Simulate an error during bot detection
      const unexpectedError = new Error("Unexpected error");
      mockGithub.rest.repos.getCollaboratorPermissionLevel.mockImplementation(() => {
        throw unexpectedError;
      });

      const result = await checkBotStatus("test[bot]", "testowner", "testrepo");

      expect(result).toEqual({
        isBot: true,
        isActive: false,
        error: "Unexpected error",
      });
    });

    it("should verify bot is installed on repository", async () => {
      mockGithub.rest.repos.getCollaboratorPermissionLevel.mockResolvedValue({
        data: { permission: "admin" },
      });

      const result = await checkBotStatus("dependabot[bot]", "testowner", "testrepo");

      expect(mockGithub.rest.repos.getCollaboratorPermissionLevel).toHaveBeenCalledWith({
        owner: "testowner",
        repo: "testrepo",
        username: "dependabot[bot]",
      });

      expect(result.isBot).toBe(true);
      expect(result.isActive).toBe(true);
    });
  });
});
