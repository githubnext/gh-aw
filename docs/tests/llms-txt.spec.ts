import { test, expect } from "@playwright/test";

test.describe("llms.txt", () => {
  test("publishes a structured index for AI consumers", async ({ request }) => {
    const response = await request.get("/gh-aw/llms.txt");
    expect(response.ok()).toBeTruthy();
    expect(response.headers()["content-type"]).toContain("text/plain");

    const body = await response.text();

    expect(body).toContain("# GitHub Agentic Workflows");
    expect(body).toContain("GitHub Agentic Workflows (gh-aw) is a GitHub CLI extension");
    expect(body).toContain("> Use this index to find");
    expect(body).toContain("## Documentation");
    expect(body).toContain("## Guides");
    expect(body).toContain("## Reference");
    expect(body).toContain("## Agent Prompts");
    expect(body).toContain("[Quickstart](https://github.github.com/gh-aw/setup/quick-start/)");
  });
});
