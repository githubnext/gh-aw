import { describe, it, expect, beforeEach, vi } from "vitest";
import fs from "fs";
import path from "path";

// Mock the global objects that GitHub Actions provides
const mockCore = {
  setOutput: vi.fn(),
  warning: vi.fn(),
  error: vi.fn(),
  info: vi.fn(),
  setCancelled: vi.fn(),
  setFailed: vi.fn(),
};

const mockGithub = {
  rest: {
    repos: {
      getCollaboratorPermissionLevel: vi.fn(),
    },
    actions: {
      cancelWorkflowRun: vi.fn(),
    },
  },
};

const mockContext = {
  actor: "testuser",
  repo: {
    owner: "testowner",
    repo: "testrepo",
  },
  runId: 12345,
};

// Set up global variables
global.core = mockCore;
global.github = mockGithub;
global.context = mockContext;

describe("check_team_member.cjs", () => {
  let checkTeamMemberScript;

  beforeEach(() => {
    // Reset all mocks
    vi.clearAllMocks();

    // Reset context to default state
    global.context.actor = "testuser";
    global.context.repo = {
      owner: "testowner",
      repo: "testrepo",
    };

    // Read the script content
    const scriptPath = path.join(
      process.cwd(),
      "pkg/workflow/js/check_team_member.cjs"
    );
    checkTeamMemberScript = fs.readFileSync(scriptPath, "utf8");
  });

  it("should set is_team_member to true for admin permission", async () => {
    mockGithub.rest.repos.getCollaboratorPermissionLevel.mockResolvedValue({
      data: { permission: "admin" },
    });

    const consoleSpy = vi.spyOn(console, "log").mockImplementation(() => {});

    // Execute the script
    await eval(`(async () => { ${checkTeamMemberScript} })()`);

    expect(
      mockGithub.rest.repos.getCollaboratorPermissionLevel
    ).toHaveBeenCalledWith({
      owner: "testowner",
      repo: "testrepo",
      username: "testuser",
    });

    expect(consoleSpy).toHaveBeenCalledWith(
      "Checking if user 'testuser' is admin or maintainer of testowner/testrepo"
    );
    expect(consoleSpy).toHaveBeenCalledWith(
      "Repository permission level: admin"
    );
    expect(consoleSpy).toHaveBeenCalledWith(
      "User has admin access to repository"
    );
    expect(mockCore.setOutput).toHaveBeenCalledWith("is_team_member", "true");

    consoleSpy.mockRestore();
  });

  it("should set is_team_member to true for maintain permission", async () => {
    mockGithub.rest.repos.getCollaboratorPermissionLevel.mockResolvedValue({
      data: { permission: "maintain" },
    });

    const consoleSpy = vi.spyOn(console, "log").mockImplementation(() => {});

    // Execute the script
    await eval(`(async () => { ${checkTeamMemberScript} })()`);

    expect(
      mockGithub.rest.repos.getCollaboratorPermissionLevel
    ).toHaveBeenCalledWith({
      owner: "testowner",
      repo: "testrepo",
      username: "testuser",
    });

    expect(consoleSpy).toHaveBeenCalledWith(
      "Checking if user 'testuser' is admin or maintainer of testowner/testrepo"
    );
    expect(consoleSpy).toHaveBeenCalledWith(
      "Repository permission level: maintain"
    );
    expect(consoleSpy).toHaveBeenCalledWith(
      "User has maintain access to repository"
    );
    expect(mockCore.setOutput).toHaveBeenCalledWith("is_team_member", "true");

    consoleSpy.mockRestore();
  });

  it("should set is_team_member to false for write permission", async () => {
    mockGithub.rest.repos.getCollaboratorPermissionLevel.mockResolvedValue({
      data: { permission: "write" },
    });

    const consoleSpy = vi.spyOn(console, "log").mockImplementation(() => {});

    // Execute the script
    await eval(`(async () => { ${checkTeamMemberScript} })()`);

    expect(
      mockGithub.rest.repos.getCollaboratorPermissionLevel
    ).toHaveBeenCalledWith({
      owner: "testowner",
      repo: "testrepo",
      username: "testuser",
    });

    expect(consoleSpy).toHaveBeenCalledWith(
      "Checking if user 'testuser' is admin or maintainer of testowner/testrepo"
    );
    expect(consoleSpy).toHaveBeenCalledWith(
      "Repository permission level: write"
    );
    expect(mockCore.setOutput).toHaveBeenCalledWith("is_team_member", "false");

    consoleSpy.mockRestore();
  });

  it("should use self-cancellation when team membership fails", async () => {
    mockGithub.rest.repos.getCollaboratorPermissionLevel.mockResolvedValue({
      data: { permission: "read" },
    });

    mockGithub.rest.actions.cancelWorkflowRun.mockResolvedValue({});

    const consoleSpy = vi.spyOn(console, "log").mockImplementation(() => {});

    // Execute the script
    await eval(`(async () => { ${checkTeamMemberScript} })()`);

    expect(
      mockGithub.rest.repos.getCollaboratorPermissionLevel
    ).toHaveBeenCalledWith({
      owner: "testowner",
      repo: "testrepo",
      username: "testuser",
    });

    expect(consoleSpy).toHaveBeenCalledWith(
      "Checking if user 'testuser' is admin or maintainer of testowner/testrepo"
    );
    expect(consoleSpy).toHaveBeenCalledWith(
      "Repository permission level: read"
    );

    // Check that self-cancellation was attempted
    expect(mockGithub.rest.actions.cancelWorkflowRun).toHaveBeenCalledWith({
      owner: "testowner",
      repo: "testrepo",
      run_id: 12345,
    });

    expect(mockCore.info).toHaveBeenCalledWith(
      expect.stringContaining("Cancellation requested for this workflow run")
    );
    expect(mockCore.warning).toHaveBeenCalledWith(
      expect.stringContaining(
        "Access denied: User 'testuser' is not authorized"
      )
    );
    expect(mockCore.setOutput).toHaveBeenCalledWith("is_team_member", "false");

    consoleSpy.mockRestore();
  });

  it("should fallback to core.setCancelled when API call fails", async () => {
    mockGithub.rest.repos.getCollaboratorPermissionLevel.mockResolvedValue({
      data: { permission: "read" },
    });

    const apiError = new Error("API Error: Forbidden");
    mockGithub.rest.actions.cancelWorkflowRun.mockRejectedValue(apiError);

    const consoleSpy = vi.spyOn(console, "log").mockImplementation(() => {});

    // Execute the script
    await eval(`(async () => { ${checkTeamMemberScript} })()`);

    expect(
      mockGithub.rest.repos.getCollaboratorPermissionLevel
    ).toHaveBeenCalledWith({
      owner: "testowner",
      repo: "testrepo",
      username: "testuser",
    });

    // Check that self-cancellation was attempted but failed
    expect(mockGithub.rest.actions.cancelWorkflowRun).toHaveBeenCalledWith({
      owner: "testowner",
      repo: "testrepo",
      run_id: 12345,
    });

    // Check that fallback occurred
    expect(mockCore.warning).toHaveBeenCalledWith(
      "Failed to cancel workflow run: API Error: Forbidden"
    );
    expect(mockCore.setCancelled).toHaveBeenCalledWith(
      expect.stringContaining(
        "Access denied: User 'testuser' is not authorized"
      )
    );
    expect(mockCore.setOutput).toHaveBeenCalledWith("is_team_member", "false");

    consoleSpy.mockRestore();
  });

  it("should set is_team_member to false for none permission", async () => {
    mockGithub.rest.repos.getCollaboratorPermissionLevel.mockResolvedValue({
      data: { permission: "none" },
    });

    const consoleSpy = vi.spyOn(console, "log").mockImplementation(() => {});

    // Execute the script
    await eval(`(async () => { ${checkTeamMemberScript} })()`);

    expect(
      mockGithub.rest.repos.getCollaboratorPermissionLevel
    ).toHaveBeenCalledWith({
      owner: "testowner",
      repo: "testrepo",
      username: "testuser",
    });

    expect(consoleSpy).toHaveBeenCalledWith(
      "Checking if user 'testuser' is admin or maintainer of testowner/testrepo"
    );
    expect(consoleSpy).toHaveBeenCalledWith(
      "Repository permission level: none"
    );
    expect(mockCore.setOutput).toHaveBeenCalledWith("is_team_member", "false");

    consoleSpy.mockRestore();
  });

  it("should handle API errors and set is_team_member to false", async () => {
    const apiError = new Error("API Error: Not Found");
    mockGithub.rest.repos.getCollaboratorPermissionLevel.mockRejectedValue(
      apiError
    );

    const consoleSpy = vi.spyOn(console, "log").mockImplementation(() => {});

    // Execute the script
    await eval(`(async () => { ${checkTeamMemberScript} })()`);

    expect(
      mockGithub.rest.repos.getCollaboratorPermissionLevel
    ).toHaveBeenCalledWith({
      owner: "testowner",
      repo: "testrepo",
      username: "testuser",
    });

    expect(consoleSpy).toHaveBeenCalledWith(
      "Checking if user 'testuser' is admin or maintainer of testowner/testrepo"
    );
    expect(mockCore.warning).toHaveBeenCalledWith(
      "Repository permission check failed: API Error: Not Found"
    );
    expect(mockCore.setOutput).toHaveBeenCalledWith("is_team_member", "false");

    consoleSpy.mockRestore();
  });

  it("should handle different actor names correctly", async () => {
    global.context.actor = "different-user";

    mockGithub.rest.repos.getCollaboratorPermissionLevel.mockResolvedValue({
      data: { permission: "admin" },
    });

    const consoleSpy = vi.spyOn(console, "log").mockImplementation(() => {});

    // Execute the script
    await eval(`(async () => { ${checkTeamMemberScript} })()`);

    expect(
      mockGithub.rest.repos.getCollaboratorPermissionLevel
    ).toHaveBeenCalledWith({
      owner: "testowner",
      repo: "testrepo",
      username: "different-user",
    });

    expect(consoleSpy).toHaveBeenCalledWith(
      "Checking if user 'different-user' is admin or maintainer of testowner/testrepo"
    );
    expect(mockCore.setOutput).toHaveBeenCalledWith("is_team_member", "true");

    consoleSpy.mockRestore();
  });

  it("should handle different repository contexts correctly", async () => {
    global.context.repo = {
      owner: "different-owner",
      repo: "different-repo",
    };

    mockGithub.rest.repos.getCollaboratorPermissionLevel.mockResolvedValue({
      data: { permission: "maintain" },
    });

    const consoleSpy = vi.spyOn(console, "log").mockImplementation(() => {});

    // Execute the script
    await eval(`(async () => { ${checkTeamMemberScript} })()`);

    expect(
      mockGithub.rest.repos.getCollaboratorPermissionLevel
    ).toHaveBeenCalledWith({
      owner: "different-owner",
      repo: "different-repo",
      username: "testuser",
    });

    expect(consoleSpy).toHaveBeenCalledWith(
      "Checking if user 'testuser' is admin or maintainer of different-owner/different-repo"
    );
    expect(mockCore.setOutput).toHaveBeenCalledWith("is_team_member", "true");

    consoleSpy.mockRestore();
  });

  it("should handle authentication errors gracefully", async () => {
    const authError = new Error("Bad credentials");
    authError.status = 401;
    mockGithub.rest.repos.getCollaboratorPermissionLevel.mockRejectedValue(
      authError
    );

    const consoleSpy = vi.spyOn(console, "log").mockImplementation(() => {});

    // Execute the script
    await eval(`(async () => { ${checkTeamMemberScript} })()`);

    expect(mockCore.warning).toHaveBeenCalledWith(
      "Repository permission check failed: Bad credentials"
    );
    expect(mockCore.setOutput).toHaveBeenCalledWith("is_team_member", "false");

    consoleSpy.mockRestore();
  });

  it("should handle rate limiting errors gracefully", async () => {
    const rateLimitError = new Error("API rate limit exceeded");
    rateLimitError.status = 403;
    mockGithub.rest.repos.getCollaboratorPermissionLevel.mockRejectedValue(
      rateLimitError
    );

    const consoleSpy = vi.spyOn(console, "log").mockImplementation(() => {});

    // Execute the script
    await eval(`(async () => { ${checkTeamMemberScript} })()`);

    expect(mockCore.warning).toHaveBeenCalledWith(
      "Repository permission check failed: API rate limit exceeded"
    );
    expect(mockCore.setOutput).toHaveBeenCalledWith("is_team_member", "false");

    consoleSpy.mockRestore();
  });
});
