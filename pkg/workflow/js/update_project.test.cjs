import { describe, it, expect, beforeAll, beforeEach, afterEach, vi } from "vitest";

let updateProject;
let parseProjectInput;
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
  rest: {
    issues: {
      addLabels: vi.fn().mockResolvedValue({}),
    },
  },
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
  const mod = await import("./update_project.cjs");
  const exports = mod.default || mod;
  updateProject = exports.updateProject;
  parseProjectInput = exports.parseProjectInput;
  generateCampaignId = exports.generateCampaignId;
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
  mockGithub.rest.issues.addLabels.mockClear();
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

const ownerProjectsResponse = (nodes, ownerType = "Organization") =>
  ownerType === "User" ? { user: { projectsV2: { nodes } } } : { organization: { projectsV2: { nodes } } };

const linkResponse = { linkProjectV2ToRepository: { repository: { id: "repo123" } } };

const issueResponse = id => ({ repository: { issue: { id } } });

const pullRequestResponse = id => ({ repository: { pullRequest: { id } } });

const emptyItemsResponse = () => ({
  node: {
    items: {
      nodes: [],
      pageInfo: { hasNextPage: false, endCursor: null },
    },
  },
});

const existingItemResponse = (contentId, itemId = "existing-item") => ({
  node: {
    items: {
      nodes: [{ id: itemId, content: { id: contentId } }],
      pageInfo: { hasNextPage: false, endCursor: null },
    },
  },
});

const fieldsResponse = nodes => ({ node: { fields: { nodes } } });

const updateFieldValueResponse = () => ({
  updateProjectV2ItemFieldValue: {
    projectV2Item: {
      id: "item123",
    },
  },
});

function queueResponses(responses) {
  responses.forEach(response => {
    mockGithub.graphql.mockResolvedValueOnce(response);
  });
}

function getOutput(name) {
  const call = mockCore.setOutput.mock.calls.find(([key]) => key === name);
  return call ? call[1] : undefined;
}

describe("parseProjectInput", () => {
  it("extracts the project number from a GitHub URL", () => {
    expect(parseProjectInput("https://github.com/orgs/acme/projects/42")).toEqual({
      projectNumber: "42",
      projectName: null,
    });
  });

  it("treats a numeric string as a project number", () => {
    expect(parseProjectInput("17")).toEqual({ projectNumber: "17", projectName: null });
  });

  it("returns the project name when no number is present", () => {
    expect(parseProjectInput("Engineering Roadmap")).toEqual({ projectNumber: null, projectName: "Engineering Roadmap" });
  });

  it("throws when the project input is missing", () => {
    expect(() => parseProjectInput(undefined)).toThrow(/Invalid project input/);
  });
});

describe("generateCampaignId", () => {
  it("builds a slug with a timestamp suffix", () => {
    const nowSpy = vi.spyOn(Date, "now").mockReturnValue(1734470400000);
    const id = generateCampaignId("Bug Bash Q1 2025");
    expect(id).toBe("bug-bash-q1-2025-m4syw5xc");
    nowSpy.mockRestore();
  });
});

describe("updateProject", () => {
  it("creates a new project when none exist", async () => {
    const output = { type: "update_project", project: "New Campaign" };

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

    await updateProject(output);

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
    const output = { type: "update_project", project: "Custom Campaign", campaign_id: "custom-id-2025" };

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

    await updateProject(output);

    expect(getOutput("campaign-id")).toBe("custom-id-2025");
    expect(mockCore.info).toHaveBeenCalledWith("✓ Created project: Custom Campaign");
  });

  it("finds an existing project by title", async () => {
    const output = { type: "update_project", project: "Existing Campaign" };

    queueResponses([
      repoResponse(),
      ownerProjectsResponse([{ id: "existing-project-123", title: "Existing Campaign", number: 5 }]),
      linkResponse,
    ]);

    await updateProject(output);

    const createCall = mockGithub.graphql.mock.calls.find(([query]) => query.includes("createProjectV2"));
    expect(createCall).toBeUndefined();
  });

  it("finds an existing project by number", async () => {
    const output = { type: "update_project", project: "7" };

    queueResponses([repoResponse(), ownerProjectsResponse([{ id: "project-by-number", title: "Bug Tracking", number: 7 }]), linkResponse]);

    await updateProject(output);

    const createCall = mockGithub.graphql.mock.calls.find(([query]) => query.includes("createProjectV2"));
    expect(createCall).toBeUndefined();
  });

  it("adds an issue to a project board", async () => {
    const output = { type: "update_project", project: "Bug Tracking", content_type: "issue", content_number: 42 };

    queueResponses([
      repoResponse(),
      ownerProjectsResponse([{ id: "project123", title: "Bug Tracking", number: 1 }]),
      linkResponse,
      issueResponse("issue-id-42"),
      emptyItemsResponse(),
      { addProjectV2ItemById: { item: { id: "item123" } } },
    ]);

    await updateProject(output);

    const labelCall = mockGithub.rest.issues.addLabels.mock.calls[0][0];
    expect(labelCall).toEqual(
      expect.objectContaining({
        owner: "testowner",
        repo: "testrepo",
        issue_number: 42,
      })
    );
    expect(labelCall.labels).toEqual([expect.stringMatching(/^campaign:bug-tracking-[a-z0-9]{8}$/)]);
    expect(getOutput("item-id")).toBe("item123");
  });

  it("skips adding an issue that already exists on the board", async () => {
    const output = { type: "update_project", project: "Bug Tracking", content_type: "issue", content_number: 99 };

    queueResponses([
      repoResponse(),
      ownerProjectsResponse([{ id: "project123", title: "Bug Tracking", number: 1 }]),
      linkResponse,
      issueResponse("issue-id-99"),
      existingItemResponse("issue-id-99", "item-existing"),
    ]);

    await updateProject(output);

    expect(mockGithub.rest.issues.addLabels).not.toHaveBeenCalled();
    expect(mockCore.info).toHaveBeenCalledWith("✓ Item already on board");
    expect(getOutput("item-id")).toBe("item-existing");
  });

  it("adds a pull request to the project board", async () => {
    const output = { type: "update_project", project: "PR Review Board", content_type: "pull_request", content_number: 17 };

    queueResponses([
      repoResponse(),
      ownerProjectsResponse([{ id: "project-pr", title: "PR Review Board", number: 9 }]),
      linkResponse,
      pullRequestResponse("pr-id-17"),
      emptyItemsResponse(),
      { addProjectV2ItemById: { item: { id: "pr-item" } } },
    ]);

    await updateProject(output);

    const labelCall = mockGithub.rest.issues.addLabels.mock.calls[0][0];
    expect(labelCall).toEqual(
      expect.objectContaining({
        owner: "testowner",
        repo: "testrepo",
        issue_number: 17,
      })
    );
    expect(labelCall.labels).toEqual([expect.stringMatching(/^campaign:pr-review-board-[a-z0-9]{8}$/)]);
  });

  it("falls back to legacy issue field when content_number missing", async () => {
    const output = { type: "update_project", project: "Legacy Board", issue: "101" };

    queueResponses([
      repoResponse(),
      ownerProjectsResponse([{ id: "legacy-project", title: "Legacy Board", number: 6 }]),
      linkResponse,
      issueResponse("issue-id-101"),
      emptyItemsResponse(),
      { addProjectV2ItemById: { item: { id: "legacy-item" } } },
    ]);

    await updateProject(output);

    expect(mockCore.warning).toHaveBeenCalledWith('Field "issue" deprecated; use "content_number" instead.');

    const labelCall = mockGithub.rest.issues.addLabels.mock.calls[0][0];
    expect(labelCall.issue_number).toBe(101);
    expect(getOutput("item-id")).toBe("legacy-item");
  });

  it("rejects invalid content numbers", async () => {
    const output = { type: "update_project", project: "Invalid Board", content_number: "ABC" };

    queueResponses([repoResponse(), ownerProjectsResponse([{ id: "invalid-project", title: "Invalid Board", number: 7 }]), linkResponse]);

    await expect(updateProject(output)).rejects.toThrow(/Invalid content number/);
  });

  it("updates an existing text field", async () => {
    const output = {
      type: "update_project",
      project: "Field Test",
      content_type: "issue",
      content_number: 10,
      fields: { Status: "In Progress" },
    };

    queueResponses([
      repoResponse(),
      ownerProjectsResponse([{ id: "project-field", title: "Field Test", number: 12 }]),
      linkResponse,
      issueResponse("issue-id-10"),
      existingItemResponse("issue-id-10", "item-field"),
      fieldsResponse([{ id: "field-status", name: "Status" }]),
      updateFieldValueResponse(),
    ]);

    await updateProject(output);

    const updateCall = mockGithub.graphql.mock.calls.find(([query]) => query.includes("updateProjectV2ItemFieldValue"));
    expect(updateCall).toBeDefined();
    expect(mockGithub.rest.issues.addLabels).not.toHaveBeenCalled();
  });

  it("updates a single select field when the option exists", async () => {
    const output = {
      type: "update_project",
      project: "Priority Board",
      content_type: "issue",
      content_number: 15,
      fields: { Priority: "High" },
    };

    queueResponses([
      repoResponse(),
      ownerProjectsResponse([{ id: "project-priority", title: "Priority Board", number: 3 }]),
      linkResponse,
      issueResponse("issue-id-15"),
      existingItemResponse("issue-id-15", "item-priority"),
      fieldsResponse([
        {
          id: "field-priority",
          name: "Priority",
          options: [
            { id: "opt-low", name: "Low" },
            { id: "opt-high", name: "High" },
          ],
        },
      ]),
      updateFieldValueResponse(),
    ]);

    await updateProject(output);

    const updateCall = mockGithub.graphql.mock.calls.find(([query]) => query.includes("updateProjectV2ItemFieldValue"));
    expect(updateCall).toBeDefined();
  });

  it("warns when a field cannot be created", async () => {
    const output = {
      type: "update_project",
      project: "Test Project",
      content_type: "issue",
      content_number: 20,
      fields: { NonExistentField: "Some Value" },
    };

    queueResponses([
      repoResponse(),
      ownerProjectsResponse([{ id: "project-test", title: "Test Project", number: 4 }]),
      linkResponse,
      issueResponse("issue-id-20"),
      existingItemResponse("issue-id-20", "item-test"),
      fieldsResponse([]),
    ]);

    mockGithub.graphql.mockRejectedValueOnce(new Error("Failed to create field"));

    await updateProject(output);

    expect(mockCore.warning).toHaveBeenCalledWith(expect.stringContaining('Failed to create field "NonExistentField"'));
  });

  it("warns when adding the campaign label fails", async () => {
    const output = { type: "update_project", project: "Label Test", content_type: "issue", content_number: 50 };

    queueResponses([
      repoResponse(),
      ownerProjectsResponse([{ id: "project-label", title: "Label Test", number: 11 }]),
      linkResponse,
      issueResponse("issue-id-50"),
      emptyItemsResponse(),
      { addProjectV2ItemById: { item: { id: "item-label" } } },
    ]);

    mockGithub.rest.issues.addLabels.mockRejectedValueOnce(new Error("Labels disabled"));

    await updateProject(output);

    expect(mockCore.warning).toHaveBeenCalledWith(expect.stringContaining("Failed to add campaign label"));
  });

  it("surfaces project creation failures", async () => {
    const output = { type: "update_project", project: "Fail Project" };

    queueResponses([repoResponse(), ownerProjectsResponse([])]);

    mockGithub.graphql.mockRejectedValueOnce(new Error("GraphQL error: Insufficient permissions"));

    await expect(updateProject(output)).rejects.toThrow(/Insufficient permissions/);
    expect(mockCore.error).toHaveBeenCalledWith(expect.stringContaining("Failed to manage project"));
  });
});
