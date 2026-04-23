// @ts-check
import { describe, it, expect, beforeEach, afterEach, vi } from "vitest";
import { createRequire } from "module";

const require = createRequire(import.meta.url);
const { main: createDiscussionMain } = require("./create_discussion.cjs");

describe("create_discussion body sanitization", () => {
  let mockGithub;
  let mockCore;
  let mockContext;
  let mockExec;
  let originalEnv;

  beforeEach(() => {
    // Save original environment
    originalEnv = { ...process.env };

    // Mock GitHub API with discussion categories
    mockGithub = {
      rest: {},
      graphql: vi.fn().mockImplementation((query, variables) => {
        // Handle repository query (fetch categories)
        if (query.includes("discussionCategories")) {
          return Promise.resolve({
            repository: {
              id: "R_test123",
              discussionCategories: {
                nodes: [
                  {
                    id: "DIC_kwDOGFsHUM4BsUn1",
                    name: "General",
                    slug: "general",
                    description: "General discussions",
                  },
                ],
              },
            },
          });
        }
        // Handle create discussion mutation — echo back the body for assertion
        if (query.includes("createDiscussion")) {
          return Promise.resolve({
            createDiscussion: {
              discussion: {
                id: "D_test456",
                number: 42,
                title: variables.title,
                url: "https://github.com/test-owner/test-repo/discussions/42",
              },
            },
          });
        }
        return Promise.reject(new Error("Unknown GraphQL query"));
      }),
    };

    // Mock Core
    mockCore = {
      info: vi.fn(),
      warning: vi.fn(),
      error: vi.fn(),
      setOutput: vi.fn(),
    };

    // Mock Context
    mockContext = {
      repo: { owner: "test-owner", repo: "test-repo" },
      runId: 12345,
      payload: {
        repository: {
          html_url: "https://github.com/test-owner/test-repo",
        },
      },
    };

    // Mock Exec
    mockExec = {
      exec: vi.fn().mockResolvedValue(0),
    };

    // Set globals
    global.github = mockGithub;
    global.core = mockCore;
    global.context = mockContext;
    global.exec = mockExec;

    // Set required environment variables
    process.env.GH_AW_WORKFLOW_NAME = "Test Workflow";
    process.env.GH_AW_WORKFLOW_ID = "test-workflow";
    process.env.GH_AW_WORKFLOW_SOURCE_URL = "https://github.com/owner/repo/blob/main/workflow.md";
    process.env.GITHUB_SERVER_URL = "https://github.com";
  });

  afterEach(() => {
    // Restore environment by mutating process.env in place
    for (const key of Object.keys(process.env)) {
      if (!(key in originalEnv)) {
        delete process.env[key];
      }
    }
    Object.assign(process.env, originalEnv);
    vi.clearAllMocks();
  });

  it("should neutralize @mentions in discussion body", async () => {
    const handler = await createDiscussionMain({ max: 5, category: "general" });
    const result = await handler(
      {
        title: "Test Discussion",
        body: "This was reported by @malicious-user and CC @another-user.",
      },
      {}
    );

    expect(result.success).toBe(true);

    const createMutationCall = mockGithub.graphql.mock.calls.find(call => call[0].includes("createDiscussion"));
    expect(createMutationCall).toBeDefined();
    const body = createMutationCall[1].body;
    // @mentions must be neutralized to backtick form
    expect(body).toContain("`@malicious-user`");
    expect(body).toContain("`@another-user`");
    // Raw unescaped @mentions must not appear
    expect(body).not.toMatch(/(?<![`])@malicious-user(?![`])/);
    expect(body).not.toMatch(/(?<![`])@another-user(?![`])/);
  });

  it("should expose hidden markdown link title XPIA payloads in discussion body (closing the XPIA channel)", async () => {
    const handler = await createDiscussionMain({ max: 5, category: "general" });
    const result = await handler(
      {
        title: "Security Test",
        body: 'Click [here](https://example.com "XPIA hidden payload") for details.',
      },
      {}
    );

    expect(result.success).toBe(true);

    const createMutationCall = mockGithub.graphql.mock.calls.find(call => call[0].includes("createDiscussion"));
    expect(createMutationCall).toBeDefined();
    const body = createMutationCall[1].body;
    // The hidden title must be moved to visible text (closing the XPIA channel)
    // sanitizeContent converts [text](url "hidden") → [text (hidden)](url)
    expect(body).toContain("XPIA hidden payload");
    // The payload must no longer be in a hidden markdown link title attribute
    expect(body).not.toMatch(/\[.*\]\([^)]*"XPIA hidden payload"[^)]*\)/);
  });

  it("should sanitize body but preserve the footer workflow marker", async () => {
    const handler = await createDiscussionMain({ max: 5, category: "general" });
    const result = await handler(
      {
        title: "Test Discussion",
        body: "Please notify @someone about this finding.",
      },
      {}
    );

    expect(result.success).toBe(true);

    const createMutationCall = mockGithub.graphql.mock.calls.find(call => call[0].includes("createDiscussion"));
    expect(createMutationCall).toBeDefined();
    const body = createMutationCall[1].body;
    // @mention must be neutralized
    expect(body).toContain("`@someone`");
    // System-generated footer marker must still be present
    expect(body).toContain("gh-aw-workflow-id");
  });
});
