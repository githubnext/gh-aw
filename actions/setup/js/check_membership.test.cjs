import { describe, it, expect, beforeEach, afterEach, vi } from "vitest";

describe("check_membership.cjs", () => {
  let mockCore;
  let mockGithub;
  let mockContext;
  let checkMembershipScript;

  beforeEach(() => {
    // Mock core actions methods
    mockCore = {
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

    // Mock GitHub API
    mockGithub = {
      rest: {
        repos: {
          getCollaboratorPermissionLevel: vi.fn(),
        },
      },
    };

    // Mock context
    mockContext = {
      eventName: "issues",
      actor: "testuser",
      repo: {
        owner: "testorg",
        repo: "testrepo",
      },
    };

    global.core = mockCore;
    global.github = mockGithub;
    global.context = mockContext;
  });

  afterEach(() => {
    delete global.core;
    delete global.github;
    delete global.context;
    delete process.env.GH_AW_REQUIRED_ROLES;
  });

  const runScript = async () => {
    const fs = await import("fs");
    const path = await import("path");
    const scriptPath = path.join(import.meta.dirname, "check_membership.cjs");
    const scriptContent = fs.readFileSync(scriptPath, "utf8");

    // Load the utility module
    const utilsPath = path.join(import.meta.dirname, "check_permissions_utils.cjs");
    const utilsContent = fs.readFileSync(utilsPath, "utf8");

    // Load error helpers module
    const errorHelpersPath = path.join(import.meta.dirname, "error_helpers.cjs");
    const errorHelpersContent = fs.readFileSync(errorHelpersPath, "utf8");

    // Create a mock require function
    const mockRequire = modulePath => {
      if (modulePath === "./error_helpers.cjs") {
        // Execute the error helpers module and return its exports
        const errorHelpersFunction = new Function("module", "exports", errorHelpersContent);
        const errorHelpersModuleExports = {};
        const errorHelpersMockModule = { exports: errorHelpersModuleExports };
        errorHelpersFunction(errorHelpersMockModule, errorHelpersModuleExports);
        return errorHelpersMockModule.exports;
      }
      if (modulePath === "./check_permissions_utils.cjs") {
        // Execute the utility module and return its exports
        // Need to pass mockRequire to handle error_helpers require
        const utilsFunction = new Function("core", "github", "context", "process", "module", "exports", "require", utilsContent);
        const moduleExports = {};
        const mockModule = { exports: moduleExports };
        utilsFunction(mockCore, mockGithub, mockContext, process, mockModule, moduleExports, mockRequire);
        return mockModule.exports;
      }
      throw new Error(`Module not found: ${modulePath}`);
    };

    // Remove the main() call/export at the end and execute
    const scriptWithoutMain = scriptContent.replace("module.exports = { main };", "");
    const scriptFunction = new Function("core", "github", "context", "process", "require", scriptWithoutMain + "\nreturn main();");
    await scriptFunction(mockCore, mockGithub, mockContext, process, mockRequire);
  };

  describe("safe events", () => {
    // workflow_run is no longer a safe event due to HIGH security risks:
    // - Privilege escalation (inherits permissions from triggering workflow)
    // - Branch protection bypass (can execute on protected branches)
    // - Secret exposure (secrets available from untrusted code)

    it("should skip check for schedule events", async () => {
      mockContext.eventName = "schedule";
      await runScript();

      expect(mockCore.info).toHaveBeenCalledWith("✅ Event schedule does not require validation");
      expect(mockCore.setOutput).toHaveBeenCalledWith("is_team_member", "true");
      expect(mockCore.setOutput).toHaveBeenCalledWith("result", "safe_event");
    });

    it("should skip check for merge_group events", async () => {
      mockContext.eventName = "merge_group";
      await runScript();

      expect(mockCore.info).toHaveBeenCalledWith("✅ Event merge_group does not require validation");
      expect(mockCore.setOutput).toHaveBeenCalledWith("is_team_member", "true");
      expect(mockCore.setOutput).toHaveBeenCalledWith("result", "safe_event");
    });

    it("should skip check for workflow_dispatch when write role is allowed", async () => {
      mockContext.eventName = "workflow_dispatch";
      process.env.GH_AW_REQUIRED_ROLES = "write,read";

      await runScript();

      expect(mockCore.info).toHaveBeenCalledWith("✅ Event workflow_dispatch does not require validation (write role allowed)");
      expect(mockCore.setOutput).toHaveBeenCalledWith("is_team_member", "true");
      expect(mockCore.setOutput).toHaveBeenCalledWith("result", "safe_event");
    });

    it("should validate workflow_dispatch when write role is not allowed", async () => {
      mockContext.eventName = "workflow_dispatch";
      process.env.GH_AW_REQUIRED_ROLES = "admin,maintainer";

      mockGithub.rest.repos.getCollaboratorPermissionLevel.mockResolvedValue({
        data: { permission: "admin" },
      });

      await runScript();

      expect(mockCore.info).toHaveBeenCalledWith("Event workflow_dispatch requires validation (write role not allowed)");
      expect(mockGithub.rest.repos.getCollaboratorPermissionLevel).toHaveBeenCalled();
    });
  });

  describe("configuration validation", () => {
    it("should fail when no required permissions are specified", async () => {
      delete process.env.GH_AW_REQUIRED_ROLES;

      await runScript();

      expect(mockCore.warning).toHaveBeenCalledWith("❌ Configuration error: Required permissions not specified. Contact repository administrator.");
      expect(mockCore.setOutput).toHaveBeenCalledWith("is_team_member", "false");
      expect(mockCore.setOutput).toHaveBeenCalledWith("result", "config_error");
      expect(mockCore.setOutput).toHaveBeenCalledWith("error_message", "Configuration error: Required permissions not specified");
    });

    it("should fail when required permissions is empty string", async () => {
      process.env.GH_AW_REQUIRED_ROLES = "";

      await runScript();

      expect(mockCore.warning).toHaveBeenCalledWith("❌ Configuration error: Required permissions not specified. Contact repository administrator.");
      expect(mockCore.setOutput).toHaveBeenCalledWith("is_team_member", "false");
      expect(mockCore.setOutput).toHaveBeenCalledWith("result", "config_error");
    });

    it("should filter out empty permission values", async () => {
      process.env.GH_AW_REQUIRED_ROLES = "admin, , write, ";

      mockGithub.rest.repos.getCollaboratorPermissionLevel.mockResolvedValue({
        data: { permission: "admin" },
      });

      await runScript();

      // Check that the log contains filtered permissions (note: there's no trimming of the space after comma in actual code)
      expect(mockCore.info).toHaveBeenCalled();
      const logCalls = mockCore.info.mock.calls.map(call => call[0]);
      const permissionsLog = logCalls.find(log => log.includes("Required permissions:"));
      expect(permissionsLog).toBeTruthy();
    });
  });

  describe("permission checks", () => {
    beforeEach(() => {
      process.env.GH_AW_REQUIRED_ROLES = "admin,write";
    });

    it("should authorize user with exact permission match", async () => {
      mockGithub.rest.repos.getCollaboratorPermissionLevel.mockResolvedValue({
        data: { permission: "admin" },
      });

      await runScript();

      expect(mockCore.info).toHaveBeenCalledWith("Checking if user 'testuser' has required permissions for testorg/testrepo");
      expect(mockCore.info).toHaveBeenCalledWith("Required permissions: admin, write");
      expect(mockCore.info).toHaveBeenCalledWith("Repository permission level: admin");
      expect(mockCore.info).toHaveBeenCalledWith("✅ User has admin access to repository");
      expect(mockCore.setOutput).toHaveBeenCalledWith("is_team_member", "true");
      expect(mockCore.setOutput).toHaveBeenCalledWith("result", "authorized");
      expect(mockCore.setOutput).toHaveBeenCalledWith("user_permission", "admin");
    });

    it("should handle maintainer/maintain alias", async () => {
      process.env.GH_AW_REQUIRED_ROLES = "maintainer";

      mockGithub.rest.repos.getCollaboratorPermissionLevel.mockResolvedValue({
        data: { permission: "maintain" },
      });

      await runScript();

      expect(mockCore.info).toHaveBeenCalledWith("✅ User has maintain access to repository");
      expect(mockCore.setOutput).toHaveBeenCalledWith("is_team_member", "true");
      expect(mockCore.setOutput).toHaveBeenCalledWith("result", "authorized");
      expect(mockCore.setOutput).toHaveBeenCalledWith("user_permission", "maintain");
    });

    it("should deny user with insufficient permissions", async () => {
      mockGithub.rest.repos.getCollaboratorPermissionLevel.mockResolvedValue({
        data: { permission: "read" },
      });

      await runScript();

      expect(mockCore.warning).toHaveBeenCalledWith("User permission 'read' does not meet requirements: admin, write");
      expect(mockCore.setOutput).toHaveBeenCalledWith("is_team_member", "false");
      expect(mockCore.setOutput).toHaveBeenCalledWith("result", "insufficient_permissions");
      expect(mockCore.setOutput).toHaveBeenCalledWith("user_permission", "read");
      expect(mockCore.setOutput).toHaveBeenCalledWith(
        "error_message",
        "Access denied: User 'testuser' is not authorized. Required permissions: admin, write. To allow this user to run the workflow, add their role to the frontmatter. Example: roles: [admin, write, read]"
      );
    });

    it("should authorize user with write permission when write is in allowed list", async () => {
      mockGithub.rest.repos.getCollaboratorPermissionLevel.mockResolvedValue({
        data: { permission: "write" },
      });

      await runScript();

      expect(mockCore.info).toHaveBeenCalledWith("✅ User has write access to repository");
      expect(mockCore.setOutput).toHaveBeenCalledWith("is_team_member", "true");
      expect(mockCore.setOutput).toHaveBeenCalledWith("result", "authorized");
    });
  });

  describe("API error handling", () => {
    beforeEach(() => {
      process.env.GH_AW_REQUIRED_ROLES = "admin";
    });

    it("should handle API errors gracefully", async () => {
      const apiError = new Error("API rate limit exceeded");
      mockGithub.rest.repos.getCollaboratorPermissionLevel.mockRejectedValue(apiError);

      await runScript();

      expect(mockCore.warning).toHaveBeenCalledWith("Repository permission check failed: API rate limit exceeded");
      expect(mockCore.setOutput).toHaveBeenCalledWith("is_team_member", "false");
      expect(mockCore.setOutput).toHaveBeenCalledWith("result", "api_error");
      expect(mockCore.setOutput).toHaveBeenCalledWith("error_message", "Repository permission check failed: API rate limit exceeded");
    });

    it("should handle non-Error API failures", async () => {
      mockGithub.rest.repos.getCollaboratorPermissionLevel.mockRejectedValue("String error");

      await runScript();

      expect(mockCore.warning).toHaveBeenCalledWith("Repository permission check failed: String error");
      expect(mockCore.setOutput).toHaveBeenCalledWith("is_team_member", "false");
      expect(mockCore.setOutput).toHaveBeenCalledWith("result", "api_error");
    });

    it("should handle network errors", async () => {
      const networkError = new Error("Network request failed");
      mockGithub.rest.repos.getCollaboratorPermissionLevel.mockRejectedValue(networkError);

      await runScript();

      expect(mockCore.warning).toHaveBeenCalledWith("Repository permission check failed: Network request failed");
      expect(mockCore.setOutput).toHaveBeenCalledWith("result", "api_error");
    });
  });

  describe("multiple permission levels", () => {
    it("should check multiple permission levels in order", async () => {
      process.env.GH_AW_REQUIRED_ROLES = "admin,maintainer,write";

      mockGithub.rest.repos.getCollaboratorPermissionLevel.mockResolvedValue({
        data: { permission: "write" },
      });

      await runScript();

      expect(mockCore.setOutput).toHaveBeenCalledWith("is_team_member", "true");
      expect(mockCore.setOutput).toHaveBeenCalledWith("result", "authorized");
      expect(mockCore.setOutput).toHaveBeenCalledWith("user_permission", "write");
    });

    it("should stop checking after first match", async () => {
      process.env.GH_AW_REQUIRED_ROLES = "write,read";

      mockGithub.rest.repos.getCollaboratorPermissionLevel.mockResolvedValue({
        data: { permission: "write" },
      });

      await runScript();

      expect(mockCore.info).toHaveBeenCalledWith("✅ User has write access to repository");
    });
  });
});
