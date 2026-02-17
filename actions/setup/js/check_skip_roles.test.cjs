import { describe, it, expect, beforeEach, afterEach, vi } from "vitest";

describe("check_skip_roles.cjs", () => {
  let mockCore;
  let mockGithub;
  let mockContext;

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
    delete process.env.GH_AW_SKIP_ROLES;
  });

  const runScript = async () => {
    const fs = await import("fs");
    const path = await import("path");
    const scriptPath = path.join(import.meta.dirname, "check_skip_roles.cjs");
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

  describe("no skip-roles configured", () => {
    it("should proceed when GH_AW_SKIP_ROLES is not set", async () => {
      delete process.env.GH_AW_SKIP_ROLES;

      await runScript();

      expect(mockCore.setOutput).toHaveBeenCalledWith("skip_roles_ok", "true");
      expect(mockCore.setOutput).toHaveBeenCalledWith("result", "no_skip_roles");
    });

    it("should proceed when GH_AW_SKIP_ROLES is empty string", async () => {
      process.env.GH_AW_SKIP_ROLES = "";

      await runScript();

      expect(mockCore.setOutput).toHaveBeenCalledWith("skip_roles_ok", "true");
      expect(mockCore.setOutput).toHaveBeenCalledWith("result", "no_skip_roles");
    });

    it("should proceed when GH_AW_SKIP_ROLES is whitespace only", async () => {
      process.env.GH_AW_SKIP_ROLES = "   ";

      await runScript();

      expect(mockCore.setOutput).toHaveBeenCalledWith("skip_roles_ok", "true");
      expect(mockCore.setOutput).toHaveBeenCalledWith("result", "no_skip_roles");
    });
  });

  describe("user has skip-role", () => {
    it("should skip workflow when user has admin role", async () => {
      process.env.GH_AW_SKIP_ROLES = "admin,maintainer,write";
      mockGithub.rest.repos.getCollaboratorPermissionLevel.mockResolvedValue({
        data: { permission: "admin" },
      });

      await runScript();

      expect(mockCore.setOutput).toHaveBeenCalledWith("skip_roles_ok", "false");
      expect(mockCore.setOutput).toHaveBeenCalledWith("result", "skipped");
      expect(mockCore.setOutput).toHaveBeenCalledWith("user_permission", "admin");
      expect(mockCore.setOutput).toHaveBeenCalledWith("error_message", "Workflow skipped: User 'testuser' has role 'admin' which is in skip-roles: [admin, maintainer, write]");
    });

    it("should skip workflow when user has write role", async () => {
      process.env.GH_AW_SKIP_ROLES = "admin,write";
      mockGithub.rest.repos.getCollaboratorPermissionLevel.mockResolvedValue({
        data: { permission: "write" },
      });

      await runScript();

      expect(mockCore.setOutput).toHaveBeenCalledWith("skip_roles_ok", "false");
      expect(mockCore.setOutput).toHaveBeenCalledWith("result", "skipped");
      expect(mockCore.setOutput).toHaveBeenCalledWith("user_permission", "write");
    });

    it("should skip workflow when user has maintain role", async () => {
      process.env.GH_AW_SKIP_ROLES = "admin,maintain";
      mockGithub.rest.repos.getCollaboratorPermissionLevel.mockResolvedValue({
        data: { permission: "maintain" },
      });

      await runScript();

      expect(mockCore.setOutput).toHaveBeenCalledWith("skip_roles_ok", "false");
      expect(mockCore.setOutput).toHaveBeenCalledWith("result", "skipped");
      expect(mockCore.setOutput).toHaveBeenCalledWith("user_permission", "maintain");
    });
  });

  describe("user does not have skip-role", () => {
    it("should proceed when user has read role and skip-roles is admin,write", async () => {
      process.env.GH_AW_SKIP_ROLES = "admin,write";
      mockGithub.rest.repos.getCollaboratorPermissionLevel.mockResolvedValue({
        data: { permission: "read" },
      });

      await runScript();

      expect(mockCore.setOutput).toHaveBeenCalledWith("skip_roles_ok", "true");
      expect(mockCore.setOutput).toHaveBeenCalledWith("result", "not_skipped");
      expect(mockCore.setOutput).toHaveBeenCalledWith("user_permission", "read");
    });

    it("should proceed when user has triage role and skip-roles is admin,write", async () => {
      process.env.GH_AW_SKIP_ROLES = "admin,write";
      mockGithub.rest.repos.getCollaboratorPermissionLevel.mockResolvedValue({
        data: { permission: "triage" },
      });

      await runScript();

      expect(mockCore.setOutput).toHaveBeenCalledWith("skip_roles_ok", "true");
      expect(mockCore.setOutput).toHaveBeenCalledWith("result", "not_skipped");
      expect(mockCore.setOutput).toHaveBeenCalledWith("user_permission", "triage");
    });

    it("should proceed when user has none role and skip-roles is admin,write", async () => {
      process.env.GH_AW_SKIP_ROLES = "admin,write";
      mockGithub.rest.repos.getCollaboratorPermissionLevel.mockResolvedValue({
        data: { permission: "none" },
      });

      await runScript();

      expect(mockCore.setOutput).toHaveBeenCalledWith("skip_roles_ok", "true");
      expect(mockCore.setOutput).toHaveBeenCalledWith("result", "not_skipped");
      expect(mockCore.setOutput).toHaveBeenCalledWith("user_permission", "none");
    });
  });

  describe("API error handling", () => {
    it("should proceed when API call fails (fail-safe)", async () => {
      process.env.GH_AW_SKIP_ROLES = "admin,write";
      mockGithub.rest.repos.getCollaboratorPermissionLevel.mockRejectedValue(new Error("API error"));

      await runScript();

      expect(mockCore.setOutput).toHaveBeenCalledWith("skip_roles_ok", "true");
      expect(mockCore.setOutput).toHaveBeenCalledWith("result", "api_error");
      expect(mockCore.warning).toHaveBeenCalled();
    });

    it("should proceed when API returns 404", async () => {
      process.env.GH_AW_SKIP_ROLES = "admin,write";
      const error = new Error("Not Found");
      // @ts-expect-error - Adding status property
      error.status = 404;
      mockGithub.rest.repos.getCollaboratorPermissionLevel.mockRejectedValue(error);

      await runScript();

      expect(mockCore.setOutput).toHaveBeenCalledWith("skip_roles_ok", "true");
      expect(mockCore.setOutput).toHaveBeenCalledWith("result", "api_error");
    });
  });

  describe("skip-roles parsing", () => {
    it("should handle comma-separated skip-roles with spaces", async () => {
      process.env.GH_AW_SKIP_ROLES = " admin , write , maintain ";
      mockGithub.rest.repos.getCollaboratorPermissionLevel.mockResolvedValue({
        data: { permission: "admin" },
      });

      await runScript();

      expect(mockCore.setOutput).toHaveBeenCalledWith("skip_roles_ok", "false");
      expect(mockCore.setOutput).toHaveBeenCalledWith("result", "skipped");
    });

    it("should handle single skip-role", async () => {
      process.env.GH_AW_SKIP_ROLES = "admin";
      mockGithub.rest.repos.getCollaboratorPermissionLevel.mockResolvedValue({
        data: { permission: "admin" },
      });

      await runScript();

      expect(mockCore.setOutput).toHaveBeenCalledWith("skip_roles_ok", "false");
      expect(mockCore.setOutput).toHaveBeenCalledWith("result", "skipped");
    });
  });
});
