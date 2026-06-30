// @ts-check
import { describe, expect, it } from "vitest";

const ISSUE_FIELDS_DISCOVERY_QUERY = `query($owner: String!, $repo: String!) {
  repository(owner: $owner, name: $repo) {
    issueFields(first: 100) {
      nodes {
        __typename
        ... on IssueFieldText { id name }
        ... on IssueFieldNumber { id name }
        ... on IssueFieldDate { id name }
        ... on IssueFieldSingleSelect { id name options { id name } }
        ... on IssueFieldMultiSelect { id name options { id name } }
      }
    }
  }
}`;

describe("set_issue_field GraphQL discovery query integration", () => {
  it("validates against live schema and excludes the removed IssueField fragment", async () => {
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
      const result = await octokit.graphql(ISSUE_FIELDS_DISCOVERY_QUERY, { owner, repo });
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
