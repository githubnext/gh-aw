import { describe, it, expect } from "vitest";
import { createRequire } from "module";

const req = createRequire(import.meta.url);
const { extractIssueStateFromData, extractPullRequestStateFromData, mergePullRequestState, hashNormalizedBody } = req("./safe_output_execution_metadata.cjs");

describe("safe_output_execution_metadata", () => {
  it("captures normalized issue body hashes in execution state", () => {
    expect(
      extractIssueStateFromData({
        title: "Issue title",
        body: "Body line 1  \r\nBody line 2\n",
        state: "open",
        labels: [{ name: "bug" }],
        assignees: [{ login: "octo" }],
      })
    ).toEqual({
      title: "Issue title",
      body_hash: hashNormalizedBody("Body line 1\nBody line 2"),
      state: "open",
      labels: ["bug"],
      assignees: ["octo"],
    });
  });

  it("captures pull request comparison fields needed for retained-update evaluation", () => {
    const beforeState = extractPullRequestStateFromData({
      title: "Old title",
      body: "Old body",
      state: "open",
      base: { ref: "main" },
      draft: true,
      head: { sha: "abc123" },
    });
    expect(beforeState).toEqual({
      title: "Old title",
      body_hash: hashNormalizedBody("Old body"),
      state: "open",
      base: "main",
      draft: true,
      head_sha: "abc123",
    });

    expect(
      mergePullRequestState(beforeState, {
        title: "New title",
        body: "New body",
        base: { ref: "release" },
        draft: false,
        head: { sha: "def456" },
      })
    ).toEqual({
      title: "New title",
      body_hash: hashNormalizedBody("New body"),
      state: "open",
      base: "release",
      draft: false,
      head_sha: "def456",
    });
  });
});
