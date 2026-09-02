// @ts-check
import { afterEach, describe, expect, it, vi } from "vitest";

const { createJiraClient, formatJiraError, normalizeJiraBaseUrl, textToADF } = require("./jira_client.cjs");

afterEach(() => {
  vi.unstubAllGlobals();
});

describe("jira client", () => {
  it("normalizes site and Atlassian gateway base URLs", () => {
    expect(normalizeJiraBaseUrl("https://example.atlassian.net/")).toBe("https://example.atlassian.net");
    expect(normalizeJiraBaseUrl("https://example.atlassian.net/rest/api/3")).toBe("https://example.atlassian.net");
    expect(normalizeJiraBaseUrl("https://api.atlassian.com/ex/jira/cloud-id/")).toBe("https://api.atlassian.com/ex/jira/cloud-id");
  });

  it("rejects unsafe base URLs", () => {
    expect(() => normalizeJiraBaseUrl("http://example.atlassian.net")).toThrow("must use HTTPS");
    expect(() => normalizeJiraBaseUrl("https://example.atlassian.net?token=value")).toThrow("query string");
  });

  it("converts plain text and newlines to ADF version 1", () => {
    expect(textToADF("first\n\nsecond")).toEqual({
      type: "doc",
      version: 1,
      content: [
        { type: "paragraph", content: [{ type: "text", text: "first" }] },
        { type: "paragraph", content: [] },
        { type: "paragraph", content: [{ type: "text", text: "second" }] },
      ],
    });
  });

  it("sends credentials only in the HTTP Authorization header and accepts 204", async () => {
    const fetchMock = vi.fn(async () => ({
      ok: true,
      status: 204,
      statusText: "No Content",
      text: async () => "",
    }));
    const client = createJiraClient(
      {
        JIRA_BASE_URL: "https://example.atlassian.net",
        JIRA_USER_EMAIL: "jira@example.com",
        JIRA_API_TOKEN: "secret-token",
      },
      fetchMock
    );

    await expect(client.request("/issue/ENG-1", { method: "PUT", body: { fields: { summary: "Updated" } } })).resolves.toBeNull();
    const [url, options] = fetchMock.mock.calls[0];
    expect(url).toBe("https://example.atlassian.net/rest/api/3/issue/ENG-1");
    expect(options.headers.Authorization).toBe(`Basic ${Buffer.from("jira@example.com:secret-token").toString("base64")}`);
    expect(options.body).not.toContain("secret-token");
  });

  it("surfaces structured Jira errors without leaking credentials", async () => {
    const fetchMock = vi.fn(async () => ({
      ok: false,
      status: 400,
      statusText: "Bad Request",
      text: async () =>
        JSON.stringify({
          errorMessages: ["Cannot create issue for jira@example.com"],
          errors: { summary: "secret-token is invalid" },
        }),
    }));
    const client = createJiraClient(
      {
        JIRA_BASE_URL: "https://example.atlassian.net",
        JIRA_USER_EMAIL: "jira@example.com",
        JIRA_API_TOKEN: "secret-token",
      },
      fetchMock
    );

    await expect(client.request("/issue", { method: "POST", body: {} })).rejects.toThrow("summary: *** is invalid");
    await expect(client.request("/issue", { method: "POST", body: {} })).rejects.not.toThrow(/secret-token|jira@example\.com/);
  });

  it("reports missing configuration without values", () => {
    expect(() => createJiraClient({})).toThrow("JIRA_BASE_URL");
    expect(() => createJiraClient({ JIRA_BASE_URL: "https://example.atlassian.net" })).toThrow("JIRA_USER_EMAIL and JIRA_API_TOKEN");
  });

  it("formats field and global errors", () => {
    expect(formatJiraError(400, "Bad Request", { errorMessages: ["Invalid request"], errors: { project: "Unknown project" } }, [])).toContain("Invalid request; project: Unknown project");
  });
});
