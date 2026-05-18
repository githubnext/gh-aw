// @ts-check
/**
 * Integration tests for graph-based safe-output handlers with temporary IDs.
 *
 * These tests verify the end-to-end cross-handler flows where a temporary ID
 * produced by one handler is consumed by a subsequent handler via the shared
 * `temporaryIdMap`.  The covered chains are:
 *
 *   create_project → create_project_status_update
 *   create_project → update_project  (project field)
 *   issue (pre-seeded map entry) → create_project (item_url) → update_project (content_number)
 */

import { describe, it, expect, beforeAll, beforeEach, vi } from "vitest";

// ─── shared mocks ─────────────────────────────────────────────────────────────

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
  summary: { addRaw: vi.fn().mockReturnThis(), write: vi.fn().mockResolvedValue(undefined) },
};

const mockGithub = {
  graphql: vi.fn(),
  rest: { issues: { addLabels: vi.fn().mockResolvedValue({}) } },
  request: vi.fn(),
};

const mockContext = {
  runId: 1,
  repo: { owner: "test-org", repo: "test-repo" },
  payload: { repository: { html_url: "https://github.com/test-org/test-repo" } },
};

global.core = mockCore;
global.github = mockGithub;
global.context = mockContext;

// ─── handler modules ──────────────────────────────────────────────────────────

let createProjectMain;
let createProjectStatusUpdateMain;
let updateProjectMain;

beforeAll(async () => {
  const cpMod = await import("./create_project.cjs");
  createProjectMain = cpMod.main;

  const cpsudMod = await import("./create_project_status_update.cjs");
  createProjectStatusUpdateMain = cpsudMod.main;

  const upMod = await import("./update_project.cjs");
  updateProjectMain = (upMod.default || upMod).main;
});

beforeEach(() => {
  // resetAllMocks clears the mockResolvedValueOnce queue in addition to call history,
  // preventing leftover queued responses from leaking between tests.
  vi.resetAllMocks();
});

// ─── GraphQL mock helpers ──────────────────────────────────────────────────────

const ORG_OWNER_RESPONSE = { organization: { id: "ORG_abc123" } };

function createProjectResponse(projectUrl, number = 1) {
  return {
    createProjectV2: {
      projectV2: {
        id: "PVT_proj1",
        number,
        title: "Test Project",
        url: projectUrl,
      },
    },
  };
}

function orgProjectV2Response(url, number = 1, id = "PVT_proj1") {
  return {
    organization: {
      projectV2: {
        id,
        number,
        title: "Test Project",
        url,
        owner: { __typename: "Organization", login: "test-org" },
      },
    },
  };
}

function statusUpdateResponse(id = "PVTSU_1") {
  return {
    createProjectV2StatusUpdate: {
      statusUpdate: {
        id,
        body: "Integration test status",
        bodyHTML: "<p>Integration test status</p>",
        startDate: null,
        targetDate: null,
        status: "ON_TRACK",
        createdAt: "2025-01-01T00:00:00Z",
      },
    },
  };
}

function repoResponse() {
  return {
    repository: { id: "repo123", owner: { id: "owner123", __typename: "Organization" } },
  };
}

function viewerResponse() {
  return { viewer: { login: "test-bot" } };
}

function issueResponse(id) {
  return { repository: { issue: { id, body: null } } };
}

function emptyItemsResponse() {
  return { node: { items: { nodes: [], pageInfo: { hasNextPage: false, endCursor: null } } } };
}

function addItemResponse(itemId = "PVTI_item1") {
  return { addProjectV2ItemById: { item: { id: itemId } } };
}

function fieldsResponse(nodes = []) {
  return { node: { fields: { nodes, pageInfo: { hasNextPage: false, endCursor: null } } } };
}

// ─── Integration: create_project → create_project_status_update ───────────────

describe("integration: create_project → create_project_status_update via temporary ID", () => {
  it("project URL stored by create_project is resolved by create_project_status_update", async () => {
    const projectUrl = "https://github.com/orgs/test-org/projects/42";

    // create_project: getOwnerId + createProjectV2
    mockGithub.graphql
      .mockResolvedValueOnce(ORG_OWNER_RESPONSE)
      .mockResolvedValueOnce(createProjectResponse(projectUrl, 42))
      // create_project_status_update: resolve project + create status update
      .mockResolvedValueOnce(orgProjectV2Response(projectUrl, 42))
      .mockResolvedValueOnce(statusUpdateResponse());

    const temporaryIdMap = new Map();

    const cpHandler = await createProjectMain({ target_owner: "test-org", max: 2 }, mockGithub);
    const cpResult = await cpHandler({ title: "Integration Project", temporary_id: "aw_iproj1" }, {}, temporaryIdMap);

    expect(cpResult.success).toBe(true);
    expect(cpResult.temporaryId).toBe("aw_iproj1");
    expect(temporaryIdMap.get("aw_iproj1")).toEqual({ projectUrl });

    // Now create_project_status_update references the project via temporary ID
    const cpsuHandler = await createProjectStatusUpdateMain({ max: 2 }, mockGithub);
    const cpsuResult = await cpsuHandler({ project: "#aw_iproj1", body: "Integration test status", status: "ON_TRACK" }, Object.fromEntries(temporaryIdMap), temporaryIdMap);

    expect(cpsuResult.success).toBe(true);
    expect(cpsuResult.status_update_id).toBe("PVTSU_1");
    expect(mockCore.info).toHaveBeenCalledWith(expect.stringContaining("Resolved temporary project ID #aw_iproj1"));
  });

  it("bare aw_xxx form also resolves in create_project_status_update", async () => {
    const projectUrl = "https://github.com/orgs/test-org/projects/43";

    mockGithub.graphql
      .mockResolvedValueOnce(ORG_OWNER_RESPONSE)
      .mockResolvedValueOnce(createProjectResponse(projectUrl, 43))
      .mockResolvedValueOnce(orgProjectV2Response(projectUrl, 43))
      .mockResolvedValueOnce(statusUpdateResponse("PVTSU_2"));

    const temporaryIdMap = new Map();

    const cpHandler = await createProjectMain({ target_owner: "test-org", max: 2 }, mockGithub);
    await cpHandler({ title: "Another Project", temporary_id: "aw_iproj2" }, {}, temporaryIdMap);

    const cpsuHandler = await createProjectStatusUpdateMain({ max: 2 }, mockGithub);
    const cpsuResult = await cpsuHandler({ project: "aw_iproj2", body: "Status update", status: "AT_RISK" }, Object.fromEntries(temporaryIdMap), temporaryIdMap);

    expect(cpsuResult.success).toBe(true);
    expect(cpsuResult.status_update_id).toBe("PVTSU_2");
  });

  it("create_project_status_update returns error when project temporary ID is not yet in map", async () => {
    const temporaryIdMap = new Map(); // empty — create_project not yet called

    const cpsuHandler = await createProjectStatusUpdateMain({ max: 2 }, mockGithub);
    const result = await cpsuHandler({ project: "#aw_unresolved", body: "Status", status: "ON_TRACK" }, Object.fromEntries(temporaryIdMap), temporaryIdMap);

    expect(result.success).toBe(false);
    expect(result.error).toContain("aw_unresolved");
    expect(mockGithub.graphql).not.toHaveBeenCalled();
  });
});

// ─── Integration: create_project → update_project ────────────────────────────

describe("integration: create_project → update_project via temporary ID", () => {
  it("project URL stored by create_project is resolved by update_project (project field)", async () => {
    const projectUrl = "https://github.com/orgs/test-org/projects/50";

    // create_project calls
    mockGithub.graphql
      .mockResolvedValueOnce(ORG_OWNER_RESPONSE)
      .mockResolvedValueOnce(createProjectResponse(projectUrl, 50))
      // update_project calls: repo, viewer, orgProject, issue, items, addItem
      .mockResolvedValueOnce(repoResponse())
      .mockResolvedValueOnce(viewerResponse())
      .mockResolvedValueOnce(orgProjectV2Response(projectUrl, 50, "PVT_proj50"))
      .mockResolvedValueOnce(issueResponse("ISSUE_node1"))
      .mockResolvedValueOnce(emptyItemsResponse())
      .mockResolvedValueOnce(addItemResponse("PVTI_item50"));

    const temporaryIdMap = new Map();

    const cpHandler = await createProjectMain({ target_owner: "test-org", max: 2 }, mockGithub);
    const cpResult = await cpHandler({ title: "Project 50", temporary_id: "#aw_proj50" }, {}, temporaryIdMap);

    expect(cpResult.success).toBe(true);
    expect(cpResult.temporaryId).toBe("aw_proj50");
    expect(temporaryIdMap.has("aw_proj50")).toBe(true);

    // update_project uses "#aw_proj50" as the project reference
    const upHandler = await updateProjectMain({ max: 5 }, mockGithub);
    const upResult = await upHandler({ project: "#aw_proj50", content_type: "issue", content_number: 99 }, Object.fromEntries(temporaryIdMap), temporaryIdMap);

    expect(upResult.success).toBe(true);
    expect(mockCore.info).toHaveBeenCalledWith(expect.stringContaining("Resolved temporary project ID"));
  });
});

// ─── Integration: issue → create_project (item_url) → update_project ─────────

describe("integration: pre-seeded issue → create_project (item_url) → update_project (content_number)", () => {
  it("create_project resolves issue temporary ID in item_url; update_project resolves issue and project IDs", async () => {
    const projectUrl = "https://github.com/orgs/test-org/projects/60";

    // The "issue" was created by an earlier create_issue handler and stored in the map
    const temporaryIdMap = new Map();
    temporaryIdMap.set("aw_issue1", { repo: "test-org/test-repo", number: 100 });

    // create_project: getOwnerId + createProjectV2 + getIssueNodeId + addItemToProject
    mockGithub.graphql
      .mockResolvedValueOnce(ORG_OWNER_RESPONSE)
      .mockResolvedValueOnce(createProjectResponse(projectUrl, 60))
      .mockResolvedValueOnce({ repository: { issue: { id: "ISSUE_node100" } } }) // getIssueNodeId
      .mockResolvedValueOnce(addItemResponse("PVTI_init_item"))
      // update_project calls: repo, viewer, orgProject, issue, items, addItem
      .mockResolvedValueOnce(repoResponse())
      .mockResolvedValueOnce(viewerResponse())
      .mockResolvedValueOnce(orgProjectV2Response(projectUrl, 60, "PVT_proj60"))
      .mockResolvedValueOnce(issueResponse("ISSUE_node100"))
      .mockResolvedValueOnce(emptyItemsResponse())
      .mockResolvedValueOnce(addItemResponse("PVTI_item60"));

    const cpHandler = await createProjectMain({ target_owner: "test-org", max: 2 }, mockGithub);
    const cpResult = await cpHandler(
      {
        title: "Project 60",
        temporary_id: "aw_proj60",
        item_url: "#aw_issue1", // resolved from temporaryIdMap
      },
      Object.fromEntries(temporaryIdMap),
      temporaryIdMap
    );

    expect(cpResult.success).toBe(true);
    expect(cpResult.temporaryId).toBe("aw_proj60");
    // project URL now in map
    expect(temporaryIdMap.get("aw_proj60")).toEqual({ projectUrl });
    expect(mockCore.info).toHaveBeenCalledWith(expect.stringContaining("Resolved temporary ID #aw_issue1 in item_url"));

    // update_project references both project and issue via temporary IDs
    const upHandler = await updateProjectMain({ max: 5 }, mockGithub);
    const upResult = await upHandler(
      {
        project: "#aw_proj60",
        content_type: "issue",
        content_number: "#aw_issue1",
      },
      Object.fromEntries(temporaryIdMap),
      temporaryIdMap
    );

    expect(upResult.success).toBe(true);
    expect(mockCore.info).toHaveBeenCalledWith(expect.stringContaining("Resolved temporary project ID"));
  });
});

// ─── Integration: normalisation across the pipeline ──────────────────────────

describe("integration: temporary ID normalisation across the pipeline", () => {
  it("project created with uppercase temporary_id is found by subsequent handlers using lowercase", async () => {
    const projectUrl = "https://github.com/orgs/test-org/projects/70";

    // create_project: getOwnerId + createProjectV2
    mockGithub.graphql
      .mockResolvedValueOnce(ORG_OWNER_RESPONSE)
      .mockResolvedValueOnce(createProjectResponse(projectUrl, 70))
      // create_project_status_update: resolve project + create status update
      .mockResolvedValueOnce(orgProjectV2Response(projectUrl, 70))
      .mockResolvedValueOnce(statusUpdateResponse("PVTSU_norm"));

    const temporaryIdMap = new Map();

    const cpHandler = await createProjectMain({ target_owner: "test-org", max: 2 }, mockGithub);
    const cpResult = await cpHandler({ title: "Project 70", temporary_id: "#aw_PROJ70" }, {}, temporaryIdMap);

    expect(cpResult.success).toBe(true);
    expect(cpResult.temporaryId).toBe("aw_proj70"); // normalised
    expect(temporaryIdMap.has("aw_proj70")).toBe(true); // stored under lowercase

    // create_project_status_update uses a different casing — still resolves
    const cpsuHandler = await createProjectStatusUpdateMain({ max: 2 }, mockGithub);
    const cpsuResult = await cpsuHandler({ project: "#aw_proj70", body: "Normalisation test", status: "ON_TRACK" }, Object.fromEntries(temporaryIdMap), temporaryIdMap);

    expect(cpsuResult.success).toBe(true);
    expect(cpsuResult.status_update_id).toBe("PVTSU_norm");
  });
});
