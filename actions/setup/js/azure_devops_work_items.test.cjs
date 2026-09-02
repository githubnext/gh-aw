import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { createAzureDevOpsWorkItemHandler, resolveWorkItemReference } from "./azure_devops_work_items.cjs";

global.core = {
  info: vi.fn(),
  warning: vi.fn(),
};

describe("azure_devops_work_items", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    process.env.SYSTEM_ACCESSTOKEN = "test-token";
    process.env.AZURE_DEVOPS_ORG_URL = "https://dev.azure.com/test-org";
    process.env.SYSTEM_TEAMPROJECT = "test-project";
    process.env.GITHUB_RUN_ID = "123";
    process.env.GITHUB_RUN_ATTEMPT = "1";
    global.fetch = vi.fn();
  });

  afterEach(() => {
    delete process.env.SYSTEM_ACCESSTOKEN;
    delete process.env.AZURE_DEVOPS_ORG_URL;
    delete process.env.SYSTEM_TEAMPROJECT;
    delete process.env.GITHUB_RUN_ID;
    delete process.env.GITHUB_RUN_ATTEMPT;
    delete global.fetch;
  });

  it("creates a work item through the configured organization and project", async () => {
    global.fetch.mockResolvedValue({
      ok: true,
      status: 200,
      statusText: "OK",
      text: vi.fn().mockResolvedValue(JSON.stringify({ id: 42, url: "https://dev.azure.com/test-org/_apis/wit/workItems/42" })),
    });

    const result = await createAzureDevOpsWorkItemHandler("create_work_item", {
      work_item_type: "Task",
      area_path: "test-project\\Platform",
      max: 1,
    })(
      {
        temporary_id: "#aw_item",
        title: "Fix the build",
        description: "Detailed description of the build failure.",
      },
      {}
    );

    expect(result).toMatchObject({
      success: true,
      temporaryId: "#aw_item",
      number: 42,
    });
    expect(global.fetch).toHaveBeenCalledOnce();
    expect(global.fetch.mock.calls[0][0]).toBe("https://dev.azure.com/test-org/test-project/_apis/wit/workitems/$Task?api-version=7.0");
    expect(global.fetch.mock.calls[0][1]).toMatchObject({
      method: "POST",
      redirect: "manual",
      headers: {
        Accept: "application/json",
        "Content-Type": "application/json-patch+json",
      },
    });
  });

  it("rejects updates to fields not enabled by configuration", async () => {
    const result = await createAzureDevOpsWorkItemHandler("update_work_item", {
      target: "*",
      title: false,
    })({ id: 42, title: "New title" }, {});

    expect(result).toEqual({
      success: false,
      error: "title updates are not enabled by update-work-item",
    });
    expect(global.fetch).not.toHaveBeenCalled();
  });

  it("rejects area paths outside configured prefixes", async () => {
    const result = await createAzureDevOpsWorkItemHandler("update_work_item", {
      staged: true,
      area_path: true,
      allowed_area_prefixes: ["test-project\\Platform"],
    })({ id: 42, area_path: "test-project\\Other" }, {});

    expect(result).toEqual({
      success: false,
      error: "area_path is not permitted by the configured area-path prefixes",
    });
    expect(global.fetch).not.toHaveBeenCalled();
  });

  it("rejects reserved agent identities", async () => {
    const result = await createAzureDevOpsWorkItemHandler("assign_work_item", {
      target: "*",
    })({ id: 42, assignee: "GitHub Copilot" }, {});

    expect(result).toEqual({
      success: false,
      error: "assignee 'GitHub Copilot' is a reserved identity",
    });
    expect(global.fetch).not.toHaveBeenCalled();
  });

  it("rejects temporary IDs from a different provider", () => {
    expect(() =>
      resolveWorkItemReference(
        "#aw_issue",
        {
          aw_issue: {
            repo: "owner/repo",
            number: 42,
          },
        },
        false
      )
    ).toThrow("has not been resolved by create-work-item in this run");
  });
});
