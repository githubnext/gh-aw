import { describe, it, expect, beforeAll, beforeEach, afterEach, vi } from "vitest";

let createProject;
let generateCampaignId;

const mockCore = {
  debug: vi.fn(),
  info: vi.fn(),
  notice: vi.fn(),
  warning: vi.fn(),
  error: vi.fn(),
  setFailed: vi.fn(),
  setOutput: vi.fn(),
  exportVariable: vi.fn(),
  getInput: vi.fn(),
  summary: {
    addRaw: vi.fn().mockReturnThis(),
    write: vi.fn().mockResolvedValue(),
  },
};

const mockGithub = {
  rest: {},
  graphql: vi.fn(),
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

global.core = mockCore;
global.github = mockGithub;
global.context = mockContext;

beforeAll(async () => {
  const mod = await import("./create_project.cjs");
  const exports = mod.default || mod;
  createProject = exports.createProject;

  // Import generateCampaignId from helper
  const helperMod = await import("./project_helpers.cjs");
  const helperExports = helperMod.default || helperMod;
  generateCampaignId = helperExports.generateCampaignId;
});

function clearMock(fn) {
  if (fn && typeof fn.mockClear === "function") {
    fn.mockClear();
  }
}

function clearCoreMocks() {
  clearMock(mockCore.debug);
  clearMock(mockCore.info);
  clearMock(mockCore.notice);
  clearMock(mockCore.warning);
  clearMock(mockCore.error);
  clearMock(mockCore.setFailed);
  clearMock(mockCore.setOutput);
  clearMock(mockCore.exportVariable);
  clearMock(mockCore.getInput);
  clearMock(mockCore.summary.addRaw);
  clearMock(mockCore.summary.write);
}

beforeEach(() => {
  mockGithub.graphql.mockReset();
  clearCoreMocks();
  vi.useRealTimers();
});

afterEach(() => {
  vi.useRealTimers();
});

const repoResponse = (ownerType = "Organization") => ({
  repository: {
    id: "repo123",
    owner: {
      id: ownerType === "User" ? "owner-user-123" : "owner123",
      __typename: ownerType,
    },
  },
});

const ownerProjectsResponse = nodes => ({ organization: { projectsV2: { nodes } } });

const linkResponse = { linkProjectV2ToRepository: { repository: { id: "repo123" } } };

function queueResponses(responses) {
  responses.forEach(response => {
    mockGithub.graphql.mockResolvedValueOnce(response);
  });
}

function getOutput(name) {
  const call = mockCore.setOutput.mock.calls.find(([key]) => key === name);
  return call ? call[1] : undefined;
}

describe("generateCampaignId", () => {
  it("builds a slug with a timestamp suffix", () => {
    const nowSpy = vi.spyOn(Date, "now").mockReturnValue(1734470400000);
    const id = generateCampaignId("Bug Bash Q1 2025");
    expect(id).toBe("bug-bash-q1-2025-m4syw5xc");
    nowSpy.mockRestore();
  });
});

describe("createProject", () => {
  it("creates a new project when none exist", async () => {
    const output = { type: "create_project", project: "New Campaign" };

    queueResponses([
      repoResponse(),
      ownerProjectsResponse([]),
      {
        createProjectV2: {
          projectV2: {
            id: "project123",
            title: "New Campaign",
            url: "https://github.com/orgs/testowner/projects/1",
            number: 1,
          },
        },
      },
      linkResponse,
    ]);

    await createProject(output);

    expect(mockCore.info).toHaveBeenCalledWith("✓ Created project: New Campaign");
    expect(getOutput("project-id")).toBe("project123");
    expect(getOutput("project-number")).toBe(1);
    expect(getOutput("project-url")).toBe("https://github.com/orgs/testowner/projects/1");
    expect(getOutput("campaign-id")).toMatch(/^new-campaign-[a-z0-9]{8}$/);

    expect(mockGithub.graphql).toHaveBeenCalledWith(
      expect.stringContaining("createProjectV2"),
      expect.objectContaining({ ownerId: "owner123", title: "New Campaign" })
    );
  });

  it("respects a custom campaign id", async () => {
    const output = { type: "create_project", project: "Custom Campaign", campaign_id: "custom-id-2025" };

    queueResponses([
      repoResponse(),
      ownerProjectsResponse([]),
      {
        createProjectV2: {
          projectV2: {
            id: "project456",
            title: "Custom Campaign",
            url: "https://github.com/orgs/testowner/projects/2",
            number: 2,
          },
        },
      },
      linkResponse,
    ]);

    await createProject(output);

    expect(getOutput("campaign-id")).toBe("custom-id-2025");
    expect(mockCore.info).toHaveBeenCalledWith("✓ Created project: Custom Campaign");
  });

  it("handles existing project gracefully", async () => {
    const output = { type: "create_project", project: "Existing Campaign" };

    queueResponses([
      repoResponse(),
      ownerProjectsResponse([{ id: "existing-project-123", title: "Existing Campaign", number: 5 }]),
      linkResponse,
    ]);

    await createProject(output);

    const createCall = mockGithub.graphql.mock.calls.find(([query]) => query.includes("createProjectV2"));
    expect(createCall).toBeUndefined();
    expect(mockCore.info).toHaveBeenCalledWith(expect.stringContaining("✓ Project already exists"));
    expect(getOutput("project-id")).toBe("existing-project-123");
    expect(getOutput("project-number")).toBe(5);
  });

  it("throws error for user accounts", async () => {
    const output = { type: "create_project", project: "User Project" };

    queueResponses([repoResponse("User")]);

    await expect(createProject(output)).rejects.toThrow(/Cannot create project.*on user account/);
    expect(mockCore.error).toHaveBeenCalledWith(expect.stringContaining("Cannot create projects on user accounts"));
  });

  it("throws error for invalid project name", async () => {
    const output = { type: "create_project", project: null };

    await expect(createProject(output)).rejects.toThrow(/Invalid project name/);
  });

  it("surfaces project creation failures with helpful messages", async () => {
    const output = { type: "create_project", project: "Fail Project" };

    queueResponses([repoResponse(), ownerProjectsResponse([])]);

    mockGithub.graphql.mockRejectedValueOnce(new Error("does not have permission to create projects"));

    await expect(createProject(output)).rejects.toThrow(/permission to create projects/);
    expect(mockCore.error).toHaveBeenCalledWith(expect.stringContaining("Failed to create project"));
    expect(mockCore.error).toHaveBeenCalledWith(expect.stringContaining("Troubleshooting"));
  });

  it("warns when linking project fails (non-fatal)", async () => {
    const output = { type: "create_project", project: "Existing Campaign" };

    queueResponses([repoResponse(), ownerProjectsResponse([{ id: "existing-project-123", title: "Existing Campaign", number: 5 }])]);

    mockGithub.graphql.mockRejectedValueOnce(new Error("Link failed"));

    await createProject(output);

    expect(mockCore.warning).toHaveBeenCalledWith(expect.stringContaining("Could not link project"));
  });
});
