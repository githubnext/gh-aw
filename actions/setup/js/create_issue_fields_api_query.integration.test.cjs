// @ts-check
import { describe, expect, it } from "vitest";
const { ISSUE_FIELDS_QUERY } = require("./create_issue.cjs");

describe("create_issue GraphQL field discovery query integration", () => {
  it("validates against live schema and excludes the removed IssueField/IssueFieldIteration fragments", async () => {
    const token = process.env.GITHUB_TOKEN || process.env.GH_TOKEN;
    if (!token) {
      console.log("Skipping live GraphQL schema test - no GITHUB_TOKEN or GH_TOKEN available");
      return;
    }

    const owner = process.env.GITHUB_REPOSITORY_OWNER || "github";
    const repo = process.env.GITHUB_REPOSITORY?.split("/")[1] || "gh-aw";

    const { getOctokit } = await import("@actions/github");
    const octokit = getOctokit(token);

    try {
      const result = await octokit.graphql(ISSUE_FIELDS_QUERY, { owner, repo });
      expect(result?.repository).toBeDefined();
    } catch (error) {
      const message = error instanceof Error ? error.message : String(error);
      if (message.includes("Blocked by DNS monitoring proxy")) {
        console.log("Skipping live GraphQL schema test - api.github.com blocked by DNS monitoring proxy");
        return;
      }
      throw error;
    }
  });
});
