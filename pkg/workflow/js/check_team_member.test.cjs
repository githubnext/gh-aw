import { describe, it, expect, beforeEach, vi } from "vitest";
import fs from "fs";
import path from "path";

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
  exportVariable: vi.fn(),
  setSecret: vi.fn(),

  // Input/state functions (less commonly used but included for completeness)
  getInput: vi.fn(),
  getBooleanInput: vi.fn(),
  getMultilineInput: vi.fn(),
  getState: vi.fn(),
  saveState: vi.fn(),

  // Group functions
  startGroup: vi.fn(),
  endGroup: vi.fn(),
  group: vi.fn(),

  // Other utility functions
  addPath: vi.fn(),
  setCommandEcho: vi.fn(),
  isDebug: vi.fn().mockReturnValue(false),
  getIDToken: vi.fn(),
  toPlatformPath: vi.fn(),
  toPosixPath: vi.fn(),
  toWin32Path: vi.fn(),

  // Summary object with chainable methods
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

const mockContext = {
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
    const scriptPath = path.join(process.cwd(), "check_team_member.cjs");
    checkTeamMemberScript = fs.readFileSync(scriptPath, "utf8");
  });

  it("should set is_team_member to true for admin permission", async () => {
    mockGithub.rest.repos.getCollaboratorPermissionLevel.mockResolvedValue({
      data: { permission: "admin" },
    });

    // Execute the script
    await eval(`(async () => { ${checkTeamMemberScript} })()`);

    expect(mockGithub.rest.repos.getCollaboratorPermissionLevel).toHaveBeenCalledWith({
      owner: "testowner",
      repo: "testrepo",
      username: "testuser",
    });

    expect(mockCore.info).toHaveBeenCalledWith(
      "Checking if user 'testuser' is admin or maintainer of testowner/testrepo"
    );
    expect(mockCore.info).toHaveBeenCalledWith("Repository permission level: admin");
    expect(mockCore.info).toHaveBeenCalledWith("User has admin access to repository");
    expect(mockCore.setOutput).toHaveBeenCalledWith("is_team_member", "true");
  });

  it("should set is_team_member to true for maintain permission", async () => {
    mockGithub.rest.repos.getCollaboratorPermissionLevel.mockResolvedValue({
      data: { permission: "maintain" },
    });

    // Execute the script
    await eval(`(async () => { ${checkTeamMemberScript} })()`);

    expect(mockGithub.rest.repos.getCollaboratorPermissionLevel).toHaveBeenCalledWith({
      owner: "testowner",
      repo: "testrepo",
      username: "testuser",
    });

    expect(mockCore.info).toHaveBeenCalledWith(
      "Checking if user 'testuser' is admin or maintainer of testowner/testrepo"
    );
    expect(mockCore.info).toHaveBeenCalledWith("Repository permission level: maintain");
    expect(mockCore.info).toHaveBeenCalledWith("User has maintain access to repository");
    expect(mockCore.setOutput).toHaveBeenCalledWith("is_team_member", "true");
  });

  it("should set is_team_member to false for write permission", async () => {
    mockGithub.rest.repos.getCollaboratorPermissionLevel.mockResolvedValue({
      data: { permission: "write" },
    });

    // Execute the script
    await eval(`(async () => { ${checkTeamMemberScript} })()`);

    expect(mockGithub.rest.repos.getCollaboratorPermissionLevel).toHaveBeenCalledWith({
      owner: "testowner",
      repo: "testrepo",
      username: "testuser",
    });

    expect(mockCore.info).toHaveBeenCalledWith(
      "Checking if user 'testuser' is admin or maintainer of testowner/testrepo"
    );
    expect(mockCore.info).toHaveBeenCalledWith("Repository permission level: write");
    expect(mockCore.setOutput).toHaveBeenCalledWith("is_team_member", "false");
  });

  it("should set is_team_member to false for read permission", async () => {
    mockGithub.rest.repos.getCollaboratorPermissionLevel.mockResolvedValue({
      data: { permission: "read" },
    });

    // Execute the script
    await eval(`(async () => { ${checkTeamMemberScript} })()`);

    expect(mockGithub.rest.repos.getCollaboratorPermissionLevel).toHaveBeenCalledWith({
      owner: "testowner",
      repo: "testrepo",
      username: "testuser",
    });

    expect(mockCore.info).toHaveBeenCalledWith(
      "Checking if user 'testuser' is admin or maintainer of testowner/testrepo"
    );
    expect(mockCore.info).toHaveBeenCalledWith("Repository permission level: read");
    expect(mockCore.setOutput).toHaveBeenCalledWith("is_team_member", "false");
  });

  it("should set is_team_member to false for none permission", async () => {
    mockGithub.rest.repos.getCollaboratorPermissionLevel.mockResolvedValue({
      data: { permission: "none" },
    });

    // Execute the script
    await eval(`(async () => { ${checkTeamMemberScript} })()`);

    expect(mockGithub.rest.repos.getCollaboratorPermissionLevel).toHaveBeenCalledWith({
      owner: "testowner",
      repo: "testrepo",
      username: "testuser",
    });

    expect(mockCore.info).toHaveBeenCalledWith(
      "Checking if user 'testuser' is admin or maintainer of testowner/testrepo"
    );
    expect(mockCore.info).toHaveBeenCalledWith("Repository permission level: none");
    expect(mockCore.setOutput).toHaveBeenCalledWith("is_team_member", "false");
  });

  it("should handle API errors and set is_team_member to false", async () => {
    const apiError = new Error("API Error: Not Found");
    mockGithub.rest.repos.getCollaboratorPermissionLevel.mockRejectedValue(apiError);

    // Execute the script
    await eval(`(async () => { ${checkTeamMemberScript} })()`);

    expect(mockGithub.rest.repos.getCollaboratorPermissionLevel).toHaveBeenCalledWith({
      owner: "testowner",
      repo: "testrepo",
      username: "testuser",
    });

    expect(mockCore.info).toHaveBeenCalledWith(
      "Checking if user 'testuser' is admin or maintainer of testowner/testrepo"
    );
    expect(mockCore.warning).toHaveBeenCalledWith("Repository permission check failed: API Error: Not Found");
    expect(mockCore.setOutput).toHaveBeenCalledWith("is_team_member", "false");
  });

  it("should handle different actor names correctly", async () => {
    global.context.actor = "different-user";

    mockGithub.rest.repos.getCollaboratorPermissionLevel.mockResolvedValue({
      data: { permission: "admin" },
    });

    // Execute the script
    await eval(`(async () => { ${checkTeamMemberScript} })()`);

    expect(mockGithub.rest.repos.getCollaboratorPermissionLevel).toHaveBeenCalledWith({
      owner: "testowner",
      repo: "testrepo",
      username: "different-user",
    });

    expect(mockCore.info).toHaveBeenCalledWith(
      "Checking if user 'different-user' is admin or maintainer of testowner/testrepo"
    );
    expect(mockCore.setOutput).toHaveBeenCalledWith("is_team_member", "true");
  });

  it("should handle different repository contexts correctly", async () => {
    global.context.repo = {
      owner: "different-owner",
      repo: "different-repo",
    };

    mockGithub.rest.repos.getCollaboratorPermissionLevel.mockResolvedValue({
      data: { permission: "maintain" },
    });

    // Execute the script
    await eval(`(async () => { ${checkTeamMemberScript} })()`);

    expect(mockGithub.rest.repos.getCollaboratorPermissionLevel).toHaveBeenCalledWith({
      owner: "different-owner",
      repo: "different-repo",
      username: "testuser",
    });

    expect(mockCore.info).toHaveBeenCalledWith(
      "Checking if user 'testuser' is admin or maintainer of different-owner/different-repo"
    );
    expect(mockCore.setOutput).toHaveBeenCalledWith("is_team_member", "true");
  });

  it("should handle authentication errors gracefully", async () => {
    const authError = new Error("Bad credentials");
    authError.status = 401;
    mockGithub.rest.repos.getCollaboratorPermissionLevel.mockRejectedValue(authError);

    // Execute the script
    await eval(`(async () => { ${checkTeamMemberScript} })()`);

    expect(mockCore.warning).toHaveBeenCalledWith("Repository permission check failed: Bad credentials");
    expect(mockCore.setOutput).toHaveBeenCalledWith("is_team_member", "false");
  });

  it("should handle rate limiting errors gracefully", async () => {
    const rateLimitError = new Error("API rate limit exceeded");
    rateLimitError.status = 403;
    mockGithub.rest.repos.getCollaboratorPermissionLevel.mockRejectedValue(rateLimitError);

    // Execute the script
    await eval(`(async () => { ${checkTeamMemberScript} })()`);

    expect(mockCore.warning).toHaveBeenCalledWith("Repository permission check failed: API rate limit exceeded");
    expect(mockCore.setOutput).toHaveBeenCalledWith("is_team_member", "false");
  });
});
