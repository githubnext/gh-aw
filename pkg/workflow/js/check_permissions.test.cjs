import { describe, it, expect, beforeEach, afterEach, vi } from "vitest";
import fs from "fs";
import path from "path";

// Mock the global objects that GitHub Actions provides
const mockCore = {
  setOutput: vi.fn(),
  warning: vi.fn(),
  error: vi.fn(),
  setFailed: vi.fn(),
  setCancelled: vi.fn(),
  setError: vi.fn(),
};

const mockGithub = {
  rest: {
    repos: {
      getCollaboratorPermissionLevel: vi.fn(),
    },
  },
};

const mockContext = {
  eventName: "issues",
  actor: "testuser",
  repo: {
    owner: "testowner",
    repo: "testrepo",
  },
};

// Set up global variables
global.core = mockCore;
global.github = mockGithub;
global.context = mockContext;

describe("check_permissions.cjs", () => {
  let checkPermissionsScript;
  let originalEnv;

  beforeEach(() => {
    // Reset all mocks
    vi.clearAllMocks();

    // Store original environment
    originalEnv = process.env.GITHUB_AW_REQUIRED_ROLES;

    // Reset context to default state
    global.context.eventName = "issues";
    global.context.actor = "testuser";
    global.context.repo = {
      owner: "testowner",
      repo: "testrepo",
    };

    // Read the script content
    const scriptPath = path.join(
      process.cwd(),
      "pkg/workflow/js/check_permissions.cjs"
    );
    checkPermissionsScript = fs.readFileSync(scriptPath, "utf8");
  });

  afterEach(() => {
    // Restore original environment
    if (originalEnv !== undefined) {
      process.env.GITHUB_AW_REQUIRED_ROLES = originalEnv;
    } else {
      delete process.env.GITHUB_AW_REQUIRED_ROLES;
    }
  });

  it("should fail job when no permissions specified", async () => {
    delete process.env.GITHUB_AW_REQUIRED_ROLES;

    // Mock process.exit to prevent the test from actually exiting
    const processExitSpy = vi
      .spyOn(process, "exit")
      .mockImplementation(code => {
        // Throw an error to stop script execution, simulating process.exit behavior
        throw new Error(`Process exit called with code ${code}`);
      });

    try {
      // Execute the script - this should throw due to process.exit
      await eval(`(async () => { ${checkPermissionsScript} })()`);
    } catch (error) {
      // Expected to throw due to process.exit mock
      expect(error.message).toBe("Process exit called with code 1");
    }

    expect(mockCore.error).toHaveBeenCalledWith(
      "❌ Configuration error: Required permissions not specified. Contact repository administrator."
    );
    expect(processExitSpy).toHaveBeenCalledWith(1);
    expect(
      mockGithub.rest.repos.getCollaboratorPermissionLevel
    ).not.toHaveBeenCalled();

    processExitSpy.mockRestore();
  });

  it("should fail job when permissions are empty", async () => {
    process.env.GITHUB_AW_REQUIRED_ROLES = "";

    // Mock process.exit to prevent the test from actually exiting
    const processExitSpy = vi
      .spyOn(process, "exit")
      .mockImplementation(code => {
        // Throw an error to stop script execution, simulating process.exit behavior
        throw new Error(`Process exit called with code ${code}`);
      });

    try {
      // Execute the script - this should throw due to process.exit
      await eval(`(async () => { ${checkPermissionsScript} })()`);
    } catch (error) {
      // Expected to throw due to process.exit mock
      expect(error.message).toBe("Process exit called with code 1");
    }

    expect(mockCore.error).toHaveBeenCalledWith(
      "❌ Configuration error: Required permissions not specified. Contact repository administrator."
    );
    expect(processExitSpy).toHaveBeenCalledWith(1);
    expect(
      mockGithub.rest.repos.getCollaboratorPermissionLevel
    ).not.toHaveBeenCalled();

    processExitSpy.mockRestore();
  });

  it("should skip validation for safe events", async () => {
    process.env.GITHUB_AW_REQUIRED_ROLES = "admin";
    global.context.eventName = "workflow_dispatch";

    const consoleSpy = vi.spyOn(console, "log").mockImplementation(() => {});

    // Execute the script
    await eval(`(async () => { ${checkPermissionsScript} })()`);

    expect(consoleSpy).toHaveBeenCalledWith(
      "✅ Event workflow_dispatch does not require validation"
    );
    expect(
      mockGithub.rest.repos.getCollaboratorPermissionLevel
    ).not.toHaveBeenCalled();
    expect(mockCore.error).not.toHaveBeenCalled();
    expect(mockCore.warning).not.toHaveBeenCalled();

    consoleSpy.mockRestore();
  });
  it("should pass validation for admin permission", async () => {
    process.env.GITHUB_AW_REQUIRED_ROLES = "admin,maintainer,write";

    mockGithub.rest.repos.getCollaboratorPermissionLevel.mockResolvedValue({
      data: { permission: "admin" },
    });

    const consoleSpy = vi.spyOn(console, "log").mockImplementation(() => {});

    // Execute the script
    await eval(`(async () => { ${checkPermissionsScript} })()`);

    expect(
      mockGithub.rest.repos.getCollaboratorPermissionLevel
    ).toHaveBeenCalledWith({
      owner: "testowner",
      repo: "testrepo",
      username: "testuser",
    });

    expect(consoleSpy).toHaveBeenCalledWith(
      "Checking if user 'testuser' has required permissions for testowner/testrepo"
    );
    expect(consoleSpy).toHaveBeenCalledWith(
      "Required permissions: admin, maintainer, write"
    );
    expect(consoleSpy).toHaveBeenCalledWith(
      "Repository permission level: admin"
    );
    expect(consoleSpy).toHaveBeenCalledWith(
      "✅ User has admin access to repository"
    );

    // Should not call any error or warning methods
    expect(mockCore.error).not.toHaveBeenCalled();
    expect(mockCore.warning).not.toHaveBeenCalled();

    consoleSpy.mockRestore();
  });

  it("should pass validation for maintain permission when maintainer is required", async () => {
    process.env.GITHUB_AW_REQUIRED_ROLES = "admin,maintainer";

    mockGithub.rest.repos.getCollaboratorPermissionLevel.mockResolvedValue({
      data: { permission: "maintain" },
    });

    const consoleSpy = vi.spyOn(console, "log").mockImplementation(() => {});

    // Execute the script
    await eval(`(async () => { ${checkPermissionsScript} })()`);

    expect(consoleSpy).toHaveBeenCalledWith(
      "✅ User has maintain access to repository"
    );

    // Should not call any error or warning methods
    expect(mockCore.error).not.toHaveBeenCalled();
    expect(mockCore.warning).not.toHaveBeenCalled();

    consoleSpy.mockRestore();
  });

  it("should pass validation for write permission when write is required", async () => {
    process.env.GITHUB_AW_REQUIRED_ROLES = "admin,write,triage";

    mockGithub.rest.repos.getCollaboratorPermissionLevel.mockResolvedValue({
      data: { permission: "write" },
    });

    const consoleSpy = vi.spyOn(console, "log").mockImplementation(() => {});

    // Execute the script
    await eval(`(async () => { ${checkPermissionsScript} })()`);

    expect(consoleSpy).toHaveBeenCalledWith(
      "✅ User has write access to repository"
    );

    // Should not call any error or warning methods
    expect(mockCore.error).not.toHaveBeenCalled();
    expect(mockCore.warning).not.toHaveBeenCalled();

    consoleSpy.mockRestore();
  });

  it("should fail the job for insufficient permission", async () => {
    process.env.GITHUB_AW_REQUIRED_ROLES = "admin,maintainer";

    mockGithub.rest.repos.getCollaboratorPermissionLevel.mockResolvedValue({
      data: { permission: "write" },
    });

    const consoleSpy = vi.spyOn(console, "log").mockImplementation(() => {});

    // Mock process.exit to prevent the test from actually exiting
    const processExitSpy = vi
      .spyOn(process, "exit")
      .mockImplementation(code => {
        // Throw an error to stop script execution, simulating process.exit behavior
        throw new Error(`Process exit called with code ${code}`);
      });

    try {
      // Execute the script - this should throw due to process.exit
      await eval(`(async () => { ${checkPermissionsScript} })()`);
    } catch (error) {
      // Expected to throw due to process.exit mock
      expect(error.message).toBe("Process exit called with code 78");
    }

    expect(consoleSpy).toHaveBeenCalledWith(
      "Repository permission level: write"
    );
    expect(consoleSpy).toHaveBeenCalledWith(
      "User permission 'write' does not meet requirements: admin, maintainer"
    );
    expect(mockCore.warning).toHaveBeenCalledWith(
      "❌ Access denied: Only authorized users can trigger this workflow. User 'testuser' is not authorized. Required permissions: admin, maintainer"
    );
    expect(processExitSpy).toHaveBeenCalledWith(78);

    consoleSpy.mockRestore();
    processExitSpy.mockRestore();
  });

  it("should fail the job for read permission", async () => {
    process.env.GITHUB_AW_REQUIRED_ROLES = "admin,write";

    mockGithub.rest.repos.getCollaboratorPermissionLevel.mockResolvedValue({
      data: { permission: "read" },
    });

    const consoleSpy = vi.spyOn(console, "log").mockImplementation(() => {});

    // Mock process.exit to prevent the test from actually exiting
    const processExitSpy = vi
      .spyOn(process, "exit")
      .mockImplementation(code => {
        // Throw an error to stop script execution, simulating process.exit behavior
        throw new Error(`Process exit called with code ${code}`);
      });

    try {
      // Execute the script - this should throw due to process.exit
      await eval(`(async () => { ${checkPermissionsScript} })()`);
    } catch (error) {
      // Expected to throw due to process.exit mock
      expect(error.message).toBe("Process exit called with code 78");
    }

    expect(consoleSpy).toHaveBeenCalledWith(
      "Repository permission level: read"
    );
    expect(consoleSpy).toHaveBeenCalledWith(
      "User permission 'read' does not meet requirements: admin, write"
    );
    expect(mockCore.warning).toHaveBeenCalledWith(
      "❌ Access denied: Only authorized users can trigger this workflow. User 'testuser' is not authorized. Required permissions: admin, write"
    );
    expect(processExitSpy).toHaveBeenCalledWith(78);

    consoleSpy.mockRestore();
    processExitSpy.mockRestore();
  });

  it("should fail the job on API errors", async () => {
    process.env.GITHUB_AW_REQUIRED_ROLES = "admin";

    const apiError = new Error("API Error: Not Found");
    mockGithub.rest.repos.getCollaboratorPermissionLevel.mockRejectedValue(
      apiError
    );

    const consoleSpy = vi.spyOn(console, "log").mockImplementation(() => {});

    // Mock process.exit to prevent the test from actually exiting
    const processExitSpy = vi
      .spyOn(process, "exit")
      .mockImplementation(code => {
        // Throw an error to stop script execution, simulating process.exit behavior
        throw new Error(`Process exit called with code ${code}`);
      });

    try {
      // Execute the script - this should throw due to process.exit
      await eval(`(async () => { ${checkPermissionsScript} })()`);
    } catch (error) {
      // Expected to throw due to process.exit mock
      expect(error.message).toBe("Process exit called with code 1");
    }

    expect(mockCore.error).toHaveBeenCalledWith(
      "Repository permission check failed: API Error: Not Found"
    );
    expect(processExitSpy).toHaveBeenCalledWith(1);

    consoleSpy.mockRestore();
    processExitSpy.mockRestore();
  });

  it("should handle different actor names correctly", async () => {
    process.env.GITHUB_AW_REQUIRED_ROLES = "admin";
    global.context.actor = "different-user";

    mockGithub.rest.repos.getCollaboratorPermissionLevel.mockResolvedValue({
      data: { permission: "admin" },
    });

    const consoleSpy = vi.spyOn(console, "log").mockImplementation(() => {});

    // Execute the script
    await eval(`(async () => { ${checkPermissionsScript} })()`);

    expect(
      mockGithub.rest.repos.getCollaboratorPermissionLevel
    ).toHaveBeenCalledWith({
      owner: "testowner",
      repo: "testrepo",
      username: "different-user",
    });

    expect(consoleSpy).toHaveBeenCalledWith(
      "Checking if user 'different-user' has required permissions for testowner/testrepo"
    );

    // Should not call any error or warning methods
    expect(mockCore.error).not.toHaveBeenCalled();
    expect(mockCore.warning).not.toHaveBeenCalled();

    consoleSpy.mockRestore();
  });

  it("should handle triage permission correctly", async () => {
    process.env.GITHUB_AW_REQUIRED_ROLES = "admin,write,triage";

    mockGithub.rest.repos.getCollaboratorPermissionLevel.mockResolvedValue({
      data: { permission: "triage" },
    });

    const consoleSpy = vi.spyOn(console, "log").mockImplementation(() => {});

    // Execute the script
    await eval(`(async () => { ${checkPermissionsScript} })()`);

    expect(consoleSpy).toHaveBeenCalledWith(
      "✅ User has triage access to repository"
    );

    // Should not call any error or warning methods
    expect(mockCore.error).not.toHaveBeenCalled();
    expect(mockCore.warning).not.toHaveBeenCalled();

    consoleSpy.mockRestore();
  });

  it("should handle single permission requirement", async () => {
    process.env.GITHUB_AW_REQUIRED_ROLES = "write";

    mockGithub.rest.repos.getCollaboratorPermissionLevel.mockResolvedValue({
      data: { permission: "write" },
    });

    const consoleSpy = vi.spyOn(console, "log").mockImplementation(() => {});

    // Execute the script
    await eval(`(async () => { ${checkPermissionsScript} })()`);

    expect(consoleSpy).toHaveBeenCalledWith("Required permissions: write");
    expect(consoleSpy).toHaveBeenCalledWith(
      "✅ User has write access to repository"
    );

    // Should not call any error or warning methods
    expect(mockCore.error).not.toHaveBeenCalled();
    expect(mockCore.warning).not.toHaveBeenCalled();

    consoleSpy.mockRestore();
  });

  it("should skip validation for workflow_run events", async () => {
    process.env.GITHUB_AW_REQUIRED_ROLES = "admin";
    global.context.eventName = "workflow_run";

    const consoleSpy = vi.spyOn(console, "log").mockImplementation(() => {});

    // Execute the script
    await eval(`(async () => { ${checkPermissionsScript} })()`);

    expect(consoleSpy).toHaveBeenCalledWith(
      "✅ Event workflow_run does not require validation"
    );
    expect(
      mockGithub.rest.repos.getCollaboratorPermissionLevel
    ).not.toHaveBeenCalled();

    consoleSpy.mockRestore();
  });

  it("should skip validation for schedule events", async () => {
    process.env.GITHUB_AW_REQUIRED_ROLES = "admin";
    global.context.eventName = "schedule";

    const consoleSpy = vi.spyOn(console, "log").mockImplementation(() => {});

    // Execute the script
    await eval(`(async () => { ${checkPermissionsScript} })()`);

    expect(consoleSpy).toHaveBeenCalledWith(
      "✅ Event schedule does not require validation"
    );
    expect(
      mockGithub.rest.repos.getCollaboratorPermissionLevel
    ).not.toHaveBeenCalled();

    consoleSpy.mockRestore();
  });
});
