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
  let canonicalizeBotIdentifier;
  let isAllowedBot;
  let isConfusedDeputyAttack;
  let readAllowBotAuthoredTriggerComment;
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
    canonicalizeBotIdentifier = module.canonicalizeBotIdentifier;
    isAllowedBot = module.isAllowedBot;
    checkRepositoryPermission = module.checkRepositoryPermission;
    checkBotStatus = module.checkBotStatus;
    isConfusedDeputyAttack = module.isConfusedDeputyAttack;
    readAllowBotAuthoredTriggerComment = module.readAllowBotAuthoredTriggerComment;
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

  describe("canonicalizeBotIdentifier", () => {
    it("should strip [bot] suffix", () => {
      expect(canonicalizeBotIdentifier("dependabot[bot]")).toBe("dependabot");
    });

    it("should return name unchanged when no [bot] suffix", () => {
      expect(canonicalizeBotIdentifier("my-pipeline-app")).toBe("my-pipeline-app");
    });

    it("should handle names with [bot] suffix only once", () => {
      expect(canonicalizeBotIdentifier("github-actions[bot]")).toBe("github-actions");
    });
  });

  describe("isAllowedBot", () => {
    it("should match exact slug to slug", () => {
      expect(isAllowedBot("my-app", ["my-app"])).toBe(true);
    });

    it("should match slug to slug[bot]", () => {
      expect(isAllowedBot("my-app[bot]", ["my-app"])).toBe(true);
    });

    it("should match slug[bot] to slug", () => {
      expect(isAllowedBot("my-app", ["my-app[bot]"])).toBe(true);
    });

    it("should match slug[bot] to slug[bot]", () => {
      expect(isAllowedBot("my-app[bot]", ["my-app[bot]"])).toBe(true);
    });

    it("should return false when actor is not in the list", () => {
      expect(isAllowedBot("other-app", ["my-app"])).toBe(false);
    });

    it("should return false for empty allowed bots list", () => {
      expect(isAllowedBot("my-app", [])).toBe(false);
    });

    it("should match against any entry in the list", () => {
      expect(isAllowedBot("renovate[bot]", ["dependabot[bot]", "renovate", "github-actions[bot]"])).toBe(true);
    });

    it("should not match partial slug names", () => {
      expect(isAllowedBot("my-app-extra[bot]", ["my-app"])).toBe(false);
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

    it("should identify active bot by slug without [bot] suffix", async () => {
      mockGithub.rest.repos.getCollaboratorPermissionLevel.mockResolvedValue({
        data: { permission: "write" },
      });

      const result = await checkBotStatus("my-pipeline-app", "testowner", "testrepo");

      expect(result).toEqual({
        isBot: true,
        isActive: true,
      });

      // API should be called with the [bot]-suffixed form
      expect(mockGithub.rest.repos.getCollaboratorPermissionLevel).toHaveBeenCalledWith({
        owner: "testowner",
        repo: "testrepo",
        username: "my-pipeline-app[bot]",
      });

      expect(mockCore.info).toHaveBeenCalledWith("Checking if bot 'my-pipeline-app' is active on testowner/testrepo");
      expect(mockCore.info).toHaveBeenCalledWith("Bot 'my-pipeline-app' is active with permission level: write");
    });

    it("should return inactive bot when slug without [bot] suffix is not installed", async () => {
      const apiError = { status: 404, message: "Not Found" };
      mockGithub.rest.repos.getCollaboratorPermissionLevel.mockRejectedValue(apiError);

      const result = await checkBotStatus("my-pipeline-app", "testowner", "testrepo");

      expect(result).toEqual({
        isBot: true,
        isActive: false,
      });

      // API should still be called with the [bot]-suffixed form
      expect(mockGithub.rest.repos.getCollaboratorPermissionLevel).toHaveBeenCalledWith({
        owner: "testowner",
        repo: "testrepo",
        username: "my-pipeline-app[bot]",
      });

      expect(mockCore.warning).toHaveBeenCalledWith("Bot 'my-pipeline-app' is not active/installed on testowner/testrepo");
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

    it("should verify bot is installed on repository using [bot] form", async () => {
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

    it("should fall back to slug form when [bot] form is not found as collaborator", async () => {
      const notFoundError = { status: 404, message: "Not Found" };
      mockGithub.rest.repos.getCollaboratorPermissionLevel
        .mockRejectedValueOnce(notFoundError) // [bot] form returns 404
        .mockResolvedValueOnce({ data: { permission: "none" } }); // slug form returns none

      const result = await checkBotStatus("greptile-apps", "testowner", "testrepo");

      expect(result).toEqual({ isBot: true, isActive: true });

      // Verify [bot] form was tried first
      expect(mockGithub.rest.repos.getCollaboratorPermissionLevel).toHaveBeenNthCalledWith(1, {
        owner: "testowner",
        repo: "testrepo",
        username: "greptile-apps[bot]",
      });
      // Verify slug form was tried as fallback
      expect(mockGithub.rest.repos.getCollaboratorPermissionLevel).toHaveBeenNthCalledWith(2, {
        owner: "testowner",
        repo: "testrepo",
        username: "greptile-apps",
      });
      expect(mockCore.info).toHaveBeenCalledWith("Bot 'greptile-apps' is active (via slug form) with permission level: none");
    });

    it("should fall back to slug form when actor has [bot] suffix and [bot] form is not found", async () => {
      const notFoundError = { status: 404, message: "Not Found" };
      mockGithub.rest.repos.getCollaboratorPermissionLevel
        .mockRejectedValueOnce(notFoundError) // [bot] form returns 404
        .mockResolvedValueOnce({ data: { permission: "none" } }); // slug form returns none

      const result = await checkBotStatus("copilot[bot]", "testowner", "testrepo");

      expect(result).toEqual({ isBot: true, isActive: true });

      expect(mockGithub.rest.repos.getCollaboratorPermissionLevel).toHaveBeenNthCalledWith(1, {
        owner: "testowner",
        repo: "testrepo",
        username: "copilot[bot]",
      });
      expect(mockGithub.rest.repos.getCollaboratorPermissionLevel).toHaveBeenNthCalledWith(2, {
        owner: "testowner",
        repo: "testrepo",
        username: "copilot",
      });
      expect(mockCore.info).toHaveBeenCalledWith("Bot 'copilot[bot]' is active (via slug form) with permission level: none");
    });

    it("should return inactive when both [bot] and slug forms return 404", async () => {
      const notFoundError = { status: 404, message: "Not Found" };
      mockGithub.rest.repos.getCollaboratorPermissionLevel.mockRejectedValue(notFoundError);

      const result = await checkBotStatus("unknown-app", "testowner", "testrepo");

      expect(result).toEqual({ isBot: true, isActive: false });
      expect(mockCore.warning).toHaveBeenCalledWith("Bot 'unknown-app' is not active/installed on testowner/testrepo");
    });

    it("should return inactive with error when slug form returns non-404 error", async () => {
      const notFoundError = { status: 404, message: "Not Found" };
      const rateLimit = new Error("API rate limit exceeded");
      mockGithub.rest.repos.getCollaboratorPermissionLevel
        .mockRejectedValueOnce(notFoundError) // [bot] form returns 404
        .mockRejectedValueOnce(rateLimit); // slug form returns rate limit error

      const result = await checkBotStatus("greptile-apps", "testowner", "testrepo");

      expect(result).toEqual({ isBot: true, isActive: false, error: "API rate limit exceeded" });
      expect(mockCore.warning).toHaveBeenCalledWith("Failed to check bot status: API rate limit exceeded");
    });
  });

  describe("isConfusedDeputyAttack", () => {
    describe("pull_request events", () => {
      it("should return false when actor matches PR author on synchronize (genuine dependabot PR)", () => {
        const payload = { action: "synchronize", pull_request: { user: { login: "dependabot[bot]" } } };
        expect(isConfusedDeputyAttack("dependabot[bot]", "pull_request", payload)).toBe(false);
      });

      it("should return true when actor differs from PR author on synchronize (confused deputy via @dependabot recreate)", () => {
        const payload = { action: "synchronize", pull_request: { user: { login: "attacker" } } };
        expect(isConfusedDeputyAttack("dependabot[bot]", "pull_request", payload)).toBe(true);
      });

      it("should return false when actor matches PR author for a human PR on synchronize", () => {
        const payload = { action: "synchronize", pull_request: { user: { login: "octocat" } } };
        expect(isConfusedDeputyAttack("octocat", "pull_request", payload)).toBe(false);
      });

      it("should return false when PR author is absent from payload on synchronize", () => {
        const payload = { action: "synchronize", pull_request: {} };
        expect(isConfusedDeputyAttack("dependabot[bot]", "pull_request", payload)).toBe(false);
      });

      it("should return false when pull_request is absent from payload on synchronize", () => {
        const payload = { action: "synchronize" };
        expect(isConfusedDeputyAttack("dependabot[bot]", "pull_request", payload)).toBe(false);
      });

      it("should return false for pull_request:labeled even if actor differs from PR author", () => {
        // A team member can label a PR authored by someone else — NOT a confused deputy attack
        const payload = { action: "labeled", pull_request: { user: { login: "pr-author" } } };
        expect(isConfusedDeputyAttack("pelikhan", "pull_request", payload)).toBe(false);
      });

      it("should return false for pull_request:opened even if actor differs from PR author", () => {
        // opened is not the synchronize attack vector — skip the check
        const payload = { action: "opened", pull_request: { user: { login: "attacker" } } };
        expect(isConfusedDeputyAttack("dependabot[bot]", "pull_request", payload)).toBe(false);
      });

      it("should return false for pull_request:unlabeled even if actor differs from PR author", () => {
        const payload = { action: "unlabeled", pull_request: { user: { login: "pr-author" } } };
        expect(isConfusedDeputyAttack("pelikhan", "pull_request", payload)).toBe(false);
      });

      it("should return false for pull_request:review_requested even if actor differs from PR author", () => {
        const payload = { action: "review_requested", pull_request: { user: { login: "pr-author" } } };
        expect(isConfusedDeputyAttack("pelikhan", "pull_request", payload)).toBe(false);
      });

      it("should return false for pull_request_review when actor matches review author (genuine review)", () => {
        const payload = {
          pull_request: { user: { login: "pr-author" } },
          review: { user: { login: "dependabot[bot]" } },
        };
        expect(isConfusedDeputyAttack("dependabot[bot]", "pull_request_review", payload)).toBe(false);
      });

      it("should return true for pull_request_review when actor differs from review author", () => {
        const payload = {
          pull_request: { user: { login: "pr-author" } },
          review: { user: { login: "attacker" } },
        };
        expect(isConfusedDeputyAttack("dependabot[bot]", "pull_request_review", payload)).toBe(true);
      });

      it("should return false for pull_request_review when actor is reviewer even if PR author differs", () => {
        // Normal case: reviewer is different from PR author — must NOT be false positive
        const payload = {
          pull_request: { user: { login: "pr-author" } },
          review: { user: { login: "dependabot[bot]" } },
        };
        expect(isConfusedDeputyAttack("dependabot[bot]", "pull_request_review", payload)).toBe(false);
      });

      it("should return false for pull_request_review_comment when actor matches comment author (genuine comment)", () => {
        const payload = {
          pull_request: { user: { login: "pr-author" } },
          comment: { user: { login: "dependabot[bot]" } },
        };
        expect(isConfusedDeputyAttack("dependabot[bot]", "pull_request_review_comment", payload)).toBe(false);
      });

      it("should return true for pull_request_review_comment when actor differs from comment author", () => {
        const payload = {
          pull_request: { user: { login: "pr-author" } },
          comment: { user: { login: "attacker" } },
        };
        expect(isConfusedDeputyAttack("dependabot[bot]", "pull_request_review_comment", payload)).toBe(true);
      });

      it("should return false for pull_request_review_comment when actor is commenter even if PR author differs", () => {
        // Normal case: commenter is different from PR author — must NOT be false positive
        const payload = {
          pull_request: { user: { login: "pr-author" } },
          comment: { user: { login: "dependabot[bot]" } },
        };
        expect(isConfusedDeputyAttack("dependabot[bot]", "pull_request_review_comment", payload)).toBe(false);
      });
    });

    describe("issue_comment events", () => {
      it("should return false when actor matches comment author (genuine bot comment)", () => {
        const payload = { comment: { user: { login: "dependabot[bot]" } } };
        expect(isConfusedDeputyAttack("dependabot[bot]", "issue_comment", payload)).toBe(false);
      });

      it("should return true when actor differs from comment author on created action", () => {
        // The dependabot @dependabot show attack goes via issue_comment:created
        const payload = { action: "created", comment: { user: { login: "dependabot[bot]" } } };
        expect(isConfusedDeputyAttack("attacker", "issue_comment", payload)).toBe(true);
      });

      it("should return true when actor differs from human comment author (no action field)", () => {
        const payload = { comment: { user: { login: "attacker" } } };
        expect(isConfusedDeputyAttack("dependabot[bot]", "issue_comment", payload)).toBe(true);
      });

      it("should return false when comment author is absent from payload", () => {
        const payload = { comment: {} };
        expect(isConfusedDeputyAttack("dependabot[bot]", "issue_comment", payload)).toBe(false);
      });

      it("should return false when comment is absent from payload", () => {
        const payload = {};
        expect(isConfusedDeputyAttack("dependabot[bot]", "issue_comment", payload)).toBe(false);
      });

      describe("bot-posted-menu / user-checks-box pattern (issue_comment:edited by bot author)", () => {
        it("should return false for issue_comment:edited with [bot]-authored comment (auto-detection from payload)", () => {
          // The legitimate pattern: workflow posts checkbox-menu comment (github-actions[bot]),
          // human maintainer edits it to tick a box → actor != comment.user.login, action=edited.
          // Derived directly from the native webhook payload — no aw_context flag needed.
          const payload = { action: "edited", comment: { user: { login: "github-actions[bot]" } } };
          expect(isConfusedDeputyAttack("theletterf", "issue_comment", payload)).toBe(false);
        });

        it("should return false for issue_comment:edited with any [bot]-suffixed comment author", () => {
          const payload = { action: "edited", comment: { user: { login: "custom-bot[bot]" } } };
          expect(isConfusedDeputyAttack("maintainer", "issue_comment", payload)).toBe(false);
        });

        it("should return true for issue_comment:created with [bot]-authored comment (Dependabot attack vector)", () => {
          // The Dependabot attack always fires via created, so this must still be caught.
          const payload = { action: "created", comment: { user: { login: "dependabot[bot]" } } };
          expect(isConfusedDeputyAttack("attacker", "issue_comment", payload)).toBe(true);
        });

        it("should return true for issue_comment:edited with human comment author (not a bot-menu)", () => {
          // A human edited a human's comment — mismatch is still suspicious.
          const payload = { action: "edited", comment: { user: { login: "human-author" } } };
          expect(isConfusedDeputyAttack("different-actor", "issue_comment", payload)).toBe(true);
        });

        it("should return false for issue_comment:edited when actor matches bot comment author", () => {
          // No mismatch — not a confused deputy
          const payload = { action: "edited", comment: { user: { login: "github-actions[bot]" } } };
          expect(isConfusedDeputyAttack("github-actions[bot]", "issue_comment", payload)).toBe(false);
        });

        describe("GH_AW_ALLOW_BOT_AUTHORED_TRIGGER_COMMENT env var (on.allow-bot-authored-trigger-comment frontmatter opt-in)", () => {
          beforeEach(() => {
            process.env.GH_AW_ALLOW_BOT_AUTHORED_TRIGGER_COMMENT = "true";
          });
          afterEach(() => {
            delete process.env.GH_AW_ALLOW_BOT_AUTHORED_TRIGGER_COMMENT;
          });

          it("should return false for issue_comment:edited with non-[bot]-author when env var is set (custom bot naming)", () => {
            // Frontmatter opt-in covers bots that don't follow [bot] naming convention
            const payload = { action: "edited", comment: { user: { login: "my-custom-automation" } } };
            expect(isConfusedDeputyAttack("theletterf", "issue_comment", payload)).toBe(false);
          });

          it("should still return true for issue_comment:created even when env var is set", () => {
            // The Dependabot attack vector (created) must remain guarded regardless of frontmatter
            const payload = { action: "created", comment: { user: { login: "dependabot[bot]" } } };
            expect(isConfusedDeputyAttack("attacker", "issue_comment", payload)).toBe(true);
          });

          it("should still return true for issue_comment:edited when env var is set but action is absent (no action = no bypass)", () => {
            // No action field → payload.action is undefined, not "edited"
            const payload = { comment: { user: { login: "some-bot" } } };
            expect(isConfusedDeputyAttack("attacker", "issue_comment", payload)).toBe(true);
          });
        });
      });
    });

    describe("other event types", () => {
      it("should return false for push events (no PR/comment context)", () => {
        const payload = { sender: { login: "dependabot[bot]" } };
        expect(isConfusedDeputyAttack("dependabot[bot]", "push", payload)).toBe(false);
      });

      it("should return false for issues events", () => {
        const payload = { issue: { user: { login: "attacker" } } };
        expect(isConfusedDeputyAttack("dependabot[bot]", "issues", payload)).toBe(false);
      });

      it("should return false for schedule events", () => {
        expect(isConfusedDeputyAttack("github-actions[bot]", "schedule", {})).toBe(false);
      });

      it("should return false for workflow_call events (no PR/comment in payload)", () => {
        // In workflow_call, context.payload = { inputs: { aw_context: "..." } }
        // aw_context carries event_type/item_number but NOT pull_request.user.login
        const payload = { inputs: { aw_context: '{"event_type":"pull_request","item_number":"42","actor":"attacker"}' } };
        expect(isConfusedDeputyAttack("dependabot[bot]", "workflow_call", payload)).toBe(false);
      });

      it("should return false for workflow_call even if payload contains unrelated pull_request data", () => {
        // Even if someone injects pull_request data in the payload, the eventName check
        // guards against false positives: workflow_call is never in prEvents
        const payload = { pull_request: { user: { login: "attacker" } }, inputs: {} };
        expect(isConfusedDeputyAttack("dependabot[bot]", "workflow_call", payload)).toBe(false);
      });
    });

    describe("edge cases", () => {
      it("should return false when payload is null", () => {
        expect(isConfusedDeputyAttack("dependabot[bot]", "pull_request", null)).toBe(false);
      });

      it("should return false when payload is undefined", () => {
        expect(isConfusedDeputyAttack("dependabot[bot]", "pull_request", undefined)).toBe(false);
      });
    });
  });

  describe("readAllowBotAuthoredTriggerComment", () => {
    it("should return false when payload is undefined", () => {
      expect(readAllowBotAuthoredTriggerComment(undefined)).toBe(false);
    });

    it("should return false when payload has no aw_context inputs", () => {
      expect(readAllowBotAuthoredTriggerComment({ inputs: {} })).toBe(false);
    });

    it("should return false when aw_context is empty string", () => {
      expect(readAllowBotAuthoredTriggerComment({ inputs: { aw_context: "" } })).toBe(false);
    });

    it("should return false when aw_context does not contain the flag", () => {
      const payload = { inputs: { aw_context: '{"event_type":"issue_comment","actor":"theletterf"}' } };
      expect(readAllowBotAuthoredTriggerComment(payload)).toBe(false);
    });

    it("should return true when inputs.aw_context has allow_bot_authored_trigger_comment: true", () => {
      const awContext = JSON.stringify({ allow_bot_authored_trigger_comment: true, event_type: "issue_comment" });
      expect(readAllowBotAuthoredTriggerComment({ inputs: { aw_context: awContext } })).toBe(true);
    });

    it("should return false when allow_bot_authored_trigger_comment is a string 'true' (not boolean)", () => {
      const awContext = JSON.stringify({ allow_bot_authored_trigger_comment: "true" });
      expect(readAllowBotAuthoredTriggerComment({ inputs: { aw_context: awContext } })).toBe(false);
    });

    it("should return false when allow_bot_authored_trigger_comment is false", () => {
      const awContext = JSON.stringify({ allow_bot_authored_trigger_comment: false });
      expect(readAllowBotAuthoredTriggerComment({ inputs: { aw_context: awContext } })).toBe(false);
    });

    it("should return true when client_payload.aw_context has the flag (repository_dispatch)", () => {
      const awContext = JSON.stringify({ allow_bot_authored_trigger_comment: true });
      expect(readAllowBotAuthoredTriggerComment({ client_payload: { aw_context: awContext } })).toBe(true);
    });

    it("should return false when aw_context is invalid JSON", () => {
      expect(readAllowBotAuthoredTriggerComment({ inputs: { aw_context: "not-json{" } })).toBe(false);
    });

    it("should return false when aw_context is a JSON array (not an object)", () => {
      expect(readAllowBotAuthoredTriggerComment({ inputs: { aw_context: "[1,2,3]" } })).toBe(false);
    });

    it("should return true when aw_context is passed as a plain object (repository_dispatch object form)", () => {
      expect(readAllowBotAuthoredTriggerComment({ client_payload: { aw_context: { allow_bot_authored_trigger_comment: true } } })).toBe(true);
    });
  });
});
